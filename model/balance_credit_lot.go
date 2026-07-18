package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	BalanceTypePaid        = "paid"
	BalanceTypePromotional = "promotional"

	CreditLotActive    = "active"
	CreditLotExhausted = "exhausted"
	CreditLotExpired   = "expired"
	CreditLotReversed  = "reversed"
)

type BalanceCreditLot struct {
	Id                  int64  `json:"id" gorm:"primaryKey"`
	UserId              int    `json:"user_id" gorm:"index;not null"`
	OrderID             string `json:"order_id" gorm:"type:varchar(64);uniqueIndex:idx_credit_lot_order_type;not null"`
	BalanceType         string `json:"balance_type" gorm:"type:varchar(16);uniqueIndex:idx_credit_lot_order_type;index;not null"`
	OriginalQuota       int    `json:"original_quota" gorm:"not null"`
	RemainingQuota      int    `json:"remaining_quota" gorm:"not null"`
	Currency            string `json:"currency" gorm:"type:varchar(8);not null"`
	OriginalAmountMinor int64  `json:"original_amount_minor" gorm:"not null;default:0"`
	ExpiresAt           int64  `json:"expires_at" gorm:"index;not null;default:0"`
	Status              string `json:"status" gorm:"type:varchar(16);index;not null"`
	CreatedAt           int64  `json:"created_at" gorm:"index;not null"`
	UpdatedAt           int64  `json:"updated_at" gorm:"not null"`
}

func (BalanceCreditLot) TableName() string {
	return "balance_credit_lots"
}

type BalanceLotAllocation struct {
	LotID       int64  `json:"lot_id"`
	OrderID     string `json:"order_id,omitempty"`
	BalanceType string `json:"balance_type"`
	Amount      int    `json:"amount"`
	ExpiresAt   int64  `json:"expires_at,omitempty"`
}

func createTopupCreditLotWithTx(tx *gorm.DB, lot *BalanceCreditLot) error {
	if tx == nil || lot == nil || lot.UserId <= 0 || lot.OrderID == "" || lot.OriginalQuota <= 0 {
		return errors.New("invalid balance credit lot")
	}
	if lot.BalanceType != BalanceTypePaid && lot.BalanceType != BalanceTypePromotional {
		return errors.New("invalid balance type")
	}
	if lot.BalanceType == BalanceTypePaid {
		lot.ExpiresAt = 0
	}
	lot.RemainingQuota = lot.OriginalQuota
	lot.Status = CreditLotActive
	now := common.GetTimestamp()
	if lot.CreatedAt == 0 {
		lot.CreatedAt = now
	}
	lot.UpdatedAt = now
	return tx.Create(lot).Error
}

func ExpireUserPromotionLots(userID int) (int, error) {
	if userID <= 0 {
		return 0, errors.New("invalid user id")
	}
	expired := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id", "quota", "promo_quota").First(&user, userID).Error; err != nil {
			return err
		}
		var err error
		expired, err = expireUserPromotionLotsWithTx(tx, &user, time.Now().Unix())
		return err
	})
	if err != nil {
		return 0, err
	}
	if expired > 0 {
		_ = invalidateUserCache(userID)
	}
	return expired, nil
}

func expireUserPromotionLotsWithTx(tx *gorm.DB, user *User, now int64) (int, error) {
	var lots []BalanceCreditLot
	if err := lockForUpdate(tx).
		Where("user_id = ? AND balance_type = ? AND status = ? AND remaining_quota > 0 AND expires_at > 0 AND expires_at <= ?", user.Id, BalanceTypePromotional, CreditLotActive, now).
		Order("expires_at asc, id asc").Find(&lots).Error; err != nil {
		return 0, err
	}
	if len(lots) == 0 {
		return 0, nil
	}

	expired := 0
	for i := range lots {
		lot := &lots[i]
		amount := lot.RemainingQuota
		if amount <= 0 {
			continue
		}
		expired += amount
		if err := tx.Model(&BalanceCreditLot{}).Where("id = ?", lot.Id).Updates(map[string]any{
			"remaining_quota": 0,
			"status":          CreditLotExpired,
			"updated_at":      now,
		}).Error; err != nil {
			return 0, err
		}
		if err := tx.Create(&BalanceLedger{
			UserId:      user.Id,
			OrderID:     lot.OrderID,
			LotID:       lot.Id,
			BalanceType: BalanceTypePromotional,
			EntryType:   LedgerTypeExpiration,
			AmountQuota: -amount,
			Currency:    lot.Currency,
			Note:        "promotional credits expired",
			CreatedAt:   now,
		}).Error; err != nil {
			return 0, err
		}
	}

	remove := expired
	if remove > user.PromoQuota {
		remove = user.PromoQuota
	}
	if remove > user.Quota {
		remove = user.Quota
	}
	user.PromoQuota -= remove
	user.Quota -= remove
	if err := tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"quota":       user.Quota,
		"promo_quota": user.PromoQuota,
	}).Error; err != nil {
		return 0, err
	}
	return remove, nil
}

