package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"gorm.io/gorm"
)

// Commission records a single cash commission credit to an inviter for one
// invitee top-up. One row per (topup_id, inviter_id) — the UniqueIndex enforces
// idempotency so a top-up never pays commission twice.
//
// Lifecycle: pending (frozen, in inviter.PendingCommissionCents) → available
// (released by ReleaseMaturedCommissions to CommissionBalanceCents) → reverted
// (MVP+ refund clawback; field reserved, not used in MVP).
type Commission struct {
	ID              int64   `json:"id" gorm:"primaryKey"`
	InviterId       int     `json:"inviter_id" gorm:"column:inviter_id;index;uniqueIndex:idx_commission_topup_inviter"`
	InviteeId       int     `json:"invitee_id" gorm:"column:invitee_id;index"`
	TopUpId         string  `json:"topup_id" gorm:"column:topup_id;type:varchar(255);index;uniqueIndex:idx_commission_topup_inviter"`
	PaidAmountCents int64   `json:"paid_amount_cents" gorm:"column:paid_amount_cents"`
	PaidCurrency    string  `json:"paid_currency" gorm:"column:paid_currency;type:varchar(8)"`
	Rate            float64 `json:"rate" gorm:"column:rate"`
	CommissionCents int64   `json:"commission_cents" gorm:"column:commission_cents"`
	Status          string  `json:"status" gorm:"column:status;type:varchar(16);index"`
	AvailableAt     int64   `json:"available_at" gorm:"column:available_at;index"`
	RevertedAt      int64   `json:"reverted_at" gorm:"column:reverted_at"`
	RevertReason    string  `json:"revert_reason" gorm:"column:revert_reason;type:text"`
	CreatedAt       int64   `json:"created_at" gorm:"autoCreateTime;column:created_at"`
}

const (
	CommissionStatusPending   = "pending"
	CommissionStatusAvailable = "available"
	CommissionStatusReverted  = "reverted"
)

// commissionReleaseBatchSize bounds the number of rows ReleaseMaturedCommissions
// processes per transaction, keeping lock holding time bounded.
const commissionReleaseBatchSize = 100

// ErrCommissionNotFound is returned by RevertCommission when no commission row
// exists for the given topUpId. Refund handlers treat this as "no commission
// was credited for this payment" (e.g. the user had no inviter) and ignore it.
var ErrCommissionNotFound = errors.New("commission record not found")

// clampAffCommissionRate clamps the admin-configured commission rate to [0,1]
// and rejects NaN/Inf (returns 0). Per AGENTS.md billing-safety: any
// user-controlled multiplier on a billing path must be bounded.
func clampAffCommissionRate(rate float64) float64 {
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 0
	}
	if rate < 0 {
		return 0
	}
	if rate > 1 {
		return 1
	}
	return rate
}

// ConvertAmountToUSDCents converts a paid amount (in major units, e.g. 10.00 =
// $10) in the given ISO currency to USD cents (int64). USD passes through
// directly; CNY converts via setting.EffectiveUSDCNYRate; any other currency
// safe-fails to 0 (commission skipped, never negative). Used by recharge entry
// points to derive the commission base from each provider's payment-success
// callback. Saturates via QuotaFromFloatChecked so a corrupt callback cannot
// overflow into a negative commission.
func ConvertAmountToUSDCents(amount float64, currency string) int64 {
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0
	}
	cur := strings.ToUpper(strings.TrimSpace(currency))
	var usdMajor float64
	switch cur {
	case "USD", "":
		usdMajor = amount
	case "CNY":
		rate := setting.EffectiveUSDCNYRate(operation_setting.USDExchangeRate)
		if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
			return 0
		}
		usdMajor = amount / rate
	default:
		// Unsupported currency — safe-fail (no commission). Caller logs.
		return 0
	}
	if usdMajor <= 0 || math.IsNaN(usdMajor) || math.IsInf(usdMajor, 0) {
		return 0
	}
	cents, clamp := common.QuotaFromFloatChecked(usdMajor * 100)
	if clamp != nil {
		common.SysError(fmt.Sprintf("commission paid-amount saturation: amount=%g currency=%s usd_major=%g clamp=%s clamped=%d",
			amount, cur, usdMajor, clamp.Kind, cents))
	}
	return int64(cents)
}

