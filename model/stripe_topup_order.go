package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Stripe top-up order statuses (one-time Checkout only).
const (
	StripeOrderPending        = "pending"
	StripeOrderCheckoutCreated = "checkout_created"
	StripeOrderPaid           = "paid"
	StripeOrderCredited       = "credited"
	StripeOrderFailed         = "failed"
	StripeOrderExpired        = "expired"
	StripeOrderRefundPending  = "refund_pending"
	StripeOrderRefunded       = "refunded"
	StripeOrderManualReview   = "manual_review"
)

// StripeTopupOrder is the durable one-time top-up order for NovaPura Stripe Checkout.
type StripeTopupOrder struct {
	Id                       int    `json:"id" gorm:"primaryKey"`
	OrderID                  string `json:"order_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	UserId                   int    `json:"user_id" gorm:"index;not null"`
	StripeCustomerID         string `json:"stripe_customer_id" gorm:"type:varchar(64);index;default:''"`
	StripeCheckoutSessionID  string `json:"stripe_checkout_session_id" gorm:"type:varchar(128);uniqueIndex;default:''"`
	StripePaymentIntentID    string `json:"stripe_payment_intent_id" gorm:"type:varchar(128);index;default:''"`
	Status                   string `json:"status" gorm:"type:varchar(32);index;not null;default:'pending'"`
	PresentmentCurrency      string `json:"presentment_currency" gorm:"type:varchar(8);not null"`
	PresentmentAmountMinor   int64  `json:"presentment_amount_minor" gorm:"not null"`
	// FxRateSnapshot: presentment currency units per 1 USD, locked at order creation.
	FxRateSnapshot           float64 `json:"fx_rate_snapshot" gorm:"type:decimal(18,8);not null"`
	PaidCreditMicroUSD       int64   `json:"paid_credit_micro_usd" gorm:"not null"`
	PromoCreditMicroUSD      int64   `json:"promo_credit_micro_usd" gorm:"not null;default:0"`
	TotalCreditMicroUSD      int64   `json:"total_credit_micro_usd" gorm:"not null"`
	PaidQuota                int     `json:"paid_quota" gorm:"not null;default:0"`
	PromoQuota               int     `json:"promo_quota" gorm:"not null;default:0"`
	PromotionSnapshotJSON    string  `json:"promotion_snapshot_json" gorm:"type:text"`
	PromotionTierID          int     `json:"promotion_tier_id" gorm:"default:0"`
	IdempotencyKey           string  `json:"idempotency_key" gorm:"type:varchar(128);uniqueIndex;default:''"`
	FailureReason            string  `json:"failure_reason" gorm:"type:varchar(512);default:''"`
	CheckoutExpiresAt        int64   `json:"checkout_expires_at" gorm:"default:0"`
	PaidAt                   int64   `json:"paid_at" gorm:"default:0"`
	CreditedAt               int64   `json:"credited_at" gorm:"default:0"`
	RefundedAt               int64   `json:"refunded_at" gorm:"default:0"`
	CreatedAt                int64   `json:"created_at" gorm:"index"`
	UpdatedAt                int64   `json:"updated_at"`
}

func (StripeTopupOrder) TableName() string {
	return "stripe_topup_orders"
}

func (o *StripeTopupOrder) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if o.CreatedAt == 0 {
		o.CreatedAt = now
	}
	o.UpdatedAt = now
	return nil
}

func (o *StripeTopupOrder) BeforeUpdate(tx *gorm.DB) error {
	o.UpdatedAt = time.Now().Unix()
	return nil
}

func CreateStripeTopupOrder(o *StripeTopupOrder) error {
	return DB.Create(o).Error
}

func GetStripeTopupOrderByOrderID(orderID string) (*StripeTopupOrder, error) {
	var o StripeTopupOrder
	err := DB.Where("order_id = ?", orderID).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func GetStripeTopupOrderBySessionID(sessionID string) (*StripeTopupOrder, error) {
	var o StripeTopupOrder
	err := DB.Where("stripe_checkout_session_id = ?", sessionID).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func GetUserStripeTopupOrders(userId int, limit int) ([]*StripeTopupOrder, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var list []*StripeTopupOrder
	err := DB.Where("user_id = ?", userId).Order("id desc").Limit(limit).Find(&list).Error
	return list, err
}

func AdminListStripeTopupOrders(status string, limit, offset int) ([]*StripeTopupOrder, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := DB.Model(&StripeTopupOrder{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*StripeTopupOrder
	err := q.Order("id desc").Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

// CreditStripeTopupOrder atomically credits paid + promo quota and marks order credited.
// Idempotent: if already credited, returns (true, nil).
func CreditStripeTopupOrder(orderID string, customerID, paymentIntentID, sessionID string) (already bool, err error) {
	if orderID == "" {
		return false, errors.New("missing order id")
	}
	orderCol := "`order_id`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		orderCol = `"order_id"`
	}

	var creditedUser int
	var paidQ, promoQ int
	var logMsg string

	err = DB.Transaction(func(tx *gorm.DB) error {
		o := &StripeTopupOrder{}
		if err := lockForUpdate(tx).Where(orderCol+" = ?", orderID).First(o).Error; err != nil {
			return errors.New("order not found")
		}
		if o.Status == StripeOrderCredited {
			already = true
			return nil
		}
		if o.Status == StripeOrderRefunded || o.Status == StripeOrderManualReview {
			return fmt.Errorf("order status %s cannot credit", o.Status)
		}
		if o.Status != StripeOrderCheckoutCreated && o.Status != StripeOrderPending && o.Status != StripeOrderPaid {
			return fmt.Errorf("order status %s cannot credit", o.Status)
		}

		user := &User{}
		if err := lockForUpdate(tx).Where("id = ?", o.UserId).First(user).Error; err != nil {
			return errors.New("user not found")
		}

		paidQ = o.PaidQuota
		promoQ = o.PromoQuota
		if paidQ <= 0 && o.PaidCreditMicroUSD > 0 {
			// fallback if not precomputed
			paidQ = int(float64(o.PaidCreditMicroUSD) / 1e6 * common.QuotaPerUnit)
		}
		if promoQ < 0 {
			promoQ = 0
		}
		totalAdd := paidQ + promoQ
		if totalAdd <= 0 {
			return errors.New("invalid credit amount")
		}
		if user.Quota > common.MaxQuota-totalAdd {
			return errors.New("quota overflow")
		}

		now := common.GetTimestamp()
		o.Status = StripeOrderCredited
		o.PaidAt = now
		o.CreditedAt = now
		if customerID != "" {
			o.StripeCustomerID = customerID
		}
		if paymentIntentID != "" {
			o.StripePaymentIntentID = paymentIntentID
		}
		if sessionID != "" && o.StripeCheckoutSessionID == "" {
			o.StripeCheckoutSessionID = sessionID
		}
		o.PaidQuota = paidQ
		o.PromoQuota = promoQ
		if err := tx.Save(o).Error; err != nil {
			return err
		}

		// Cash (paid): Quota only. Promo: both Quota and PromoQuota.
		updates := map[string]interface{}{
			"quota": user.Quota + totalAdd,
		}
		if promoQ > 0 {
			updates["promo_quota"] = user.PromoQuota + promoQ
		}
		if customerID != "" {
			updates["stripe_customer"] = customerID
		}
		if err := tx.Model(&User{}).Where("id = ?", o.UserId).Updates(updates).Error; err != nil {
			return err
		}

		// Ledger entries (immutable)
		if paidQ > 0 {
			if err := tx.Create(&BalanceLedger{
				UserId:      o.UserId,
				OrderID:     o.OrderID,
				EntryType:   LedgerTypeTopupPaid,
				AmountQuota: paidQ,
				AmountMicro: o.PaidCreditMicroUSD,
				Currency:    o.PresentmentCurrency,
				Note:        "stripe top-up paid credits",
				CreatedAt:   now,
			}).Error; err != nil {
				return err
			}
		}
		if promoQ > 0 {
			if err := tx.Create(&BalanceLedger{
				UserId:      o.UserId,
				OrderID:     o.OrderID,
				EntryType:   LedgerTypeTopupPromo,
				AmountQuota: promoQ,
				AmountMicro: o.PromoCreditMicroUSD,
				Currency:    o.PresentmentCurrency,
				Note:        "stripe top-up promotional credits",
				CreatedAt:   now,
			}).Error; err != nil {
				return err
			}
		}

		// Mirror into legacy TopUp for admin history compatibility
		_ = tx.Create(&TopUp{
			UserId:          o.UserId,
			Amount:          o.PresentmentAmountMinor,
			Money:           float64(o.PresentmentAmountMinor) / 100.0,
			TradeNo:         o.OrderID,
			PaymentMethod:   PaymentMethodStripe,
			PaymentProvider: PaymentProviderStripe,
			CreateTime:      o.CreatedAt,
			CompleteTime:    now,
			Status:          common.TopUpStatusSuccess,
		}).Error

		creditedUser = o.UserId
		logMsg = fmt.Sprintf("Stripe top-up credited order=%s paid_quota=%d promo_quota=%d currency=%s minor=%d",
			o.OrderID, paidQ, promoQ, o.PresentmentCurrency, o.PresentmentAmountMinor)
		return nil
	})
	if err != nil {
		return false, err
	}
	if already {
		return true, nil
	}
	if creditedUser > 0 {
		_ = invalidateUserCache(creditedUser)
		RecordTopupLog(creditedUser, logMsg, "", PaymentMethodStripe, PaymentMethodStripe)
	}
	return false, nil
}

// MarkStripeTopupOrderStatus updates pending/checkout orders only (idempotent guards inside).
func MarkStripeTopupOrderStatus(orderID, fromStatus, toStatus, reason string) error {
	orderCol := "`order_id`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		orderCol = `"order_id"`
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		o := &StripeTopupOrder{}
		if err := lockForUpdate(tx).Where(orderCol+" = ?", orderID).First(o).Error; err != nil {
			return ErrTopUpNotFound
		}
		if fromStatus != "" && o.Status != fromStatus {
			if o.Status == toStatus {
				return nil
			}
			// allow multi-from for fail paths
			if !(fromStatus == "*" || o.Status == StripeOrderPending || o.Status == StripeOrderCheckoutCreated || o.Status == StripeOrderPaid) {
				return ErrTopUpStatusInvalid
			}
		}
		if o.Status == StripeOrderCredited || o.Status == StripeOrderRefunded {
			return ErrTopUpStatusInvalid
		}
		o.Status = toStatus
		if reason != "" {
			o.FailureReason = reason
		}
		if toStatus == StripeOrderExpired || toStatus == StripeOrderFailed {
			// keep
		}
		return tx.Save(o).Error
	})
}
