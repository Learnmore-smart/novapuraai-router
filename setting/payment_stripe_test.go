package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshStripeSecretsFromEnvPreservesDashboardTopupOptions(t *testing.T) {
	originalProductID := StripeTopupProductID
	originalEnabled := StripeTopupEnabled
	originalSecret := StripeApiSecret
	t.Cleanup(func() {
		StripeTopupProductID = originalProductID
		StripeTopupEnabled = originalEnabled
		StripeApiSecret = originalSecret
	})

	t.Setenv("STRIPE_TOPUP_PRODUCT_ID", "prod_env")
	t.Setenv("STRIPE_TOPUP_ENABLED", "true")
	t.Setenv("STRIPE_TEST_SECRET_KEY", "sk_test_env")

	StripeTopupProductID = ""
	StripeTopupEnabled = false
	StripeApiSecret = ""
	InitStripeEnv()
	require.Equal(t, "prod_env", StripeTopupProductID)
	require.True(t, StripeTopupEnabled)

	StripeTopupProductID = "prod_dashboard"
	StripeTopupEnabled = false
	RefreshStripeSecretsFromEnv()

	assert.Equal(t, "prod_dashboard", StripeTopupProductID)
	assert.False(t, StripeTopupEnabled)
	assert.Equal(t, "sk_test_env", StripeApiSecret)
}
