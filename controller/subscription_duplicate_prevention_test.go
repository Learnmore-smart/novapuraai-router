package controller

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupSubscriptionDuplicatePreventionTestDB initialises an in-memory SQLite
// DB with the tables the Phase 10 duplicate-prevention helpers touch
// (UserSubscription, SubscriptionOrder, SubscriptionPlan, TopUp, Log). Mirrors
// setupSubscriptionCouponTestDB, including the subscription plan cache purge
// (CompletePrepaidSubscriptionOrderOrExtend reads the plan via
// getSubscriptionPlanByIdTx, which caches; without the purge a plan cached
// under a previous test's DB can mask a different plan row with the same ID).
func setupSubscriptionDuplicatePreventionTestDB(t *testing.T) {
	t.Helper()
	initModelListColumnNames(t)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:sub-dup-prev-%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.RedisEnabled = originalRedisEnabled
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.SubscriptionPlan{},
		&model.SubscriptionOrder{},
		&model.UserSubscription{},
		&model.SubscriptionCouponRedemption{},
		&model.TopUp{},
		&model.Log{},
	))
	model.DB = db
	model.LOG_DB = db
	model.PurgeSubscriptionPlanCache()
}

// ---------------------------------------------------------------------------
// HasActiveAutoRenewSubscription — Phase 10 hardened duplicate-prevention guard
// ---------------------------------------------------------------------------

// TestHasActiveAutoRenewSubscription_NoSubscriptions verifies the baseline: a
// user with no subscriptions at all is allowed to start an auto-renew checkout.
func TestHasActiveAutoRenewSubscription_NoSubscriptions(t *testing.T) {
	setupSubscriptionDuplicatePreventionTestDB(t)

	user := &model.User{Username: "fresh-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	exists, err := model.HasActiveAutoRenewSubscription(user.Id)
	require.NoError(t, err)
	assert.False(t, exists, "user with no subscriptions must be allowed to subscribe")
}

// TestHasActiveAutoRenewSubscription_BlocksAcrossPlans verifies the core
// Phase 10 hardening invariant: a user who already has an active auto-renew
// subscription for plan A is BLOCKED from starting a new auto-renew checkout
// for plan B. The legacy HasActiveAutoRenewSubscriptionForPlan only checked
// the same plan_id, allowing users to stack auto-renew subscriptions across
// plans; HasActiveAutoRenewSubscription replaced it with a user-wide check.
func TestHasActiveAutoRenewSubscription_BlocksAcrossPlans(t *testing.T) {
	setupSubscriptionDuplicatePreventionTestDB(t)

	user := &model.User{Username: "multi-plan-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	planA := &model.SubscriptionPlan{Title: "Plan A", Enabled: true, DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, model.DB.Create(planA).Error)
	planB := &model.SubscriptionPlan{Title: "Plan B", Enabled: true, DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, model.DB.Create(planB).Error)

	// Active auto-renew subscription for plan A.
	subA := &model.UserSubscription{
		UserId:    user.Id,
		PlanId:    planA.Id,
		Status:    model.SubscriptionStatusActive,
		StartTime: common.GetTimestamp() - 86400,
		EndTime:   common.GetTimestamp() + 2592000,
	}
	require.NoError(t, model.DB.Create(subA).Error)

	// The user must be blocked from starting a new auto-renew checkout for
	// plan B (different plan) — this is the cross-plan block that the
	// per-plan check missed.
	exists, err := model.HasActiveAutoRenewSubscription(user.Id)
	require.NoError(t, err)
	assert.True(t, exists, "active auto-renew for plan A must block a new auto-renew for plan B")
}

// TestHasActiveAutoRenewSubscription_IncludesCancelingAndPastDue verifies
// that canceling and past_due subscriptions still block a new auto-renew
// checkout. Policy rationale (from HasActiveAutoRenewSubscription's doc
// comment): allowing a new auto-renew subscription while a previous one is
// still "canceling" can lead to double-charges if the user reactivates the
// old one or if a webhook race delays the cancellation. A past_due
// subscription is in a grace period and still an active entitlement.
func TestHasActiveAutoRenewSubscription_IncludesCancelingAndPastDue(t *testing.T) {
	setupSubscriptionDuplicatePreventionTestDB(t)

	plan := &model.SubscriptionPlan{Title: "Pro", Enabled: true, DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, model.DB.Create(plan).Error)

	// User 1: canceling subscription (cancel_at_period_end=true, still
	// entitled until period end).
	user1 := &model.User{Username: "canceling-user", Password: "password123", AffCode: "aff_canceling"}
	require.NoError(t, model.DB.Create(user1).Error)
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		UserId:            user1.Id,
		PlanId:            plan.Id,
		Status:            model.SubscriptionStatusCanceling,
		StartTime:         common.GetTimestamp() - 86400,
		EndTime:           common.GetTimestamp() + 2592000,
		CancelAtPeriodEnd: true,
	}).Error)
	exists, err := model.HasActiveAutoRenewSubscription(user1.Id)
	require.NoError(t, err)
	assert.True(t, exists, "canceling subscription must block a new auto-renew checkout")

	// User 2: past_due subscription (renewal charge failed, grace period).
	user2 := &model.User{Username: "pastdue-user", Password: "password123", AffCode: "aff_pastdue"}
	require.NoError(t, model.DB.Create(user2).Error)
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		UserId:    user2.Id,
		PlanId:    plan.Id,
		Status:    model.SubscriptionStatusPastDue,
		StartTime: common.GetTimestamp() - 86400,
		EndTime:   common.GetTimestamp() + 2592000,
	}).Error)
	exists, err = model.HasActiveAutoRenewSubscription(user2.Id)
	require.NoError(t, err)
	assert.True(t, exists, "past_due subscription must block a new auto-renew checkout")

	// User 3: only canceled / expired / payment_failed subscriptions — must
	// NOT block (these confer no current entitlement).
	user3 := &model.User{Username: "ended-user", Password: "password123", AffCode: "aff_ended"}
	require.NoError(t, model.DB.Create(user3).Error)
	for _, status := range []string{
		model.SubscriptionStatusCanceled,
		model.SubscriptionStatusExpired,
		model.SubscriptionStatusPaymentFailed,
	} {
		require.NoError(t, model.DB.Create(&model.UserSubscription{
			UserId:    user3.Id,
			PlanId:    plan.Id,
			Status:    status,
			StartTime: common.GetTimestamp() - 86400,
			EndTime:   common.GetTimestamp() - 3600,
		}).Error)
	}
	exists, err = model.HasActiveAutoRenewSubscription(user3.Id)
	require.NoError(t, err)
	assert.False(t, exists, "canceled/expired/payment_failed subscriptions must NOT block a new auto-renew checkout")
}

