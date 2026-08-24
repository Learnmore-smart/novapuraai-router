package deepseekfairuse

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testClock struct {
	currentTime time.Time
}

func newTestClock(now time.Time) *testClock {
	return &testClock{currentTime: now}
}

func (c *testClock) Now() time.Time                 { return c.currentTime }
func (c *testClock) now() time.Time                 { return c.currentTime }
func (c *testClock) advance(duration time.Duration) { c.currentTime = c.currentTime.Add(duration) }

func leaseID(index int) string { return fmt.Sprintf("lease-%d", index) }

func TestMemoryLimiterAllowsTenConcurrentRequestsAndRejectsEleventh(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)

	for i := 0; i < PeakConcurrency; i++ {
		decision := limiter.acquire("account-a", leaseID(i))
		require.True(t, decision.Allowed, "lease %d should be admitted", i+1)
	}

	eleventh := limiter.acquire("account-a", leaseID(PeakConcurrency))
	assert.False(t, eleventh.Allowed)
	assert.Equal(t, ReasonConcurrencyLimit, eleventh.Reason)

	limiter.release("account-a", leaseID(0), false)
	assert.True(t, limiter.acquire("account-a", "replacement").Allowed)
}

func TestMemoryLimiterAggregatesAcrossTokensForOneAccount(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)

	for i := 0; i < PeakConcurrency; i++ {
		require.True(t, limiter.acquire("same-user", "token-a-"+leaseID(i)).Allowed)
	}
	assert.False(t, limiter.acquire("same-user", "token-b-1").Allowed)
	assert.True(t, limiter.acquire("different-user", "token-b-1").Allowed)
}

func TestMemoryLimiterMarksRiskFanoutButNeverRejectsForRiskAlone(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)

	for i := 0; i < RiskIPThreshold; i++ {
		decision := limiter.acquireWithRisk("risk-ip", leaseID(i), RiskSignals{
			IPHMAC:        fmt.Sprintf("ip-%d", i),
			CountryHMAC:   "country-ca",
			UserAgentHMAC: "ua-curl",
		})
		assert.True(t, decision.Allowed)
		if i == RiskIPThreshold-1 {
			assert.True(t, decision.RiskMarked)
		}
		limiter.release("risk-ip", leaseID(i), false)
	}

	for i := 0; i < RiskCountryThreshold; i++ {
		decision := limiter.acquireWithRisk("risk-country", leaseID(i), RiskSignals{
			IPHMAC:        "ip-fixed",
			CountryHMAC:   fmt.Sprintf("country-%d", i),
			UserAgentHMAC: "ua-curl",
		})
		assert.True(t, decision.Allowed)
		if i == RiskCountryThreshold-1 {
			assert.True(t, decision.RiskMarked)
		}
		limiter.release("risk-country", leaseID(i), false)
	}
}

func TestMemoryLimiterUsesExactBudgetEdges(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)

	limiter.setConcurrentSeconds("edge", ConcurrentSecondsBudget-1)
	assert.True(t, limiter.acquire("edge", "under-budget").Allowed)
	limiter.release("edge", "under-budget", false)
	limiter.setConcurrentSeconds("edge", ConcurrentSecondsBudget)
	assert.Equal(t, ReasonConcurrentSecondsLimit, limiter.acquire("edge", "at-budget").Reason)

	for i := 0; i < SuccessRequestLimit; i++ {
		lease := "success-" + leaseID(i)
		require.True(t, limiter.acquire("success", lease).Allowed)
		limiter.release("success", lease, true)
	}
	assert.Equal(t, ReasonSuccessRequestLimit, limiter.acquire("success", "success-601").Reason)

	for i := 0; i < AdmittedRequestLimit; i++ {
		lease := "admitted-" + leaseID(i)
		require.True(t, limiter.acquire("admitted", lease).Allowed)
		limiter.release("admitted", lease, false)
	}
	assert.Equal(t, ReasonAdmittedRequestLimit, limiter.acquire("admitted", "admitted-751").Reason)
}

func TestMemoryLimiterHeartbeatPreventsStaleRecoveryUntilItStops(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)
	require.True(t, limiter.acquire("heartbeat", "lease").Allowed)

	clock.advance(StaleLeaseRecovery - time.Second)
	require.NoError(t, limiter.heartbeat("heartbeat", "lease"))
	clock.advance(StaleLeaseRecovery - time.Second)
	assert.Equal(t, 1, limiter.active("heartbeat"))

	clock.advance(2 * time.Second)
	assert.True(t, limiter.acquire("heartbeat", "replacement").Allowed)
	assert.Equal(t, 1, limiter.active("heartbeat"))
}

