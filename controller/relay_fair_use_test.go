package controller

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/deepseekfairuse"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestIsDeepSeekFairUseEligibleRequiresRecurringGroupAndModel(t *testing.T) {
	tests := []struct {
		name       string
		format     types.RelayFormat
		userGroup  string
		tokenGroup string
		free       bool
		model      string
		want       bool
	}{
		{
			name:      "openai chat",
			format:    types.RelayFormatOpenAI,
			userGroup: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:      true,
			model:     deepseekfairuse.DeepSeekV4Flash0731Model,
			want:      true,
		},
		{
			name:      "openai responses",
			format:    types.RelayFormatOpenAIResponses,
			userGroup: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:      true,
			model:     deepseekfairuse.DeepSeekV4Flash0731Model,
			want:      true,
		},
		{
			name:      "openai responses compact exact upstream model",
			format:    types.RelayFormatOpenAIResponsesCompaction,
			userGroup: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:      true,
			model:     deepseekfairuse.DeepSeekV4Flash0731Model + ratio_setting.CompactModelSuffix,
			want:      true,
		},
		{
			name:      "claude format",
			format:    types.RelayFormatClaude,
			userGroup: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:      true,
			model:     deepseekfairuse.DeepSeekV4Flash0731Model,
			want:      true,
		},
		{
			name:       "explicit ordinary token group cannot bypass account entitlement",
			format:     types.RelayFormatOpenAI,
			userGroup:  deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			tokenGroup: "default",
			free:       false,
			model:      deepseekfairuse.DeepSeekV4Flash0731Model,
			want:       true,
		},
		{
			name:      "ordinary account",
			format:    types.RelayFormatOpenAI,
			userGroup: "default",
			free:      true,
			model:     deepseekfairuse.DeepSeekV4Flash0731Model,
			want:      false,
		},
		{
			name:      "different platform model",
			format:    types.RelayFormatOpenAI,
			userGroup: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:      true,
			model:     "gpt-5",
			want:      true,
		},
		{
			name:      "missing model",
			format:    types.RelayFormatOpenAI,
			userGroup: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:      true,
			model:     " ",
			want:      false,
		},
		{
			name:      "finite token still uses account fair use",
			format:    types.RelayFormatOpenAI,
			userGroup: deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
			free:      false,
			model:     deepseekfairuse.DeepSeekV4Flash0731Model,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				UserGroup:       tt.userGroup,
				TokenGroup:      tt.tokenGroup,
				TokenUnlimited:  tt.free,
				OriginModelName: tt.model,
			}
			assert.Equal(t, tt.want, isDeepSeekFairUseEligible(info, tt.format))
		})
	}
}

func TestDeepSeekFairUseRedisFailureIsEligibleOnly(t *testing.T) {
	originalRedis := common.RDB
	t.Cleanup(func() { common.RDB = originalRedis })
	common.RDB = nil

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	ctx.Request.Header.Set("User-Agent", "test-client/1")
	info := &relaycommon.RelayInfo{
		UserId:          7,
		TokenId:         9,
		UserGroup:       deepseekfairuse.DeepSeekV4FlashDedicatedGroup,
		TokenUnlimited:  true,
		OriginModelName: deepseekfairuse.DeepSeekV4Flash0731Model,
	}

	session, apiErr := beginDeepSeekFairUse(ctx, info, types.RelayFormatOpenAI)
	assert.Nil(t, session)
	require.NotNil(t, apiErr)
	assert.Equal(t, 503, apiErr.StatusCode)
	assert.Equal(t, deepSeekFairUseRedisErrorCode, apiErr.GetErrorCode())
	assert.True(t, types.IsSkipRetryError(apiErr))

	normalInfo := *info
	normalInfo.UserGroup = "default"
	session, apiErr = beginDeepSeekFairUse(ctx, &normalInfo, types.RelayFormatOpenAI)
	assert.Nil(t, session)
	assert.Nil(t, apiErr)
}

