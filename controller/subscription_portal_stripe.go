package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/billingportal/session"
)

// SubscriptionStripePortalRequest is the DTO for creating a Stripe Customer
// Portal session. The return_url is optional and defaults to the server's
// account subscription page.
type SubscriptionStripePortalRequest struct {
	ReturnURL string `json:"return_url"`
}

// SubscriptionRequestStripePortal creates a Stripe Billing Portal session for
// the authenticated user, allowing them to manage their subscription (update
// payment method, cancel, view invoices) directly in Stripe's hosted UI.
//
// Wired to POST /api/subscription/stripe/portal under middleware.UserAuth().
func SubscriptionRequestStripePortal(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
		return
	}

	if !strings.HasPrefix(setting.StripeApiSecret, "sk_") && !strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		common.ApiErrorMsg(c, "Stripe 未配置或密钥无效")
		return
	}

	user, err := model.GetUserById(userId, false)
	if err != nil || user == nil {
		common.ApiErrorMsg(c, "用户不存在")
		return
	}

	if strings.TrimSpace(user.StripeCustomer) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "no stripe customer associated with this account",
		})
		return
	}

	var req SubscriptionStripePortalRequest
	_ = c.ShouldBindJSON(&req)
	returnURL := strings.TrimSpace(req.ReturnURL)
	if returnURL == "" {
		returnURL = paymentReturnPath("/account/subscription")
	} else if common.ValidateRedirectURL(returnURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "return_url not trusted"})
		return
	}

	stripe.Key = setting.StripeApiSecret
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(user.StripeCustomer),
		ReturnURL: stripe.String(returnURL),
	}
	portalSession, err := session.New(params)
	if err != nil {
		logger.LogError(c.Request.Context(), "stripe billing portal session failed user="+strconv.Itoa(userId)+" err="+err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "创建管理面板失败", "data": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url": portalSession.URL,
		},
	})
}
