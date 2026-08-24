package deepseekfairuse

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type redisIntegrationFixture struct {
	client  *redis.Client
	ctx     context.Context
	account string
	keys    []string
}

func newRedisIntegrationFixture(t *testing.T) *redisIntegrationFixture {
	t.Helper()
	connectionString := strings.TrimSpace(os.Getenv("DEEPSEEK_FUP_REDIS_CONN_STRING"))
	address := strings.TrimSpace(os.Getenv("DEEPSEEK_FUP_REDIS_ADDR"))
	if connectionString == "" && address == "" {
		connectionString = strings.TrimSpace(os.Getenv("REDIS_CONN_STRING"))
	}
	var options *redis.Options
	if connectionString != "" {
		var err error
		options, err = redis.ParseURL(connectionString)
		if err != nil {
			t.Skip("configured Redis connection string is unavailable for the fair-use integration suite")
		}
	} else if address != "" {
		options = &redis.Options{
			Addr:     address,
			Password: os.Getenv("DEEPSEEK_FUP_REDIS_PASSWORD"),
		}
	} else {
		t.Skip("set DEEPSEEK_FUP_REDIS_CONN_STRING, DEEPSEEK_FUP_REDIS_ADDR, or REDIS_CONN_STRING to run real Redis/Lua integration tests")
	}

	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := client.Ping(ctx).Err(); err != nil {
		cancel()
		_ = client.Close()
		t.Skip("configured Redis endpoint is unavailable for the fair-use integration suite")
	}
	cancel()

	account := fmt.Sprintf("integration-%x", time.Now().UnixNano())
	fixture := &redisIntegrationFixture{
		client:  client,
		ctx:     context.Background(),
		account: account,
		keys:    accountKeys(account),
	}
	require.NoError(t, client.Del(fixture.ctx, fixture.keys...).Err())
	t.Cleanup(func() {
		_ = client.Del(context.Background(), fixture.keys...).Err()
		_ = client.Close()
	})
	return fixture
}

func (f *redisIntegrationFixture) limiter() *Limiter {
	return New(f.client)
}

func (f *redisIntegrationFixture) releaseLease(t *testing.T, lease *Lease, completed bool) {
	t.Helper()
	if lease == nil {
		return
	}
	_, err := lease.Release(f.ctx, completed)
	require.NoError(t, err)
}

func (f *redisIntegrationFixture) serverMilliseconds(t *testing.T) int64 {
	t.Helper()
	now, err := f.client.Time(f.ctx).Result()
	require.NoError(t, err)
	return now.UnixMilli()
}

func (f *redisIntegrationFixture) seedSuccessLimit(t *testing.T, score int64) {
	t.Helper()
	members := make([]*redis.Z, SuccessRequestLimit)
	for i := range members {
		members[i] = &redis.Z{Score: float64(score), Member: fmt.Sprintf("success-%d", i)}
	}
	require.NoError(t, f.client.ZAdd(f.ctx, f.keys[4], members...).Err())
}

func (f *redisIntegrationFixture) seedOccupancy(t *testing.T, nowMS int64, partialMS int64) {
	t.Helper()
	lower := nowMS - WindowDuration.Milliseconds()
	segments := make([]*redis.Z, 0, 4)
	for i := 0; i < 3; i++ {
		start := lower + 1000
		segments = append(segments, &redis.Z{
			Score:  float64(nowMS),
			Member: fmt.Sprintf("%d|%d|seed-long-%d", start, nowMS, i),
		})
	}
	segments = append(segments, &redis.Z{
		Score:  float64(nowMS),
		Member: fmt.Sprintf("%d|%d|seed-partial", nowMS-partialMS, nowMS),
	})
	require.NoError(t, f.client.ZAdd(f.ctx, f.keys[2], segments...).Err())
}

func (f *redisIntegrationFixture) clearLimiterStatePreserveEpisodes(t *testing.T) {
	t.Helper()
	// Keep only the Lua-owned exhaustion-event zset so each subsequent phase
	// starts with a clean budget while retaining the episodes already produced
	// by the script.
	require.NoError(t, f.client.Del(f.ctx, f.keys[:9]...).Err())
}