func TestDeepSeekFairUseErrorsSkipRetryAndCooldown(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	for _, apiErr := range []*types.NewAPIError{newDeepSeekFairUseLimitError(), newDeepSeekFairUseRedisError()} {
		assert.True(t, isDeepSeekFairUseError(apiErr))
		assert.True(t, types.IsSkipRetryError(apiErr))
		assert.False(t, shouldRetry(ctx, apiErr, 3))
	}
	assert.False(t, isDeepSeekFairUseError(types.NewErrorWithStatusCode(
		assert.AnError,
		types.ErrorCodeBadResponseStatusCode,
		429,
	)))
}

func TestHeartbeatRedisFailurePromotesAnUncommittedResponseTo503(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	upstreamErr := types.NewErrorWithStatusCode(assert.AnError, types.ErrorCodeUpstreamTimeout, 500)
	assert.False(t, isDeepSeekFairUseError(upstreamErr))
	apiErr := upstreamErr
	promoteDeepSeekFairUseRedisError(ctx, &relaycommon.RelayInfo{}, &apiErr, true)
	assert.Equal(t, 503, apiErr.StatusCode)
	assert.Equal(t, deepSeekFairUseRedisErrorCode, apiErr.GetErrorCode())
}

func TestDeepSeekFairUseCompletionRequiresNormalStreamEnd(t *testing.T) {
	assert.True(t, deepSeekFairUseCompleted(&relaycommon.RelayInfo{}))

	done := relaycommon.NewStreamStatus()
	done.SetEndReason(relaycommon.StreamEndReasonDone, nil)
	assert.True(t, deepSeekFairUseCompleted(&relaycommon.RelayInfo{IsStream: true, StreamStatus: done}))

	clientGone := relaycommon.NewStreamStatus()
	clientGone.SetEndReason(relaycommon.StreamEndReasonClientGone, assert.AnError)
	assert.False(t, deepSeekFairUseCompleted(&relaycommon.RelayInfo{IsStream: true, StreamStatus: clientGone}))

	withSoftError := relaycommon.NewStreamStatus()
	withSoftError.SetEndReason(relaycommon.StreamEndReasonDone, nil)
	withSoftError.RecordError("upstream event error")
	assert.False(t, deepSeekFairUseCompleted(&relaycommon.RelayInfo{IsStream: true, StreamStatus: withSoftError}))
	assert.False(t, deepSeekFairUseCompleted(&relaycommon.RelayInfo{IsStream: true}))
}

func TestDeepSeekFairUseCompletionHandlesEveryStreamTermination(t *testing.T) {
	tests := []struct {
		name       string
		reason     relaycommon.StreamEndReason
		endError   error
		softErrors bool
		want       bool
	}{
		{name: "upstream failure", reason: relaycommon.StreamEndReasonPanic, endError: assert.AnError},
		{name: "timeout", reason: relaycommon.StreamEndReasonTimeout},
		{name: "scanner error", reason: relaycommon.StreamEndReasonScannerErr, endError: assert.AnError},
		{name: "ping failure", reason: relaycommon.StreamEndReasonPingFail, endError: assert.AnError},
		{name: "client eof", reason: relaycommon.StreamEndReasonEOF, want: true},
		{name: "handler stop without error", reason: relaycommon.StreamEndReasonHandlerStop, want: true},
		{name: "handler stop with error", reason: relaycommon.StreamEndReasonHandlerStop, endError: assert.AnError},
		{name: "done with a soft error", reason: relaycommon.StreamEndReasonDone, softErrors: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := relaycommon.NewStreamStatus()
			status.SetEndReason(tt.reason, tt.endError)
			if tt.softErrors {
				status.RecordError("upstream event error")
			}
			assert.Equal(t, tt.want, deepSeekFairUseCompleted(&relaycommon.RelayInfo{
				IsStream:     true,
				StreamStatus: status,
			}))
		})
	}
}

type controllerFairUseBillingFixture struct {
	db                   *gorm.DB
	originalDB           *gorm.DB
	originalLogDB        *gorm.DB
	originalMainDBType   common.DatabaseType
	originalLogDBType    common.DatabaseType
	originalRedisEnabled bool
	originalBatchUpdates bool
	refundMu             sync.Mutex
	refundWait           chan struct{}
}

