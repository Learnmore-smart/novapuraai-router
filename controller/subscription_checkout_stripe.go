package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/checkout/session"
	"github.com/thanhpk/randstr"
	"gorm.io/gorm"
)

// SubscriptionStripeCheckoutRequest is the DTO for the NovaPura v2 Stripe
// Checkout endpoint. It supports both auto-renew (Stripe subscription mode)
// and prepaid (Stripe payment mode with a one-time charge for N months).
type SubscriptionStripeCheckoutRequest struct {
	PlanId        int    `json:"plan_id" binding:"required"`
	Mode          string `json:"mode" binding:"required"`     // "auto_renew" or "prepaid"
	Currency      string `json:"currency" binding:"required"` // "CNY" or "USD"
	PrepaidMonths int    `json:"prepaid_months"`              // required if mode=="prepaid"; must be in plan.PrepaidMonths
	CouponCode    string `json:"coupon_code"`                 // optional
	SuccessURL    string `json:"success_url"`                 // optional, defaults to /checkout/success
	CancelURL     string `json:"cancel_url"`                  // optional, defaults to /checkout/cancel
}

const (
	subscriptionCheckoutModeAutoRenew = "auto_renew"
	subscriptionCheckoutModePrepaid   = "prepaid"
)

// validateCheckoutRequest validates the request against the plan and returns
// the resolved currency + the per-currency price amount. Extracted as a
// standalone function so it can be unit-tested without invoking the Stripe
// API or DB writes.
func validateCheckoutRequest(req *SubscriptionStripeCheckoutRequest, plan *model.SubscriptionPlan) error {
	if plan == nil {
		return errors.New("plan is nil")
	}
	if !plan.Enabled {
		return errors.New("plan is not enabled")
	}
	if req.Mode != subscriptionCheckoutModeAutoRenew && req.Mode != subscriptionCheckoutModePrepaid {
		return errors.New("invalid mode; must be auto_renew or prepaid")
	}
	curr := strings.ToUpper(strings.TrimSpace(req.Currency))
	if curr != "CNY" && curr != "USD" {
		return errors.New("invalid currency; must be CNY or USD")
	}
	if req.Mode == subscriptionCheckoutModeAutoRenew {
		if curr == "CNY" && strings.TrimSpace(plan.StripePriceIdCNY) == "" {
			return errors.New("plan does not have a Stripe CNY price configured for auto-renew")
		}
		if curr == "USD" && strings.TrimSpace(plan.StripePriceIdUSD) == "" {
			return errors.New("plan does not have a Stripe USD price configured for auto-renew")
		}
	}
	if req.Mode == subscriptionCheckoutModePrepaid {
		if strings.TrimSpace(plan.StripeProductId) == "" {
			return errors.New("plan does not have a Stripe product id configured for prepaid mode")
		}
		if req.PrepaidMonths <= 0 {
			return errors.New("prepaid_months must be > 0 for prepaid mode")
		}
		if !isPrepaidMonthsAllowed(plan.PrepaidMonths, req.PrepaidMonths) {
			return fmt.Errorf("prepaid_months=%d is not in plan's allowed list %q", req.PrepaidMonths, plan.PrepaidMonths)
		}
	}
	return nil
}

// isPrepaidMonthsAllowed parses the plan's PrepaidMonths CSV (e.g. "1,3,6,12")
// and reports whether the requested months is in the list.
func isPrepaidMonthsAllowed(csv string, months int) bool {
	for _, raw := range strings.Split(csv, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(raw, "%d", &n); err != nil {
			continue
		}
		if n == months {
			return true
		}
	}
	return false
}

// resolvePriceAmount returns the plan's display price for the given currency.
func resolvePriceAmount(plan *model.SubscriptionPlan, currency string) float64 {
	if currency == "CNY" {
		return plan.PriceAmountCNY
	}
	return plan.PriceAmountUSD
}

// resolveStripePriceId returns the plan's Stripe Price ID for the given currency.
func resolveStripePriceId(plan *model.SubscriptionPlan, currency string) string {
	if currency == "CNY" {
		return strings.TrimSpace(plan.StripePriceIdCNY)
	}
	return strings.TrimSpace(plan.StripePriceIdUSD)
}

// computePrepaidAmountMinor computes the original, discount and final amounts
// (in the selected currency's minor units) for a prepaid checkout. Uses
// common.QuotaFromDecimal for the safe float -> int conversion (saturating,
// never overflows). Returns 0 discount when no coupon is applied.
func computePrepaidAmountMinor(plan *model.SubscriptionPlan, currency string, prepaidMonths int, percentOff int) (original, discount, final int64) {
	priceAmount := resolvePriceAmount(plan, currency)
	// original = price * 100 * months (minor units)
	originalInt := common.QuotaFromDecimal(
		decimal.NewFromFloat(priceAmount).
			Mul(decimal.NewFromInt(100)).
			Mul(decimal.NewFromInt(int64(prepaidMonths))),
	)
	original = int64(originalInt)
	if percentOff <= 0 || percentOff >= 100 {
		return original, 0, original
	}
	discountInt := common.QuotaFromDecimal(
		decimal.NewFromInt(int64(originalInt)).
			Mul(decimal.NewFromInt(int64(percentOff))).
			Div(decimal.NewFromInt(100)),
	)
	discount = int64(discountInt)
	final = original - discount
	if final < 0 {
		final = 0
	}
	return original, discount, final
}