func TestRedisLuaAdmissionIsAtomicAtThePeakBoundary(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	const attempts = PeakConcurrency + 1
	start := make(chan struct{})
	results := make(chan struct {
		lease    *Lease
		decision Decision
		err      error
	}, attempts)

	var wg sync.WaitGroup
	wg.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func(index int) {
			defer wg.Done()
			<-start
			lease, decision, err := f.limiter().Acquire(f.ctx, f.account, fmt.Sprintf("atomic-%d", index), RiskSignals{})
			results <- struct {
				lease    *Lease
				decision Decision
				err      error
			}{lease: lease, decision: decision, err: err}
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	allowed := 0
	leases := make([]*Lease, 0, PeakConcurrency)
	for result := range results {
		require.NoError(t, result.err)
		if result.decision.Allowed {
			allowed++
			leases = append(leases, result.lease)
			assert.LessOrEqual(t, result.decision.Snapshot.Active, PeakConcurrency)
		}
	}
	assert.Equal(t, PeakConcurrency, allowed)
	for _, lease := range leases {
		f.releaseLease(t, lease, false)
	}
}

func TestRedisLuaUsesServerTimeAndSetsBoundedStateAndRiskTTLs(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	before := f.serverMilliseconds(t)
	boundaryMember := fmt.Sprintf("%d|%d|boundary", before, before)
	require.NoError(t, f.client.ZAdd(f.ctx, f.keys[2], &redis.Z{Score: float64(before), Member: boundaryMember}).Err())
	require.NoError(t, f.client.ZAdd(f.ctx, f.keys[4], &redis.Z{Score: float64(before), Member: "seed-success"}).Err())
	require.NoError(t, f.client.HSet(f.ctx, f.keys[5], "seed", "1").Err())
	require.NoError(t, f.client.ZAdd(f.ctx, f.keys[9], &redis.Z{Score: float64(before), Member: "seed-event"}).Err())

	lease, decision, err := f.limiter().Acquire(f.ctx, f.account, "server-time", RiskSignals{
		IPHMAC:        "ip-hmac",
		CountryHMAC:   "country-hmac",
		UserAgentHMAC: "ua-hmac",
	})
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.NotNil(t, lease)
	after := f.serverMilliseconds(t)
	rawLease, err := f.client.HGet(f.ctx, f.keys[0], "server-time").Result()
	require.NoError(t, err)
	parts := strings.Split(rawLease, "|")
	require.Len(t, parts, 3)
	startedMS, err := strconv.ParseInt(parts[0], 10, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, startedMS, before-1000)
	assert.LessOrEqual(t, startedMS, after+1000)

	for index, key := range f.keys {
		ttl, ttlErr := f.client.TTL(f.ctx, key).Result()
		require.NoError(t, ttlErr, "key %d", index)
		assert.Greater(t, ttl, time.Duration(0), "key %d should have a bounded TTL", index)
	}
	f.releaseLease(t, lease, false)
}

func TestRedisLuaKeepsRollingOccupancyExactAcrossTheLowerBoundary(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	nowMS := f.serverMilliseconds(t)
	f.seedOccupancy(t, nowMS, 1000)

	underLease, under, err := f.limiter().Acquire(f.ctx, f.account, "under-boundary", RiskSignals{})
	require.NoError(t, err)
	require.True(t, under.Allowed)
	assert.Less(t, under.Snapshot.ConcurrentSeconds, ConcurrentSecondsBudget)
	f.releaseLease(t, underLease, false)

	require.NoError(t, f.client.Del(f.ctx, f.keys...).Err())
	nowMS = f.serverMilliseconds(t)
	f.seedOccupancy(t, nowMS, 3000)
	exactLease, exactDecision, err := f.limiter().Acquire(f.ctx, f.account, "exact-boundary", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, exactLease)
	assert.False(t, exactDecision.Allowed)
	assert.Equal(t, ReasonConcurrentSecondsLimit, exactDecision.Reason)

	require.NoError(t, f.client.Del(f.ctx, f.keys...).Err())
	nowMS = f.serverMilliseconds(t)
	f.seedOccupancy(t, nowMS, 5000)
	at, atDecision, err := f.limiter().Acquire(f.ctx, f.account, "at-boundary", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, at)
	assert.False(t, atDecision.Allowed)
	assert.Equal(t, ReasonConcurrentSecondsLimit, atDecision.Reason)
}

func TestRedisLuaRejectsExactSuccessAndAdmittedRequestBoundaries(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	nowMS := f.serverMilliseconds(t)
	successes := make([]*redis.Z, SuccessRequestLimit)
	for i := range successes {
		successes[i] = &redis.Z{Score: float64(nowMS), Member: fmt.Sprintf("success-boundary-%d", i)}
	}
	require.NoError(t, f.client.ZAdd(f.ctx, f.keys[4], successes...).Err())

	lease, decision, err := f.limiter().Acquire(f.ctx, f.account, "success-boundary", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, lease)
	assert.False(t, decision.Allowed)
	assert.Equal(t, ReasonSuccessRequestLimit, decision.Reason)

	require.NoError(t, f.client.Del(f.ctx, f.keys...).Err())
	admitted := make([]*redis.Z, AdmittedRequestLimit)
	for i := range admitted {
		admitted[i] = &redis.Z{Score: float64(nowMS), Member: fmt.Sprintf("admitted-boundary-%d", i)}
	}
	require.NoError(t, f.client.ZAdd(f.ctx, f.keys[3], admitted...).Err())

	lease, decision, err = f.limiter().Acquire(f.ctx, f.account, "admitted-boundary", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, lease)
	assert.False(t, decision.Allowed)
	assert.Equal(t, ReasonAdmittedRequestLimit, decision.Reason)
}

func TestRedisLuaRecoversAStaleLeaseAtTheExactRecoveryBoundary(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	lease, decision, err := f.limiter().Acquire(f.ctx, f.account, "stale-lease", RiskSignals{})
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.NotNil(t, lease)

	nowMS := f.serverMilliseconds(t)
	staleHeartbeat := nowMS - StaleLeaseRecovery.Milliseconds() - 1
	raw := fmt.Sprintf("%d|%d|%d", staleHeartbeat, staleHeartbeat, staleHeartbeat)
	require.NoError(t, f.client.HSet(f.ctx, f.keys[0], "stale-lease", raw).Err())
	require.NoError(t, f.client.ZAdd(f.ctx, f.keys[1], &redis.Z{Score: float64(staleHeartbeat), Member: "stale-lease"}).Err())

	replacement, replacementDecision, err := f.limiter().Acquire(f.ctx, f.account, "replacement", RiskSignals{})
	require.NoError(t, err)
	require.True(t, replacementDecision.Allowed)
	require.NotNil(t, replacement)
	assert.Equal(t, 1, replacementDecision.Snapshot.Active)
	assert.NotContains(t, f.client.HGetAll(f.ctx, f.keys[0]).Val(), "stale-lease")
	f.releaseLease(t, replacement, false)
}

func TestRedisLuaDuplicateReleaseCountsOneCompletion(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	lease, decision, err := f.limiter().Acquire(f.ctx, f.account, "duplicate-release", RiskSignals{})
	require.NoError(t, err)
	require.True(t, decision.Allowed)

	f.releaseLease(t, lease, true)
	f.releaseLease(t, lease, true)

	successes, err := f.client.ZCard(f.ctx, f.keys[4]).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), successes)
	active, err := f.client.HLen(f.ctx, f.keys[0]).Result()
	require.NoError(t, err)
	assert.Zero(t, active)
}