// CommissionSettlement describes a credited commission. Returned by
// SettleRechargeCommission so the caller can write a post-commit user log
// without re-querying. Nil when no commission was credited (no inviter, not
// approved, idempotent skip, commissionCents<=0, etc.).
type CommissionSettlement struct {
	InviterId       int
	InviteeId       int
	TopUpId         string
	CommissionCents int64
	AvailableAt     int64
}

// SettleRechargeCommission credits an approved inviter's cash commission for one
// successful invitee top-up. Must be called inside the same transaction that
// flips the top-up status Pending→Success (so the top-up's own idempotency
// protects commission idempotency too).
//
// Commission enters PendingCommissionCents (frozen); ReleaseMaturedCommissions
// later moves it to CommissionBalanceCents (withdrawable) after
// CommissionFreezeDays. This hold mitigates refund/chargeback risk.
//
// paidAmountCents: actual paid amount captured from the provider's success
// callback, already converted to USD cents (1 USD = 100). NOT topUp.Money/Amount
// (those may include group-ratio/gift markups and would over-pay commission).
// paidCurrency: ISO code of the original payment (USD/CNY/...), kept for audit.
//
// Returns a non-nil *CommissionSettlement when a commission was actually
// credited; nil with nil err when skipped.
func SettleRechargeCommission(tx *gorm.DB, inviteeId int, topUpId string, paidAmountCents int64, paidCurrency string) (*CommissionSettlement, error) {
	if paidAmountCents <= 0 || inviteeId <= 0 || topUpId == "" {
		return nil, nil
	}

	var invitee User
	if err := lockForUpdate(tx).First(&invitee, inviteeId).Error; err != nil {
		return nil, err
	}
	if invitee.InviterId == 0 || invitee.InviterId == inviteeId {
		// No inviter or self-invite: no commission.
		return nil, nil
	}

	var inviter User
	if err := lockForUpdate(tx).First(&inviter, invitee.InviterId).Error; err != nil {
		return nil, err
	}
	if !inviter.CommissionApproved {
		// Not an approved affiliate; the legacy ¥100 invite path handles them.
		return nil, nil
	}

	// Idempotency: skip if this top-up already credited commission to this inviter.
	var existing Commission
	err := tx.Where("topup_id = ? AND inviter_id = ?", topUpId, inviter.Id).First(&existing).Error
	if err == nil {
		return nil, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	rate := clampAffCommissionRate(common.AffCommissionRate)
	// QuotaFromFloatChecked saturates to int32 range. A single commission would
	// only hit that cap at ~$21M (MaxInt32 cents), i.e. an ~$84M single payment —
	// not realistic. On clamp, log and proceed with the saturated value.
	commissionInt, clamp := common.QuotaFromFloatChecked(float64(paidAmountCents) * rate)
	if clamp != nil {
		common.SysError(fmt.Sprintf("commission saturation: topup=%s inviter=%d paid_cents=%d rate=%g clamp=%s clamped=%d",
			topUpId, inviter.Id, paidAmountCents, rate, clamp.Kind, commissionInt))
	}
	commissionCents := int64(commissionInt)
	if commissionCents <= 0 {
		return nil, nil
	}

	now := time.Now().Unix()
	freezeSeconds := int64(common.CommissionFreezeDays) * 86400
	if common.CommissionFreezeDays < 0 {
		freezeSeconds = 0
	}
	availableAt := now + freezeSeconds

	if err := tx.Create(&Commission{
		InviterId:       inviter.Id,
		InviteeId:       inviteeId,
		TopUpId:         topUpId,
		PaidAmountCents: paidAmountCents,
		PaidCurrency:    paidCurrency,
		Rate:            rate,
		CommissionCents: commissionCents,
		Status:          CommissionStatusPending,
		AvailableAt:     availableAt,
	}).Error; err != nil {
		return nil, err
	}

	inviter.PendingCommissionCents += commissionCents
	inviter.CommissionTotalCents += commissionCents
	if err := tx.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]any{
		"pending_commission_cents": inviter.PendingCommissionCents,
		"commission_total_cents":   inviter.CommissionTotalCents,
	}).Error; err != nil {
		return nil, err
	}

	return &CommissionSettlement{
		InviterId:       inviter.Id,
		InviteeId:       inviteeId,
		TopUpId:         topUpId,
		CommissionCents: commissionCents,
		AvailableAt:     availableAt,
	}, nil
}

