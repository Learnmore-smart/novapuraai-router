package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CampaignClaim records one-time campaign grants (register promo, invite pair, share).
type CampaignClaim struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"uniqueIndex:uk_campaign_user_kind;index"`
	Kind      string `json:"kind" gorm:"type:varchar(64);uniqueIndex:uk_campaign_user_kind"`
	Amount    int    `json:"amount" gorm:"type:int;default:0"`
	Meta      string `json:"meta" gorm:"type:varchar(255)"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
}

func (CampaignClaim) TableName() string { return "campaign_claims" }

// CampaignCounter tracks atomic campaign counters (e.g. register promo slots used).
type CampaignCounter struct {
	Id    int    `json:"id" gorm:"primaryKey"`
	Name  string `json:"name" gorm:"type:varchar(64);uniqueIndex"`
	Count int    `json:"count" gorm:"type:int;default:0"`
}

func (CampaignCounter) TableName() string { return "campaign_counters" }

const (
	CampaignKindRegisterPromo = "register_promo_v1"
	CampaignKindInviteInvitee = "invite_invitee_v1"
	CampaignKindInviteInviter = "invite_inviter_v1"
	CampaignKindShare         = "share_reward_v1"
	CounterRegisterPromo      = "register_promo_v1"
)

var errRegisterPromoFull = errors.New("register promo is full")

func exchangeRateForCampaign() float64 {
	// Prefer the live Bank of Canada CNY rate when available — this is the
	// same rate shown on the billing settings page and refreshed daily by
	// service.StartBillingFXRefreshTask. Falls back to the legacy static
	// operation_setting.USDExchangeRate when the BoC feed is unavailable.
	return setting.EffectiveUSDCNYRate(operation_setting.USDExchangeRate)
}

// TryGrantRegisterPromo grants verified users a promo balance (atomic).
// Safe under concurrency: unique claim per user + counter UPDATE WHERE count < max.
// When RegisterPromoMax <= 0 the promo is unlimited (no counter check).
func TryGrantRegisterPromo(userId int) (granted bool, amount int, err error) {
	if !common.RegisterPromoEnabled || !common.EmailVerificationEnabled || userId <= 0 {
		return false, 0, nil
	}
	amount = common.CNYYuanToQuota(common.RegisterPromoCNYYuan, exchangeRateForCampaign())
	if amount <= 0 {
		return false, 0, nil
	}
	max := common.RegisterPromoMax
	unlimited := max <= 0

	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if e := tx.Select("id", "email").First(&user, userId).Error; e != nil {
			return e
		}
		if user.Email == "" {
			return nil
		}

		claim := CampaignClaim{
			UserId: userId,
			Kind:   CampaignKindRegisterPromo,
			Amount: amount,
			Meta:   fmt.Sprintf("cny=%.0f", common.RegisterPromoCNYYuan),
		}
		claimResult := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "kind"}},
			DoNothing: true,
		}).Create(&claim)
		if claimResult.Error != nil {
			return claimResult.Error
		}
		if claimResult.RowsAffected == 0 {
			return nil
		}

		if !unlimited {
			counter := CampaignCounter{Name: CounterRegisterPromo}
			if e := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoNothing: true,
			}).Create(&counter).Error; e != nil {
				return e
			}
			if e := lockForUpdate(tx).Where("name = ?", CounterRegisterPromo).First(&counter).Error; e != nil {
				return e
			}

			res := tx.Model(&CampaignCounter{}).
				Where("name = ? AND count < ?", CounterRegisterPromo, max).
				Update("count", gorm.Expr("count + 1"))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errRegisterPromoFull
			}
		}

		// Credit promo in same tx
		if e := tx.Model(&User{}).Where("id = ?", userId).Updates(map[string]any{
			"quota":       gorm.Expr("quota + ?", amount),
			"promo_quota": gorm.Expr("promo_quota + ?", amount),
		}).Error; e != nil {
			return e
		}
		granted = true
		return nil
	})
	if errors.Is(err, errRegisterPromoFull) {
		return false, amount, nil
	}
	if err != nil {
		return false, 0, err
	}
	if granted {
		if unlimited {
			RecordLog(userId, LogTypeSystem, fmt.Sprintf("注册活动赠送 %s", logger.LogQuota(amount)))
		} else {
			RecordLog(userId, LogTypeSystem, fmt.Sprintf("注册活动赠送 %s（前 %d 名）", logger.LogQuota(amount), max))
		}
		_ = invalidateUserCache(userId)
	}
	return granted, amount, nil
}

