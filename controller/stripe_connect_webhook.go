package controller

import (
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v85/webhook"
)

// StripeConnectWebhook receives Stripe Connect events. It is a public endpoint
// (no auth) — protected by webhook signature verification against
// setting.StripeConnectWebhookSecret.
//
// POST /api/stripe/connect_webhook
//
// On handler error we still return 200 so Stripe does not retry the event; the
// error is logged and reconciliation (Task 11) catches any missed updates.
func StripeConnectWebhook(c *gin.Context) {
	if !setting.StripeConnectEnabled {
		c.JSON(http.StatusOK, gin.H{"received": false})
		return
	}
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read payload"})
		return
	}
	sig := c.GetHeader("Stripe-Signature")
	// Match the v2 topup webhook: ignore API version mismatch so a newer Stripe
	// API version on the event payload does not fail signature verification.
	evt, err := webhook.ConstructEventWithOptions(payload, sig, setting.StripeConnectWebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		common.SysError("stripe_connect webhook signature verification failed: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "signature verification failed"})
		return
	}
	client := service.NewStripeConnectClient()
	// client may be nil if Stripe Connect was disabled mid-flight; handlers
	// that only touch local state (account.updated, payout.*) tolerate nil,
	// and the payout-retry handlers log+skip when ProcessStripeConnectPayout
	// rejects a nil client.
	if err := service.HandleConnectEvent(c.Request.Context(), client, &evt); err != nil {
		common.SysError(fmt.Sprintf("stripe_connect webhook handler error (type=%s): %v", evt.Type, err))
		c.JSON(http.StatusOK, gin.H{"received": true, "handler_error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}
