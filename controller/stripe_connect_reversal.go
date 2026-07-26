package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

// AdminReverseStripeConnectTransfer — POST /api/user/withdrawal/:id/reverse
// Admin-only. Manually reverses the Stripe Transfer for a withdrawal that has
// a stripe_transfer_id. Body: {"reason": "..."}.
// Use case: admin decides to give up on a stuck withdrawal and refund the user.
func AdminReverseStripeConnectTransfer(c *gin.Context) {
	if !setting.StripeConnectEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "stripe connect is not enabled"})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid withdrawal id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body) // optional body
	client := service.NewStripeConnectClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "stripe connect is not configured"})
		return
	}
	result, err := service.ReverseStripeConnectTransfer(c.Request.Context(), client, id, body.Reason)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error(), "data": result})
		return
	}
	common.ApiSuccess(c, result)
}
