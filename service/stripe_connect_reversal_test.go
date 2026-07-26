package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/model"

	stripe "github.com/stripe/stripe-go/v85"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createActionRequiredForReversal builds a withdrawal in action_required with a
// stripe_transfer_id and the given amount already reversed. AmountCents is 1000
// (the createAwaitingFundsWithdrawal default). Used as the starting state for
// most reversal tests.
func createActionRequiredForReversal(t *testing.T, userId int, stripeAcctId, transferId string, amountReversed int64) int64 {
	t.Helper()
	reqID := createAwaitingFundsWithdrawal(t, userId, stripeAcctId, transferId)
	_, err := model.TransitionWithdrawalStatus(reqID, model.WithdrawalStatusAwaitingFunds, model.WithdrawalStatusActionRequired, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripeTransferAmountReversed = amountReversed
		return nil
	})
	require.NoError(t, err)
	return reqID
}

func TestReverseStripeConnectTransfer(t *testing.T) {
	const (
		stripeAcctID = "acct_test_rev"
		transferID   = "tr_test_rev"
		reversalID   = "trr_test_rev"
		userBalance  = 50000
		withdrawAmt  = 1000
	)

	// balanceAfterRequest is the user balance after CreateWithdrawalRequest
	// debited withdrawAmt (used to assert non-refund cases).
	const balanceAfterRequest = userBalance - withdrawAmt

	tests := []struct {
		name string
		// setup returns the withdrawal request id to reverse.
		setup func(t *testing.T) (requestId int64)
		// mock builds the mockSCC; reverseTransfer defaults to t.Fatal so
		// unintended Stripe calls fail the test.
		mock func(t *testing.T, requestId int64) *mockSCC
		// wantErrSentinel is the expected sentinel error (nil if none).
		wantErrSentinel error
		// wantErrNonNil asserts some non-sentinel error is returned.
		wantErrNonNil bool
		// wantStatus is the expected final withdrawal status.
		wantStatus string
		// wantAmountReversed is the expected stripe_transfer_amount_reversed.
		wantAmountReversed int64
		// wantBalanceRefunded: true → balance restored to userBalance; false →
		// balance stays at balanceAfterRequest.
		wantBalanceRefunded bool
	}{
		{
			name: "happy path: action_required → failed, balance refunded",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createActionRequiredForReversal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, requestId int64) *mockSCC {
				return &mockSCC{
					reverseTransfer: func(_ context.Context, trID string, amount int64, key string) (*ReversalResult, error) {
						assert.Equal(t, transferID, trID)
						assert.Equal(t, int64(withdrawAmt), amount, "should reverse full remaining amount")
						assert.Equal(t, ReversalIdempotencyKey(requestId), key)
						return &ReversalResult{ID: reversalID, AmountReversed: withdrawAmt}, nil
					},
				}
			},
			wantStatus:          model.WithdrawalStatusFailed,
			wantAmountReversed:  withdrawAmt,
			wantBalanceRefunded: true,
		},
		{
			name: "insufficient funds → stays action_required, NOT refunded",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createActionRequiredForReversal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					reverseTransfer: func(_ context.Context, _ string, _ int64, _ string) (*ReversalResult, error) {
						return nil, fmt.Errorf("stripe transferreversal.New: %w", &stripe.Error{
							Code: stripe.ErrorCodeInsufficientFunds,
							Msg:  "insufficient funds on connected account",
						})
					},
				}
			},
			wantErrSentinel:    ErrReversalInsufficientFunds,
			wantStatus:         model.WithdrawalStatusActionRequired,
			wantAmountReversed: 0,
		},
		{
			name: "other stripe error → stays action_required, NOT refunded",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				return createActionRequiredForReversal(t, uid, stripeAcctID, transferID, 0)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					reverseTransfer: func(_ context.Context, _ string, _ int64, _ string) (*ReversalResult, error) {
						return nil, errors.New("stripe: rate limited")
					},
				}
			},
			wantErrNonNil:      true,
			wantStatus:         model.WithdrawalStatusActionRequired,
			wantAmountReversed: 0,
		},
		{
			name: "no transfer_id → error, no state change",
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
					reverseTransfer: func(context.Context, string, int64, string) (*ReversalResult, error) {
						t.Fatal("reverseTransfer must not be called when transfer_id is missing")
						return nil, nil
					},
				}
			},
			wantErrNonNil: true,
			wantStatus:    model.WithdrawalStatusActionRequired,
		},
		{
			name: "terminal state (paid) → ErrReversalAlreadyTerminal",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				reqID := createActionRequiredForReversal(t, uid, stripeAcctID, transferID, 0)
				_, err := model.TransitionWithdrawalStatus(reqID, model.WithdrawalStatusActionRequired, model.WithdrawalStatusPaid, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
					r.StripePayoutStatus = "paid"
					return nil
				})
				require.NoError(t, err)
				return reqID
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					reverseTransfer: func(context.Context, string, int64, string) (*ReversalResult, error) {
						t.Fatal("reverseTransfer must not be called when withdrawal is terminal")
						return nil, nil
					},
				}
			},
			wantErrSentinel:   ErrReversalAlreadyTerminal,
			wantStatus:        model.WithdrawalStatusPaid,
			wantAmountReversed: 0,
		},
		{
			name: "already fully reversed (non-terminal) → no-op, returns nil",
			setup: func(t *testing.T) int64 {
				uid := createPayoutTestUser(t, userBalance)
				// amount_reversed == amount_cents, still in action_required.
				return createActionRequiredForReversal(t, uid, stripeAcctID, transferID, withdrawAmt)
			},
			mock: func(t *testing.T, _ int64) *mockSCC {
				return &mockSCC{
					reverseTransfer: func(context.Context, string, int64, string) (*ReversalResult, error) {
						t.Fatal("reverseTransfer must not be called when already fully reversed")
						return nil, nil
					},
				}
			},
			wantStatus:         model.WithdrawalStatusActionRequired,
			wantAmountReversed: withdrawAmt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupPayoutTestDB(t)
			withStripeConnectEnabled(t, true)

			requestId := tt.setup(t)
			mock := tt.mock(t, requestId)

			// Capture the user id for balance verification.
			var wreq model.WithdrawalRequest
			require.NoError(t, model.DB.First(&wreq, requestId).Error)

			result, err := ReverseStripeConnectTransfer(context.Background(), mock, requestId, "admin manual reversal")

			switch {
			case tt.wantErrSentinel != nil:
				require.ErrorIs(t, err, tt.wantErrSentinel)
			case tt.wantErrNonNil:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}

			final := reloadWithdrawal(t, requestId)
			assert.Equal(t, tt.wantStatus, final.Status)
			assert.Equal(t, tt.wantAmountReversed, final.StripeTransferAmountReversed)

			u := reloadUser(t, wreq.UserId)
			if tt.wantBalanceRefunded {
				assert.Equal(t, int64(userBalance), u.CommissionBalanceCents,
					"balance should be restored to original after refund")
			} else {
				assert.Equal(t, int64(balanceAfterRequest), u.CommissionBalanceCents,
					"balance must NOT be refunded in this case")
			}

			if result == nil {
				t.Logf("result is nil (err=%v)", err)
			}
		})
	}
}

