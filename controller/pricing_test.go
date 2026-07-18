package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyBillingCurrencyPricesUsesCanonicalPricingAndFX(t *testing.T) {
	pricing := []model.Pricing{
		{ModelName: "token-model", ModelRatio: 1.5, CompletionRatio: 3},
		{ModelName: "fixed-model", QuotaType: 1, ModelPrice: 0.25},
	}

	applyBillingCurrencyPrices(pricing, "cad", 1.4)

	assert.Equal(t, "cad", pricing[0].BillingCurrency)
	assert.Equal(t, 1.4, pricing[0].BillingFXRate)
	require.NotNil(t, pricing[0].BillingInputPerMillion)
	require.NotNil(t, pricing[0].BillingOutputPerMillion)
	assert.InDelta(t, 4.2, *pricing[0].BillingInputPerMillion, 0.000001)
	assert.InDelta(t, 12.6, *pricing[0].BillingOutputPerMillion, 0.000001)
	require.NotNil(t, pricing[1].BillingPerRequest)
	assert.InDelta(t, 0.35, *pricing[1].BillingPerRequest, 0.000001)
}
