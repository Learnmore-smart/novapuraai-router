package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

// reconcileBatchSize limits how many rows each pass touches.
const reconcileBatchSize = 50

// reconcileStuckPayoutCreatingMinutes: a payout_creating row older than this
// (by last_reconcile_at) is considered stuck and retried.
const reconcileStuckPayoutCreatingMinutes = 5

// reconcileMaxPayoutAttempts: after this many payout attempts without success,
// mark the withdrawal action_required (manual intervention needed).
const reconcileMaxPayoutAttempts = 5

// reconcileProcessingStuckHours: a processing row older than this is logged
// as a stuck-payout warning (webhook may have been missed).
const reconcileProcessingStuckHours = 24

const reconcileTickInterval = 5 * time.Minute

var (
	stripeConnectReconcileOnce    sync.Once
	stripeConnectReconcileRunning atomic.Bool
)

// StartStripeConnectReconciliationTask starts the background reconciliation
// job (master node only) that retries stuck/awaiting Stripe Connect payouts
// every ~5 minutes. Mirrors the once/master/running-guard pattern of
// StartCommissionMaturityTask. Safe to call multiple times.
func StartStripeConnectReconciliationTask() {
	stripeConnectReconcileOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			common.SysLog(fmt.Sprintf("stripe connect reconciliation task started: tick=%s", reconcileTickInterval))
			ticker := time.NewTicker(reconcileTickInterval)
			defer ticker.Stop()
			// Run once at startup after a short delay to let other systems init.
			time.Sleep(30 * time.Second)
			RunStripeConnectReconciliation()
			for range ticker.C {
				RunStripeConnectReconciliation()
			}
		})
	})
}

// RunStripeConnectReconciliation is the background job entry point (single
// pass). Called by StartStripeConnectReconciliationTask on a ticker (~5 min).
// One pass:
//  1. awaiting_funds: check balance, retry payout.
//  2. payout_creating (stuck > 5min): retry payout. After max attempts, mark action_required.
//  3. action_required: check external account usability; if usable, retry payout.
//  4. processing (stuck > 24h): log warning (do NOT auto-reverse).
//  5. transfer_creating (stuck > 5min): log only (needs manual investigation).
//
// Each row is processed in its own short transaction; no Stripe call is made
// inside a transaction. Errors are logged but don't abort the whole pass.
func RunStripeConnectReconciliation() {
	if !stripeConnectReconcileRunning.CompareAndSwap(false, true) {
		return
	}
	defer stripeConnectReconcileRunning.Store(false)

	if !setting.StripeConnectEnabled {
		return
	}
	client := NewStripeConnectClient()
	if client == nil {
		common.SysError("stripe connect reconciliation: client unavailable (not configured)")
		return
	}
	runStripeConnectReconciliationPass(context.Background(), client)
}

// runStripeConnectReconciliationPass runs one full reconciliation pass against
// the given client. Split out from RunStripeConnectReconciliation so tests can
// inject a mock client without touching setting.StripeConnectEnabled or the
// running guard.
func runStripeConnectReconciliationPass(ctx context.Context, client StripeConnectClient) {
	defer func() {
		if r := recover(); r != nil {
			common.SysError(fmt.Sprintf("stripe_connect reconcile: panic in pass: %v", r))
		}
	}()
	runReconcileBucket("awaiting_funds", func() { reconcileAwaitingFunds(ctx, client) })
	runReconcileBucket("payout_creating", func() { reconcileStuckPayoutCreating(ctx, client) })
	runReconcileBucket("action_required", func() { reconcileActionRequired(ctx, client) })
	runReconcileBucket("processing", func() { reconcileStuckProcessing(ctx, client) })
	runReconcileBucket("transfer_creating", func() { reconcileStuckTransferCreating(ctx, client) })
}

// runReconcileBucket runs one bucket's pass, isolating panics so a failure in
// one bucket does not abort the others. recover does not catch runtime.Goexit,
// so test t.Fatal/require failures inside a bucket still propagate.
func runReconcileBucket(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			common.SysError(fmt.Sprintf("stripe_connect reconcile: panic in %s bucket: %v", name, r))
		}
	}()
	fn()
}

// reconcileAwaitingFunds picks up rows waiting for funds on the connected
// account. When the available balance covers the withdrawal, CAS to
// payout_creating and retry the payout.
func reconcileAwaitingFunds(ctx context.Context, client StripeConnectClient) {
	var reqs []model.WithdrawalRequest
	if err := model.DB.Where("status = ?", model.WithdrawalStatusAwaitingFunds).
		Limit(reconcileBatchSize).Find(&reqs).Error; err != nil {
		common.SysError(fmt.Sprintf("stripe_connect reconcile: awaiting_funds load failed: %v", err))
		return
	}
	for i := range reqs {
		req := reqs[i]
		bal, balErr := client.GetBalanceAvailableUSD(ctx, req.StripeAccountId)
		if balErr != nil {
			common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d balance check failed: %v", req.ID, balErr))
			continue
		}
		if bal < req.AmountCents {
			continue // still awaiting funds
		}
		_, cerr := model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusAwaitingFunds, model.WithdrawalStatusPayoutCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
			r.LastReconcileError = ""
			r.LastReconcileAt = time.Now().Unix()
			return nil
		})
		if cerr != nil {
			common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d CAS awaiting_funds → payout_creating failed: %v", req.ID, cerr))
			continue
		}
		if _, pErr := ProcessStripeConnectPayout(ctx, client, req.ID); pErr != nil {
			common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d payout retry failed: %v", req.ID, pErr))
		}
	}
}

