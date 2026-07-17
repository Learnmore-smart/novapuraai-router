package stripetopup

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectDefaultCurrency(t *testing.T) {
	assert.Equal(t, "cny", DetectDefaultCurrency("CN", "en-US"))
	assert.Equal(t, "cny", DetectDefaultCurrency("", "zh-CN"))
	assert.Equal(t, "cad", DetectDefaultCurrency("CA", "en"))
	assert.Equal(t, "usd", DetectDefaultCurrency("US", "en-US"))
	assert.Equal(t, "usd", DetectDefaultCurrency("", ""))
}

func TestBuildQuoteRejectsUnsupportedCurrency(t *testing.T) {
	_, err := BuildQuote(1, QuoteRequest{Currency: "eur", AmountMajor: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestBuildQuoteRejectsBelowMinimum(t *testing.T) {
	_, err := BuildQuote(1, QuoteRequest{Currency: "usd", AmountMajor: 0})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minimum")
}

func TestBuildQuoteRejectsAboveMaximum(t *testing.T) {
	_, err := BuildQuote(1, QuoteRequest{Currency: "usd", AmountMajor: 99999})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum")
}

func TestBuildQuoteUSDConversion(t *testing.T) {
	q, err := BuildQuote(1, QuoteRequest{Currency: "usd", AmountMajor: 10})
	require.NoError(t, err)
	assert.Equal(t, "usd", q.Currency)
	assert.Equal(t, int64(1000), q.AmountMinor)
	assert.Equal(t, float64(1), q.FxRateSnapshot)
	assert.Equal(t, int64(10)*setting.MicroUSDPerUSD, q.PaidCreditMicroUSD)
	assert.Greater(t, q.PaidQuota, 0)
	assert.Equal(t, q.PaidQuota+q.PromoQuota, q.TotalQuota)
	// Client cannot inject bonus — promo only from server tiers (none → 0)
	assert.Equal(t, int64(0), q.PromoCreditMicroUSD)
}

func TestBuildQuoteCNYAndCAD(t *testing.T) {
	cny, err := BuildQuote(1, QuoteRequest{Currency: "cny", AmountMajor: 50})
	require.NoError(t, err)
	assert.Equal(t, int64(5000), cny.AmountMinor)
	assert.InDelta(t, setting.StripeFXCNYPerUSD, cny.FxRateSnapshot, 0.0001)
	assert.Greater(t, cny.PaidCreditMicroUSD, int64(0))

	cad, err := BuildQuote(1, QuoteRequest{Currency: "cad", AmountMajor: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2000), cad.AmountMinor)
	assert.Greater(t, cad.PaidCreditMicroUSD, int64(0))
}

func TestPresentmentMinorToMicroUSDStable(t *testing.T) {
	// $5.00 at fx=1 → 5e6 micro
	m := setting.PresentmentMinorToMicroUSD("usd", 500, 1)
	assert.Equal(t, int64(5)*setting.MicroUSDPerUSD, m)
}

func TestValidateStripeSecretsRejectLive(t *testing.T) {
	origSecret := setting.StripeApiSecret
	origProd := setting.StripeTopupProductID
	origReq := setting.StripeRequireTestKeys
	t.Cleanup(func() {
		setting.StripeApiSecret = origSecret
		setting.StripeTopupProductID = origProd
		setting.StripeRequireTestKeys = origReq
	})
	setting.StripeRequireTestKeys = true
	setting.StripeTopupProductID = "prod_test"
	setting.StripeApiSecret = "sk_live_xxx"
	err := ValidateStripeSecrets()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "live")
}
