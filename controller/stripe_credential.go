package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

const maxStripeCredentialBytes = 4096

type stripeCredentialUpdateRequest struct {
	SecretKey      string `json:"secret_key"`
	PublishableKey string `json:"publishable_key"`
	WebhookSecret  string `json:"webhook_secret"`
}

func stripeEnvironmentFromRequest(c *gin.Context) (string, bool) {
	environment := strings.ToLower(strings.TrimSpace(c.Param("environment")))
	if environment != model.StripeEnvironmentTest && environment != model.StripeEnvironmentProduction {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid Stripe environment"})
		return "", false
	}
	return environment, true
}

func GetStripeCredentials(c *gin.Context) {
	environment, ok := stripeEnvironmentFromRequest(c)
	if !ok {
		return
	}
	status, err := model.GetStripeCredentialStatus(environment)
	if err != nil {
		common.SysError("failed to read Stripe credential status")
		common.ApiErrorMsg(c, "failed to read Stripe credential status")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": status})
}

func UpdateStripeCredentials(c *gin.Context) {
	environment, ok := stripeEnvironmentFromRequest(c)
	if !ok {
		return
	}
	var request stripeCredentialUpdateRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid Stripe credential request"})
		return
	}
	request.SecretKey = strings.TrimSpace(request.SecretKey)
	request.PublishableKey = strings.TrimSpace(request.PublishableKey)
	request.WebhookSecret = strings.TrimSpace(request.WebhookSecret)
	if request.SecretKey == "" && request.PublishableKey == "" && request.WebhookSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "at least one credential change is required"})
		return
	}
	if len(request.SecretKey) > maxStripeCredentialBytes || len(request.PublishableKey) > maxStripeCredentialBytes || len(request.WebhookSecret) > maxStripeCredentialBytes {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "credential field is too long"})
		return
	}
	secretPrefix := "sk_test_"
	restrictedPrefix := "rk_test_"
	publishablePrefix := "pk_test_"
	if environment == model.StripeEnvironmentProduction {
		secretPrefix = "sk_live_"
		restrictedPrefix = "rk_live_"
		publishablePrefix = "pk_live_"
	}
	if request.SecretKey != "" && !strings.HasPrefix(request.SecretKey, secretPrefix) && !strings.HasPrefix(request.SecretKey, restrictedPrefix) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Stripe secret key does not match the selected environment"})
		return
	}
	if request.PublishableKey != "" && !strings.HasPrefix(request.PublishableKey, publishablePrefix) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Stripe publishable key does not match the selected environment"})
		return
	}
	if request.WebhookSecret != "" && !strings.HasPrefix(request.WebhookSecret, "whsec_") {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid Stripe webhook signing secret"})
		return
	}

	status, err := model.SaveStripeCredentials(environment, model.StripeCredentialUpdate{
		SecretKey:      request.SecretKey,
		PublishableKey: request.PublishableKey,
		WebhookSecret:  request.WebhookSecret,
	})
	if err != nil {
		common.SysError("failed to save Stripe credentials")
		common.ApiErrorMsg(c, "failed to save Stripe credentials")
		return
	}
	credentials, found, err := model.LoadStripeCredentials(environment)
	if err != nil || !found {
		common.SysError("failed to reload Stripe credentials")
		common.ApiErrorMsg(c, "failed to reload Stripe credentials")
		return
	}
	setting.SetStripeCredentialProfile(environment, credentials.SecretKey, credentials.PublishableKey, credentials.WebhookSecret)
	recordManageAudit(c, "stripe.credentials_update", map[string]interface{}{
		"environment":              environment,
		"secret_key_replaced":      request.SecretKey != "",
		"publishable_key_replaced": request.PublishableKey != "",
		"webhook_secret_replaced":  request.WebhookSecret != "",
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": status})
}

func DeleteStripeCredentials(c *gin.Context) {
	environment, ok := stripeEnvironmentFromRequest(c)
	if !ok {
		return
	}
	if err := model.DeleteStripeCredentials(environment); err != nil {
		common.SysError("failed to delete Stripe credentials")
		common.ApiErrorMsg(c, "failed to delete Stripe credentials")
		return
	}
	setting.ClearStripeCredentialProfile(environment)
	recordManageAudit(c, "stripe.credentials_delete", map[string]interface{}{"environment": environment})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": model.StripeCredentialStatus{}})
}
