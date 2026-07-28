package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// getSubscriptionLifecycleSub loads a UserSubscription by id within lifecycle
// tests. Centralizing the load keeps the no-op / illegal-transition tests
// terse and makes the "capture preUpdatedAt, run op, reload, compare" pattern
// obvious.
func getSubscriptionLifecycleSub(t *testing.T, id int) UserSubscription {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", id).First(&sub).Error)
	return sub
}

// TestIsAllowedSubscriptionTransition covers the legal state-machine edges and
// confirms that terminal states (canceled / expired) have no outgoing
// transitions. old == new is always allowed (no-op).
func TestIsAllowedSubscriptionTransition(t *testing.T) {
	tests := []struct {
		name string
		old  string
		new  string
		want bool
	}{
		// No-op transitions are always allowed.
		{"active -> active (noop)", SubscriptionStatusActive, SubscriptionStatusActive, true},
		{"canceled -> canceled (noop)", SubscriptionStatusCanceled, SubscriptionStatusCanceled, true},
		{"expired -> expired (noop)", SubscriptionStatusExpired, SubscriptionStatusExpired, true},

		// Legal edges from active.
		{"active -> canceling", SubscriptionStatusActive, SubscriptionStatusCanceling, true},
		{"active -> past_due", SubscriptionStatusActive, SubscriptionStatusPastDue, true},
		{"active -> payment_failed", SubscriptionStatusActive, SubscriptionStatusPaymentFailed, true},
		{"active -> canceled", SubscriptionStatusActive, SubscriptionStatusCanceled, true},
		{"active -> expired", SubscriptionStatusActive, SubscriptionStatusExpired, true},

		// prepaid_active is restricted: only expired / canceled.
		{"prepaid_active -> expired", SubscriptionStatusPrepaidActive, SubscriptionStatusExpired, true},
		{"prepaid_active -> canceled", SubscriptionStatusPrepaidActive, SubscriptionStatusCanceled, true},
		{"prepaid_active -> active (illegal)", SubscriptionStatusPrepaidActive, SubscriptionStatusActive, false},
		{"prepaid_active -> canceling (illegal)", SubscriptionStatusPrepaidActive, SubscriptionStatusCanceling, false},

		// canceling can revert to active, or progress to terminal / past_due.
		{"canceling -> active (revival)", SubscriptionStatusCanceling, SubscriptionStatusActive, true},
		{"canceling -> canceled", SubscriptionStatusCanceling, SubscriptionStatusCanceled, true},
		{"canceling -> past_due", SubscriptionStatusCanceling, SubscriptionStatusPastDue, true},
		{"canceling -> expired", SubscriptionStatusCanceling, SubscriptionStatusExpired, true},
		{"canceling -> payment_failed (illegal)", SubscriptionStatusCanceling, SubscriptionStatusPaymentFailed, false},

		// past_due can recover or degrade.
		{"past_due -> active (recovery)", SubscriptionStatusPastDue, SubscriptionStatusActive, true},
		{"past_due -> payment_failed", SubscriptionStatusPastDue, SubscriptionStatusPaymentFailed, true},
		{"past_due -> canceled", SubscriptionStatusPastDue, SubscriptionStatusCanceled, true},
		{"past_due -> expired (illegal)", SubscriptionStatusPastDue, SubscriptionStatusExpired, false},

		// payment_failed can only terminate.
		{"payment_failed -> canceled", SubscriptionStatusPaymentFailed, SubscriptionStatusCanceled, true},
		{"payment_failed -> expired", SubscriptionStatusPaymentFailed, SubscriptionStatusExpired, true},
		{"payment_failed -> active (illegal)", SubscriptionStatusPaymentFailed, SubscriptionStatusActive, false},

		// Terminal states have no outgoing transitions.
		{"canceled -> active (terminal, illegal)", SubscriptionStatusCanceled, SubscriptionStatusActive, false},
		{"canceled -> expired (terminal, illegal)", SubscriptionStatusCanceled, SubscriptionStatusExpired, false},
		{"expired -> active (terminal, illegal)", SubscriptionStatusExpired, SubscriptionStatusActive, false},
		{"expired -> canceled (terminal, illegal)", SubscriptionStatusExpired, SubscriptionStatusCanceled, false},

		// Unknown source status has no edges.
		{"unknown -> active (illegal)", "unknown", SubscriptionStatusActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isAllowedSubscriptionTransition(tt.old, tt.new))
		})
	}
}

