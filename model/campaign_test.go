package model

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var campaignTestSequence atomic.Int64

func setupCampaignTest(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&CampaignClaim{}, &CampaignCounter{}, &ShareSubmission{}))
	require.NoError(t, ensureShareSubmissionIndexes())
	for _, table := range []any{&ShareSubmission{}, &CampaignClaim{}, &CampaignCounter{}, &Token{}, &User{}, &Log{}} {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
	}

	originalRegisterEnabled := common.RegisterPromoEnabled
	originalEmailVerification := common.EmailVerificationEnabled
	originalRegisterMax := common.RegisterPromoMax
	originalRegisterCNY := common.RegisterPromoCNYYuan
	originalDelayedInvite := common.DelayedInviteReward
	originalInviteCNY := common.InviteRewardCNYYuan
	originalMaxInvites := common.MaxValidInvites
	originalShareCNY := common.ShareRewardCNYYuan
	originalExchangeRate := operation_setting.USDExchangeRate

	common.RegisterPromoEnabled = true
	common.EmailVerificationEnabled = true
	common.RegisterPromoMax = 5
	common.RegisterPromoCNYYuan = 2
	common.DelayedInviteReward = true
	common.InviteRewardCNYYuan = 1
	common.MaxValidInvites = 2
	common.ShareRewardCNYYuan = 1
	operation_setting.USDExchangeRate = common.DefaultUSDExchangeRate

	t.Cleanup(func() {
		for _, table := range []any{&ShareSubmission{}, &CampaignClaim{}, &CampaignCounter{}, &Token{}, &User{}, &Log{}} {
			_ = DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error
		}
		common.RegisterPromoEnabled = originalRegisterEnabled
		common.EmailVerificationEnabled = originalEmailVerification
		common.RegisterPromoMax = originalRegisterMax
		common.RegisterPromoCNYYuan = originalRegisterCNY
		common.DelayedInviteReward = originalDelayedInvite
		common.InviteRewardCNYYuan = originalInviteCNY
		common.MaxValidInvites = originalMaxInvites
		common.ShareRewardCNYYuan = originalShareCNY
		operation_setting.USDExchangeRate = originalExchangeRate
	})
}

func createCampaignTestUser(t *testing.T, configure func(*User)) *User {
	t.Helper()
	sequence := campaignTestSequence.Add(1)
	user := &User{
		Username: fmt.Sprintf("camp_%d", sequence),
		Password: "test-password",
		AffCode:  fmt.Sprintf("aff_%d", sequence),
		Email:    fmt.Sprintf("camp-%d@example.com", sequence),
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if configure != nil {
		configure(user)
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestCampaignRewardDefaults(t *testing.T) {
	assert.Equal(t, 20.0, common.RegisterPromoCNYYuan)
	assert.Equal(t, 50.0, common.InviteRewardCNYYuan)
}

func TestRegistrationFinalizationSharesIdempotentPromoClaim(t *testing.T) {
	setupCampaignTest(t)
	originalNewUserQuota := common.QuotaForNewUser
	common.QuotaForNewUser = 0
	t.Cleanup(func() {
		common.QuotaForNewUser = originalNewUserQuota
	})

	passwordUser := &User{
		Username: fmt.Sprintf("password-register-%d", campaignTestSequence.Add(1)),
		Password: "password123",
		Email:    "password-register@example.com",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, passwordUser.Insert(0))
	passwordUser.FinishInsert(0)
	passwordUser.FinishInsert(0)

	oauthUser := &User{
		Username: fmt.Sprintf("oauth-register-%d", campaignTestSequence.Add(1)),
		Email:    "oauth-register@example.com",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return oauthUser.InsertWithTx(tx, 0)
	}))
	oauthUser.FinalizeOAuthUserCreation(0)
	oauthUser.FinalizeOAuthUserCreation(0)

	expectedAmount := common.CNYYuanToQuota(2, exchangeRateForCampaign())
	for _, userID := range []int{passwordUser.Id, oauthUser.Id} {
		var refreshed User
		require.NoError(t, DB.First(&refreshed, userID).Error)
		assert.Equal(t, expectedAmount, refreshed.Quota)
		assert.Equal(t, expectedAmount, refreshed.PromoQuota)

		var claimCount int64
		require.NoError(t, DB.Model(&CampaignClaim{}).
			Where("user_id = ? AND kind = ?", userID, CampaignKindRegisterPromo).
			Count(&claimCount).Error)
		assert.EqualValues(t, 1, claimCount)
	}
}

func qualifyCampaignInvitee(t *testing.T, inviterId int) *User {
	t.Helper()
	invitee := createCampaignTestUser(t, func(user *User) {
		user.Email = fmt.Sprintf("invitee-%d@example.com", campaignTestSequence.Load()+1)
		user.InviterId = inviterId
		user.InviteRewardPending = true
		user.UsedQuota = 1
	})
	tokenKey := fmt.Sprintf("campaign-token-%d-%d", invitee.Id, time.Now().UnixNano())
	require.NoError(t, DB.Create(&Token{
		UserId:      invitee.Id,
		Key:         tokenKey,
		Name:        "campaign-test",
		Status:      common.TokenStatusEnabled,
		CreatedTime: common.GetTimestamp(),
	}).Error)
	return invitee
}

func TestTryGrantRegisterPromoIsIdempotent(t *testing.T) {
	setupCampaignTest(t)
	user := createCampaignTestUser(t, nil)

	granted, amount, err := TryGrantRegisterPromo(user.Id)
	require.NoError(t, err)
	assert.True(t, granted)
	assert.Positive(t, amount)

	grantedAgain, amountAgain, err := TryGrantRegisterPromo(user.Id)
	require.NoError(t, err)
	assert.False(t, grantedAgain)
	assert.Equal(t, amount, amountAgain)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, amount, refreshed.Quota)
	assert.Equal(t, amount, refreshed.PromoQuota)

	var claimCount int64
	require.NoError(t, DB.Model(&CampaignClaim{}).
		Where("user_id = ? AND kind = ?", user.Id, CampaignKindRegisterPromo).
		Count(&claimCount).Error)
	assert.EqualValues(t, 1, claimCount)

	var counter CampaignCounter
	require.NoError(t, DB.Where("name = ?", CounterRegisterPromo).First(&counter).Error)
	assert.Equal(t, 1, counter.Count)
}

