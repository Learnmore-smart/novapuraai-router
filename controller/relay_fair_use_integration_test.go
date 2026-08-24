package controller

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/deepseekfairuse"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newControllerFairUseRedisFixture(t *testing.T) (*redis.Client, []string, string) {
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
			t.Skip("configured Redis connection string is unavailable for the controller fair-use integration suite")
		}
	} else if address != "" {
		options = &redis.Options{Addr: address, Password: os.Getenv("DEEPSEEK_FUP_REDIS_PASSWORD")}
	} else {
		t.Skip("set a FUP Redis connection environment variable to run the controller fair-use integration test")
	}

	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := client.Ping(ctx).Err(); err != nil {
		cancel()
		_ = client.Close()
		t.Skip("configured Redis endpoint is unavailable for the controller fair-use integration suite")
	}
	cancel()

	account := fmt.Sprintf("controller-integration-%x", time.Now().UnixNano())
	keys := controllerFairUseAccountKeys(account)
	cleanupCtx := context.Background()
	require.NoError(t, client.Del(cleanupCtx, keys...).Err())
	t.Cleanup(func() {
		_ = client.Del(context.Background(), keys...).Err()
		_ = client.Close()
	})
	return client, keys, account
}

func controllerFairUseAccountKeys(account string) []string {
	prefix := "deepseek:fup:v1:{" + account + "}"
	return []string{
		prefix + ":live",
		prefix + ":lease_expiry",
		prefix + ":window",
		prefix + ":total",
		prefix + ":success",
		prefix + ":penalty",
		prefix + ":risk_ip",
		prefix + ":risk_country",
		prefix + ":risk_ua",
		prefix + ":penalty:events",
	}
}

