package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stripe/stripe-go/v85"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// commissionTestUserSeq generates unique usernames/aff_codes so inviter and
// invitee rows in the same test (and across tests sharing a process) never
// collide on users.aff_code's unique index.
var commissionTestUserSeq atomic.Int64

// createCommissionUser creates a user with a unique username and aff_code,
// applying the optional configure callback first (so the caller can set
// CommissionApproved, InviterId, etc.).
func createCommissionUser(t *testing.T, configure func(*model.User)) *model.User {
	t.Helper()
	seq := commissionTestUserSeq.Add(1)
	user := &model.User{
		Username: fmt.Sprintf("commuser_%d", seq),
		Password: "password123",
		AffCode:  fmt.Sprintf("aff_%d", seq),
	}
	if configure != nil {
		configure(user)
	}
	require.NoError(t, model.DB.Create(user).Error)
	return user
}

// setupSubscriptionCommissionTestDB extends setupSubscriptionStripeTestDB with
// the Commission table, which the renewal commission path writes to but the
// base helper does not migrate.
func setupSubscriptionCommissionTestDB(t *testing.T) {
	t.Helper()
	setupSubscriptionStripeTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Commission{}))
}

// invoicePaidRenewalEvent builds an invoice.paid event carrying the fields the
// renewal commission path reads: id (invoice id, used as commission topUpId),
// subscription, period_end, billing_reason, amount_paid (minor units), and
// currency. billing_reason MUST differ from "subscription_create" so the
// handler treats this as a renewal (the first invoice's commission was already
// settled by CompleteSubscriptionOrderV2).
//
// Object is set to the typed payload directly (not round-tripped through JSON
// unmarshal) so int64 fields like period_end keep their type — otherwise
// unmarshal converts them to float64 and large timestamps render in scientific
// notation, which strconv.ParseInt(base 10) cannot parse. This mirrors the
// existing invoicePaidEvent helper.
func invoicePaidRenewalEvent(eventID, invoiceID, stripeSubId string, periodEnd int64, amountPaid int64, currency, billingReason string) stripe.Event {
	payload := map[string]any{
		"id":             invoiceID,
		"object":         "invoice",
		"subscription":   stripeSubId,
		"period_end":     periodEnd,
		"status":         "paid",
		"amount_paid":    amountPaid,
		"currency":       currency,
		"billing_reason": billingReason,
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return stripe.Event{
		ID:       eventID,
		Livemode: false,
		Type:     stripe.EventTypeInvoicePaid,
		Data: &stripe.EventData{
			Raw:    json.RawMessage(raw),
			Object: payload,
		},
	}
}

// TestProcessSubscriptionStripeEvent_InvoicePaidSettlesRenewalCommission
// verifies that a renewal invoice.paid event credits the invitee's inviter with
// affiliate commission. The commission topUpId is the Stripe invoice id (not
// the order TradeNo, which was used for the initial purchase), so each renewal
// gets its own commission row.
func TestProcessSubscriptionStripeEvent_InvoicePaidSettlesRenewalCommission(t *testing.T) {
	setupSubscriptionCommissionTestDB(t)
	originalRequireTest := setting.StripeRequireTestKeys
	originalAccountID := setting.StripeAccountID
	originalRate := common.AffCommissionRate
	t.Cleanup(func() {
		setting.StripeRequireTestKeys = originalRequireTest
		setting.StripeAccountID = originalAccountID
		common.AffCommissionRate = originalRate
	})
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""
	// 25% commission rate, matching the model commission tests.
	common.AffCommissionRate = 0.25

	inviter := createCommissionUser(t, func(u *model.User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionUser(t, func(u *model.User) {
		u.InviterId = inviter.Id
	})

	plan := &model.SubscriptionPlan{
		Title:          "NovaPura Renewal Commission",
		Enabled:        true,
		DurationUnit:   model.SubscriptionDurationMonth,
		DurationValue:  1,
		TotalAmount:    1_000_000,
		PriceAmountUSD: 19.99,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	oldEndTime := common.GetTimestamp() + 86400
	sub := &model.UserSubscription{
		UserId:               invitee.Id,
		PlanId:               plan.Id,
		Status:               model.SubscriptionStatusActive,
		StartTime:            common.GetTimestamp() - 86400,
		EndTime:              oldEndTime,
		StripeSubscriptionId: "sub_renew_comm_1",
	}
	require.NoError(t, model.DB.Create(sub).Error)

	// $20.00 paid = 2000 minor units. 25% commission = 500 cents = $5.00.
	const invoiceID = "in_renew_comm_1"
	newPeriodEnd := common.GetTimestamp() + 2592000
	event := invoicePaidRenewalEvent(
		"evt_renew_comm_1", invoiceID, "sub_renew_comm_1",
		newPeriodEnd, 2000, "usd", "renewal_cycle",
	)
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))

	// Commission must be credited to the inviter keyed by the invoice id.
	var comm model.Commission
	require.NoError(t, model.DB.Where("topup_id = ?", invoiceID).First(&comm).Error)
	assert.Equal(t, inviter.Id, comm.InviterId)
	assert.Equal(t, invitee.Id, comm.InviteeId)
	assert.Equal(t, int64(2000), comm.PaidAmountCents)
	assert.Equal(t, "USD", comm.PaidCurrency)
	assert.Equal(t, model.CommissionStatusPending, comm.Status)
	assert.Equal(t, int64(500), comm.CommissionCents)

	var refreshedInviter model.User
	require.NoError(t, model.DB.First(&refreshedInviter, inviter.Id).Error)
	assert.Equal(t, int64(500), refreshedInviter.PendingCommissionCents)
	assert.Equal(t, int64(500), refreshedInviter.CommissionTotalCents)
	assert.Zero(t, refreshedInviter.CommissionBalanceCents)

	// The renewal must also have extended the subscription's EndTime, proving
	// commission settlement did not block the renewal path.
	var refreshedSub model.UserSubscription
	require.NoError(t, model.DB.First(&refreshedSub, sub.Id).Error)
	assert.Equal(t, newPeriodEnd, refreshedSub.EndTime)
	assert.Equal(t, model.SubscriptionStatusActive, refreshedSub.Status)
}

// TestProcessSubscriptionStripeEvent_InvoicePaidRenewalCommissionIdempotent
// verifies that a replayed renewal invoice (Stripe retry with a NEW event id
// but the SAME invoice id) does not credit commission twice. The webhook event
// claim dedups by event id, so to exercise the commission-level idempotency
// (unique index on (topup_id, inviter_id)) we send a second event with a fresh
// id pointing at the same invoice.
func TestProcessSubscriptionStripeEvent_InvoicePaidRenewalCommissionIdempotent(t *testing.T) {
	setupSubscriptionCommissionTestDB(t)
	originalRequireTest := setting.StripeRequireTestKeys
	originalAccountID := setting.StripeAccountID
	originalRate := common.AffCommissionRate
	t.Cleanup(func() {
		setting.StripeRequireTestKeys = originalRequireTest
		setting.StripeAccountID = originalAccountID
		common.AffCommissionRate = originalRate
	})
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""
	common.AffCommissionRate = 0.25

	inviter := createCommissionUser(t, func(u *model.User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionUser(t, func(u *model.User) {
		u.InviterId = inviter.Id
	})

	plan := &model.SubscriptionPlan{
		Title:         "NovaPura Renewal Idempotent",
		Enabled:       true,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	sub := &model.UserSubscription{
		UserId:               invitee.Id,
		PlanId:               plan.Id,
		Status:               model.SubscriptionStatusActive,
		StartTime:            common.GetTimestamp() - 86400,
		EndTime:              common.GetTimestamp() + 86400,
		StripeSubscriptionId: "sub_renew_comm_idem_1",
	}
	require.NoError(t, model.DB.Create(sub).Error)

	const invoiceID = "in_renew_comm_idem_1"
	periodEnd := common.GetTimestamp() + 2592000

	// First delivery: credits 500 cents (25% of $20).
	first := invoicePaidRenewalEvent(
		"evt_renew_comm_idem_1", invoiceID, "sub_renew_comm_idem_1",
		periodEnd, 2000, "usd", "renewal_cycle",
	)
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), first))

	// Stripe retry: same invoice, fresh event id. The webhook event claim
	// inserts a new row, so the handler runs again — but SettleRechargeCommission's
	// unique-index check must turn the second settlement into a no-op.
	second := invoicePaidRenewalEvent(
		"evt_renew_comm_idem_2", invoiceID, "sub_renew_comm_idem_1",
		periodEnd, 2000, "usd", "renewal_cycle",
	)
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), second))

	var commissionCount int64
	require.NoError(t, model.DB.Model(&model.Commission{}).Where("topup_id = ?", invoiceID).Count(&commissionCount).Error)
	assert.EqualValues(t, 1, commissionCount, "replay must not create a second commission row")

	var refreshedInviter model.User
	require.NoError(t, model.DB.First(&refreshedInviter, inviter.Id).Error)
	assert.Equal(t, int64(500), refreshedInviter.PendingCommissionCents, "must not double-credit")
	assert.Equal(t, int64(500), refreshedInviter.CommissionTotalCents)
}