func TestTryGrantRegisterPromoStopsExactlyAtConfiguredMaximum(t *testing.T) {
	setupCampaignTest(t)
	const attempts = 20
	users := make([]*User, 0, attempts)
	for range attempts {
		users = append(users, createCampaignTestUser(t, nil))
	}

	var waitGroup sync.WaitGroup
	results := make(chan bool, attempts)
	errors := make(chan error, attempts)
	for _, user := range users {
		waitGroup.Add(1)
		go func(userId int) {
			defer waitGroup.Done()
			granted, _, err := TryGrantRegisterPromo(userId)
			errors <- err
			results <- granted
		}(user.Id)
	}
	waitGroup.Wait()
	close(results)
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
	grantCount := 0
	for granted := range results {
		if granted {
			grantCount++
		}
	}
	assert.Equal(t, common.RegisterPromoMax, grantCount)

	var counter CampaignCounter
	require.NoError(t, DB.Where("name = ?", CounterRegisterPromo).First(&counter).Error)
	assert.Equal(t, common.RegisterPromoMax, counter.Count)

	var claimCount int64
	require.NoError(t, DB.Model(&CampaignClaim{}).
		Where("kind = ?", CampaignKindRegisterPromo).
		Count(&claimCount).Error)
	assert.EqualValues(t, common.RegisterPromoMax, claimCount)
}

func TestTrySettleDelayedInviteRewardRequiresQualificationAndIsIdempotent(t *testing.T) {
	setupCampaignTest(t)
	inviter := createCampaignTestUser(t, nil)
	invitee := createCampaignTestUser(t, func(user *User) {
		user.Email = "not-qualified@example.com"
		user.InviterId = inviter.Id
		user.InviteRewardPending = true
		user.UsedQuota = 1
	})

	require.NoError(t, TrySettleDelayedInviteReward(invitee.Id))
	var unqualified User
	require.NoError(t, DB.First(&unqualified, invitee.Id).Error)
	assert.True(t, unqualified.InviteRewardPending)
	assert.Zero(t, unqualified.Quota)

	require.NoError(t, DB.Create(&Token{
		UserId:      invitee.Id,
		Key:         fmt.Sprintf("qualified-%d", invitee.Id),
		Name:        "qualified",
		Status:      common.TokenStatusEnabled,
		CreatedTime: common.GetTimestamp(),
	}).Error)
	require.NoError(t, TrySettleDelayedInviteReward(invitee.Id))
	require.NoError(t, TrySettleDelayedInviteReward(invitee.Id))

	amount := common.CNYYuanToQuota(common.InviteRewardCNYYuan, exchangeRateForCampaign())
	var refreshedInvitee User
	var refreshedInviter User
	require.NoError(t, DB.First(&refreshedInvitee, invitee.Id).Error)
	require.NoError(t, DB.First(&refreshedInviter, inviter.Id).Error)
	assert.False(t, refreshedInvitee.InviteRewardPending)
	assert.Equal(t, amount, refreshedInvitee.Quota)
	assert.Equal(t, amount, refreshedInvitee.PromoQuota)
	assert.Equal(t, amount, refreshedInviter.Quota)
	assert.Equal(t, amount, refreshedInviter.PromoQuota)
	assert.Equal(t, 1, refreshedInviter.AffCount)
	assert.Equal(t, 1, refreshedInviter.RewardedInviteCount)

	var claimCount int64
	require.NoError(t, DB.Model(&CampaignClaim{}).Count(&claimCount).Error)
	assert.EqualValues(t, 2, claimCount)
}

