package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const LaunchTopupPromotionCampaignID = 1

var (
	ErrTopupPromotionUnavailable   = errors.New("top-up promotion is unavailable")
	ErrTopupPromotionUserLimit     = errors.New("top-up promotion user limit reached")
	ErrTopupPromotionBudgetReached = errors.New("top-up promotion budget reached")
)

type TopupPromotionCampaign struct {
	Id                     int    `json:"id" gorm:"primaryKey"`
	Name                   string `json:"name" gorm:"type:varchar(128);not null"`
	Enabled                bool   `json:"enabled" gorm:"not null"`
	StartAt                int64  `json:"start_at" gorm:"not null;default:0"`
	EndAt                  int64  `json:"end_at" gorm:"not null;default:0"`
	GlobalBudgetMicroUSD   int64  `json:"global_budget_micro_usd" gorm:"not null;default:0"`
	ReservedPromoMicroUSD  int64  `json:"reserved_promo_micro_usd" gorm:"not null;default:0"`
	IssuedPromoMicroUSD    int64  `json:"issued_promo_micro_usd" gorm:"not null;default:0"`
	PerUserLimit           int    `json:"per_user_limit" gorm:"not null;default:0"`
	DefaultPromoExpiryDays int    `json:"default_promo_expiry_days" gorm:"not null;default:30"`
	CreatedAt              int64  `json:"created_at" gorm:"not null"`
	UpdatedAt              int64  `json:"updated_at" gorm:"not null"`
}

func (TopupPromotionCampaign) TableName() string {
	return "topup_promotion_campaigns"
}

func (campaign *TopupPromotionCampaign) ActiveAt(now int64) bool {
	if campaign == nil || !campaign.Enabled {
		return false
	}
	if campaign.StartAt > 0 && now < campaign.StartAt {
		return false
	}
	return campaign.EndAt <= 0 || now <= campaign.EndAt
}

func ValidateTopupPromotionCampaign(campaign *TopupPromotionCampaign) error {
	if campaign == nil {
		return errors.New("campaign is required")
	}
	campaign.Name = strings.TrimSpace(campaign.Name)
	if campaign.Name == "" {
		return errors.New("campaign name is required")
	}
	if campaign.StartAt > 0 && campaign.EndAt > 0 && campaign.EndAt <= campaign.StartAt {
		return errors.New("campaign end time must be after start time")
	}
	if campaign.GlobalBudgetMicroUSD < 0 {
		return errors.New("global promotional budget cannot be negative")
	}
	if campaign.PerUserLimit < 0 {
		return errors.New("per-user redemption limit cannot be negative")
	}
	if campaign.DefaultPromoExpiryDays < 0 || campaign.DefaultPromoExpiryDays > 3650 {
		return errors.New("promotional expiry days must be between 0 and 3650")
	}
	if campaign.GlobalBudgetMicroUSD > 0 && campaign.ReservedPromoMicroUSD+campaign.IssuedPromoMicroUSD > campaign.GlobalBudgetMicroUSD {
		return errors.New("global promotional budget is below already reserved and issued credits")
	}
	return nil
}

func GetTopupPromotionCampaign() (*TopupPromotionCampaign, error) {
	var campaign TopupPromotionCampaign
	if err := DB.First(&campaign, LaunchTopupPromotionCampaignID).Error; err != nil {
		return nil, err
	}
	return &campaign, nil
}

func SaveTopupPromotionCampaign(campaign *TopupPromotionCampaign) error {
	if err := ValidateTopupPromotionCampaign(campaign); err != nil {
		return err
	}
	if campaign.Id == 0 {
		campaign.Id = LaunchTopupPromotionCampaignID
	}
	now := common.GetTimestamp()
	if campaign.CreatedAt == 0 {
		campaign.CreatedAt = now
	}
	campaign.UpdatedAt = now
	return DB.Save(campaign).Error
}