// TestProcessSubscriptionStripeEvent_InvoicePaidSkipsCommissionOnSubscriptionCreate
// verifies that the first invoice (billing_reason == "subscription_create")
// does NOT settle commission here — that commission was already settled by
// CompleteSubscriptionOrderV2 using the order TradeNo as topUpId. Settling it
// again under the invoice id would create a duplicate commission.
func TestProcessSubscriptionStripeEvent_InvoicePaidSkipsCommissionOnSubscriptionCreate(t *testing.T) {
	setupSubscriptionCommissionTestDB(t)
	originalRequireTest := setting.StripeRequireTestKeys
	originalAccountID := setting.StripeAccountID
	originalRate := common.AffCommissionRate
	t.Cleanup(func() {
		setting.StripeRequireTestKeys = originalRequireTest
		setting.StripeAccountID = originalAccountID
		common.AffCommissionRate = originalRate
	})
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""
	common.AffCommissionRate = 0.25

	inviter := createCommissionUser(t, func(u *model.User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionUser(t, func(u *model.User) {
		u.InviterId = inviter.Id
	})

	plan := &model.SubscriptionPlan{
		Title:         "NovaPura First Invoice",
		Enabled:       true,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	sub := &model.UserSubscription{
		UserId:               invitee.Id,
		PlanId:               plan.Id,
		Status:               model.SubscriptionStatusActive,
		StartTime:            common.GetTimestamp() - 86400,
		EndTime:              common.GetTimestamp() + 86400,
		StripeSubscriptionId: "sub_create_comm_1",
	}
	require.NoError(t, model.DB.Create(sub).Error)

	const invoiceID = "in_create_comm_1"
	event := invoicePaidRenewalEvent(
		"evt_create_comm_1", invoiceID, "sub_create_comm_1",
		common.GetTimestamp()+2592000, 2000, "usd", "subscription_create",
	)
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))

	// No commission row must exist for this invoice id.
	var commissionCount int64
	require.NoError(t, model.DB.Model(&model.Commission{}).Where("topup_id = ?", invoiceID).Count(&commissionCount).Error)
	assert.Zero(t, commissionCount, "first invoice must not settle commission (handled at checkout)")

	var refreshedInviter model.User
	require.NoError(t, model.DB.First(&refreshedInviter, inviter.Id).Error)
	assert.Zero(t, refreshedInviter.PendingCommissionCents)
	assert.Zero(t, refreshedInviter.CommissionTotalCents)
}

// TestSettleSubscriptionCommissionInTx_SettlementResult verifies the lower-
// level helper returns a non-nil settlement when commission is credited, so
// the renewal path can log the inviter id and amount. This locks the contract
// the webhook handler relies on for its success log line.
func TestSettleSubscriptionCommissionInTx_SettlementResult(t *testing.T) {
	setupSubscriptionCommissionTestDB(t)
	originalRate := common.AffCommissionRate
	t.Cleanup(func() { common.AffCommissionRate = originalRate })
	common.AffCommissionRate = 0.25

	inviter := createCommissionUser(t, func(u *model.User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionUser(t, func(u *model.User) {
		u.InviterId = inviter.Id
	})

	const topUpId = "in_intx_1"
	settlement, err := settleSubscriptionCommissionInTx(invitee.Id, topUpId, 2000)
	require.NoError(t, err)
	require.NotNil(t, settlement)
	assert.Equal(t, inviter.Id, settlement.InviterId)
	assert.Equal(t, invitee.Id, settlement.InviteeId)
	assert.Equal(t, topUpId, settlement.TopUpId)
	assert.Equal(t, int64(500), settlement.CommissionCents)
	assert.Positive(t, settlement.AvailableAt)

	// A second call with the same topUpId is idempotent: nil settlement, nil err.
	dup, err := settleSubscriptionCommissionInTx(invitee.Id, topUpId, 2000)
	require.NoError(t, err)
	assert.Nil(t, dup, "duplicate settlement must be a no-op")

	var commissionCount int64
	require.NoError(t, model.DB.Model(&model.Commission{}).Where("topup_id = ?", topUpId).Count(&commissionCount).Error)
	assert.EqualValues(t, 1, commissionCount)

	// Sanity: the inviter was credited exactly once.
	var refreshedInviter model.User
	require.NoError(t, model.DB.First(&refreshedInviter, inviter.Id).Error)
	assert.Equal(t, int64(500), refreshedInviter.PendingCommissionCents)
}

// TestHandleSubscriptionChargeRefunded_RevertsRenewalCommission verifies that
// a charge.refunded event carrying an invoice id reverses the commission that
// was credited for that renewal invoice. This closes the loop: renewal settles
// commission, refund reverts it.
func TestHandleSubscriptionChargeRefunded_RevertsRenewalCommission(t *testing.T) {
	setupSubscriptionCommissionTestDB(t)
	originalRate := common.AffCommissionRate
	t.Cleanup(func() { common.AffCommissionRate = originalRate })
	common.AffCommissionRate = 0.25

	inviter := createCommissionUser(t, func(u *model.User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionUser(t, func(u *model.User) {
		u.InviterId = inviter.Id
	})

	const invoiceID = "in_refund_revert_1"
	// Credit a renewal commission first (mimics a successful invoice.paid).
	settlement, err := settleSubscriptionCommissionInTx(invitee.Id, invoiceID, 2000)
	require.NoError(t, err)
	require.NotNil(t, settlement)
	require.Equal(t, int64(500), settlement.CommissionCents)

	// Build a charge.refunded event whose `invoice` field points at the same
	// invoice id — the handler maps this to topup_id = invoiceID.
	payload := map[string]any{
		"id":      "ch_refund_revert_1",
		"object":  "charge",
		"invoice": invoiceID,
		"refunds": map[string]any{"total_count": 1},
	}
	raw, err := common.Marshal(payload)
	require.NoError(t, err)
	var obj map[string]any
	require.NoError(t, common.Unmarshal(raw, &obj))
	event := stripe.Event{
		ID:       "evt_refund_revert_1",
		Livemode: false,
		Type:     stripe.EventTypeChargeRefunded,
		Data: &stripe.EventData{
			Raw:    json.RawMessage(raw),
			Object: obj,
		},
	}
	require.NoError(t, handleSubscriptionChargeRefunded(context.Background(), event))

	var comm model.Commission
	require.NoError(t, model.DB.Where("topup_id = ?", invoiceID).First(&comm).Error)
	assert.Equal(t, model.CommissionStatusReverted, comm.Status)
	assert.Positive(t, comm.RevertedAt)
	assert.Equal(t, "charge refunded", comm.RevertReason)

	var refreshedInviter model.User
	require.NoError(t, model.DB.First(&refreshedInviter, inviter.Id).Error)
	assert.Zero(t, refreshedInviter.PendingCommissionCents, "revert must debit pending commission")
	assert.Zero(t, refreshedInviter.CommissionTotalCents)

	// Idempotent: a second refund event for the same invoice is a no-op.
	payload2 := map[string]any{
		"id":      "ch_refund_revert_2",
		"object":  "charge",
		"invoice": invoiceID,
	}
	raw2, err := common.Marshal(payload2)
	require.NoError(t, err)
	var obj2 map[string]any
	require.NoError(t, common.Unmarshal(raw2, &obj2))
	replay := stripe.Event{
		ID:       "evt_refund_revert_2",
		Livemode: false,
		Type:     stripe.EventTypeChargeRefunded,
		Data: &stripe.EventData{
			Raw:    json.RawMessage(raw2),
			Object: obj2,
		},
	}
	require.NoError(t, handleSubscriptionChargeRefunded(context.Background(), replay))
	require.NoError(t, model.DB.First(&refreshedInviter, inviter.Id).Error)
	assert.Zero(t, refreshedInviter.PendingCommissionCents, "second revert must not double-debit")
	assert.Zero(t, refreshedInviter.CommissionTotalCents)
}
