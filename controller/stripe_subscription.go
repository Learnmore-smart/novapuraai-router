package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/stripesubscription"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetStripeSubscriptionCancel is a public browser return from Stripe. The
// random reservation reference is the capability; no authenticated dashboard
// session is assumed to survive the hosted Checkout redirect.
func GetStripeSubscriptionCancel(c *gin.Context) {
	if err := stripesubscription.CancelCheckout(c.Request.Context(), c.Query("reference_id")); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("failed to cancel recurring Stripe Checkout: %v", err))
	}
	c.Redirect(http.StatusSeeOther, paymentReturnPath("/"))
}

// GetStripeSubscriptionOffer exposes only the exact configured recurring
// sandbox offer. It returns enabled:false while the feature is disabled so
// the public UI can hide the card without treating that as an outage.
func GetStripeSubscriptionOffer(c *gin.Context) {
	planID := 0
	if raw := strings.TrimSpace(c.Query("plan_id")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			stripeSubscriptionHTTPError(c, errors.New("invalid plan_id"))
			return
		}
		planID = parsed
	}
	// The endpoint remains public, but an authenticated request may carry the
	// local user id. The service only uses it for server-derived duplicate
	// checkout state; capacity and checkout authorization are still enforced by
	// the transactional reservation path.
	offer, err := stripesubscription.GetStripeSubscriptionOffer(planID, c.GetInt("id"))
	if err != nil {
		stripeSubscriptionHTTPError(c, err)
		return
	}
	common.ApiSuccess(c, offer)
}

func GetStripeSubscriptionSummary(c *gin.Context) {
	summary, err := stripesubscription.GetStripeSubscriptionSummary(c.GetInt("id"))
	if err != nil {
		stripeSubscriptionHTTPError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

// PostStripeSubscriptionPortal deliberately ignores the request body. The
// customer and return URL are derived from the authenticated local record,
// so clients cannot open a portal for another account or redirect to an
// attacker-controlled URL.
func PostStripeSubscriptionPortal(c *gin.Context) {
	portal, err := stripesubscription.CreatePortalSession(c.Request.Context(), stripesubscription.PortalInput{
		UserID:    c.GetInt("id"),
		ReturnURL: paymentReturnPath("/console/subscription"),
	})
	if err != nil {
		stripeSubscriptionHTTPError(c, err)
		return
	}
	common.ApiSuccess(c, portal)
}

func stripeSubscriptionHTTPError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := "subscription_error"
	switch {
	case errors.Is(err, model.ErrSubscriptionCapacityFull), errors.Is(err, model.ErrStripeSubscriptionFounderSoldOut):
		status = http.StatusConflict
		code = "subscription_capacity_full"
	case errors.Is(err, model.ErrSubscriptionAlreadyActive):
		status = http.StatusConflict
		code = "subscription_already_active"
	case errors.Is(err, model.ErrSubscriptionAlreadyPending):
		status = http.StatusConflict
		code = "subscription_already_pending"
	case errors.Is(err, model.ErrStripeSubscriptionDisabled):
		status = http.StatusNotFound
		code = "subscription_disabled"
	case errors.Is(err, stripesubscription.ErrRecurringPaymentMismatch), errors.Is(err, model.ErrStripeSubscriptionPlanInvalid):
		status = http.StatusBadRequest
		code = "subscription_configuration_invalid"
	case errors.Is(err, gorm.ErrRecordNotFound):
		status = http.StatusNotFound
		code = "subscription_not_found"
	}
	c.JSON(status, gin.H{"success": false, "code": code, "message": err.Error()})
}
