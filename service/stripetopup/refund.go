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

// ReconcileRefund reverses promotional credits fully when possible, then paid credits.
// If the user has already spent more than can be reversed safely, order → manual_review
// and remaining balance is clamped to zero (never uncontrolled negative).
func ReconcileRefund(ctx context.Context, orderID string) error {
	if orderID == "" {
		return errors.New("missing order id")
	}
	orderCol := "`order_id`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		orderCol = `"order_id"`
	}

	return model.DB.Transaction(func(tx *gorm.DB) error {
		o := &model.StripeTopupOrder{}
		if err := model.LockForUpdate(tx).Where(orderCol+" = ?", orderID).First(o).Error; err != nil {
			return err
		}
		if o.Status == model.StripeOrderRefunded {
			return nil
		}
		if o.Status != model.StripeOrderCredited && o.Status != model.StripeOrderRefundPending {
			// allow refunds only after credit
			if o.Status != model.StripeOrderPaid {
				o.Status = model.StripeOrderManualReview
				o.FailureReason = "refund on non-credited order"
				return tx.Save(o).Error
			}
		}

		user := &model.User{}
		if err := model.LockForUpdate(tx).Where("id = ?", o.UserId).First(user).Error; err != nil {
			return err
		}

		// Reverse promo first (from PromoQuota), then paid (from cash = Quota-PromoQuota).
		revPromo := o.PromoQuota
		if revPromo > user.PromoQuota {
			revPromo = user.PromoQuota
		}
		cash := user.Quota - user.PromoQuota
		if cash < 0 {
			cash = 0
		}
		revPaid := o.PaidQuota
		if revPaid > cash {
			revPaid = cash
		}

		needPromo := o.PromoQuota
		needPaid := o.PaidQuota
		shortfall := (needPromo - revPromo) + (needPaid - revPaid)

		newPromo := user.PromoQuota - revPromo
		newQuota := user.Quota - revPromo - revPaid
		if newQuota < 0 {
			newQuota = 0
		}
		if newPromo < 0 {
			newPromo = 0
		}
		if newPromo > newQuota {
			newPromo = newQuota
		}

		now := common.GetTimestamp()
		if err := tx.Model(&model.User{}).Where("id = ?", o.UserId).Updates(map[string]interface{}{
			"quota":       newQuota,
			"promo_quota": newPromo,
		}).Error; err != nil {
			return err
		}

		if revPromo > 0 {
			if err := tx.Create(&model.BalanceLedger{
				UserId:      o.UserId,
				OrderID:     o.OrderID,
				EntryType:   model.LedgerTypeRefundPromo,
				AmountQuota: -revPromo,
				AmountMicro: -o.PromoCreditMicroUSD,
				Currency:    o.PresentmentCurrency,
				Note:        "stripe refund promo reverse",
				CreatedAt:   now,
			}).Error; err != nil {
				return err
			}
		}
		if revPaid > 0 {
			if err := tx.Create(&model.BalanceLedger{
				UserId:      o.UserId,
				OrderID:     o.OrderID,
				EntryType:   model.LedgerTypeRefundPaid,
				AmountQuota: -revPaid,
				AmountMicro: -o.PaidCreditMicroUSD,
				Currency:    o.PresentmentCurrency,
				Note:        "stripe refund paid reverse",
				CreatedAt:   now,
			}).Error; err != nil {
				return err
			}
		}

		if shortfall > 0 {
			o.Status = model.StripeOrderManualReview
			o.FailureReason = fmt.Sprintf("refund shortfall quota=%d (credits already consumed)", shortfall)
			logger.LogWarn(ctx, fmt.Sprintf("stripe refund manual_review order=%s shortfall=%d", orderID, shortfall))
		} else {
			o.Status = model.StripeOrderRefunded
			o.FailureReason = ""
		}
		o.RefundedAt = now
		return tx.Save(o).Error
	})
}
