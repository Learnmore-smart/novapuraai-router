package model

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var manualTopUpTestSequence atomic.Int64

func setupManualTopUpTest(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&TopUp{}, &User{}, &Log{}))
	for _, table := range []any{&TopUp{}, &User{}, &Log{}} {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []any{&TopUp{}, &User{}, &Log{}} {
			_ = DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error
		}
	})
}

func createManualTopUpFixture(t *testing.T, quota int, amount int64) (*User, *TopUp) {
	t.Helper()
	sequence := manualTopUpTestSequence.Add(1)
	user := &User{
		Username: fmt.Sprintf("manual_topup_%d", sequence),
		Password: "test-password",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Quota:    quota,
	}
	require.NoError(t, DB.Create(user).Error)

	topUp := &TopUp{
		UserId:          user.Id,
		Amount:          amount,
		Money:           float64(amount),
		TradeNo:         fmt.Sprintf("manual-topup-%d", sequence),
		PaymentMethod:   "manual",
		PaymentProvider: PaymentProviderEpay,
		CreateTime:      common.GetTimestamp(),
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)
	return user, topUp
}

func TestManualCompleteTopUpReturnsBalanceTransitionAndIsIdempotent(t *testing.T) {
	setupManualTopUpTest(t)
	user, topUp := createManualTopUpFixture(t, 1_000, 2)
	expectedAdded := common.QuotaFromDecimal(
		decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Truncate(0),
	)

	result, err := ManualCompleteTopUp(topUp.TradeNo)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.AlreadyCompleted)
	assert.Equal(t, user.Id, result.UserId)
	assert.Equal(t, expectedAdded, result.QuotaAdded)
	assert.Equal(t, 1_000, result.QuotaBefore)
	assert.Equal(t, 1_000+expectedAdded, result.QuotaAfter)

	var refreshedUser User
	require.NoError(t, DB.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, result.QuotaAfter, refreshedUser.Quota)
	var refreshedTopUp TopUp
	require.NoError(t, DB.First(&refreshedTopUp, topUp.Id).Error)
	assert.Equal(t, common.TopUpStatusSuccess, refreshedTopUp.Status)
	assert.Positive(t, refreshedTopUp.CompleteTime)

	secondResult, err := ManualCompleteTopUp(topUp.TradeNo)
	require.NoError(t, err)
	require.NotNil(t, secondResult)
	assert.True(t, secondResult.AlreadyCompleted)
	assert.Zero(t, secondResult.QuotaAdded)
	assert.Equal(t, result.QuotaAfter, secondResult.QuotaBefore)
	assert.Equal(t, result.QuotaAfter, secondResult.QuotaAfter)

	require.NoError(t, DB.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, result.QuotaAfter, refreshedUser.Quota)
}

func TestManualCompleteTopUpRejectsBalanceOverflow(t *testing.T) {
	setupManualTopUpTest(t)
	user, topUp := createManualTopUpFixture(t, common.MaxQuota-10, 1)

	result, err := ManualCompleteTopUp(topUp.TradeNo)
	require.Error(t, err)
	assert.Nil(t, result)

	var refreshedUser User
	require.NoError(t, DB.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, common.MaxQuota-10, refreshedUser.Quota)
	var refreshedTopUp TopUp
	require.NoError(t, DB.First(&refreshedTopUp, topUp.Id).Error)
	assert.Equal(t, common.TopUpStatusPending, refreshedTopUp.Status)
}
