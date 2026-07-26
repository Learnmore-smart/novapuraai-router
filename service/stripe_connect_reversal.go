package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

// ErrReversalAlreadyTerminal is returned when the withdrawal is already in a
// terminal state (paid/failed/canceled) and cannot be reversed again.
var ErrReversalAlreadyTerminal = errors.New("withdrawal is already in a terminal state")

// ErrReversalInsufficientFunds is returned when the Stripe reversal fails
// because the connected account's available balance cannot cover the reversal
// (e.g. a Payout already drained the balance). The withdrawal stays in its
// current state and the commission balance is NOT refunded; reconciliation
// retries later.
var ErrReversalInsufficientFunds = errors.New("reversal failed: connected account has insufficient funds")

// ReverseStripeConnectTransfer reverses the Transfer for a withdrawal and
// refunds the user's commission balance. Only call this when the withdrawal
// has a stripe_transfer_id and the Payout has failed (or admin manually
// requests reversal).
//
// Flow (spec §十三):
//  1. Load withdrawal; must have stripe_transfer_id.
//  2. OUTSIDE any tx: call client.ReverseTransfer with ReversalIdempotencyKey.
//  3a. On success: CAS current_state → failed, storing
//      stripe_transfer_amount_reversed += reversedAmount, then call
//      model.RefundWithdrawalBalance within the same tx.
//  3b. On insufficient_funds error: stay in current state, record
//      last_reconcile_error, return error. Do NOT refund.
//  3c. On other error: stay in current state, record last_reconcile_error,
//      return error.
//
// amountCents: how much to reverse. If 0 or > remaining Transfer amount, reverse
// the full remaining amount (call with req.AmountCents - req.StripeTransferAmountReversed).
func ReverseStripeConnectTransfer(ctx context.Context, client StripeConnectClient, requestId int64, reason string) (*model.WithdrawalRequest, error) {
	if client == nil {
		return nil, errors.New("stripe connect client unavailable")
	}
	if requestId <= 0 {
		return nil, errors.New("invalid request id")
	}

	var req model.WithdrawalRequest
	if err := model.DB.First(&req, requestId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("withdrawal request %d not found", requestId)
		}
		return nil, fmt.Errorf("load withdrawal: %w", err)
	}
	if req.StripeTransferId == "" {
		return &req, errors.New("cannot reverse: no stripe_transfer_id")
	}
	if isWithdrawalTerminal(req.Status) {
		return &req, ErrReversalAlreadyTerminal
	}
	amountToReverse := req.AmountCents - req.StripeTransferAmountReversed
	if amountToReverse <= 0 {
		// Already fully reversed (e.g. a prior partial reversal completed, or an
		// idempotent retry). Nothing to do — never refund twice.
		return &req, nil
	}

	// Stripe reversal call — OUTSIDE any DB transaction/row lock.
	rev, err := client.ReverseTransfer(ctx, req.StripeTransferId, amountToReverse, ReversalIdempotencyKey(req.ID))
	if err != nil {
		errReason := fmt.Sprintf("reversal failed: %v", err)
		// Record the error via same-state CAS (best-effort; ignore conflict).
		_, _ = model.TransitionWithdrawalStatus(requestId, req.Status, req.Status, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
			r.LastReconcileError = truncateReasonLocal(errReason, 2000)
			r.LastReconcileAt = time.Now().Unix()
			return nil
		})
		if isInsufficientFundsErr(err) {
			return &req, ErrReversalInsufficientFunds
		}
		return &req, fmt.Errorf("stripe transferreversal.New: %w", err)
	}

	// Success — CAS current_state → failed, store reversed amount and refund
	// the user's commission balance within the same transaction. Refunding only
	// rev.AmountReversed (not AmountCents) keeps us safe if a partial reversal
	// occurred earlier.
	result, err := model.TransitionWithdrawalStatus(requestId, req.Status, model.WithdrawalStatusFailed, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripeTransferAmountReversed += rev.AmountReversed
		r.AdminRemark = truncateReasonLocal(reason, 2000)
		r.ReviewedAt = time.Now().Unix()
		return model.RefundWithdrawalBalance(tx, r.UserId, rev.AmountReversed)
	})
	if err != nil {
		// CAS conflict — another worker moved it. Re-read.
		var cur model.WithdrawalRequest
		model.DB.First(&cur, requestId)
		// CRITICAL: the reversal succeeded on Stripe but our DB update failed.
		// Log loudly — reconciliation will detect (amount_reversed < amount)
		// and retry the CAS, but the funds are already on the way back.
		common.SysError(fmt.Sprintf("stripe_connect withdrawal %d: reversal succeeded (rev_id=%s amount=%d) but CAS to failed conflicted: %v", req.ID, rev.ID, rev.AmountReversed, err))
		return &cur, fmt.Errorf("cas conflict after successful reversal: %w", err)
	}
	// Log to user.
	model.RecordLog(result.UserId, model.LogTypeSystem, fmt.Sprintf(
		"提现失败 %s，金额已退回可用余额（Stripe Transfer 已反转）",
		model.FormatCommissionCents(rev.AmountReversed)))
	_ = model.InvalidateUserCache(result.UserId)
	return result, nil
}

// isWithdrawalTerminal reports whether the status is a terminal state from
// which no further reversal is possible (paid/failed/canceled).
func isWithdrawalTerminal(status string) bool {
	switch status {
	case model.WithdrawalStatusPaid, model.WithdrawalStatusFailed, model.WithdrawalStatusCanceled:
		return true
	}
	return false
}
