package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	stripe "github.com/stripe/stripe-go/v85"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// mockSCC is a configurable mock StripeConnectClient for the payout/reversal/
// reconciliation tests. The function fields let each subtest inject the
// behavior it needs; unconfigured methods return zero values or a sentinel
// error (for methods whose nil result would be unsafe to use).
type mockSCC struct {
	createTransfer       func(ctx context.Context, p TransferParams, key string) (*TransferResult, error)
	createPayout         func(ctx context.Context, p PayoutParams, key string, acct string) (*PayoutResult, error)
	getBalanceAvailable  func(ctx context.Context, acct string) (int64, error)
	reverseTransfer      func(ctx context.Context, transferID string, amountCents int64, key string) (*ReversalResult, error)
	listExternalAccounts func(ctx context.Context, acct string) ([]ExternalAccount, error)
}

func (m *mockSCC) CreateExpressAccount(context.Context, CreateAccountParams) (*AccountResult, error) {
	return nil, nil
}
func (m *mockSCC) CreateAccountLink(context.Context, string, string, string) (*AccountLinkResult, error) {
	return nil, nil
}
func (m *mockSCC) CreateTransfer(ctx context.Context, p TransferParams, key string) (*TransferResult, error) {
	if m.createTransfer != nil {
		return m.createTransfer(ctx, p, key)
	}
	return nil, errors.New("mockSCC: createTransfer not configured")
}
func (m *mockSCC) ReverseTransfer(ctx context.Context, transferID string, amountCents int64, key string) (*ReversalResult, error) {
	if m.reverseTransfer != nil {
		return m.reverseTransfer(ctx, transferID, amountCents, key)
	}
	return nil, errors.New("mockSCC: reverseTransfer not configured")
}
func (m *mockSCC) CreatePayout(ctx context.Context, p PayoutParams, key string, acct string) (*PayoutResult, error) {
	if m.createPayout != nil {
		return m.createPayout(ctx, p, key, acct)
	}
	return nil, errors.New("mockSCC: createPayout not configured")
}
func (m *mockSCC) GetBalanceAvailableUSD(ctx context.Context, acct string) (int64, error) {
	if m.getBalanceAvailable != nil {
		return m.getBalanceAvailable(ctx, acct)
	}
	return 0, nil
}
func (m *mockSCC) RetrieveAccount(context.Context, string) (*AccountResult, error) {
	return nil, nil
}
func (m *mockSCC) ListExternalAccounts(ctx context.Context, acct string) ([]ExternalAccount, error) {
	if m.listExternalAccounts != nil {
		return m.listExternalAccounts(ctx, acct)
	}
	return nil, nil
}

// payoutTestUserSeq generates unique suffixes for test users so the
// username/aff_code/email unique indexes are never violated across subtests.
var payoutTestUserSeq atomic.Int64

func setupPayoutTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, model.DB.AutoMigrate(&model.WithdrawalRequest{}, &model.StripeConnectAccount{}))
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM withdrawal_requests")
		model.DB.Exec("DELETE FROM stripe_connect_accounts")
		model.DB.Exec("DELETE FROM users")
	})
}

func createPayoutTestUser(t *testing.T, balanceCents int64) int {
	t.Helper()
	seq := payoutTestUserSeq.Add(1)
	user := &model.User{
		Username:               fmt.Sprintf("payout_%d", seq),
		Password:               "test-password",
		Email:                  fmt.Sprintf("payout_%d@example.com", seq),
		AffCode:                fmt.Sprintf("payout_aff_%d", seq),
		Role:                   common.RoleCommonUser,
		Status:                 common.UserStatusEnabled,
		CommissionApproved:     true,
		CommissionBalanceCents: balanceCents,
	}
	require.NoError(t, model.DB.Create(user).Error)
	return user.Id
}

func createEnabledConnectAccount(t *testing.T, userId int, stripeAcctId string) {
	t.Helper()
	acc := &model.StripeConnectAccount{
		UserId:                 userId,
		StripeAccountId:        stripeAcctId,
		OnboardingState:        model.ConnectOnboardingEnabled,
		PayoutScheduleInterval: "manual",
	}
	require.NoError(t, model.DB.Create(acc).Error)
}

