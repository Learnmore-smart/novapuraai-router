package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitStripeEnvSelectsTestProfileLocally(t *testing.T) {
	original := snapshotStripeRuntimeForTest()
	t.Cleanup(func() { restoreStripeRuntimeForTest(original) })

	t.Setenv("GIN_MODE", "debug")
	t.Setenv("SESSION_COOKIE_TRUSTED_URL", "http://localhost:3000")
	t.Setenv("STRIPE_TEST_SECRET_KEY", "sk_test_local")
	t.Setenv("STRIPE_TEST_PUBLISHABLE_KEY", "pk_test_local")
	t.Setenv("STRIPE_TEST_WEBHOOK_SECRET", "whsec_test_local")
	t.Setenv("STRIPE_PROD_SECRET_KEY", "sk_live_prod")
	t.Setenv("STRIPE_PROD_PUBLISHABLE_KEY", "pk_live_prod")
	t.Setenv("STRIPE_PROD_WEBHOOK_SECRET", "whsec_prod")
	t.Setenv("STRIPE_TEST_TOPUP_PRODUCT_ID", "prod_test")
	t.Setenv("STRIPE_PROD_TOPUP_PRODUCT_ID", "prod_live")
	t.Setenv("STRIPE_TEST_ACCOUNT_ID", "acct_test")
	t.Setenv("STRIPE_PROD_ACCOUNT_ID", "acct_live")

	InitStripeEnv()

	assert.Equal(t, StripeRuntimeTest, StripeRuntimeEnvironment)
	assert.Equal(t, "sk_test_local", StripeApiSecret)
	assert.Equal(t, "pk_test_local", StripePublishableKey)
	assert.Equal(t, "whsec_test_local", StripeWebhookSecret)
	assert.Equal(t, "prod_test", StripeTopupProductID)
	assert.Equal(t, "acct_test", StripeAccountID)
}

func TestInitStripeEnvSelectsProductionProfileOnTrustedDomain(t *testing.T) {
	original := snapshotStripeRuntimeForTest()
	t.Cleanup(func() { restoreStripeRuntimeForTest(original) })

	t.Setenv("GIN_MODE", "release")
	t.Setenv("SESSION_COOKIE_TRUSTED_URL", "https://www.novapuraai.com")
	t.Setenv("STRIPE_TEST_SECRET_KEY", "sk_test_local")
	t.Setenv("STRIPE_PROD_SECRET_KEY", "sk_live_prod")
	t.Setenv("STRIPE_PROD_PUBLISHABLE_KEY", "pk_live_prod")
	t.Setenv("STRIPE_PROD_WEBHOOK_SECRET", "whsec_prod")
	t.Setenv("STRIPE_PROD_TOPUP_PRODUCT_ID", "prod_live")
	t.Setenv("STRIPE_PROD_ACCOUNT_ID", "acct_live")

	InitStripeEnv()

	assert.Equal(t, StripeRuntimeProduction, StripeRuntimeEnvironment)
	assert.Equal(t, "sk_live_prod", StripeApiSecret)
	assert.Equal(t, "pk_live_prod", StripePublishableKey)
	assert.Equal(t, "whsec_prod", StripeWebhookSecret)
	assert.Equal(t, "prod_live", StripeTopupProductID)
	assert.Equal(t, "acct_live", StripeAccountID)
}

