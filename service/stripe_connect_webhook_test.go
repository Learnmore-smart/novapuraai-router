package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	stripe "github.com/stripe/stripe-go/v85"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// makeConnectEvent builds a *stripe.Event for tests by marshaling obj into
// Data.Raw via common.Marshal (per AGENTS.md). json.RawMessage is referenced
// only as a type (allowed); the byte→RawMessage conversion is not a
// marshal/unmarshal call.
func makeConnectEvent(t *testing.T, eventType string, obj any, account string) *stripe.Event {
	t.Helper()
	raw, err := common.Marshal(obj)
	require.NoError(t, err)
	return &stripe.Event{
		Type:    stripe.EventType(eventType),
		Data:    &stripe.EventData{Raw: json.RawMessage(raw)},
		Account: account,
	}
}

// createProcessingWithdrawalWithPayout builds a withdrawal that has reached
// processing with the given stripe_payout_id / status. Used by payout.paid and
// payout.failed tests.
func createProcessingWithdrawalWithPayout(t *testing.T, userId int, stripeAcctId, transferId, payoutId, payoutStatus string) int64 {
	t.Helper()
	reqID := createPayoutCreatingWithdrawal(t, userId, stripeAcctId, transferId, 0)
	_, err := model.TransitionWithdrawalStatus(reqID, model.WithdrawalStatusPayoutCreating, model.WithdrawalStatusProcessing, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripePayoutId = payoutId
		r.StripePayoutStatus = payoutStatus
		return nil
	})
	require.NoError(t, err)
	return reqID
}

// createAwaitingFundsWithdrawal builds a withdrawal stuck in awaiting_funds
// (Transfer created, connected-account balance insufficient for the payout).
func createAwaitingFundsWithdrawal(t *testing.T, userId int, stripeAcctId, transferId string) int64 {
	t.Helper()
	req, err := model.CreateWithdrawalRequest(userId, 1000)
	require.NoError(t, err)
	_, err = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusPending, model.WithdrawalStatusTransferCreating, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripeAccountId = stripeAcctId
		r.PayoutChannel = "stripe_connect"
		return nil
	})
	require.NoError(t, err)
	_, err = model.TransitionWithdrawalStatus(req.ID, model.WithdrawalStatusTransferCreating, model.WithdrawalStatusAwaitingFunds, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
		r.StripeTransferId = transferId
		r.StripeTransferStatus = "created"
		return nil
	})
	require.NoError(t, err)
	return req.ID
}