func TestBeginDeepSeekFairUseNotifiesOnceWithDerivedPayloads(t *testing.T) {
	client, _, _ := newControllerFairUseRedisFixture(t)
	ctx := context.Background()
	secret := "controller-fup-integration-secret"
	identifiers := deepseekfairuse.BuildIdentifiers([]byte(secret), deepseekfairuse.IdentityInput{
		UserID:    77,
		TokenID:   88,
		ClientIP:  "203.0.113.88",
		Country:   "CA",
		UserAgent: "controller-fup-raw-agent/1",
	})
	keys := controllerFairUseAccountKeys(identifiers.AccountHMAC)
	require.NoError(t, client.Del(ctx, keys...).Err())
	t.Cleanup(func() { _ = client.Del(context.Background(), keys...).Err() })
	originalRedis, originalEnabled := common.RDB, common.RedisEnabled
	originalSecret := common.CryptoSecret
	originalNotify := notifyDeepSeekFairUseAdmin
	t.Cleanup(func() {
		common.RDB = originalRedis
		common.RedisEnabled = originalEnabled
		common.CryptoSecret = originalSecret
		notifyDeepSeekFairUseAdmin = originalNotify
	})
	common.RDB = client
	common.RedisEnabled = true
	common.CryptoSecret = secret

	type notification struct {
		account  string
		snapshot deepseekfairuse.Snapshot
	}
	notifications := make([]notification, 0, 2)
	notifyDeepSeekFairUseAdmin = func(accountHMAC string, snapshot deepseekfairuse.Snapshot) {
		notifications = append(notifications, notification{account: accountHMAC, snapshot: snapshot})
	}

	gin.SetMode(gin.TestMode)
	newContext := func() *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"deepseek-v4-flash-0731"}`))
		c.Request.Header.Set("Authorization", "Bearer sk-controller-fup-raw-token")
		c.Request.Header.Set("User-Agent", "controller-fup-raw-agent/1")
		c.Request.Header.Set("CF-IPCountry", "CA")
		return c
	}
	info := &relaycommon.RelayInfo{
		UserId:          77,
		TokenId:         88,
		TokenGroup:      deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
		TokenUnlimited:  true,
		OriginModelName: deepseekfairuse.DeepSeekV4Flash0731Model,
	}

	invokeBudgetRejection := func() *types.NewAPIError {
		session, apiErr := beginDeepSeekFairUse(newContext(), info, types.RelayFormatOpenAI)
		assert.Nil(t, session)
		require.NotNil(t, apiErr)
		assert.Equal(t, deepSeekFairUseLimitErrorCode, apiErr.GetErrorCode())
		return apiErr
	}

	now, err := client.Time(ctx).Result()
	require.NoError(t, err)
	nowMS := now.UnixMilli()
	successes := make([]*redis.Z, deepseekfairuse.SuccessRequestLimit)
	for i := range successes {
		successes[i] = &redis.Z{Score: float64(nowMS - 1000), Member: fmt.Sprintf("success-%d", i)}
	}
	require.NoError(t, client.ZAdd(ctx, keys[4], successes...).Err())
	firstErr := invokeBudgetRejection()
	assert.Equal(t, int64(1), redisSortedSetLength(t, client, keys[9]))
	require.NoError(t, client.Del(ctx, keys[:9]...).Err())

	totalMembers := make([]*redis.Z, deepseekfairuse.AdmittedRequestLimit)
	for i := range totalMembers {
		totalMembers[i] = &redis.Z{Score: float64(nowMS), Member: fmt.Sprintf("admitted-%d", i)}
	}
	require.NoError(t, client.ZAdd(ctx, keys[3], totalMembers...).Err())
	secondErr := invokeBudgetRejection()
	assert.Equal(t, int64(2), redisSortedSetLength(t, client, keys[9]))
	require.NoError(t, client.Del(ctx, keys[:9]...).Err())

	occupancyNow, err := client.Time(ctx).Result()
	require.NoError(t, err)
	occupancyMS := occupancyNow.UnixMilli()
	occupancyMembers := []*redis.Z{
		{Score: float64(occupancyMS), Member: fmt.Sprintf("%d|%d|occupancy-long-1", occupancyMS-deepseekfairuse.WindowDuration.Milliseconds()+1000, occupancyMS)},
		{Score: float64(occupancyMS), Member: fmt.Sprintf("%d|%d|occupancy-long-2", occupancyMS-deepseekfairuse.WindowDuration.Milliseconds()+1000, occupancyMS)},
		{Score: float64(occupancyMS), Member: fmt.Sprintf("%d|%d|occupancy-long-3", occupancyMS-deepseekfairuse.WindowDuration.Milliseconds()+1000, occupancyMS)},
		{Score: float64(occupancyMS), Member: fmt.Sprintf("%d|%d|occupancy-partial", occupancyMS-30000, occupancyMS)},
	}
	require.NoError(t, client.ZAdd(ctx, keys[2], occupancyMembers...).Err())
	thirdErr := invokeBudgetRejection()
	assert.Len(t, notifications, 1)
	assert.NotEmpty(t, notifications[0].account)
	assert.Equal(t, identifiers.AccountHMAC, notifications[0].account)
	assert.NotEqual(t, identifiers.RiskHMAC, notifications[0].account)
	assert.Equal(t, 3, notifications[0].snapshot.ExhaustionEvents)
	assert.True(t, notifications[0].snapshot.Degraded)

	repeatErr := invokeBudgetRejection()
	assert.Len(t, notifications, 1)

	for _, payload := range []string{notifications[0].account, firstErr.Error(), secondErr.Error(), thirdErr.Error(), repeatErr.Error()} {
		assert.NotContains(t, payload, "sk-controller-fup-raw-token")
		assert.NotContains(t, payload, "Bearer sk-controller-fup-raw-token")
		assert.NotContains(t, payload, "203.0.113.88")
		assert.NotContains(t, payload, "controller-fup-raw-agent")
	}
	assert.NotContains(t, identifiers.AccountHMAC, "203.0.113.88")
}

func TestDeepSeekFairUseControllerSessionDefersReleaseAndSettlementToCompletion(t *testing.T) {
	client, _, _ := newControllerFairUseRedisFixture(t)
	billingFixture := newControllerFairUseBillingFixture(t)
	secret := "controller-fup-lifecycle-secret"
	ctx := context.Background()

	originalRedis, originalEnabled := common.RDB, common.RedisEnabled
	originalSecret := common.CryptoSecret
	t.Cleanup(func() {
		common.RDB = originalRedis
		common.RedisEnabled = originalEnabled
		common.CryptoSecret = originalSecret
	})
	common.RDB = client
	common.RedisEnabled = true
	common.CryptoSecret = secret

	newContext := func() *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"deepseek-v4-flash-0731"}`))
		c.Request.Header.Set("User-Agent", "controller-fup-lifecycle-agent")
		return c
	}
	newInfo := func(userID int) (*relaycommon.RelayInfo, *gin.Context) {
		info, billingContext := billingFixture.newBillingSession(t, userID)
		info.TokenId = userID + 1000
		info.TokenGroup = deepseekfairuse.DeepSeekV4FlashDedicatedGroup
		info.TokenUnlimited = true
		info.OriginModelName = deepseekfairuse.DeepSeekV4Flash0731Model
		return info, billingContext
	}

	failedInfo, failedBillingContext := newInfo(91)
	failedIdentifiers := deepseekfairuse.BuildIdentifiers([]byte(secret), deepseekfairuse.IdentityInput{UserID: failedInfo.UserId})
	failedKeys := controllerFairUseAccountKeys(failedIdentifiers.AccountHMAC)
	require.NoError(t, client.Del(ctx, failedKeys...).Err())
	t.Cleanup(func() { _ = client.Del(context.Background(), failedKeys...).Err() })
	failedSession, failedBeginErr := beginDeepSeekFairUse(newContext(), failedInfo, types.RelayFormatOpenAI)
	require.NoError(t, failedBeginErr)
	require.NotNil(t, failedSession)
	assert.Equal(t, int64(1), redisHashLength(t, client, failedKeys[0]))
	failedAPIError := types.NewErrorWithStatusCode(assert.AnError, types.ErrorCodeUpstreamTimeout, 500)
	billingFixture.armRefundWait()
	failedInfo.Billing.Refund(failedBillingContext)
	failedSession.finish(failedBillingContext, failedInfo, &failedAPIError)
	billingFixture.waitForRefund(t)
	assert.False(t, failedInfo.Billing.NeedsRefund())
	assert.Equal(t, int64(0), redisHashLength(t, client, failedKeys[0]))
	assert.Equal(t, int64(0), redisSortedSetLength(t, client, failedKeys[4]))

	successInfo, successBillingContext := newInfo(92)
	successIdentifiers := deepseekfairuse.BuildIdentifiers([]byte(secret), deepseekfairuse.IdentityInput{UserID: successInfo.UserId})
	successKeys := controllerFairUseAccountKeys(successIdentifiers.AccountHMAC)
	require.NoError(t, client.Del(ctx, successKeys...).Err())
	t.Cleanup(func() { _ = client.Del(context.Background(), successKeys...).Err() })
	successSession, successBeginErr := beginDeepSeekFairUse(newContext(), successInfo, types.RelayFormatOpenAI)
	require.NoError(t, successBeginErr)
	require.NotNil(t, successSession)
	assert.Equal(t, int64(1), redisHashLength(t, client, successKeys[0]))
	initialQuota := billingFixture.quota(t, successInfo.UserId)
	require.NoError(t, service.SettleBilling(successBillingContext, successInfo, 12))
	assert.Equal(t, initialQuota-2, billingFixture.quota(t, successInfo.UserId))
	assert.False(t, successInfo.Billing.NeedsRefund())
	var successAPIError *types.NewAPIError
	successSession.finish(successBillingContext, successInfo, &successAPIError)
	assert.Equal(t, int64(0), redisHashLength(t, client, successKeys[0]))
	assert.Equal(t, int64(1), redisSortedSetLength(t, client, successKeys[4]))

	// finishOnce makes a deferred cleanup safe when the relay's outer error path
	// and request cleanup both reach the session.
	successSession.finish(successBillingContext, successInfo, &successAPIError)
	assert.Equal(t, int64(1), redisSortedSetLength(t, client, successKeys[4]))
}