func newControllerFairUseBillingFixture(t *testing.T) *controllerFairUseBillingFixture {
	t.Helper()
	fixture := &controllerFairUseBillingFixture{
		originalDB:           model.DB,
		originalLogDB:        model.LOG_DB,
		originalMainDBType:   common.MainDatabaseType(),
		originalLogDBType:    common.LogDatabaseType(),
		originalRedisEnabled: common.RedisEnabled,
		originalBatchUpdates: common.BatchUpdateEnabled,
	}
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:controller-fup-billing-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	fixture.db = db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.BalanceCreditLot{}, &model.BalanceLedger{}))
	require.NoError(t, db.Callback().Update().After("gorm:commit_or_rollback_transaction").Register("controller_fup_refund_observer", func(tx *gorm.DB) {
		if tx.Statement.Table != "users" {
			return
		}
		fixture.refundMu.Lock()
		defer fixture.refundMu.Unlock()
		if fixture.refundWait != nil {
			select {
			case fixture.refundWait <- struct{}{}:
			default:
			}
		}
	}))
	t.Cleanup(func() {
		model.DB = fixture.originalDB
		model.LOG_DB = fixture.originalLogDB
		common.SetDatabaseTypes(fixture.originalMainDBType, fixture.originalLogDBType)
		common.RedisEnabled = fixture.originalRedisEnabled
		common.BatchUpdateEnabled = fixture.originalBatchUpdates
		_ = sqlDB.Close()
	})
	return fixture
}

func (f *controllerFairUseBillingFixture) newBillingSession(t *testing.T, userID int) (*relaycommon.RelayInfo, *gin.Context) {
	t.Helper()
	user := &model.User{
		Id:       userID,
		Username: fmt.Sprintf("controller_fup_billing_%d", userID),
		Quota:    1_000_000_000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, f.db.Create(user).Error)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		UserId:          userID,
		UserQuota:       user.Quota,
		OriginModelName: deepseekfairuse.DeepSeekV4Flash0731Model,
		IsPlayground:    true,
		ForcePreConsume: true,
		UserSetting:     dto.UserSetting{BillingPreference: "wallet_only"},
	}
	require.Nil(t, service.PreConsumeBilling(c, 10, info))
	require.NotNil(t, info.Billing)
	return info, c
}

func (f *controllerFairUseBillingFixture) quota(t *testing.T, userID int) int {
	t.Helper()
	var user model.User
	require.NoError(t, f.db.Select("quota").First(&user, userID).Error)
	return user.Quota
}

func (f *controllerFairUseBillingFixture) armRefundWait() {
	f.refundMu.Lock()
	defer f.refundMu.Unlock()
	f.refundWait = make(chan struct{}, 1)
}

func (f *controllerFairUseBillingFixture) waitForRefund(t *testing.T) {
	t.Helper()
	f.refundMu.Lock()
	wait := f.refundWait
	f.refundMu.Unlock()
	require.NotNil(t, wait)
	waitContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	select {
	case <-wait:
		f.refundMu.Lock()
		f.refundWait = nil
		f.refundMu.Unlock()
	case <-waitContext.Done():
		t.Fatal("actual BillingSession refund did not commit")
	}
}

func TestDeepSeekFairUseControllerSettlesActualBillingSessionOnCompletion(t *testing.T) {
	f := newControllerFairUseBillingFixture(t)
	info, c := f.newBillingSession(t, 901)
	initialQuota := f.quota(t, info.UserId)

	require.NoError(t, service.SettleBilling(c, info, 12))
	assert.Equal(t, initialQuota-2, f.quota(t, info.UserId), "settlement should apply only the post-pre-consume delta")
	assert.False(t, info.Billing.NeedsRefund())

	// A deferred completion path must not settle the same BillingSession twice.
	require.NoError(t, service.SettleBilling(c, info, 12))
	assert.Equal(t, initialQuota-2, f.quota(t, info.UserId))
}

