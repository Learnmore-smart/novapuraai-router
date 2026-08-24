package stripetopup

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupExactTierQuoteTest(t *testing.T) (*model.TopupPromotionCampaign, []model.TopupPromoTier) {
	t.Helper()
	originalDB := model.DB
	originalCurrencyConfig := setting.BillingCurrencyConfigJSON()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:quote-tier-%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB = originalDB
		require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(originalCurrencyConfig))
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&model.TopupPromotionCampaign{}, &model.TopupPromoTier{}, &model.TopupPromoRedemption{}))
	model.DB = db
	require.NoError(t, model.SeedLaunchTopupPromotion(db))
	campaign, err := model.GetTopupPromotionCampaign()
	require.NoError(t, err)
	var tiers []model.TopupPromoTier
	require.NoError(t, db.Where("campaign_id = ? AND payment_amount_minor > ?", campaign.Id, 0).Order("sort_order asc").Find(&tiers).Error)
	return campaign, tiers
}

func TestBuildQuoteCreditsOneToOneAcrossFormerPromotionBoundaries(t *testing.T) {
	setupExactTierQuoteTest(t)

	tests := []struct {
		currency     string
		paymentMinor int64
	}{
		{currency: "cny", paymentMinor: 999},
		{currency: "cny", paymentMinor: 1000},
		{currency: "usd", paymentMinor: 1999},
		{currency: "usd", paymentMinor: 2000},
		{currency: "cad", paymentMinor: 4999},
		{currency: "cad", paymentMinor: 5000},
		{currency: "cny", paymentMinor: 9999},
		{currency: "usd", paymentMinor: 10000},
		{currency: "cad", paymentMinor: 19999},
		{currency: "cny", paymentMinor: 20000},
		{currency: "usd", paymentMinor: 49999},
		{currency: "cad", paymentMinor: 50000},
	}

	for _, test := range tests {
		name := fmt.Sprintf("%s-%d", test.currency, test.paymentMinor)
		t.Run(name, func(t *testing.T) {
			quote, err := BuildQuote(42, QuoteRequest{Currency: test.currency, AmountMinor: test.paymentMinor})
			require.NoError(t, err)
			assert.Equal(t, test.paymentMinor, quote.PaidCreditAmountMinor)
			assert.Zero(t, quote.PromoCreditAmountMinor)
			assert.Equal(t, test.paymentMinor, quote.TotalCreditAmountMinor)
			assert.Zero(t, quote.PromotionTierID)
		})
	}
}

func TestBuildQuoteUsesExactTierAndIgnoresClientAmount(t *testing.T) {
	_, tiers := setupExactTierQuoteTest(t)
	require.NotEmpty(t, tiers)

	quote, err := BuildQuote(42, QuoteRequest{Currency: "cny", TierID: tiers[0].Id, AmountMajor: 99999})
	require.NoError(t, err)
	assert.Zero(t, quote.TierID)
	assert.Equal(t, int64(1000), quote.AmountMinor)
	assert.Equal(t, int64(1000), quote.PaidCreditAmountMinor)
	assert.Zero(t, quote.PromoCreditAmountMinor)
	assert.Equal(t, int64(1000), quote.TotalCreditAmountMinor)
	assert.Equal(t, "¥10.00", quote.PaymentDisplay)
	assert.Equal(t, "¥0.00", quote.BonusDisplay)
	assert.Equal(t, "¥10.00", quote.TotalDisplay)
	assert.Zero(t, quote.PromoExpiryDays)
}

func TestBuildQuoteRejectsDisabledCampaignTierAndCurrency(t *testing.T) {
	campaign, tiers := setupExactTierQuoteTest(t)
	tier := tiers[0]

	tier.Enabled = false
	require.NoError(t, model.DB.Save(&tier).Error)
	_, err := BuildQuote(1, QuoteRequest{Currency: "cny", TierID: tier.Id})
	require.ErrorIs(t, err, model.ErrTopupPromotionUnavailable)

	tier.Enabled = true
	require.NoError(t, model.DB.Save(&tier).Error)
	campaign.Enabled = false
	require.NoError(t, model.DB.Save(campaign).Error)
	_, err = BuildQuote(1, QuoteRequest{Currency: "cny", TierID: tier.Id})
	require.ErrorIs(t, err, model.ErrTopupPromotionUnavailable)

	require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(`{"default_currency":"usd","currencies":{"cny":{"enabled":false,"fx_presentment_per_usd":7.3},"usd":{"enabled":true,"fx_presentment_per_usd":1},"cad":{"enabled":true,"fx_presentment_per_usd":1.37}}}`))
	_, err = BuildQuote(1, QuoteRequest{Currency: "cny", TierID: tier.Id})
	require.ErrorContains(t, err, "unsupported currency")
}

func TestBuildQuoteIgnoresLegacyPromotionRedemptionLimits(t *testing.T) {
	campaign, tiers := setupExactTierQuoteTest(t)
	tier := tiers[0]

	for i := 0; i < 3; i++ {
		require.NoError(t, model.ReserveTopupPromotion(fmt.Sprintf("repeat-%d", i), 9, tier.Id, 2_000_000))
	}
	_, err := BuildQuote(9, QuoteRequest{Currency: "cny", TierID: tier.Id})
	require.NoError(t, err)

	campaign.PerUserLimit = 3
	require.NoError(t, model.DB.Save(campaign).Error)
	quote, err := BuildQuote(9, QuoteRequest{Currency: "cny", TierID: tier.Id})
	require.NoError(t, err)
	assert.Zero(t, quote.PromoCreditAmountMinor)
}

func TestBuildQuoteCreditsExactlyWhatTheUserPaysDespiteLegacyPromotionRows(t *testing.T) {
	setupExactTierQuoteTest(t)

	quote, err := BuildQuote(42, QuoteRequest{Currency: "cny", AmountMinor: 10_000})
	require.NoError(t, err)
	assert.Equal(t, int64(10_000), quote.PaidCreditAmountMinor)
	assert.Zero(t, quote.PromoCreditAmountMinor)
	assert.Equal(t, int64(10_000), quote.TotalCreditAmountMinor)
	assert.Equal(t, quote.PaidQuota, quote.TotalQuota)
	assert.Zero(t, quote.PromotionTierID)
	assert.Equal(t, "¥100.00", quote.PaymentDisplay)
	assert.Equal(t, "¥100.00", quote.TotalDisplay)
}
