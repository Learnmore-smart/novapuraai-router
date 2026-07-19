package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLegacyStripeTopupEndpointsAreDisabledByProductCheckout(t *testing.T) {
	// Initialize i18n so ApiErrorI18n returns the translated message instead
	// of the raw key. In production this is done by main.go on startup.
	if err := i18n.Init(); err != nil {
		t.Fatalf("i18n.Init() failed: %v", err)
	}

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
		// ApiErrorI18n -> GetLangFromContext reads c.GetHeader, which needs a
		// real *http.Request. gin.CreateTestContext leaves ctx.Request nil.
		ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		invoke(ctx)
		// HTTP 200 + success:false so the frontend's skipBusinessError flag
		// suppresses the global axios error toast (a 409 bypassed it and
		// surfaced a literal "error" toast on every wallet page load).
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `"success":false`)
		// Message is i18n-ized; the test context has no language set so the
		// default English translation is returned.
		assert.Contains(t, recorder.Body.String(), "Product Checkout")
	}
}
