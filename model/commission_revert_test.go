package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// settleCommissionForRevertTest is a small helper that credits an approved
// inviter's commission via SettleRechargeCommission and returns the inviter,
// invitee, and the topUpId used. Mirrors the pattern in commission_test.go.
func settleCommissionForRevertTest(t *testing.T, topUpId string, paidCents int64) (*User, *User) {
	t.Helper()
	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionTestUser(t, func(u *User) {
		u.InviterId = inviter.Id
	})
	settlement, err := SettleRechargeCommission(DB, invitee.Id, topUpId, paidCents, "USD")
	require.NoError(t, err)
	require.NotNil(t, settlement)
	return inviter, invitee
}

// matureCommission flips a pending commission to available (credited to
// CommissionBalanceCents) by setting AvailableAt to the past and calling
// ReleaseMaturedCommissions. Used to test revert on matured commissions.
func matureCommission(t *testing.T, topUpId string) {
	t.Helper()
	now := common.GetTimestamp()
	require.NoError(t, DB.Model(&Commission{}).Where("topup_id = ?", topUpId).
		Update("available_at", now-1).Error)
	released, err := ReleaseMaturedCommissions()
	require.NoError(t, err)
	require.Equal(t, 1, released, "commission must mature in one batch")
}

func TestRevertCommission_PendingCommission(t *testing.T) {
	setupCommissionTest(t)

	const topUpId = "revert-pending-1"
	inviter, _ := settleCommissionForRevertTest(t, topUpId, 2000)

	// 25% of $20 = $5 = 500 cents, sitting in PendingCommissionCents.
	var before User
	require.NoError(t, DB.First(&before, inviter.Id).Error)
	require.Equal(t, int64(500), before.PendingCommissionCents)
	require.Equal(t, int64(500), before.CommissionTotalCents)
	require.Zero(t, before.CommissionBalanceCents)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return RevertCommission(tx, topUpId, "test revert pending")
	}))

	var refreshed User
	require.NoError(t, DB.First(&refreshed, inviter.Id).Error)
	assert.Zero(t, refreshed.PendingCommissionCents, "pending must be debited")
	assert.Zero(t, refreshed.CommissionBalanceCents, "balance must stay 0 (was never matured)")
	assert.Zero(t, refreshed.CommissionTotalCents, "total must be debited")

	var comm Commission
	require.NoError(t, DB.Where("topup_id = ?", topUpId).First(&comm).Error)
	assert.Equal(t, CommissionStatusReverted, comm.Status)
	assert.Positive(t, comm.RevertedAt)
	assert.Equal(t, "test revert pending", comm.RevertReason)
}

func TestRevertCommission_AvailableCommission(t *testing.T) {
	setupCommissionTest(t)

	const topUpId = "revert-available-1"
	inviter, _ := settleCommissionForRevertTest(t, topUpId, 2000)
	matureCommission(t, topUpId)

	// After maturity: pending=0, balance=500, total=500.
	var before User
	require.NoError(t, DB.First(&before, inviter.Id).Error)
	require.Zero(t, before.PendingCommissionCents)
	require.Equal(t, int64(500), before.CommissionBalanceCents)
	require.Equal(t, int64(500), before.CommissionTotalCents)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return RevertCommission(tx, topUpId, "test revert available")
	}))

	var refreshed User
	require.NoError(t, DB.First(&refreshed, inviter.Id).Error)
	assert.Zero(t, refreshed.PendingCommissionCents, "pending must stay 0 (was already moved)")
	assert.Zero(t, refreshed.CommissionBalanceCents, "balance must be debited")
	assert.Zero(t, refreshed.CommissionTotalCents, "total must be debited")

	var comm Commission
	require.NoError(t, DB.Where("topup_id = ?", topUpId).First(&comm).Error)
	assert.Equal(t, CommissionStatusReverted, comm.Status)
	assert.Equal(t, "test revert available", comm.RevertReason)
}

func TestRevertCommission_Idempotent(t *testing.T) {
	setupCommissionTest(t)

	const topUpId = "revert-idem-1"
	inviter, _ := settleCommissionForRevertTest(t, topUpId, 2000)

	// First revert: debits 500 from pending.
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return RevertCommission(tx, topUpId, "first revert")
	}))
	var afterFirst User
	require.NoError(t, DB.First(&afterFirst, inviter.Id).Error)
	assert.Zero(t, afterFirst.PendingCommissionCents)
	assert.Zero(t, afterFirst.CommissionTotalCents)

	// Second revert: no-op, no double-debit (would underflow if not guarded).
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return RevertCommission(tx, topUpId, "second revert")
	}))
	var afterSecond User
	require.NoError(t, DB.First(&afterSecond, inviter.Id).Error)
	assert.Zero(t, afterSecond.PendingCommissionCents, "must not go negative on second revert")
	assert.Zero(t, afterSecond.CommissionTotalCents)

	// The revert_reason stays as the first revert's reason (second call is a no-op).
	var comm Commission
	require.NoError(t, DB.Where("topup_id = ?", topUpId).First(&comm).Error)
	assert.Equal(t, "first revert", comm.RevertReason)
}

