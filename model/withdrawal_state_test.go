package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTransitionWithdrawalStatus_CASSucceeds(t *testing.T) {
	truncateTables(t)
	uid := CreateTestUserWithBalance(t, 50000) // $500
	req, err := CreateWithdrawalRequest(uid, 1000)
	require.NoError(t, err)
	require.Equal(t, WithdrawalStatusPending, req.Status)

	out, err := TransitionWithdrawalStatus(req.ID, WithdrawalStatusPending, WithdrawalStatusTransferCreating, func(tx *gorm.DB, req *WithdrawalRequest) error {
		req.StripeAccountId = "acct_test_x"
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, WithdrawalStatusTransferCreating, out.Status)
	assert.Equal(t, "acct_test_x", out.StripeAccountId)
}

func TestTransitionWithdrawalStatus_ConflictWhenStatusChanged(t *testing.T) {
	truncateTables(t)
	uid := CreateTestUserWithBalance(t, 50000)
	req, _ := CreateWithdrawalRequest(uid, 1000)
	// 先翻到 transfer_creating
	_, _ = TransitionWithdrawalStatus(req.ID, WithdrawalStatusPending, WithdrawalStatusTransferCreating, nil)

	// 再从 pending 翻 → 冲突
	_, err := TransitionWithdrawalStatus(req.ID, WithdrawalStatusPending, WithdrawalStatusFailed, nil)
	assert.ErrorIs(t, err, ErrWithdrawalStatusConflict)
}

func TestMarkWithdrawalFailed_RefundsBalance(t *testing.T) {
	truncateTables(t)
	uid := CreateTestUserWithBalance(t, 50000)
	req, _ := CreateWithdrawalRequest(uid, 1000) // balance 50000-1000=49000
	_, err := TransitionWithdrawalStatus(req.ID, WithdrawalStatusPending, WithdrawalStatusTransferCreating, nil)
	require.NoError(t, err)

	_, err = MarkWithdrawalFailed(req.ID, WithdrawalStatusTransferCreating, "transfer_error: boom")
	require.NoError(t, err)

	var u User
	DB.First(&u, uid)
	assert.Equal(t, int64(50000), u.CommissionBalanceCents) // 退回后恢复原值
}
