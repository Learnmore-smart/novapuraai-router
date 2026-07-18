package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/emaildelivery"
)

var (
	getTransactionalEmailHealth           = emaildelivery.GetHealth
	retryTransactionalEmails              = emaildelivery.RetrySafeDeliveries
	updateTransactionalEmailProvider      = model.UpdateEmailProvider
	getTransactionalEmailCredentialStatus = emaildelivery.GetSESCredentialStatus
	saveTransactionalEmailCredentials     = emaildelivery.SaveSESCredentials
	deleteTransactionalEmailCredentials   = emaildelivery.DeleteSESCredentials
)

const maxTransactionalEmailCredentialBytes = 4096

func GetTransactionalEmailHealth(c *gin.Context) {
	report, err := getTransactionalEmailHealth(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    report,
	})
}

func SwitchTransactionalEmailProvider(c *gin.Context) {
	var request struct {
		Provider string `json:"provider"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid email provider request"})
		return
	}

	provider := emaildelivery.ProviderName(strings.ToLower(strings.TrimSpace(request.Provider)))
	if provider != emaildelivery.ProviderBrevo && provider != emaildelivery.ProviderSES {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "provider must be brevo or ses"})
		return
	}

	report, err := getTransactionalEmailHealth(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var target *emaildelivery.ProviderHealth
	for index := range report.Providers {
		if report.Providers[index].Provider == provider {
			target = &report.Providers[index]
			break
		}
	}
	if target == nil || !target.Configured {
		reason := emaildelivery.FailureConfiguration
		if target != nil && target.FailureReason != "" {
			reason = target.FailureReason
		}
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "email provider is not ready: " + reason,
		})
		return
	}

	if err := updateTransactionalEmailProvider(string(provider)); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "email.provider_switch", map[string]interface{}{"provider": provider})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"provider": provider},
	})
}

func RetryTransactionalEmailQueue(c *gin.Context) {
	result, err := retryTransactionalEmails(c.Request.Context(), 50)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "email.safe_retry", map[string]interface{}{
		"processed": result.Processed,
		"sent":      result.Sent,
		"queued":    result.Queued,
		"failed":    result.Failed,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

func GetTransactionalEmailSESCredentials(c *gin.Context) {
	status, err := getTransactionalEmailCredentialStatus(c.Request.Context())
	if err != nil {
		common.SysError("failed to read SES credential status")
		common.ApiErrorMsg(c, "failed to read SES credential status")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    status,
	})
}

func UpdateTransactionalEmailSESCredentials(c *gin.Context) {
	var request emaildelivery.SESCredentialUpdate
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid SES credential request"})
		return
	}
	if request.AccessKeyID == "" && request.SecretAccessKey == "" && request.SessionToken == "" && !request.ClearSessionToken {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "at least one credential change is required"})
		return
	}
	if request.SessionToken != "" && request.ClearSessionToken {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "session token cannot be replaced and cleared together"})
		return
	}
	if len(request.AccessKeyID) > maxTransactionalEmailCredentialBytes ||
		len(request.SecretAccessKey) > maxTransactionalEmailCredentialBytes ||
		len(request.SessionToken) > maxTransactionalEmailCredentialBytes {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "credential field is too long"})
		return
	}

	status, err := saveTransactionalEmailCredentials(c.Request.Context(), request)
	if err != nil {
		if errors.Is(err, emaildelivery.ErrSESCredentialsEnvironmentManaged) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.SysError("failed to save SES credentials")
		common.ApiErrorMsg(c, "failed to save SES credentials")
		return
	}
	recordManageAudit(c, "email.ses_credentials_update", map[string]interface{}{
		"access_key_replaced":   request.AccessKeyID != "",
		"secret_key_replaced":   request.SecretAccessKey != "",
		"session_token_changed": request.SessionToken != "" || request.ClearSessionToken,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    status,
	})
}

func DeleteTransactionalEmailSESCredentials(c *gin.Context) {
	status, err := deleteTransactionalEmailCredentials(c.Request.Context())
	if err != nil {
		common.SysError("failed to delete SES credentials")
		common.ApiErrorMsg(c, "failed to delete SES credentials")
		return
	}
	recordManageAudit(c, "email.ses_credentials_delete", nil)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    status,
	})
}