func SeedLaunchTopupPromotion(db *gorm.DB) error {
	if db == nil {
		return errors.New("database is not initialized")
	}
	now := common.GetTimestamp()
	campaign := TopupPromotionCampaign{Id: LaunchTopupPromotionCampaignID}
	if err := db.Where("id = ?", LaunchTopupPromotionCampaignID).Attrs(TopupPromotionCampaign{
		Name:                   "NovaPuraAI launch top-up promotion",
		Enabled:                true,
		PerUserLimit:           0,
		GlobalBudgetMicroUSD:   0,
		DefaultPromoExpiryDays: 30,
		CreatedAt:              now,
		UpdatedAt:              now,
	}).FirstOrCreate(&campaign).Error; err != nil {
		return err
	}

	tiers := []TopupPromoTier{
		{CampaignID: campaign.Id, Code: "launch-cny-10", Name: "¥10 → ¥30", Currency: "cny", PaymentAmountMinor: 1000, BonusAmountMinor: 2000, TotalCreditAmountMinor: 3000, Enabled: true, SortOrder: 10},
		{CampaignID: campaign.Id, Code: "launch-cny-20", Name: "¥20 → ¥80", Currency: "cny", PaymentAmountMinor: 2000, BonusAmountMinor: 6000, TotalCreditAmountMinor: 8000, Enabled: true, Recommended: true, SortOrder: 20},
		{CampaignID: campaign.Id, Code: "launch-cny-50", Name: "¥50 → ¥250", Currency: "cny", PaymentAmountMinor: 5000, BonusAmountMinor: 20000, TotalCreditAmountMinor: 25000, Enabled: true, SortOrder: 30},
		{CampaignID: campaign.Id, Code: "launch-cny-100", Name: "¥100 → ¥600", Currency: "cny", PaymentAmountMinor: 10000, BonusAmountMinor: 50000, TotalCreditAmountMinor: 60000, Enabled: true, SortOrder: 40},
	}
	for i := range tiers {
		tier := &tiers[i]
		tier.CreatedAt = now
		tier.UpdatedAt = now
		if err := db.Where("code = ?", tier.Code).Attrs(*tier).FirstOrCreate(tier).Error; err != nil {
			return fmt.Errorf("seed top-up tier %s: %w", tier.Code, err)
		}
	}
	bands := []TopupPromoTier{
		{CampaignID: campaign.Id, Code: "launch-band-10", Name: "10-19.99: 2x bonus", Currency: "*", MinPresentmentMajor: 10, MaxPresentmentMajor: 19, PercentBonusBps: 20000, Enabled: true, SortOrder: 110},
		{CampaignID: campaign.Id, Code: "launch-band-20", Name: "20-49.99: 3x bonus", Currency: "*", MinPresentmentMajor: 20, MaxPresentmentMajor: 49, PercentBonusBps: 30000, Enabled: true, SortOrder: 120},
		{CampaignID: campaign.Id, Code: "launch-band-50", Name: "50-99.99: 4x bonus", Currency: "*", MinPresentmentMajor: 50, MaxPresentmentMajor: 99, PercentBonusBps: 40000, Enabled: true, SortOrder: 130},
		{CampaignID: campaign.Id, Code: "launch-band-100", Name: "100-199.99: 5x bonus", Currency: "*", MinPresentmentMajor: 100, MaxPresentmentMajor: 199, PercentBonusBps: 50000, Enabled: true, Recommended: true, SortOrder: 140},
		{CampaignID: campaign.Id, Code: "launch-band-200", Name: "200-499.99: 6x bonus", Currency: "*", MinPresentmentMajor: 200, MaxPresentmentMajor: 499, PercentBonusBps: 60000, Enabled: true, SortOrder: 150},
		{CampaignID: campaign.Id, Code: "launch-band-500", Name: "500+: 7x bonus", Currency: "*", MinPresentmentMajor: 500, MaxPresentmentMajor: 0, PercentBonusBps: 70000, Enabled: true, SortOrder: 160},
	}
	for i := range bands {
		band := &bands[i]
		band.CreatedAt = now
		band.UpdatedAt = now
		if err := db.Where("code = ?", band.Code).Attrs(*band).FirstOrCreate(band).Error; err != nil {
			return fmt.Errorf("seed top-up band %s: %w", band.Code, err)
		}
	}
	return nil
}

func topupPromotionNow() int64 {
	return time.Now().Unix()
}