func TestMemoryLimiterHeartbeatStopsLeaseAtConcurrentSecondsBudget(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)
	require.True(t, limiter.acquire("heartbeat-budget", "lease").Allowed)
	limiter.setConcurrentSeconds("heartbeat-budget", ConcurrentSecondsBudget-1)
	clock.advance(2 * time.Second)

	err := limiter.heartbeat("heartbeat-budget", "lease")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrConcurrentSecondsBudget)
	assert.Zero(t, limiter.active("heartbeat-budget"))
}

func TestMemoryLimiterRecoversAtTheExactStaleLeaseBoundary(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)
	require.True(t, limiter.acquire("stale-edge", "lease").Allowed)

	clock.advance(StaleLeaseRecovery)
	assert.True(t, limiter.acquire("stale-edge", "replacement").Allowed)
	assert.Equal(t, 1, limiter.active("stale-edge"))
}

func TestMemoryLimiterChargesLeaseOccupancyBeforeAdmissionDecision(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)
	require.True(t, limiter.acquire("occupancy", "first").Allowed)
	clock.advance(2 * time.Second)
	require.True(t, limiter.acquire("occupancy", "second").Allowed)
	clock.advance(2 * time.Second)
	decision := limiter.acquire("occupancy", "third")

	assert.True(t, decision.Allowed)
	assert.Equal(t, int64(6), decision.Snapshot.ConcurrentSeconds)
}

func TestMemoryLimiterCountsCompletionOnceAndOnlyNormalCompletionAsSuccess(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)
	require.True(t, limiter.acquire("completion", "failed").Allowed)
	limiter.release("completion", "failed", false)
	assert.Zero(t, limiter.successes("completion"))

	require.True(t, limiter.acquire("completion", "success").Allowed)
	limiter.release("completion", "success", true)
	limiter.release("completion", "success", true)
	assert.Equal(t, 1, limiter.successes("completion"))
}

func TestLeaseLifecycleUsesOneAdmissionAcrossRetriesAndOnlyFirstCompletedReleaseCounts(t *testing.T) {
	actions := make([]string, 0, 5)
	completed := make([]bool, 0, 5)
	limiter := &Limiter{
		runOverride: func(_ context.Context, action, _, _ string, _ RiskSignals, isCompleted bool) (Decision, error) {
			actions = append(actions, action)
			completed = append(completed, isCompleted)
			return Decision{Allowed: true, Snapshot: Snapshot{Active: 1}}, nil
		},
	}

	lease, _, err := limiter.Acquire(context.Background(), "account", "lease", RiskSignals{})
	require.NoError(t, err)
	require.NotNil(t, lease)
	// A channel retry keeps the request's original lease alive instead of
	// acquiring a second admission for the same external request.
	require.NoError(t, lease.Heartbeat(context.Background()))
	require.NoError(t, lease.Heartbeat(context.Background()))
	_, err = lease.Release(context.Background(), true)
	require.NoError(t, err)
	_, err = lease.Release(context.Background(), true)
	require.NoError(t, err)

	assert.Equal(t, []string{"acquire", "heartbeat", "heartbeat", "release", "release"}, actions)
	assert.Equal(t, []bool{false, false, false, true, false}, completed)
}

func TestMemoryLimiterDegradesAfterThreeDistinctBudgetExhaustionEvents(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)

	for i := 0; i < ExhaustionStrikeThreshold; i++ {
		limiter.setConcurrentSeconds("degrade", ConcurrentSecondsBudget)
		decision := limiter.acquire("degrade", "blocked-"+leaseID(i))
		assert.False(t, decision.Allowed)
		if i < ExhaustionStrikeThreshold-1 {
			clock.advance(WindowDuration + time.Second)
		}
	}

	assert.Equal(t, clock.now().Add(DegradationDuration), limiter.degradeUntil("degrade"))
	assert.Equal(t, DegradedConcurrency, limiter.effectiveConcurrencyFor("degrade"))
	assert.True(t, limiter.notificationDue("degrade"))
	assert.False(t, limiter.notificationDue("degrade"))
}

