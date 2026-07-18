package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const emailDeliveryManualReviewAfter = 5 * time.Minute

const (
	EmailProviderBrevo = "brevo"
	EmailProviderSES   = "ses"

	EmailMessageTypeVerification  = "verification"
	EmailMessageTypePasswordReset = "password_reset"
	EmailMessageTypeReceipt       = "receipt"
	EmailMessageTypeNotification  = "notification"

	EmailDeliveryStatusSending     = "sending"
	EmailDeliveryStatusSent        = "sent"
	EmailDeliveryStatusFailed      = "failed"
	EmailDeliveryStatusRetryQueued = "retry_queued"

	EmailFailureConfiguration    = "configuration"
	EmailFailureAuthentication   = "authentication"
	EmailFailureRejected         = "rejected"
	EmailFailureRateLimited      = "rate_limited"
	EmailFailureUnavailable      = "provider_unavailable"
	EmailFailureTimeoutAmbiguous = "timeout_ambiguous"
	EmailFailureInternal         = "internal"
)

type EmailDelivery struct {
	Id                int        `json:"id" gorm:"primaryKey"`
	IdempotencyKey    string     `json:"-" gorm:"type:varchar(80);not null;uniqueIndex"`
	Provider          string     `json:"provider" gorm:"type:varchar(16);not null;index"`
	MessageType       string     `json:"message_type" gorm:"type:varchar(32);not null;index"`
	RecipientHash     string     `json:"recipient_hash" gorm:"type:char(64);not null;index"`
	ProviderMessageId string     `json:"provider_message_id" gorm:"type:varchar(255)"`
	Status            string     `json:"status" gorm:"type:varchar(24);not null;index"`
	FailureReason     string     `json:"failure_reason" gorm:"type:varchar(64)"`
	EncryptedPayload  string     `json:"-" gorm:"type:text"`
	AttemptCount      int        `json:"attempt_count" gorm:"not null"`
	RetryDeadline     *time.Time `json:"retry_deadline" gorm:"index"`
	SentAt            *time.Time `json:"sent_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (EmailDelivery) TableName() string {
	return "email_deliveries"
}

type EmailDeliverySummary struct {
	Latest            *EmailDelivery
	SafeRetryCount    int64
	ManualReviewCount int64
}

func ReserveEmailDelivery(delivery *EmailDelivery) (*EmailDelivery, bool, error) {
	result := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(delivery)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 1 {
		return delivery, true, nil
	}

	existing, err := GetEmailDeliveryByKey(delivery.IdempotencyKey)
	if err != nil {
		return nil, false, err
	}
	return existing, false, nil
}

func GetEmailDeliveryByKey(idempotencyKey string) (*EmailDelivery, error) {
	var delivery EmailDelivery
	if err := DB.Where("idempotency_key = ?", idempotencyKey).First(&delivery).Error; err != nil {
		return nil, err
	}
	return &delivery, nil
}

func MarkEmailDeliverySent(id int, providerMessageId string, sentAt time.Time) error {
	result := DB.Model(&EmailDelivery{}).
		Where("id = ? AND status = ?", id, EmailDeliveryStatusSending).
		Updates(map[string]any{
			"status":              EmailDeliveryStatusSent,
			"provider_message_id": providerMessageId,
			"failure_reason":      "",
			"encrypted_payload":   "",
			"retry_deadline":      nil,
			"sent_at":             sentAt,
		})
	return emailDeliveryTransitionError(result, id, EmailDeliveryStatusSent)
}

func MarkEmailDeliveryFailed(id int, failureReason string) error {
	result := DB.Model(&EmailDelivery{}).
		Where("id = ? AND status = ?", id, EmailDeliveryStatusSending).
		Updates(map[string]any{
			"status":              EmailDeliveryStatusFailed,
			"failure_reason":      failureReason,
			"encrypted_payload":   "",
			"retry_deadline":      nil,
			"provider_message_id": "",
		})
	return emailDeliveryTransitionError(result, id, EmailDeliveryStatusFailed)
}

func MarkEmailDeliveryRetryQueued(id int, failureReason string, retryDeadline *time.Time) error {
	result := DB.Model(&EmailDelivery{}).
		Where("id = ? AND status = ?", id, EmailDeliveryStatusSending).
		Updates(map[string]any{
			"status":              EmailDeliveryStatusRetryQueued,
			"failure_reason":      failureReason,
			"retry_deadline":      retryDeadline,
			"provider_message_id": "",
		})
	return emailDeliveryTransitionError(result, id, EmailDeliveryStatusRetryQueued)
}

func ClaimFailedEmailDelivery(id int, provider string, recipientHash string, encryptedPayload string) (bool, error) {
	result := DB.Model(&EmailDelivery{}).
		Where("id = ? AND status = ?", id, EmailDeliveryStatusFailed).
		Updates(map[string]any{
			"provider":            provider,
			"recipient_hash":      recipientHash,
			"status":              EmailDeliveryStatusSending,
			"failure_reason":      "",
			"provider_message_id": "",
			"encrypted_payload":   encryptedPayload,
			"retry_deadline":      nil,
			"attempt_count":       gorm.Expr("attempt_count + ?", 1),
		})
	return result.RowsAffected == 1, result.Error
}

func ListSafeRetryEmailDeliveries(now time.Time, limit int) ([]EmailDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	var deliveries []EmailDelivery
	err := DB.Where(
		"status = ? AND provider = ? AND retry_deadline IS NOT NULL AND retry_deadline >= ?",
		EmailDeliveryStatusRetryQueued,
		EmailProviderBrevo,
		now,
	).Order("id asc").Limit(limit).Find(&deliveries).Error
	return deliveries, err
}

func ClaimEmailDeliveryForSafeRetry(id int, now time.Time) (bool, error) {
	result := DB.Model(&EmailDelivery{}).
		Where(
			"id = ? AND status = ? AND provider = ? AND retry_deadline IS NOT NULL AND retry_deadline >= ?",
			id,
			EmailDeliveryStatusRetryQueued,
			EmailProviderBrevo,
			now,
		).
		Updates(map[string]any{
			"status":         EmailDeliveryStatusSending,
			"failure_reason": "",
			"attempt_count":  gorm.Expr("attempt_count + ?", 1),
		})
	return result.RowsAffected == 1, result.Error
}

func GetEmailDeliverySummary(now time.Time) (*EmailDeliverySummary, error) {
	summary := &EmailDeliverySummary{}
	var latest EmailDelivery
	err := DB.Order("id desc").First(&latest).Error
	if err == nil {
		summary.Latest = &latest
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if err := DB.Model(&EmailDelivery{}).
		Where(
			"status = ? AND provider = ? AND retry_deadline IS NOT NULL AND retry_deadline >= ?",
			EmailDeliveryStatusRetryQueued,
			EmailProviderBrevo,
			now,
		).
		Count(&summary.SafeRetryCount).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&EmailDelivery{}).
		Where(
			"(status = ? AND (provider <> ? OR retry_deadline IS NULL OR retry_deadline < ?)) OR (status = ? AND updated_at < ?)",
			EmailDeliveryStatusRetryQueued,
			EmailProviderBrevo,
			now,
			EmailDeliveryStatusSending,
			now.Add(-emailDeliveryManualReviewAfter),
		).
		Count(&summary.ManualReviewCount).Error; err != nil {
		return nil, err
	}
	return summary, nil
}

func emailDeliveryTransitionError(result *gorm.DB, id int, targetStatus string) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("email delivery %d cannot transition to %s", id, targetStatus)
	}
	return nil
}