// RevertCommission reverses a previously-settled commission. It marks the
// commission row as "reverted" and debits the inviter's commission balance
// (or pending, if not yet matured) by the commission amount. The inviter is
// looked up from the commission row itself, so callers only need the topUpId.
//
// Idempotent: if the commission is already reverted, returns nil without
// re-debiting. If the commission doesn't exist, returns ErrCommissionNotFound
// (refund handlers usually treat this as a non-error since not every payment
// has an associated commission — e.g. the user had no inviter).
//
// Billing safety: underflow is guarded on every debit. If the inviter's
// PendingCommissionCents or CommissionBalanceCents is less than the commission
// amount (a race or manual data corruption), the value is clamped to 0 and the
// anomaly is logged via common.SysError. A revert must NEVER produce a negative
// balance (that would be a credit to the inviter from arithmetic underflow).
//
// topUpId is the identifier used when the commission was originally settled:
// for top-ups it is the top-up TradeNo; for subscription purchases it is the
// SubscriptionOrder TradeNo; for subscription renewals it is the Stripe
// invoice ID. Must be called inside a transaction (the caller wraps it in
// DB.Transaction) so the commission row update and the inviter balance debit
// commit atomically.
func RevertCommission(tx *gorm.DB, topUpId string, reason string) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if topUpId == "" {
		return errors.New("topUpId is empty")
	}

	var commission Commission
	if err := lockForUpdate(tx).Where("topup_id = ?", topUpId).First(&commission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCommissionNotFound
		}
		return err
	}

	// Idempotent: a reverted commission is a no-op.
	if commission.Status == CommissionStatusReverted {
		return nil
	}

	// Only pending or available commissions can be reverted. Any other status
	// is a state-machine violation (shouldn't happen given the lifecycle).
	if commission.Status != CommissionStatusPending && commission.Status != CommissionStatusAvailable {
		return fmt.Errorf("commission %d cannot be reverted from status %s", commission.ID, commission.Status)
	}

	var inviter User
	if err := lockForUpdate(tx).First(&inviter, commission.InviterId).Error; err != nil {
		return err
	}

	updates := map[string]any{}

	// Debit the balance that currently holds the commission. Pending → debit
	// PendingCommissionCents; Available → debit CommissionBalanceCents.
	switch commission.Status {
	case CommissionStatusPending:
		if inviter.PendingCommissionCents < commission.CommissionCents {
			common.SysError(fmt.Sprintf("commission revert underflow: inviter=%d pending=%d commission=%d topup=%s",
				inviter.Id, inviter.PendingCommissionCents, commission.CommissionCents, topUpId))
			inviter.PendingCommissionCents = 0
		} else {
			inviter.PendingCommissionCents -= commission.CommissionCents
		}
		updates["pending_commission_cents"] = inviter.PendingCommissionCents
	case CommissionStatusAvailable:
		if inviter.CommissionBalanceCents < commission.CommissionCents {
			common.SysError(fmt.Sprintf("commission revert underflow: inviter=%d balance=%d commission=%d topup=%s",
				inviter.Id, inviter.CommissionBalanceCents, commission.CommissionCents, topUpId))
			inviter.CommissionBalanceCents = 0
		} else {
			inviter.CommissionBalanceCents -= commission.CommissionCents
		}
		updates["commission_balance_cents"] = inviter.CommissionBalanceCents
	}

	// CommissionTotalCents is the lifetime-earned counter; a revert decrements
	// it (guarded against underflow — a corrupt state must not go negative).
	if inviter.CommissionTotalCents < commission.CommissionCents {
		common.SysError(fmt.Sprintf("commission revert total underflow: inviter=%d total=%d commission=%d topup=%s",
			inviter.Id, inviter.CommissionTotalCents, commission.CommissionCents, topUpId))
		inviter.CommissionTotalCents = 0
	} else {
		inviter.CommissionTotalCents -= commission.CommissionCents
	}
	updates["commission_total_cents"] = inviter.CommissionTotalCents

	if err := tx.Model(&User{}).Where("id = ?", inviter.Id).Updates(updates).Error; err != nil {
		return err
	}

	now := time.Now().Unix()
	if err := tx.Model(&Commission{}).Where("id = ?", commission.ID).Updates(map[string]any{
		"status":        CommissionStatusReverted,
		"reverted_at":   now,
		"revert_reason": reason,
	}).Error; err != nil {
		return err
	}

	_ = invalidateUserCache(inviter.Id)
	return nil
}