func TestRedisLuaPeakRejectionIsNotASustainedStrike(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	for i := 0; i < PeakConcurrency; i++ {
		require.NoError(t, f.client.HSet(f.ctx, f.keys[0], fmt.Sprintf("peak-%d", i), "1|1|1").Err())
	}
	decision, err := f.client.ZCard(f.ctx, f.keys[9]).Result()
	require.NoError(t, err)
	assert.Zero(t, decision)

	lease, rejected, err := f.limiter().Acquire(f.ctx, f.account, "peak-rejected", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, lease)
	assert.False(t, rejected.Allowed)
	assert.Equal(t, ReasonConcurrencyLimit, rejected.Reason)
	strikes, err := f.client.ZCard(f.ctx, f.keys[9]).Result()
	require.NoError(t, err)
	assert.Equal(t, decision, strikes)
}

func TestRedisLuaRepeatedBudgetRejectionIsOneEpisode(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	f.seedSuccessLimit(t, f.serverMilliseconds(t)-1000)

	firstLease, first, err := f.limiter().Acquire(f.ctx, f.account, "budget-first", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, firstLease)
	assert.False(t, first.Allowed)
	secondLease, second, err := f.limiter().Acquire(f.ctx, f.account, "budget-second", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, secondLease)
	assert.False(t, second.Allowed)
	assert.Equal(t, first.Reason, second.Reason)

	strikes, err := f.client.ZCard(f.ctx, f.keys[9]).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), strikes)
}

