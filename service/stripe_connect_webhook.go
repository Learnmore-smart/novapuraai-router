package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	stripe "github.com/stripe/stripe-go/v85"
	"gorm.io/gorm"
)

// HandleConnectEvent dispatches a Stripe Connect event to the appropriate handler.
// Returns nil for unknown event types (Stripe may add new types; we ignore them).
// All handlers are idempotent: replaying an event is safe.
//
// Handlers that need a Stripe client (balance.available,
// account.external_account.updated) call ProcessStripeConnectPayout, which
// requires a non-nil client. Handlers that only touch local state
// (account.updated, payout.*) tolerate a nil client so they still work when
// Stripe Connect was disabled mid-flight.
func HandleConnectEvent(ctx context.Context, client StripeConnectClient, evt *stripe.Event) error {
	if evt == nil {
		return nil
	}
	switch evt.Type {
	case "account.updated":
		return handleAccountUpdated(evt)
	case "account.external_account.updated":
		return handleExternalAccountUpdated(ctx, client, evt)
	case "balance.available":
		return handleBalanceAvailable(ctx, client, evt)
	case "payout.created":
		return handlePayoutCreated(evt)
	case "payout.updated":
		return handlePayoutUpdated(evt)
	case "payout.paid":
		return handlePayoutPaid(evt)
	case "payout.failed":
		return handlePayoutFailed(evt)
	}
	return nil
}

// handleAccountUpdated refreshes the local StripeConnectAccount snapshot from
// the account.updated payload. Idempotent: replaying re-applies the same
// snapshot. Unknown accounts (e.g. from another integration) are ignored.
func handleAccountUpdated(evt *stripe.Event) error {
	var acc stripe.Account
	if err := common.Unmarshal(evt.Data.Raw, &acc); err != nil {
		return fmt.Errorf("parse account.updated payload: %w", err)
	}
	local, err := model.GetStripeConnectAccountByStripeId(acc.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // we don't track this account
		}
		return fmt.Errorf("lookup connect account by stripe id: %w", err)
	}

	interval := ""
	if acc.Settings != nil && acc.Settings.Payouts != nil && acc.Settings.Payouts.Schedule != nil {
		interval = string(acc.Settings.Payouts.Schedule.Interval)
	}
	currentlyDueJSON := stringSliceJSON(acc.Requirements, func(r *stripe.AccountRequirements) []string {
		return r.CurrentlyDue
	})
	eventuallyDueJSON := stringSliceJSON(acc.Requirements, func(r *stripe.AccountRequirements) []string {
		return r.EventuallyDue
	})

	// Use the local record's UserId: Stripe's metadata may be missing or stale.
	if err := model.UpdateStripeConnectAccountFromStripe(
		local.UserId, acc.ID, acc.Email, acc.Country,
		acc.PayoutsEnabled, acc.DetailsSubmitted, interval,
		currentlyDueJSON, eventuallyDueJSON,
	); err != nil {
		return fmt.Errorf("update connect account from stripe: %w", err)
	}
	return nil
}