// TrySettleDelayedInviteReward pays inviter+invitee promo once after invitee qualifies.
func TrySettleDelayedInviteReward(inviteeId int) error {
	if !common.DelayedInviteReward {
		return nil
	}
	amount := common.CNYYuanToQuota(common.InviteRewardCNYYuan, exchangeRateForCampaign())
	if amount <= 0 {
		return nil
	}

	var paidInvitee bool
	var paidInviter bool
	var inviterId int
	var inviteeStateChanged bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var invitee User
		if err := lockForUpdate(tx).First(&invitee, inviteeId).Error; err != nil {
			return err
		}
		if !invitee.InviteRewardPending || invitee.InviterId == 0 {
			return nil
		}
		// Qualify: must have email (verified path sets email), at least one token, used_quota or request_count > 0
		if invitee.Email == "" {
			return nil
		}
		var tokenCount int64
		if err := tx.Model(&Token{}).Where("user_id = ?", inviteeId).Count(&tokenCount).Error; err != nil {
			return err
		}
		if tokenCount == 0 {
			return nil
		}
		if invitee.UsedQuota <= 0 && invitee.RequestCount <= 0 {
			return nil
		}

		var inviter User
		if err := lockForUpdate(tx).First(&inviter, invitee.InviterId).Error; err != nil {
			return err
		}
		if inviter.Id == inviteeId {
			if err := tx.Model(&invitee).Update("invite_reward_pending", false).Error; err != nil {
				return err
			}
			inviteeStateChanged = true
			return nil
		}
		inviterId = inviter.Id

		// Idempotent invitee claim
		var existing CampaignClaim
		if err := tx.Where("user_id = ? AND kind = ?", inviteeId, CampaignKindInviteInvitee).First(&existing).Error; err == nil {
			// already paid
			if err := tx.Model(&invitee).Update("invite_reward_pending", false).Error; err != nil {
				return err
			}
			inviteeStateChanged = true
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// Pay invitee promo
		if err := tx.Create(&CampaignClaim{
			UserId: inviteeId,
			Kind:   CampaignKindInviteInvitee,
			Amount: amount,
			Meta:   fmt.Sprintf("inviter=%d", invitee.InviterId),
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", inviteeId).Updates(map[string]any{
			"quota":                 gorm.Expr("quota + ?", amount),
			"promo_quota":           gorm.Expr("promo_quota + ?", amount),
			"invite_reward_pending": false,
		}).Error; err != nil {
			return err
		}
		paidInvitee = true
		inviteeStateChanged = true

		// Pay inviter if under cap. Approved affiliates are mutually exclusive with
		// the fixed ¥50 invite reward: they earn cash commission via
		// SettleRechargeCommission on the invitee's paid top-ups instead, so skip
		// the quota grant here (AffCount still increments for affiliate tracking).
		inviter.AffCount++
		updates := map[string]any{"aff_count": inviter.AffCount}
		if !inviter.CommissionApproved && inviter.RewardedInviteCount < common.MaxValidInvites {
			// unique kind per invitee for inviter side
			kind := fmt.Sprintf("%s:%d", CampaignKindInviteInviter, inviteeId)
			if err := tx.Where("user_id = ? AND kind = ?", inviter.Id, kind).First(&CampaignClaim{}).Error; err == nil {
				// already
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&CampaignClaim{
					UserId: inviter.Id,
					Kind:   kind,
					Amount: amount,
					Meta:   fmt.Sprintf("invitee=%d", inviteeId),
				}).Error; err != nil {
					return err
				}
				updates["rewarded_invite_count"] = inviter.RewardedInviteCount + 1
				updates["quota"] = gorm.Expr("quota + ?", amount)
				updates["promo_quota"] = gorm.Expr("promo_quota + ?", amount)
				paidInviter = true
			} else {
				return err
			}
		}
		if err := tx.Model(&User{}).Where("id = ?", inviter.Id).Updates(updates).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if paidInvitee {
		RecordLog(inviteeId, LogTypeSystem, fmt.Sprintf("Invite reward %s", logger.LogQuota(amount)))
	}
	if inviteeStateChanged {
		_ = invalidateUserCache(inviteeId)
	}
	if inviterId > 0 {
		_ = invalidateUserCache(inviterId)
	}
	if paidInviter {
		RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("Invite reward %s", logger.LogQuota(amount)))
	}
	return nil
}

// TryGrantShareReward atomic one-time share reward (called from admin approve).
func TryGrantShareReward(userId int, amount int) (bool, error) {
	if userId <= 0 || amount <= 0 {
		return false, nil
	}
	var granted bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing CampaignClaim
		if e := tx.Where("user_id = ? AND kind = ?", userId, CampaignKindShare).First(&existing).Error; e == nil {
			granted = false
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if e := tx.Create(&CampaignClaim{
			UserId: userId,
			Kind:   CampaignKindShare,
			Amount: amount,
			Meta:   time.Now().UTC().Format(time.RFC3339),
		}).Error; e != nil {
			return e
		}
		if e := tx.Model(&User{}).Where("id = ?", userId).Updates(map[string]any{
			"quota":       gorm.Expr("quota + ?", amount),
			"promo_quota": gorm.Expr("promo_quota + ?", amount),
		}).Error; e != nil {
			return e
		}
		granted = true
		return nil
	})
	if err != nil {
		return false, err
	}
	if granted {
		RecordLog(userId, LogTypeSystem, fmt.Sprintf("分享奖励 %s", logger.LogQuota(amount)))
		_ = invalidateUserCache(userId)
	}
	return granted, nil
}
