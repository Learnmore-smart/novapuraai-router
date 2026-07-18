package controller

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestStripeCredentialUpdateIsWriteOnlyAndReloadsRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalCryptoSecret := common.CryptoSecret
	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.CryptoSecret = originalCryptoSecret
		common.RedisEnabled = originalRedisEnabled
		setting.ClearStripeCredentialProfile(setting.StripeRuntimeTest)
	})

	var err error
	model.DB, err = gorm.Open(sqlite.Open("file:stripe-credential-controller?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, model.DB.AutoMigrate(&model.StripeCredential{}, &model.User{}, &model.Log{}))
	model.LOG_DB = model.DB
	common.RedisEnabled = false
	common.CryptoSecret = "stripe-controller-test-secret"
	t.Setenv("GIN_MODE", "debug")
	setting.InitStripeEnv()

	body := []byte(`{"secret_key":"sk_test_do_not_return","publishable_key":"pk_test_do_not_return","webhook_secret":"whsec_do_not_return"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("PUT", "/api/option/stripe/test/credentials", bytes.NewReader(body))
	ctx.Params = gin.Params{{Key: "environment", Value: "test"}}
	ctx.Set("id", 1)
	UpdateStripeCredentials(ctx)

	require.Equal(t, 200, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "do_not_return")
	assert.Contains(t, recorder.Body.String(), `"secret_configured":true`)
	assert.Equal(t, "sk_test_do_not_return", setting.StripeApiSecret)
	assert.Equal(t, "pk_test_do_not_return", setting.StripePublishableKey)
	assert.Equal(t, "whsec_do_not_return", setting.StripeWebhookSecret)

	statusRecorder := httptest.NewRecorder()
	statusContext, _ := gin.CreateTestContext(statusRecorder)
	statusContext.Request = httptest.NewRequest("GET", "/api/option/stripe/test/credentials", nil)
	statusContext.Params = gin.Params{{Key: "environment", Value: "test"}}
	GetStripeCredentials(statusContext)
	require.Equal(t, 200, statusRecorder.Code)
	assert.NotContains(t, statusRecorder.Body.String(), "do_not_return")
}

func TestStripeCredentialUpdateRejectsInvalidEnvironmentAndNoOp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		environment string
		body        string
	}{
		{name: "invalid environment", environment: "staging", body: `{"secret_key":"sk_test_value"}`},
		{name: "no changes", environment: "test", body: `{}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest("PUT", "/api/option/stripe/credentials", bytes.NewBufferString(testCase.body))
			ctx.Params = gin.Params{{Key: "environment", Value: testCase.environment}}
			UpdateStripeCredentials(ctx)
			assert.Equal(t, 400, recorder.Code)
		})
	}
}