// stringSliceJSON serializes a requirements string slice to JSON ("[]" if nil
// or empty). kept as a small helper because it is used twice with different
// accessors on the same struct.
func stringSliceJSON(reqs *stripe.AccountRequirements, pick func(*stripe.AccountRequirements) []string) string {
	if reqs == nil {
		return "[]"
	}
	v := pick(reqs)
	if len(v) == 0 {
		return "[]"
	}
	b, err := common.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// handleExternalAccountUpdated retries Payout creation for withdrawals stuck in
// action_required for the connected account that owns the updated bank account.
// On failure the withdrawal is left for reconciliation (Task 11).
func handleExternalAccountUpdated(ctx context.Context, client StripeConnectClient, evt *stripe.Event) error {
	var ba stripe.BankAccount
	if err := common.Unmarshal(evt.Data.Raw, &ba); err != nil {
		return fmt.Errorf("parse account.external_account.updated payload: %w", err)
	}
	stripeAccountID := ""
	if ba.Account != nil {
		stripeAccountID = ba.Account.ID
	}
	if stripeAccountID == "" {
		return nil
	}
	acc, err := model.GetStripeConnectAccountByStripeId(stripeAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("lookup connect account by stripe id: %w", err)
	}
	return retryPayoutsForStatus(ctx, client, acc.UserId, model.WithdrawalStatusActionRequired)
}

// handleBalanceAvailable retries Payout creation for withdrawals in
// awaiting_funds on the connected account whose balance just became available.
// Platform balance events (evt.Account == "") are ignored.
func handleBalanceAvailable(ctx context.Context, client StripeConnectClient, evt *stripe.Event) error {
	if evt.Account == "" {
		return nil // platform balance, not a connected account's
	}
	acc, err := model.GetStripeConnectAccountByStripeId(evt.Account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("lookup connect account by stripe id: %w", err)
	}
	return retryPayoutsForStatus(ctx, client, acc.UserId, model.WithdrawalStatusAwaitingFunds)
}

// retryPayoutsForStatus lists withdrawals for a user in `from` status,
// CAS-transitions each to payout_creating (clearing last_reconcile_error), then
// calls ProcessStripeConnectPayout. CAS conflicts and per-withdrawal payout
// failures are logged and skipped so one bad withdrawal can't block the others.
func retryPayoutsForStatus(ctx context.Context, client StripeConnectClient, userId int, from string) error {
	var reqs []model.WithdrawalRequest
	if err := model.DB.Where("user_id = ? AND status = ?", userId, from).Find(&reqs).Error; err != nil {
		return fmt.Errorf("list %s withdrawals: %w", from, err)
	}
	for i := range reqs {
		reqID := reqs[i].ID
		if _, cerr := model.TransitionWithdrawalStatus(reqID, from, model.WithdrawalStatusPayoutCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
			r.LastReconcileError = ""
			r.LastReconcileAt = time.Now().Unix()
			return nil
		}); cerr != nil {
			if errors.Is(cerr, model.ErrWithdrawalStatusConflict) {
				continue // someone else moved it
			}
			common.SysError(fmt.Sprintf("stripe_connect webhook: CAS %s → payout_creating failed for %d: %v", from, reqID, cerr))
			continue
		}
		if _, pErr := ProcessStripeConnectPayout(ctx, client, reqID); pErr != nil {
			common.SysError(fmt.Sprintf("stripe_connect webhook: retry payout %d failed: %v", reqID, pErr))
			// Withdrawal is now in payout_creating / awaiting_funds / processing
			// depending on the failure mode; reconciliation (Task 11) recovers.
		}
	}
	return nil
}

