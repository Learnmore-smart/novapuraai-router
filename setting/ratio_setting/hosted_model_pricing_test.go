package ratio_setting

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// Hosted catalog model IDs that returned price_unset from
// https://www.novapuraai.com/api/pricing on 2026-08-26.
var hostedCatalogModelsMissingPricing = []string{
	"01-ai/yi-large",
	"adept/fuyu-8b",
	"ai21labs/jamba-1.5-large-instruct",
	"aisingapore/sea-lion-7b-instruct",
	"bigcode/starcoder2-15b",
	"databricks/dbrx-instruct",
	"deepseek-ai/deepseek-coder-6.7b-instruct",
	"deepseek-ai/deepseek-v4-pro-0813",
	"google/codegemma-1.1-7b",
	"google/codegemma-7b",
	"google/deplot",
	"google/gemma-2b",
	"google/gemma-3-12b-it",
	"google/gemma-3-4b-it",
	"google/recurrentgemma-2b",
	"ibm/granite-3.0-3b-a800m-instruct",
	"ibm/granite-3.0-8b-instruct",
	"ibm/granite-34b-code-instruct",
	"ibm/granite-8b-code-instruct",
	"meta/codellama-70b",
	"meta/llama-guard-4-12b",
	"meta/llama2-70b",
	"microsoft/kosmos-2",
	"microsoft/phi-3-vision-128k-instruct",
	"microsoft/phi-3.5-moe-instruct",
	"mistralai/codestral-22b-instruct-v0.1",
	"mistralai/mistral-7b-instruct-v0.3",
	"mistralai/mistral-large",
	"mistralai/mistral-large-2-instruct",
	"mistralai/mixtral-8x22b-v0.1",
	"muse-glimmer-30b",
	"nemotron-3.5-lightning-30b-a3b",
	"nv-mistralai/mistral-nemo-12b-instruct",
	"nvidia/ai-synthetic-video-detector",
	"nvidia/cosmos-reason2-8b",
	"nvidia/embed-qa-4",
	"nvidia/ising-calibration-1.5-31b",
	"nvidia/llama-3.1-nemoguard-8b-content-safety",
	"nvidia/llama-3.1-nemoguard-8b-topic-control",
	"nvidia/llama-3.1-nemotron-51b-instruct",
	"nvidia/llama-3.1-nemotron-70b-instruct",
	"nvidia/llama-3.1-nemotron-safety-guard-8b-v3",
	"nvidia/llama-3.1-nemotron-ultra-253b-v1",
	"nvidia/llama-3.2-nemoretriever-1b-vlm-embed-v1",
	"nvidia/llama-3.2-nv-embedqa-1b-v1",
	"nvidia/llama-nemotron-embed-vl-1b-v2",
	"nvidia/llama3-chatqa-1.5-70b",
	"nvidia/mistral-nemo-minitron-8b-8k-instruct",
	"nvidia/nemotron-3-embed-1b",
	"nvidia/nemotron-3.5-content-safety",
	"nvidia/nemotron-4-340b-instruct",
	"nvidia/nemotron-4-340b-reward",
	"nvidia/nemotron-nano-3-30b-a3b",
	"nvidia/nemotron-parse",
	"nvidia/neva-22b",
	"nvidia/nv-embedqa-mistral-7b-v2",
	"nvidia/nvclip",
	"nvidia/riva-translate-4b-instruct",
	"nvidia/riva-translate-4b-instruct-v1.1",
	"nvidia/riva-translate-4b-instruct-v2",
	"nvidia/vila",
	"snowflake/arctic-embed-l",
	"writer/palmyra-creative-122b",
	"writer/palmyra-fin-70b-32k",
	"writer/palmyra-med-70b",
	"writer/palmyra-med-70b-32k",
	"zyphra/zamba2-7b-instruct",
}

func TestHostedCatalogModelsHaveDefaultTokenPricing(t *testing.T) {
	defaults := GetDefaultModelRatioMap()
	require.Len(t, hostedCatalogModelsMissingPricing, 67)
	for _, name := range hostedCatalogModelsMissingPricing {
		ratio, ok := defaults[name]
		require.True(t, ok, name)
		require.Greater(t, ratio, 0.0, name)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	sqlPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "bin", "add_hosted_model_pricing.sql")
	sql, err := os.ReadFile(sqlPath)
	require.NoError(t, err)
	for _, name := range hostedCatalogModelsMissingPricing {
		require.Contains(t, string(sql), name)
	}
}

func TestDeepSeekV4Pro0813HasDefaultTokenPricing(t *testing.T) {
	const name = "deepseek-v4-pro-0813"
	ratio, ok := GetDefaultModelRatioMap()[name]
	require.True(t, ok, name)
	require.Equal(t, 0.33, ratio)
	completion, ok := defaultCompletionRatio[name]
	require.True(t, ok, name)
	require.Equal(t, 3.0, completion)

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	sqlPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "bin", "add_deepseek_v4_pro_0813.sql")
	sql, err := os.ReadFile(sqlPath)
	require.NoError(t, err)
	require.Contains(t, string(sql), name)
	require.Contains(t, string(sql), `"deepseek-v4-pro-0813":0.33`)
	require.Contains(t, string(sql), `"deepseek-v4-pro-0813":3`)
}

func TestKimiK3HasOfficialTokenPricing(t *testing.T) {
	const name = "kimi-k3"
	ratio, ok := GetDefaultModelRatioMap()[name]
	require.True(t, ok, name)
	require.Equal(t, 1.5, ratio)

	completion, ok := defaultCompletionRatio[name]
	require.True(t, ok, name)
	require.Equal(t, 5.0, completion)

	cache, ok := defaultCacheRatio[name]
	require.True(t, ok, name)
	require.Equal(t, 0.1, cache)

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	sqlPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "bin", "fix_kimi_k3_pricing.sql")
	sql, err := os.ReadFile(sqlPath)
	require.NoError(t, err)
	require.Contains(t, string(sql), `"kimi-k3":1.5`)
	require.Contains(t, string(sql), `"kimi-k3":5`)
	require.Contains(t, string(sql), `"kimi-k3":0.1`)
}
