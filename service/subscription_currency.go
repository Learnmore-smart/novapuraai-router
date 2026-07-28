package service

import (
	"strings"

	"github.com/QuantumNous/new-api/setting"
)

// SubscriptionCurrencyCNY/USD are the only two currencies a NovaPura
// subscription plan can be priced in (see SubscriptionPlan.PriceAmountCNY/USD
// and StripePriceIdCNY/USD). They are uppercase to match the plan's Currency
// column default ('USD') and the Stripe Checkout currency convention.
const (
	SubscriptionCurrencyCNY = "CNY"
	SubscriptionCurrencyUSD = "USD"
)

// ResolveSubscriptionCurrency picks the subscription currency for a user.
// Priority: explicit user setting > region-based system default > USD.
//
// The user setting and setting.DefaultBillingCurrency() both use lowercase
// codes ("cny"/"usd"/"cad"); NovaPura subscription plans only support CNY and
// USD, so any other value (including CAD) falls back to the system default and
// finally to USD. The return value is always uppercase to match the
// SubscriptionPlan.Currency column and Stripe's ISO 4217 convention.
func ResolveSubscriptionCurrency(userSettingBillingCurrency string) string {
	if uc := normalizeSubscriptionCurrency(userSettingBillingCurrency); uc != "" {
		return uc
	}
	if uc := normalizeSubscriptionCurrency(setting.DefaultBillingCurrency()); uc != "" {
		return uc
	}
	return SubscriptionCurrencyUSD
}

// normalizeSubscriptionCurrency upper-cases and trims the input, returning the
// canonical "CNY"/"USD" form when it is one of the supported subscription
// currencies, or "" otherwise.
func normalizeSubscriptionCurrency(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case SubscriptionCurrencyCNY:
		return SubscriptionCurrencyCNY
	case SubscriptionCurrencyUSD:
		return SubscriptionCurrencyUSD
	}
	return ""
}
