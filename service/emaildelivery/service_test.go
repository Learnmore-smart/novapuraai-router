package emaildelivery

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func setupEmailServiceTest(t *testing.T) time.Time {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	previousSecret := common.CryptoSecret
	model.DB = db
	common.CryptoSecret = "email-delivery-test-secret"
	require.NoError(t, model.DB.AutoMigrate(&model.EmailDelivery{}))
	t.Cleanup(func() {
		model.DB = previousDB
		common.CryptoSecret = previousSecret
	})
	return time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
}

func TestServicePreventsDuplicateAcrossProviderSwitch(t *testing.T) {
	now := setupEmailServiceTest(t)
	ses := &fakeProvider{name: ProviderSES, results: []ProviderResult{{MessageID: "ses-id"}}, safeRetry: true}
	selected := ProviderSES
	service := NewService(map[ProviderName]Provider{ProviderSES: ses}, func() ProviderName {
		return selected
	}, func() time.Time { return now })
	request := SendRequest{BusinessKey: "verification:user-42:code-7", Message: testMessage()}

	first, err := service.Send(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, DeliveryStatusSent, first.Status)
	assert.False(t, first.Duplicate)
	assert.Equal(t, ProviderSES, first.Provider)

	duplicate, err := service.Send(context.Background(), request)
	require.NoError(t, err)
	assert.True(t, duplicate.Duplicate)
	assert.Equal(t, DeliveryStatusSent, duplicate.Status)
	assert.Equal(t, ProviderSES, duplicate.Provider)
	assert.Equal(t, 1, ses.sendCalls)

	deliveries := make([]model.EmailDelivery, 0)
	require.NoError(t, model.DB.Find(&deliveries).Error)
	require.Len(t, deliveries, 1)
	assert.NotEqual(t, request.BusinessKey, deliveries[0].IdempotencyKey)
	assert.NotEqual(t, request.Message.To, deliveries[0].RecipientHash)
	assert.Len(t, deliveries[0].RecipientHash, 64)
	assert.Empty(t, deliveries[0].EncryptedPayload)
}

func TestServiceAllowsRetryAfterDefinitiveFailure(t *testing.T) {
	now := setupEmailServiceTest(t)
	ses := &fakeProvider{
		name:      ProviderSES,
		errors:    []*DeliveryError{{Reason: FailureRejected}, nil},
		results:   []ProviderResult{{}, {MessageID: "ses-id-2"}},
		safeRetry: true,
	}
	selected := ProviderSES
	service := NewService(map[ProviderName]Provider{ProviderSES: ses}, func() ProviderName {
		return selected
	}, func() time.Time { return now })
	request := SendRequest{BusinessKey: "password-reset:user-42:token-1", Message: testMessage()}
	request.Message.Type = MessageTypePasswordReset

	failed, err := service.Send(context.Background(), request)
	require.Error(t, err)
	assert.Equal(t, DeliveryStatusFailed, failed.Status)
	assert.Equal(t, FailureRejected, failed.FailureReason)

	sent, err := service.Send(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, DeliveryStatusSent, sent.Status)
	assert.Equal(t, ProviderSES, sent.Provider)
	assert.Equal(t, 2, ses.sendCalls)

	delivery, err := model.GetEmailDeliveryByKey(service.idempotencyKey(request.BusinessKey, request.Message.Type))
	require.NoError(t, err)
	assert.Equal(t, 2, delivery.AttemptCount)
	assert.Equal(t, model.EmailProviderSES, delivery.Provider)
}

