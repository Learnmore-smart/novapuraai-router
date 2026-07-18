package stripetopup

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v85"
	"gorm.io/gorm"
)

func checkoutCompletedEvent(eventID, orderID string, includeAmount bool) stripe.Event {
	payload := map[string]any{
		"id":                  "cs_" + eventID,
		"object":              "checkout.session",
		"client_reference_id": orderID,
		"status":              "complete",
		"payment_status":      "paid",
		"currency":            "usd",
		"customer":            "cus_" + eventID,
		"payment_intent":      "pi_" + eventID,
		"metadata": map[string]string{
			"novapura_order_id": orderID,
		},
	}
	if includeAmount {
		payload["amount_total"] = 1000
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return stripe.Event{
		ID:       eventID,
		Livemode: false,
		Type:     stripe.EventTypeCheckoutSessionCompleted,
		Data: &stripe.EventData{
			Raw:    json.RawMessage(raw),
			Object: payload,
		},
	}
}

func setupStripeTestDB(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.RedisEnabled = originalRedisEnabled
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.StripeTopupOrder{},
		&model.BalanceLedger{},
		&model.BalanceCreditLot{},
		&model.StripeWebhookEvent{},
		&model.TopupPromoTier{},
		&model.TopupPromotionCampaign{},
		&model.TopupPromoRedemption{},
		&model.TopUp{},
		&model.Log{},
	))
	model.DB = db
	model.LOG_DB = db
}

func TestWebhookDuplicateEventDoesNotDoubleCredit(t *testing.T) {
	setupStripeTestDB(t)
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = "acct_1Tta8mPKe8UWYDw1"

	user := &model.User{Username: "u1", Password: "password123", Quota: 0, PromoQuota: 0}
	require.NoError(t, model.DB.Create(user).Error)

	order := &model.StripeTopupOrder{
		OrderID:                "np_test_order_1",
		UserId:                 user.Id,
		Status:                 model.StripeOrderCheckoutCreated,
		PresentmentCurrency:    "usd",
		PresentmentAmountMinor: 1000,
		FxRateSnapshot:         1,
		PaidCreditMicroUSD:     10_000_000,
		PromoCreditMicroUSD:    0,
		TotalCreditMicroUSD:    10_000_000,
		PaidQuota:              5_000_000,
		PromoQuota:             0,
	}
	require.NoError(t, model.DB.Create(order).Error)

	// Simulate process: first credit via model (as webhook would)
	already, err := model.CreditStripeTopupOrder(order.OrderID, "cus_x", "pi_x", "cs_x")
	require.NoError(t, err)
	assert.False(t, already)

	var u1 model.User
	require.NoError(t, model.DB.First(&u1, user.Id).Error)
	q1 := u1.Quota
	assert.Greater(t, q1, 0)

	// Second credit same order — idempotent
	already2, err := model.CreditStripeTopupOrder(order.OrderID, "cus_x", "pi_x", "cs_x")
	require.NoError(t, err)
	assert.True(t, already2)

	var u2 model.User
	require.NoError(t, model.DB.First(&u2, user.Id).Error)
	assert.Equal(t, q1, u2.Quota, "duplicate credit must not increase balance")

	// Webhook event table uniqueness
	ev := &model.StripeWebhookEvent{EventID: "evt_dup_1", EventType: "checkout.session.completed", CreatedAt: 1}
	ins1, err := model.TryInsertStripeWebhookEvent(ev)
	require.NoError(t, err)
	assert.True(t, ins1)
	ins2, err := model.TryInsertStripeWebhookEvent(&model.StripeWebhookEvent{EventID: "evt_dup_1", EventType: "checkout.session.completed", CreatedAt: 2})
	require.NoError(t, err)
	assert.False(t, ins2)

	// ProcessVerifiedEvent with livemode=true under test policy fails before credit
	err = ProcessVerifiedEvent(context.Background(), stripe.Event{ID: "evt_live", Livemode: true, Type: stripe.EventTypeCheckoutSessionCompleted})
	require.Error(t, err)
}

