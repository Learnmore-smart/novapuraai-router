package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestIsValidSubscriptionStatus verifies that the seven canonical status
// values are recognized and that case-sensitivity / unknown values are
// rejected. This guards the contract that downstream phases (status
// refactors, lifecycle transitions) depend on.
func TestIsValidSubscriptionStatus(t *testing.T) {
	valid := []string{
		SubscriptionStatusActive,
		SubscriptionStatusPrepaidActive,
		SubscriptionStatusCanceling,
		SubscriptionStatusCanceled,
		SubscriptionStatusPastDue,
		SubscriptionStatusPaymentFailed,
		SubscriptionStatusExpired,
	}
	for _, s := range valid {
		assert.True(t, IsValidSubscriptionStatus(s), "expected %q to be valid", s)
	}

	invalid := []string{"", "unknown", "ACTIVE", "Active", "expired ", " expired", "pending", "trialing"}
	for _, s := range invalid {
		assert.False(t, IsValidSubscriptionStatus(s), "expected %q to be invalid", s)
	}
}

// TestGetPlanCoveredModels verifies that only enabled covered-model rows are
// returned and that disabled rows are filtered out.
func TestGetPlanCoveredModels(t *testing.T) {
	truncateTables(t)

	plan := &SubscriptionPlan{
		Id:            7001,
		Title:         "NovaPura-Pro",
		PriceAmount:   10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, DB.Create(plan).Error)

	now := GetDBTimestamp()
	rows := []SubscriptionPlanCoveredModel{
		{PlanId: plan.Id, ModelId: "gpt-4o", Enabled: true, CreatedAt: now},
		{PlanId: plan.Id, ModelId: "claude-3-5-sonnet", Enabled: true, CreatedAt: now},
		{PlanId: plan.Id, ModelId: "gemini-1.5-pro", Enabled: false, CreatedAt: now},
	}
	for i := range rows {
		require.NoError(t, DB.Create(&rows[i]).Error)
	}

	models, err := GetPlanCoveredModels(plan.Id)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"gpt-4o", "claude-3-5-sonnet"}, models)
}

// TestSetPlanCoveredModels verifies that the covered-model list is fully
// replaced on each call, and that duplicates / empty strings are ignored.
func TestSetPlanCoveredModels(t *testing.T) {
	truncateTables(t)

	plan := &SubscriptionPlan{
		Id:            7002,
		Title:         "NovaPura-Team",
		PriceAmount:   20,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, DB.Create(plan).Error)

	// First call: includes a duplicate and an empty string. Both must be
	// ignored, leaving exactly 2 distinct enabled rows.
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return SetPlanCoveredModels(tx, plan.Id, []string{"gpt-4o", "gpt-4o", "", "claude-3-5-sonnet"})
	}))

	var count int64
	require.NoError(t, DB.Model(&SubscriptionPlanCoveredModel{}).Where("plan_id = ?", plan.Id).Count(&count).Error)
	assert.EqualValues(t, 2, count)

	models, err := GetPlanCoveredModels(plan.Id)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"gpt-4o", "claude-3-5-sonnet"}, models)

	// Second call: a different set must replace the previous rows entirely.
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return SetPlanCoveredModels(tx, plan.Id, []string{"gemini-1.5-pro", "o1-preview"})
	}))

	require.NoError(t, DB.Model(&SubscriptionPlanCoveredModel{}).Where("plan_id = ?", plan.Id).Count(&count).Error)
	assert.EqualValues(t, 2, count)

	models, err = GetPlanCoveredModels(plan.Id)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"gemini-1.5-pro", "o1-preview"}, models)

	// The previously inserted rows must no longer exist (replacement, not append).
	var oldRows []SubscriptionPlanCoveredModel
	require.NoError(t, DB.Where("plan_id = ? AND model_id IN ?", plan.Id, []string{"gpt-4o", "claude-3-5-sonnet"}).Find(&oldRows).Error)
	assert.Empty(t, oldRows)
}