func TestServiceQueuesAmbiguousTimeout(t *testing.T) {
	now := setupEmailServiceTest(t)
	ses := &fakeProvider{
		name:      ProviderSES,
		errors:    []*DeliveryError{{Reason: FailureTimeoutAmbiguous, Ambiguous: true}},
		safeRetry: true,
	}
	selected := ProviderSES
	service := NewService(map[ProviderName]Provider{ProviderSES: ses}, func() ProviderName {
		return selected
	}, func() time.Time { return now })
	request := SendRequest{BusinessKey: "notification:event-9", Message: testMessage()}
	request.Message.Type = MessageTypeNotification

	queued, err := service.Send(context.Background(), request)
	require.Error(t, err)
	assert.Equal(t, DeliveryStatusRetryQueued, queued.Status)
	assert.Equal(t, FailureTimeoutAmbiguous, queued.FailureReason)

	stored, err := model.GetEmailDeliveryByKey(service.idempotencyKey(request.BusinessKey, request.Message.Type))
	require.NoError(t, err)
	require.NotNil(t, stored.RetryDeadline)
	assert.Equal(t, now.Add(safeRetryWindow), *stored.RetryDeadline)
	assert.NotEmpty(t, stored.EncryptedPayload)
	assert.False(t, strings.Contains(stored.EncryptedPayload, request.Message.To))

	duplicate, err := service.Send(context.Background(), request)
	require.NoError(t, err)
	assert.True(t, duplicate.Duplicate)
	assert.Equal(t, DeliveryStatusRetryQueued, duplicate.Status)
	assert.Equal(t, ProviderSES, duplicate.Provider)
	assert.Equal(t, 1, ses.sendCalls)
}

func TestServiceRetriesOnlyEligibleDeliveryWithSameKey(t *testing.T) {
	now := setupEmailServiceTest(t)
	ses := &fakeProvider{
		name:      ProviderSES,
		errors:    []*DeliveryError{{Reason: FailureTimeoutAmbiguous, Ambiguous: true}, nil},
		results:   []ProviderResult{{}, {MessageID: "ses-retry-id"}},
		safeRetry: true,
	}
	service := NewService(map[ProviderName]Provider{ProviderSES: ses}, func() ProviderName {
		return ProviderSES
	}, func() time.Time { return now })
	request := SendRequest{BusinessKey: "receipt:payment-17", Message: testMessage()}
	request.Message.Type = MessageTypeReceipt

	_, err := service.Send(context.Background(), request)
	require.Error(t, err)
	retryResult, err := service.RetrySafeQueue(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, retryResult.Processed)
	assert.Equal(t, 1, retryResult.Sent)
	assert.Zero(t, retryResult.Failed)
	assert.Equal(t, 2, ses.sendCalls)
	require.Len(t, ses.idempotencyKeys, 2)
	assert.Equal(t, ses.idempotencyKeys[0], ses.idempotencyKeys[1])

	stored, err := model.GetEmailDeliveryByKey(service.idempotencyKey(request.BusinessKey, request.Message.Type))
	require.NoError(t, err)
	assert.Equal(t, model.EmailDeliveryStatusSent, stored.Status)
	assert.Equal(t, "ses-retry-id", stored.ProviderMessageId)
	assert.Empty(t, stored.EncryptedPayload)
}

func TestServiceReportsAmbiguousSafeRetryAsQueuedNotFailed(t *testing.T) {
	now := setupEmailServiceTest(t)
	ses := &fakeProvider{
		name: ProviderSES,
		errors: []*DeliveryError{
			{Reason: FailureTimeoutAmbiguous, Ambiguous: true},
			{Reason: FailureTimeoutAmbiguous, Ambiguous: true},
		},
		safeRetry: true,
	}
	service := NewService(map[ProviderName]Provider{ProviderSES: ses}, func() ProviderName {
		return ProviderSES
	}, func() time.Time { return now })
	request := SendRequest{BusinessKey: "receipt:payment-ambiguous", Message: testMessage()}
	request.Message.Type = MessageTypeReceipt

	_, err := service.Send(context.Background(), request)
	require.Error(t, err)
	retryResult, err := service.RetrySafeQueue(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, retryResult.Processed)
	assert.Zero(t, retryResult.Sent)
	assert.Equal(t, 1, retryResult.Queued)
	assert.Zero(t, retryResult.Failed)

	stored, err := model.GetEmailDeliveryByKey(service.idempotencyKey(request.BusinessKey, request.Message.Type))
	require.NoError(t, err)
	assert.Equal(t, model.EmailDeliveryStatusRetryQueued, stored.Status)
}

