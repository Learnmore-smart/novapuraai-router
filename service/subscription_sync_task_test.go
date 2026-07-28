package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedSyncSub inserts a UserSubscription row with explicit fields. The
// service-package TestMain (in task_billing_test.go) already migrates
// user_subscriptions, so no extra setup is needed.
func seedSyncSub(t *testing.T, sub *model.UserSubscription) {
	t.Helper()
	require.NoError(t, model.DB.Create(sub).Error)
}

func loadSyncSub(t *testing.T, id int) model.UserSubscription {
	t.Helper()
	var sub model.UserSubscription
	require.NoError(t, model.DB.Where("id = ?", id).First(&sub).Error)
	return sub
}

// TestApplySubscriptionStatusTransition_AppliesLegalTransition verifies the
// service-level wrapper delegates to model.TransitionSubscriptionStatus and
// commits the transaction on a legal edge.
func TestApplySubscriptionStatusTransition_AppliesLegalTransition(t *testing.T) {
	truncate(t)
	now := common.GetTimestamp()
	seedSyncSub(t, &model.UserSubscription{
		Id: 8601, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: now + 3600,
		Status: model.SubscriptionStatusActive, UpdatedAt: now - 600,
	})

	before := common.GetTimestamp()
	require.NoError(t, applySubscriptionStatusTransition(8601, model.SubscriptionStatusCanceling))

	sub := loadSyncSub(t, 8601)
	assert.Equal(t, model.SubscriptionStatusCanceling, sub.Status)
	assert.GreaterOrEqual(t, sub.UpdatedAt, before)
}

// TestApplySubscriptionStatusTransition_RejectsIllegalTransition verifies the
// service-level wrapper propagates the state-machine violation and does not
// mutate the row.
func TestApplySubscriptionStatusTransition_RejectsIllegalTransition(t *testing.T) {
	truncate(t)
	now := common.GetTimestamp()
	seedSyncSub(t, &model.UserSubscription{
		Id: 8602, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: now - 60,
		Status: model.SubscriptionStatusCanceled, UpdatedAt: now - 60,
	})

	err := applySubscriptionStatusTransition(8602, model.SubscriptionStatusActive)
	require.ErrorIs(t, err, model.ErrInvalidSubscriptionStatusTransition)

	sub := loadSyncSub(t, 8602)
	assert.Equal(t, model.SubscriptionStatusCanceled, sub.Status, "status must not mutate on illegal transition")
}

// TestSyncSubscriptionEndTime_AdvancesWhenLagging verifies the EndTime sync
// helper advances a lagging local EndTime up to Stripe's period end and bumps
// UpdatedAt.
func TestSyncSubscriptionEndTime_AdvancesWhenLagging(t *testing.T) {
	truncate(t)
	now := common.GetTimestamp()
	localEnd := now + 60
	stripePeriodEnd := now + 30 * 24 * 3600 // ~1 month ahead
	seedSyncSub(t, &model.UserSubscription{
		Id: 8610, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: localEnd,
		Status: model.SubscriptionStatusActive, UpdatedAt: now - 600,
	})

	before := common.GetTimestamp()
	require.NoError(t, syncSubscriptionEndTime(8610, stripePeriodEnd))

	sub := loadSyncSub(t, 8610)
	assert.Equal(t, stripePeriodEnd, sub.EndTime, "EndTime must advance to Stripe's period end")
	assert.GreaterOrEqual(t, sub.UpdatedAt, before)
}

// TestSyncSubscriptionEndTime_NoOpWhenLocalAtOrAhead verifies the EndTime sync
// helper is a no-op when the local EndTime already equals or exceeds Stripe's
// period end. This guards against the sync task regressing EndTime backward.
// The UserSubscription BeforeCreate hook overwrites UpdatedAt to now() on
// insert, so we capture the post-create UpdatedAt and compare against that.
func TestSyncSubscriptionEndTime_NoOpWhenLocalAtOrAhead(t *testing.T) {
	now := common.GetTimestamp()

	tests := []struct {
		name      string
		localEnd  int64
		stripeEnd int64
		wantEnd   int64
	}{
		{
			name:      "local equal to stripe: no-op",
			localEnd:  now + 30 * 24 * 3600,
			stripeEnd: now + 30 * 24 * 3600,
			wantEnd:   now + 30 * 24 * 3600,
		},
		{
			name:      "local ahead of stripe: no-op (no backward sync)",
			localEnd:  now + 60 * 24 * 3600,
			stripeEnd: now + 30 * 24 * 3600,
			wantEnd:   now + 60 * 24 * 3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncate(t)
			seedSyncSub(t, &model.UserSubscription{
				Id: 8611, UserId: 101, PlanId: 7001,
				StartTime: now - 3600, EndTime: tt.localEnd,
				Status: model.SubscriptionStatusActive,
			})

			preSub := loadSyncSub(t, 8611)
			preUpdatedAt := preSub.UpdatedAt

			require.NoError(t, syncSubscriptionEndTime(8611, tt.stripeEnd))

			postSub := loadSyncSub(t, 8611)
			assert.Equal(t, tt.wantEnd, postSub.EndTime, "EndTime must not regress")
			assert.Equal(t, preUpdatedAt, postSub.UpdatedAt, "UpdatedAt must not bump on no-op")
		})
	}
}