func TestHandleConnectEvent(t *testing.T) {
	const (
		stripeAcctID = "acct_test_wh"
		transferID   = "tr_test_wh"
		payoutID     = "po_test_wh"
		userBalance  = 50000
	)

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "account.updated: existing account → OnboardingState=enabled",
			run: func(t *testing.T) {
				uid := createPayoutTestUser(t, userBalance)
				acc := &model.StripeConnectAccount{
					UserId:                 uid,
					StripeAccountId:        stripeAcctID,
					OnboardingState:        model.ConnectOnboardingOnboarding,
					PayoutScheduleInterval: "manual",
				}
				require.NoError(t, model.DB.Create(acc).Error)

				evt := makeConnectEvent(t, "account.updated", &stripe.Account{
					ID:               stripeAcctID,
					Email:            "updated@example.com",
					Country:          "US",
					PayoutsEnabled:   true,
					DetailsSubmitted: true,
					Settings: &stripe.AccountSettings{
						Payouts: &stripe.AccountSettingsPayouts{
							Schedule: &stripe.AccountSettingsPayoutsSchedule{
								Interval: stripe.AccountSettingsPayoutsScheduleIntervalManual,
							},
						},
					},
					Requirements: &stripe.AccountRequirements{
						CurrentlyDue:  []string{},
						EventuallyDue: []string{},
					},
				}, "")

				require.NoError(t, HandleConnectEvent(context.Background(), nil, evt))

				updated, err := model.GetStripeConnectAccountByStripeId(stripeAcctID)
				require.NoError(t, err)
				assert.Equal(t, model.ConnectOnboardingEnabled, updated.OnboardingState)
				assert.Equal(t, "updated@example.com", updated.Email)
				assert.Equal(t, "US", updated.Country)
				assert.True(t, updated.PayoutsEnabled)
				assert.True(t, updated.DetailsSubmitted)
				assert.Equal(t, "manual", updated.PayoutScheduleInterval)
			},
		},
		{
			name: "account.updated: unknown account → no-op",
			run: func(t *testing.T) {
				evt := makeConnectEvent(t, "account.updated", &stripe.Account{
					ID:               "acct_unknown",
					Email:            "nobody@example.com",
					PayoutsEnabled:   true,
					DetailsSubmitted: true,
				}, "")

				require.NoError(t, HandleConnectEvent(context.Background(), nil, evt))

				_, err := model.GetStripeConnectAccountByStripeId("acct_unknown")
				assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
			},
		},
		{
			name: "payout.paid: processing → paid + user log",
			run: func(t *testing.T) {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				reqID := createProcessingWithdrawalWithPayout(t, uid, stripeAcctID, transferID, payoutID, "pending")
				balanceBefore := reloadUser(t, uid).CommissionBalanceCents

				evt := makeConnectEvent(t, "payout.paid", &stripe.Payout{
					ID:     payoutID,
					Status: stripe.PayoutStatusPaid,
					Amount: 1000,
				}, stripeAcctID)

				require.NoError(t, HandleConnectEvent(context.Background(), nil, evt))

				final := reloadWithdrawal(t, reqID)
				assert.Equal(t, model.WithdrawalStatusPaid, final.Status)
				assert.Equal(t, "paid", final.StripePayoutStatus)
				assert.Empty(t, final.LastReconcileError)

				// Balance must NOT change on payout.paid (debited at request time).
				assert.Equal(t, balanceBefore, reloadUser(t, uid).CommissionBalanceCents)

				// A system log was recorded for the user.
				var logs []model.Log
				require.NoError(t, model.LOG_DB.Where("user_id = ? AND type = ?", uid, model.LogTypeSystem).Find(&logs).Error)
				found := false
				for _, l := range logs {
					if strings.Contains(l.Content, "提现已完成") && strings.Contains(l.Content, "Stripe Connect") {
						found = true
						break
					}
				}
				assert.True(t, found, "expected a payout-completed user log")
			},
		},
		{
			name: "payout.paid: already paid → idempotent",
			run: func(t *testing.T) {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				reqID := createProcessingWithdrawalWithPayout(t, uid, stripeAcctID, transferID, payoutID, "pending")
				_, err := model.TransitionWithdrawalStatus(reqID, model.WithdrawalStatusProcessing, model.WithdrawalStatusPaid, func(tx *gorm.DB, r *model.WithdrawalRequest) error {
					r.StripePayoutStatus = "paid"
					return nil
				})
				require.NoError(t, err)

				evt := makeConnectEvent(t, "payout.paid", &stripe.Payout{
					ID:     payoutID,
					Status: stripe.PayoutStatusPaid,
				}, stripeAcctID)

				require.NoError(t, HandleConnectEvent(context.Background(), nil, evt))

				final := reloadWithdrawal(t, reqID)
				assert.Equal(t, model.WithdrawalStatusPaid, final.Status) // unchanged
			},
		},
		{
			name: "payout.failed: processing → action_required, balance NOT refunded",
			run: func(t *testing.T) {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				reqID := createProcessingWithdrawalWithPayout(t, uid, stripeAcctID, transferID, payoutID, "in_transit")
				balanceBefore := reloadUser(t, uid).CommissionBalanceCents

				evt := makeConnectEvent(t, "payout.failed", &stripe.Payout{
					ID:             payoutID,
					Status:         stripe.PayoutStatusFailed,
					FailureMessage: "bank account closed",
				}, stripeAcctID)

				require.NoError(t, HandleConnectEvent(context.Background(), nil, evt))

				final := reloadWithdrawal(t, reqID)
				assert.Equal(t, model.WithdrawalStatusActionRequired, final.Status)
				assert.Contains(t, final.LastReconcileError, "payout.failed")
				assert.Contains(t, final.LastReconcileError, "bank account closed")

				// Balance must NOT be refunded — the Transfer still exists on the
				// connected account; reversal (Task 11) is required first.
				assert.Equal(t, balanceBefore, reloadUser(t, uid).CommissionBalanceCents)
			},
		},
		{
			name: "payout.created: payout_creating with empty payout_id → processing",
			run: func(t *testing.T) {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				// createPayoutCreatingWithdrawal leaves stripe_payout_id empty.
				reqID := createPayoutCreatingWithdrawal(t, uid, stripeAcctID, transferID, 0)

				// Metadata carries the withdrawal_id so findWithdrawalByPayout can
				// locate the row when stripe_payout_id is still empty.
				evt := makeConnectEvent(t, "payout.created", &stripe.Payout{
					ID:       payoutID,
					Status:   stripe.PayoutStatusPending,
					Amount:   1000,
					Metadata: map[string]string{"withdrawal_id": strconv.FormatInt(reqID, 10)},
				}, stripeAcctID)

				require.NoError(t, HandleConnectEvent(context.Background(), nil, evt))

				final := reloadWithdrawal(t, reqID)
				assert.Equal(t, model.WithdrawalStatusProcessing, final.Status)
				assert.Equal(t, payoutID, final.StripePayoutId)
				assert.Equal(t, "pending", final.StripePayoutStatus)
			},
		},
		{
			name: "payout.created: idempotent, already processing → just updates status",
			run: func(t *testing.T) {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				reqID := createProcessingWithdrawalWithPayout(t, uid, stripeAcctID, transferID, payoutID, "pending")

				evt := makeConnectEvent(t, "payout.created", &stripe.Payout{
					ID:     payoutID,
					Status: stripe.PayoutStatusInTransit, // status changed pending → in_transit
				}, stripeAcctID)

				require.NoError(t, HandleConnectEvent(context.Background(), nil, evt))

				final := reloadWithdrawal(t, reqID)
				assert.Equal(t, model.WithdrawalStatusProcessing, final.Status) // unchanged
				assert.Equal(t, "in_transit", final.StripePayoutStatus)          // refreshed
				assert.Equal(t, payoutID, final.StripePayoutId)
			},
		},
		{
			name: "balance.available: awaiting_funds → payout_creating → processing",
			run: func(t *testing.T) {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				reqID := createAwaitingFundsWithdrawal(t, uid, stripeAcctID, transferID)

				evt := makeConnectEvent(t, "balance.available", &stripe.Balance{}, stripeAcctID)
				mock := &mockSCC{
					createPayout: func(_ context.Context, p PayoutParams, _ string, acct string) (*PayoutResult, error) {
						assert.Equal(t, stripeAcctID, acct)
						assert.Equal(t, int64(1000), p.AmountCents)
						return &PayoutResult{ID: payoutID, Status: "pending"}, nil
					},
				}

				require.NoError(t, HandleConnectEvent(context.Background(), mock, evt))

				final := reloadWithdrawal(t, reqID)
				assert.Equal(t, model.WithdrawalStatusProcessing, final.Status)
				assert.Equal(t, payoutID, final.StripePayoutId)
				assert.Equal(t, "pending", final.StripePayoutStatus)
				assert.Equal(t, 1, final.StripePayoutAttempt, "attempt must be incremented")
			},
		},
		{
			name: "balance.available: platform account (evt.Account empty) → no-op",
			run: func(t *testing.T) {
				uid := createPayoutTestUser(t, userBalance)
				createEnabledConnectAccount(t, uid, stripeAcctID)
				reqID := createAwaitingFundsWithdrawal(t, uid, stripeAcctID, transferID)

				evt := makeConnectEvent(t, "balance.available", &stripe.Balance{}, "")

				require.NoError(t, HandleConnectEvent(context.Background(), nil, evt))

				final := reloadWithdrawal(t, reqID)
				assert.Equal(t, model.WithdrawalStatusAwaitingFunds, final.Status) // unchanged
			},
		},
		{
			name: "unknown event type → no-op",
			run: func(t *testing.T) {
				uid := createPayoutTestUser(t, userBalance)
				evt := makeConnectEvent(t, "some.unknown.event", &stripe.Account{ID: stripeAcctID}, "")

				require.NoError(t, HandleConnectEvent(context.Background(), nil, evt))

				// No withdrawals created; nothing touched.
				var count int64
				require.NoError(t, model.DB.Model(&model.WithdrawalRequest{}).Where("user_id = ?", uid).Count(&count).Error)
				assert.Equal(t, int64(0), count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupPayoutTestDB(t)
			tt.run(t)
		})
	}
}

// TestHandleConnectEvent_MalformedPayload verifies the dispatcher surfaces parse
// errors (so the controller can log them) instead of silently dropping events.
func TestHandleConnectEvent_MalformedPayload(t *testing.T) {
	setupPayoutTestDB(t)
	evt := &stripe.Event{
		Type: stripe.EventType("account.updated"),
		Data: &stripe.EventData{Raw: json.RawMessage("not valid json")},
	}
	err := HandleConnectEvent(context.Background(), nil, evt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse account.updated")
}
