package stripetopup

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateReservedOrderSnapshotsTierAndReleasesFailedCheckout(t *testing.T) {
	campaign, tiers := setupExactTierQuoteTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.User{}, &model.StripeTopupOrder{}))
	user := &model.User{Username: "reserved-order", Email: "reserved@example.com"}
	require.NoError(t, model.DB.Create(user).Error)
	quote, err := BuildQuote(user.Id, QuoteRequest{Currency: "cny", TierID: tiers[0].Id})
	require.NoError(t, err)

	order, err := createReservedOrder(user, quote, "np_reserved", "idem_reserved")
	require.NoError(t, err)
	assert.Equal(t, quote.TierID, order.PromotionTierID)
	assert.Equal(t, quote.PaidCreditAmountMinor, order.PaidCreditAmountMinor)
	assert.Equal(t, quote.PromoCreditAmountMinor, order.PromoCreditAmountMinor)
	assert.Equal(t, quote.TotalCreditAmountMinor, order.TotalCreditAmountMinor)
	assert.Equal(t, quote.PromoExpiryDays, order.PromoExpiryDays)

	var redemption model.TopupPromoRedemption
	require.NoError(t, model.DB.Where("order_id = ?", order.OrderID).First(&redemption).Error)
	assert.Equal(t, model.TopupPromoRedemptionReserved, redemption.Status)
	require.NoError(t, model.DB.First(campaign, campaign.Id).Error)
	assert.Equal(t, quote.PromoCreditMicroUSD, campaign.ReservedPromoMicroUSD)

	require.NoError(t, model.MarkStripeTopupOrderStatus(order.OrderID, "*", model.StripeOrderFailed, "checkout creation failed"))
	require.NoError(t, model.DB.Where("order_id = ?", order.OrderID).First(&redemption).Error)
	assert.Equal(t, model.TopupPromoRedemptionReleased, redemption.Status)
	require.NoError(t, model.DB.First(campaign, campaign.Id).Error)
	assert.Zero(t, campaign.ReservedPromoMicroUSD)
}