func TestMemoryLimiterDoesNotRepeatDegradationNotificationForTheSameStrikeSet(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)

	for i := 0; i < ExhaustionStrikeThreshold; i++ {
		limiter.setConcurrentSeconds("single-notice", ConcurrentSecondsBudget)
		decision := limiter.acquire("single-notice", "blocked-"+leaseID(i))
		assert.False(t, decision.Allowed)
		if i < ExhaustionStrikeThreshold-1 {
			clock.advance(WindowDuration + time.Second)
		}
	}
	assert.True(t, limiter.notificationDue("single-notice"))

	clock.advance(DegradationDuration + time.Second)
	limiter.setConcurrentSeconds("single-notice", ConcurrentSecondsBudget)
	decision := limiter.acquire("single-notice", "same-strike-window")
	assert.False(t, decision.Allowed)
	assert.False(t, decision.NotifyRootAdmin)
	assert.False(t, limiter.notificationDue("single-notice"))
}

func TestMemoryLimiterKeepsExhaustionAtTheExactStrikeWindowBoundary(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)

	limiter.setConcurrentSeconds("strike-edge", ConcurrentSecondsBudget)
	first := limiter.acquire("strike-edge", "first")
	assert.Equal(t, ReasonConcurrentSecondsLimit, first.Reason)

	clock.advance(ExhaustionStrikeWindow)
	second := limiter.acquire("strike-edge", "second")
	assert.Equal(t, ReasonConcurrentSecondsLimit, second.Reason)
	assert.Equal(t, 2, second.Snapshot.ExhaustionEvents)
}

func TestRedisLimiterFailsClosedOnlyWhenCalledForEligibleTraffic(t *testing.T) {
	limiter := New(nil)
	_, _, err := limiter.Acquire(context.Background(), "account", "lease", RiskSignals{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRedisUnavailable))
	assert.False(t, IsEligible(EligibilityInput{
		DedicatedEntitlement: false,
		DedicatedGroup:       "deepseek-v4-flash-unlimited",
		Group:                "default",
		OriginalModelName:    DeepSeekV4Flash0731Model,
	}))
}

func TestFairUseLuaContractUsesRedisTimeAndExpectedActions(t *testing.T) {
	assert.Contains(t, FairUseScript, "redis.call('TIME')")
	assert.Contains(t, FairUseScript, "acquire")
	assert.Contains(t, FairUseScript, "heartbeat")
	assert.Contains(t, FairUseScript, "release")
	assert.Contains(t, FairUseScript, "deepseek:fup:v1:{")
}

func TestFairUseLuaPrunesRollingWindowsAndExhaustionEventsWithoutRawSecrets(t *testing.T) {
	assert.Contains(t, FairUseScript, "local events_key = KEYS[10]")
	assert.Contains(t, FairUseScript, "redis.call('EXPIRE', events_key, 7200)")
	assert.NotContains(t, FairUseScript, "Authorization")
	assert.NotContains(t, FairUseScript, "client_ip")
}

func TestMemoryLimiterPeakRejectionDoesNotCreateSustainedStrike(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)

	for i := 0; i < PeakConcurrency; i++ {
		require.True(t, limiter.acquire("peak-only", leaseID(i)).Allowed)
	}

	decision := limiter.acquire("peak-only", "peak-rejected")
	assert.False(t, decision.Allowed)
	assert.Equal(t, ReasonConcurrencyLimit, decision.Reason)
	assert.Zero(t, decision.Snapshot.ExhaustionEvents)
	assert.False(t, decision.NotifyRootAdmin)
}

func TestMemoryLimiterRepeatedSameSustainedRejectionCountsOneEpisode(t *testing.T) {
	clock := newTestClock(time.Unix(1_700_000_000, 0))
	limiter := newMemoryLimiter(clock)
	limiter.setConcurrentSeconds("repeated-budget", ConcurrentSecondsBudget)

	first := limiter.acquire("repeated-budget", "first")
	second := limiter.acquire("repeated-budget", "second")

	assert.False(t, first.Allowed)
	assert.False(t, second.Allowed)
	assert.Equal(t, ReasonConcurrentSecondsLimit, first.Reason)
	assert.Equal(t, ReasonConcurrentSecondsLimit, second.Reason)
	assert.Equal(t, 1, first.Snapshot.ExhaustionEvents)
	assert.Equal(t, 1, second.Snapshot.ExhaustionEvents)
	assert.False(t, first.NotifyRootAdmin)
	assert.False(t, second.NotifyRootAdmin)
}

