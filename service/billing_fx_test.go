package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchBankOfCanadaBillingFXRatesDerivesUSDBaseCrossRates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{
  "observations": [
    {"d":"2025-07-17","FXUSDCAD":{"v":"1.35"},"FXCNYCAD":{"v":"0.195"}},
    {"d":"2025-07-18","FXUSDCAD":{"v":"1.36"},"FXCNYCAD":{"v":"0.1971014492753623"}}
  ]
}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	rates, publishedAt, err := fetchBankOfCanadaBillingFXRates(context.Background(), server.Client(), server.URL)
	require.NoError(t, err)
	assert.InDelta(t, 1, rates["usd"], 0.0000001)
	assert.InDelta(t, 6.9, rates["cny"], 0.0000001)
	assert.InDelta(t, 1.36, rates["cad"], 0.0000001)
	assert.Equal(t, int64(1_752_796_800), publishedAt)
}

func TestFetchBankOfCanadaBillingFXRatesRejectsIncompleteProviderData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"observations":[{"d":"2025-07-18","FXUSDCAD":{"v":"1.36"}}]}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	_, _, err := fetchBankOfCanadaBillingFXRates(context.Background(), server.Client(), server.URL)
	require.ErrorContains(t, err, "missing")
}