func TestRedisLuaHeartbeatStopsLeaseAtConcurrentSecondsBudget(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	lease, admitted, err := f.limiter().Acquire(f.ctx, f.account, "heartbeat-budget", RiskSignals{})
	require.NoError(t, err)
	require.True(t, admitted.Allowed)
	require.NotNil(t, lease)
	f.seedOccupancy(t, f.serverMilliseconds(t), 10_000)

	err = lease.Heartbeat(f.ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), string(ReasonConcurrentSecondsLimit))
	assert.Zero(t, f.client.HLen(f.ctx, f.keys[0]).Val())
}

func TestRedisLuaThreeDistinctBudgetEpisodesDegradeAndNotifyOnce(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	for i := 0; i < PeakConcurrency; i++ {
		require.NoError(t, f.client.HSet(f.ctx, f.keys[0], fmt.Sprintf("peak-only-%d", i), "1|1|1").Err())
	}
	_, peakOnly, err := f.limiter().Acquire(f.ctx, f.account, "peak-only", RiskSignals{})
	require.NoError(t, err)
	assert.False(t, peakOnly.Allowed)
	assert.Equal(t, ReasonConcurrencyLimit, peakOnly.Reason)
	assert.Equal(t, int64(0), f.client.ZCard(f.ctx, f.keys[9]).Val())
	f.clearLimiterStatePreserveEpisodes(t)

	// Each phase below seeds one budget boundary and lets the embedded Lua
	// transition create its own reason-specific episode. No event members are
	// fabricated by the test.
	f.seedSuccessLimit(t, f.serverMilliseconds(t)-1000)

	firstLease, first, err := f.limiter().Acquire(f.ctx, f.account, "success-episode", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, firstLease)
	assert.False(t, first.Allowed)
	assert.Equal(t, ReasonSuccessRequestLimit, first.Reason)
	assert.False(t, first.NotifyRootAdmin)
	assert.False(t, first.Snapshot.Degraded)
	assert.Equal(t, int64(1), f.client.ZCard(f.ctx, f.keys[9]).Val())

	secondLease, second, err := f.limiter().Acquire(f.ctx, f.account, "success-episode-repeat", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, secondLease)
	assert.False(t, second.Allowed)
	assert.Equal(t, first.Reason, second.Reason)
	assert.False(t, second.NotifyRootAdmin)
	assert.Equal(t, first.Snapshot.ExhaustionEvents, second.Snapshot.ExhaustionEvents)
	f.clearLimiterStatePreserveEpisodes(t)

	nowMS := f.serverMilliseconds(t)
	totalMembers := make([]*redis.Z, AdmittedRequestLimit)
	for i := range totalMembers {
		totalMembers[i] = &redis.Z{Score: float64(nowMS), Member: fmt.Sprintf("admitted-episode-%d", i)}
	}
	require.NoError(t, f.client.ZAdd(f.ctx, f.keys[3], totalMembers...).Err())
	thirdLease, third, err := f.limiter().Acquire(f.ctx, f.account, "admitted-episode", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, thirdLease)
	assert.False(t, third.Allowed)
	assert.Equal(t, ReasonAdmittedRequestLimit, third.Reason)
	assert.False(t, third.NotifyRootAdmin)
	assert.Equal(t, int64(2), f.client.ZCard(f.ctx, f.keys[9]).Val())

	fourthLease, fourth, err := f.limiter().Acquire(f.ctx, f.account, "admitted-episode-repeat", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, fourthLease)
	assert.False(t, fourth.Allowed)
	assert.Equal(t, third.Reason, fourth.Reason)
	assert.False(t, fourth.NotifyRootAdmin)
	assert.Equal(t, third.Snapshot.ExhaustionEvents, fourth.Snapshot.ExhaustionEvents)
	f.clearLimiterStatePreserveEpisodes(t)

	f.seedOccupancy(t, f.serverMilliseconds(t), 30000)
	fifthLease, fifth, err := f.limiter().Acquire(f.ctx, f.account, "occupancy-episode", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, fifthLease)
	assert.False(t, fifth.Allowed)
	assert.Equal(t, ReasonConcurrentSecondsLimit, fifth.Reason)
	assert.True(t, fifth.NotifyRootAdmin)
	assert.True(t, fifth.Snapshot.Degraded)
	assert.Equal(t, DegradedConcurrency, fifth.Snapshot.EffectiveConcurrency)
	assert.Equal(t, int64(3), f.client.ZCard(f.ctx, f.keys[9]).Val())

	sixthLease, sixth, err := f.limiter().Acquire(f.ctx, f.account, "occupancy-episode-repeat", RiskSignals{})
	require.NoError(t, err)
	assert.Nil(t, sixthLease)
	assert.False(t, sixth.Allowed)
	assert.Equal(t, fifth.Reason, sixth.Reason)
	assert.False(t, sixth.NotifyRootAdmin)
	assert.Equal(t, fifth.Snapshot.ExhaustionEvents, sixth.Snapshot.ExhaustionEvents)
	assert.Equal(t, int64(3), f.client.ZCard(f.ctx, f.keys[9]).Val())

	events, err := f.client.ZRange(f.ctx, f.keys[9], 0, -1).Result()
	require.NoError(t, err)
	assert.Len(t, events, 3)
	assert.True(t, anyEventWithPrefix(events, string(ReasonSuccessRequestLimit)+":"))
	assert.True(t, anyEventWithPrefix(events, string(ReasonAdmittedRequestLimit)+":"))
	assert.True(t, anyEventWithPrefix(events, string(ReasonConcurrentSecondsLimit)+":"))
}

