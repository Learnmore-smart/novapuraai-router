package setting

import (
	"os"
	"strconv"
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

// InitStripeEnv loads Stripe secrets and top-up config from environment / Secret Manager names.
// Test credentials are used outside release. Production credentials are used in
// release mode; session cookie origin configuration remains independent.
func InitStripeEnv() {
	loadStripeCredentialsFromEnv()
	StripeTestTopupProductID = firstEnv("STRIPE_TEST_TOPUP_PRODUCT_ID", "STRIPE_TOPUP_PRODUCT_ID", "STRIPE_PRODUCT_ID")
	StripeProdTopupProductID = firstEnv("STRIPE_PROD_TOPUP_PRODUCT_ID")
	StripeTestAccountID = firstEnv("STRIPE_TEST_ACCOUNT_ID", "STRIPE_ACCOUNT_ID")
	StripeProdAccountID = firstEnv("STRIPE_PROD_ACCOUNT_ID")
	if v := firstEnv("STRIPE_PRICE_ID", "StripePriceId"); v != "" {
		StripePriceId = v
	}
	if v := firstEnv("STRIPE_TOPUP_ENABLED"); v != "" {
		StripeTopupEnabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := firstEnv("STRIPE_FX_CNY_PER_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			StripeFXCNYPerUSD = f
			_ = SetBillingCurrencyFXRate(BillingCurrencyCNY, f)
		}
	}
	if v := firstEnv("STRIPE_FX_CAD_PER_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			StripeFXCADPerUSD = f
			_ = SetBillingCurrencyFXRate(BillingCurrencyCAD, f)
		}
	}

	ApplyStripeRuntimeProfile()
}

// RefreshStripeSecretsFromEnv reapplies only credential values after database-backed
// options are loaded, so persisted wallet top-up settings remain authoritative.
func RefreshStripeSecretsFromEnv() {
	loadStripeCredentialsFromEnv()
	ApplyStripeRuntimeProfile()
}

func loadStripeCredentialsFromEnv() {
	stripeTestCredentials = stripeCredentialProfile{
		apiSecret:      firstEnv("STRIPE_TEST_SECRET_KEY", "STRIPE_API_SECRET", "StripeApiSecret"),
		publishableKey: firstEnv("STRIPE_TEST_PUBLISHABLE_KEY", "STRIPE_PUBLISHABLE_KEY"),
		webhookSecret:  firstEnv("STRIPE_TEST_WEBHOOK_SECRET", "STRIPE_WEBHOOK_SECRET", "StripeWebhookSecret"),
	}
	stripeProdCredentials = stripeCredentialProfile{
		apiSecret:      firstEnv("STRIPE_PROD_SECRET_KEY"),
		publishableKey: firstEnv("STRIPE_PROD_PUBLISHABLE_KEY"),
		webhookSecret:  firstEnv("STRIPE_PROD_WEBHOOK_SECRET"),
	}
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

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}