// TestReverseStripeConnectTransfer_Idempotent verifies that calling reversal
// twice is safe: the first call reverses the Transfer and refunds the balance;
// the second call short-circuits at the terminal-state check (the withdrawal is
// now `failed`) without calling Stripe again or refunding a second time. The
// idempotency key is the same on both calls, but the local terminal check fires
// before any Stripe call — so Stripe is never asked twice.
func TestReverseStripeConnectTransfer_Idempotent(t *testing.T) {
	setupPayoutTestDB(t)
	withStripeConnectEnabled(t, true)

	const (
		stripeAcctID = "acct_test_rev_idem"
		transferID   = "tr_test_rev_idem"
		reversalID   = "trr_test_rev_idem"
		userBalance  = 50000
		withdrawAmt  = 1000
	)

	uid := createPayoutTestUser(t, userBalance)
	requestId := createActionRequiredForReversal(t, uid, stripeAcctID, transferID, 0)

	reverseCalls := 0
	mock := &mockSCC{
		reverseTransfer: func(_ context.Context, trID string, amount int64, key string) (*ReversalResult, error) {
			reverseCalls++
			assert.Equal(t, transferID, trID)
			assert.Equal(t, int64(withdrawAmt), amount)
			assert.Equal(t, ReversalIdempotencyKey(requestId), key)
			return &ReversalResult{ID: reversalID, AmountReversed: withdrawAmt}, nil
		},
	}

	// First call: action_required → failed, refund 1000.
	r1, err := ReverseStripeConnectTransfer(context.Background(), mock, requestId, "admin manual reversal")
	require.NoError(t, err)
	require.NotNil(t, r1)
	assert.Equal(t, model.WithdrawalStatusFailed, r1.Status)
	assert.Equal(t, int64(withdrawAmt), r1.StripeTransferAmountReversed)
	assert.Equal(t, 1, reverseCalls, "reverseTransfer must be called exactly once on first call")
	assert.Equal(t, int64(userBalance), reloadUser(t, uid).CommissionBalanceCents, "balance restored after first call")

	// Second call: withdrawal is now `failed` (terminal) → short-circuit, no
	// Stripe call, no double refund.
	r2, err := ReverseStripeConnectTransfer(context.Background(), mock, requestId, "admin manual reversal")
	require.ErrorIs(t, err, ErrReversalAlreadyTerminal)
	require.NotNil(t, r2)
	assert.Equal(t, 1, reverseCalls, "reverseTransfer must NOT be called a second time")
	assert.Equal(t, int64(userBalance), reloadUser(t, uid).CommissionBalanceCents,
		"balance must not change on the second (no-op) call")
}
