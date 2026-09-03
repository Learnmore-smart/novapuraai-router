package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v85/webhook"
	"gorm.io/gorm"
)

func postSignedStripeWebhook(t *testing.T, payload []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload: payload,
		Secret:  setting.StripeWebhookSecret,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/stripe/webhook", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signed.Header)
	recorder := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/stripe/webhook", StripeWebhookV2)
	router.ServeHTTP(recorder, req)
	return recorder
}

func setupStripeWebhookController(t *testing.T) *gorm.DB {
	t.Helper()
	originalAPISecret := setting.StripeApiSecret
	originalWebhookSecret := setting.StripeWebhookSecret
	originalPriceID := setting.StripePriceId
	originalProductID := setting.StripeTopupProductID
	originalTopupEnabled := setting.StripeTopupEnabled
	originalAccountID := setting.StripeAccountID
	originalRequireTest := setting.StripeRequireTestKeys
	originalRuntime := setting.StripeRuntimeEnvironment
	originalDB := model.DB
	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		setting.StripeApiSecret = originalAPISecret
		setting.StripeWebhookSecret = originalWebhookSecret
		setting.StripePriceId = originalPriceID
		setting.StripeTopupProductID = originalProductID
		setting.StripeTopupEnabled = originalTopupEnabled
		setting.StripeAccountID = originalAccountID
		setting.StripeRequireTestKeys = originalRequireTest
		setting.StripeRuntimeEnvironment = originalRuntime
		model.DB = originalDB
		common.RedisEnabled = originalRedisEnabled
	})
	setting.StripeApiSecret = "sk_test_controller"
	setting.StripeWebhookSecret = "whsec_controller"
	setting.StripePriceId = ""
	setting.StripeTopupProductID = "prod_controller"
	setting.StripeTopupEnabled = true
	setting.StripeAccountID = ""
	setting.StripeRequireTestKeys = true
	setting.StripeRuntimeEnvironment = setting.StripeRuntimeTest
	common.RedisEnabled = false
	t.Setenv("GIN_MODE", "debug")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.StripeWebhookEvent{},
		&model.SubscriptionPlan{},
		&model.StripeSubscription{},
		&model.StripeSubscriptionReservation{},
		&model.StripeSubscriptionInvoice{},
		&model.User{},
		&model.UserSubscription{},
	))
	model.DB = db
	return db
}

