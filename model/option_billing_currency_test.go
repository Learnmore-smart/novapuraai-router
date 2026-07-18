package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionMapAppliesValidatedBillingCurrencyConfig(t *testing.T) {
	originalConfig := setting.BillingCurrencyConfigJSON()
	optionMapWasNil := false
	common.OptionMapRWMutex.RLock()
	optionMapWasNil = common.OptionMap == nil
	originalOption, hadOption := common.OptionMap["BillingCurrencyConfig"]
	common.OptionMapRWMutex.RUnlock()
	if optionMapWasNil {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = make(map[string]string)
		common.OptionMapRWMutex.Unlock()
	}
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(originalConfig))
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if optionMapWasNil {
			common.OptionMap = nil
			return
		}
		if hadOption {
			common.OptionMap["BillingCurrencyConfig"] = originalOption
		} else {
			delete(common.OptionMap, "BillingCurrencyConfig")
		}
	})

	valid := `{"default_currency":"cad","currencies":{"cny":{"enabled":true,"fx_presentment_per_usd":7.3},"usd":{"enabled":true,"fx_presentment_per_usd":1},"cad":{"enabled":true,"fx_presentment_per_usd":1.4}}}`
	require.NoError(t, updateOptionMap("BillingCurrencyConfig", valid))
	assert.Equal(t, "cad", setting.DefaultBillingCurrency())
	assert.InDelta(t, 1.4, setting.BillingCurrencyFXRate("cad"), 0.000001)

	invalid := `{"default_currency":"cad","currencies":{"cad":{"enabled":false,"fx_presentment_per_usd":1.4}}}`
	require.Error(t, updateOptionMap("BillingCurrencyConfig", invalid))
	assert.Equal(t, "cad", setting.DefaultBillingCurrency())
	common.OptionMapRWMutex.RLock()
	assert.Equal(t, valid, common.OptionMap["BillingCurrencyConfig"])
	common.OptionMapRWMutex.RUnlock()
}
