package stripetopup

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestReconcileRefundReversesOnlySourceOrderLotsAndIsIdempotent(t *testing.T) {
	setupStripeTestDB(t)
	user := &model.User{Username: "refund-lots"}
	require.NoError(t, model.DB.Create(user).Error)
	campaign := &model.TopupPromotionCampaign{Id: 1, Name: "refund", Enabled: true, DefaultPromoExpiryDays: 30}
	require.NoError(t, model.DB.Create(campaign).Error)
	tier := &model.TopupPromoTier{CampaignID: 1, Code: "refund-tier", Name: "refund-tier", Currency: "cny", PaymentAmountMinor: 1000, BonusAmountMinor: 2000, TotalCreditAmountMinor: 3000, Enabled: true}
	require.NoError(t, model.DB.Create(tier).Error)
	order := &model.StripeTopupOrder{
		OrderID:                "refund-source-order",
		UserId:                 user.Id,
		Status:                 model.StripeOrderPaid,
		PresentmentCurrency:    "cny",
		PresentmentAmountMinor: 1000,
		PaidCreditAmountMinor:  1000,
		PromoCreditAmountMinor: 2000,
		TotalCreditAmountMinor: 3000,
		FxRateSnapshot:         7.3,
		PaidCreditMicroUSD:     1_000_000,
		PromoCreditMicroUSD:    2_000_000,
		TotalCreditMicroUSD:    3_000_000,
		PaidQuota:              100,
		PromoQuota:             200,
		PromotionTierID:        tier.Id,
		PromoExpiryDays:        30,
		PromotionSnapshotJSON:  `{"applied":true}`,
	}
	require.NoError(t, model.DB.Create(order).Error)
	require.NoError(t, model.ReserveTopupPromotion(order.OrderID, user.Id, tier.Id, order.PromoCreditMicroUSD))
	_, err := model.CreditStripeTopupOrder(order.OrderID, "cus_refund", "pi_refund", "cs_refund")
	require.NoError(t, err)

	now := time.Now().Unix()
	unrelated := &model.BalanceCreditLot{UserId: user.Id, OrderID: "unrelated-promo", BalanceType: model.BalanceTypePromotional, OriginalQuota: 50, RemainingQuota: 50, Currency: "cny", ExpiresAt: 0, Status: model.CreditLotActive, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, model.DB.Create(unrelated).Error)
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"quota":       gorm.Expr("quota + ?", 50),
		"promo_quota": gorm.Expr("promo_quota + ?", 50),
	}).Error)
	_, err = model.DecreaseUserQuotaWithSplit(user.Id, 50, true)
	require.NoError(t, err)

	require.NoError(t, ReconcileRefund(context.Background(), order.OrderID))
	require.NoError(t, ReconcileRefund(context.Background(), order.OrderID))

	var refreshed model.User
	require.NoError(t, model.DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, 50, refreshed.Quota)
	assert.Equal(t, 50, refreshed.PromoQuota)
	require.NoError(t, model.DB.First(unrelated, unrelated.Id).Error)
	assert.Equal(t, 50, unrelated.RemainingQuota)

	var sourceLots []model.BalanceCreditLot
	require.NoError(t, model.DB.Where("order_id = ?", order.OrderID).Find(&sourceLots).Error)
	require.Len(t, sourceLots, 2)
	for _, lot := range sourceLots {
		assert.Zero(t, lot.RemainingQuota)
		assert.Equal(t, model.CreditLotReversed, lot.Status)
	}
	var updatedOrder model.StripeTopupOrder
	require.NoError(t, model.DB.Where("order_id = ?", order.OrderID).First(&updatedOrder).Error)
	assert.Equal(t, model.StripeOrderRefunded, updatedOrder.Status)
	var redemption model.TopupPromoRedemption
	require.NoError(t, model.DB.Where("order_id = ?", order.OrderID).First(&redemption).Error)
	assert.Equal(t, model.TopupPromoRedemptionReversed, redemption.Status)
	require.NoError(t, model.DB.First(campaign, campaign.Id).Error)
	assert.Zero(t, campaign.IssuedPromoMicroUSD)

	var refundEntries int64
	require.NoError(t, model.DB.Model(&model.BalanceLedger{}).Where("order_id = ? AND entry_type IN ?", order.OrderID, []string{model.LedgerTypeRefundPaid, model.LedgerTypeRefundPromo}).Count(&refundEntries).Error)
	assert.Equal(t, int64(2), refundEntries)
}
