package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// 提现状态机（stripe_connect 渠道）。手工打款渠道仍用 pending/paid/rejected。
const (
	WithdrawalStatusTransferCreating = "transfer_creating"
	WithdrawalStatusAwaitingFunds    = "awaiting_funds"
	WithdrawalStatusPayoutCreating   = "payout_creating"
	WithdrawalStatusProcessing       = "processing"
	WithdrawalStatusActionRequired   = "action_required"
	WithdrawalStatusFailed           = "failed"
	WithdrawalStatusCanceled         = "canceled"
)

// ErrWithdrawalStatusConflict 表示 CAS 翻转失败（状态已被并发改变）。
var ErrWithdrawalStatusConflict = errors.New("withdrawal status conflict")

// TransitionWithdrawalStatus 用 CAS（UPDATE...WHERE id=? AND status=from）翻转状态。
// mutator 在同一事务内修改 req 的字段（Stripe IDs 等）。from==to 时 mutator 仍执行
// （用于幂等重放存 ID）。返回更新后的记录；若状态不匹配 from 返回 ErrWithdrawalStatusConflict。
//
// 关键：此函数只做 DB 操作，绝不调用 Stripe 网络 API（见 spec §四铁律）。
func TransitionWithdrawalStatus(requestId int64, from string, to string, mutator func(tx *gorm.DB, req *WithdrawalRequest) error) (*WithdrawalRequest, error) {
	if requestId <= 0 {
		return nil, errors.New("invalid request id")
	}
	if from == "" || to == "" {
		return nil, errors.New("empty status")
	}
	var result WithdrawalRequest
	err := DB.Transaction(func(tx *gorm.DB) error {
		// lockForUpdate 在 MySQL/PostgreSQL 上发 SELECT ... FOR UPDATE，串行化同行的并发翻转；
		// SQLite 忽略（无 FOR UPDATE 语法），依赖单写者锁。配合下方 UPDATE 的 status=from
		// 谓词构成真正的 CAS（spec §四铁律）。
		if e := lockForUpdate(tx).Where("id = ? AND status = ?", requestId, from).First(&result).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return ErrWithdrawalStatusConflict
			}
			return e
		}
		if mutator != nil {
			if e := mutator(tx, &result); e != nil {
				return e
			}
		}
		result.Status = to
		// UPDATE 同时带 id 与 status=from 谓词，确保即便行锁未生效（如 SQLite）也不会覆盖
		// 已被并发改掉的 status。RowsAffected=0 时返回冲突错误。
		res := tx.Model(&WithdrawalRequest{}).Where("id = ? AND status = ?", requestId, from).
			Updates(map[string]any{
				"status":                          to,
				"stripe_account_id":               result.StripeAccountId,
				"stripe_transfer_id":              result.StripeTransferId,
				"stripe_transfer_status":          result.StripeTransferStatus,
				"stripe_transfer_amount_reversed": result.StripeTransferAmountReversed,
				"stripe_payout_id":                result.StripePayoutId,
				"stripe_payout_status":            result.StripePayoutStatus,
				"stripe_payout_attempt":           result.StripePayoutAttempt,
				"last_reconcile_at":               result.LastReconcileAt,
				"last_reconcile_error":            result.LastReconcileError,
				"next_action_at":                  result.NextActionAt,
				"reviewed_at":                     result.ReviewedAt,
				"reviewed_by":                     result.ReviewedBy,
				"payout_channel":                  result.PayoutChannel,
				"payout_tx_id":                    result.PayoutTxId,
				"admin_remark":                    result.AdminRemark,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// 行存在但 status 已被并发改掉——回退为冲突，调用方重读后重试或放弃。
			return ErrWithdrawalStatusConflict
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RefundWithdrawalBalance 把提现金额退回用户 commission_balance_cents。
// 仅在 Transfer 未创建或 Reversal 成功后调用（spec §十三不变量 1/6）。
// 在调用方提供的事务内执行。
func RefundWithdrawalBalance(tx *gorm.DB, userId int, amountCents int64) error {
	if amountCents <= 0 {
		return nil
	}
	var user User
	if e := lockForUpdate(tx).First(&user, userId).Error; e != nil {
		return e
	}
	user.CommissionBalanceCents += amountCents
	return tx.Model(&User{}).Where("id = ?", userId).
		Update("commission_balance_cents", user.CommissionBalanceCents).Error
}

// MarkWithdrawalFailed 退款并翻到 failed，记录原因。用于 Transfer 创建失败 / Reversal 成功后。
func MarkWithdrawalFailed(requestId int64, from string, reason string) (*WithdrawalRequest, error) {
	return TransitionWithdrawalStatus(requestId, from, WithdrawalStatusFailed, func(tx *gorm.DB, req *WithdrawalRequest) error {
		req.AdminRemark = truncateReason(reason, 2000)
		req.ReviewedAt = time.Now().Unix()
		return RefundWithdrawalBalance(tx, req.UserId, req.AmountCents)
	})
}

func truncateReason(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
