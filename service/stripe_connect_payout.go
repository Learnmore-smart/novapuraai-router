package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	stripe "github.com/stripe/stripe-go/v85"
	"gorm.io/gorm"
)

// ErrWithdrawalAlreadyProcessing is returned when the withdrawal is no longer
// in pending state (already processed by a concurrent approver, or replayed).
var ErrWithdrawalAlreadyProcessing = errors.New("withdrawal is no longer pending")

// ApproveStripeConnectWithdrawal is called by the admin-approve controller when
// payout_channel=stripe_connect. It runs the Transfer flow synchronously and
// returns the resulting WithdrawalRequest (in transfer_creating, awaiting_funds,
// payout_creating, or failed state).
//
// Flow (per spec §四铁律):
//  1. Validate: Stripe Connect enabled, user has enabled StripeConnectAccount.
//  2. CAS pending → transfer_creating (store stripe_account_id, reviewed_by/at, payout_channel).
//  3. OUTSIDE any tx: call client.CreateTransfer with idempotency key.
//  4a. On success: CAS transfer_creating → payout_creating or awaiting_funds,
//      storing stripe_transfer_id="tr_...", stripe_transfer_status="created".
//  4b. On failure: model.MarkWithdrawalFailed(req.ID, transfer_creating, reason).
//
// If CAS fails at step 2 (status no longer pending), return the current
// WithdrawalRequest and a sentinel error ErrWithdrawalAlreadyProcessing.
func ApproveStripeConnectWithdrawal(ctx context.Context, client StripeConnectClient, requestId int64, adminId int) (*model.WithdrawalRequest, error) {
	if !setting.StripeConnectEnabled {
		return nil, errors.New("stripe connect is not enabled")
	}
	if client == nil {
		return nil, errors.New("stripe connect client unavailable")
	}

	var req model.WithdrawalRequest
	if err := model.DB.First(&req, requestId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("withdrawal request %d not found", requestId)
		}
		return nil, fmt.Errorf("load withdrawal: %w", err)
	}
	if req.Status != model.WithdrawalStatusPending {
		return &req, ErrWithdrawalAlreadyProcessing
	}

	acc, err := model.GetStripeConnectAccount(req.UserId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("load stripe connect account: %w", err)
		}
		reason := fmt.Sprintf("user %d has no stripe connect account", req.UserId)
		failed, ferr := model.MarkWithdrawalFailed(req.ID, model.WithdrawalStatusPending, reason)
		if ferr != nil {
			return nil, fmt.Errorf("mark-failed (%s) also failed: %w", reason, ferr)
		}
		return failed, errors.New(reason)
	}
	if acc.OnboardingState != model.ConnectOnboardingEnabled {
		reason := fmt.Sprintf("user %d stripe connect account not enabled (state=%s)", req.UserId, acc.OnboardingState)
		failed, ferr := model.MarkWithdrawalFailed(req.ID, model.WithdrawalStatusPending, reason)
		if ferr != nil {
			return nil, fmt.Errorf("mark-failed (%s) also failed: %w", reason, ferr)
		}
		return failed, errors.New(reason)
	}

	// CAS pending → transfer_creating (short tx; no Stripe call inside).
	transitioned, err := model.TransitionWithdrawalStatus(requestId, model.WithdrawalStatusPending, model.WithdrawalStatusTransferCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripeAccountId = acc.StripeAccountId
		r.PayoutChannel = "stripe_connect"
		r.ReviewedBy = adminId
		r.ReviewedAt = time.Now().Unix()
		return nil
	})
	if errors.Is(err, model.ErrWithdrawalStatusConflict) {
		// Re-read and return ErrWithdrawalAlreadyProcessing.
		var cur model.WithdrawalRequest
		model.DB.First(&cur, requestId)
		return &cur, ErrWithdrawalAlreadyProcessing
	}
	if err != nil {
		return nil, fmt.Errorf("cas pending → transfer_creating: %w", err)
	}

	// Stripe Transfer call — OUTSIDE any DB transaction/row lock.
	tr, err := client.CreateTransfer(ctx, TransferParams{
		AmountCents:  transitioned.AmountCents,
		Currency:     "usd",
		Destination:  transitioned.StripeAccountId,
		WithdrawalID: transitioned.ID,
		UserID:       transitioned.UserId,
	}, TransferIdempotencyKey(transitioned.ID))
	if err != nil {
		// Transfer was not created — safe to refund the debited balance.
		reason := fmt.Sprintf("stripe transfer.New: %v", err)
		failed, ferr := model.MarkWithdrawalFailed(transitioned.ID, model.WithdrawalStatusTransferCreating, reason)
		if ferr != nil {
			return nil, fmt.Errorf("transfer failed (%v) and mark-failed also failed: %w", err, ferr)
		}
		return failed, fmt.Errorf("transfer creation failed: %w", err)
	}

	// Balance check on the connected account. Even if this fails, the Transfer
	// was already created — transition to awaiting_funds so reconciliation can
	// recover. Log the balance error for investigation.
	balCents, balErr := client.GetBalanceAvailableUSD(ctx, transitioned.StripeAccountId)
	if balErr != nil {
		common.SysError(fmt.Sprintf("stripe_connect withdrawal %d: balance check failed after transfer %s: %v", transitioned.ID, tr.ID, balErr))
		balCents = 0 // force awaiting_funds
	}
	targetStatus := model.WithdrawalStatusPayoutCreating
	if balCents < transitioned.AmountCents {
		targetStatus = model.WithdrawalStatusAwaitingFunds
	}

	result, err := model.TransitionWithdrawalStatus(requestId, model.WithdrawalStatusTransferCreating, targetStatus, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripeTransferId = tr.ID
		r.StripeTransferStatus = "created"
		r.LastReconcileAt = time.Now().Unix()
		return nil
	})
	if err != nil {
		// CAS conflict — another worker already moved it. Re-read and surface
		// the conflict. The Transfer exists on Stripe's side; reconciliation
		// (Task 11) must reconcile the orphaned transfer by metadata lookup.
		var cur model.WithdrawalRequest
		model.DB.First(&cur, requestId)
		return &cur, fmt.Errorf("cas conflict storing transfer_id: %w", err)
	}

	// Funds are available on the connected account — try to create the Payout
	// immediately so the admin sees the final state (processing). If this
	// fails, reconciliation (Task 11) will retry. The Transfer already
	// succeeded, so a payout-creation failure is NOT a hard failure of the
	// approve flow; we return the pre-payout request (in payout_creating).
	if result.Status == model.WithdrawalStatusPayoutCreating {
		finalReq, pErr := ProcessStripeConnectPayout(ctx, client, result.ID)
		if pErr != nil {
			common.SysError(fmt.Sprintf("stripe_connect withdrawal %d: immediate payout creation failed: %v", result.ID, pErr))
			return result, nil
		}
		return finalReq, nil
	}
	return result, nil
}

