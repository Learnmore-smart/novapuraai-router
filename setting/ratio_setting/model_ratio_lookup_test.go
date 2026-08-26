package ratio_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfiguredModelRatioDoesNotInventDefault(t *testing.T) {
	originalRatio := ModelRatio2JSONString()
	originalSelfUse := operation_setting.SelfUseModeEnabled
	t.Cleanup(func() {
		require.NoError(t, UpdateModelRatioByJSONString(originalRatio))
		operation_setting.SelfUseModeEnabled = originalSelfUse
	})

	require.NoError(t, UpdateModelRatioByJSONString(`{"priced-model":1.5}`))
	operation_setting.SelfUseModeEnabled = false

	ratio, ok := GetConfiguredModelRatio("priced-model")
	require.True(t, ok)
	assert.InDelta(t, 1.5, ratio, 0.000001)

	_, ok = GetConfiguredModelRatio("adept/fuyu-8b")
	assert.False(t, ok)

	fallback, found, matchName := GetModelRatio("adept/fuyu-8b")
	assert.InDelta(t, 37.5, fallback, 0.000001)
	assert.False(t, found)
	assert.Equal(t, "adept/fuyu-8b", matchName)

	operation_setting.SelfUseModeEnabled = true
	fallback, found, _ = GetModelRatio("adept/fuyu-8b")
	assert.InDelta(t, 37.5, fallback, 0.000001)
	assert.True(t, found)
	_, ok = GetConfiguredModelRatio("adept/fuyu-8b")
	assert.False(t, ok)
}

func TestUpdateModelRatioDropsNullEntries(t *testing.T) {
	originalRatio := ModelRatio2JSONString()
	originalCompletion := CompletionRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelRatioByJSONString(originalRatio))
		require.NoError(t, UpdateCompletionRatioByJSONString(originalCompletion))
	})

	require.NoError(t, UpdateModelRatioByJSONString(`{"keep":1.5,"adept/fuyu-8b":null}`))
	require.NoError(t, UpdateCompletionRatioByJSONString(`{"keep":3}`))

	ratio, ok := GetConfiguredModelRatio("keep")
	require.True(t, ok)
	assert.InDelta(t, 1.5, ratio, 0.000001)

	_, ok = GetConfiguredModelRatio("adept/fuyu-8b")
	assert.False(t, ok)
	assert.NotContains(t, ModelRatio2JSONString(), "adept/fuyu-8b")
}