// TestValidateSubscriptionCoupon covers every validation branch via explicit
// DB state: not found, disabled, not-started, expired, usage-cap reached,
// per-user-limit reached, new-user-only violation, and the happy path.
func TestValidateSubscriptionCoupon(t *testing.T) {
	now := GetDBTimestamp()

	tests := []struct {
		name      string
		coupon    *SubscriptionCoupon
		setup     func(t *testing.T, couponId int64)
		code      string
		userId    int
		wantErrIs error
	}{
		{
			name:      "not found",
			code:      "NOPE",
			userId:    8001,
			wantErrIs: ErrSubscriptionCouponNotFound,
		},
		{
			name: "disabled",
			coupon: &SubscriptionCoupon{
				Code: "DISABLED", Name: "Disabled", StripeCouponId: "stripe_disabled",
				PercentOff: 10, DurationMonths: 1, Enabled: false,
			},
			code:      "DISABLED",
			userId:    8002,
			wantErrIs: ErrSubscriptionCouponDisabled,
		},
		{
			name: "not started",
			coupon: &SubscriptionCoupon{
				Code: "FUTURE", Name: "Future", StripeCouponId: "stripe_future",
				PercentOff: 10, DurationMonths: 1, Enabled: true, StartAt: now + 86400,
			},
			code:      "FUTURE",
			userId:    8003,
			wantErrIs: ErrSubscriptionCouponNotStarted,
		},
		{
			name: "expired",
			coupon: &SubscriptionCoupon{
				Code: "PAST", Name: "Past", StripeCouponId: "stripe_past",
				PercentOff: 10, DurationMonths: 1, Enabled: true, EndAt: now - 86400,
			},
			code:      "PAST",
			userId:    8004,
			wantErrIs: ErrSubscriptionCouponExpired,
		},
		{
			name: "usage cap reached",
			coupon: &SubscriptionCoupon{
				Code: "CAPPED", Name: "Capped", StripeCouponId: "stripe_capped",
				PercentOff: 10, DurationMonths: 1, Enabled: true,
				MaxRedemptions: 2, TimesRedeemed: 2,
			},
			code:      "CAPPED",
			userId:    8005,
			wantErrIs: ErrSubscriptionCouponUsageCapReached,
		},
		{
			name: "per-user limit reached",
			coupon: &SubscriptionCoupon{
				Code: "PERUSER", Name: "PerUser", StripeCouponId: "stripe_peruser",
				PercentOff: 10, DurationMonths: 1, Enabled: true, PerUserLimit: 1,
			},
			setup: func(t *testing.T, couponId int64) {
				require.NoError(t, DB.Create(&SubscriptionCouponRedemption{
					OrderId: "order-peruser-1", CouponId: couponId, UserId: 8006, PlanId: 0,
					Status: CouponRedemptionStatusReserved, PercentOff: 10,
					OriginalAmount: 1000, DiscountAmount: 100, FinalAmount: 900,
					Currency: "USD", CreatedAt: now, UpdatedAt: now,
				}).Error)
			},
			code:      "PERUSER",
			userId:    8006,
			wantErrIs: ErrSubscriptionCouponPerUserLimitReached,
		},
		{
			name: "new user only violation",
			coupon: &SubscriptionCoupon{
				Code: "NEWBIE", Name: "Newbie", StripeCouponId: "stripe_newbie",
				PercentOff: 10, DurationMonths: 1, Enabled: true, NewUserOnly: true,
			},
			setup: func(t *testing.T, couponId int64) {
				require.NoError(t, DB.Create(&UserSubscription{
					UserId: 8007, PlanId: 0, Status: SubscriptionStatusActive,
					StartTime: now - 3600, EndTime: now + 3600,
				}).Error)
			},
			code:      "NEWBIE",
			userId:    8007,
			wantErrIs: ErrSubscriptionCouponNewUserOnly,
		},
		{
			name: "valid coupon",
			coupon: &SubscriptionCoupon{
				Code: "VALID", Name: "Valid", StripeCouponId: "stripe_valid",
				PercentOff: 15, DurationMonths: 3, Enabled: true,
			},
			code:      "VALID",
			userId:    8008,
			wantErrIs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncateTables(t)

			if tt.coupon == nil {
				got, err := ValidateSubscriptionCoupon(tt.code, tt.userId)
				require.ErrorIs(t, err, tt.wantErrIs)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, DB.Create(tt.coupon).Error)
			if tt.setup != nil {
				tt.setup(t, tt.coupon.Id)
			}

			got, err := ValidateSubscriptionCoupon(tt.code, tt.userId)
			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.coupon.Id, got.Id)
			assert.Equal(t, tt.coupon.Code, got.Code)
			assert.Equal(t, tt.coupon.PercentOff, got.PercentOff)
			// Validation must NOT mutate TimesRedeemed or create a redemption.
			assert.Equal(t, 0, got.TimesRedeemed)
			var redemptionCount int64
			require.NoError(t, DB.Model(&SubscriptionCouponRedemption{}).Where("coupon_id = ?", tt.coupon.Id).Count(&redemptionCount).Error)
			assert.EqualValues(t, 0, redemptionCount)
		})
	}
}

// TestSubscriptionCouponRedemption_StatusConstants guards the redemption
// lifecycle vocabulary so later phases (reserve/issue/release/reverse) can
// rely on these exact string values.
func TestSubscriptionCouponRedemption_StatusConstants(t *testing.T) {
	assert.Equal(t, "reserved", CouponRedemptionStatusReserved)
	assert.Equal(t, "issued", CouponRedemptionStatusIssued)
	assert.Equal(t, "released", CouponRedemptionStatusReleased)
	assert.Equal(t, "reversed", CouponRedemptionStatusReversed)
}