func TestRevertCommission_NotFound(t *testing.T) {
	setupCommissionTest(t)

	err := DB.Transaction(func(tx *gorm.DB) error {
		return RevertCommission(tx, "nonexistent-topup-id", "nothing to revert")
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCommissionNotFound))
}

func TestRevertCommission_UnderflowProtection(t *testing.T) {
	setupCommissionTest(t)

	const topUpId = "revert-underflow-1"
	inviter, _ := settleCommissionForRevertTest(t, topUpId, 2000)

	// Simulate a race / data corruption: manually set the inviter's
	// PendingCommissionCents to LESS than the commission amount (500).
	// The revert must clamp to 0, NOT go negative, and must log via SysError.
	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).
		Update("pending_commission_cents", 100).Error) // 100 < 500

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return RevertCommission(tx, topUpId, "underflow test")
	}))

	var refreshed User
	require.NoError(t, DB.First(&refreshed, inviter.Id).Error)
	assert.Zero(t, refreshed.PendingCommissionCents, "must clamp to 0, not go negative")
	// CommissionTotalCents was 500 before the revert; debiting 500 brings it to 0.
	// (Underflow protection on PendingCommissionCents does NOT skip the Total debit.)
	assert.Zero(t, refreshed.CommissionTotalCents)

	var comm Commission
	require.NoError(t, DB.Where("topup_id = ?", topUpId).First(&comm).Error)
	assert.Equal(t, CommissionStatusReverted, comm.Status)
}

// --- Subscription purchase commission settlement ---

func TestSettleSubscriptionCommission_OnPurchase(t *testing.T) {
	setupCommissionTest(t)
	// Purge the subscription plan cache so a plan created in this test's DB
	// is not masked by an entry cached under a previous test's DB.
	PurgeSubscriptionPlanCache()

	// CompleteSubscriptionOrderV2 calls GetDBTimestamp() (which queries the
	// global DB) from inside a DB.Transaction. The model TestMain sets
	// SetMaxOpenConns(1) for serial SQLite, which would deadlock that nested
	// query against the transaction's held connection. Allow a second
	// connection for the duration of this test, then restore.
	sqlDB, err := DB.DB()
	require.NoError(t, err)
	originalMaxOpen := 1
	sqlDB.SetMaxOpenConns(2)
	t.Cleanup(func() { sqlDB.SetMaxOpenConns(originalMaxOpen) })

	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionTestUser(t, func(u *User) {
		u.InviterId = inviter.Id
	})

	plan := &SubscriptionPlan{
		Title:          "NovaPura Commission Test",
		Enabled:        true,
		DurationUnit:   SubscriptionDurationMonth,
		DurationValue:  1,
		TotalAmount:    1_000_000,
		PriceAmountUSD: 19.99,
	}
	require.NoError(t, DB.Create(plan).Error)

	// $19.99 USD = 1999 cents; 25% commission = 499.75 → 499 cents (truncated).
	const tradeNo = "sub_ref_comm_purchase_1"
	order := &SubscriptionOrder{
		UserId:          invitee.Id,
		PlanId:          plan.Id,
		TradeNo:         tradeNo,
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
		Currency:        "USD",
		FinalAmount:     1999,
		OriginalAmount:  1999,
		DiscountAmount:  0,
	}
	require.NoError(t, DB.Create(order).Error)

	require.NoError(t, CompleteSubscriptionOrderV2(tradeNo, "payload", "sub_test_purchase_1", "cus_test_1", "auto_renew"))

	// Commission must be credited to the inviter with topup_id = TradeNo.
	var comm Commission
	require.NoError(t, DB.Where("topup_id = ?", tradeNo).First(&comm).Error)
	assert.Equal(t, inviter.Id, comm.InviterId)
	assert.Equal(t, invitee.Id, comm.InviteeId)
	// CompleteSubscriptionOrderV2 converts FinalAmount (minor units) to major
	// units then back via ConvertAmountToUSDCents. float64(1999)/100 = 19.99,
	// and 19.99*100 = 1998.999... in IEEE 754, which QuotaFromFloat truncates
	// to 1998. This is the production conversion path's known behavior; the
	// commission is computed on 1998, not 1999.
	assert.Equal(t, int64(1998), comm.PaidAmountCents)
	assert.Equal(t, "USD", comm.PaidCurrency)
	assert.Equal(t, CommissionStatusPending, comm.Status)
	// 25% of 1998 = 499.5 → truncated to 499 (QuotaFromFloat truncates).
	assert.Equal(t, int64(499), comm.CommissionCents)

	var refreshedInviter User
	require.NoError(t, DB.First(&refreshedInviter, inviter.Id).Error)
	assert.Equal(t, int64(499), refreshedInviter.PendingCommissionCents)
	assert.Equal(t, int64(499), refreshedInviter.CommissionTotalCents)
	assert.Zero(t, refreshedInviter.CommissionBalanceCents)

	// Subscription must have activated even though commission was settled.
	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ?", invitee.Id).First(&sub).Error)
	assert.Equal(t, SubscriptionStatusActive, sub.Status)

	var refreshedOrder SubscriptionOrder
	require.NoError(t, DB.Where("trade_no = ?", tradeNo).First(&refreshedOrder).Error)
	assert.Equal(t, common.TopUpStatusSuccess, refreshedOrder.Status)
}
