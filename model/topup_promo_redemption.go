package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	TopupPromoRedemptionReserved = "reserved"
	TopupPromoRedemptionIssued   = "issued"
	TopupPromoRedemptionReleased = "released"
	TopupPromoRedemptionReversed = "reversed"
)

type TopupPromoRedemption struct {
	Id            int64  `json:"id" gorm:"primaryKey"`
	OrderID       string `json:"order_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	CampaignID    int    `json:"campaign_id" gorm:"index;not null"`
	TierID        int    `json:"tier_id" gorm:"index;not null"`
	UserID        int    `json:"user_id" gorm:"index;not null"`
	PromoMicroUSD int64  `json:"promo_micro_usd" gorm:"not null"`
	Status        string `json:"status" gorm:"type:varchar(16);index;not null"`
	CreatedAt     int64  `json:"created_at" gorm:"not null"`
	UpdatedAt     int64  `json:"updated_at" gorm:"not null"`
	IssuedAt      int64  `json:"issued_at" gorm:"not null;default:0"`
	ReleasedAt    int64  `json:"released_at" gorm:"not null;default:0"`
	ReversedAt    int64  `json:"reversed_at" gorm:"not null;default:0"`
}

func (TopupPromoRedemption) TableName() string {
	return "topup_promo_redemptions"
}

func ReserveTopupPromotion(orderID string, userID, tierID int, promoMicroUSD int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return ReserveTopupPromotionWithTx(tx, orderID, userID, tierID, promoMicroUSD)
	})
}

func GetExactTopupPromoTier(tierID int, currency string, now int64) (*TopupPromoTier, *TopupPromotionCampaign, error) {
	if tierID <= 0 {
		return nil, nil, ErrTopupPromotionUnavailable
	}
	var tier TopupPromoTier
	if err := DB.First(&tier, tierID).Error; err != nil {
		return nil, nil, ErrTopupPromotionUnavailable
	}
	if tier.PaymentAmountMinor <= 0 || tier.Currency != currency || !tier.ActiveAt(now) {
		return nil, nil, ErrTopupPromotionUnavailable
	}
	var campaign TopupPromotionCampaign
	if err := DB.First(&campaign, tier.CampaignID).Error; err != nil || !campaign.ActiveAt(now) {
		return nil, nil, ErrTopupPromotionUnavailable
	}
	return &tier, &campaign, nil
}

func FindExactTopupPromoTier(currency string, paymentAmountMinor int64, now int64) (*TopupPromoTier, *TopupPromotionCampaign, error) {
	var tier TopupPromoTier
	if err := DB.Where("currency = ? AND payment_amount_minor = ? AND enabled = ?", currency, paymentAmountMinor, true).
		Order("sort_order asc, id asc").First(&tier).Error; err != nil {
		return nil, nil, ErrTopupPromotionUnavailable
	}
	return GetExactTopupPromoTier(tier.Id, currency, now)
}

func FindTopupPromoBand(currency string, paymentAmountMinor int64, now int64) (*TopupPromoTier, *TopupPromotionCampaign, error) {
	if paymentAmountMinor <= 0 {
		return nil, nil, ErrTopupPromotionUnavailable
	}
	tiers, err := ListEnabledPromoTiers()
	if err != nil {
		return nil, nil, ErrTopupPromotionUnavailable
	}
	currency = strings.ToLower(strings.TrimSpace(currency))
	var best *TopupPromoTier
	for _, tier := range tiers {
		if tier == nil || tier.PaymentAmountMinor != 0 || tier.PercentBonusBps <= 0 || !tier.ActiveAt(now) {
			continue
		}
		tierCurrency := strings.ToLower(strings.TrimSpace(tier.Currency))
		if tierCurrency != "*" && tierCurrency != currency {
			continue
		}
		if paymentAmountMinor < int64(tier.MinPresentmentMajor)*100 {
			continue
		}
		if tier.MaxPresentmentMajor > 0 && paymentAmountMinor >= int64(tier.MaxPresentmentMajor+1)*100 {
			continue
		}
		if best == nil || tier.PercentBonusBps > best.PercentBonusBps {
			best = tier
		}
	}
	if best == nil {
		return nil, nil, ErrTopupPromotionUnavailable
	}
	var campaign TopupPromotionCampaign
	if err := DB.First(&campaign, best.CampaignID).Error; err != nil || !campaign.ActiveAt(now) {
		return nil, nil, ErrTopupPromotionUnavailable
	}
	return best, &campaign, nil
}

func CheckTopupPromotionCapacity(userID int, tier *TopupPromoTier, campaign *TopupPromotionCampaign, promoMicroUSD int64) error {
	if tier == nil || campaign == nil || promoMicroUSD <= 0 {
		return ErrTopupPromotionUnavailable
	}
	activeStatuses := []string{TopupPromoRedemptionReserved, TopupPromoRedemptionIssued}
	if userID > 0 && campaign.PerUserLimit > 0 {
		var used int64
		if err := DB.Model(&TopupPromoRedemption{}).
			Where("campaign_id = ? AND user_id = ? AND status IN ?", campaign.Id, userID, activeStatuses).
			Count(&used).Error; err != nil {
			return err
		}
		if used >= int64(campaign.PerUserLimit) {
			return ErrTopupPromotionUserLimit
		}
	}
	if userID > 0 && tier.PerUserLimit > 0 {
		var used int64
		if err := DB.Model(&TopupPromoRedemption{}).
			Where("tier_id = ? AND user_id = ? AND status IN ?", tier.Id, userID, activeStatuses).
			Count(&used).Error; err != nil {
			return err
		}
		if used >= int64(tier.PerUserLimit) {
			return ErrTopupPromotionUserLimit
		}
	}
	if campaign.GlobalBudgetMicroUSD > 0 {
		remaining := campaign.GlobalBudgetMicroUSD - campaign.ReservedPromoMicroUSD - campaign.IssuedPromoMicroUSD
		if remaining < promoMicroUSD {
			return ErrTopupPromotionBudgetReached
		}
	}
	return nil
}

func ReserveTopupPromotionWithTx(tx *gorm.DB, orderID string, userID, tierID int, promoMicroUSD int64) error {
	if tx == nil || orderID == "" || userID <= 0 || tierID <= 0 || promoMicroUSD <= 0 {
		return errors.New("invalid top-up promotion reservation")
	}
	var existing TopupPromoRedemption
	if err := tx.Where("order_id = ?", orderID).First(&existing).Error; err == nil {
		if existing.UserID == userID && existing.TierID == tierID && existing.PromoMicroUSD == promoMicroUSD {
			return nil
		}
		return errors.New("top-up promotion order already reserved with different terms")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var tier TopupPromoTier
	if err := tx.First(&tier, tierID).Error; err != nil {
		return ErrTopupPromotionUnavailable
	}
	var campaign TopupPromotionCampaign
	if err := lockForUpdate(tx).First(&campaign, tier.CampaignID).Error; err != nil {
		return ErrTopupPromotionUnavailable
	}
	now := common.GetTimestamp()
	if !campaign.ActiveAt(now) || !tier.ActiveAt(now) {
		return ErrTopupPromotionUnavailable
	}

	activeStatuses := []string{TopupPromoRedemptionReserved, TopupPromoRedemptionIssued}
	if campaign.PerUserLimit > 0 {
		var used int64
		if err := tx.Model(&TopupPromoRedemption{}).
			Where("campaign_id = ? AND user_id = ? AND status IN ?", campaign.Id, userID, activeStatuses).
			Count(&used).Error; err != nil {
			return err
		}
		if used >= int64(campaign.PerUserLimit) {
			return ErrTopupPromotionUserLimit
		}
	}
	if tier.PerUserLimit > 0 {
		var used int64
		if err := tx.Model(&TopupPromoRedemption{}).
			Where("tier_id = ? AND user_id = ? AND status IN ?", tier.Id, userID, activeStatuses).
			Count(&used).Error; err != nil {
			return err
		}
		if used >= int64(tier.PerUserLimit) {
			return ErrTopupPromotionUserLimit
		}
	}
	if campaign.GlobalBudgetMicroUSD > 0 {
		remaining := campaign.GlobalBudgetMicroUSD - campaign.ReservedPromoMicroUSD - campaign.IssuedPromoMicroUSD
		if remaining < promoMicroUSD {
			return ErrTopupPromotionBudgetReached
		}
	}

	redemption := TopupPromoRedemption{
		OrderID:       orderID,
		CampaignID:    campaign.Id,
		TierID:        tier.Id,
		UserID:        userID,
		PromoMicroUSD: promoMicroUSD,
		Status:        TopupPromoRedemptionReserved,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := tx.Create(&redemption).Error; err != nil {
		return err
	}
	campaign.ReservedPromoMicroUSD += promoMicroUSD
	campaign.UpdatedAt = now
	return tx.Save(&campaign).Error
}

func ReleaseTopupPromotion(orderID string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseTopupPromotionWithTx(tx, orderID)
	})
}

func ReleaseTopupPromotionWithTx(tx *gorm.DB, orderID string) error {
	redemption, campaign, err := lockTopupRedemptionAndCampaign(tx, orderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if redemption.Status == TopupPromoRedemptionReleased || redemption.Status == TopupPromoRedemptionReversed {
		return nil
	}
	if redemption.Status != TopupPromoRedemptionReserved {
		return fmt.Errorf("cannot release promotion in status %s", redemption.Status)
	}
	now := common.GetTimestamp()
	campaign.ReservedPromoMicroUSD -= redemption.PromoMicroUSD
	if campaign.ReservedPromoMicroUSD < 0 {
		campaign.ReservedPromoMicroUSD = 0
	}
	campaign.UpdatedAt = now
	redemption.Status = TopupPromoRedemptionReleased
	redemption.ReleasedAt = now
	redemption.UpdatedAt = now
	if err := tx.Save(campaign).Error; err != nil {
		return err
	}
	return tx.Save(redemption).Error
}

func IssueTopupPromotion(orderID string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return IssueTopupPromotionWithTx(tx, orderID)
	})
}

func IssueTopupPromotionWithTx(tx *gorm.DB, orderID string) error {
	redemption, campaign, err := lockTopupRedemptionAndCampaign(tx, orderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if redemption.Status == TopupPromoRedemptionIssued {
		return nil
	}
	if redemption.Status != TopupPromoRedemptionReserved {
		return fmt.Errorf("cannot issue promotion in status %s", redemption.Status)
	}
	now := common.GetTimestamp()
	campaign.ReservedPromoMicroUSD -= redemption.PromoMicroUSD
	if campaign.ReservedPromoMicroUSD < 0 {
		campaign.ReservedPromoMicroUSD = 0
	}
	campaign.IssuedPromoMicroUSD += redemption.PromoMicroUSD
	campaign.UpdatedAt = now
	redemption.Status = TopupPromoRedemptionIssued
	redemption.IssuedAt = now
	redemption.UpdatedAt = now
	if err := tx.Save(campaign).Error; err != nil {
		return err
	}
	return tx.Save(redemption).Error
}

func ReverseTopupPromotionWithTx(tx *gorm.DB, orderID string) error {
	redemption, campaign, err := lockTopupRedemptionAndCampaign(tx, orderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if redemption.Status == TopupPromoRedemptionReversed {
		return nil
	}
	if redemption.Status != TopupPromoRedemptionIssued {
		return fmt.Errorf("cannot reverse promotion in status %s", redemption.Status)
	}
	now := common.GetTimestamp()
	campaign.IssuedPromoMicroUSD -= redemption.PromoMicroUSD
	if campaign.IssuedPromoMicroUSD < 0 {
		campaign.IssuedPromoMicroUSD = 0
	}
	campaign.UpdatedAt = now
	redemption.Status = TopupPromoRedemptionReversed
	redemption.ReversedAt = now
	redemption.UpdatedAt = now
	if err := tx.Save(campaign).Error; err != nil {
		return err
	}
	return tx.Save(redemption).Error
}

func lockTopupRedemptionAndCampaign(tx *gorm.DB, orderID string) (*TopupPromoRedemption, *TopupPromotionCampaign, error) {
	var redemption TopupPromoRedemption
	if err := lockForUpdate(tx).Where("order_id = ?", orderID).First(&redemption).Error; err != nil {
		return nil, nil, err
	}
	var campaign TopupPromotionCampaign
	if err := lockForUpdate(tx).First(&campaign, redemption.CampaignID).Error; err != nil {
		return nil, nil, err
	}
	return &redemption, &campaign, nil
}