func decreaseUserQuotaWithLots(id, amount int) (QuotaWalletSplit, error) {
	var split QuotaWalletSplit
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id", "quota", "promo_quota").First(&user, id).Error; err != nil {
			return err
		}
		expired, err := expireUserPromotionLotsWithTx(tx, &user, time.Now().Unix())
		if err != nil {
			return err
		}
		split.Expired = expired
		if user.Quota < amount {
			return fmt.Errorf("insufficient quota: have %d need %d", user.Quota, amount)
		}

		remaining := amount
		promoLots, err := activeCreditLotsWithTx(tx, id, BalanceTypePromotional)
		if err != nil {
			return err
		}
		trackedPromo := sumRemainingCreditLots(promoLots)
		for i := range promoLots {
			if remaining == 0 {
				break
			}
			allocation, err := consumeCreditLotWithTx(tx, &promoLots[i], remaining)
			if err != nil {
				return err
			}
			if allocation.Amount > 0 {
				split.Allocations = append(split.Allocations, allocation)
				split.Promo += allocation.Amount
				remaining -= allocation.Amount
			}
		}
		untrackedPromo := user.PromoQuota - trackedPromo
		if untrackedPromo < 0 {
			untrackedPromo = 0
		}
		if take := minInt(remaining, untrackedPromo); take > 0 {
			split.Allocations = append(split.Allocations, BalanceLotAllocation{BalanceType: BalanceTypePromotional, Amount: take})
			split.Promo += take
			remaining -= take
		}

		paidLots, err := activeCreditLotsWithTx(tx, id, BalanceTypePaid)
		if err != nil {
			return err
		}
		for i := range paidLots {
			if remaining == 0 {
				break
			}
			allocation, err := consumeCreditLotWithTx(tx, &paidLots[i], remaining)
			if err != nil {
				return err
			}
			if allocation.Amount > 0 {
				split.Allocations = append(split.Allocations, allocation)
				split.Cash += allocation.Amount
				remaining -= allocation.Amount
			}
		}
		if remaining > 0 {
			split.Allocations = append(split.Allocations, BalanceLotAllocation{BalanceType: BalanceTypePaid, Amount: remaining})
			split.Cash += remaining
			remaining = 0
		}

		user.Quota -= amount
		user.PromoQuota -= split.Promo
		if user.PromoQuota < 0 {
			return errors.New("promotional balance invariant violated")
		}
		if err := tx.Model(&User{}).Where("id = ?", id).Updates(map[string]any{
			"quota":       user.Quota,
			"promo_quota": user.PromoQuota,
		}).Error; err != nil {
			return err
		}
		now := common.GetTimestamp()
		for _, allocation := range split.Allocations {
			if err := tx.Create(&BalanceLedger{
				UserId:      id,
				OrderID:     allocation.OrderID,
				LotID:       allocation.LotID,
				BalanceType: allocation.BalanceType,
				EntryType:   LedgerTypeAPIUsage,
				AmountQuota: -allocation.Amount,
				Note:        "API usage",
				CreatedAt:   now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return split, err
}

func activeCreditLotsWithTx(tx *gorm.DB, userID int, balanceType string) ([]BalanceCreditLot, error) {
	var lots []BalanceCreditLot
	query := lockForUpdate(tx).Where("user_id = ? AND balance_type = ? AND status = ? AND remaining_quota > 0", userID, balanceType, CreditLotActive)
	if balanceType == BalanceTypePromotional {
		query = query.Order("CASE WHEN expires_at = 0 THEN 1 ELSE 0 END asc, expires_at asc, id asc")
	} else {
		query = query.Order("id asc")
	}
	err := query.Find(&lots).Error
	return lots, err
}

func sumRemainingCreditLots(lots []BalanceCreditLot) int {
	total := 0
	for _, lot := range lots {
		if lot.RemainingQuota > 0 && total <= common.MaxQuota-lot.RemainingQuota {
			total += lot.RemainingQuota
		}
	}
	return total
}

func consumeCreditLotWithTx(tx *gorm.DB, lot *BalanceCreditLot, requested int) (BalanceLotAllocation, error) {
	allocation := BalanceLotAllocation{}
	if lot == nil || requested <= 0 || lot.RemainingQuota <= 0 {
		return allocation, nil
	}
	take := minInt(requested, lot.RemainingQuota)
	remaining := lot.RemainingQuota - take
	status := CreditLotActive
	if remaining == 0 {
		status = CreditLotExhausted
	}
	if err := tx.Model(&BalanceCreditLot{}).Where("id = ?", lot.Id).Updates(map[string]any{
		"remaining_quota": remaining,
		"status":          status,
		"updated_at":      common.GetTimestamp(),
	}).Error; err != nil {
		return allocation, err
	}
	lot.RemainingQuota = remaining
	lot.Status = status
	return BalanceLotAllocation{
		LotID:       lot.Id,
		OrderID:     lot.OrderID,
		BalanceType: lot.BalanceType,
		Amount:      take,
		ExpiresAt:   lot.ExpiresAt,
	}, nil
}

func restoreUserQuotaSplitWithLots(id int, split QuotaWalletSplit) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id", "quota", "promo_quota").First(&user, id).Error; err != nil {
			return err
		}
		if len(split.Allocations) == 0 {
			return restoreLegacyQuotaSplitWithTx(tx, &user, split)
		}

		restoredPromo := 0
		restoredCash := 0
		now := common.GetTimestamp()
		for _, allocation := range split.Allocations {
			if allocation.Amount <= 0 {
				continue
			}
			if allocation.LotID > 0 {
				var lot BalanceCreditLot
				if err := lockForUpdate(tx).First(&lot, allocation.LotID).Error; err != nil {
					return err
				}
				if lot.UserId != id || lot.BalanceType != allocation.BalanceType || lot.RemainingQuota > lot.OriginalQuota-allocation.Amount {
					return errors.New("credit lot restore invariant violated")
				}
				if lot.BalanceType == BalanceTypePromotional && lot.ExpiresAt > 0 && lot.ExpiresAt <= now {
					return errors.New("cannot restore expired promotional credits")
				}
				if lot.Status == CreditLotExpired || lot.Status == CreditLotReversed {
					return fmt.Errorf("cannot restore credit lot in status %s", lot.Status)
				}
				if err := tx.Model(&BalanceCreditLot{}).Where("id = ?", lot.Id).Updates(map[string]any{
					"remaining_quota": lot.RemainingQuota + allocation.Amount,
					"status":          CreditLotActive,
					"updated_at":      now,
				}).Error; err != nil {
					return err
				}
			}
			if allocation.BalanceType == BalanceTypePromotional {
				restoredPromo += allocation.Amount
			} else {
				restoredCash += allocation.Amount
			}
			if err := tx.Create(&BalanceLedger{
				UserId:      id,
				OrderID:     allocation.OrderID,
				LotID:       allocation.LotID,
				BalanceType: allocation.BalanceType,
				EntryType:   LedgerTypeReversal,
				AmountQuota: allocation.Amount,
				Note:        "API usage reservation restored",
				CreatedAt:   now,
			}).Error; err != nil {
				return err
			}
		}
		total := restoredPromo + restoredCash
		if total > common.MaxQuota-user.Quota {
			return errors.New("quota overflow")
		}
		return tx.Model(&User{}).Where("id = ?", id).Updates(map[string]any{
			"quota":       user.Quota + total,
			"promo_quota": user.PromoQuota + restoredPromo,
		}).Error
	})
}

func restoreLegacyQuotaSplitWithTx(tx *gorm.DB, user *User, split QuotaWalletSplit) error {
	total := split.Promo + split.Cash
	if total <= 0 {
		return nil
	}
	if total > common.MaxQuota-user.Quota {
		return errors.New("quota overflow")
	}
	return tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"quota":       user.Quota + total,
		"promo_quota": user.PromoQuota + split.Promo,
	}).Error
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