func anyEventWithPrefix(events []string, prefix string) bool {
	for _, event := range events {
		if strings.HasPrefix(event, prefix) {
			return true
		}
	}
	return false
}

func TestRedisLuaNeverStoresRawTokenIPAuthorizationOrUserAgent(t *testing.T) {
	f := newRedisIntegrationFixture(t)
	rawToken := "sk-fup-raw-token"
	rawIP := "203.0.113.77"
	rawAuthorization := "Bearer " + rawToken
	rawUserAgent := "fup-test-agent/1.0"
	identifiers := BuildIdentifiers([]byte("integration-hmac-secret"), IdentityInput{
		UserID:    51,
		TokenID:   52,
		ClientIP:  rawIP,
		Country:   "CA",
		UserAgent: rawUserAgent,
	})

	lease, decision, err := f.limiter().Acquire(f.ctx, f.account, "privacy-lease", identifiers.RiskSignals())
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	f.releaseLease(t, lease, false)

	for _, key := range f.keys {
		dump, dumpErr := f.client.Dump(f.ctx, key).Result()
		if dumpErr == redis.Nil {
			continue
		}
		require.NoError(t, dumpErr)
		payload := string(dump)
		assert.NotContains(t, payload, rawToken)
		assert.NotContains(t, payload, rawIP)
		assert.NotContains(t, payload, rawAuthorization)
		assert.NotContains(t, payload, rawUserAgent)
	}
}
