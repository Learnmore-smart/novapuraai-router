package controller

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/stripetopup"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v85/webhook"
)

// GetBillingTopupConfig returns presets, limits, FX (no secrets), and default currency hint.
func GetBillingTopupConfig(c *gin.Context) {
	country := pickFirstNonEmpty(
		c.GetHeader("CF-IPCountry"),
		c.GetHeader("X-Vercel-IP-Country"),
		c.GetHeader("CloudFront-Viewer-Country"),
	)
	locale := pickFirstNonEmpty(c.GetHeader("Accept-Language"), c.Query("locale"))
	if i := strings.Index(locale, ","); i > 0 {
		locale = locale[:i]
	}
	def := stripetopup.DetectDefaultCurrency(country, locale)

	// Optional user preference from setting JSON field is left to frontend localStorage;
	// if logged in we can still return detected default.
	userId := c.GetInt("id")
	bal := 0
	promo := 0
	if userId > 0 {
		if u, err := model.GetUserById(userId, false); err == nil && u != nil {
			bal = u.Quota
			promo = u.PromoQuota
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"config":           setting.ExportTopupConfigJSON(),
			"default_currency": def,
			"country_hint":     country,
			"api_balance": gin.H{
				"total_quota": bal,
				"promo_quota": promo,
				"cash_quota":  bal - promo,
				"label":       "API Credits",
				"label_zh":    "平台调用额度",
			},
			"payment_methods_note": "可用支付方式将在安全结账页面中根据您的地区和设备显示。",
			"sandbox":              setting.StripeRequireTestKeys || strings.HasPrefix(setting.StripeApiSecret, "sk_test"),
		},
	})
}

// PostBillingTopupQuote recalculates conversion + promo server-side.
func PostBillingTopupQuote(c *gin.Context) {
	userId := c.GetInt("id")
	var req stripetopup.QuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}
	// Ignore client bonus fields if any — only amount_major + currency
	q, err := stripetopup.BuildQuote(userId, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": q})
}

// PostBillingTopupCheckout creates pending order + Stripe Checkout Session.
func PostBillingTopupCheckout(c *gin.Context) {
	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
		return
	}
	var body struct {
		Currency    string `json:"currency"`
		AmountMajor int    `json:"amount_major"`
		SuccessURL  string `json:"success_url"`
		CancelURL   string `json:"cancel_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}
	if body.SuccessURL != "" && common.ValidateRedirectURL(body.SuccessURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "success_url not trusted"})
		return
	}
	if body.CancelURL != "" && common.ValidateRedirectURL(body.CancelURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "cancel_url not trusted"})
		return
	}

	res, err := stripetopup.CreateCheckout(user, stripetopup.QuoteRequest{
		Currency:    body.Currency,
		AmountMajor: body.AmountMajor,
	}, body.SuccessURL, body.CancelURL)
	if err != nil {
		logger.LogError(c.Request.Context(), "stripe topup checkout: "+err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}

// GetBillingTopupOrder returns order status for success-page polling (owner only).
func GetBillingTopupOrder(c *gin.Context) {
	userId := c.GetInt("id")
	orderID := c.Param("order_id")
	if orderID == "" {
		orderID = c.Query("order_id")
	}
	o, err := model.GetStripeTopupOrderByOrderID(orderID)
	if err != nil || o == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "order not found"})
		return
	}
	if o.UserId != userId {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"order_id":                 o.OrderID,
			"status":                   o.Status,
			"presentment_currency":     o.PresentmentCurrency,
			"presentment_amount_minor": o.PresentmentAmountMinor,
			"paid_quota":               o.PaidQuota,
			"promo_quota":              o.PromoQuota,
			"total_quota":              o.PaidQuota + o.PromoQuota,
			"paid_credit_micro_usd":    o.PaidCreditMicroUSD,
			"promo_credit_micro_usd":   o.PromoCreditMicroUSD,
			"credited_at":              o.CreditedAt,
			"failure_reason":           o.FailureReason,
		},
	})
}

// AdminBillingTopupOrders lists recent stripe top-up orders.
func AdminBillingTopupOrders(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 0 {
		page = 0
	}
	list, total, err := model.AdminListStripeTopupOrders(status, size, page*size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": list, "total": total}})
}

// AdminBillingPromoTiers lists or upserts promo tiers.
func AdminBillingPromoTiers(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		list, err := model.ListAllPromoTiers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
		return
	}
	var t model.TopupPromoTier
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid body"})
		return
	}
	if err := model.SavePromoTier(&t); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": t})
}

// AdminBillingTopupConfig returns sandbox indicators for admin UI.
func AdminBillingTopupConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"config":     setting.ExportTopupConfigJSON(),
			"account_id": setting.StripeAccountID,
			"environment": func() string {
				if strings.HasPrefix(setting.StripeApiSecret, "sk_live") {
					return "live"
				}
				return "sandbox"
			}(),
		},
	})
}

// AdminRetryTopupCredit re-runs credit for paid orders stuck without credit (safe/idempotent).
func AdminRetryTopupCredit(c *gin.Context) {
	orderID := c.Param("order_id")
	o, err := model.GetStripeTopupOrderByOrderID(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "not found"})
		return
	}
	already, err := model.CreditStripeTopupOrder(orderID, o.StripeCustomerID, o.StripePaymentIntentID, o.StripeCheckoutSessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"already": already}})
}

// StripeWebhookV2 processes webhooks with idempotent event table + new orders.
// Also falls back to legacy fulfillOrder for old TopUp trade_no.
func StripeWebhookV2(c *gin.Context) {
	ctx := c.Request.Context()
	if !isStripeWebhookEnabled() {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	signature := c.GetHeader("Stripe-Signature")
	event, err := webhook.ConstructEventWithOptions(payload, signature, setting.StripeWebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		logger.LogWarn(ctx, "stripe webhook v2 sig fail: "+err.Error())
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := stripetopup.ProcessVerifiedEvent(ctx, event); err != nil {
		// If new-order path failed because order missing, try legacy handlers for same event types.
		logger.LogWarn(ctx, "stripe webhook v2 process: "+err.Error())
		if errors.Is(err, stripetopup.ErrWebhookPaymentMismatch) {
			c.Status(http.StatusOK)
			return
		}
		if strings.Contains(err.Error(), "order not found") && !strings.HasPrefix(event.GetObjectValue("client_reference_id"), "np_") {
			// legacy path uses same event construction
			callerIp := c.ClientIP()
			switch event.Type {
			case "checkout.session.completed":
				sessionCompleted(ctx, event, callerIp)
			case "checkout.session.expired":
				sessionExpired(ctx, event)
			case "checkout.session.async_payment_succeeded":
				sessionAsyncPaymentSucceeded(ctx, event, callerIp)
			case "checkout.session.async_payment_failed":
				sessionAsyncPaymentFailed(ctx, event, callerIp)
			}
			c.Status(http.StatusOK)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

func pickFirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