// FormatCommissionCents renders USD cents as "$X.XX" for user-facing logs.
func FormatCommissionCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s$%d.%02d", sign, cents/100, cents%100)
}

// LogCommissionSettlement writes a user-facing system log after a commission
// credit commits. Safe to call with a nil settlement (no-op). Called by
// recharge entry points post-commit so the inviter sees the credit.
func LogCommissionSettlement(s *CommissionSettlement) {
	if s == nil || s.InviterId <= 0 {
		return
	}
	freezeDays := common.CommissionFreezeDays
	if freezeDays > 0 {
		RecordLog(s.InviterId, LogTypeSystem, fmt.Sprintf(
			"现金佣金收入 %s（冻结中，%d 天后可提现）",
			FormatCommissionCents(s.CommissionCents), freezeDays))
	} else {
		RecordLog(s.InviterId, LogTypeSystem, fmt.Sprintf(
			"现金佣金收入 %s（已可提现）", FormatCommissionCents(s.CommissionCents)))
	}
	_ = invalidateUserCache(s.InviterId)
}

// ReleaseMaturedCommissions moves commissions past their AvailableAt hold from
// PendingCommissionCents to CommissionBalanceCents (withdrawable). Designed to
// be called by a hourly background job. Processes in batches of
// commissionReleaseBatchSize to bound lock duration.
//
// Invariant: inviter.PendingCommissionCents must equal the sum of that inviter's
// status=pending Commission rows. Each release debits PendingCommissionCents and
// credits CommissionBalanceCents by the same per-inviter sum, and flips the rows
// to available, all in one transaction. If Pending would go negative (shouldn't
// happen given the invariant) it logs and skips that inviter.
func ReleaseMaturedCommissions() (released int, err error) {
	now := time.Now().Unix()
	for {
		var batch []Commission
		if err := DB.Where("status = ? AND available_at <= ?", CommissionStatusPending, now).
			Order("id ASC").Limit(commissionReleaseBatchSize).Find(&batch).Error; err != nil {
			return released, err
		}
		if len(batch) == 0 {
			return released, nil
		}

		// Group by inviter to update each user row once.
		byInviter := make(map[int]int64)
		ids := make([]int64, 0, len(batch))
		for _, c := range batch {
			byInviter[c.InviterId] += c.CommissionCents
			ids = append(ids, c.ID)
		}

		batchErr := DB.Transaction(func(tx *gorm.DB) error {
			for inviterId, sum := range byInviter {
				var inviter User
				if err := lockForUpdate(tx).First(&inviter, inviterId).Error; err != nil {
					return err
				}
				if inviter.PendingCommissionCents < sum {
					// Invariant violation — log and skip this inviter's rows this cycle.
					common.SysError(fmt.Sprintf("commission release underflow: inviter=%d pending=%d sum=%d",
						inviterId, inviter.PendingCommissionCents, sum))
					delete(byInviter, inviterId)
					continue
				}
				inviter.PendingCommissionCents -= sum
				inviter.CommissionBalanceCents += sum
				if err := tx.Model(&User{}).Where("id = ?", inviterId).Updates(map[string]any{
					"pending_commission_cents":   inviter.PendingCommissionCents,
					"commission_balance_cents":   inviter.CommissionBalanceCents,
				}).Error; err != nil {
					return err
				}
			}
			// Flip all released rows to available. Only rows for inviters we
			// successfully debited (still in byInviter) get flipped; underflow
			// rows stay pending for investigation.
			releaseIds := make([]int64, 0, len(ids))
			// Re-derive which IDs belong to successfully-processed inviters.
			idToInviter := make(map[int64]int, len(batch))
			for _, c := range batch {
				idToInviter[c.ID] = c.InviterId
			}
			for _, id := range ids {
				if _, ok := byInviter[idToInviter[id]]; ok {
					releaseIds = append(releaseIds, id)
				}
			}
			if len(releaseIds) > 0 {
				if err := tx.Model(&Commission{}).Where("id IN ?", releaseIds).
					Updates(map[string]any{"status": CommissionStatusAvailable}).Error; err != nil {
					return err
				}
				released += len(releaseIds)
			}
			return nil
		})
		if batchErr != nil {
			return released, batchErr
		}

		if len(batch) < commissionReleaseBatchSize {
			return released, nil
		}
	}
}