func TestInitStripeEnvSelectsProductionProfileWithoutTrustedCookieURL(t *testing.T) {
	original := snapshotStripeRuntimeForTest()
	t.Cleanup(func() { restoreStripeRuntimeForTest(original) })

	t.Setenv("GIN_MODE", "release")
	t.Setenv("SESSION_COOKIE_TRUSTED_URL", "https://staging.example.com")
	t.Setenv("STRIPE_TEST_SECRET_KEY", "sk_test_local")
	t.Setenv("STRIPE_PROD_SECRET_KEY", "sk_live_prod")
	t.Setenv("STRIPE_PROD_PUBLISHABLE_KEY", "pk_live_prod")
	t.Setenv("STRIPE_PROD_WEBHOOK_SECRET", "whsec_prod")
	t.Setenv("STRIPE_PROD_TOPUP_PRODUCT_ID", "prod_live")
	t.Setenv("STRIPE_PROD_ACCOUNT_ID", "acct_live")

	InitStripeEnv()

	assert.Equal(t, StripeRuntimeProduction, StripeRuntimeEnvironment)
	assert.Equal(t, "sk_live_prod", StripeApiSecret)
	assert.Equal(t, "pk_live_prod", StripePublishableKey)
	assert.Equal(t, "whsec_prod", StripeWebhookSecret)
	assert.Equal(t, "prod_live", StripeTopupProductID)
	assert.Equal(t, "acct_live", StripeAccountID)
}

func TestRefreshStripeSecretsFromEnvPreservesDashboardTopupOptions(t *testing.T) {
	original := snapshotStripeRuntimeForTest()
	t.Cleanup(func() { restoreStripeRuntimeForTest(original) })

	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STRIPE_TEST_TOPUP_PRODUCT_ID", "prod_env")
	t.Setenv("STRIPE_TOPUP_ENABLED", "true")
	t.Setenv("STRIPE_TEST_SECRET_KEY", "sk_test_env")

	StripeTopupProductID = ""
	StripeTopupEnabled = false
	StripeApiSecret = ""
	InitStripeEnv()
	require.Equal(t, "prod_env", StripeTopupProductID)
	require.True(t, StripeTopupEnabled)

	StripeTestTopupProductID = "prod_dashboard"
	StripeTestAccountID = "acct_dashboard"
	StripeTopupEnabled = false
	RefreshStripeSecretsFromEnv()

	assert.Equal(t, "prod_dashboard", StripeTopupProductID)
	assert.Equal(t, "acct_dashboard", StripeAccountID)
	assert.False(t, StripeTopupEnabled)
	assert.Equal(t, "sk_test_env", StripeApiSecret)
}

type stripeRuntimeSnapshot struct {
	apiSecret          string
	webhookSecret      string
	publishableKey     string
	topupProductID     string
	accountID          string
	testTopupProductID string
	prodTopupProductID string
	testAccountID      string
	prodAccountID      string
	runtimeEnvironment string
	topupEnabled       bool
	requireTestKeys    bool
}

func snapshotStripeRuntimeForTest() stripeRuntimeSnapshot {
	return stripeRuntimeSnapshot{
		apiSecret:          StripeApiSecret,
		webhookSecret:      StripeWebhookSecret,
		publishableKey:     StripePublishableKey,
		topupProductID:     StripeTopupProductID,
		accountID:          StripeAccountID,
		testTopupProductID: StripeTestTopupProductID,
		prodTopupProductID: StripeProdTopupProductID,
		testAccountID:      StripeTestAccountID,
		prodAccountID:      StripeProdAccountID,
		runtimeEnvironment: StripeRuntimeEnvironment,
		topupEnabled:       StripeTopupEnabled,
		requireTestKeys:    StripeRequireTestKeys,
	}
}

func restoreStripeRuntimeForTest(snapshot stripeRuntimeSnapshot) {
	StripeApiSecret = snapshot.apiSecret
	StripeWebhookSecret = snapshot.webhookSecret
	StripePublishableKey = snapshot.publishableKey
	StripeTopupProductID = snapshot.topupProductID
	StripeAccountID = snapshot.accountID
	StripeTestTopupProductID = snapshot.testTopupProductID
	StripeProdTopupProductID = snapshot.prodTopupProductID
	StripeTestAccountID = snapshot.testAccountID
	StripeProdAccountID = snapshot.prodAccountID
	StripeRuntimeEnvironment = snapshot.runtimeEnvironment
	StripeTopupEnabled = snapshot.topupEnabled
	StripeRequireTestKeys = snapshot.requireTestKeys
}
