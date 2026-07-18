package setting

import (
	"os"
	"strings"
)

const (
	StripeRuntimeTest       = "test"
	StripeRuntimeProduction = "production"
	StripeRuntimeDisabled   = "disabled"
)

type StripeCredentialConfiguration struct {
	TestSecretConfigured      bool
	TestPublishableConfigured bool
	TestWebhookConfigured     bool
	ProdSecretConfigured      bool
	ProdPublishableConfigured bool
	ProdWebhookConfigured     bool
}

type stripeCredentialProfile struct {
	apiSecret      string
	publishableKey string
	webhookSecret  string
}

var StripeApiSecret = ""
var StripeWebhookSecret = ""
var StripePriceId = ""
var StripeUnitPrice = 8.0
var StripeMinTopUp = 1
var StripePromotionCodesEnabled = false
var StripeRuntimeEnvironment = StripeRuntimeDisabled
var StripeTestTopupProductID = ""
var StripeProdTopupProductID = ""
var StripeTestAccountID = ""
var StripeProdAccountID = ""

var stripeTestCredentials stripeCredentialProfile
var stripeProdCredentials stripeCredentialProfile

// InitStripeEnv initializes the runtime selector without reading Stripe values
// from the process environment. Stripe credentials are loaded from encrypted
// database rows after database initialization.
func InitStripeEnv() {
	stripeTestCredentials = stripeCredentialProfile{}
	stripeProdCredentials = stripeCredentialProfile{}
	ApplyStripeRuntimeProfile()
}

func SetStripeCredentialProfile(environment string, apiSecret string, publishableKey string, webhookSecret string) {
	profile := stripeCredentialProfile{
		apiSecret:      strings.TrimSpace(apiSecret),
		publishableKey: strings.TrimSpace(publishableKey),
		webhookSecret:  strings.TrimSpace(webhookSecret),
	}
	if environment == StripeRuntimeProduction {
		stripeProdCredentials = profile
	} else if environment == StripeRuntimeTest {
		stripeTestCredentials = profile
	}
	ApplyStripeRuntimeProfile()
}

func ClearStripeCredentialProfile(environment string) {
	SetStripeCredentialProfile(environment, "", "", "")
}

// ApplyStripeRuntimeProfile activates the already-loaded credential profile and
// dashboard-managed identifiers. It never reloads non-secret values from env.
func ApplyStripeRuntimeProfile() {
	StripeApiSecret = ""
	StripePublishableKey = ""
	StripeWebhookSecret = ""
	StripeTopupProductID = ""
	StripeAccountID = ""

	if os.Getenv("GIN_MODE") != "release" {
		StripeRuntimeEnvironment = StripeRuntimeTest
		StripeRequireTestKeys = true
		StripeApiSecret = stripeTestCredentials.apiSecret
		StripePublishableKey = stripeTestCredentials.publishableKey
		StripeWebhookSecret = stripeTestCredentials.webhookSecret
		StripeTopupProductID = StripeTestTopupProductID
		StripeAccountID = StripeTestAccountID
		applyStripeTestKeyPolicy()
		return
	}

	StripeRuntimeEnvironment = StripeRuntimeProduction
	StripeRequireTestKeys = false
	StripeApiSecret = stripeProdCredentials.apiSecret
	StripePublishableKey = stripeProdCredentials.publishableKey
	StripeWebhookSecret = stripeProdCredentials.webhookSecret
	StripeTopupProductID = StripeProdTopupProductID
	StripeAccountID = StripeProdAccountID
}

func GetStripeCredentialConfiguration() StripeCredentialConfiguration {
	return StripeCredentialConfiguration{
		TestSecretConfigured:      strings.TrimSpace(stripeTestCredentials.apiSecret) != "",
		TestPublishableConfigured: strings.TrimSpace(stripeTestCredentials.publishableKey) != "",
		TestWebhookConfigured:     strings.TrimSpace(stripeTestCredentials.webhookSecret) != "",
		ProdSecretConfigured:      strings.TrimSpace(stripeProdCredentials.apiSecret) != "",
		ProdPublishableConfigured: strings.TrimSpace(stripeProdCredentials.publishableKey) != "",
		ProdWebhookConfigured:     strings.TrimSpace(stripeProdCredentials.webhookSecret) != "",
	}
}

func applyStripeTestKeyPolicy() {
	if strings.HasPrefix(StripeApiSecret, "sk_live") || strings.HasPrefix(StripeApiSecret, "rk_live") {
		StripeApiSecret = ""
	}
	if strings.HasPrefix(StripePublishableKey, "pk_live") {
		StripePublishableKey = ""
	}
}
