package setting

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitStripeEnvDoesNotReadStripeConfiguration(t *testing.T) {
	snapshot := snapshotStripeRuntimeForTest()
	t.Cleanup(func() { restoreStripeRuntimeForTest(snapshot) })

	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STRIPE_TEST_SECRET_KEY", "sk_test_must_be_ignored")
	t.Setenv("STRIPE_TEST_PUBLISHABLE_KEY", "pk_test_must_be_ignored")
	t.Setenv("STRIPE_TEST_WEBHOOK_SECRET", "whsec_must_be_ignored")
	t.Setenv("STRIPE_TEST_ACCOUNT_ID", "acct_must_be_ignored")
	t.Setenv("STRIPE_TEST_TOPUP_PRODUCT_ID", "prod_must_be_ignored")
	t.Setenv("STRIPE_TOPUP_ENABLED", "true")

	StripeTestAccountID = ""
	StripeTestTopupProductID = ""
	StripeTopupEnabled = false
	InitStripeEnv()

	assert.Equal(t, StripeRuntimeTest, StripeRuntimeEnvironment)
	assert.Empty(t, StripeApiSecret)
	assert.Empty(t, StripePublishableKey)
	assert.Empty(t, StripeWebhookSecret)
	assert.Empty(t, StripeAccountID)
	assert.Empty(t, StripeTopupProductID)
	assert.False(t, StripeTopupEnabled)
	assert.Equal(t, "sk_test_must_be_ignored", os.Getenv("STRIPE_TEST_SECRET_KEY"))
}

func TestDatabaseCredentialProfilesFollowRuntimeEnvironment(t *testing.T) {
	snapshot := snapshotStripeRuntimeForTest()
	t.Cleanup(func() { restoreStripeRuntimeForTest(snapshot) })

	t.Setenv("GIN_MODE", "debug")
	InitStripeEnv()
	StripeTestAccountID = "acct_test"
	StripeTestTopupProductID = "prod_test"
	SetStripeCredentialProfile(StripeRuntimeTest, "rk_test_secret", "pk_test_public", "whsec_test")
	SetStripeCredentialProfile(StripeRuntimeProduction, "rk_live_secret", "pk_live_public", "whsec_live")

	assert.Equal(t, StripeRuntimeTest, StripeRuntimeEnvironment)
	assert.Equal(t, "rk_test_secret", StripeApiSecret)
	assert.Equal(t, "pk_test_public", StripePublishableKey)
	assert.Equal(t, "whsec_test", StripeWebhookSecret)
	assert.Equal(t, "acct_test", StripeAccountID)
	assert.Equal(t, "prod_test", StripeTopupProductID)

	t.Setenv("GIN_MODE", "release")
	StripeProdAccountID = "acct_live"
	StripeProdTopupProductID = "prod_live"
	ApplyStripeRuntimeProfile()

	assert.Equal(t, StripeRuntimeProduction, StripeRuntimeEnvironment)
	assert.Equal(t, "rk_live_secret", StripeApiSecret)
	assert.Equal(t, "pk_live_public", StripePublishableKey)
	assert.Equal(t, "whsec_live", StripeWebhookSecret)
	assert.Equal(t, "acct_live", StripeAccountID)
	assert.Equal(t, "prod_live", StripeTopupProductID)
}

func TestApplyStripeRuntimeProfileDoesNotReloadEnvironment(t *testing.T) {
	snapshot := snapshotStripeRuntimeForTest()
	t.Cleanup(func() { restoreStripeRuntimeForTest(snapshot) })

	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STRIPE_TEST_SECRET_KEY", "sk_test_must_be_ignored")
	InitStripeEnv()
	SetStripeCredentialProfile(StripeRuntimeTest, "rk_test_database", "pk_test_database", "whsec_database")
	StripeTestTopupProductID = "prod_dashboard"
	StripeTestAccountID = "acct_dashboard"
	ApplyStripeRuntimeProfile()

	assert.Equal(t, "rk_test_database", StripeApiSecret)
	assert.Equal(t, "pk_test_database", StripePublishableKey)
	assert.Equal(t, "whsec_database", StripeWebhookSecret)
	assert.Equal(t, "prod_dashboard", StripeTopupProductID)
	assert.Equal(t, "acct_dashboard", StripeAccountID)
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
	testCredentials    stripeCredentialProfile
	prodCredentials    stripeCredentialProfile
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
		testCredentials:    stripeTestCredentials,
		prodCredentials:    stripeProdCredentials,
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
	stripeTestCredentials = snapshot.testCredentials
	stripeProdCredentials = snapshot.prodCredentials
}