func TestServiceNeverAutomaticallyRetriesAmbiguousSESDeliveryWithoutSafeRetry(t *testing.T) {
	now := setupEmailServiceTest(t)
	ses := &fakeProvider{
		name:   ProviderSES,
		errors: []*DeliveryError{{Reason: FailureTimeoutAmbiguous, Ambiguous: true}},
	}
	service := NewService(map[ProviderName]Provider{ProviderSES: ses}, func() ProviderName {
		return ProviderSES
	}, func() time.Time { return now })
	request := SendRequest{BusinessKey: "verification:user-8:code-3", Message: testMessage()}

	_, err := service.Send(context.Background(), request)
	require.Error(t, err)
	retryResult, err := service.RetrySafeQueue(context.Background(), 10)
	require.NoError(t, err)
	assert.Zero(t, retryResult.Processed)
	assert.Equal(t, 1, ses.sendCalls)

	stored, err := model.GetEmailDeliveryByKey(service.idempotencyKey(request.BusinessKey, request.Message.Type))
	require.NoError(t, err)
	assert.Equal(t, model.EmailDeliveryStatusRetryQueued, stored.Status)
	assert.Nil(t, stored.RetryDeadline)

	summary, err := model.GetEmailDeliverySummary(now)
	require.NoError(t, err)
	assert.EqualValues(t, 1, summary.ManualReviewCount)
}

func TestServiceHealthCombinesProviderReadinessAndSanitizedQueueSummary(t *testing.T) {
	now := setupEmailServiceTest(t)
	ses := &fakeProvider{
		name: ProviderSES,
		health: ProviderHealth{
			Provider:         ProviderSES,
			Configured:       true,
			Reachable:        true,
			SendingEnabled:   true,
			ProductionAccess: true,
			Ready:            true,
		},
		safeRetry: true,
	}
	service := NewService(map[ProviderName]Provider{ProviderSES: ses}, func() ProviderName {
		return ProviderSES
	}, func() time.Time { return now })
	request := SendRequest{BusinessKey: "health-summary", Message: testMessage()}
	request.Message.To = "private-recipient@example.com"
	ses.errors = []*DeliveryError{{Reason: FailureTimeoutAmbiguous, Ambiguous: true}}
	_, err := service.Send(context.Background(), request)
	require.Error(t, err)

	report, err := service.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ProviderSES, report.SelectedProvider)
	require.Len(t, report.Providers, 1)
	assert.Equal(t, ProviderSES, report.Providers[0].Provider)
	assert.True(t, report.Providers[0].Ready)
	assert.EqualValues(t, 1, report.SafeRetryCount)
	assert.Zero(t, report.ManualReviewCount)
	require.NotNil(t, report.LatestDelivery)
	assert.Equal(t, request.Message.Type, report.LatestDelivery.MessageType)
	assert.NotEqual(t, request.Message.To, report.LatestDelivery.RecipientHash)
	assert.NotContains(t, report.LatestDelivery.RecipientHash, "@")
}

type fakeProvider struct {
	name            ProviderName
	results         []ProviderResult
	errors          []*DeliveryError
	health          ProviderHealth
	safeRetry       bool
	sendCalls       int
	idempotencyKeys []string
}

func (provider *fakeProvider) Name() ProviderName {
	return provider.name
}

func (provider *fakeProvider) Send(_ context.Context, _ Message, idempotencyKey string) (ProviderResult, *DeliveryError) {
	index := provider.sendCalls
	provider.sendCalls++
	provider.idempotencyKeys = append(provider.idempotencyKeys, idempotencyKey)
	var result ProviderResult
	if index < len(provider.results) {
		result = provider.results[index]
	}
	if index < len(provider.errors) {
		return result, provider.errors[index]
	}
	return result, nil
}

func (provider *fakeProvider) Health(context.Context) ProviderHealth {
	return provider.health
}

func (provider *fakeProvider) SupportsSafeRetry() bool {
	return provider.safeRetry
}
