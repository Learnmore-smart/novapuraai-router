package service

import (
	"context"
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestRunStripeConnectReconciliationPass exercises one full reconciliation pass
// against a mock client. Each subtest sets up exactly one withdrawal in a
// specific bucket state and verifies the observable outcome (final status +
// key fields). The other buckets find no rows and are no-ops.
func TestRunStripeConnectReconciliationPass(t *testing.T) {
	const (
		stripeAcctID = "acct_test_recon"
		transferID   = "tr_test_recon"
		payoutID     = "po_test_recon"
		userBalance  = 50000
		withdrawAmt  = 1000
	)

	tests := []struct {
		name       string
		setup      func(t *testing.T) (requestId int64)
		mock       func(t *testing.T, requestId int64) *mockSCC
		wantStatus string
		checkExtra func(t *testing.T, final model.WithdrawalRequest)
	}{
		{
			name: "awaiting_funds with sufficient balance → payout processed → processing",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createAwaitingFundsWithdrawal(t, uid, stripeAcctID, transferID)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					getBalanceAvailable: func(_ context.Context, acct string) (int64, error) {
						assert.Equal(t, stripeAcctID, acct)
						return int64(withdrawAmt), nil // exactly sufficient
					},
					createPayout: func(_ context.Context, p PayoutParams, _ string, acct string) (*PayoutResult, error) {
						assert.Equal(t, int64(withdrawAmt), p.AmountCents)
						assert.Equal(t, stripeAcctID, acct)
						return &PayoutResult{ID: payoutID, Status: "pending"}, nil
					},
				}
			},
			wantStatus: model.WithdrawalStatusProcessing,
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Equal(t, payoutID, final.StripePayoutId)
				assert.Equal(t, "pending", final.StripePayoutStatus)
				assert.Equal(t, 1, final.StripePayoutAttempt, "attempt must be incremented to 1")
			},
		},
		{
			name: "awaiting_funds with insufficient balance → stays awaiting_funds",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createAwaitingFundsWithdrawal(t, uid, stripeAcctID, transferID)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					getBalanceAvailable: func(_ context.Context, _ string) (int64, error) {
						return 500, nil // < withdrawAmt
					},
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						t.Fatal("createPayout must not be called when balance is insufficient")
						return nil, nil
					},
				}
			},
			wantStatus: model.WithdrawalStatusAwaitingFunds,
		},
		{
			name: "awaiting_funds balance check error → stays awaiting_funds",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createAwaitingFundsWithdrawal(t, uid, stripeAcctID, transferID)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					getBalanceAvailable: func(_ context.Context, _ string) (int64, error) {
						return 0, errors.New("stripe: balance.Get timed out")
					},
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						t.Fatal("createPayout must not be called when balance check fails")
						return nil, nil
					},
				}
			},
			wantStatus: model.WithdrawalStatusAwaitingFunds,
		},
		{
			name: "payout_creating stuck, attempts < max → retries payout → processing",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				// last_reconcile_at defaults to 0 → always considered stuck.
				return createPayoutCreatingWithdrawal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					createPayout: func(_ context.Context, _ PayoutParams, _ string, acct string) (*PayoutResult, error) {
						assert.Equal(t, stripeAcctID, acct)
						return &PayoutResult{ID: payoutID, Status: "pending"}, nil
					},
				}
			},
			wantStatus: model.WithdrawalStatusProcessing,
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Equal(t, payoutID, final.StripePayoutId)
				assert.Equal(t, 1, final.StripePayoutAttempt, "attempt must be incremented from 0 to 1")
			},
		},
		{
			name: "payout_creating stuck, attempts >= max → action_required",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createPayoutCreatingWithdrawal(t, uid, stripeAcctID, transferID, reconcileMaxPayoutAttempts)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						t.Fatal("createPayout must not be called when max attempts reached")
						return nil, nil
					},
				}
			},
			wantStatus: model.WithdrawalStatusActionRequired,
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Contains(t, final.LastReconcileError, "max payout attempts")
			},
		},
		{
			name: "action_required with usable external account → retries payout → processing",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createActionRequiredForReversal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					listExternalAccounts: func(_ context.Context, acct string) ([]ExternalAccount, error) {
						assert.Equal(t, stripeAcctID, acct)
						return []ExternalAccount{{ID: "ba_1", IsUsable: true}}, nil
					},
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						return &PayoutResult{ID: payoutID, Status: "pending"}, nil
					},
				}
			},
			wantStatus: model.WithdrawalStatusProcessing,
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Equal(t, payoutID, final.StripePayoutId)
			},
		},
		{
			name: "action_required with no usable external account → stays action_required",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createActionRequiredForReversal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					listExternalAccounts: func(_ context.Context, _ string) ([]ExternalAccount, error) {
						return []ExternalAccount{{ID: "ba_1", IsUsable: false}}, nil
					},
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						t.Fatal("createPayout must not be called when no usable external account")
						return nil, nil
					},
				}
			},
			wantStatus: model.WithdrawalStatusActionRequired,
		},
		{
			name: "action_required without transfer_id → stays action_required (manual investigation)",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmt)
				require.NoError(t, err)
				_, err = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPending, model.WithdrawalStatusActionRequired, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
					r.StripeAccountId = stripeAcctID
					r.PayoutChannel = "stripe_connect"
					return nil
				})
				require.NoError(t, err)
				return req.ID
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					listExternalAccounts: func(_ context.Context, _ string) ([]ExternalAccount, error) {
						t.Fatal("listExternalAccounts must not be called when transfer_id is missing")
						return nil, nil
					},
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						t.Fatal("createPayout must not be called when transfer_id is missing")
						return nil, nil
					},
				}
			},
			wantStatus: model.WithdrawalStatusActionRequired,
		},
		{
			name: "processing stuck >24h → stays processing (log only, no auto-reverse)",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				// last_reconcile_at defaults to 0 → always considered stuck (>24h).
				return createProcessingWithdrawalWithPayout(t, uid, stripeAcctID, transferID, payoutID, "pending")
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{} // no Stripe calls expected for processing bucket
			},
			wantStatus: model.WithdrawalStatusProcessing,
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Equal(t, payoutID, final.StripePayoutId, "payout_id must be unchanged")
			},
		},
		{
			name: "transfer_creating stuck → stays transfer_creating (log only, manual investigation)",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmt)
				require.NoError(t, err)
				// last_reconcile_at defaults to 0 → always considered stuck.
				_, err = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPending, model.WithdrawalStatusTransferCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
					r.StripeAccountId = stripeAcctID
					r.PayoutChannel = "stripe_connect"
					return nil
				})
				require.NoError(t, err)
				return req.ID
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{} // no Stripe calls expected for transfer_creating bucket
			},
			wantStatus: model.WithdrawalStatusTransferCreating,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupPayoutTestDB(t)
			withStripeConnectEnabled(t, true)

			requestId := tt.setup(t)
			mock := tt.mock(t, requestId)

			runStripeConnectReconciliationPass(context.Background(), mock)

			final := reloadWithdrawal(t, requestId)
			assert.Equal(t, tt.wantStatus, final.Status)
			if tt.checkExtra != nil {
				tt.checkExtra(t, final)
			}
		})
	}
}

