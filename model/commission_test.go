package model

import (
	"fmt"
	"math"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var commissionTestSequence atomic.Int64

func setupCommissionTest(t *testing.T) {
	t.Helper()
	truncateTables(t)
	// CampaignClaim/CampaignCounter are needed by TrySettleDelayedInviteReward tests.
	require.NoError(t, DB.AutoMigrate(&CampaignClaim{}, &CampaignCounter{}))

	originalRate := common.AffCommissionRate
	originalFreezeDays := common.CommissionFreezeDays
	originalMinWithdrawal := common.MinWithdrawalCents
	originalExchangeRate := operation_setting.USDExchangeRate
	originalDelayedInvite := common.DelayedInviteReward
	originalInviteCNY := common.InviteRewardCNYYuan
	originalMaxInvites := common.MaxValidInvites

	common.AffCommissionRate = 0.25
	common.CommissionFreezeDays = 14
	common.MinWithdrawalCents = 1000
	operation_setting.USDExchangeRate = common.DefaultUSDExchangeRate

	t.Cleanup(func() {
		common.AffCommissionRate = originalRate
		common.CommissionFreezeDays = originalFreezeDays
		common.MinWithdrawalCents = originalMinWithdrawal
		operation_setting.USDExchangeRate = originalExchangeRate
		common.DelayedInviteReward = originalDelayedInvite
		common.InviteRewardCNYYuan = originalInviteCNY
		common.MaxValidInvites = originalMaxInvites
	})
}

func createCommissionTestUser(t *testing.T, configure func(*User)) *User {
	t.Helper()
	sequence := commissionTestSequence.Add(1)
	user := &User{
		Username: fmt.Sprintf("comm_%d", sequence),
		Password: "test-password",
		AffCode:  fmt.Sprintf("comm_aff_%d", sequence),
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if configure != nil {
		configure(user)
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

// --- ConvertAmountToUSDCents ---

func TestConvertAmountToUSDCents(t *testing.T) {
	// ConvertAmountToUSDCents reads setting.EffectiveUSDCNYRate, which prefers
	// the live billing-currency CNY rate over the legacy fallback. Set the
	// billing-currency rate directly so the test controls the conversion.
	originalConfig := setting.GetBillingCurrencyConfig()
	require.NoError(t, setting.SetBillingCurrencyFXRate(setting.BillingCurrencyCNY, 7.0))
	t.Cleanup(func() {
		data, _ := common.Marshal(originalConfig)
		_ = setting.UpdateBillingCurrencyConfigByJSON(string(data))
	})

	tests := []struct {
		name     string
		amount   float64
		currency string
		expected int64
	}{
		{"usd pass-through", 10.0, "USD", 1000},
		{"usd empty currency defaults to usd", 5.0, "", 500},
		{"cny converts via rate", 70.0, "CNY", 1000}, // 70/7 = 10 USD = 1000 cents
		{"cny lowercase", 70.0, "cny", 1000},
		{"zero amount", 0.0, "USD", 0},
		{"negative amount", -10.0, "USD", 0},
		{"nan amount", math.NaN(), "USD", 0},
		{"inf amount", math.Inf(1), "USD", 0},
		{"unsupported currency", 10.0, "EUR", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ConvertAmountToUSDCents(tc.amount, tc.currency)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvertAmountToUSDCentsUnsupportedCurrencySafeFails(t *testing.T) {
	// Unsupported currency must safe-fail to 0, never panic or produce negative.
	assert.Equal(t, int64(0), ConvertAmountToUSDCents(100.0, "EUR"))
	assert.Equal(t, int64(0), ConvertAmountToUSDCents(100.0, "JPY"))
}

// --- SettleRechargeCommission ---

func TestSettleRechargeCommission_ApprovedInviterGetsCommission(t *testing.T) {
	setupCommissionTest(t)

	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionTestUser(t, func(u *User) {
		u.InviterId = inviter.Id
	})

	// $20.00 paid = 2000 cents. 25% commission = 500 cents.
	settlement, err := SettleRechargeCommission(DB, invitee.Id, "topup-test-1", 2000, "USD")
	require.NoError(t, err)
	require.NotNil(t, settlement)
	assert.Equal(t, inviter.Id, settlement.InviterId)
	assert.Equal(t, int64(500), settlement.CommissionCents)
	assert.Positive(t, settlement.AvailableAt)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, inviter.Id).Error)
	assert.Equal(t, int64(500), refreshed.PendingCommissionCents)
	assert.Equal(t, int64(500), refreshed.CommissionTotalCents)
	assert.Equal(t, int64(0), refreshed.CommissionBalanceCents) // still frozen

	var commissionCount int64
	require.NoError(t, DB.Model(&Commission{}).Where("topup_id = ?", "topup-test-1").Count(&commissionCount).Error)
	assert.EqualValues(t, 1, commissionCount)
}

func TestSettleRechargeCommission_NonApprovedInviterGetsNothing(t *testing.T) {
	setupCommissionTest(t)

	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = false
	})
	invitee := createCommissionTestUser(t, func(u *User) {
		u.InviterId = inviter.Id
	})

	settlement, err := SettleRechargeCommission(DB, invitee.Id, "topup-test-2", 2000, "USD")
	require.NoError(t, err)
	assert.Nil(t, settlement)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, inviter.Id).Error)
	assert.Zero(t, refreshed.PendingCommissionCents)
	assert.Zero(t, refreshed.CommissionTotalCents)

	var commissionCount int64
	require.NoError(t, DB.Model(&Commission{}).Where("topup_id = ?", "topup-test-2").Count(&commissionCount).Error)
	assert.Zero(t, commissionCount)
}

func TestSettleRechargeCommission_IsIdempotent(t *testing.T) {
	setupCommissionTest(t)

	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionTestUser(t, func(u *User) {
		u.InviterId = inviter.Id
	})

	// First call credits commission.
	s1, err := SettleRechargeCommission(DB, invitee.Id, "topup-idem-1", 2000, "USD")
	require.NoError(t, err)
	require.NotNil(t, s1)
	assert.Equal(t, int64(500), s1.CommissionCents)

	// Second call with same topup_id is a no-op.
	s2, err := SettleRechargeCommission(DB, invitee.Id, "topup-idem-1", 2000, "USD")
	require.NoError(t, err)
	assert.Nil(t, s2)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, inviter.Id).Error)
	assert.Equal(t, int64(500), refreshed.PendingCommissionCents) // not doubled

	var commissionCount int64
	require.NoError(t, DB.Model(&Commission{}).Where("topup_id = ?", "topup-idem-1").Count(&commissionCount).Error)
	assert.EqualValues(t, 1, commissionCount)
}

