package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetUserSuccessfulRequestStatus exposes only the authenticated user's strict
// consume-log signal used by the adaptive dashboard.
func GetUserSuccessfulRequestStatus(c *gin.Context) {
	hasSuccessfulRequest, err := model.HasSuccessfulRequestForUser(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"has_successful_request": hasSuccessfulRequest,
		},
	})
}
