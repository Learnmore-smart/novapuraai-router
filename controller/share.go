package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type shareSubmitRequest struct {
	URL      string `json:"url"`
	Platform string `json:"platform"`
	Note     string `json:"note"`
}

// SubmitShareReward user submits a public share link for admin review (MVP §9.5).
func SubmitShareReward(c *gin.Context) {
	var req shareSubmitRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid request"})
		return
	}
	userId := c.GetInt("id")
	s, err := model.CreateShareSubmission(userId, req.URL, req.Platform, req.Note)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, s)
}

// ListMyShareSubmissions lists current user's share submissions.
func ListMyShareSubmissions(c *gin.Context) {
	userId := c.GetInt("id")
	rows, err := model.ListUserShareSubmissions(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rows)
}

// AdminListShareSubmissions lists share queue for admins.
func AdminListShareSubmissions(c *gin.Context) {
	status := c.Query("status")
	pageInfo := common.GetPageQuery(c)
	rows, total, err := model.ListShareSubmissions(status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(rows)
	common.ApiSuccess(c, pageInfo)
}

type shareReviewRequest struct {
	Approve bool   `json:"approve"`
	Reason  string `json:"reason"`
}

// AdminReviewShareSubmission approve/reject share reward.
func AdminReviewShareSubmission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req shareReviewRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid request"})
		return
	}
	reviewerId := c.GetInt("id")
	if err := model.ReviewShareSubmission(id, reviewerId, req.Approve, req.Reason); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	recordManageAudit(c, "share.review", map[string]any{
		"id":      id,
		"approve": req.Approve,
		"reason":  req.Reason,
	})
	common.ApiSuccess(c, nil)
}
