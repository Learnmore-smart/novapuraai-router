package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateModelDiscountByJSONStringValidation(t *testing.T) {
	saved := ModelDiscount2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelDiscountByJSONString(saved))
	})

	require.NoError(t, UpdateModelDiscountByJSONString(`{"model-a":0.8,"model-b":1}`))

	rate, ok := GetModelDiscount("model-a")
	assert.True(t, ok)
	assert.Equal(t, 0.8, rate)

	// A rate of exactly 1 is valid but reported as "no active discount".
	rate, ok = GetModelDiscount("model-b")
	assert.False(t, ok)
	assert.Equal(t, 1.0, rate)

	rate, ok = GetModelDiscount("model-unknown")
	assert.False(t, ok)
	assert.Equal(t, 1.0, rate)

	// Out-of-range rates must be rejected so billing can never be inflated
	// (rate > 1) or zeroed/negated (rate <= 0) by a bad option payload.
	assert.Error(t, UpdateModelDiscountByJSONString(`{"model-a":0}`))
	assert.Error(t, UpdateModelDiscountByJSONString(`{"model-a":-0.5}`))
	assert.Error(t, UpdateModelDiscountByJSONString(`{"model-a":1.5}`))
	assert.Error(t, UpdateModelDiscountByJSONString(`not json`))

	// Failed updates must not clobber the previously loaded discounts.
	rate, ok = GetModelDiscount("model-a")
	assert.True(t, ok)
	assert.Equal(t, 0.8, rate)

	// Reloading with a smaller map drops stale entries entirely.
	require.NoError(t, UpdateModelDiscountByJSONString(`{"model-b":0.5}`))
	_, ok = GetModelDiscount("model-a")
	assert.False(t, ok)
}

func TestGlobalModelDiscountOverridesAndRestoresPerModelRates(t *testing.T) {
	saved := ModelDiscount2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelDiscountByJSONString(saved))
	})

	require.NoError(t, UpdateModelDiscountByJSONString(`{"*":0.6,"model-a":0.8}`))

	rate, ok := GetModelDiscount("model-a")
	assert.True(t, ok)
	assert.Equal(t, 0.6, rate)

	rate, ok = GetModelDiscount("model-without-own-rate")
	assert.True(t, ok)
	assert.Equal(t, 0.6, rate)

	require.NoError(t, UpdateModelDiscountByJSONString(`{"*":1,"model-a":0.8}`))
	rate, ok = GetModelDiscount("model-a")
	assert.True(t, ok)
	assert.Equal(t, 1.0, rate)

	require.NoError(t, UpdateModelDiscountByJSONString(`{"model-a":0.8}`))
	rate, ok = GetModelDiscount("model-a")
	assert.True(t, ok)
	assert.Equal(t, 0.8, rate)

	_, ok = GetGlobalModelDiscount()
	assert.False(t, ok)
}
