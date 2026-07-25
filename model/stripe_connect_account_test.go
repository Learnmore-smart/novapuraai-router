package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreateStripeConnectAccount_CreatesThenReturns(t *testing.T) {
	truncateTables(t)
	acc, err := GetOrCreateStripeConnectAccount(9001, "acct_test_001")
	require.NoError(t, err)
	require.Equal(t, "acct_test_001", acc.StripeAccountId)
	require.Equal(t, ConnectOnboardingCreated, acc.OnboardingState)

	// 第二次调用返回已存在记录
	acc2, err := GetOrCreateStripeConnectAccount(9001, "acct_test_001")
	require.NoError(t, err)
	assert.Equal(t, acc.ID, acc2.ID)
}

func TestUpdateStripeConnectAccountFromStripe_EnabledWhenPayoutsAndDetails(t *testing.T) {
	truncateTables(t)
	_, err := GetOrCreateStripeConnectAccount(9002, "acct_test_002")
	require.NoError(t, err)

	err = UpdateStripeConnectAccountFromStripe(9002, "acct_test_002", "u@e.com", "US", true, true, "manual", "[]", "[]")
	require.NoError(t, err)

	acc, _ := GetStripeConnectAccountByStripeId("acct_test_002")
	assert.Equal(t, ConnectOnboardingEnabled, acc.OnboardingState)
	assert.True(t, acc.PayoutsEnabled)
}

func TestUpdateStripeConnectAccountFromStripe_RestrictedWhenCurrentlyDue(t *testing.T) {
	truncateTables(t)
	_, err := GetOrCreateStripeConnectAccount(9003, "acct_test_003")
	require.NoError(t, err)

	err = UpdateStripeConnectAccountFromStripe(9003, "acct_test_003", "u@e.com", "US", false, false, "manual", `["individual.verification.document"]`, "[]")
	require.NoError(t, err)

	acc, _ := GetStripeConnectAccountByStripeId("acct_test_003")
	assert.Equal(t, ConnectOnboardingRestricted, acc.OnboardingState)
}