// TestSyncSubscriptionEndTime_OnMissingRow surfaces the underlying record-not-
// found error so the sync task can log and skip rather than silently no-op.
func TestSyncSubscriptionEndTime_OnMissingRow(t *testing.T) {
	truncate(t)
	err := syncSubscriptionEndTime(999999, common.GetTimestamp()+3600)
	require.Error(t, err)
}

// TestRunSubscriptionSyncOnce_SkipsWhenStripeUnconfigured verifies the
// defensive guard: when no Stripe API secret is configured, the sync task
// skips the Stripe API entirely (avoids auth errors / noise). A row that
// would otherwise be picked up must remain untouched.
//
// The UserSubscription BeforeCreate hook overwrites UpdatedAt to now() on
// insert, so we manually push updated_at back via raw SQL after insert
// (bypassing the BeforeUpdate hook) to make the row "stale" enough to be
// picked up by the sync query.
func TestRunSubscriptionSyncOnce_SkipsWhenStripeUnconfigured(t *testing.T) {
	truncate(t)
	now := common.GetTimestamp()
	// A row eligible for sync (active + stripe_subscription_id set + stale
	// updated_at). The task should load it but then skip the Stripe call.
	seedSyncSub(t, &model.UserSubscription{
		Id: 8620, UserId: 101, PlanId: 7001,
		StartTime: now - 7200, EndTime: now + 3600,
		Status:               model.SubscriptionStatusActive,
		StripeSubscriptionId: "sub_test_unconfigured",
	})

	// Push updated_at back beyond the stale cutoff (now-300s) so the sync
	// query selects this row. Raw SQL bypasses the BeforeUpdate hook that
	// would otherwise reset UpdatedAt to now().
	staleUpdatedAt := now - 7200
	require.NoError(t, model.DB.Exec("UPDATE user_subscriptions SET updated_at = ? WHERE id = ?", staleUpdatedAt, 8620).Error)

	preSub := loadSyncSub(t, 8620)
	require.Equal(t, staleUpdatedAt, preSub.UpdatedAt, "raw SQL must have set the stale updated_at")

	originalSecret := setting.StripeApiSecret
	setting.StripeApiSecret = ""
	t.Cleanup(func() { setting.StripeApiSecret = originalSecret })

	// Must not panic, must not error, must not touch the row.
	assert.NotPanics(t, func() { runSubscriptionSyncOnce() })

	postSub := loadSyncSub(t, 8620)
	assert.Equal(t, model.SubscriptionStatusActive, postSub.Status, "sync must skip when Stripe is unconfigured")
	assert.Equal(t, staleUpdatedAt, postSub.UpdatedAt, "sync must not bump UpdatedAt when skipped")
}

// TestRunSubscriptionSyncOnce_NoOpWhenNoEligibleRows verifies that the task
// exits cleanly when there are no subscriptions matching the sync query
// (no Stripe subscription id, or status out of scope). Nothing panics and
// nothing is logged to the test output as an error.
func TestRunSubscriptionSyncOnce_NoOpWhenNoEligibleRows(t *testing.T) {
	truncate(t)
	now := common.GetTimestamp()

	// Row without a Stripe subscription id — excluded by the sync query.
	seedSyncSub(t, &model.UserSubscription{
		Id: 8630, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: now + 3600,
		Status: model.SubscriptionStatusActive, UpdatedAt: now - 3600,
		StripeSubscriptionId: "",
	})
	// Row in a terminal status — excluded by the sync query's status filter.
	seedSyncSub(t, &model.UserSubscription{
		Id: 8631, UserId: 101, PlanId: 7001,
		StartTime: now - 7200, EndTime: now - 60,
		Status: model.SubscriptionStatusExpired, UpdatedAt: now - 60,
		StripeSubscriptionId: "sub_expired",
	})

	originalSecret := setting.StripeApiSecret
	setting.StripeApiSecret = "sk_test_configured"
	t.Cleanup(func() { setting.StripeApiSecret = originalSecret })

	assert.NotPanics(t, func() { runSubscriptionSyncOnce() })

	// Both rows must remain unchanged (sync query excluded them).
	activeRow := loadSyncSub(t, 8630)
	assert.Equal(t, model.SubscriptionStatusActive, activeRow.Status)
	assert.Empty(t, activeRow.StripeSubscriptionId, "row without stripe id must remain untouched")

	expiredRow := loadSyncSub(t, 8631)
	assert.Equal(t, model.SubscriptionStatusExpired, expiredRow.Status, "terminal-status row must remain untouched")
	assert.Equal(t, "sub_expired", expiredRow.StripeSubscriptionId)
}