func TestDeepSeekFairUseControllerRetryReusesOneLease(t *testing.T) {
	client, _, _ := newControllerFairUseRedisFixture(t)
	billingFixture := newControllerFairUseBillingFixture(t)
	secret := "controller-fup-retry-lifecycle-secret"
	originalRedis, originalEnabled := common.RDB, common.RedisEnabled
	originalSecret := common.CryptoSecret
	t.Cleanup(func() {
		common.RDB = originalRedis
		common.RedisEnabled = originalEnabled
		common.CryptoSecret = originalSecret
	})
	common.RDB = client
	common.RedisEnabled = true
	common.CryptoSecret = secret

	info, billingContext := billingFixture.newBillingSession(t, 1001)
	info.TokenGroup = deepseekfairuse.DeepSeekV4FlashDedicatedGroup
	info.TokenUnlimited = true
	info.OriginModelName = deepseekfairuse.DeepSeekV4Flash0731Model
	identifiers := deepseekfairuse.BuildIdentifiers([]byte(secret), deepseekfairuse.IdentityInput{UserID: info.UserId})
	keys := controllerFairUseAccountKeys(identifiers.AccountHMAC)
	require.NoError(t, client.Del(context.Background(), keys...).Err())
	t.Cleanup(func() { _ = client.Del(context.Background(), keys...).Err() })

	session, beginErr := beginDeepSeekFairUse(newControllerLifecycleContext(), info, types.RelayFormatOpenAI)
	require.NoError(t, beginErr)
	require.NotNil(t, session)
	assert.Equal(t, int64(1), redisHashLength(t, client, keys[0]))
	assert.Equal(t, int64(1), redisSortedSetLength(t, client, keys[3]))

	upstreamErr := types.NewErrorWithStatusCode(assert.AnError, types.ErrorCodeUpstreamTimeout, 500)
	assert.True(t, shouldRetry(newControllerLifecycleContext(), upstreamErr, 1))
	// The retry is another upstream attempt within the same controller request;
	// heartbeat the existing lease instead of admitting a second one.
	require.NoError(t, session.lease.Heartbeat(context.Background()))
	assert.Equal(t, int64(1), redisHashLength(t, client, keys[0]))
	assert.Equal(t, int64(1), redisSortedSetLength(t, client, keys[3]))

	require.NoError(t, service.SettleBilling(billingContext, info, 10))
	var apiErr *types.NewAPIError
	session.finish(billingContext, info, &apiErr)
	assert.Equal(t, int64(0), redisHashLength(t, client, keys[0]))
	assert.Equal(t, int64(1), redisSortedSetLength(t, client, keys[4]))
}