// handlePayoutCreated stores the Stripe Payout ID on the matching withdrawal and
// CAS-transitions payout_creating → processing. If the withdrawal is already
// processing (concurrent webhook / our CAS landed first), the payout_id is
// backfilled via a same-state CAS. Idempotent.
func handlePayoutCreated(evt *stripe.Event) error {
	var po stripe.Payout
	if err := common.Unmarshal(evt.Data.Raw, &po); err != nil {
		return fmt.Errorf("parse payout.created payload: %w", err)
	}
	req, err := findWithdrawalByPayout(&po)
	if err != nil {
		return fmt.Errorf("lookup withdrawal for payout %s: %w", po.ID, err)
	}
	if req == nil {
		return nil // payout not from our system
	}

	// Already have this payout_id recorded: just refresh status if changed.
	if req.StripePayoutId == po.ID {
		if req.StripePayoutStatus != string(po.Status) {
			if uErr := model.DB.Model(&model.WithdrawalRequest{}).Where("id = ?", req.ID).
				Update("stripe_payout_status", string(po.Status)).Error; uErr != nil {
				return fmt.Errorf("update stripe_payout_status: %w", uErr)
			}
		}
		return nil
	}

	// Don't have the payout_id yet (or it's empty). CAS to processing, storing
	// both the id and status. Try payout_creating → processing first; if the
	// withdrawal is already processing (concurrent worker), do a same-state CAS
	// to backfill the ids.
	mutator := func(r *model.WithdrawalRequest) {
		r.StripePayoutId = po.ID
		r.StripePayoutStatus = string(po.Status)
		r.LastReconcileAt = time.Now().Unix()
		r.LastReconcileError = ""
	}
	if _, cerr := model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusProcessing, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		mutator(r)
		return nil
	}); cerr == nil {
		return nil
	} else if !errors.Is(cerr, model.ErrWithdrawalStatusConflict) {
		return fmt.Errorf("cas payout_creating → processing: %w", cerr)
	}

	if _, cerr := model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusProcessing, model.WithdrawalStatusProcessing, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		mutator(r)
		return nil
	}); cerr != nil {
		// Not in payout_creating nor processing — leave it; reconciliation
		// (Task 11) reconciles by metadata lookup. Don't fail the webhook.
		common.SysError(fmt.Sprintf("stripe_connect payout.created: could not store payout_id %s on withdrawal %d (status=%s): %v", po.ID, req.ID, req.Status, cerr))
	}
	return nil
}

// handlePayoutUpdated refreshes stripe_payout_status on the matching withdrawal.
// No state transition: payout.paid / payout.failed drive terminal transitions.
func handlePayoutUpdated(evt *stripe.Event) error {
	var po stripe.Payout
	if err := common.Unmarshal(evt.Data.Raw, &po); err != nil {
		return fmt.Errorf("parse payout.updated payload: %w", err)
	}
	req, err := findWithdrawalByPayout(&po)
	if err != nil {
		return fmt.Errorf("lookup withdrawal for payout %s: %w", po.ID, err)
	}
	if req == nil {
		return nil
	}
	if err := model.DB.Model(&model.WithdrawalRequest{}).Where("id = ?", req.ID).
		Update("stripe_payout_status", string(po.Status)).Error; err != nil {
		return fmt.Errorf("update stripe_payout_status: %w", err)
	}
	return nil
}

// handlePayoutPaid CAS-transitions the matching withdrawal to paid (terminal)
// and records a user log. Tries processing → paid first, then payout_creating →
// paid (in case the webhook beat our CAS). Idempotent: a withdrawal already
// paid is a no-op.
func handlePayoutPaid(evt *stripe.Event) error {
	var po stripe.Payout
	if err := common.Unmarshal(evt.Data.Raw, &po); err != nil {
		return fmt.Errorf("parse payout.paid payload: %w", err)
	}
	req, err := findWithdrawalByPayout(&po)
	if err != nil {
		return fmt.Errorf("lookup withdrawal for payout %s: %w", po.ID, err)
	}
	if req == nil {
		return nil
	}
	if req.Status == model.WithdrawalStatusPaid {
		return nil
	}

	mutator := func(r *model.WithdrawalRequest) {
		r.StripePayoutId = po.ID
		r.StripePayoutStatus = "paid"
		r.LastReconcileError = ""
		r.LastReconcileAt = time.Now().Unix()
	}
	transitioned, cerr := model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusProcessing, model.WithdrawalStatusPaid, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		mutator(r)
		return nil
	})
	if cerr != nil && errors.Is(cerr, model.ErrWithdrawalStatusConflict) {
		// Webhook beat our payout_creating → processing CAS; retry from there.
		transitioned, cerr = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusPaid, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
			mutator(r)
			return nil
		})
	}
	if cerr != nil {
		if errors.Is(cerr, model.ErrWithdrawalStatusConflict) {
			// Re-read; if already paid it's an idempotent replay.
			var cur model.WithdrawalRequest
			if e := model.DB.First(&cur, req.ID).Error; e == nil && cur.Status == model.WithdrawalStatusPaid {
				return nil
			}
			common.SysError(fmt.Sprintf("stripe_connect payout.paid: could not transition withdrawal %d to paid: %v", req.ID, cerr))
			return nil
		}
		return fmt.Errorf("cas → paid: %w", cerr)
	}

	model.RecordLog(transitioned.UserId, model.LogTypeSystem, fmt.Sprintf(
		"提现已完成 %s（Stripe Connect 打款成功）", model.FormatCommissionCents(transitioned.AmountCents)))
	_ = model.InvalidateUserCache(transitioned.UserId)
	return nil
}

