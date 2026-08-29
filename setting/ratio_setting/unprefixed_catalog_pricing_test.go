package ratio_setting

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnprefixedSquareModelsHaveDefaultTokenPricing(t *testing.T) {
	models := []string{
		"bevformer",
		"cosmos-transfer1-7b",
		"cosmos-transfer2.5-2b",
		"cosmos3-nano",
		"cosmos3-nano-reasoner",
		"ising-calibration-1-35b-a3b",
		"ising-calibration-1.5-31b",
		"llama-3.1-nemotron-safety-guard-8b-v3",
		"llama-guard-4-12b",
	}
	defaults := GetDefaultModelRatioMap()
	for _, name := range models {
		ratio, ok := defaults[name]
		require.True(t, ok, name)
		require.Greater(t, ratio, 0.0, name)
	}
	require.Equal(t, 3.8333333333, defaultCompletionRatio["cosmos3-nano"])
	require.Equal(t, 3.8333333333, defaultCompletionRatio["cosmos3-nano-reasoner"])
	require.Equal(t, 4.0, defaultCompletionRatio["ising-calibration-1-35b-a3b"])
	require.Equal(t, 3.8333333333, defaultCompletionRatio["llama-3.1-nemotron-safety-guard-8b-v3"])

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	sqlPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "bin", "add_unprefixed_catalog_pricing.sql")
	sql, err := os.ReadFile(sqlPath)
	require.NoError(t, err)
	for _, name := range models {
		require.Contains(t, string(sql), name)
	}
}
