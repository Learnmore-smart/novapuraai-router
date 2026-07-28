package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withBillingCurrencyDefault swaps setting.DefaultBillingCurrency() for the
// duration of a test by rewriting the billing-currency config JSON. The
// previous config is restored on cleanup. Pass "" to leave only USD enabled
// (default currency USD).
func withBillingCurrencyDefault(t *testing.T, defaultCurrency string) {
	t.Helper()

	original := setting.BillingCurrencyConfigJSON()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(original))
	})

	// Always include CNY, USD so the supported-currency guards pass; the
	// default decides which one ResolveSubscriptionCurrency falls back to.
	cfg := `{"default_currency":"` + defaultCurrency + `","auto_update_fx":true,` +
		`"currencies":{` +
		`"cny":{"enabled":true,"fx_presentment_per_usd":7.3},` +
		`"usd":{"enabled":true,"fx_presentment_per_usd":1},` +
		`"cad":{"enabled":true,"fx_presentment_per_usd":1.37}` +
		`}}`
	require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(cfg))
}

// TestResolveSubscriptionCurrency_UserSettingWins verifies the top priority:
// when the user has explicitly chosen CNY or USD, that currency is returned
// regardless of the system default.
func TestResolveSubscriptionCurrency_UserSettingWins(t *testing.T) {
	withBillingCurrencyDefault(t, "usd")

	tests := []struct {
		name    string
		userSet string
		want    string
	}{
		{"user cny wins over usd default", "cny", SubscriptionCurrencyCNY},
		{"user CNY uppercase wins", "CNY", SubscriptionCurrencyCNY},
		{"user CNY with whitespace wins", "  cny ", SubscriptionCurrencyCNY},
		{"user usd wins over usd default", "usd", SubscriptionCurrencyUSD},
		{"user USD uppercase wins", "USD", SubscriptionCurrencyUSD},
		{"user cad not supported, falls through to default", "cad", SubscriptionCurrencyUSD},
		{"user eur not supported, falls through to default", "eur", SubscriptionCurrencyUSD},
		{"user empty falls through to default", "", SubscriptionCurrencyUSD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ResolveSubscriptionCurrency(tt.userSet))
		})
	}
}

// TestResolveSubscriptionCurrency_SystemDefaultFallback verifies the second
// priority: when the user setting is empty/unsupported, the system default
// wins if it is a supported subscription currency (CNY/USD).
func TestResolveSubscriptionCurrency_SystemDefaultFallback(t *testing.T) {
	withBillingCurrencyDefault(t, "cny")

	tests := []struct {
		name    string
		userSet string
		want    string
	}{
		{"empty user setting falls back to cny default", "", SubscriptionCurrencyCNY},
		{"cad user setting falls back to cny default", "cad", SubscriptionCurrencyCNY},
		{"eur user setting falls back to cny default", "eur", SubscriptionCurrencyCNY},
		{"explicit user usd still wins over cny default", "usd", SubscriptionCurrencyUSD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ResolveSubscriptionCurrency(tt.userSet))
		})
	}
}

// TestResolveSubscriptionCurrency_USDFinalFallback verifies the terminal
// fallback: when both user setting and system default are unsupported for
// subscriptions (e.g. system default is CAD), USD is returned.
func TestResolveSubscriptionCurrency_USDFinalFallback(t *testing.T) {
	withBillingCurrencyDefault(t, "cad")

	// CAD is a valid billing currency but not a valid subscription currency,
	// so the system-default branch returns "" from normalizeSubscriptionCurrency
	// and the final fallback to USD kicks in.
	assert.Equal(t, SubscriptionCurrencyUSD, ResolveSubscriptionCurrency(""))
	assert.Equal(t, SubscriptionCurrencyUSD, ResolveSubscriptionCurrency("cad"))
	// An explicit user choice still wins.
	assert.Equal(t, SubscriptionCurrencyCNY, ResolveSubscriptionCurrency("cny"))
	assert.Equal(t, SubscriptionCurrencyUSD, ResolveSubscriptionCurrency("usd"))
}

// TestNormalizeSubscriptionCurrency verifies the canonicalization helper
// directly: case-insensitive match, trim whitespace, reject unknowns.
func TestNormalizeSubscriptionCurrency(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"cny lowercase", "cny", SubscriptionCurrencyCNY},
		{"CNY uppercase", "CNY", SubscriptionCurrencyCNY},
		{"Cny mixed case", "Cny", SubscriptionCurrencyCNY},
		{"cny with whitespace", "  cny\t", SubscriptionCurrencyCNY},
		{"usd lowercase", "usd", SubscriptionCurrencyUSD},
		{"USD uppercase", "USD", SubscriptionCurrencyUSD},
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
		{"cad unsupported", "cad", ""},
		{"eur unsupported", "eur", ""},
		{"unknown", "jpy", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeSubscriptionCurrency(tt.raw))
		})
	}
}

// TestSubscriptionCurrencyConstants guards the exact wire values: the resolver
// returns uppercase ISO 4217 codes that match SubscriptionPlan.Currency and
// Stripe Checkout's currency convention.
func TestSubscriptionCurrencyConstants(t *testing.T) {
	assert.Equal(t, "CNY", SubscriptionCurrencyCNY)
	assert.Equal(t, "USD", SubscriptionCurrencyUSD)
}
