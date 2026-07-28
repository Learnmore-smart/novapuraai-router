package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCoverageTestFixtures creates a plan with covered models and a user
// subscription, returning the plan and subscription for assertions. The cache
// is purged at entry so prior tests do not leak cached coverage results.
func setupCoverageTestFixtures(t *testing.T, userId int, planId int, status string, endOffset int64, coveredModels []string) (*SubscriptionPlan, *UserSubscription) {
	t.Helper()
	InvalidatePlanCoveredModelsCache()

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:            planId,
		Title:         "NovaPura-Coverage-Test",
		PriceAmount:   10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
	}
	require.NoError(t, DB.Create(plan).Error)

	for _, m := range coveredModels {
		require.NoError(t, DB.Create(&SubscriptionPlanCoveredModel{
			PlanId:    plan.Id,
			ModelId:   m,
			Enabled:   true,
			CreatedAt: now,
		}).Error)
	}

	sub := &UserSubscription{
		UserId:    userId,
		PlanId:    plan.Id,
		Status:    status,
		StartTime: now - 3600,
		EndTime:   now + endOffset,
	}
	require.NoError(t, DB.Create(sub).Error)
	return plan, sub
}

// TestUserHasSubscriptionCoveringModel_Covered verifies that a user with an
// active subscription whose plan covers the requested model is reported as
// covered, and the returned subscription/plan match the DB rows.
func TestUserHasSubscriptionCoveringModel_Covered(t *testing.T) {
	truncateTables(t)

	plan, sub := setupCoverageTestFixtures(t, 9001, 7101, SubscriptionStatusActive, 86400, []string{"gpt-4o", "claude-3-5-sonnet"})

	covered, gotSub, gotPlan, err := UserHasSubscriptionCoveringModel(9001, "gpt-4o")
	require.NoError(t, err)
	assert.True(t, covered)
	require.NotNil(t, gotSub)
	require.NotNil(t, gotPlan)
	assert.Equal(t, sub.Id, gotSub.Id)
	assert.Equal(t, plan.Id, gotPlan.Id)
	assert.Equal(t, plan.Title, gotPlan.Title)
}

// TestUserHasSubscriptionCoveringModel_NotCoveredModel verifies that a model
// not in the plan's covered list returns false even when the user has an
// active subscription.
func TestUserHasSubscriptionCoveringModel_NotCoveredModel(t *testing.T) {
	truncateTables(t)

	setupCoverageTestFixtures(t, 9002, 7102, SubscriptionStatusActive, 86400, []string{"gpt-4o"})

	covered, gotSub, gotPlan, err := UserHasSubscriptionCoveringModel(9002, "gpt-3.5-turbo")
	require.NoError(t, err)
	assert.False(t, covered)
	assert.Nil(t, gotSub)
	assert.Nil(t, gotPlan)
}

// TestUserHasSubscriptionCoveringModel_NoActiveSubscription verifies that
// expired/canceled subscriptions do not provide coverage.
func TestUserHasSubscriptionCoveringModel_NoActiveSubscription(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()

	// Expired subscription (end_time in the past)
	plan := &SubscriptionPlan{
		Id:            7103,
		Title:         "Expired-Plan",
		PriceAmount:   10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
	}
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, DB.Create(&SubscriptionPlanCoveredModel{
		PlanId: plan.Id, ModelId: "gpt-4o", Enabled: true, CreatedAt: now,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		UserId: 9003, PlanId: plan.Id, Status: SubscriptionStatusActive,
		StartTime: now - 7200, EndTime: now - 3600,
	}).Error)
	InvalidatePlanCoveredModelsCache()

	covered, _, _, err := UserHasSubscriptionCoveringModel(9003, "gpt-4o")
	require.NoError(t, err)
	assert.False(t, covered)

	// Canceled status subscription
	truncateTables(t)
	setupCoverageTestFixtures(t, 9004, 7104, SubscriptionStatusCanceled, 86400, []string{"gpt-4o"})
	covered, _, _, err = UserHasSubscriptionCoveringModel(9004, "gpt-4o")
	require.NoError(t, err)
	assert.False(t, covered)
}