func TestFairUseLuaExpiresEveryRiskDimension(t *testing.T) {
	assert.Contains(t, FairUseScript, "redis.call('EXPIRE', risk_ip_key, 172800)")
	assert.Contains(t, FairUseScript, "redis.call('EXPIRE', risk_country_key, 172800)")
	assert.Contains(t, FairUseScript, "redis.call('EXPIRE', risk_ua_key, 172800)")
	assert.Contains(t, FairUseScript, "local ip_count = redis.call('ZCARD', risk_ip_key)")
	assert.Contains(t, FairUseScript, "local country_count = redis.call('ZCARD', risk_country_key)")
	assert.Contains(t, FairUseScript, "last_notified")
	assert.Contains(t, FairUseScript, "record_exhaustion(reason)\n    expire_all()")
}

func TestFairUseLuaKeepsEventsAtTheExactRollingWindowBoundary(t *testing.T) {
	assert.Contains(t, FairUseScript, "ZREMRANGEBYSCORE', total_key, '-inf', '(' .. (now_ms - WINDOW_MS)")
	assert.Contains(t, FairUseScript, "ZREMRANGEBYSCORE', success_key, '-inf', '(' .. (now_ms - WINDOW_MS)")
	assert.Contains(t, FairUseScript, "ZREMRANGEBYSCORE', events_key, '-inf', '(' .. (now_ms - STRIKE_WINDOW_MS)")
}

func TestFairUseLuaTracksRollingOccupancyAsExactIntervals(t *testing.T) {
	assert.Contains(t, FairUseScript, "redis.call('ZADD', window_key")
	assert.Contains(t, FairUseScript, "string.match(member, '^(%d+)|(%d+)|')")
	assert.NotContains(t, FairUseScript, "redis.call('HINCRBYFLOAT', window_key")
	assert.NotContains(t, FairUseScript, "local minimum_bucket = math.floor((now_ms - WINDOW_MS) / BUCKET_MS)")
}

func TestRedisLimiterPropagatesOpenAIRelevantDecision(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	limiter := New(client)
	_, _, err := limiter.Acquire(context.Background(), "account", "lease", RiskSignals{})
	assert.Error(t, err)
}

func TestDecodeDecisionAcceptsLuaFloatingPointConcurrentSeconds(t *testing.T) {
	decision, err := decodeDecision([]interface{}{
		int64(1), "", int64(1), "3.5", int64(2), int64(1), int64(PeakConcurrency), int64(0), int64(0), int64(0), int64(0),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), decision.Snapshot.ConcurrentSeconds)
}

func TestDecodeDecisionRejectsCorruptNumericFields(t *testing.T) {
	_, err := decodeDecision([]interface{}{
		int64(1), "", "not-a-number", int64(3), int64(2), int64(1), int64(PeakConcurrency), int64(0), int64(0), int64(0), int64(0),
	})
	assert.Error(t, err)
}

func TestLeaseHeartbeatReturnsErrorWhenRedisReportsStaleLease(t *testing.T) {
	limiter := &Limiter{
		runOverride: func(context.Context, string, string, string, RiskSignals, bool) (Decision, error) {
			return Decision{Allowed: false, Reason: ReasonLeaseNotFound}, nil
		},
	}
	lease := &Lease{limiter: limiter, account: "account", id: "lease"}

	err := lease.Heartbeat(context.Background())
	assert.ErrorIs(t, err, ErrLeaseNotFound)
}

func TestLeaseHeartbeatStopsImmediatelyWhenRequestContextIsCanceled(t *testing.T) {
	actions := make([]string, 0, 2)
	limiter := &Limiter{
		runOverride: func(_ context.Context, action, _, _ string, _ RiskSignals, _ bool) (Decision, error) {
			actions = append(actions, action)
			return Decision{Allowed: true}, nil
		},
	}
	lease, _, err := limiter.Acquire(context.Background(), "account", "lease", RiskSignals{})
	require.NoError(t, err)

	requestContext, cancel := context.WithCancel(context.Background())
	stop := lease.StartHeartbeat(requestContext, func(error) { t.Error("heartbeat callback must not run after cancellation") })
	cancel()
	stop()

	assert.Equal(t, []string{"acquire"}, actions)
}