func TestDeepSeekFairUseControllerRefundsActualBillingSessionOnFailure(t *testing.T) {
	f := newControllerFairUseBillingFixture(t)
	info, c := f.newBillingSession(t, 902)
	preConsumedQuota := f.quota(t, info.UserId)

	f.armRefundWait()
	info.Billing.Refund(c)
	f.waitForRefund(t)

	assert.Equal(t, preConsumedQuota+10, f.quota(t, info.UserId))
	assert.False(t, info.Billing.NeedsRefund())
	info.Billing.Refund(c)
}

type controllerFairUseBillingProbe struct {
	settleCalls []int
	refunds     int
}

func (p *controllerFairUseBillingProbe) Settle(actualQuota int) error {
	p.settleCalls = append(p.settleCalls, actualQuota)
	return nil
}

func (p *controllerFairUseBillingProbe) Refund(*gin.Context) { p.refunds++ }

func (p *controllerFairUseBillingProbe) NeedsRefund() bool { return p.refunds == 0 }

func (p *controllerFairUseBillingProbe) GetPreConsumedQuota() int { return 10 }

func (p *controllerFairUseBillingProbe) Reserve(int) error { return nil }

func TestDeepSeekFairUseHeartbeatFailureUsesControllerBillingRefundSeam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	probe := &controllerFairUseBillingProbe{}
	info := &relaycommon.RelayInfo{Billing: probe}
	apiErr := (*types.NewAPIError)(nil)

	promoteDeepSeekFairUseRedisError(ctx, info, &apiErr, true)

	require.NotNil(t, apiErr)
	assert.Equal(t, 503, apiErr.StatusCode)
	assert.Equal(t, 1, probe.refunds)
	assert.Empty(t, probe.settleCalls, "heartbeat failure must not settle billing")
	assert.False(t, probe.NeedsRefund())

	committedCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, writeErr := committedCtx.Writer.Write([]byte("already committed"))
	require.NoError(t, writeErr)
	secondErr := (*types.NewAPIError)(nil)
	promoteDeepSeekFairUseRedisError(committedCtx, info, &secondErr, true)
	assert.Nil(t, secondErr)
	assert.Equal(t, 1, probe.refunds)
}

func TestRecordDeepSeekFairUseAuditContainsOnlyDerivedValues(t *testing.T) {
	identifiers := deepseekfairuse.BuildIdentifiers([]byte("audit-test-secret"), deepseekfairuse.IdentityInput{
		UserID:    11,
		TokenID:   12,
		ClientIP:  "203.0.113.17",
		Country:   "CA",
		UserAgent: "curl/8.0",
	})
	info := &relaycommon.RelayInfo{}
	recordDeepSeekFairUseAudit(info, identifiers, deepseekfairuse.Snapshot{
		Active:               2,
		ConcurrentSeconds:    12,
		Admitted:             3,
		Successful:           1,
		EffectiveConcurrency: deepseekfairuse.PeakConcurrency,
	}, true)
	require.NotNil(t, info.DeepSeekFairUse)
	assert.Equal(t, identifiers.AccountHMAC, info.DeepSeekFairUse.AccountHMAC)
	assert.Equal(t, identifiers.RiskHMAC, info.DeepSeekFairUse.RiskHMAC)
	assert.Equal(t, identifiers.RiskIPHMAC, info.DeepSeekFairUse.RiskIPHMAC)
	assert.Equal(t, identifiers.RiskCountryHMAC, info.DeepSeekFairUse.RiskCountryHMAC)
	assert.Equal(t, identifiers.RiskUserAgentHMAC, info.DeepSeekFairUse.RiskUserAgentHMAC)
	assert.NotContains(t, info.DeepSeekFairUse.AccountHMAC, "203.0.113.17")
	assert.NotContains(t, info.DeepSeekFairUse.RiskHMAC, "curl")
	assert.Equal(t, 2, info.DeepSeekFairUse.Active)
	assert.True(t, info.DeepSeekFairUse.RiskMarked)
}