func TestDeepSeekFairUseControllerHeartbeatFailureCancelsAndRefunds(t *testing.T) {
	client, _, _ := newControllerFairUseRedisFixture(t)
	billingFixture := newControllerFairUseBillingFixture(t)
	secret := "controller-fup-heartbeat-lifecycle-secret"
	originalRedis, originalEnabled := common.RDB, common.RedisEnabled
	originalSecret := common.CryptoSecret
	t.Cleanup(func() {
		common.RDB = originalRedis
		common.RedisEnabled = originalEnabled
		common.CryptoSecret = originalSecret
	})
	common.RDB = client
	common.RedisEnabled = true
	common.CryptoSecret = secret

	info, billingContext := billingFixture.newBillingSession(t, 1101)
	info.TokenGroup = deepseekfairuse.DeepSeekV4FlashDedicatedGroup
	info.TokenUnlimited = true
	info.OriginModelName = deepseekfairuse.DeepSeekV4Flash0731Model
	identifiers := deepseekfairuse.BuildIdentifiers([]byte(secret), deepseekfairuse.IdentityInput{UserID: info.UserId})
	keys := controllerFairUseAccountKeys(identifiers.AccountHMAC)
	require.NoError(t, client.Del(context.Background(), keys...).Err())
	t.Cleanup(func() { _ = client.Del(context.Background(), keys...).Err() })

	session, beginErr := beginDeepSeekFairUse(newControllerLifecycleContext(), info, types.RelayFormatOpenAI)
	require.NoError(t, beginErr)
	require.NotNil(t, session)
	require.NoError(t, client.Del(context.Background(), keys[0], keys[1]).Err())
	heartbeatErr := session.lease.Heartbeat(context.Background())
	require.ErrorIs(t, heartbeatErr, deepseekfairuse.ErrLeaseNotFound)
	session.heartbeatMu.Lock()
	session.heartbeatErr = heartbeatErr
	session.heartbeatMu.Unlock()

	var apiErr *types.NewAPIError
	billingFixture.armRefundWait()
	session.finish(billingContext, info, &apiErr)
	billingFixture.waitForRefund(t)
	require.NotNil(t, apiErr)
	assert.Equal(t, deepSeekFairUseRedisErrorCode, apiErr.GetErrorCode())
	assert.False(t, info.Billing.NeedsRefund())
}