// ---------------------------------------------------------------------------
// FindRecentPendingSubscriptionOrder — rapid double-click dedup
// ---------------------------------------------------------------------------

// TestFindRecentPendingSubscriptionOrder_ReturnsRecentOrder verifies the
// dedup happy path: a pending order for the same user+plan+mode created
// within the dedup window is returned so the checkout handler can reuse its
// Stripe Checkout URL instead of creating a second session.
func TestFindRecentPendingSubscriptionOrder_ReturnsRecentOrder(t *testing.T) {
	setupSubscriptionDuplicatePreventionTestDB(t)

	user := &model.User{Username: "dedup-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)
	plan := &model.SubscriptionPlan{Title: "Pro", Enabled: true, DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, model.DB.Create(plan).Error)

	// A pending order created 10 seconds ago (well within the 60s window),
	// with a Stripe Checkout URL already persisted.
	require.NoError(t, model.DB.Create(&model.SubscriptionOrder{
		UserId:        user.Id,
		PlanId:        plan.Id,
		TradeNo:       "sub_ref_recent_1",
		Status:        common.TopUpStatusPending,
		Mode:          subscriptionCheckoutModeAutoRenew,
		CheckoutUrl:   "https://checkout.stripe.com/c/pay/cs_test_1",
		CreateTime:    common.GetTimestamp() - 10,
		PaymentMethod: model.PaymentMethodStripe,
	}).Error)

	order, err := model.FindRecentPendingSubscriptionOrder(user.Id, plan.Id, subscriptionCheckoutModeAutoRenew, 60)
	require.NoError(t, err)
	require.NotNil(t, order, "recent pending order within the window must be returned")
	assert.Equal(t, "sub_ref_recent_1", order.TradeNo)
	assert.Equal(t, "https://checkout.stripe.com/c/pay/cs_test_1", order.CheckoutUrl)
}

// TestFindRecentPendingSubscriptionOrder_FiltersByPlanModeAndAge verifies
// the dedup scoping: orders that don't match the user+plan+mode triple or
// that fall outside the dedup window are ignored. This protects the
// invariant that the dedup only catches genuine rapid double-clicks for the
// same checkout, not unrelated pending orders.
func TestFindRecentPendingSubscriptionOrder_FiltersByPlanModeAndAge(t *testing.T) {
	setupSubscriptionDuplicatePreventionTestDB(t)

	user := &model.User{Username: "scope-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)
	planA := &model.SubscriptionPlan{Title: "Plan A", Enabled: true, DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, model.DB.Create(planA).Error)
	planB := &model.SubscriptionPlan{Title: "Plan B", Enabled: true, DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, model.DB.Create(planB).Error)

	now := common.GetTimestamp()

	// (a) Old pending order for the SAME plan+mode but outside the window —
	// must be ignored (it's a stale abandoned checkout, not a double-click).
	require.NoError(t, model.DB.Create(&model.SubscriptionOrder{
		UserId:     user.Id,
		PlanId:     planA.Id,
		TradeNo:    "sub_ref_old",
		Status:     common.TopUpStatusPending,
		Mode:       subscriptionCheckoutModeAutoRenew,
		CreateTime: now - 120, // 120s ago, outside the 60s window
	}).Error)

	// (b) Pending order for a DIFFERENT plan — must be ignored.
	require.NoError(t, model.DB.Create(&model.SubscriptionOrder{
		UserId:     user.Id,
		PlanId:     planB.Id,
		TradeNo:    "sub_ref_other_plan",
		Status:     common.TopUpStatusPending,
		Mode:       subscriptionCheckoutModeAutoRenew,
		CreateTime: now - 5,
	}).Error)

	// (c) Pending order for the same plan but DIFFERENT mode — must be
	// ignored (a prepaid checkout and an auto-renew checkout are distinct
	// intents and should not dedup against each other).
	require.NoError(t, model.DB.Create(&model.SubscriptionOrder{
		UserId:     user.Id,
		PlanId:     planA.Id,
		TradeNo:    "sub_ref_other_mode",
		Status:     common.TopUpStatusPending,
		Mode:       subscriptionCheckoutModePrepaid,
		CreateTime: now - 5,
	}).Error)

	// (d) SUCCESS order for the same plan+mode within the window — must be
	// ignored (the dedup only catches PENDING orders; a success order means
	// the checkout already completed and a new one is legitimate).
	require.NoError(t, model.DB.Create(&model.SubscriptionOrder{
		UserId:     user.Id,
		PlanId:     planA.Id,
		TradeNo:    "sub_ref_success",
		Status:     common.TopUpStatusSuccess,
		Mode:       subscriptionCheckoutModeAutoRenew,
		CreateTime: now - 5,
	}).Error)

	// No matching pending order exists for planA + auto_renew within 60s.
	order, err := model.FindRecentPendingSubscriptionOrder(user.Id, planA.Id, subscriptionCheckoutModeAutoRenew, 60)
	require.NoError(t, err)
	assert.Nil(t, order, "old / other-plan / other-mode / success orders must all be ignored")
}

// ---------------------------------------------------------------------------
// CompletePrepaidSubscriptionOrderOrExtend — prepaid stacking & idempotency
// ---------------------------------------------------------------------------

// TestCompletePrepaidSubscriptionOrderOrExtend_StacksOnExistingPrepaid
// verifies the Phase 10 prepaid stacking invariant: completing a second
// prepaid order for the same plan EXTENDS the existing prepaid
// subscription's EndTime instead of creating a second UserSubscription row.
// This is what allows a user to "top up" their prepaid months without
// fragmenting their subscription history.
func TestCompletePrepaidSubscriptionOrderOrExtend_StacksOnExistingPrepaid(t *testing.T) {
	setupSubscriptionDuplicatePreventionTestDB(t)

	user := &model.User{Username: "stack-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)
	plan := &model.SubscriptionPlan{
		Title:         "Prepaid Pro",
		Enabled:       true,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	// First prepaid order: 3 months. Completes and creates a new
	// prepaid_active subscription.
	order1 := &model.SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "sub_ref_stack_1",
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
		PrepaidMonths:   3,
		Mode:            subscriptionCheckoutModePrepaid,
	}
	require.NoError(t, model.DB.Create(order1).Error)
	require.NoError(t, model.CompletePrepaidSubscriptionOrderOrExtend("sub_ref_stack_1", "{}", 3))

	var subs []model.UserSubscription
	require.NoError(t, model.DB.Where("user_id = ?", user.Id).Find(&subs).Error)
	require.Len(t, subs, 1, "first prepaid order must create exactly one subscription")
	firstEnd := subs[0].EndTime
	assert.Equal(t, model.SubscriptionStatusPrepaidActive, subs[0].Status)
	assert.Greater(t, firstEnd, subs[0].StartTime, "prepaid EndTime must be in the future")

	// Second prepaid order: 6 months. Must EXTEND the existing subscription
	// (not create a new row).
	order2 := &model.SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "sub_ref_stack_2",
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
		PrepaidMonths:   6,
		Mode:            subscriptionCheckoutModePrepaid,
	}
	require.NoError(t, model.DB.Create(order2).Error)
	require.NoError(t, model.CompletePrepaidSubscriptionOrderOrExtend("sub_ref_stack_2", "{}", 6))

	require.NoError(t, model.DB.Where("user_id = ?", user.Id).Find(&subs).Error)
	require.Len(t, subs, 1, "second prepaid order for the same plan must stack, not create a new row")

	// EndTime must have grown by ~6 months (6 * 30 * 86400 = 15552000
	// seconds, per the prepaidMonthSeconds constant). Allow a small delta
	// because the extension uses max(now, existing.EndTime) as the base, and
	// "now" may have advanced by a few seconds between the two calls.
	expectedExtension := int64(6 * 30 * 24 * 3600)
	assert.Greater(t, subs[0].EndTime, firstEnd, "stacked EndTime must be later than the original")
	assert.InDelta(t, float64(firstEnd+expectedExtension), float64(subs[0].EndTime), float64(60),
		"stacking must add ~6 prepaid months (6*30 days) to the existing EndTime")

	// Both orders must be marked Success (the second one too, even though it
	// stacked rather than creating a new row).
	var refreshedOrder1, refreshedOrder2 model.SubscriptionOrder
	require.NoError(t, model.DB.Where("trade_no = ?", "sub_ref_stack_1").First(&refreshedOrder1).Error)
	require.NoError(t, model.DB.Where("trade_no = ?", "sub_ref_stack_2").First(&refreshedOrder2).Error)
	assert.Equal(t, common.TopUpStatusSuccess, refreshedOrder1.Status)
	assert.Equal(t, common.TopUpStatusSuccess, refreshedOrder2.Status)
}

// TestCompletePrepaidSubscriptionOrderOrExtend_Idempotent verifies that
// replaying the same tradeNo (e.g. a duplicate webhook delivery) does NOT
// double-extend the EndTime. The function must detect the already-Success
// order and return nil without modifying the subscription.
func TestCompletePrepaidSubscriptionOrderOrExtend_Idempotent(t *testing.T) {
	setupSubscriptionDuplicatePreventionTestDB(t)

	user := &model.User{Username: "idem-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)
	plan := &model.SubscriptionPlan{
		Title:         "Idem Pro",
		Enabled:       true,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	order := &model.SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "sub_ref_idem_1",
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
		PrepaidMonths:   3,
		Mode:            subscriptionCheckoutModePrepaid,
	}
	require.NoError(t, model.DB.Create(order).Error)

	// First call: completes the order and creates the subscription.
	require.NoError(t, model.CompletePrepaidSubscriptionOrderOrExtend("sub_ref_idem_1", "{}", 3))

	var sub model.UserSubscription
	require.NoError(t, model.DB.Where("user_id = ?", user.Id).First(&sub).Error)
	firstEnd := sub.EndTime

	// Second call with the same tradeNo: must be a no-op (the order is
	// already Success). This protects against webhook replays double-stacking.
	require.NoError(t, model.CompletePrepaidSubscriptionOrderOrExtend("sub_ref_idem_1", "{}", 3),
		"replaying an already-Success order must return nil (idempotent)")

	// The EndTime must not have changed.
	require.NoError(t, model.DB.Where("user_id = ?", user.Id).First(&sub).Error)
	assert.Equal(t, firstEnd, sub.EndTime, "idempotent replay must not extend EndTime again")

	// Still exactly one subscription row.
	var count int64
	require.NoError(t, model.DB.Model(&model.UserSubscription{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.EqualValues(t, 1, count, "idempotent replay must not create a duplicate subscription")
}