// reconcileStuckPayoutCreating retries payout_creating rows that have not made
// progress in reconcileStuckPayoutCreatingMinutes. After reconcileMaxPayoutAttempts
// without success, marks the withdrawal action_required for manual intervention.
func reconcileStuckPayoutCreating(ctx context.Context, client StripeConnectClient) {
	stuckBefore := time.Now().Unix() - int64(reconcileStuckPayoutCreatingMinutes*60)
	var reqs []model.WithdrawalRequest
	if err := model.DB.Where("status = ? AND last_reconcile_at < ?", model.WithdrawalStatusPayoutCreating, stuckBefore).
		Limit(reconcileBatchSize).Find(&reqs).Error; err != nil {
		common.SysError(fmt.Sprintf("stripe_connect reconcile: payout_creating load failed: %v", err))
		return
	}
	for i := range reqs {
		req := reqs[i]
		if req.StripePayoutAttempt >= reconcileMaxPayoutAttempts {
			_, cerr := model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusActionRequired, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
				r.LastReconcileError = truncateReasonLocal("max payout attempts reached", 2000)
				r.LastReconcileAt = time.Now().Unix()
				return nil
			})
			if cerr != nil {
				common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d CAS payout_creating → action_required failed: %v", req.ID, cerr))
			}
			continue
		}
		if _, pErr := ProcessStripeConnectPayout(ctx, client, req.ID); pErr != nil {
			common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d payout retry failed: %v", req.ID, pErr))
		}
	}
}

// reconcileActionRequired checks whether the connected account now has a usable
// external account (bank fixed). If so, re-enters the payout flow.
func reconcileActionRequired(ctx context.Context, client StripeConnectClient) {
	var reqs []model.WithdrawalRequest
	if err := model.DB.Where("status = ?", model.WithdrawalStatusActionRequired).
		Limit(reconcileBatchSize).Find(&reqs).Error; err != nil {
		common.SysError(fmt.Sprintf("stripe_connect reconcile: action_required load failed: %v", err))
		return
	}
	for i := range reqs {
		req := reqs[i]
		if req.StripeTransferId == "" {
			// Pre-Transfer failure; nothing to reverse or retry here.
			common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d action_required without transfer_id — needs manual investigation", req.ID))
			continue
		}
		extAccts, extErr := client.ListExternalAccounts(ctx, req.StripeAccountId)
		if extErr != nil {
			common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d list external accounts failed: %v", req.ID, extErr))
			continue
		}
		usable := false
		for _, ea := range extAccts {
			if ea.IsUsable {
				usable = true
				break
			}
		}
		if !usable {
			continue // still waiting for bank fix
		}
		_, cerr := model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusActionRequired, model.WithdrawalStatusPayoutCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
			r.LastReconcileError = ""
			r.LastReconcileAt = time.Now().Unix()
			return nil
		})
		if cerr != nil {
			common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d CAS action_required → payout_creating failed: %v", req.ID, cerr))
			continue
		}
		if _, pErr := ProcessStripeConnectPayout(ctx, client, req.ID); pErr != nil {
			common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d payout retry failed: %v", req.ID, pErr))
		}
	}
}

// reconcileStuckProcessing logs a warning for processing rows stuck longer than
// reconcileProcessingStuckHours (a webhook may have been missed). It does NOT
// auto-reverse — the payout may still settle, or an admin should decide.
func reconcileStuckProcessing(ctx context.Context, _ StripeConnectClient) {
	stuckBefore := time.Now().Unix() - int64(reconcileProcessingStuckHours*3600)
	var reqs []model.WithdrawalRequest
	if err := model.DB.Where("status = ? AND last_reconcile_at < ?", model.WithdrawalStatusProcessing, stuckBefore).
		Limit(reconcileBatchSize).Find(&reqs).Error; err != nil {
		common.SysError(fmt.Sprintf("stripe_connect reconcile: processing load failed: %v", err))
		return
	}
	for i := range reqs {
		req := reqs[i]
		common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d stuck in processing for >24h, payout_id=%s", req.ID, req.StripePayoutId))
	}
}

// reconcileStuckTransferCreating logs a warning for transfer_creating rows
// stuck longer than reconcileStuckPayoutCreatingMinutes. This state is rare
// (ApproveStripeConnectWithdrawal crashed between CAS and the Stripe call);
// it needs manual investigation rather than automatic retry.
func reconcileStuckTransferCreating(ctx context.Context, _ StripeConnectClient) {
	stuckBefore := time.Now().Unix() - int64(reconcileStuckPayoutCreatingMinutes*60)
	var reqs []model.WithdrawalRequest
	if err := model.DB.Where("status = ? AND last_reconcile_at < ?", model.WithdrawalStatusTransferCreating, stuckBefore).
		Limit(reconcileBatchSize).Find(&reqs).Error; err != nil {
		common.SysError(fmt.Sprintf("stripe_connect reconcile: transfer_creating load failed: %v", err))
		return
	}
	for i := range reqs {
		req := reqs[i]
		common.SysError(fmt.Sprintf("stripe_connect reconcile: withdrawal %d stuck in transfer_creating — needs manual investigation", req.ID))
	}
}