// TestUserHasSubscriptionCoveringModel_DisabledCoveredModel verifies that a
// disabled covered-model row does not provide coverage.
func TestUserHasSubscriptionCoveringModel_DisabledCoveredModel(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:            7105,
		Title:         "Disabled-Coverage-Plan",
		PriceAmount:   10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
	}
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, DB.Create(&SubscriptionPlanCoveredModel{
		PlanId: plan.Id, ModelId: "gpt-4o", Enabled: false, CreatedAt: now,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		UserId: 9005, PlanId: plan.Id, Status: SubscriptionStatusActive,
		StartTime: now - 3600, EndTime: now + 3600,
	}).Error)
	InvalidatePlanCoveredModelsCache()

	covered, _, _, err := UserHasSubscriptionCoveringModel(9005, "gpt-4o")
	require.NoError(t, err)
	assert.False(t, covered)
}

// TestUserHasSubscriptionCoveringModel_EarliestExpiringFirst verifies the
// selection policy: among multiple active subscriptions whose plans cover the
// model, the one with the earliest end_time is returned.
func TestUserHasSubscriptionCoveringModel_EarliestExpiringFirst(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()

	// Plan A covers gpt-4o, subscription expires in 1 hour
	planA := &SubscriptionPlan{Id: 7106, Title: "Plan-A", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth, DurationValue: 1, Enabled: true}
	require.NoError(t, DB.Create(planA).Error)
	require.NoError(t, DB.Create(&SubscriptionPlanCoveredModel{PlanId: planA.Id, ModelId: "gpt-4o", Enabled: true, CreatedAt: now}).Error)
	require.NoError(t, DB.Create(&UserSubscription{UserId: 9006, PlanId: planA.Id, Status: SubscriptionStatusActive, StartTime: now - 3600, EndTime: now + 3600}).Error)

	// Plan B also covers gpt-4o, subscription expires in 7 days (later)
	planB := &SubscriptionPlan{Id: 7107, Title: "Plan-B", PriceAmount: 20, DurationUnit: SubscriptionDurationMonth, DurationValue: 1, Enabled: true}
	require.NoError(t, DB.Create(planB).Error)
	require.NoError(t, DB.Create(&SubscriptionPlanCoveredModel{PlanId: planB.Id, ModelId: "gpt-4o", Enabled: true, CreatedAt: now}).Error)
	require.NoError(t, DB.Create(&UserSubscription{UserId: 9006, PlanId: planB.Id, Status: SubscriptionStatusActive, StartTime: now - 3600, EndTime: now + 7*86400}).Error)

	InvalidatePlanCoveredModelsCache()

	covered, gotSub, gotPlan, err := UserHasSubscriptionCoveringModel(9006, "gpt-4o")
	require.NoError(t, err)
	assert.True(t, covered)
	require.NotNil(t, gotSub)
	require.NotNil(t, gotPlan)
	// The earliest-expiring subscription (Plan A, +1h) must be selected.
	assert.Equal(t, planA.Id, gotPlan.Id, "earliest-expiring plan should be selected")
	assert.Equal(t, "Plan-A", gotPlan.Title)
}