func TestSettleRechargeCommission_NoInviterOrSelfInviteSkips(t *testing.T) {
	setupCommissionTest(t)

	// No inviter.
	invitee := createCommissionTestUser(t, nil)
	s, err := SettleRechargeCommission(DB, invitee.Id, "topup-noinv-1", 2000, "USD")
	require.NoError(t, err)
	assert.Nil(t, s)

	// Self-invite.
	selfInviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
		u.InviterId = u.Id // will be set after create
	})
	require.NoError(t, DB.Model(&User{}).Where("id = ?", selfInviter.Id).Update("inviter_id", selfInviter.Id).Error)
	s2, err := SettleRechargeCommission(DB, selfInviter.Id, "topup-self-1", 2000, "USD")
	require.NoError(t, err)
	assert.Nil(t, s2)
}

func TestSettleRechargeCommission_ZeroPaidAmountSkips(t *testing.T) {
	setupCommissionTest(t)

	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionTestUser(t, func(u *User) {
		u.InviterId = inviter.Id
	})

	s, err := SettleRechargeCommission(DB, invitee.Id, "topup-zero-1", 0, "USD")
	require.NoError(t, err)
	assert.Nil(t, s)
}

// --- ReleaseMaturedCommissions ---

func TestReleaseMaturedCommissions_ReleasesPastHold(t *testing.T) {
	setupCommissionTest(t)
	common.CommissionFreezeDays = 14

	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionTestUser(t, func(u *User) {
		u.InviterId = inviter.Id
	})

	// Create a commission whose AvailableAt is in the past.
	pastTimestamp := 1 // year 1970 — definitely past
	require.NoError(t, DB.Create(&Commission{
		InviterId:       inviter.Id,
		InviteeId:       invitee.Id,
		TopUpId:         "topup-release-1",
		PaidAmountCents: 2000,
		PaidCurrency:    "USD",
		Rate:            0.25,
		CommissionCents: 500,
		Status:          CommissionStatusPending,
		AvailableAt:     int64(pastTimestamp),
	}).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).
		Updates(map[string]any{
			"pending_commission_cents": 500,
			"commission_total_cents":   500,
		}).Error)

	released, err := ReleaseMaturedCommissions()
	require.NoError(t, err)
	assert.Equal(t, 1, released)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, inviter.Id).Error)
	assert.Zero(t, refreshed.PendingCommissionCents)    // moved out
	assert.Equal(t, int64(500), refreshed.CommissionBalanceCents) // moved in
	assert.Equal(t, int64(500), refreshed.CommissionTotalCents)   // unchanged

	var comm Commission
	require.NoError(t, DB.Where("topup_id = ?", "topup-release-1").First(&comm).Error)
	assert.Equal(t, CommissionStatusAvailable, comm.Status)
}

