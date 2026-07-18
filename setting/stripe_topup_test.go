package setting

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopupMinMaxMinorUsesCurrencySpecificLaunchMinimums(t *testing.T) {
	cases := map[string][2]int64{
		"cny": {500, 50000},
		"usd": {50, 50000},
		"cad": {50, 50000},
	}
	for currency, expected := range cases {
		t.Run(currency, func(t *testing.T) {
			minimum, maximum := TopupMinMaxMinor(currency)
			assert.Equal(t, expected[0], minimum)
			assert.Equal(t, expected[1], maximum)
		})
	}
}

func TestExportTopupConfigIncludesExactMinorUnitLimits(t *testing.T) {
	var config struct {
		MinMaxMinor map[string][2]int64 `json:"min_max_minor"`
	}
	require.NoError(t, common.Unmarshal(ExportTopupConfigJSON(), &config))
	assert.Equal(t, [2]int64{500, 50000}, config.MinMaxMinor["cny"])
	assert.Equal(t, [2]int64{50, 50000}, config.MinMaxMinor["usd"])
	assert.Equal(t, [2]int64{50, 50000}, config.MinMaxMinor["cad"])
}
