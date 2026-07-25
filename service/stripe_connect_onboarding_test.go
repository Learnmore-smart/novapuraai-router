package service

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStripeConnectClient struct {
	createExpressCalls int
	createLinkCalls    int
	accountResult      *AccountResult
	accountErr         error
	linkResult         *AccountLinkResult
	linkErr            error
}

func (m *mockStripeConnectClient) CreateExpressAccount(_ context.Context, p CreateAccountParams) (*AccountResult, error) {
	m.createExpressCalls++
	if m.accountErr != nil {
		return nil, m.accountErr
	}
	return m.accountResult, nil
}

func (m *mockStripeConnectClient) CreateAccountLink(_ context.Context, _, _, _ string) (*AccountLinkResult, error) {
	m.createLinkCalls++
	if m.linkErr != nil {
		return nil, m.linkErr
	}
	return m.linkResult, nil
}

func (m *mockStripeConnectClient) CreateTransfer(_ context.Context, _ TransferParams, _ string) (*TransferResult, error) {
	return nil, nil
}

func (m *mockStripeConnectClient) ReverseTransfer(_ context.Context, _ string, _ int64, _ string) (*ReversalResult, error) {
	return nil, nil
}

func (m *mockStripeConnectClient) CreatePayout(_ context.Context, _ PayoutParams, _ string, _ string) (*PayoutResult, error) {
	return nil, nil
}

func (m *mockStripeConnectClient) GetBalanceAvailableUSD(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *mockStripeConnectClient) RetrieveAccount(_ context.Context, _ string) (*AccountResult, error) {
	return nil, nil
}

func (m *mockStripeConnectClient) ListExternalAccounts(_ context.Context, _ string) ([]ExternalAccount, error) {
	return nil, nil
}

func setupOnboardingDB(t *testing.T) {
	t.Helper()
	require.NoError(t, model.DB.AutoMigrate(&model.StripeConnectAccount{}))
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM stripe_connect_accounts")
	})
}

func withStripeConnectEnabled(t *testing.T, enabled bool) {
	t.Helper()
	saved := setting.StripeConnectEnabled
	setting.StripeConnectEnabled = enabled
	t.Cleanup(func() { setting.StripeConnectEnabled = saved })
}

func TestStartConnectOnboarding_CreatesAccountWhenMissing(t *testing.T) {
	setupOnboardingDB(t)
	withStripeConnectEnabled(t, true)

	mock := &mockStripeConnectClient{
		accountResult: &AccountResult{StripeAccountID: "acct_test_new"},
		linkResult:    &AccountLinkResult{URL: "https://onboarding.stripe.com/xyz"},
	}
	url, err := StartConnectOnboarding(context.Background(), mock, 9101, "u@e.com")
	require.NoError(t, err)
	require.Equal(t, "https://onboarding.stripe.com/xyz", url)
	assert.Equal(t, 1, mock.createExpressCalls)
	assert.Equal(t, 1, mock.createLinkCalls)

	acc, err := model.GetStripeConnectAccount(9101)
	require.NoError(t, err)
	assert.Equal(t, "acct_test_new", acc.StripeAccountId)
}

func TestStartConnectOnboarding_ReusesExistingAccount(t *testing.T) {
	setupOnboardingDB(t)
	withStripeConnectEnabled(t, true)

	_, err := model.CreateStripeConnectAccountRecord(9102, "acct_test_existing")
	require.NoError(t, err)

	mock := &mockStripeConnectClient{
		accountResult: &AccountResult{StripeAccountID: "acct_test_should_not_be_used"},
		linkResult:    &AccountLinkResult{URL: "https://onboarding.stripe.com/abc"},
	}
	url, err := StartConnectOnboarding(context.Background(), mock, 9102, "u@e.com")
	require.NoError(t, err)
	require.Equal(t, "https://onboarding.stripe.com/abc", url)
	assert.Equal(t, 0, mock.createExpressCalls)
	assert.Equal(t, 1, mock.createLinkCalls)
}

func TestStartConnectOnboarding_DisabledReturnsError(t *testing.T) {
	withStripeConnectEnabled(t, false)

	mock := &mockStripeConnectClient{
		accountResult: &AccountResult{StripeAccountID: "acct_test_new"},
		linkResult:    &AccountLinkResult{URL: "https://onboarding.stripe.com/xyz"},
	}
	_, err := StartConnectOnboarding(context.Background(), mock, 9103, "u@e.com")
	require.Error(t, err)
	assert.Equal(t, "stripe connect is not enabled", err.Error())
	assert.Equal(t, 0, mock.createExpressCalls)
	assert.Equal(t, 0, mock.createLinkCalls)
}

func TestStartConnectOnboarding_NilClientReturnsError(t *testing.T) {
	withStripeConnectEnabled(t, true)

	_, err := StartConnectOnboarding(context.Background(), nil, 9104, "u@e.com")
	require.Error(t, err)
	assert.Equal(t, "stripe connect client unavailable", err.Error())
}
