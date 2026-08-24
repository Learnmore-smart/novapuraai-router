package controller

import (
	"net/http"
	"net/mail"
	"strings"
	"time"

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
	sendTransactionalEmailTest            = emaildelivery.SendTransactionalEmail
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

func SendTransactionalEmailTest(c *gin.Context) {
	var request struct {
		Recipient string `json:"recipient"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid test email request"})
		return
	}
	recipient, err := mail.ParseAddress(strings.TrimSpace(request.Recipient))
	if err != nil || recipient.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "a valid test recipient email is required"})
		return
	}

	result, err := sendTransactionalEmailTest(c.Request.Context(), emaildelivery.SendRequest{
		BusinessKey: "dashboard-test:" + time.Now().UTC().Format(time.RFC3339Nano),
		Message: emaildelivery.Message{
			Type:     emaildelivery.MessageTypeNotification,
			To:       recipient.Address,
			Subject:  "Transactional email test",
			HTMLBody: "<p>This is a transactional email delivery test.</p>",
			TextBody: "This is a transactional email delivery test.",
		},
	})
	if err != nil {
		recordManageAudit(c, "email.test_send", map[string]interface{}{
			"provider": result.Provider,
			"status":   result.Status,
			"failure":  result.FailureReason,
		})
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "test email delivery failed: " + result.FailureReason})
		return
	}
	recordManageAudit(c, "email.test_send", map[string]interface{}{
		"provider": result.Provider,
		"status":   result.Status,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"provider": result.Provider,
			"status":   result.Status,
		},
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
	if provider != emaildelivery.ProviderSES {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "provider must be ses"})
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
	hasCredentialChange := request.AccessKeyID != "" ||
		request.SecretAccessKey != "" ||
		request.SessionToken != "" ||
		request.ClearSessionToken
	hasSettingsChange := request.Region != nil || request.FromAddress != nil
	if !hasCredentialChange && !hasSettingsChange {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "at least one SES setting change is required"})
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
		if strings.Contains(err.Error(), "invalid SES from address") {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid verified sender address"})
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
		"region_updated":        request.Region != nil,
		"from_address_updated":  request.FromAddress != nil,
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
