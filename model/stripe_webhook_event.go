package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StripeWebhookEventProcessing   = "processing"
	StripeWebhookEventProcessed    = "processed"
	StripeWebhookEventManualReview = "manual_review"
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
	Status      string `json:"status" gorm:"type:varchar(32);index"`
	LeaseUntil  int64  `json:"-" gorm:"bigint;index"`
	Attempts    int    `json:"attempts" gorm:"default:0"`
	LastError   string `json:"-" gorm:"type:varchar(255)"`
	ProcessedAt int64  `json:"-" gorm:"bigint;index"`
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

// ClaimStripeWebhookEvent claims a recurring event with a lease. A processed
// or manual-review row is terminal; an unexpired processing lease is treated as
// an in-flight duplicate; an expired lease can be reclaimed by another worker.
func ClaimStripeWebhookEvent(ev *StripeWebhookEvent, now int64, lease time.Duration) (claimed bool, terminal bool, err error) {
	if DB == nil || ev == nil || strings.TrimSpace(ev.EventID) == "" {
		return false, false, gorm.ErrInvalidDB
	}
	if now <= 0 {
		now = time.Now().Unix()
	}
	if lease <= 0 {
		lease = 5 * time.Minute
	}
	if ev.CreatedAt == 0 {
		ev.CreatedAt = now
	}
	returnClaimed := false
	returnTerminal := false
	err = DB.Transaction(func(tx *gorm.DB) error {
		record := *ev
		record.Status = StripeWebhookEventProcessing
		record.LeaseUntil = now + int64(lease/time.Second)
		record.Attempts = 1
		createResult := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record)
		if createResult.Error != nil {
			return createResult.Error
		}
		if createResult.RowsAffected > 0 {
			*ev = record
			returnClaimed = true
			return nil
		}

		var existing StripeWebhookEvent
		if err := lockForUpdate(tx).Where("event_id = ?", ev.EventID).First(&existing).Error; err != nil {
			return err
		}
		ev.Status = existing.Status
		switch existing.Status {
		case StripeWebhookEventProcessed, StripeWebhookEventManualReview:
			returnTerminal = true
			return nil
		case StripeWebhookEventProcessing:
			if existing.LeaseUntil > now {
				returnTerminal = true
				return nil
			}
		}
		updates := map[string]any{
			"status":      StripeWebhookEventProcessing,
			"lease_until": now + int64(lease/time.Second),
			"attempts":    existing.Attempts + 1,
			"last_error":  "",
		}
		if err := tx.Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
		returnClaimed = true
		return nil
	})
	return returnClaimed, returnTerminal, err
}

func FinalizeStripeWebhookEvent(eventID string, status string, lastError string, now int64) error {
	if DB == nil || strings.TrimSpace(eventID) == "" {
		return gorm.ErrInvalidDB
	}
	if status != StripeWebhookEventProcessed && status != StripeWebhookEventManualReview {
		return errors.New("invalid Stripe webhook terminal status")
	}
	if now <= 0 {
		now = time.Now().Unix()
	}
	updates := map[string]any{
		"status":       status,
		"lease_until":  0,
		"last_error":   strings.TrimSpace(lastError),
		"processed_at": now,
	}
	result := DB.Model(&StripeWebhookEvent{}).
		Where("event_id = ? AND status = ? AND lease_until >= ?", strings.TrimSpace(eventID), StripeWebhookEventProcessing, now).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var existing StripeWebhookEvent
		if err := DB.Where("event_id = ?", strings.TrimSpace(eventID)).First(&existing).Error; err != nil {
			return err
		}
		if existing.Status == status {
			return nil
		}
		return errors.New("Stripe webhook event is not claimable for finalization")
	}
	return nil
}
