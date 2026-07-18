package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEmailDeliveryTest(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&EmailDelivery{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&EmailDelivery{}).Error)
	t.Cleanup(func() {
		_ = DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&EmailDelivery{}).Error
	})
}

func newEmailDeliveryFixture(key string, provider string) *EmailDelivery {
	return &EmailDelivery{
		IdempotencyKey:   key,
		Provider:         provider,
		MessageType:      EmailMessageTypeVerification,
		RecipientHash:    "recipient-hash",
		Status:           EmailDeliveryStatusSending,
		EncryptedPayload: "enc:string:v1:ciphertext",
		AttemptCount:     1,
	}
}

func TestEmailDeliveryReservationIsUniqueAcrossProviders(t *testing.T) {
	setupEmailDeliveryTest(t)

	first, created, err := ReserveEmailDelivery(newEmailDeliveryFixture("event-key", EmailProviderBrevo))
	require.NoError(t, err)
	require.True(t, created)
	assert.Equal(t, EmailProviderBrevo, first.Provider)

	duplicate, created, err := ReserveEmailDelivery(newEmailDeliveryFixture("event-key", EmailProviderSES))
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, first.Id, duplicate.Id)
	assert.Equal(t, EmailProviderBrevo, duplicate.Provider)
	assert.Equal(t, 1, duplicate.AttemptCount)
}

func TestEmailDeliveryTerminalTransitionsClearQueuedPayload(t *testing.T) {
	setupEmailDeliveryTest(t)
	now := time.Now().UTC().Truncate(time.Second)

	delivery, created, err := ReserveEmailDelivery(newEmailDeliveryFixture("sent-key", EmailProviderBrevo))
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, MarkEmailDeliverySent(delivery.Id, "brevo-message-id", now))

	stored, err := GetEmailDeliveryByKey("sent-key")
	require.NoError(t, err)
	assert.Equal(t, EmailDeliveryStatusSent, stored.Status)
	assert.Equal(t, "brevo-message-id", stored.ProviderMessageId)
	assert.Empty(t, stored.EncryptedPayload)
	require.NotNil(t, stored.SentAt)
	assert.WithinDuration(t, now, *stored.SentAt, time.Second)

	failed, created, err := ReserveEmailDelivery(newEmailDeliveryFixture("failed-key", EmailProviderSES))
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, MarkEmailDeliveryFailed(failed.Id, EmailFailureRejected))

	stored, err = GetEmailDeliveryByKey("failed-key")
	require.NoError(t, err)
	assert.Equal(t, EmailDeliveryStatusFailed, stored.Status)
	assert.Equal(t, EmailFailureRejected, stored.FailureReason)
	assert.Empty(t, stored.EncryptedPayload)
}

func TestEmailDeliveryRetryQueueKeepsOriginalProviderAndClassifiesSafety(t *testing.T) {
	setupEmailDeliveryTest(t)
	now := time.Now().UTC().Truncate(time.Second)
	deadline := now.Add(10 * time.Minute)

	brevo, created, err := ReserveEmailDelivery(newEmailDeliveryFixture("brevo-retry", EmailProviderBrevo))
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, MarkEmailDeliveryRetryQueued(brevo.Id, EmailFailureTimeoutAmbiguous, &deadline))

	ses, created, err := ReserveEmailDelivery(newEmailDeliveryFixture("ses-review", EmailProviderSES))
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, MarkEmailDeliveryRetryQueued(ses.Id, EmailFailureTimeoutAmbiguous, nil))

	summary, err := GetEmailDeliverySummary(now)
	require.NoError(t, err)
	assert.EqualValues(t, 1, summary.SafeRetryCount)
	assert.EqualValues(t, 1, summary.ManualReviewCount)
	require.NotNil(t, summary.Latest)
	assert.Equal(t, "ses-review", summary.Latest.IdempotencyKey)

	rows, err := ListSafeRetryEmailDeliveries(now, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, brevo.Id, rows[0].Id)
	assert.NotEmpty(t, rows[0].EncryptedPayload)

	claimed, err := ClaimEmailDeliveryForSafeRetry(brevo.Id, now)
	require.NoError(t, err)
	assert.True(t, claimed)
	claimedAgain, err := ClaimEmailDeliveryForSafeRetry(brevo.Id, now)
	require.NoError(t, err)
	assert.False(t, claimedAgain)

	stored, err := GetEmailDeliveryByKey("brevo-retry")
	require.NoError(t, err)
	assert.Equal(t, EmailDeliveryStatusSending, stored.Status)
	assert.Equal(t, EmailProviderBrevo, stored.Provider)
	assert.Equal(t, 2, stored.AttemptCount)
}

func TestEmailDeliveryDefinitiveFailureCanBeClaimedByNewProvider(t *testing.T) {
	setupEmailDeliveryTest(t)
	delivery, created, err := ReserveEmailDelivery(newEmailDeliveryFixture("definitive-failure", EmailProviderBrevo))
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, MarkEmailDeliveryFailed(delivery.Id, EmailFailureAuthentication))

	claimed, err := ClaimFailedEmailDelivery(delivery.Id, EmailProviderSES, "new-recipient-hash", "enc:string:v1:new-payload")
	require.NoError(t, err)
	assert.True(t, claimed)

	stored, err := GetEmailDeliveryByKey("definitive-failure")
	require.NoError(t, err)
	assert.Equal(t, EmailDeliveryStatusSending, stored.Status)
	assert.Equal(t, EmailProviderSES, stored.Provider)
	assert.Equal(t, "new-recipient-hash", stored.RecipientHash)
	assert.Equal(t, 2, stored.AttemptCount)
	assert.Equal(t, "enc:string:v1:new-payload", stored.EncryptedPayload)
}

func TestEmailDeliverySummaryFlagsStaleSendingRowsForManualReview(t *testing.T) {
	setupEmailDeliveryTest(t)
	now := time.Now().UTC().Truncate(time.Second)
	delivery, created, err := ReserveEmailDelivery(newEmailDeliveryFixture("stale-send", EmailProviderSES))
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, DB.Model(&EmailDelivery{}).Where("id = ?", delivery.Id).Update("updated_at", now.Add(-10*time.Minute)).Error)

	summary, err := GetEmailDeliverySummary(now)
	require.NoError(t, err)
	assert.EqualValues(t, 1, summary.ManualReviewCount)
}
