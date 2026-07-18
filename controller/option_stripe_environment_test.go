package controller

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOptionsReturnsStripeConfigurationStatusWithoutSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalRuntime := setting.StripeRuntimeEnvironment
	t.Cleanup(func() {
		setting.RefreshStripeSecretsFromEnv()
		setting.StripeRuntimeEnvironment = originalRuntime
	})

	t.Setenv("STRIPE_TEST_SECRET_KEY", "sk_test_do_not_return")
	t.Setenv("STRIPE_TEST_PUBLISHABLE_KEY", "pk_test_do_not_return")
	t.Setenv("STRIPE_TEST_WEBHOOK_SECRET", "whsec_test_do_not_return")
	t.Setenv("STRIPE_PROD_SECRET_KEY", "sk_live_do_not_return")
	t.Setenv("STRIPE_PROD_PUBLISHABLE_KEY", "")
	t.Setenv("STRIPE_PROD_WEBHOOK_SECRET", "")
	setting.StripeRuntimeEnvironment = setting.StripeRuntimeTest
	setting.RefreshStripeSecretsFromEnv()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	GetOptions(ctx)

	require.Equal(t, 200, recorder.Code)
	var response struct {
		Success bool            `json:"success"`
		Data    []*model.Option `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	returned := make(map[string]string, len(response.Data))
	for _, option := range response.Data {
		returned[option.Key] = option.Value
	}
	assert.Equal(t, setting.StripeRuntimeTest, returned["StripeRuntimeEnvironment"])
	assert.Equal(t, "true", returned["StripeTestSecretConfigured"])
	assert.Equal(t, "true", returned["StripeTestPublishableConfigured"])
	assert.Equal(t, "true", returned["StripeTestWebhookConfigured"])
	assert.Equal(t, "true", returned["StripeProdSecretConfigured"])
	assert.Equal(t, "false", returned["StripeProdPublishableConfigured"])
	assert.Equal(t, "false", returned["StripeProdWebhookConfigured"])
	assert.NotContains(t, recorder.Body.String(), "do_not_return")
	for key := range returned {
		assert.False(t, strings.HasSuffix(key, "SecretKey"))
		assert.False(t, strings.HasSuffix(key, "WebhookSecret"))
	}
}
