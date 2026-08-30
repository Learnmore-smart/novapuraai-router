package ratio_setting

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		"nemotron-3.5-content-safety",
		"riva-translate-4b-instruct-v1_1",
		"riva-translate-4b-instruct-v2",
		"sparsedrive",
		"streampetr",
		"synthetic-video-detector",
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
	require.Equal(t, 3.8333333333, defaultCompletionRatio["nemotron-3.5-content-safety"])
	require.Equal(t, 4.0, defaultCompletionRatio["riva-translate-4b-instruct-v1_1"])
	require.Equal(t, 4.0, defaultCompletionRatio["riva-translate-4b-instruct-v2"])

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	sqlFiles := []string{
		"add_unprefixed_catalog_pricing.sql",
		"add_remaining_unprefixed_pricing.sql",
	}
	for _, name := range models {
		found := false
		for _, file := range sqlFiles {
			sqlPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "bin", file)
			sql, err := os.ReadFile(sqlPath)
			require.NoError(t, err, file)
			if strings.Contains(string(sql), name) {
				found = true
				break
			}
		}
		require.True(t, found, name)
	}
}