// WithdrawalRequest records a user's request to withdraw cash commission to a
// real-world payout. MVP uses admin manual review + manual payment; PayoutChannel
// and PayoutTxId are reserved for a future Stripe Connect integration.
//
// Lifecycle: pending (user requested, CommissionBalanceCents already debited) →
// paid (admin confirmed manual payout) | rejected (admin refused, balance refunded).
type WithdrawalRequest struct {
	ID            int64  `json:"id" gorm:"primaryKey"`
	UserId        int    `json:"user_id" gorm:"column:user_id;index"`
	AmountCents   int64  `json:"amount_cents" gorm:"column:amount_cents"`
	Status        string `json:"status" gorm:"column:status;type:varchar(16);index"`
	PayoutChannel string `json:"payout_channel" gorm:"column:payout_channel;type:varchar(32)"`
	PayoutTxId    string `json:"payout_tx_id" gorm:"column:payout_tx_id;type:varchar(255)"`
	AdminRemark   string `json:"admin_remark" gorm:"column:admin_remark;type:text"`
	ReviewedBy    int    `json:"reviewed_by" gorm:"column:reviewed_by"`
	RequestedAt   int64  `json:"requested_at" gorm:"column:requested_at"`
	ReviewedAt    int64  `json:"reviewed_at" gorm:"column:reviewed_at"`
	// Stripe Connect 打款字段（仅 payout_channel=stripe_connect 时填充）
	StripeAccountId              string `json:"stripe_account_id" gorm:"column:stripe_account_id;type:varchar(64);index"`
	StripeTransferId             string `json:"stripe_transfer_id" gorm:"column:stripe_transfer_id;type:varchar(64);index"`
	StripeTransferStatus         string `json:"stripe_transfer_status" gorm:"column:stripe_transfer_status;type:varchar(16)"`
	StripeTransferAmountReversed int64  `json:"stripe_transfer_amount_reversed" gorm:"column:stripe_transfer_amount_reversed;type:bigint;default:0"`
	StripePayoutId               string `json:"stripe_payout_id" gorm:"column:stripe_payout_id;type:varchar(64);index"`
	StripePayoutStatus           string `json:"stripe_payout_status" gorm:"column:stripe_payout_status;type:varchar(16)"`
	StripePayoutAttempt          int    `json:"stripe_payout_attempt" gorm:"column:stripe_payout_attempt;type:int;default:0"`
	LastReconcileAt              int64  `json:"last_reconcile_at" gorm:"column:last_reconcile_at;index"`
	LastReconcileError           string `json:"last_reconcile_error" gorm:"column:last_reconcile_error;type:text"`
	NextActionAt                 int64  `json:"next_action_at" gorm:"column:next_action_at;index"`
	CreatedAt                    int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
}