func TestWebhookDuplicateEventIssuesExactlyOnePromotionalBonus(t *testing.T) {
	setupStripeTestDB(t)
	originalRequireTest := setting.StripeRequireTestKeys
	originalAccountID := setting.StripeAccountID
	t.Cleanup(func() {
		setting.StripeRequireTestKeys = originalRequireTest
		setting.StripeAccountID = originalAccountID
	})
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	user := &model.User{Username: "promo-webhook-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)
	campaign := &model.TopupPromotionCampaign{Id: 1, Name: "webhook promo", Enabled: true, DefaultPromoExpiryDays: 30}
	require.NoError(t, model.DB.Create(campaign).Error)
	tier := &model.TopupPromoTier{CampaignID: campaign.Id, Code: "webhook-usd-10", Name: "USD 10", Currency: "usd", PaymentAmountMinor: 1000, BonusAmountMinor: 2000, TotalCreditAmountMinor: 3000, Enabled: true}
	require.NoError(t, model.DB.Create(tier).Error)
	order := &model.StripeTopupOrder{
		OrderID:                "np_webhook_promo",
		UserId:                 user.Id,
		Status:                 model.StripeOrderCheckoutCreated,
		PresentmentCurrency:    "usd",
		PresentmentAmountMinor: 1000,
		PaidCreditAmountMinor:  1000,
		PromoCreditAmountMinor: 2000,
		TotalCreditAmountMinor: 3000,
		FxRateSnapshot:         1,
		PaidCreditMicroUSD:     10_000_000,
		PromoCreditMicroUSD:    20_000_000,
		TotalCreditMicroUSD:    30_000_000,
		PaidQuota:              5_000_000,
		PromoQuota:             10_000_000,
		PromotionTierID:        tier.Id,
		PromotionSnapshotJSON:  `{"applied":true}`,
		PromoExpiryDays:        30,
	}
	require.NoError(t, model.DB.Create(order).Error)
	require.NoError(t, model.ReserveTopupPromotion(order.OrderID, user.Id, tier.Id, order.PromoCreditMicroUSD))

	event := checkoutCompletedEvent("evt_promo_once", order.OrderID, true)
	require.NoError(t, ProcessVerifiedEvent(context.Background(), event))
	require.NoError(t, ProcessVerifiedEvent(context.Background(), event))

	var refreshed model.User
	require.NoError(t, model.DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, 15_000_000, refreshed.Quota)
	assert.Equal(t, 10_000_000, refreshed.PromoQuota)
	var promoLots int64
	require.NoError(t, model.DB.Model(&model.BalanceCreditLot{}).Where("order_id = ? AND balance_type = ?", order.OrderID, model.BalanceTypePromotional).Count(&promoLots).Error)
	assert.EqualValues(t, 1, promoLots)
	var promoEntries int64
	require.NoError(t, model.DB.Model(&model.BalanceLedger{}).Where("order_id = ? AND entry_type = ?", order.OrderID, model.LedgerTypeTopupPromotionalBonus).Count(&promoEntries).Error)
	assert.EqualValues(t, 1, promoEntries)
	var redemption model.TopupPromoRedemption
	require.NoError(t, model.DB.Where("order_id = ?", order.OrderID).First(&redemption).Error)
	assert.Equal(t, model.TopupPromoRedemptionIssued, redemption.Status)
}

func TestWebhookProcessingFailureCanBeRetried(t *testing.T) {
	setupStripeTestDB(t)
	originalRequireTest := setting.StripeRequireTestKeys
	originalAccountID := setting.StripeAccountID
	t.Cleanup(func() {
		setting.StripeRequireTestKeys = originalRequireTest
		setting.StripeAccountID = originalAccountID
	})
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	event := checkoutCompletedEvent("evt_retry", "np_retry_order", true)
	require.ErrorContains(t, ProcessVerifiedEvent(context.Background(), event), "order not found")

	var claimed int64
	require.NoError(t, model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", event.ID).Count(&claimed).Error)
	assert.Zero(t, claimed, "failed processing must release the event claim so Stripe can retry")

	user := &model.User{Username: "retry-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)
	require.NoError(t, model.DB.Create(&model.StripeTopupOrder{
		OrderID:                "np_retry_order",
		UserId:                 user.Id,
		Status:                 model.StripeOrderCheckoutCreated,
		PresentmentCurrency:    "usd",
		PresentmentAmountMinor: 1000,
		FxRateSnapshot:         1,
		PaidCreditMicroUSD:     10_000_000,
		TotalCreditMicroUSD:    10_000_000,
		PaidQuota:              5_000_000,
	}).Error)

	require.NoError(t, ProcessVerifiedEvent(context.Background(), event))
	var updated model.User
	require.NoError(t, model.DB.First(&updated, user.Id).Error)
	assert.Equal(t, 5_000_000, updated.Quota)
	require.NoError(t, model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", event.ID).Count(&claimed).Error)
	assert.EqualValues(t, 1, claimed)
}

func TestWebhookRejectsMissingPaymentAmount(t *testing.T) {
	setupStripeTestDB(t)
	originalRequireTest := setting.StripeRequireTestKeys
	originalAccountID := setting.StripeAccountID
	t.Cleanup(func() {
		setting.StripeRequireTestKeys = originalRequireTest
		setting.StripeAccountID = originalAccountID
	})
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	user := &model.User{Username: "amount-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)
	require.NoError(t, model.DB.Create(&model.StripeTopupOrder{
		OrderID:                "np_missing_amount",
		UserId:                 user.Id,
		Status:                 model.StripeOrderCheckoutCreated,
		PresentmentCurrency:    "usd",
		PresentmentAmountMinor: 1000,
		FxRateSnapshot:         1,
		PaidCreditMicroUSD:     10_000_000,
		TotalCreditMicroUSD:    10_000_000,
		PaidQuota:              5_000_000,
	}).Error)

	err := ProcessVerifiedEvent(context.Background(), checkoutCompletedEvent("evt_missing_amount", "np_missing_amount", false))
	require.ErrorContains(t, err, "amount")

	var updated model.User
	require.NoError(t, model.DB.First(&updated, user.Id).Error)
	assert.Zero(t, updated.Quota)
	var order model.StripeTopupOrder
	require.NoError(t, model.DB.Where("order_id = ?", "np_missing_amount").First(&order).Error)
	assert.Equal(t, model.StripeOrderManualReview, order.Status)
}
