package ratio_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepairModelPricingJSONRepairsExactMalformedKeyAndIsIdempotent(t *testing.T) {
	malformed := `{"deepseek-v4-flash-0731\"":0.11,"other-model":0.8}`

	repaired, changed, err := RepairModelPricingJSON(malformed)
	require.NoError(t, err)
	assert.True(t, changed)

	values := make(map[string]float64)
	require.NoError(t, common.Unmarshal([]byte(repaired), &values))
	assert.Equal(t, 0.11, values[common.CanonicalDeepSeekV4Flash0731])
	assert.Equal(t, 0.8, values["other-model"])
	assert.NotContains(t, values, common.CanonicalDeepSeekV4Flash0731+`"`)

	secondRepair, changed, err := RepairModelPricingJSON(repaired)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.JSONEq(t, repaired, secondRepair)
}

func TestRepairModelPricingJSONFailsClosedOnConflictingCanonicalValues(t *testing.T) {
	conflicting := `{"deepseek-v4-flash-0731":0.11,"deepseek-v4-flash-0731\"":0.12}`

	_, changed, err := RepairModelPricingJSON(conflicting)
	require.Error(t, err)
	assert.False(t, changed)
	assert.Contains(t, err.Error(), "conflicting")
}

func TestRepairModelPricingJSONRejectsOtherMalformedNamesWithoutPartialRepair(t *testing.T) {
	malformed := `{"deepseek-v4-flash-0731\"":0.11,"other-model\"":0.8}`

	_, changed, err := RepairModelPricingJSON(malformed)
	require.Error(t, err)
	assert.False(t, changed)
}

func TestParseModelPricingNumberMapOmitsNullEntries(t *testing.T) {
	values, err := parseModelPricingNumberMap(`{"keep":1.5,"adept/fuyu-8b":null}`)
	require.NoError(t, err)
	assert.Equal(t, 1.5, values["keep"])
	assert.NotContains(t, values, "adept/fuyu-8b")
}

func TestValidateModelPricingJSONAllowsNullValues(t *testing.T) {
	require.NoError(t, ValidateModelPricingJSON(`{"adept/fuyu-8b":null,"gpt-4o":1.25}`))
}

func TestValidateModelPricingOptionValueCoversTieredBillingMaps(t *testing.T) {
	valid := `{"deepseek-v4-flash-0731":"tiered_expr"}`
	invalid := `{"deepseek-v4-flash-0731\"":"tiered_expr"}`

	require.NoError(t, ValidateModelPricingOptionValue("billing_setting.billing_mode", valid))
	require.NoError(t, ValidateModelPricingOptionValue("billing_setting.billing_expr", valid))
	assert.Error(t, ValidateModelPricingOptionValue("billing_setting.billing_mode", invalid))
	assert.Error(t, ValidateModelPricingOptionValue("billing_setting.billing_expr", invalid))
}

func TestValidateModelPricingOptionValueRejectsUnsafeNumericValues(t *testing.T) {
	assert.Error(t, ValidateModelPricingOptionValue("ModelPrice", `{"model":-0.01}`))
	require.NoError(t, ValidateModelPricingOptionValue("ModelRatio", `{"free-model":0}`))
	assert.Error(t, ValidateModelPricingOptionValue("ModelRatio", `{"model":-0.01}`))
	assert.Error(t, ValidateModelPricingOptionValue("CompletionRatio", `{"model":-1}`))
	assert.Error(t, ValidateModelPricingOptionValue("ModelDiscount", `{"model":1.01}`))
	require.NoError(t, ValidateModelPricingOptionValue("AudioCompletionRatio", `{"tts-1":0}`))
}

func TestValidateModelPricingOptionChangesRequiresOutputForNewTokenPricing(t *testing.T) {
	current := map[string]string{
		"ModelPrice":      `{}`,
		"ModelRatio":      `{"legacy":1}`,
		"CompletionRatio": `{}`,
	}
	assert.Error(t, ValidateModelPricingOptionChanges(map[string]string{
		"ModelRatio": `{"legacy":1,"new-model":0.2}`,
	}, current))
	require.NoError(t, ValidateModelPricingOptionChanges(map[string]string{
		"ModelRatio":      `{"legacy":1,"new-model":0.2}`,
		"CompletionRatio": `{"new-model":3}`,
	}, current))
	// A no-op save does not make an unrelated legacy fallback impossible to
	// administer; only new or changed token pricing must be explicit.
	require.NoError(t, ValidateModelPricingOptionChanges(map[string]string{
		"ModelRatio": `{"legacy":1}`,
	}, current))
}

func TestRepairModelPricingOptionValueBackfillsCanonicalDeepSeekDefaults(t *testing.T) {
	for key, expected := range map[string]float64{"ModelRatio": 0.11, "CompletionRatio": 3} {
		repaired, changed, err := RepairModelPricingOptionValue(key, `{"other-model":1}`)
		require.NoError(t, err)
		assert.True(t, changed)
		var values map[string]float64
		require.NoError(t, common.Unmarshal([]byte(repaired), &values))
		assert.Equal(t, expected, values[common.CanonicalDeepSeekV4Flash0731])
		assert.Equal(t, float64(1), values["other-model"])
	}
}

func TestModelPricingJSONUpdatesRejectTrailingQuoteNames(t *testing.T) {
	invalid := `{"deepseek-v4-flash-0731\"":0.11}`
	updates := map[string]func(string) error{
		"ModelPrice":           UpdateModelPriceByJSONString,
		"ModelRatio":           UpdateModelRatioByJSONString,
		"CompletionRatio":      UpdateCompletionRatioByJSONString,
		"CacheRatio":           UpdateCacheRatioByJSONString,
		"CreateCacheRatio":     UpdateCreateCacheRatioByJSONString,
		"ImageRatio":           UpdateImageRatioByJSONString,
		"AudioRatio":           UpdateAudioRatioByJSONString,
		"AudioCompletionRatio": UpdateAudioCompletionRatioByJSONString,
		"ModelDiscount":        UpdateModelDiscountByJSONString,
	}

	for name, update := range updates {
		t.Run(name, func(t *testing.T) {
			require.Error(t, update(invalid))
		})
	}
}

func TestRepairModelPricingOptionValuePreservesUnrelatedEntries(t *testing.T) {
	values := map[string]struct {
		json          string
		otherExpected float64
	}{
		"CacheRatio":    {json: `{"deepseek-v4-flash-0731\"":0.5,"other-model":0.25}`, otherExpected: 0.25},
		"ModelDiscount": {json: `{"deepseek-v4-flash-0731\"":0.8,"other-model":0.6}`, otherExpected: 0.6},
	}

	for optionKey, value := range values {
		t.Run(optionKey, func(t *testing.T) {
			repaired, changed, err := RepairModelPricingOptionValue(optionKey, value.json)
			require.NoError(t, err)
			assert.True(t, changed)

			var parsed map[string]float64
			require.NoError(t, common.Unmarshal([]byte(repaired), &parsed))
			assert.Equal(t, value.otherExpected, parsed["other-model"])
			assert.NotContains(t, parsed, common.CanonicalDeepSeekV4Flash0731+`"`)
			assert.Contains(t, parsed, common.CanonicalDeepSeekV4Flash0731)
		})
	}
}
