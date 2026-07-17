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