// TestRunStripeConnectReconciliation_DisabledNoOp verifies that the public
// entry point is a no-op when StripeConnectEnabled is false: no rows are
// touched even if the database has awaiting_funds work to do.
func TestRunStripeConnectReconciliation_DisabledNoOp(t *testing.T) {
	const userBalance = 50000
	setupPayoutTestDB(t)
	withStripeConnectEnabled(t, false)

	uid := createPayoutTestUser(t, userBalance)
	reqID := createAwaitingFundsWithdrawal(t, uid, "acct_disabled", "tr_disabled")

	// Must not touch any rows when the feature flag is off.
	RunStripeConnectReconciliation()

	final := reloadWithdrawal(t, reqID)
	assert.Equal(t, model.WithdrawalStatusAwaitingFunds, final.Status,
		"disabled reconciliation must not change withdrawal status")
}

// TestRunStripeConnectReconciliationPass_PanicIsolation verifies that a panic
// in one bucket (e.g. a buggy Stripe client) does not abort the remaining
// buckets. The awaiting_funds bucket panics via the mock; the payout_creating
// bucket must still process its row.
func TestRunStripeConnectReconciliationPass_PanicIsolation(t *testing.T) {
	const userBalance = 50000
	setupPayoutTestDB(t)
	withStripeConnectEnabled(t, true)

	// Row 1: awaiting_funds row whose balance check panics.
	uid1 := createPayoutTestUser(t, userBalance)
	reqID1 := createAwaitingFundsWithdrawal(t, uid1, "acct_panic", "tr_panic")

	// Row 2: payout_creating row that should still be processed despite the
	// panic in the awaiting_funds bucket.
	uid2 := createPayoutTestUser(t, userBalance)
	reqID2 := createPayoutCreatingWithdrawal(t, uid2, "acct_ok", "tr_ok", 0)

	mock := &mockSCC{
		getBalanceAvailable: func(_ context.Context, _ string) (int64, error) {
			panic("simulated stripe client panic")
		},
		createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
			return &PayoutResult{ID: "po_ok", Status: "pending"}, nil
		},
	}

	// The pass must not propagate the panic.
	assert.NotPanics(t, func() {
		runStripeConnectReconciliationPass(context.Background(), mock)
	})

	// Row 1 stays awaiting_funds (panic aborted its processing).
	assert.Equal(t, model.WithdrawalStatusAwaitingFunds, reloadWithdrawal(t, reqID1).Status,
		"panicked row must remain in its original state")

	// Row 2 was processed despite the panic in the other bucket.
	final2 := reloadWithdrawal(t, reqID2)
	assert.Equal(t, model.WithdrawalStatusProcessing, final2.Status,
		"payout_creating bucket must still run after awaiting_funds bucket panic")
	assert.Equal(t, "po_ok", final2.StripePayoutId)
}
