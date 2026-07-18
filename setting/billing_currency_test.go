package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBillingCurrencyLaunchDefaults(t *testing.T) {
	cfg := GetBillingCurrencyConfig()

	assert.Equal(t, "cny", cfg.DefaultCurrency)
	assert.True(t, cfg.AutoUpdateFX)
	assert.Equal(t, []string{"cny", "usd", "cad"}, EnabledBillingCurrencies())
	assert.True(t, cfg.Currencies["cny"].Enabled)
	assert.InDelta(t, 7.3, cfg.Currencies["cny"].FXPresentmentPerUSD, 0.000001)
}

func TestApplyBillingCurrencyFXRatesPreservesPolicyAndRecordsSource(t *testing.T) {
	original := BillingCurrencyConfigJSON()
	t.Cleanup(func() {
		require.NoError(t, UpdateBillingCurrencyConfigByJSON(original))
	})
	require.NoError(t, UpdateBillingCurrencyConfigByJSON(`{
		"default_currency":"cad",
		"auto_update_fx":true,
		"currencies":{
			"cny":{"enabled":false,"fx_presentment_per_usd":7.3},
			"usd":{"enabled":true,"fx_presentment_per_usd":1},
			"cad":{"enabled":true,"fx_presentment_per_usd":1.37}
		}
	}`))

	raw, err := ApplyBillingCurrencyFXRates(map[string]float64{
		"cny": 6.87,
		"usd": 1,
		"cad": 1.36,
	}, "ecb", 1_752_796_800)
	require.NoError(t, err)
	require.NoError(t, UpdateBillingCurrencyConfigByJSON(raw))

	cfg := GetBillingCurrencyConfig()
	assert.Equal(t, "cad", cfg.DefaultCurrency)
	assert.False(t, cfg.Currencies["cny"].Enabled)
	assert.InDelta(t, 6.87, cfg.Currencies["cny"].FXPresentmentPerUSD, 0.000001)
	assert.InDelta(t, 1.36, cfg.Currencies["cad"].FXPresentmentPerUSD, 0.000001)
	assert.Equal(t, "ecb", cfg.FXSource)
	assert.Equal(t, int64(1_752_796_800), cfg.FXUpdatedAt)
	assert.InDelta(t, 6.87, cfg.ReferenceCurrencies["cny"], 0.000001)
}

func TestApplyBillingCurrencyFXRatesKeepsCustomRateAndUpdatesReference(t *testing.T) {
	original := BillingCurrencyConfigJSON()
	t.Cleanup(func() { require.NoError(t, UpdateBillingCurrencyConfigByJSON(original)) })
	require.NoError(t, UpdateBillingCurrencyConfigByJSON(`{
		"default_currency":"cny",
		"auto_update_fx":false,
		"currencies":{
			"cny":{"enabled":true,"fx_presentment_per_usd":7.3},
			"usd":{"enabled":true,"fx_presentment_per_usd":1},
			"cad":{"enabled":true,"fx_presentment_per_usd":1.37}
		}
	}`))

	raw, err := ApplyBillingCurrencyFXRates(map[string]float64{"cny": 6.91, "usd": 1, "cad": 1.36}, "ECB", 1_752_796_800)
	require.NoError(t, err)
	require.NoError(t, UpdateBillingCurrencyConfigByJSON(raw))

	cfg := GetBillingCurrencyConfig()
	assert.InDelta(t, 7.3, cfg.Currencies["cny"].FXPresentmentPerUSD, 0.000001)
	assert.InDelta(t, 6.91, cfg.ReferenceCurrencies["cny"], 0.000001)
	assert.Equal(t, "ECB", cfg.FXSource)
}

func TestUpdateBillingCurrencyConfigRejectsDisabledDefaultWithoutChangingState(t *testing.T) {
	original := BillingCurrencyConfigJSON()
	t.Cleanup(func() {
		require.NoError(t, UpdateBillingCurrencyConfigByJSON(original))
	})

	err := UpdateBillingCurrencyConfigByJSON(`{
		"default_currency":"cad",
		"currencies":{
			"cny":{"enabled":true,"fx_presentment_per_usd":7.3},
			"usd":{"enabled":true,"fx_presentment_per_usd":1},
			"cad":{"enabled":false,"fx_presentment_per_usd":1.37}
		}
	}`)

	require.ErrorContains(t, err, "default currency must be enabled")
	assert.Equal(t, original, BillingCurrencyConfigJSON())
}

func TestResolveBillingCurrencyUsesSavedThenLocaleThenAdminDefault(t *testing.T) {
	original := BillingCurrencyConfigJSON()
	t.Cleanup(func() {
		require.NoError(t, UpdateBillingCurrencyConfigByJSON(original))
	})
	require.NoError(t, UpdateBillingCurrencyConfigByJSON(`{
		"default_currency":"cny",
		"currencies":{
			"cny":{"enabled":true,"fx_presentment_per_usd":7.3},
			"usd":{"enabled":false,"fx_presentment_per_usd":1},
			"cad":{"enabled":true,"fx_presentment_per_usd":1.37}
		}
	}`))

	assert.Equal(t, "cad", ResolveBillingCurrency("cad", "CN", "zh-CN"))
	assert.Equal(t, "cad", ResolveBillingCurrency("usd", "CA", "en-CA"))
	assert.Equal(t, "cny", ResolveBillingCurrency("", "US", "en-US"))
	assert.Equal(t, "cny", ResolveBillingCurrency("", "FR", "fr-FR"))
}

func TestUpdateBillingCurrencyConfigRejectsUnknownCurrencyAndInvalidFX(t *testing.T) {
	original := BillingCurrencyConfigJSON()
	t.Cleanup(func() {
		require.NoError(t, UpdateBillingCurrencyConfigByJSON(original))
	})

	unknown := `{"default_currency":"cny","currencies":{"cny":{"enabled":true,"fx_presentment_per_usd":7.3},"eur":{"enabled":true,"fx_presentment_per_usd":0.9}}}`
	require.ErrorContains(t, UpdateBillingCurrencyConfigByJSON(unknown), "unsupported currency")

	invalidFX := `{"default_currency":"cny","currencies":{"cny":{"enabled":true,"fx_presentment_per_usd":0}}}`
	require.ErrorContains(t, UpdateBillingCurrencyConfigByJSON(invalidFX), "exchange rate")
	assert.Equal(t, original, BillingCurrencyConfigJSON())
}
