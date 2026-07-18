package stripetopup

import (
	"context"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

// ReconcileRefund reverses only the remaining paid/promotional lots created by
// the refunded order. Credits already consumed are already absent from the
// aggregate and must not be deducted twice. Historical orders without lots use
// the aggregate fallback retained for backward compatibility.
func ReconcileRefund(ctx context.Context, orderID string) error {
	if orderID == "" {
		return errors.New("missing order id")
	}
	orderCol := "`order_id`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		orderCol = `"order_id"`
	}
	refundedUserID := 0
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		order := &model.StripeTopupOrder{}
		if err := model.LockForUpdate(tx).Where(orderCol+" = ?", orderID).First(order).Error; err != nil {
			return err
		}
		if order.Status == model.StripeOrderRefunded {
			return nil
		}
		if order.Status != model.StripeOrderCredited && order.Status != model.StripeOrderRefundPending && order.Status != model.StripeOrderPaid {
			order.Status = model.StripeOrderManualReview
			order.FailureReason = "refund on non-credited order"
			return tx.Save(order).Error
		}

		user := &model.User{}
		if err := model.LockForUpdate(tx).Where("id = ?", order.UserId).First(user).Error; err != nil {
			return err
		}
		refundedUserID = order.UserId
		var lots []model.BalanceCreditLot
		if err := model.LockForUpdate(tx).Where("order_id = ?", order.OrderID).Order("id asc").Find(&lots).Error; err != nil {
			return err
		}
		if len(lots) == 0 {
			return reconcileLegacyRefundWithTx(ctx, tx, user, order)
		}

		removeTotal := 0
		removePromo := 0
		for _, lot := range lots {
			if lot.RemainingQuota <= 0 || lot.Status == model.CreditLotReversed {
				continue
			}
			if removeTotal > common.MaxQuota-lot.RemainingQuota {
				return errors.New("refund quota overflow")
			}
			removeTotal += lot.RemainingQuota
			if lot.BalanceType == model.BalanceTypePromotional {
				removePromo += lot.RemainingQuota
			}
		}
		if user.Quota < removeTotal || user.PromoQuota < removePromo {
			order.Status = model.StripeOrderManualReview
			order.FailureReason = "credit lot balance invariant failed during refund"
			logger.LogWarn(ctx, fmt.Sprintf("stripe refund manual_review order=%s source lot invariant", orderID))
			return tx.Save(order).Error
		}

		now := common.GetTimestamp()
		for _, lot := range lots {
			amount := lot.RemainingQuota
			if amount <= 0 || lot.Status == model.CreditLotReversed {
				continue
			}
			if err := tx.Model(&model.BalanceCreditLot{}).Where("id = ?", lot.Id).Updates(map[string]any{
				"remaining_quota": 0,
				"status":          model.CreditLotReversed,
				"updated_at":      now,
			}).Error; err != nil {
				return err
			}
			entryType := model.LedgerTypeRefundPaid
			if lot.BalanceType == model.BalanceTypePromotional {
				entryType = model.LedgerTypeRefundPromo
			}
			if err := tx.Create(&model.BalanceLedger{
				UserId:      order.UserId,
				OrderID:     order.OrderID,
				LotID:       lot.Id,
				BalanceType: lot.BalanceType,
				EntryType:   entryType,
				AmountQuota: -amount,
				Currency:    order.PresentmentCurrency,
				Note:        "stripe refund source credit reversal",
				CreatedAt:   now,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.User{}).Where("id = ?", order.UserId).Updates(map[string]any{
			"quota":       user.Quota - removeTotal,
			"promo_quota": user.PromoQuota - removePromo,
		}).Error; err != nil {
			return err
		}
		if err := model.ReverseTopupPromotionWithTx(tx, order.OrderID); err != nil {
			return err
		}
		order.Status = model.StripeOrderRefunded
		order.FailureReason = ""
		order.RefundedAt = now
		return tx.Save(order).Error
	})
	if err != nil {
		return err
	}
	if refundedUserID > 0 {
		_ = model.InvalidateUserCache(refundedUserID)
	}
	return nil
}

func reconcileLegacyRefundWithTx(ctx context.Context, tx *gorm.DB, user *model.User, order *model.StripeTopupOrder) error {
	reversePromo := order.PromoQuota
	if reversePromo > user.PromoQuota {
		reversePromo = user.PromoQuota
	}
	cash := user.Quota - user.PromoQuota
	if cash < 0 {
		cash = 0
	}
	reversePaid := order.PaidQuota
	if reversePaid > cash {
		reversePaid = cash
	}
	shortfall := (order.PromoQuota - reversePromo) + (order.PaidQuota - reversePaid)
	now := common.GetTimestamp()
	if err := tx.Model(&model.User{}).Where("id = ?", order.UserId).Updates(map[string]any{
		"quota":       user.Quota - reversePromo - reversePaid,
		"promo_quota": user.PromoQuota - reversePromo,
	}).Error; err != nil {
		return err
	}
	if reversePromo > 0 {
		if err := tx.Create(&model.BalanceLedger{UserId: order.UserId, OrderID: order.OrderID, BalanceType: model.BalanceTypePromotional, EntryType: model.LedgerTypeRefundPromo, AmountQuota: -reversePromo, Currency: order.PresentmentCurrency, Note: "legacy stripe refund promo reverse", CreatedAt: now}).Error; err != nil {
			return err
		}
	}
	if reversePaid > 0 {
		if err := tx.Create(&model.BalanceLedger{UserId: order.UserId, OrderID: order.OrderID, BalanceType: model.BalanceTypePaid, EntryType: model.LedgerTypeRefundPaid, AmountQuota: -reversePaid, Currency: order.PresentmentCurrency, Note: "legacy stripe refund paid reverse", CreatedAt: now}).Error; err != nil {
			return err
		}
	}
	if shortfall > 0 {
		order.Status = model.StripeOrderManualReview
		order.FailureReason = fmt.Sprintf("legacy refund shortfall quota=%d", shortfall)
		logger.LogWarn(ctx, fmt.Sprintf("stripe legacy refund manual_review order=%s shortfall=%d", order.OrderID, shortfall))
	} else {
		order.Status = model.StripeOrderRefunded
		order.FailureReason = ""
	}
	order.RefundedAt = now
	return tx.Save(order).Error
}