// truncateReasonLocal truncates a reason string to max bytes. Mirrors the
// unexported model.truncateReason so service-layer code can bound
// last_reconcile_error without reaching across packages.
func truncateReasonLocal(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// isInsufficientFundsErr reports whether err is a Stripe insufficient_funds
// error. Stripe returns this from payout.New when the connected account's
// available balance can't cover the payout; the Transfer already exists, so we
// must NOT refund commission balance — instead we wait for balance.available
// (reconciliation retries in Task 11).
func isInsufficientFundsErr(err error) bool {
	if err == nil {
		return false
	}
	var se *stripe.Error
	if errors.As(err, &se) {
		return se.Code == stripe.ErrorCodeInsufficientFunds
	}
	// Fallback: string match for wrapped errors from the client impl (which
	// uses fmt.Errorf("stripe payout.New: %w", err)) and any non-SDK path that
	// still surfaces the Stripe code in the message.
	return strings.Contains(err.Error(), "insufficient_funds")
}

// ProcessStripeConnectPayout takes a withdrawal in payout_creating state and
// creates a Stripe Payout on the connected account. Returns the updated
// WithdrawalRequest (in processing, awaiting_funds, or still payout_creating).
//
// Flow (per spec §九):
//  1. Load withdrawal; must be in payout_creating with stripe_transfer_id set.
//  2. CAS payout_creating → payout_creating (same-state) to increment
//     stripe_payout_attempt and stamp last_reconcile_at. This makes the
//     attempt number durable before the Stripe call, so a crash + retry
//     produces the same idempotency key.
//  3. OUTSIDE any tx: client.CreatePayout with PayoutIdempotencyKey(req.ID, req.StripePayoutAttempt).
//  4a. On success: CAS payout_creating → processing, store stripe_payout_id,
//      stripe_payout_status (pending/in_transit).
//  4b. On insufficient_funds error: CAS payout_creating → awaiting_funds,
//      record last_reconcile_error.
//  4c. On other error: stay in payout_creating, update last_reconcile_error,
//      return the error (caller decides whether to mark action_required).
func ProcessStripeConnectPayout(ctx context.Context, client StripeConnectClient, requestId int64) (*model.WithdrawalRequest, error) {
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
	if req.Status != model.WithdrawalStatusPayoutCreating {
		// Caller invoked us on a withdrawal that isn't waiting for a payout
		// (already processing, awaiting_funds, or terminal). No-op.
		common.SysError(fmt.Sprintf("stripe_connect withdrawal %d: skipping payout creation, status=%s", requestId, req.Status))
		return &req, nil
	}
	if req.StripeTransferId == "" {
		return &req, errors.New("cannot create payout: no transfer_id")
	}
	if req.StripeAccountId == "" {
		return &req, errors.New("cannot create payout: no stripe_account_id")
	}

	// CAS same-state to make the attempt number durable BEFORE the Stripe call.
	// A crash after this point + retry reuses the same idempotency key, so Stripe
	// dedupes the second attempt instead of creating a second Payout.
	reqPtr, err := model.TransitionWithdrawalStatus(requestId, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusPayoutCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripePayoutAttempt = r.StripePayoutAttempt + 1
		r.LastReconcileAt = time.Now().Unix()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cas failed incrementing payout attempt: %w", err)
	}
	req = *reqPtr

	// Stripe Payout call — OUTSIDE any DB transaction/row lock.
	po, err := client.CreatePayout(ctx, PayoutParams{
		AmountCents:  req.AmountCents,
		Currency:     "usd",
		WithdrawalID: req.ID,
	}, PayoutIdempotencyKey(req.ID, req.StripePayoutAttempt), req.StripeAccountId)
	if err != nil {
		reason := fmt.Sprintf("stripe payout.New (attempt %d): %v", req.StripePayoutAttempt, err)
		if isInsufficientFundsErr(err) {
			// Transfer exists on the connected account but its available
			// balance can't cover the payout yet. Wait for balance.available;
			// reconciliation (Task 11) retries. Do NOT refund — the Transfer
			// already moved the funds to the connected account.
			afReq, cerr := model.TransitionWithdrawalStatus(requestId, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusAwaitingFunds, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
				r.LastReconcileError = truncateReasonLocal(reason, 2000)
				r.LastReconcileAt = time.Now().Unix()
				return nil
			})
			if cerr != nil {
				return nil, fmt.Errorf("cas failed transitioning to awaiting_funds: %w", cerr)
			}
			return afReq, nil // not an error for the caller — withdrawal is now awaiting_funds
		}
		// Other Payout failure (card declined, bank rejected, etc.). Stay in
		// payout_creating, record the error, and surface it so the caller can
		// decide on action_required (Task 11). Stripe has disabled the external
		// account; reconciliation + account.external_account.updated recovers.
		errReq, cerr := model.TransitionWithdrawalStatus(requestId, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusPayoutCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
			r.LastReconcileError = truncateReasonLocal(reason, 2000)
			r.LastReconcileAt = time.Now().Unix()
			return nil
		})
		if cerr != nil {
			return nil, fmt.Errorf("payout failed (%v) and cas-error-update also failed: %w", err, cerr)
		}
		return errReq, err // return the original Stripe error
	}

	// Success — store payout_id and move to processing. The Payout is now
	// pending/in_transit on Stripe; webhooks (payout.paid/payout.failed) drive
	// the final state in Task 10.
	procReq, err := model.TransitionWithdrawalStatus(requestId, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusProcessing, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripePayoutId = po.ID
		r.StripePayoutStatus = po.Status // "pending" or "in_transit"
		r.LastReconcileAt = time.Now().Unix()
		r.LastReconcileError = ""
		return nil
	})
	if err != nil {
		// CAS conflict — another worker already moved it. Re-read and surface.
		// The Payout exists on Stripe; reconciliation (Task 11) reconciles by
		// metadata lookup.
		var cur model.WithdrawalRequest
		model.DB.First(&cur, requestId)
		return &cur, fmt.Errorf("cas conflict storing payout_id: %w", err)
	}
	return procReq, nil
}