func reloadUser(t *testing.T, userId int) model.User {
	t.Helper()
	var u model.User
	require.NoError(t, model.DB.First(&u, userId).Error)
	return u
}

func reloadWithdrawal(t *testing.T, requestId int64) model.WithdrawalRequest {
	t.Helper()
	var r model.WithdrawalRequest
	require.NoError(t, model.DB.First(&r, requestId).Error)
	return r
}

func TestApproveStripeConnectWithdrawal(t *testing.T) {
	const (
		adminID        = 42
		stripeAcctID   = "acct_test_x"
		transferID     = "tr_test_123"
		userBalance    = 50000 // $500
		withdrawAmount = 1000  // $10 (== MinWithdrawalCents)
	)

	tests := []struct {
		name string
		// setup runs before the call; returns the withdrawal request id to approve.
		setup func(t *testing.T) (requestId int64)
		// mock configures the mockSCC function fields.
		mock func() *mockSCC
		// wantErr is the expected error sentinel (nil if no error expected).
		wantErr error
		// wantErrNonNil checks that some error is returned (non-sentinel).
		wantErrNonNil bool
		// wantStatus is the expected final status of the withdrawal.
		wantStatus string
		// wantTransferID is the expected stripe_transfer_id ("" if none).
		wantTransferID string
		// wantBalanceRefunded checks whether the user's balance should be restored.
		wantBalanceRefunded bool
	}{
		{
			name: "happy path with sufficient balance → payout_creating",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmount)
				require.NoError(t, err)
				return req.ID
			},
			mock: func() *mockSCC {
				return &mockSCC{
					createTransfer: func(_ context.Context, p TransferParams, _ string) (*TransferResult, error) {
						assert.Equal(t, int64(withdrawAmount), p.AmountCents)
						assert.Equal(t, stripeAcctID, p.Destination)
						return &TransferResult{ID: transferID, Amount: int64(withdrawAmount)}, nil
					},
					getBalanceAvailable: func(_ context.Context, acct string) (int64, error) {
						assert.Equal(t, stripeAcctID, acct)
						return int64(withdrawAmount), nil // exactly sufficient
					},
				}
			},
			wantStatus:      model.WithdrawalStatusPayoutCreating,
			wantTransferID:  transferID,
		},
		{
			name: "happy path with insufficient balance → awaiting_funds",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmount)
				require.NoError(t, err)
				return req.ID
			},
			mock: func() *mockSCC {
				return &mockSCC{
					createTransfer: func(_ context.Context, _ TransferParams, _ string) (*TransferResult, error) {
						return &TransferResult{ID: transferID, Amount: withdrawAmount}, nil
					},
					getBalanceAvailable: func(_ context.Context, _ string) (int64, error) {
						return 500, nil // < withdrawAmount → awaiting_funds
					},
				}
			},
			wantStatus:      model.WithdrawalStatusAwaitingFunds,
			wantTransferID:  transferID,
		},
		{
			name: "transfer creation fails → failed + balance refunded",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmount)
				require.NoError(t, err)
				return req.ID
			},
			mock: func() *mockSCC {
				return &mockSCC{
					createTransfer: func(_ context.Context, _ TransferParams, _ string) (*TransferResult, error) {
						return nil, errors.New("stripe: invalid request")
					},
				}
			},
			wantErrNonNil:       true,
			wantStatus:          model.WithdrawalStatusFailed,
			wantBalanceRefunded: true,
		},
		{
			name: "user has no stripe connect account → failed + balance refunded",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				// deliberately no StripeConnectAccount created
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmount)
				require.NoError(t, err)
				return req.ID
			},
			mock: func() *mockSCC {
				return &mockSCC{
					createTransfer: func(_ context.Context, _ TransferParams, _ string) (*TransferResult, error) {
						t.Fatal("createTransfer must not be called when account is missing")
						return nil, nil
					},
				}
			},
			wantErrNonNil:       true,
			wantStatus:          model.WithdrawalStatusFailed,
			wantBalanceRefunded: true,
		},
		{
			name: "withdrawal not pending → ErrWithdrawalAlreadyProcessing",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmount)
				require.NoError(t, err)
				// Move to a terminal state (paid) so the early check fires.
				_, err = model.AdminProcessWithdrawal(req.ID, model.WithdrawalStatusPaid, adminID, "manual", "", "")
				require.NoError(t, err)
				return req.ID
			},
			mock: func() *mockSCC {
				return &mockSCC{
					createTransfer: func(_ context.Context, _ TransferParams, _ string) (*TransferResult, error) {
						t.Fatal("createTransfer must not be called when withdrawal is not pending")
						return nil, nil
					},
				}
			},
			wantErr:       ErrWithdrawalAlreadyProcessing,
			wantStatus:    model.WithdrawalStatusPaid, // unchanged
			wantTransferID: "",
		},
		{
			name: "balance check fails → still transitions to awaiting_funds",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmount)
				require.NoError(t, err)
				return req.ID
			},
			mock: func() *mockSCC {
				return &mockSCC{
					createTransfer: func(_ context.Context, _ TransferParams, _ string) (*TransferResult, error) {
						return &TransferResult{ID: transferID, Amount: withdrawAmount}, nil
					},
					getBalanceAvailable: func(_ context.Context, _ string) (int64, error) {
						return 0, errors.New("stripe: balance.Get timed out")
					},
				}
			},
			wantStatus:      model.WithdrawalStatusAwaitingFunds,
			wantTransferID:  transferID,
		},
		{
			name: "CAS conflict on pending → transfer_creating → ErrWithdrawalAlreadyProcessing",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmount)
				require.NoError(t, err)
				// Simulate a concurrent approver that already moved it to transfer_creating.
				_, err = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPending, model.WithdrawalStatusTransferCreating, nil)
				require.NoError(t, err)
				return req.ID
			},
			mock: func() *mockSCC {
				return &mockSCC{
					createTransfer: func(_ context.Context, _ TransferParams, _ string) (*TransferResult, error) {
						t.Fatal("createTransfer must not be called when CAS would conflict")
						return nil, nil
					},
				}
			},
			wantErr:    ErrWithdrawalAlreadyProcessing,
			wantStatus: model.WithdrawalStatusTransferCreating, // unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupPayoutTestDB(t)
			withStripeConnectEnabled(t, true)

			requestId := tt.setup(t)
			// Capture the user id for balance verification.
			var wreq model.WithdrawalRequest
			require.NoError(t, model.DB.First(&wreq, requestId).Error)

			result, err := ApproveStripeConnectWithdrawal(context.Background(), tt.mock(), requestId, adminID)

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantErrNonNil:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}

			// Verify final withdrawal state.
			final := reloadWithdrawal(t, requestId)
			assert.Equal(t, tt.wantStatus, final.Status)
			assert.Equal(t, tt.wantTransferID, final.StripeTransferId)
			if tt.wantStatus == model.WithdrawalStatusPayoutCreating ||
				tt.wantStatus == model.WithdrawalStatusAwaitingFunds {
				assert.Equal(t, "created", final.StripeTransferStatus)
				assert.Equal(t, "stripe_connect", final.PayoutChannel)
				assert.Equal(t, adminID, final.ReviewedBy)
			}

			// Verify balance refund (or non-refund) on the user.
			u := reloadUser(t, wreq.UserId)
			if tt.wantBalanceRefunded {
				assert.Equal(t, int64(userBalance), u.CommissionBalanceCents,
					"balance should be fully refunded after failure")
			} else if tt.wantErr == ErrWithdrawalAlreadyProcessing && final.Status == model.WithdrawalStatusPaid {
				// Already-paid manual path: balance was debited at request time and
				// not refunded (manual payout confirmed). Balance = userBalance - withdrawAmount.
				assert.Equal(t, int64(userBalance-withdrawAmount), u.CommissionBalanceCents)
			}

			// result should never be nil when the withdrawal exists.
			if result == nil {
				t.Logf("result is nil (err=%v)", err)
			}
		})
	}
}

