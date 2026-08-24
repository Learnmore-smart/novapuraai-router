package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePricingSyncDataRejectsTrailingQuoteModelName(t *testing.T) {
	malformed := common.CanonicalDeepSeekV4Flash0731 + `"`
	data := map[string]any{
		"model_ratio": map[string]any{malformed: 0.11},
	}

	err := validatePricingSyncData(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid upstream model name")
}

func TestValidatePricingSyncDataAcceptsCanonicalModelName(t *testing.T) {
	data := map[string]any{
		"model_ratio":      map[string]any{common.CanonicalDeepSeekV4Flash0731: 0.11},
		"completion_ratio": map[string]any{common.CanonicalDeepSeekV4Flash0731: 3.0},
	}

	require.NoError(t, validatePricingSyncData(data))
}