func TestDeepSeekFairUseControllerSessionTerminatesEveryStreamOutcomeWithOneLease(t *testing.T) {
	client, _, _ := newControllerFairUseRedisFixture(t)
	billingFixture := newControllerFairUseBillingFixture(t)
	secret := "controller-fup-stream-lifecycle-secret"
	originalRedis, originalEnabled := common.RDB, common.RedisEnabled
	originalSecret := common.CryptoSecret
	t.Cleanup(func() {
		common.RDB = originalRedis
		common.RedisEnabled = originalEnabled
		common.CryptoSecret = originalSecret
	})
	common.RDB = client
	common.RedisEnabled = true
	common.CryptoSecret = secret

	tests := []struct {
		name       string
		reason     relaycommon.StreamEndReason
		endError   error
		softErrors bool
		completed  bool
	}{
		{name: "upstream failure", reason: relaycommon.StreamEndReasonPanic, endError: assert.AnError},
		{name: "timeout", reason: relaycommon.StreamEndReasonTimeout},
		{name: "scanner error", reason: relaycommon.StreamEndReasonScannerErr, endError: assert.AnError},
		{name: "ping failure", reason: relaycommon.StreamEndReasonPingFail, endError: assert.AnError},
		{name: "client gone", reason: relaycommon.StreamEndReasonClientGone, endError: assert.AnError},
		{name: "eof", reason: relaycommon.StreamEndReasonEOF, completed: true},
		{name: "handler stop", reason: relaycommon.StreamEndReasonHandlerStop, completed: true},
		{name: "handler stop with error", reason: relaycommon.StreamEndReasonHandlerStop, endError: assert.AnError},
		{name: "done with soft error", reason: relaycommon.StreamEndReasonDone, softErrors: true},
	}

	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := 1200 + index
			info, billingContext := billingFixture.newBillingSession(t, userID)
			info.TokenGroup = deepseekfairuse.DeepSeekV4FlashDedicatedGroup
			info.TokenUnlimited = true
			info.OriginModelName = deepseekfairuse.DeepSeekV4Flash0731Model
			info.IsStream = true
			status := relaycommon.NewStreamStatus()
			status.SetEndReason(tt.reason, tt.endError)
			if tt.softErrors {
				status.RecordError("upstream event error")
			}
			info.StreamStatus = status
			identifiers := deepseekfairuse.BuildIdentifiers([]byte(secret), deepseekfairuse.IdentityInput{UserID: userID})
			keys := controllerFairUseAccountKeys(identifiers.AccountHMAC)
			require.NoError(t, client.Del(context.Background(), keys...).Err())
			t.Cleanup(func() { _ = client.Del(context.Background(), keys...).Err() })

			session, beginErr := beginDeepSeekFairUse(newControllerLifecycleContext(), info, types.RelayFormatOpenAI)
			require.NoError(t, beginErr)
			require.NotNil(t, session)
			if tt.completed {
				require.NoError(t, service.SettleBilling(billingContext, info, 10))
			} else {
				billingFixture.armRefundWait()
				info.Billing.Refund(billingContext)
			}
			var apiErr *types.NewAPIError
			if !tt.completed {
				apiErr = types.NewErrorWithStatusCode(assert.AnError, types.ErrorCodeUpstreamTimeout, 500)
			}
			session.finish(billingContext, info, &apiErr)
			if !tt.completed {
				billingFixture.waitForRefund(t)
			}
			assert.Equal(t, int64(0), redisHashLength(t, client, keys[0]))
			wantSuccesses := int64(0)
			if tt.completed {
				wantSuccesses = 1
			}
			assert.Equal(t, wantSuccesses, redisSortedSetLength(t, client, keys[4]))
			assert.False(t, info.Billing.NeedsRefund())
		})
	}
}

func newControllerLifecycleContext() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"deepseek-v4-flash-0731"}`))
	c.Request.Header.Set("User-Agent", "controller-fup-lifecycle-agent")
	return c
}

func redisHashLength(t *testing.T, client *redis.Client, key string) int64 {
	t.Helper()
	length, err := client.HLen(context.Background(), key).Result()
	require.NoError(t, err)
	return length
}

func redisSortedSetLength(t *testing.T, client *redis.Client, key string) int64 {
	t.Helper()
	length, err := client.ZCard(context.Background(), key).Result()
	require.NoError(t, err)
	return length
}
