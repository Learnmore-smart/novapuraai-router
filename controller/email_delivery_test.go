package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/emaildelivery"
)

func setupEmailDeliveryControllerTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousHealth := getTransactionalEmailHealth
	previousRetry := retryTransactionalEmails
	previousUpdate := updateTransactionalEmailProvider
	previousCredentialStatus := getTransactionalEmailCredentialStatus
	previousCredentialSave := saveTransactionalEmailCredentials
	previousCredentialDelete := deleteTransactionalEmailCredentials
	previousTestSend := sendTransactionalEmailTest
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousOptions := common.OptionMap
	previousRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.User{}, &model.Log{}))
	common.OptionMap = map[string]string{"EmailProvider": "brevo"}
	t.Cleanup(func() {
		getTransactionalEmailHealth = previousHealth
		retryTransactionalEmails = previousRetry
		updateTransactionalEmailProvider = previousUpdate
		getTransactionalEmailCredentialStatus = previousCredentialStatus
		saveTransactionalEmailCredentials = previousCredentialSave
		deleteTransactionalEmailCredentials = previousCredentialDelete
		sendTransactionalEmailTest = previousTestSend
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.OptionMap = previousOptions
	})
}

func TestSendTransactionalEmailTestValidatesRecipientAndRedactsIt(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	var captured emaildelivery.SendRequest
	sendTransactionalEmailTest = func(_ context.Context, request emaildelivery.SendRequest) (emaildelivery.DeliveryResult, error) {
		captured = request
		return emaildelivery.DeliveryResult{Provider: emaildelivery.ProviderSES, Status: emaildelivery.DeliveryStatusSent}, nil
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/option/email-provider/test", strings.NewReader(`{"recipient":"noahzh52@gmail.com"}`))
	SendTransactionalEmailTest(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "noahzh52@gmail.com", captured.Message.To)
	assert.Equal(t, emaildelivery.MessageTypeNotification, captured.Message.Type)
	assert.Contains(t, recorder.Body.String(), `"provider":"ses"`)
	assert.NotContains(t, recorder.Body.String(), "noahzh52@gmail.com")

	invalidRecorder := httptest.NewRecorder()
	invalidCtx, _ := gin.CreateTestContext(invalidRecorder)
	invalidCtx.Request = httptest.NewRequest(http.MethodPost, "/api/option/email-provider/test", strings.NewReader(`{"recipient":"not-an-email"}`))
	SendTransactionalEmailTest(invalidCtx)
	assert.Equal(t, http.StatusBadRequest, invalidRecorder.Code)
}

func TestTransactionalEmailSESCredentialStatusIsWriteOnly(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	getTransactionalEmailCredentialStatus = func(context.Context) (emaildelivery.SESCredentialStatus, error) {
		return emaildelivery.SESCredentialStatus{
			Configured:      true,
			Source:          emaildelivery.SESCredentialSourceDatabase,
			HasSessionToken: true,
		}, nil
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/option/email-provider/ses/credentials", nil)
	GetTransactionalEmailSESCredentials(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"configured":true`)
	assert.Contains(t, recorder.Body.String(), `"source":"database"`)
	assert.NotContains(t, recorder.Body.String(), "access_key_id")
	assert.NotContains(t, recorder.Body.String(), "secret_access_key")
	assert.NotContains(t, recorder.Body.String(), `"session_token":`)
}

func TestUpdateTransactionalEmailSESCredentialsNeverReturnsOrAuditsValues(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	var captured emaildelivery.SESCredentialUpdate
	saveTransactionalEmailCredentials = func(_ context.Context, update emaildelivery.SESCredentialUpdate) (emaildelivery.SESCredentialStatus, error) {
		captured = update
		return emaildelivery.SESCredentialStatus{
			Configured:      true,
			Source:          emaildelivery.SESCredentialSourceDatabase,
			HasSessionToken: true,
		}, nil
	}

	const accessKey = "AKIA-DASHBOARD-SECRET"
	const secretKey = "dashboard-secret-access-key"
	const sessionToken = "dashboard-session-token"
	body := `{"access_key_id":"` + accessKey + `","secret_access_key":"` + secretKey + `","session_token":"` + sessionToken + `"}`
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/option/email-provider/ses/credentials", strings.NewReader(body))
	UpdateTransactionalEmailSESCredentials(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, accessKey, captured.AccessKeyID)
	assert.Equal(t, secretKey, captured.SecretAccessKey)
	assert.Equal(t, sessionToken, captured.SessionToken)
	assert.NotContains(t, recorder.Body.String(), accessKey)
	assert.NotContains(t, recorder.Body.String(), secretKey)
	assert.NotContains(t, recorder.Body.String(), sessionToken)

	var logs []model.Log
	require.NoError(t, model.LOG_DB.Find(&logs).Error)
	for _, log := range logs {
		assert.NotContains(t, log.Content, accessKey)
		assert.NotContains(t, log.Content, secretKey)
		assert.NotContains(t, log.Content, sessionToken)
		assert.NotContains(t, log.Other, accessKey)
		assert.NotContains(t, log.Other, secretKey)
		assert.NotContains(t, log.Other, sessionToken)
	}
}

func TestUpdateTransactionalEmailSESCredentialsRejectsNoOpAndConflictingTokenUpdate(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	saveCalls := 0
	saveTransactionalEmailCredentials = func(context.Context, emaildelivery.SESCredentialUpdate) (emaildelivery.SESCredentialStatus, error) {
		saveCalls++
		return emaildelivery.SESCredentialStatus{}, nil
	}

	for _, body := range []string{
		`{}`,
		`{"session_token":"replacement","clear_session_token":true}`,
		`{"secret_access_key":"` + strings.Repeat("x", maxTransactionalEmailCredentialBytes+1) + `"}`,
	} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/api/option/email-provider/ses/credentials", strings.NewReader(body))
		UpdateTransactionalEmailSESCredentials(ctx)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	}
	assert.Zero(t, saveCalls)
}

func TestDeleteTransactionalEmailSESCredentialsReturnsRedactedStatus(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	deleteCalls := 0
	deleteTransactionalEmailCredentials = func(context.Context) (emaildelivery.SESCredentialStatus, error) {
		deleteCalls++
		return emaildelivery.SESCredentialStatus{Source: emaildelivery.SESCredentialSourceNone}, nil
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/option/email-provider/ses/credentials", nil)
	DeleteTransactionalEmailSESCredentials(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, 1, deleteCalls)
	assert.Contains(t, recorder.Body.String(), `"configured":false`)
}

func TestUpdateTransactionalEmailSESCredentialsSanitizesInternalFailure(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	saveTransactionalEmailCredentials = func(context.Context, emaildelivery.SESCredentialUpdate) (emaildelivery.SESCredentialStatus, error) {
		return emaildelivery.SESCredentialStatus{}, errors.New("driver failure containing sensitive-ciphertext")
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/option/email-provider/ses/credentials",
		strings.NewReader(`{"secret_access_key":"replacement"}`),
	)
	UpdateTransactionalEmailSESCredentials(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "failed to save SES credentials")
	assert.NotContains(t, recorder.Body.String(), "sensitive-ciphertext")
}

func TestGetTransactionalEmailHealthReturnsProviderAndQueueState(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	getTransactionalEmailHealth = func(context.Context) (emaildelivery.HealthReport, error) {
		return emaildelivery.HealthReport{
			SelectedProvider: emaildelivery.ProviderBrevo,
			Providers: []emaildelivery.ProviderHealth{{
				Provider:   emaildelivery.ProviderBrevo,
				Configured: true,
				Reachable:  true,
				Ready:      true,
			}},
			SafeRetryCount:    2,
			ManualReviewCount: 1,
		}, nil
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/option/email-provider/health", nil)
	GetTransactionalEmailHealth(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool                       `json:"success"`
		Data    emaildelivery.HealthReport `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, emaildelivery.ProviderBrevo, response.Data.SelectedProvider)
	assert.EqualValues(t, 2, response.Data.SafeRetryCount)
	assert.EqualValues(t, 1, response.Data.ManualReviewCount)
}

func TestSwitchTransactionalEmailProviderPersistsConfiguredSESWithoutProductionAccess(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	getTransactionalEmailHealth = func(context.Context) (emaildelivery.HealthReport, error) {
		return emaildelivery.HealthReport{Providers: []emaildelivery.ProviderHealth{{
			Provider:         emaildelivery.ProviderSES,
			Configured:       true,
			Reachable:        true,
			SendingEnabled:   true,
			ProductionAccess: false,
			FailureReason:    emaildelivery.FailureProductionAccessRequired,
		}}}, nil
	}
	var updatedValue string
	updateTransactionalEmailProvider = func(value string) error {
		updatedValue = value
		return nil
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/option/email-provider", strings.NewReader(`{"provider":"ses"}`))
	SwitchTransactionalEmailProvider(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "ses", updatedValue)
}

func TestSwitchTransactionalEmailProviderRejectsUnconfiguredProvider(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	getTransactionalEmailHealth = func(context.Context) (emaildelivery.HealthReport, error) {
		return emaildelivery.HealthReport{Providers: []emaildelivery.ProviderHealth{{
			Provider:      emaildelivery.ProviderSES,
			Configured:    false,
			FailureReason: emaildelivery.FailureConfiguration,
		}}}, nil
	}
	updateCalls := 0
	updateTransactionalEmailProvider = func(string) error {
		updateCalls++
		return nil
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/option/email-provider", strings.NewReader(`{"provider":"ses"}`))
	SwitchTransactionalEmailProvider(ctx)

	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.Zero(t, updateCalls)
	assert.Contains(t, recorder.Body.String(), emaildelivery.FailureConfiguration)
}

func TestSwitchTransactionalEmailProviderPersistsReadyProvider(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	getTransactionalEmailHealth = func(context.Context) (emaildelivery.HealthReport, error) {
		return emaildelivery.HealthReport{Providers: []emaildelivery.ProviderHealth{{
			Provider:         emaildelivery.ProviderSES,
			Configured:       true,
			Reachable:        true,
			Ready:            true,
			SendingEnabled:   true,
			ProductionAccess: true,
		}}}, nil
	}
	var updatedValue string
	updateTransactionalEmailProvider = func(value string) error {
		updatedValue = value
		return nil
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/option/email-provider", strings.NewReader(`{"provider":"ses"}`))
	SwitchTransactionalEmailProvider(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "ses", updatedValue)
	assert.Contains(t, recorder.Body.String(), `"provider":"ses"`)
}

func TestRetryTransactionalEmailQueueReturnsBoundedResult(t *testing.T) {
	setupEmailDeliveryControllerTest(t)
	retryTransactionalEmails = func(context.Context, int) (emaildelivery.RetryResult, error) {
		return emaildelivery.RetryResult{Processed: 3, Sent: 1, Queued: 1, Failed: 1}, nil
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/option/email-provider/retry-safe", nil)
	RetryTransactionalEmailQueue(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"processed":3`)
	assert.Contains(t, recorder.Body.String(), `"sent":1`)
	assert.Contains(t, recorder.Body.String(), `"queued":1`)
}
