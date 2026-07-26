package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

type connectStartResponse struct {
	URL string `json:"url"`
}

type connectStatusResponse struct {
	Started bool                        `json:"started"`
	Account *model.StripeConnectAccount `json:"account,omitempty"`
}

// StartStripeConnect — POST /api/user/stripe_connect/start
// Returns a fresh Stripe-hosted onboarding URL for the authenticated user.
func StartStripeConnect(c *gin.Context) {
	userId := c.GetInt("id")
	if !setting.StripeConnectEnabled {
		common.ApiErrorMsg(c, "stripe connect is not enabled")
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !user.CommissionApproved {
		common.ApiErrorMsg(c, "未开通现金佣金，无法关联 Stripe")
		return
	}
	client := service.NewStripeConnectClient()
	if client == nil {
		common.ApiErrorMsg(c, "stripe connect is not configured")
		return
	}
	url, err := service.StartConnectOnboarding(c.Request.Context(), client, userId, user.Email)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, connectStartResponse{URL: url})
}

// GetStripeConnectStatus — GET /api/user/stripe_connect/status
// Returns the user's onboarding state. If onboarding has not been started,
// returns {started:false}; otherwise returns the local account record.
func GetStripeConnectStatus(c *gin.Context) {
	userId := c.GetInt("id")
	if !setting.StripeConnectEnabled {
		common.ApiSuccess(c, connectStatusResponse{Started: false})
		return
	}
	client := service.NewStripeConnectClient()
	acc, err := service.GetConnectOnboardingStatus(c.Request.Context(), client, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if acc == nil {
		common.ApiSuccess(c, connectStatusResponse{Started: false})
		return
	}
	common.ApiSuccess(c, connectStatusResponse{Started: true, Account: acc})
}
