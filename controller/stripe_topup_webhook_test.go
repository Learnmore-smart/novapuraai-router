package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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

func TestStripeWebhookReturnsRetryableStatusWhenProcessingFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalAPISecret := setting.StripeApiSecret
	originalWebhookSecret := setting.StripeWebhookSecret
	originalPriceID := setting.StripePriceId
	originalProductID := setting.StripeTopupProductID
	originalTopupEnabled := setting.StripeTopupEnabled
	originalAccountID := setting.StripeAccountID
	originalRequireTest := setting.StripeRequireTestKeys
	t.Cleanup(func() {
		setting.StripeApiSecret = originalAPISecret
		setting.StripeWebhookSecret = originalWebhookSecret
		setting.StripePriceId = originalPriceID
		setting.StripeTopupProductID = originalProductID
		setting.StripeTopupEnabled = originalTopupEnabled
		setting.StripeAccountID = originalAccountID
		setting.StripeRequireTestKeys = originalRequireTest
	})
	setting.StripeApiSecret = "sk_test_controller"
	setting.StripeWebhookSecret = "whsec_controller"
	setting.StripePriceId = ""
	setting.StripeTopupProductID = "prod_controller"
	setting.StripeTopupEnabled = true
	setting.StripeAccountID = ""
	setting.StripeRequireTestKeys = true

	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.StripeWebhookEvent{}))
	model.DB = db

	payload := []byte(`{"id":"evt_http_retry","object":"event","type":"checkout.session.completed","livemode":false,"data":{"object":{"id":"cs_missing","object":"checkout.session","client_reference_id":"np_missing","status":"complete","payment_status":"paid","amount_total":1000,"currency":"usd"}}}`)
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

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	var count int64
	require.NoError(t, model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", "evt_http_retry").Count(&count).Error)
	assert.Zero(t, count)
}
