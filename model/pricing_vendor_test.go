package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchDefaultVendorUsesPrefixBeforeSubstring(t *testing.T) {
	cases := []struct {
		model  string
		vendor string
	}{
		{model: "adept/fuyu-8b", vendor: "Adept"},
		{model: "ai21labs/jamba-1.5-large-instruct", vendor: "AI21"},
		{model: "aisingapore/sea-lion-7b-instruct", vendor: "AI Singapore"},
		{model: "bigcode/starcoder2-15b", vendor: "BigCode"},
		{model: "databricks/dbrx-instruct", vendor: "Databricks"},
		{model: "nvidia/llama-3.1-nemotron-70b-instruct", vendor: "NVIDIA"},
		{model: "meta/llama-3.1-8b-instruct", vendor: "Meta"},
		{model: "google/gemma-3-12b-it", vendor: "Google"},
		{model: "ibm/granite-3.0-8b-instruct", vendor: "IBM"},
		{model: "writer/palmyra-x-004", vendor: "Writer"},
		{model: "zyphra/zamba2-7b-instruct", vendor: "Zyphra"},
		{model: "01-ai/yi-large", vendor: "零一万物"},
		{model: "black-forest-labs/flux-1.1-pro", vendor: "Black Forest Labs"},
		{model: "gpt-4o", vendor: "OpenAI"},
		{model: "diffusiongemma-26b-a4b-it", vendor: "Gemma"},
	}

	for _, tc := range cases {
		vendor, ok := matchDefaultVendor(tc.model)
		require.True(t, ok, tc.model)
		assert.Equal(t, tc.vendor, vendor, tc.model)
	}
}

func TestDefaultVendorIconsCoverNewBrands(t *testing.T) {
	for _, vendor := range []string{
		"Adept", "AI21", "AI Singapore", "BigCode", "Databricks",
		"IBM", "Meta", "NVIDIA", "Writer", "Zyphra", "Black Forest Labs",
	} {
		icon := getDefaultVendorIcon(vendor)
		assert.NotEmpty(t, icon, vendor)
	}
}
