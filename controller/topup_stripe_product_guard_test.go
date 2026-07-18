package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLegacyStripeTopupEndpointsAreDisabledByProductCheckout(t *testing.T) {
	original := setting.StripeTopupEnabled
	setting.StripeTopupEnabled = true
	t.Cleanup(func() { setting.StripeTopupEnabled = original })

	for _, invoke := range []func(*gin.Context){
		func(ctx *gin.Context) { stripeAdaptor.RequestAmount(ctx, &StripePayRequest{Amount: 10}) },
		func(ctx *gin.Context) {
			stripeAdaptor.RequestPay(ctx, &StripePayRequest{Amount: 10, PaymentMethod: "stripe"})
		},
	} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		invoke(ctx)
		assert.Equal(t, http.StatusConflict, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "Product Checkout")
	}
}