func (WithdrawalRequest) TableName() string { return "withdrawal_requests" }

const (
	WithdrawalStatusPending  = "pending"
	WithdrawalStatusPaid     = "paid"
	WithdrawalStatusRejected = "rejected"
)

// CreateWithdrawalRequest debits the user's withdrawable commission balance and
// creates a pending withdrawal request in one transaction. The balance debit
// happens up-front (not at admin-approve time) so the user cannot double-spend
// the same funds across concurrent requests. On reject, AdminProcessWithdrawal
// refunds the amount.
//
// Idempotency: each request gets a new row; the balance debit + row insert are
// atomic. A duplicate HTTP retry would create a second request (and fail on
// insufficient balance), so clients should guard against double-submit.
func CreateWithdrawalRequest(userId int, amountCents int64) (*WithdrawalRequest, error) {
	if userId <= 0 {
		return nil, errors.New("user id is empty")
	}
	if amountCents < common.MinWithdrawalCents {
		return nil, fmt.Errorf("提现金额不能少于 %s", FormatCommissionCents(common.MinWithdrawalCents))
	}

	var req *WithdrawalRequest
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if e := lockForUpdate(tx).First(&user, userId).Error; e != nil {
			return e
		}
		if !user.CommissionApproved {
			return errors.New("未开通现金佣金，无法提现")
		}
		if user.CommissionBalanceCents < amountCents {
			return fmt.Errorf("可提现余额不足，当前可用 %s", FormatCommissionCents(user.CommissionBalanceCents))
		}
		user.CommissionBalanceCents -= amountCents
		if e := tx.Model(&User{}).Where("id = ?", userId).
			Update("commission_balance_cents", user.CommissionBalanceCents).Error; e != nil {
			return e
		}
		now := time.Now().Unix()
		req = &WithdrawalRequest{
			UserId:      userId,
			AmountCents: amountCents,
			Status:      WithdrawalStatusPending,
			RequestedAt: now,
		}
		if e := tx.Create(req).Error; e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	RecordLog(userId, LogTypeSystem, fmt.Sprintf(
		"提现申请已提交 %s（等待管理员审核）", FormatCommissionCents(amountCents)))
	_ = invalidateUserCache(userId)
	return req, nil
}

