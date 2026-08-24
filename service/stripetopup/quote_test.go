package stripetopup

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectDefaultCurrency(t *testing.T) {
	assert.Equal(t, "cny", DetectDefaultCurrency("CN", "en-US"))
	assert.Equal(t, "cny", DetectDefaultCurrency("", "zh-CN"))
	assert.Equal(t, "cad", DetectDefaultCurrency("CA", "en"))
	assert.Equal(t, "usd", DetectDefaultCurrency("US", "en-US"))
	assert.Equal(t, "cny", DetectDefaultCurrency("", ""))
}

func TestBuildQuoteRejectsUnsupportedCurrency(t *testing.T) {
	_, err := BuildQuote(1, QuoteRequest{Currency: "eur", TierID: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestBuildQuoteRejectsBelowMinimum(t *testing.T) {
	campaign, _ := setupExactTierQuoteTest(t)
	tier := &model.TopupPromoTier{CampaignID: campaign.Id, Code: "usd-too-small", Name: "small", Currency: "usd", PaymentAmountMinor: 49, BonusAmountMinor: 49, TotalCreditAmountMinor: 98, Enabled: true}
	require.NoError(t, model.DB.Create(tier).Error)
	_, err := BuildQuote(1, QuoteRequest{Currency: "usd", TierID: tier.Id})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside configured limits")
}

func TestBuildQuoteRejectsAboveMaximum(t *testing.T) {
	campaign, _ := setupExactTierQuoteTest(t)
	tier := &model.TopupPromoTier{CampaignID: campaign.Id, Code: "usd-too-large", Name: "large", Currency: "usd", PaymentAmountMinor: 9999900, BonusAmountMinor: 100, TotalCreditAmountMinor: 10000000, Enabled: true}
	require.NoError(t, model.DB.Create(tier).Error)
	_, err := BuildQuote(1, QuoteRequest{Currency: "usd", TierID: tier.Id})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside configured limits")
}

func TestBuildQuoteCustomAmountBelowPromotionThresholdCreatesPaidOnlyQuote(t *testing.T) {
	setupExactTierQuoteTest(t)

	q, err := BuildQuote(1, QuoteRequest{Currency: "usd", AmountMinor: 700})
	require.NoError(t, err)
	assert.Zero(t, q.TierID)
	assert.Equal(t, int64(700), q.AmountMinor)
	assert.Equal(t, int64(700), q.PaidCreditAmountMinor)
	assert.Zero(t, q.PromoCreditAmountMinor)
	assert.Equal(t, int64(700), q.TotalCreditAmountMinor)
	assert.Zero(t, q.PromoQuota)
	assert.Contains(t, q.PromotionSnapshotJSON, `"applied":false`)
}

func TestBuildQuoteCustomAmountRejectsBelowCurrencyMinimum(t *testing.T) {
	setupExactTierQuoteTest(t)

	_, err := BuildQuote(1, QuoteRequest{Currency: "usd", AmountMinor: 49})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside configured limits")
}

func TestBuildQuoteAcceptsConfiguredMinorUnitMinimumForEveryCurrency(t *testing.T) {
	setupExactTierQuoteTest(t)
	minimums := map[string]int64{"cny": 500, "usd": 50, "cad": 50}
	for currency, minimumMinor := range minimums {
		t.Run(currency, func(t *testing.T) {
			quote, err := BuildQuote(1, QuoteRequest{Currency: currency, AmountMinor: minimumMinor})
			require.NoError(t, err)
			assert.Equal(t, minimumMinor, quote.PaidCreditAmountMinor)
			assert.Zero(t, quote.PromoCreditAmountMinor)
		})
	}
}

func TestBuildQuoteRejectsOneMinorUnitBelowEachCurrencyMinimum(t *testing.T) {
	setupExactTierQuoteTest(t)
	minimums := map[string]int64{"cny": 500, "usd": 50, "cad": 50}
	for currency, minimumMinor := range minimums {
		t.Run(currency, func(t *testing.T) {
			_, err := BuildQuote(1, QuoteRequest{Currency: currency, AmountMinor: minimumMinor - 1})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "outside configured limits")
		})
	}
}

func TestBuildQuoteCustomAmountIgnoresSharedPromotionBand(t *testing.T) {
	_, tiers := setupExactTierQuoteTest(t)

	q, err := BuildQuote(1, QuoteRequest{Currency: "cny", AmountMinor: tiers[1].PaymentAmountMinor})
	require.NoError(t, err)
	assert.NotEqual(t, tiers[1].Id, q.TierID)
	assert.Zero(t, q.TierID)
	assert.Zero(t, q.PromoCreditAmountMinor)
	assert.Equal(t, tiers[1].PaymentAmountMinor, q.TotalCreditAmountMinor)
	assert.Contains(t, q.PromotionSnapshotJSON, `"applied":false`)
}

func TestListTopupOffersUsesProductCheckoutPresetsForEveryCurrency(t *testing.T) {
	setupExactTierQuoteTest(t)

	cases := map[string][]int64{
		"cny": {1000, 2000, 5000, 10000, 20000, 50000},
		"usd": {200, 500, 1000, 2000, 5000, 10000, 20000, 50000},
		"cad": {200, 500, 1000, 2000, 5000, 10000, 20000, 50000},
	}
	for currency, expected := range cases {
		catalog, err := ListTopupOffers(1, currency)
		require.NoError(t, err)
		actual := make([]int64, 0, len(catalog.Offers))
		for _, offer := range catalog.Offers {
			actual = append(actual, offer.PaymentAmountMinor)
			assert.True(t, offer.Available)
		}
		assert.Equal(t, expected, actual)
	}
}

func TestBuildQuoteUSDConversion(t *testing.T) {
	campaign, _ := setupExactTierQuoteTest(t)
	tier := &model.TopupPromoTier{CampaignID: campaign.Id, Code: "usd-10", Name: "USD 10", Currency: "usd", PaymentAmountMinor: 1000, BonusAmountMinor: 1000, TotalCreditAmountMinor: 2000, Enabled: true}
	require.NoError(t, model.DB.Create(tier).Error)
	q, err := BuildQuote(1, QuoteRequest{Currency: "usd", TierID: tier.Id})
	require.NoError(t, err)
	assert.Equal(t, "usd", q.Currency)
	assert.Equal(t, int64(1000), q.AmountMinor)
	assert.Equal(t, float64(1), q.FxRateSnapshot)
	assert.Equal(t, int64(10)*setting.MicroUSDPerUSD, q.PaidCreditMicroUSD)
	assert.Greater(t, q.PaidQuota, 0)
	assert.Equal(t, q.PaidQuota, q.TotalQuota)
	assert.Zero(t, q.PromoCreditMicroUSD)
}

func TestBuildQuoteCNYAndCAD(t *testing.T) {
	campaign, tiers := setupExactTierQuoteTest(t)
	cny, err := BuildQuote(1, QuoteRequest{Currency: "cny", TierID: tiers[2].Id})
	require.NoError(t, err)
	assert.Equal(t, int64(5000), cny.AmountMinor)
	assert.InDelta(t, setting.BillingCurrencyFXRate("cny"), cny.FxRateSnapshot, 0.0001)
	assert.Greater(t, cny.PaidCreditMicroUSD, int64(0))

	cadTier := &model.TopupPromoTier{CampaignID: campaign.Id, Code: "cad-20", Name: "CAD 20", Currency: "cad", PaymentAmountMinor: 2000, BonusAmountMinor: 2000, TotalCreditAmountMinor: 4000, Enabled: true}
	require.NoError(t, model.DB.Create(cadTier).Error)
	cad, err := BuildQuote(1, QuoteRequest{Currency: "cad", TierID: cadTier.Id})
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

func TestValidateStripeSecretsRejectsTestResourcesInProduction(t *testing.T) {
	originalSecret := setting.StripeApiSecret
	originalPublishable := setting.StripePublishableKey
	originalWebhook := setting.StripeWebhookSecret
	originalProduct := setting.StripeTopupProductID
	originalAccount := setting.StripeAccountID
	originalRequireTest := setting.StripeRequireTestKeys
	t.Cleanup(func() {
		setting.StripeApiSecret = originalSecret
		setting.StripePublishableKey = originalPublishable
		setting.StripeWebhookSecret = originalWebhook
		setting.StripeTopupProductID = originalProduct
		setting.StripeAccountID = originalAccount
		setting.StripeRequireTestKeys = originalRequireTest
	})

	setting.StripeRequireTestKeys = false
	setting.StripeApiSecret = "sk_test_mixed"
	setting.StripePublishableKey = "pk_test_mixed"
	setting.StripeWebhookSecret = "whsec_configured"
	setting.StripeTopupProductID = "prod_test"
	setting.StripeAccountID = "acct_test"

	err := ValidateStripeSecrets()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "production")
}

func TestValidateStripeSecretsRequiresCompleteActiveProfile(t *testing.T) {
	originalSecret := setting.StripeApiSecret
	originalPublishable := setting.StripePublishableKey
	originalWebhook := setting.StripeWebhookSecret
	originalProduct := setting.StripeTopupProductID
	originalAccount := setting.StripeAccountID
	originalRequireTest := setting.StripeRequireTestKeys
	t.Cleanup(func() {
		setting.StripeApiSecret = originalSecret
		setting.StripePublishableKey = originalPublishable
		setting.StripeWebhookSecret = originalWebhook
		setting.StripeTopupProductID = originalProduct
		setting.StripeAccountID = originalAccount
		setting.StripeRequireTestKeys = originalRequireTest
	})

	setting.StripeRequireTestKeys = true
	setting.StripeApiSecret = "sk_test_configured"
	setting.StripePublishableKey = ""
	setting.StripeWebhookSecret = "whsec_configured"
	setting.StripeTopupProductID = "prod_test"
	setting.StripeAccountID = "acct_test"

	err := ValidateStripeSecrets()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publishable")
}
