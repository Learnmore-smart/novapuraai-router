package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionMapAppliesStripeTopupOptions(t *testing.T) {
	originalProductID := setting.StripeTopupProductID
	originalEnabled := setting.StripeTopupEnabled
	common.OptionMapRWMutex.RLock()
	originalOptionMap := make(map[string]string, len(common.OptionMap))
	for key, value := range common.OptionMap {
		originalOptionMap[key] = value
	}
	common.OptionMapRWMutex.RUnlock()
	t.Cleanup(func() {
		setting.StripeTopupProductID = originalProductID
		setting.StripeTopupEnabled = originalEnabled
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		common.OptionMap = originalOptionMap
	})
	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()

	require.NoError(t, updateOptionMap("StripeTopupProductID", "prod_dashboard"))
	require.NoError(t, updateOptionMap("StripeTopupEnabled", "true"))

	assert.Equal(t, "prod_dashboard", setting.StripeTopupProductID)
	assert.True(t, setting.StripeTopupEnabled)
	common.OptionMapRWMutex.RLock()
	assert.Equal(t, "prod_dashboard", common.OptionMap["StripeTopupProductID"])
	assert.Equal(t, "true", common.OptionMap["StripeTopupEnabled"])
	common.OptionMapRWMutex.RUnlock()
}

func TestUpdateOptionMapAppliesStripeEnvironmentIdentifiers(t *testing.T) {
	originalTestProductID := setting.StripeTestTopupProductID
	originalProdProductID := setting.StripeProdTopupProductID
	originalTestAccountID := setting.StripeTestAccountID
	originalProdAccountID := setting.StripeProdAccountID
	originalRuntime := setting.StripeRuntimeEnvironment
	originalActiveProductID := setting.StripeTopupProductID
	originalActiveAccountID := setting.StripeAccountID
	common.OptionMapRWMutex.RLock()
	originalOptionMap := make(map[string]string, len(common.OptionMap))
	for key, value := range common.OptionMap {
		originalOptionMap[key] = value
	}
	common.OptionMapRWMutex.RUnlock()
	t.Cleanup(func() {
		setting.StripeTestTopupProductID = originalTestProductID
		setting.StripeProdTopupProductID = originalProdProductID
		setting.StripeTestAccountID = originalTestAccountID
		setting.StripeProdAccountID = originalProdAccountID
		setting.StripeRuntimeEnvironment = originalRuntime
		setting.StripeTopupProductID = originalActiveProductID
		setting.StripeAccountID = originalActiveAccountID
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	setting.StripeRuntimeEnvironment = setting.StripeRuntimeTest
	require.NoError(t, updateOptionMap("StripeTestTopupProductID", "prod_test_dashboard"))
	require.NoError(t, updateOptionMap("StripeProdTopupProductID", "prod_live_dashboard"))
	require.NoError(t, updateOptionMap("StripeTestAccountID", "acct_test_dashboard"))
	require.NoError(t, updateOptionMap("StripeProdAccountID", "acct_live_dashboard"))

	assert.Equal(t, "prod_test_dashboard", setting.StripeTestTopupProductID)
	assert.Equal(t, "prod_live_dashboard", setting.StripeProdTopupProductID)
	assert.Equal(t, "acct_test_dashboard", setting.StripeTestAccountID)
	assert.Equal(t, "acct_live_dashboard", setting.StripeProdAccountID)
	assert.Equal(t, "prod_test_dashboard", setting.StripeTopupProductID)
	assert.Equal(t, "acct_test_dashboard", setting.StripeAccountID)
}