// TestUserHasSubscriptionCoveringModel_CacheHit verifies that a second lookup
// for the same (user, model) returns the cached result without re-querying.
// We verify caching by checking that DB changes after the first lookup are
// not reflected in the second (cached) lookup.
func TestUserHasSubscriptionCoveringModel_CacheHit(t *testing.T) {
	truncateTables(t)

	plan, _ := setupCoverageTestFixtures(t, 9007, 7108, SubscriptionStatusActive, 86400, []string{"gpt-4o"})

	// First lookup populates the cache.
	covered, _, gotPlan, err := UserHasSubscriptionCoveringModel(9007, "gpt-4o")
	require.NoError(t, err)
	assert.True(t, covered)
	require.NotNil(t, gotPlan)
	assert.Equal(t, "NovaPura-Coverage-Test", gotPlan.Title)

	// Mutate the plan title in the DB after the cache is populated. A cached
	// lookup should still return the old title (cache hit, not DB re-query).
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Update("title", "Mutated-Title").Error)

	covered2, _, gotPlan2, err := UserHasSubscriptionCoveringModel(9007, "gpt-4o")
	require.NoError(t, err)
	assert.True(t, covered2)
	require.NotNil(t, gotPlan2)
	assert.Equal(t, "NovaPura-Coverage-Test", gotPlan2.Title, "cached plan title should not reflect DB mutation")
}

// TestInvalidateUserSubscriptionCoverageCache verifies that after invalidating
// a user's coverage cache, the next lookup re-queries the DB and reflects
// current state.
func TestInvalidateUserSubscriptionCoverageCache(t *testing.T) {
	truncateTables(t)

	plan, _ := setupCoverageTestFixtures(t, 9008, 7109, SubscriptionStatusActive, 86400, []string{"gpt-4o"})

	// Populate cache.
	covered, _, gotPlan, _ := UserHasSubscriptionCoveringModel(9008, "gpt-4o")
	require.True(t, covered)
	require.NotNil(t, gotPlan)
	assert.Equal(t, "NovaPura-Coverage-Test", gotPlan.Title)

	// Mutate plan title.
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Update("title", "Updated-Title").Error)
	// InvalidateSubscriptionPlanCache so GetSubscriptionPlanById re-reads.
	InvalidateSubscriptionPlanCache(plan.Id)
	// Invalidate the coverage cache for this user.
	InvalidateUserSubscriptionCoverageCache(9008)

	// Next lookup must re-query and reflect the updated title.
	covered2, _, gotPlan2, err := UserHasSubscriptionCoveringModel(9008, "gpt-4o")
	require.NoError(t, err)
	assert.True(t, covered2)
	require.NotNil(t, gotPlan2)
	assert.Equal(t, "Updated-Title", gotPlan2.Title)
}

// TestUserHasSubscriptionCoveringModel_InvalidInputs verifies edge cases:
// userId <= 0 and empty model name return false without error.
func TestUserHasSubscriptionCoveringModel_InvalidInputs(t *testing.T) {
	covered, gotSub, gotPlan, err := UserHasSubscriptionCoveringModel(0, "gpt-4o")
	require.NoError(t, err)
	assert.False(t, covered)
	assert.Nil(t, gotSub)
	assert.Nil(t, gotPlan)

	covered, gotSub, gotPlan, err = UserHasSubscriptionCoveringModel(-1, "gpt-4o")
	require.NoError(t, err)
	assert.False(t, covered)
	assert.Nil(t, gotSub)
	assert.Nil(t, gotPlan)

	covered, gotSub, gotPlan, err = UserHasSubscriptionCoveringModel(9999, "")
	require.NoError(t, err)
	assert.False(t, covered)
	assert.Nil(t, gotSub)
	assert.Nil(t, gotPlan)

	covered, gotSub, gotPlan, err = UserHasSubscriptionCoveringModel(9999, "   ")
	require.NoError(t, err)
	assert.False(t, covered)
	assert.Nil(t, gotSub)
	assert.Nil(t, gotPlan)
}

// TestInvalidateUserSubscriptionCoverageCache_NopForZeroUserId verifies that
// calling with userId <= 0 is a safe no-op (does not panic or purge the
// entire cache).
func TestInvalidateUserSubscriptionCoverageCache_NopForZeroUserId(t *testing.T) {
	// Should not panic.
	InvalidateUserSubscriptionCoverageCache(0)
	InvalidateUserSubscriptionCoverageCache(-1)
}