func TestReleaseMaturedCommissions_KeepsFutureHold(t *testing.T) {
	setupCommissionTest(t)

	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
	})

	// AvailableAt far in the future.
	futureTimestamp := int64(9999999999)
	require.NoError(t, DB.Create(&Commission{
		InviterId:       inviter.Id,
		TopUpId:         "topup-future-1",
		PaidAmountCents: 2000,
		PaidCurrency:    "USD",
		Rate:            0.25,
		CommissionCents: 500,
		Status:          CommissionStatusPending,
		AvailableAt:     futureTimestamp,
	}).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).
		Update("pending_commission_cents", 500).Error)

	released, err := ReleaseMaturedCommissions()
	require.NoError(t, err)
	assert.Zero(t, released)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, inviter.Id).Error)
	assert.Equal(t, int64(500), refreshed.PendingCommissionCents)
	assert.Zero(t, refreshed.CommissionBalanceCents)
}

// --- CreateWithdrawalRequest ---

func TestCreateWithdrawalRequest_DebitsBalance(t *testing.T) {
	setupCommissionTest(t)

	user := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
		u.CommissionBalanceCents = 5000 // $50.00
	})

	req, err := CreateWithdrawalRequest(user.Id, 2000) // withdraw $20.00
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, int64(2000), req.AmountCents)
	assert.Equal(t, WithdrawalStatusPending, req.Status)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, int64(3000), refreshed.CommissionBalanceCents) // 5000 - 2000
}

func TestCreateWithdrawalRequest_InsufficientBalanceFails(t *testing.T) {
	setupCommissionTest(t)

	user := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
		u.CommissionBalanceCents = 500 // $5.00
	})

	_, err := CreateWithdrawalRequest(user.Id, 1000) // try $10.00
	require.Error(t, err)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, int64(500), refreshed.CommissionBalanceCents) // unchanged
}

func TestCreateWithdrawalRequest_BelowMinimumFails(t *testing.T) {
	setupCommissionTest(t)
	common.MinWithdrawalCents = 1000

	user := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
		u.CommissionBalanceCents = 5000
	})

	_, err := CreateWithdrawalRequest(user.Id, 500) // below $10 min
	require.Error(t, err)
}

func TestCreateWithdrawalRequest_NonApprovedUserFails(t *testing.T) {
	setupCommissionTest(t)

	user := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = false
		u.CommissionBalanceCents = 5000
	})

	_, err := CreateWithdrawalRequest(user.Id, 2000)
	require.Error(t, err)
}

// --- AdminProcessWithdrawal ---

func TestAdminProcessWithdrawal_PaidKeepsBalance(t *testing.T) {
	setupCommissionTest(t)

	user := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
		u.CommissionBalanceCents = 5000
	})

	req, err := CreateWithdrawalRequest(user.Id, 2000)
	require.NoError(t, err)
	require.NotNil(t, req)

	// Balance was debited at request time.
	var afterRequest User
	require.NoError(t, DB.First(&afterRequest, user.Id).Error)
	assert.Equal(t, int64(3000), afterRequest.CommissionBalanceCents)

	result, err := AdminProcessWithdrawal(req.ID, WithdrawalStatusPaid, 1, "manual", "tx-123", "ok")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, WithdrawalStatusPaid, result.Status)
	assert.Equal(t, "manual", result.PayoutChannel)
	assert.Equal(t, "tx-123", result.PayoutTxId)

	// Paid: balance unchanged (already debited at request time).
	var afterPaid User
	require.NoError(t, DB.First(&afterPaid, user.Id).Error)
	assert.Equal(t, int64(3000), afterPaid.CommissionBalanceCents)
}

func TestAdminProcessWithdrawal_RejectedRefundsBalance(t *testing.T) {
	setupCommissionTest(t)

	user := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
		u.CommissionBalanceCents = 5000
	})

	req, err := CreateWithdrawalRequest(user.Id, 2000)
	require.NoError(t, err)

	result, err := AdminProcessWithdrawal(req.ID, WithdrawalStatusRejected, 1, "", "", "invalid account")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, WithdrawalStatusRejected, result.Status)

	// Rejected: amount refunded back to balance.
	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, int64(5000), refreshed.CommissionBalanceCents) // 3000 + 2000 refund
}

