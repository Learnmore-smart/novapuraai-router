package model

import (
	"errors"

	"gorm.io/gorm"
)

// RedemptionUsage records a single redemption of a Redemption code by a user.
// The unique composite index (redemption_id, user_id) enforces the per-user-once
// invariant: a user can redeem any given code at most once, even under concurrent
// requests, because the second insert fails with a unique constraint violation.
type RedemptionUsage struct {
	Id           int   `json:"id" gorm:"primaryKey"`
	RedemptionId int   `json:"redemption_id" gorm:"uniqueIndex:idx_redemption_user;not null;default:0"`
	UserId       int   `json:"user_id" gorm:"uniqueIndex:idx_redemption_user;not null;default:0;index"`
	CreatedTime  int64 `json:"created_time" gorm:"bigint"`
	Quota        int   `json:"quota" gorm:"default:0"` // 内部 quota 实际发放数额（便于审计）
}

// HasUserRedeemed reports whether the given user has already redeemed the
// given redemption code. Used only for friendly pre-checks; the unique
// index on (redemption_id, user_id) is the authoritative enforcer.
func HasUserRedeemed(redemptionId, userId int) (bool, error) {
	if redemptionId == 0 || userId == 0 {
		return false, errors.New("invalid redemption_id or user_id")
	}
	var count int64
	err := DB.Model(&RedemptionUsage{}).
		Where("redemption_id = ? AND user_id = ?", redemptionId, userId).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountRedemptionUsage returns how many users have redeemed the given code.
func CountRedemptionUsage(redemptionId int) (int64, error) {
	if redemptionId == 0 {
		return 0, errors.New("invalid redemption_id")
	}
	var count int64
	err := DB.Model(&RedemptionUsage{}).
		Where("redemption_id = ?", redemptionId).
		Count(&count).Error
	return count, err
}

// DeleteRedemptionUsage deletes all usage rows for the given redemption id.
// Called when a redemption code is hard-deleted to keep the audit trail consistent.
func DeleteRedemptionUsage(tx *gorm.DB, redemptionId int) error {
	if redemptionId == 0 {
		return errors.New("invalid redemption_id")
	}
	return tx.Where("redemption_id = ?", redemptionId).Delete(&RedemptionUsage{}).Error
}
