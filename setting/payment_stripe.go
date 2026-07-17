package setting

import (
	"os"
	"strconv"
	"strings"
)

var StripeApiSecret = ""
var StripeWebhookSecret = ""
var StripePriceId = ""
var StripeUnitPrice = 8.0
var StripeMinTopUp = 1
var StripePromotionCodesEnabled = false

// InitStripeEnv loads Stripe secrets and top-up config from environment / Secret Manager names.
// Preferred names (sandbox): STRIPE_TEST_SECRET_KEY, STRIPE_TEST_WEBHOOK_SECRET, STRIPE_TOPUP_PRODUCT_ID.
// Legacy names remain supported: STRIPE_API_SECRET / StripeApiSecret, STRIPE_WEBHOOK_SECRET.
func InitStripeEnv() {
	if v := firstEnv("STRIPE_TEST_SECRET_KEY", "STRIPE_API_SECRET", "StripeApiSecret"); v != "" {
		StripeApiSecret = v
	}
	if v := firstEnv("STRIPE_TEST_WEBHOOK_SECRET", "STRIPE_WEBHOOK_SECRET", "StripeWebhookSecret"); v != "" {
		StripeWebhookSecret = v
	}
	if v := firstEnv("STRIPE_TEST_PUBLISHABLE_KEY", "STRIPE_PUBLISHABLE_KEY"); v != "" {
		StripePublishableKey = v
	}
	if v := firstEnv("STRIPE_TOPUP_PRODUCT_ID", "STRIPE_PRODUCT_ID"); v != "" {
		StripeTopupProductID = v
	}
	if v := firstEnv("STRIPE_PRICE_ID", "StripePriceId"); v != "" {
		StripePriceId = v
	}
	if v := firstEnv("STRIPE_ACCOUNT_ID"); v != "" {
		StripeAccountID = v
	}
	if v := firstEnv("STRIPE_TOPUP_ENABLED"); v != "" {
		StripeTopupEnabled = strings.EqualFold(v, "true") || v == "1"
	}
	// Non-release always require test keys; release can set STRIPE_REQUIRE_TEST_KEYS=false for live.
	if os.Getenv("GIN_MODE") == "release" {
		if v := firstEnv("STRIPE_REQUIRE_TEST_KEYS"); v != "" {
			StripeRequireTestKeys = strings.EqualFold(v, "true") || v == "1"
		} else {
			StripeRequireTestKeys = false
		}
	} else {
		StripeRequireTestKeys = true
	}
	if v := firstEnv("STRIPE_FX_CNY_PER_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			StripeFXCNYPerUSD = f
		}
	}
	if v := firstEnv("STRIPE_FX_CAD_PER_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			StripeFXCADPerUSD = f
		}
	}

	// Reject live keys early when sandbox policy is on.
	if StripeRequireTestKeys {
		if strings.HasPrefix(StripeApiSecret, "sk_live") || strings.HasPrefix(StripeApiSecret, "rk_live") {
			StripeApiSecret = ""
		}
		if strings.HasPrefix(StripePublishableKey, "pk_live") {
			StripePublishableKey = ""
		}
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