// createPayoutCreatingWithdrawal inserts a withdrawal that has already reached
// payout_creating (Transfer created, funds confirmed available), with the given
// stripe_payout_attempt. Returns the request id. Skips transfer_creating so the
// caller can also build degenerate cases (e.g. empty transfer_id) inline.
func createPayoutCreatingWithdrawal(t *testing.T, userId int, stripeAcctId, transferId string, attempt int) int64 {
	t.Helper()
	req, err := model.CreateWithdrawalRequest(userId, 1000)
	require.NoError(t, err)
	_, err = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPending, model.WithdrawalStatusTransferCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripeAccountId = stripeAcctId
		r.PayoutChannel = "stripe_connect"
		return nil
	})
	require.NoError(t, err)
	_, err = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusTransferCreating, model.WithdrawalStatusPayoutCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripeTransferId = transferId
		r.StripeTransferStatus = "created"
		r.StripePayoutAttempt = attempt
		return nil
	})
	require.NoError(t, err)
	return req.ID
}

func TestProcessStripeConnectPayout(t *testing.T) {
	const (
		stripeAcctID = "acct_test_payout"
		transferID   = "tr_test_payout"
		payoutID     = "po_test_123"
		userBalance  = 50000
		withdrawAmt  = 1000
	)

	tests := []struct {
		name         string
		setup        func(t *testing.T) (requestId int64)
		mock         func(t *testing.T, requestId int64) *mockSCC
		wantErr      bool
		wantStatus   string
		wantPayoutID string
		checkExtra   func(t *testing.T, final model.WithdrawalRequest)
	}{
		{
			name: "happy path → processing, payout_id stored",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createPayoutCreatingWithdrawal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					createPayout: func(_ context.Context, p PayoutParams, _ string, acct string) (*PayoutResult, error) {
						assert.Equal(t, int64(withdrawAmt), p.AmountCents)
						assert.Equal(t, "usd", p.Currency)
						assert.Equal(t, stripeAcctID, acct)
						return &PayoutResult{ID: payoutID, Status: "pending"}, nil
					},
				}
			},
			wantStatus:   model.WithdrawalStatusProcessing,
			wantPayoutID: payoutID,
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Equal(t, "pending", final.StripePayoutStatus)
				assert.Equal(t, 1, final.StripePayoutAttempt, "attempt must be incremented to 1")
				assert.Empty(t, final.LastReconcileError, "error must be cleared on success")
			},
		},
		{
			name: "insufficient funds error → awaiting_funds",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createPayoutCreatingWithdrawal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						return nil, fmt.Errorf("stripe payout.New: %w", &stripe.Error{
							Code: stripe.ErrorCodeInsufficientFunds,
							Msg:  "insufficient funds on connected account",
						})
					},
				}
			},
			wantStatus:   model.WithdrawalStatusAwaitingFunds,
			wantPayoutID: "",
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Contains(t, final.LastReconcileError, "insufficient_funds")
				assert.Equal(t, 1, final.StripePayoutAttempt, "attempt must still be incremented")
			},
		},
		{
			name: "other stripe error → stays payout_creating, error returned",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createPayoutCreatingWithdrawal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						return nil, errors.New("stripe: card declined")
					},
				}
			},
			wantErr:      true,
			wantStatus:   model.WithdrawalStatusPayoutCreating,
			wantPayoutID: "",
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Contains(t, final.LastReconcileError, "card declined")
				assert.Equal(t, 1, final.StripePayoutAttempt)
			},
		},
		{
			name: "not in payout_creating → no-op, no error",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmt)
				require.NoError(t, err)
				// Move to awaiting_funds (a non-payout_creating state) directly.
				_, err = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPending, model.WithdrawalStatusAwaitingFunds, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
					r.StripeAccountId = stripeAcctID
					r.StripeTransferId = transferID
					r.PayoutChannel = "stripe_connect"
					return nil
				})
				require.NoError(t, err)
				return req.ID
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						t.Fatal("createPayout must not be called when withdrawal is not in payout_creating")
						return nil, nil
					},
				}
			},
			wantStatus:   model.WithdrawalStatusAwaitingFunds, // unchanged
			wantPayoutID: "",
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Equal(t, 0, final.StripePayoutAttempt, "attempt must not be touched on no-op")
				assert.Empty(t, final.LastReconcileError)
			},
		},
		{
			name: "no transfer_id → error, no state change",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				req, err := model.CreateWithdrawalRequest(uid, withdrawAmt)
				require.NoError(t, err)
				// Skip transfer_creating: go straight to payout_creating with no transfer_id.
				_, err = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPending, model.WithdrawalStatusPayoutCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
					r.StripeAccountId = stripeAcctID
					r.PayoutChannel = "stripe_connect"
					return nil
				})
				require.NoError(t, err)
				return req.ID
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						t.Fatal("createPayout must not be called when transfer_id is missing")
						return nil, nil
					},
				}
			},
			wantErr:      true,
			wantStatus:   model.WithdrawalStatusPayoutCreating, // unchanged
			wantPayoutID: "",
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Equal(t, 0, final.StripePayoutAttempt, "attempt must not be incremented when transfer_id is missing")
			},
		},
		{
			// Simulates a concurrent worker that advances the withdrawal to
			// processing while CreatePayout is in flight. ProcessStripeConnectPayout's
			// success CAS (payout_creating → processing) then conflicts because the
			// status is no longer payout_creating; the function re-reads and returns
			// an error. The Payout exists on Stripe; reconciliation (Task 11)
			// reconciles by metadata lookup.
			name: "CAS conflict storing payout_id (concurrent advance during CreatePayout)",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createPayoutCreatingWithdrawal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, requestId int64) *mockSCC {
				return &mockSCC{
					createPayout: func(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
						_, cerr := model.TransitionWithdrawalStatus(requestId, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusProcessing, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
							r.StripePayoutId = "po_concurrent"
							r.StripePayoutStatus = "pending"
							return nil
						})
						require.NoError(t, cerr)
						return &PayoutResult{ID: payoutID, Status: "pending"}, nil
					},
				}
			},
			wantErr:      true,
			wantStatus:   model.WithdrawalStatusProcessing, // moved by the concurrent worker
			wantPayoutID: "po_concurrent",
		},
		{
			name: "idempotency key uses incremented attempt",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createPayoutCreatingWithdrawal(t, uid, stripeAcctID, transferID, 2)
			},
			mock: func(t *testing.T, requestId int64) *mockSCC {
				return &mockSCC{
					createPayout: func(_ context.Context, _ PayoutParams, key string, _ string) (*PayoutResult, error) {
						// attempt was 2, incremented to 3 → key suffix :3
						expectedKey := PayoutIdempotencyKey(requestId, 3)
						assert.Equal(t, expectedKey, key, "idempotency key must use incremented attempt (3)")
						return &PayoutResult{ID: payoutID, Status: "pending"}, nil
					},
				}
			},
			wantStatus:   model.WithdrawalStatusProcessing,
			wantPayoutID: payoutID,
			checkExtra: func(t *testing.T, final model.WithdrawalRequest) {
				assert.Equal(t, 3, final.StripePayoutAttempt, "attempt must be incremented from 2 to 3")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupPayoutTestDB(t)
			requestId := tt.setup(t)
			mock := tt.mock(t, requestId)

			result, err := ProcessStripeConnectPayout(context.Background(), mock, requestId)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}

			final := reloadWithdrawal(t, requestId)
			assert.Equal(t, tt.wantStatus, final.Status)
			assert.Equal(t, tt.wantPayoutID, final.StripePayoutId)
			if tt.checkExtra != nil {
				tt.checkExtra(t, final)
			}
		})
	}
}