func TestTrySettleDelayedInviteRewardRejectsSelfInvite(t *testing.T) {
	setupCampaignTest(t)
	user := qualifyCampaignInvitee(t, 0)
	require.NoError(t, DB.Model(user).Updates(map[string]any{
		"inviter_id":            user.Id,
		"invite_reward_pending": true,
	}).Error)

	require.NoError(t, TrySettleDelayedInviteReward(user.Id))

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.False(t, refreshed.InviteRewardPending)
	assert.Zero(t, refreshed.Quota)
	assert.Zero(t, refreshed.PromoQuota)
	var claimCount int64
	require.NoError(t, DB.Model(&CampaignClaim{}).Count(&claimCount).Error)
	assert.Zero(t, claimCount)
}

func TestTrySettleDelayedInviteRewardHonorsInviterCap(t *testing.T) {
	setupCampaignTest(t)
	inviter := createCampaignTestUser(t, func(user *User) {
		user.AffCount = common.MaxValidInvites
		user.RewardedInviteCount = common.MaxValidInvites
	})
	invitee := qualifyCampaignInvitee(t, inviter.Id)

	require.NoError(t, TrySettleDelayedInviteReward(invitee.Id))

	amount := common.CNYYuanToQuota(common.InviteRewardCNYYuan, exchangeRateForCampaign())
	var refreshedInvitee User
	var refreshedInviter User
	require.NoError(t, DB.First(&refreshedInvitee, invitee.Id).Error)
	require.NoError(t, DB.First(&refreshedInviter, inviter.Id).Error)
	assert.Equal(t, amount, refreshedInvitee.Quota)
	assert.Zero(t, refreshedInviter.Quota)
	assert.Equal(t, common.MaxValidInvites+1, refreshedInviter.AffCount)
	assert.Equal(t, common.MaxValidInvites, refreshedInviter.RewardedInviteCount)
}

func TestReviewShareSubmissionApprovesOnlyOnce(t *testing.T) {
	setupCampaignTest(t)
	user := createCampaignTestUser(t, nil)
	submission, err := CreateShareSubmission(
		user.Id,
		"https://social.example/posts/novapura",
		"social",
		"public campaign post",
	)
	require.NoError(t, err)

	require.NoError(t, ReviewShareSubmission(submission.Id, 99, true, "verified public post"))
	require.Error(t, ReviewShareSubmission(submission.Id, 99, true, "duplicate click"))

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, submission.Amount, refreshed.Quota)
	assert.Equal(t, submission.Amount, refreshed.PromoQuota)

	var claimCount int64
	require.NoError(t, DB.Model(&CampaignClaim{}).
		Where("user_id = ? AND kind = ?", user.Id, CampaignKindShare).
		Count(&claimCount).Error)
	assert.EqualValues(t, 1, claimCount)
}

func TestReviewShareSubmissionRequiresRejectionReason(t *testing.T) {
	setupCampaignTest(t)
	user := createCampaignTestUser(t, nil)
	submission, err := CreateShareSubmission(
		user.Id,
		"https://social.example/posts/reject",
		"social",
		"review me",
	)
	require.NoError(t, err)

	require.Error(t, ReviewShareSubmission(submission.Id, 99, false, "  "))
	var refreshed ShareSubmission
	require.NoError(t, DB.First(&refreshed, submission.Id).Error)
	assert.Equal(t, ShareStatusPending, refreshed.Status)
}

func TestReviewShareSubmissionRequiresApprovalReason(t *testing.T) {
	setupCampaignTest(t)
	user := createCampaignTestUser(t, nil)
	submission, err := CreateShareSubmission(
		user.Id,
		"https://social.example/posts/approve-reason",
		"social",
		"review me",
	)
	require.NoError(t, err)

	require.Error(t, ReviewShareSubmission(submission.Id, 99, true, "  "))
	var refreshed ShareSubmission
	require.NoError(t, DB.First(&refreshed, submission.Id).Error)
	assert.Equal(t, ShareStatusPending, refreshed.Status)
}

func TestRejectedShareSubmissionCanBeResubmittedMoreThanOnce(t *testing.T) {
	setupCampaignTest(t)
	user := createCampaignTestUser(t, nil)

	for attempt := 1; attempt <= 2; attempt++ {
		submission, err := CreateShareSubmission(
			user.Id,
			fmt.Sprintf("https://social.example/posts/retry-%d", attempt),
			"social",
			"review me again",
		)
		require.NoError(t, err)
		require.NoError(t, ReviewShareSubmission(submission.Id, 99, false, "evidence was not public"))
	}

	var rejectedCount int64
	require.NoError(t, DB.Model(&ShareSubmission{}).
		Where("user_id = ? AND status = ?", user.Id, ShareStatusRejected).
		Count(&rejectedCount).Error)
	assert.EqualValues(t, 2, rejectedCount)
}
