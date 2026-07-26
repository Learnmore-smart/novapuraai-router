package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

// GetCommissionSummary returns the current user's cash commission balances
// (pending/frozen, available/withdrawable, lifetime total, lifetime withdrawn).
func GetCommissionSummary(c *gin.Context) {
	userId := c.GetInt("id")
	summary, err := model.GetCommissionSummaryForUser(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

type withdrawRequest struct {
	AmountCents int64 `json:"amount_cents"`
}

// RequestWithdrawal creates a pending withdrawal request, debiting the user's
// withdrawable commission balance up-front. Admin reviews and marks paid or
// rejected (refund) later.
func RequestWithdrawal(c *gin.Context) {
	var req withdrawRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid request"})
		return
	}
	userId := c.GetInt("id")
	result, err := model.CreateWithdrawalRequest(userId, req.AmountCents)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, result)
}

// ListMyWithdrawals returns the current user's withdrawal request history.
func ListMyWithdrawals(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	reqs, total, err := model.ListUserWithdrawalRequests(userId, pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(reqs)
	common.ApiSuccess(c, pageInfo)
}

// AdminListWithdrawals returns all withdrawal requests for the admin review queue.
func AdminListWithdrawals(c *gin.Context) {
	status := c.Query("status")
	pageInfo := common.GetPageQuery(c)
	reqs, total, err := model.AdminListWithdrawalRequests(status, pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(reqs)
	common.ApiSuccess(c, pageInfo)
}

type processWithdrawalRequest struct {
	Action        string `json:"action"`         // "paid" or "rejected"
	PayoutChannel string `json:"payout_channel"`  // e.g. "manual", "stripe_connect" (reserved)
	PayoutTxId    string `json:"payout_tx_id"`    // external transaction id (reserved)
	AdminRemark   string `json:"admin_remark"`
}

// AdminProcessWithdrawal marks a pending withdrawal as paid (admin confirmed
// manual payout) or rejected (funds refunded to user's withdrawable balance).
//
// When action=paid and payout_channel=stripe_connect, it delegates to the
// synchronous Stripe Connect Transfer flow (service.ApproveStripeConnectWithdrawal)
// instead of the manual paid path.
func AdminProcessWithdrawal(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req processWithdrawalRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid request"})
		return
	}
	adminId := c.GetInt("id")

	// Stripe Connect payout flow: admin approves with stripe_connect channel.
	if req.Action == "paid" && req.PayoutChannel == "stripe_connect" {
		if !setting.StripeConnectEnabled {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "stripe connect is not enabled"})
			return
		}
		client := service.NewStripeConnectClient()
		if client == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "stripe connect is not configured"})
			return
		}
		result, err := service.ApproveStripeConnectWithdrawal(c.Request.Context(), client, id, adminId)
		if err != nil {
			if errors.Is(err, service.ErrWithdrawalAlreadyProcessing) {
				// Idempotent: admin may have double-clicked. Return current state.
				if result != nil {
					recordManageAuditFor(c, result.UserId, "commission.withdrawal_process", map[string]any{
						"request_id":     result.ID,
						"action":         result.Status,
						"amount_cents":   result.AmountCents,
						"payout_channel": result.PayoutChannel,
					})
				}
				common.ApiSuccess(c, result)
				return
			}
			// Transfer creation failed — withdrawal is now in `failed` state with
			// balance refunded; surface the error to the admin.
			if result != nil {
				recordManageAuditFor(c, result.UserId, "commission.withdrawal_process", map[string]any{
					"request_id":     result.ID,
					"action":         result.Status,
					"amount_cents":   result.AmountCents,
					"payout_channel": result.PayoutChannel,
					"error":          err.Error(),
				})
			}
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		recordManageAuditFor(c, result.UserId, "commission.withdrawal_process", map[string]any{
			"request_id":         result.ID,
			"action":             result.Status,
			"amount_cents":       result.AmountCents,
			"payout_channel":     result.PayoutChannel,
			"stripe_transfer_id": result.StripeTransferId,
		})
		common.ApiSuccess(c, result)
		return
	}

	result, err := model.AdminProcessWithdrawal(id, req.Action, adminId, req.PayoutChannel, req.PayoutTxId, req.AdminRemark)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	recordManageAuditFor(c, result.UserId, "commission.withdrawal_process", map[string]any{
		"request_id":     result.ID,
		"action":         result.Status,
		"amount_cents":   result.AmountCents,
		"payout_channel": result.PayoutChannel,
		"payout_tx_id":   result.PayoutTxId,
	})
	common.ApiSuccess(c, result)
}