// SubscriptionRequestStripeCheckoutV2 is the NovaPura v2 Stripe Checkout
// endpoint. It handles both auto-renew and prepaid modes and supersedes the
// legacy SubscriptionRequestStripePay handler (which is kept for back-compat).
//
// Wired to POST /api/subscription/stripe/checkout under middleware.UserAuth().
func SubscriptionRequestStripeCheckoutV2(c *gin.Context) {
	var req SubscriptionStripeCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.Mode = strings.TrimSpace(req.Mode)
	req.CouponCode = strings.TrimSpace(req.CouponCode)

	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil || plan == nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	if err := validateCheckoutRequest(&req, plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Stripe credential guards (mirror the legacy handler's pattern).
	if !strings.HasPrefix(setting.StripeApiSecret, "sk_") && !strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		common.ApiErrorMsg(c, "Stripe 未配置或密钥无效")
		return
	}
	if setting.StripeWebhookSecret == "" {
		common.ApiErrorMsg(c, "Stripe Webhook 未配置")
		return
	}

	user, err := model.GetUserById(userId, false)
	if err != nil || user == nil {
		common.ApiErrorMsg(c, "用户不存在")
		return
	}

	// Optional redirect URL validation. Empty means "use default".
	if req.SuccessURL != "" && common.ValidateRedirectURL(req.SuccessURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "success_url not trusted"})
		return
	}
	if req.CancelURL != "" && common.ValidateRedirectURL(req.CancelURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "cancel_url not trusted"})
		return
	}

	// Duplicate-prevention (Phase 10 hardening). For auto-renew, reject if the
	// user has ANY active auto-renew subscription (active / canceling /
	// past_due) for ANY plan — a user should only have ONE auto-renew
	// subscription at a time. Even a `canceling` subscription blocks a new
	// auto-renew checkout (the user must let it expire or contact support to
	// switch plans early); see model.HasActiveAutoRenewSubscription for the
	// policy rationale.
	if req.Mode == subscriptionCheckoutModeAutoRenew {
		exists, err := model.HasActiveAutoRenewSubscription(userId)
		if err != nil {
			logger.LogError(c.Request.Context(), "subscription checkout duplicate check failed: "+err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"message": "duplicate check failed"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{
				"message":    "active subscription exists",
				"manage_url": "/account/subscription",
			})
			return
		}
	}

	// Recent-pending-order dedup (Phase 10): if the user already has a pending
	// order for the same plan+mode created within the last 60 seconds, return
	// its existing Stripe Checkout URL (when available) or a 409 "checkout in
	// progress". This catches rapid double-clicks that would otherwise create
	// two Stripe Checkout Sessions (and two order rows) before the first
	// response arrives.
	recentOrder, err := model.FindRecentPendingSubscriptionOrder(userId, plan.Id, req.Mode, 60)
	if err != nil {
		logger.LogError(c.Request.Context(), "subscription checkout recent-order dedup failed: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"message": "recent-order dedup failed"})
		return
	}
	if recentOrder != nil {
		if recentOrder.CheckoutUrl != "" {
			c.JSON(http.StatusOK, gin.H{
				"message": "success",
				"data": gin.H{
					"checkout_url": recentOrder.CheckoutUrl,
					"order_id":     recentOrder.TradeNo,
				},
			})
			return
		}
		// Pending order exists but the Stripe session URL has not been persisted
		// yet (the first request is still in flight). Ask the user to wait so
		// they retry once the first request completes.
		c.JSON(http.StatusConflict, gin.H{
			"message":    "checkout in progress, please wait",
			"manage_url": "/account/subscription",
		})
		return
	}

	// Validate the coupon (read-only). Reservation happens at order creation.
	var coupon *model.SubscriptionCoupon
	if req.CouponCode != "" {
		coupon, err = model.ValidateSubscriptionCoupon(req.CouponCode, userId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
	}

	// Compute amounts (prepaid only; auto-renew relies on Stripe Price).
	var originalAmount, discountAmount, finalAmount int64
	if req.Mode == subscriptionCheckoutModePrepaid {
		percentOff := 0
		if coupon != nil {
			percentOff = coupon.PercentOff
		}
		originalAmount, discountAmount, finalAmount = computePrepaidAmountMinor(plan, req.Currency, req.PrepaidMonths, percentOff)
	}

	// Order ID + idempotency key (mirror existing subscription_payment_stripe.go
	// and stripetopup/checkout.go patterns).
	reference := fmt.Sprintf("sub-stripe-v2-%d-%d-%s-%s-%d", user.Id, plan.Id, req.Mode, req.Currency, time.Now().UnixMilli())
	orderID := "sub_ref_" + common.Sha1([]byte(reference+randstr.String(4)))
	idem := fmt.Sprintf("sub_checkout_%d_%d_%s_%s_%d", user.Id, plan.Id, req.Mode, req.Currency, time.Now().UnixNano())

	// Create the order + (optional) coupon reservation in a single tx.
	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           resolvePriceAmount(plan, req.Currency),
		TradeNo:         orderID,
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
		Currency:        req.Currency,
		PrepaidMonths:   req.PrepaidMonths,
		OriginalAmount:  originalAmount,
		DiscountAmount:  discountAmount,
		FinalAmount:     finalAmount,
		Mode:            req.Mode,
	}
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if coupon == nil {
			return nil
		}
		redemption, err := model.ReserveSubscriptionCouponWithTx(tx, coupon.Id, userId, plan.Id, orderID, coupon.PercentOff, originalAmount, discountAmount, finalAmount, req.Currency)
		if err != nil {
			return err
		}
		redemptionId := int(redemption.Id)
		order.CouponRedemptionId = &redemptionId
		return tx.Model(order).Update("coupon_redemption_id", redemptionId).Error
	}); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("subscription checkout order create failed user=%d plan=%d err=%q", userId, plan.Id, err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "创建订单失败"})
		return
	}

	// Build the Stripe Checkout Session.
	stripe.Key = setting.StripeApiSecret

	successURL := req.SuccessURL
	if successURL == "" {
		successURL = paymentReturnPath("/checkout/success")
	}
	cancelURL := req.CancelURL
	if cancelURL == "" {
		cancelURL = paymentReturnPath("/checkout/cancel")
	}

	environment := "sandbox"
	if strings.HasPrefix(setting.StripeApiSecret, "sk_live") || strings.HasPrefix(setting.StripeApiSecret, "rk_live") {
		environment = "live"
	}

	metadata := map[string]string{
		"novapura_user_id":  fmt.Sprintf("%d", user.Id),
		"novapura_order_id": orderID,
		"novapura_plan_id":  fmt.Sprintf("%d", plan.Id),
		"novapura_mode":     req.Mode,
		"environment":       environment,
	}

	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(orderID),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		Locale:            stripe.String("auto"),
		Metadata:          metadata,
	}
	params.SetIdempotencyKey(idem)

	// Apply coupon via the top-level Discounts field (supported in both
	// subscription and payment modes in v85 SDK).
	if coupon != nil && coupon.StripeCouponId != "" {
		params.Discounts = []*stripe.CheckoutSessionDiscountParams{
			{Coupon: stripe.String(coupon.StripeCouponId)},
		}
	}

	if user.StripeCustomer != "" {
		params.Customer = stripe.String(user.StripeCustomer)
	} else {
		params.CustomerCreation = stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways))
		if user.Email != "" {
			params.CustomerEmail = stripe.String(user.Email)
		}
	}

	if req.Mode == subscriptionCheckoutModeAutoRenew {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModeSubscription))
		params.LineItems = []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(resolveStripePriceId(plan, req.Currency)),
				Quantity: stripe.Int64(1),
			},
		}
		// SubscriptionData metadata is propagated to the created Subscription
		// (the top-level Metadata above stays on the Session itself).
		params.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: metadata,
		}
	} else {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModePayment))
		params.LineItems = []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(strings.ToLower(req.Currency)),
					Product:    stripe.String(plan.StripeProductId),
					UnitAmount: stripe.Int64(finalAmount),
				},
			},
		}
		params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: metadata,
		}
	}

	sess, err := session.New(params)
	if err != nil {
		// Mark the order failed and release the coupon reservation so the
		// user can retry without burning their per-user coupon quota.
		_ = model.MarkSubscriptionOrderStatus(orderID, "*", common.TopUpStatusFailed, "stripe checkout: "+err.Error())
		if coupon != nil {
			_ = model.ReleaseSubscriptionCouponRedemption(orderID)
		}
		logger.LogError(c.Request.Context(), fmt.Sprintf("subscription stripe checkout failed order=%s err=%q", orderID, err.Error()))
		c.JSON(http.StatusBadGateway, gin.H{"message": "拉起支付失败", "data": err.Error()})
		return
	}

	// Persist the checkout session id and URL on the order for traceability
	// and so the recent-pending-order dedup path can return the URL on a rapid
	// retry instead of creating a second Stripe session.
	if err := model.DB.Model(order).Updates(map[string]interface{}{
		"stripe_checkout_session_id": sess.ID,
		"checkout_url":               sess.URL,
	}).Error; err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("subscription checkout session id persist failed order=%s err=%q", orderID, err.Error()))
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"checkout_url": sess.URL,
			"order_id":     orderID,
		},
	})
}
