package model

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique_violation") ||
		strings.Contains(msg, "constraint failed")
}

// StripeWebhookEvent stores processed Stripe event IDs for idempotency.
type StripeWebhookEvent struct {
	Id          int64  `json:"id" gorm:"primaryKey"`
	EventID     string `json:"event_id" gorm:"type:varchar(128);uniqueIndex;not null"`
	EventType   string `json:"event_type" gorm:"type:varchar(64);index"`
	Livemode    bool   `json:"livemode"`
	AccountID   string `json:"account_id" gorm:"type:varchar(64);default:''"`
	OrderID     string `json:"order_id" gorm:"type:varchar(64);index;default:''"`
	PayloadHash string `json:"payload_hash" gorm:"type:varchar(64);default:''"`
	CreatedAt   int64  `json:"created_at" gorm:"index"`
}

func (StripeWebhookEvent) TableName() string {
	return "stripe_webhook_events"
}

// TryInsertStripeWebhookEvent returns true if this is the first time we see eventID.
func TryInsertStripeWebhookEvent(ev *StripeWebhookEvent) (inserted bool, err error) {
	err = DB.Create(ev).Error
	if err != nil {
		// unique violation → already processed
		if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