// TestTransitionSubscriptionStatus_ArgumentValidation guards the entry
// validation of TransitionSubscriptionStatus: nil tx, invalid subId, and
// unrecognized status strings are rejected before any DB load.
func TestTransitionSubscriptionStatus_ArgumentValidation(t *testing.T) {
	tests := []struct {
		name      string
		tx        *gorm.DB
		subId     int
		newStatus string
		wantErr   string
	}{
		{"nil tx rejected", nil, 1, SubscriptionStatusActive, "tx is nil"},
		{"non-positive subId rejected", DB, 0, SubscriptionStatusActive, "invalid subId"},
		{"negative subId rejected", DB, -5, SubscriptionStatusActive, "invalid subId"},
		{"unknown status rejected", DB, 1, "pending", "invalid subscription status"},
		{"empty status rejected", DB, 1, "", "invalid subscription status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TransitionSubscriptionStatus(tt.tx, tt.subId, tt.newStatus)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestTransitionSubscriptionStatus_NoOpOnSameStatus verifies that a transition
// to the current status is a no-op: it returns nil without writing or
// bumping UpdatedAt. The UserSubscription BeforeCreate hook overwrites
// UpdatedAt to now() on insert, so we capture the post-create UpdatedAt and
// compare against that.
func TestTransitionSubscriptionStatus_NoOpOnSameStatus(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	require.NoError(t, DB.Create(&UserSubscription{
		Id: 8501, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: now + 3600,
		Status: SubscriptionStatusActive,
	}).Error)

	preSub := getSubscriptionLifecycleSub(t, 8501)
	preUpdatedAt := preSub.UpdatedAt

	err := DB.Transaction(func(tx *gorm.DB) error {
		return TransitionSubscriptionStatus(tx, 8501, SubscriptionStatusActive)
	})
	require.NoError(t, err)

	postSub := getSubscriptionLifecycleSub(t, 8501)
	assert.Equal(t, SubscriptionStatusActive, postSub.Status)
	assert.Equal(t, preUpdatedAt, postSub.UpdatedAt, "no-op transition must not bump UpdatedAt")
}

// TestTransitionSubscriptionStatus_AppliesLegalTransitionAndBumpsUpdatedAt
// confirms that a legal transition updates Status and UpdatedAt atomically.
func TestTransitionSubscriptionStatus_AppliesLegalTransitionAndBumpsUpdatedAt(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	require.NoError(t, DB.Create(&UserSubscription{
		Id: 8502, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: now + 3600,
		Status: SubscriptionStatusActive, UpdatedAt: now - 600,
	}).Error)

	beforeTransition := GetDBTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		return TransitionSubscriptionStatus(tx, 8502, SubscriptionStatusCanceling)
	})
	require.NoError(t, err)

	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", 8502).First(&sub).Error)
	assert.Equal(t, SubscriptionStatusCanceling, sub.Status)
	assert.GreaterOrEqual(t, sub.UpdatedAt, beforeTransition)
}

// TestTransitionSubscriptionStatus_RejectsIllegalTransition ensures the state
// machine is enforced inside a transaction: a terminal-state row cannot move
// to active, and the original row is left untouched.
func TestTransitionSubscriptionStatus_RejectsIllegalTransition(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	require.NoError(t, DB.Create(&UserSubscription{
		Id: 8503, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: now - 60,
		Status: SubscriptionStatusCanceled,
	}).Error)

	preSub := getSubscriptionLifecycleSub(t, 8503)
	preUpdatedAt := preSub.UpdatedAt

	err := DB.Transaction(func(tx *gorm.DB) error {
		return TransitionSubscriptionStatus(tx, 8503, SubscriptionStatusActive)
	})
	require.ErrorIs(t, err, ErrInvalidSubscriptionStatusTransition)

	postSub := getSubscriptionLifecycleSub(t, 8503)
	assert.Equal(t, SubscriptionStatusCanceled, postSub.Status, "illegal transition must not mutate status")
	assert.Equal(t, preUpdatedAt, postSub.UpdatedAt, "illegal transition must not bump UpdatedAt")
}

// TestTransitionSubscriptionStatus_OnMissingRow surfaces the underlying
// gorm.ErrRecordNotFound so callers can distinguish "no such subscription"
// from a state-machine violation.
func TestTransitionSubscriptionStatus_OnMissingRow(t *testing.T) {
	truncateTables(t)

	err := DB.Transaction(func(tx *gorm.DB) error {
		return TransitionSubscriptionStatus(tx, 999999, SubscriptionStatusActive)
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestExtendPrepaidSubscriptionEndTime_ArgumentValidation guards the entry
// validation: invalid subId and non-positive additionalMonths are rejected
// without touching the DB.
func TestExtendPrepaidSubscriptionEndTime_ArgumentValidation(t *testing.T) {
	tests := []struct {
		name             string
		subId            int
		additionalMonths int
		wantErr          string
	}{
		{"zero subId rejected", 0, 3, "invalid subId"},
		{"negative subId rejected", -1, 3, "invalid subId"},
		{"zero months rejected", 8501, 0, "additionalMonths must be > 0"},
		{"negative months rejected", 8501, -2, "additionalMonths must be > 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExtendPrepaidSubscriptionEndTime(tt.subId, tt.additionalMonths)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestExtendPrepaidSubscriptionEndTime_RejectsNonPrepaid verifies that only
// prepaid_active subscriptions can be extended. Active (auto-renew) and
// terminal-state rows are rejected with ErrSubscriptionNotPrepaid.
func TestExtendPrepaidSubscriptionEndTime_RejectsNonPrepaid(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	require.NoError(t, DB.Create(&UserSubscription{
		Id: 8510, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: now + 3600,
		Status: SubscriptionStatusActive,
	}).Error)

	err := ExtendPrepaidSubscriptionEndTime(8510, 3)
	require.ErrorIs(t, err, ErrSubscriptionNotPrepaid)

	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", 8510).First(&sub).Error)
	assert.Equal(t, now+3600, sub.EndTime, "non-prepaid rejection must not change EndTime")
}

// TestExtendPrepaidSubscriptionEndTime_StacksFromCurrentEndTime verifies the
// core stacking invariant: when the prepaid subscription is still live
// (EndTime > now), additional months are added to EndTime, not to now. This
// is what makes prepaid top-up "stacking" work — buying 3 more months on a
// subscription with 2 months left yields 5 months remaining, not 3.
func TestExtendPrepaidSubscriptionEndTime_StacksFromCurrentEndTime(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	// EndTime is 1 month (30 days) in the future — still active.
	originalEnd := now + prepaidMonthSeconds
	require.NoError(t, DB.Create(&UserSubscription{
		Id: 8520, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: originalEnd,
		Status: SubscriptionStatusPrepaidActive,
	}).Error)

	require.NoError(t, ExtendPrepaidSubscriptionEndTime(8520, 3))

	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", 8520).First(&sub).Error)
	assert.Equal(t, originalEnd+3*prepaidMonthSeconds, sub.EndTime,
		"active prepaid sub must stack from current EndTime")
	assert.Equal(t, SubscriptionStatusPrepaidActive, sub.Status, "status must remain prepaid_active")
}

// TestExtendPrepaidSubscriptionEndTime_StacksFromNowWhenExpired verifies the
// race-condition branch: a row still marked prepaid_active whose EndTime is
// already in the past extends from now (not from the stale EndTime), so the
// user gets the full additionalMonths of new service.
func TestExtendPrepaidSubscriptionEndTime_StacksFromNowWhenExpired(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	// EndTime is in the past but Status is still prepaid_active (race).
	staleEnd := now - 86400
	require.NoError(t, DB.Create(&UserSubscription{
		Id: 8521, UserId: 101, PlanId: 7001,
		StartTime: now - 2*86400, EndTime: staleEnd,
		Status: SubscriptionStatusPrepaidActive,
	}).Error)

	beforeExtend := GetDBTimestamp()
	require.NoError(t, ExtendPrepaidSubscriptionEndTime(8521, 1))

	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", 8521).First(&sub).Error)
	// New EndTime must be roughly now + 1 month. Allow slack for the test
	// clock advancing between beforeExtend and the UPDATE.
	expectedEarliest := beforeExtend + prepaidMonthSeconds
	assert.GreaterOrEqual(t, sub.EndTime, expectedEarliest,
		"expired prepaid_active row must stack from now, not from stale EndTime")
	assert.Less(t, sub.EndTime, expectedEarliest+60,
		"new EndTime must be within a small clock slack window of now+1month")
}

// TestExtendPrepaidSubscriptionEndTime_StacksRepeatedly verifies that calling
// ExtendPrepaidSubscriptionEndTime multiple times accumulates months on top of
// the previous EndTime, not on top of the original now.
func TestExtendPrepaidSubscriptionEndTime_StacksRepeatedly(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	originalEnd := now + prepaidMonthSeconds
	require.NoError(t, DB.Create(&UserSubscription{
		Id: 8522, UserId: 101, PlanId: 7001,
		StartTime: now - 3600, EndTime: originalEnd,
		Status: SubscriptionStatusPrepaidActive,
	}).Error)

	require.NoError(t, ExtendPrepaidSubscriptionEndTime(8522, 1))
	require.NoError(t, ExtendPrepaidSubscriptionEndTime(8522, 6))

	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", 8522).First(&sub).Error)
	assert.Equal(t, originalEnd+7*prepaidMonthSeconds, sub.EndTime,
		"two extensions must accumulate to 7 months on top of the original EndTime")
}
