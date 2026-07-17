package model

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// TopupPromoTier is an admin-configurable bonus tier for Stripe top-ups.
type TopupPromoTier struct {
	Id                 int     `json:"id" gorm:"primaryKey"`
	Name               string  `json:"name" gorm:"type:varchar(128);default:''"`
	Currency           string  `json:"currency" gorm:"type:varchar(8);index;default:'*'" ` // * = all
	MinPresentmentMajor int    `json:"min_presentment_major" gorm:"default:0"`
	MaxPresentmentMajor int    `json:"max_presentment_major" gorm:"default:0"` // 0 = no upper bound
	FixedBonusMajor    int     `json:"fixed_bonus_major" gorm:"default:0"`     // fixed promo in major units of same currency
	PercentBonusBps    int     `json:"percent_bonus_bps" gorm:"default:0"`     // 1000 = 10.00%
	FirstTopupOnly     bool    `json:"first_topup_only" gorm:"default:false"`
	PerUserLimit       int     `json:"per_user_limit" gorm:"default:0"` // 0 = unlimited redemptions
	Enabled            bool    `json:"enabled" gorm:"default:true"`
	StartAt            int64   `json:"start_at" gorm:"default:0"`
	EndAt              int64   `json:"end_at" gorm:"default:0"`
	CreatedAt          int64   `json:"created_at"`
	UpdatedAt          int64   `json:"updated_at"`
}

// TableName uses TopupPromoTier naming expected by admin.
func (TopupPromoTier) TableName() string {
	return "topup_promo_tiers"
}

// Alias type name for clarity in code.
type TopupPromoTierModel = TopupPromoTier

func ListEnabledPromoTiers() ([]*TopupPromoTier, error) {
	if DB == nil {
		return nil, nil
	}
	var list []*TopupPromoTier
	err := DB.Where("enabled = ?", true).Order("min_presentment_major desc").Find(&list).Error
	return list, err
}

func ListAllPromoTiers() ([]*TopupPromoTier, error) {
	var list []*TopupPromoTier
	err := DB.Order("id desc").Find(&list).Error
	return list, err
}

func SavePromoTier(t *TopupPromoTier) error {
	now := common.GetTimestamp()
	if t.Id == 0 {
		t.CreatedAt = now
		t.UpdatedAt = now
		return DB.Create(t).Error
	}
	t.UpdatedAt = now
	return DB.Save(t).Error
}

// PromoCalcResult is a server-side bonus snapshot.
type PromoCalcResult struct {
	TierID              int     `json:"tier_id"`
	TierName            string  `json:"tier_name"`
	PercentBonusBps     int     `json:"percent_bonus_bps"`
	FixedBonusMajor     int     `json:"fixed_bonus_major"`
	PromoCreditMicroUSD int64   `json:"promo_credit_micro_usd"`
	PromoQuota          int     `json:"promo_quota"`
	SnapshotJSON        string  `json:"snapshot_json"`
}

// CalculateTopupPromo selects the best applicable tier and returns promo micro-USD + quota.
// presentmentMajor is whole currency units (e.g. 100 for ¥100 / $100).
func CalculateTopupPromo(userId int, currency string, presentmentMajor int, paidMicro int64, paidQuota int) (*PromoCalcResult, error) {
	tiers, err := ListEnabledPromoTiers()
	if err != nil {
		return &PromoCalcResult{SnapshotJSON: `{"applied":false}`}, nil
	}
	now := time.Now().Unix()
	currency = strings.ToLower(strings.TrimSpace(currency))

	var best *TopupPromoTier
	var bestPromoMicro int64

	// Count prior successful stripe topups for first-only / per-user limits
	var creditedCount int64
	if DB != nil {
		_ = DB.Model(&StripeTopupOrder{}).
			Where("user_id = ? AND status = ?", userId, StripeOrderCredited).
			Count(&creditedCount).Error
	}

	for _, t := range tiers {
		if t == nil || !t.Enabled {
			continue
		}
		if t.StartAt > 0 && now < t.StartAt {
			continue
		}
		if t.EndAt > 0 && now > t.EndAt {
			continue
		}
		tc := strings.ToLower(strings.TrimSpace(t.Currency))
		if tc != "" && tc != "*" && tc != currency {
			continue
		}
		if presentmentMajor < t.MinPresentmentMajor {
			continue
		}
		if t.MaxPresentmentMajor > 0 && presentmentMajor > t.MaxPresentmentMajor {
			continue
		}
		if t.FirstTopupOnly && creditedCount > 0 {
			continue
		}
		if t.PerUserLimit > 0 && DB != nil {
			var used int64
			_ = DB.Model(&StripeTopupOrder{}).
				Where("user_id = ? AND status = ? AND promotion_tier_id = ?", userId, StripeOrderCredited, t.Id).
				Count(&used).Error
			if used >= int64(t.PerUserLimit) {
				continue
			}
		}

		// Promo micro from percent of paid + fixed major converted at same ratio as paid
		promoMicro := int64(0)
		if t.PercentBonusBps > 0 && paidMicro > 0 {
			promoMicro += paidMicro * int64(t.PercentBonusBps) / 10000
		}
		if t.FixedBonusMajor > 0 && paidMicro > 0 && presentmentMajor > 0 {
			// fixed major → micro proportional to paidMicro/presentmentMajor
			promoMicro += paidMicro * int64(t.FixedBonusMajor) / int64(presentmentMajor)
		}
		if promoMicro <= 0 {
			continue
		}
		if best == nil || promoMicro > bestPromoMicro {
			best = t
			bestPromoMicro = promoMicro
		}
	}

	if best == nil {
		snap, _ := common.Marshal(map[string]any{"applied": false})
		return &PromoCalcResult{SnapshotJSON: string(snap)}, nil
	}

	promoQuota := 0
	if paidQuota > 0 && paidMicro > 0 {
		promoQuota = int(int64(paidQuota) * bestPromoMicro / paidMicro)
	}
	snap, _ := common.Marshal(map[string]any{
		"applied":            true,
		"tier_id":            best.Id,
		"tier_name":          best.Name,
		"percent_bonus_bps":  best.PercentBonusBps,
		"fixed_bonus_major":  best.FixedBonusMajor,
		"promo_micro_usd":    bestPromoMicro,
		"promo_quota":        promoQuota,
		"currency":           currency,
		"presentment_major":  presentmentMajor,
	})
	return &PromoCalcResult{
		TierID:              best.Id,
		TierName:            best.Name,
		PercentBonusBps:     best.PercentBonusBps,
		FixedBonusMajor:     best.FixedBonusMajor,
		PromoCreditMicroUSD: bestPromoMicro,
		PromoQuota:          promoQuota,
		SnapshotJSON:        string(snap),
	}, nil
}