// handlePayoutFailed CAS-transitions the matching withdrawal to action_required
// and records last_reconcile_error. The Transfer still exists on the connected
// account, so the balance is NOT refunded here — reversal (Task 11) must
// recover the funds before any refund. Idempotent: terminal withdrawals are
// left untouched.
func handlePayoutFailed(evt *stripe.Event) error {
	var po stripe.Payout
	if err := common.Unmarshal(evt.Data.Raw, &po); err != nil {
		return fmt.Errorf("parse payout.failed payload: %w", err)
	}
	req, err := findWithdrawalByPayout(&po)
	if err != nil {
		return fmt.Errorf("lookup withdrawal for payout %s: %w", po.ID, err)
	}
	if req == nil {
		return nil
	}
	if isTerminalWithdrawalStatus(req.Status) {
		return nil
	}

	failureReason := "unknown"
	if po.FailureMessage != "" {
		failureReason = po.FailureMessage
	} else if po.FailureCode != "" {
		failureReason = string(po.FailureCode)
	}
	reason := truncateReasonLocal("payout.failed: "+failureReason, 2000)

	mutator := func(r *model.WithdrawalRequest) {
		r.LastReconcileError = reason
		r.LastReconcileAt = time.Now().Unix()
	}
	_, cerr := model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusProcessing, model.WithdrawalStatusActionRequired, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		mutator(r)
		return nil
	})
	if cerr != nil && errors.Is(cerr, model.ErrWithdrawalStatusConflict) {
		_, cerr = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusActionRequired, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
			mutator(r)
			return nil
		})
	}
	if cerr != nil {
		if errors.Is(cerr, model.ErrWithdrawalStatusConflict) {
			// Moved by a concurrent handler; leave for reconciliation.
			common.SysError(fmt.Sprintf("stripe_connect payout.failed: could not transition withdrawal %d to action_required: %v", req.ID, cerr))
			return nil
		}
		return fmt.Errorf("cas → action_required: %w", cerr)
	}
	return nil
}

// findWithdrawalByPayout locates the local WithdrawalRequest for a Stripe Payout
// by stripe_payout_id, falling back to the withdrawal_id in the payout's
// metadata. Returns (nil, nil) when the payout is not from our system.
func findWithdrawalByPayout(po *stripe.Payout) (*model.WithdrawalRequest, error) {
	if po.ID != "" {
		var req model.WithdrawalRequest
		if err := model.DB.Where("stripe_payout_id = ?", po.ID).First(&req).Error; err == nil {
			return &req, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if po.Metadata != nil {
		if wid, ok := po.Metadata["withdrawal_id"]; ok && wid != "" {
			if id, perr := strconv.ParseInt(wid, 10, 64); perr == nil {
				var req model.WithdrawalRequest
				if err := model.DB.First(&req, id).Error; err == nil {
					return &req, nil
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, err
				}
			}
		}
	}
	return nil, nil
}

// isTerminalWithdrawalStatus reports whether a withdrawal status is terminal
// (no further transitions possible). Used to make payout.failed idempotent.
func isTerminalWithdrawalStatus(status string) bool {
	return status == model.WithdrawalStatusPaid ||
		status == model.WithdrawalStatusFailed ||
		status == model.WithdrawalStatusCanceled
}