func TestAdminProcessWithdrawal_IsIdempotent(t *testing.T) {
	setupCommissionTest(t)

	user := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
		u.CommissionBalanceCents = 5000
	})

	req, err := CreateWithdrawalRequest(user.Id, 2000)
	require.NoError(t, err)

	// First process: paid.
	r1, err := AdminProcessWithdrawal(req.ID, WithdrawalStatusPaid, 1, "manual", "tx-1", "")
	require.NoError(t, err)
	assert.Equal(t, WithdrawalStatusPaid, r1.Status)

	var afterPaid User
	require.NoError(t, DB.First(&afterPaid, user.Id).Error)
	balanceAfterPaid := afterPaid.CommissionBalanceCents

	// Second process (reject) on already-paid: no-op, no refund.
	r2, err := AdminProcessWithdrawal(req.ID, WithdrawalStatusRejected, 1, "", "", "changed mind")
	require.NoError(t, err)
	assert.Equal(t, WithdrawalStatusPaid, r2.Status) // still paid

	var afterRetry User
	require.NoError(t, DB.First(&afterRetry, user.Id).Error)
	assert.Equal(t, balanceAfterPaid, afterRetry.CommissionBalanceCents) // no refund
}

// --- Mutual exclusion: approved inviter skips ¥100 invite reward ---

func TestTrySettleDelayedInviteReward_ApprovedInviterSkipsQuotaReward(t *testing.T) {
	setupCommissionTest(t)
	// Configure delayed invite reward.
	common.DelayedInviteReward = true
	common.InviteRewardCNYYuan = 100
	common.MaxValidInvites = 10

	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = true
	})
	invitee := createCommissionTestUser(t, func(u *User) {
		u.InviterId = inviter.Id
		u.InviteRewardPending = true
		u.Email = "approved-invitee@example.com"
		u.UsedQuota = 1
	})
	// Invitee needs a token to qualify.
	require.NoError(t, DB.Create(&Token{
		UserId:      invitee.Id,
		Key:         fmt.Sprintf("tok-%d", invitee.Id),
		Name:        "test",
		Status:      common.TokenStatusEnabled,
		CreatedTime: common.GetTimestamp(),
	}).Error)

	err := TrySettleDelayedInviteReward(invitee.Id)
	require.NoError(t, err)

	var refreshedInviter User
	require.NoError(t, DB.First(&refreshedInviter, inviter.Id).Error)
	// Approved inviter: no ¥100 quota reward (mutual exclusion).
	assert.Zero(t, refreshedInviter.Quota)
	assert.Zero(t, refreshedInviter.PromoQuota)
	assert.Zero(t, refreshedInviter.RewardedInviteCount)
	// But AffCount still increments (affiliate tracking).
	assert.Equal(t, 1, refreshedInviter.AffCount)
}

func TestTrySettleDelayedInviteReward_NonApprovedInviterGetsQuotaReward(t *testing.T) {
	setupCommissionTest(t)
	common.DelayedInviteReward = true
	common.InviteRewardCNYYuan = 100
	common.MaxValidInvites = 10

	inviter := createCommissionTestUser(t, func(u *User) {
		u.CommissionApproved = false
	})
	invitee := createCommissionTestUser(t, func(u *User) {
		u.InviterId = inviter.Id
		u.InviteRewardPending = true
		u.Email = "normal-invitee@example.com"
		u.UsedQuota = 1
	})
	require.NoError(t, DB.Create(&Token{
		UserId:      invitee.Id,
		Key:         fmt.Sprintf("tok2-%d", invitee.Id),
		Name:        "test",
		Status:      common.TokenStatusEnabled,
		CreatedTime: common.GetTimestamp(),
	}).Error)

	err := TrySettleDelayedInviteReward(invitee.Id)
	require.NoError(t, err)

	var refreshedInviter User
	require.NoError(t, DB.First(&refreshedInviter, inviter.Id).Error)
	// Non-approved inviter: gets ¥100 quota reward.
	assert.Positive(t, refreshedInviter.Quota)
	assert.Positive(t, refreshedInviter.PromoQuota)
	assert.Equal(t, 1, refreshedInviter.RewardedInviteCount)
	assert.Equal(t, 1, refreshedInviter.AffCount)
}