// ListUserWithdrawalRequests returns a user's withdrawal history, newest first.
func ListUserWithdrawalRequests(userId int, page int, pageSize int) ([]WithdrawalRequest, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var reqs []WithdrawalRequest
	var total int64
	if err := DB.Model(&WithdrawalRequest{}).Where("user_id = ?", userId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := DB.Where("user_id = ?", userId).Order("id DESC").
		Offset(offset).Limit(pageSize).Find(&reqs).Error; err != nil {
		return nil, 0, err
	}
	return reqs, total, nil
}

// AdminListWithdrawalRequests returns all withdrawal requests for the admin
// review queue, optionally filtered by status. Newest first.
func AdminListWithdrawalRequests(status string, page int, pageSize int) ([]WithdrawalRequest, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := DB.Model(&WithdrawalRequest{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var reqs []WithdrawalRequest
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&reqs).Error; err != nil {
		return nil, 0, err
	}
	return reqs, total, nil
}

// AdminProcessWithdrawal transitions a pending withdrawal to paid or rejected.
//
// paid: admin confirms manual payout completed. PayoutChannel and PayoutTxId
//   record how/where the money was sent (reserved for Stripe Connect). Funds
//   were already debited at request time, so no balance change.
// rejected: admin refuses the request. Funds are refunded to the user's
//   CommissionBalanceCents in the same transaction.
//
// Idempotent: only pending→paid and pending→rejected are allowed; re-processing
// a terminal request is a no-op (returns the current state, no error).
func AdminProcessWithdrawal(requestId int64, action string, adminId int, payoutChannel string, payoutTxId string, adminRemark string) (*WithdrawalRequest, error) {
	if requestId <= 0 {
		return nil, errors.New("invalid request id")
	}
	if action != WithdrawalStatusPaid && action != WithdrawalStatusRejected {
		return nil, errors.New("invalid action")
	}

	var result *WithdrawalRequest
	err := DB.Transaction(func(tx *gorm.DB) error {
		var req WithdrawalRequest
		if e := lockForUpdate(tx).First(&req, requestId).Error; e != nil {
			return errors.New("提现申请不存在")
		}
		if req.Status != WithdrawalStatusPending {
			// Already processed — return current state, no error (idempotent).
			result = &req
			return nil
		}

		now := time.Now().Unix()
		req.Status = action
		req.PayoutChannel = payoutChannel
		req.PayoutTxId = payoutTxId
		req.AdminRemark = adminRemark
		req.ReviewedBy = adminId
		req.ReviewedAt = now
		if e := tx.Save(&req).Error; e != nil {
			return e
		}

		if action == WithdrawalStatusRejected {
			// Refund the debited amount back to withdrawable balance.
			var user User
			if e := lockForUpdate(tx).First(&user, req.UserId).Error; e != nil {
				return e
			}
			user.CommissionBalanceCents += req.AmountCents
			if e := tx.Model(&User{}).Where("id = ?", req.UserId).
				Update("commission_balance_cents", user.CommissionBalanceCents).Error; e != nil {
				return e
			}
		}

		result = &req
		return nil
	})
	if err != nil {
		return nil, err
	}

	if result != nil {
		switch result.Status {
		case WithdrawalStatusPaid:
			RecordLog(result.UserId, LogTypeSystem, fmt.Sprintf(
				"提现已完成 %s（打款渠道：%s）", FormatCommissionCents(result.AmountCents),
				payoutChannelDisplayName(result.PayoutChannel)))
		case WithdrawalStatusRejected:
			RecordLog(result.UserId, LogTypeSystem, fmt.Sprintf(
				"提现申请被拒绝 %s，金额已退回可用余额", FormatCommissionCents(result.AmountCents)))
		}
		_ = invalidateUserCache(result.UserId)
	}
	return result, nil
}

func payoutChannelDisplayName(channel string) string {
	if channel == "" {
		return "手动打款"
	}
	return channel
}

// GetCommissionSummary returns the user's commission balances for display.
type CommissionSummary struct {
	PendingCents    int64 `json:"pending_cents"`
	BalanceCents    int64 `json:"balance_cents"`
	TotalCents      int64 `json:"total_cents"`
	WithdrawnCents  int64 `json:"withdrawn_cents"`
	MinWithdrawalCents int64 `json:"min_withdrawal_cents"`
	FreezeDays      int   `json:"freeze_days"`
	Approved        bool  `json:"approved"`
}

// GetCommissionSummaryForUser loads a user's commission balances and lifetime
// withdrawn total. withdrawn = total ever marked paid. Also surfaces the
// admin-configured MinWithdrawalCents / CommissionFreezeDays so the wallet
// card can render hints without hitting the admin-only /api/option endpoint.
func GetCommissionSummaryForUser(userId int) (*CommissionSummary, error) {
	var user User
	if err := DB.Select("pending_commission_cents", "commission_balance_cents", "commission_total_cents", "commission_approved").
		First(&user, userId).Error; err != nil {
		return nil, err
	}
	var withdrawn int64
	if err := DB.Model(&WithdrawalRequest{}).
		Where("user_id = ? AND status = ?", userId, WithdrawalStatusPaid).
		Select("COALESCE(SUM(amount_cents), 0)").Scan(&withdrawn).Error; err != nil {
		return nil, err
	}
	return &CommissionSummary{
		PendingCents:       user.PendingCommissionCents,
		BalanceCents:       user.CommissionBalanceCents,
		TotalCents:         user.CommissionTotalCents,
		WithdrawnCents:     withdrawn,
		MinWithdrawalCents: common.MinWithdrawalCents,
		FreezeDays:         common.CommissionFreezeDays,
		Approved:           user.CommissionApproved,
	}, nil
}