func TestStripeWebhookReturnsRetryableStatusWhenProcessingFails(t *testing.T) {
	setupStripeWebhookController(t)

	payload := []byte(`{"id":"evt_http_retry","object":"event","type":"checkout.session.completed","livemode":false,"data":{"object":{"id":"cs_missing","object":"checkout.session","client_reference_id":"np_missing","status":"complete","payment_status":"paid","amount_total":1000,"currency":"usd"}}}`)
	recorder := postSignedStripeWebhook(t, payload)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	var count int64
	require.NoError(t, model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", "evt_http_retry").Count(&count).Error)
	assert.Zero(t, count)
}

func TestStripeWebhookAcknowledgesCheckoutWithoutNovaPuraOrderID(t *testing.T) {
	setupStripeWebhookController(t)

	payload := []byte(`{"id":"evt_unrelated_checkout","object":"event","type":"checkout.session.completed","livemode":false,"data":{"object":{"id":"cs_unrelated","object":"checkout.session","status":"complete","payment_status":"paid","amount_total":1000,"currency":"usd"}}}`)
	recorder := postSignedStripeWebhook(t, payload)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestStripeWebhookAcknowledgesLivemodeEventUnderSandboxPolicy(t *testing.T) {
	setupStripeWebhookController(t)

	payload := []byte(`{"id":"evt_live_sandbox","object":"event","type":"checkout.session.completed","livemode":true,"data":{"object":{"id":"cs_live","object":"checkout.session","client_reference_id":"np_live","status":"complete","payment_status":"paid","amount_total":1000,"currency":"usd"}}}`)
	recorder := postSignedStripeWebhook(t, payload)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestStripeWebhookAcknowledgesEventsWhenStripeFeaturesAreOff(t *testing.T) {
	setupStripeWebhookController(t)
	setting.StripeTopupEnabled = false
	setting.StripeTopupProductID = ""
	setting.StripePriceId = ""

	payload := []byte(`{"id":"evt_disabled_features","object":"event","type":"payout.paid","livemode":false,"data":{"object":{"id":"po_disabled","object":"payout"}}}`)
	recorder := postSignedStripeWebhook(t, payload)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestStripeWebhookAcknowledgesForeignInvoiceWhenCatalogIsInvalid(t *testing.T) {
	db := setupStripeWebhookController(t)
	recurringCode := model.SandboxStripeSubscriptionPlanCode
	require.NoError(t, db.Create(&model.SubscriptionPlan{
		Code:                      model.SandboxStripeSubscriptionPlanCode,
		RecurringCode:             &recurringCode,
		Title:                     "invalid catalog row",
		PriceAmount:               19.99,
		Currency:                  "CNY",
		DurationUnit:              model.SubscriptionDurationMonth,
		DurationValue:             1,
		Enabled:                   true,
		StripeSubscriptionEnabled: true,
		StripeProductId:           "prod_wrong_catalog",
		StripeAccountId:           model.SandboxStripeSubscriptionAccountID,
		FounderStripePriceId:      model.SandboxStripeSubscriptionFounderPriceID,
		StandardStripePriceId:     model.SandboxStripeSubscriptionStandardPriceID,
		FounderAmountMinor:        1999,
		StandardAmountMinor:       9999,
		StripeCurrency:            "cny",
	}).Error)

	payload := []byte(`{"id":"evt_foreign_invoice","object":"event","type":"invoice.paid","livemode":false,"data":{"object":{"id":"in_foreign","object":"invoice","subscription":"sub_foreign","amount_paid":1999,"currency":"cny"}}}`)
	recorder := postSignedStripeWebhook(t, payload)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestStripeWebhookAcknowledgesForeignInvoiceInsteadOfRetrying(t *testing.T) {
	db := setupStripeWebhookController(t)
	setting.StripeAccountID = model.SandboxStripeSubscriptionAccountID
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "true")
	t.Cleanup(func() {
		_ = os.Unsetenv("STRIPE_SUBSCRIPTION_ENABLED")
	})
	recurringCode := model.SandboxStripeSubscriptionPlanCode
	require.NoError(t, db.Create(&model.SubscriptionPlan{
		Code:                        model.SandboxStripeSubscriptionPlanCode,
		RecurringCode:               &recurringCode,
		Title:                       "NovaPuraAI DeepSeek V4 Flash Unlimited",
		PriceAmount:                 19.99,
		Currency:                    "CNY",
		DurationUnit:                model.SubscriptionDurationMonth,
		DurationValue:               1,
		Enabled:                     true,
		StripeSubscriptionEnabled:   true,
		StripeSubscriptionModel:     model.SandboxStripeSubscriptionModel,
		UpgradeGroup:                model.SandboxStripeSubscriptionGroup,
		MaxActiveSubscriptions:      20,
		FounderPurchaseLimit:        20,
		MaxActivePerUser:            1,
		FounderStripePriceId:        model.SandboxStripeSubscriptionFounderPriceID,
		StandardStripePriceId:       model.SandboxStripeSubscriptionStandardPriceID,
		StripeProductId:             model.SandboxStripeSubscriptionProductID,
		StripeAccountId:             model.SandboxStripeSubscriptionAccountID,
		StripePortalConfigurationId: model.SandboxStripeSubscriptionPortalConfigurationID,
		FounderAmountMinor:          1999,
		StandardAmountMinor:         9999,
		StripeCurrency:              "cny",
		AllowBalancePay:             common.GetPointer(false),
		AllowWalletOverflow:         common.GetPointer(false),
	}).Error)

	payload := []byte(`{"id":"evt_foreign_sub_invoice","object":"event","type":"invoice.paid","livemode":false,"data":{"object":{"id":"in_foreign_sub","object":"invoice","subscription":"sub_not_local","amount_paid":1999,"currency":"cny"}}}`)
	recorder := postSignedStripeWebhook(t, payload)

	assert.Equal(t, http.StatusOK, recorder.Code)
}
