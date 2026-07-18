package setting

import (
	"encoding/json"
	"math"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
)

// Stripe product for dynamic Checkout price_data (no per-amount Price objects).
// The active value is selected from the Test or Production runtime profile.
var StripeTopupProductID = ""

// StripePublishableKey is optional (pk_test_…); never required for server credit path.
var StripePublishableKey = ""

// StripeAccountID is the expected webhook account for the active profile.
// Empty skips the optional account check.
var StripeAccountID = ""

// StripeTopupEnabled gates the new multi-currency Checkout path.
var StripeTopupEnabled = false

// StripeRequireTestKeys when true (default outside GIN_MODE=release) rejects sk_live/pk_live.
var StripeRequireTestKeys = true

// MicroUSDPerUSD is the canonical ledger scale: 1 USD = 1_000_000 micro-USD.
const MicroUSDPerUSD int64 = 1_000_000

// Default FX: units of presentment currency per 1 USD (locked into each order).
var (
	StripeFXCNYPerUSD = 7.3
	StripeFXCADPerUSD = 1.37
)

// Preset amounts in major currency units (not minor).
var (
	StripePresetsCNY = []int{10, 20, 50, 100, 200, 500}
	StripePresetsUSD = []int{2, 5, 10, 20, 50, 100, 200, 500}
	StripePresetsCAD = []int{2, 5, 10, 20, 50, 100, 200, 500}
)

// Min/max presentment major units per currency (¥10 / $2 minimums; 2000 maximums).
var (
	StripeMinMinorCNY int64 = 500
	StripeMaxMinorCNY int64 = 50000
	StripeMinMinorUSD int64 = 50
	StripeMaxMinorUSD int64 = 50000
	StripeMinMinorCAD int64 = 50
	StripeMaxMinorCAD int64 = 50000
)

var (
	stripeTopupMu      sync.RWMutex
	stripeTopupPresets = map[string][]int{}
)

func init() {
	stripeTopupPresets = map[string][]int{
		"cny": append([]int{}, StripePresetsCNY...),
		"usd": append([]int{}, StripePresetsUSD...),
		"cad": append([]int{}, StripePresetsCAD...),
	}
}

// SupportedTopupCurrencies returns lowercase currency codes.
func SupportedTopupCurrencies() []string {
	return EnabledBillingCurrencies()
}

func IsSupportedTopupCurrency(cur string) bool {
	return IsBillingCurrencyEnabled(cur)
}

func GetTopupPresets(currency string) []int {
	stripeTopupMu.RLock()
	defer stripeTopupMu.RUnlock()
	c := strings.ToLower(strings.TrimSpace(currency))
	src := stripeTopupPresets[c]
	out := make([]int, len(src))
	copy(out, src)
	return out
}

func SetTopupPresets(currency string, amounts []int) {
	stripeTopupMu.Lock()
	defer stripeTopupMu.Unlock()
	c := strings.ToLower(strings.TrimSpace(currency))
	if !IsSupportedTopupCurrency(c) {
		return
	}
	cp := make([]int, 0, len(amounts))
	for _, a := range amounts {
		if a > 0 {
			cp = append(cp, a)
		}
	}
	stripeTopupPresets[c] = cp
}

func TopupMinMaxMinor(currency string) (min, max int64) {
	switch strings.ToLower(strings.TrimSpace(currency)) {
	case "cny":
		return StripeMinMinorCNY, StripeMaxMinorCNY
	case "cad":
		return StripeMinMinorCAD, StripeMaxMinorCAD
	default:
		return StripeMinMinorUSD, StripeMaxMinorUSD
	}
}

// FXRatePresentmentPerUSD returns locked rate: presentment currency units per 1 USD.
func FXRatePresentmentPerUSD(currency string) float64 {
	rate := BillingCurrencyFXRate(currency)
	if rate <= 0 {
		return 1
	}
	return rate
}

// PresentmentMinorToMicroUSD converts Stripe minor units to micro-USD using a locked FX rate.
func PresentmentMinorToMicroUSD(currency string, amountMinor int64, fxPresentmentPerUSD float64) int64 {
	if amountMinor <= 0 {
		return 0
	}
	if fxPresentmentPerUSD <= 0 {
		fxPresentmentPerUSD = 1
	}
	// All MVP currencies use 2 decimal places.
	// major = minor/100; usd = major / fx; micro = usd * 1e6
	// micro = amountMinor / 100 / fx * 1e6 = amountMinor * 1e4 / fx
	microUSD := decimal.NewFromInt(amountMinor).
		Mul(decimal.NewFromInt(MicroUSDPerUSD)).
		Div(decimal.NewFromInt(100)).
		Div(decimal.NewFromFloat(fxPresentmentPerUSD)).
		Truncate(0)
	if microUSD.IsNegative() || microUSD.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0
	}
	return microUSD.IntPart()
}

// MicroUSDToQuota maps micro-USD into existing platform quota (QuotaPerUnit per USD).
func MicroUSDToQuota(microUSD int64) int {
	if microUSD <= 0 {
		return 0
	}
	// Keep monetary arithmetic decimal until the centralized quota conversion.
	q := decimal.NewFromInt(microUSD).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Div(decimal.NewFromInt(MicroUSDPerUSD)).
		Truncate(0)
	return common.QuotaFromDecimal(q)
}

// ExportTopupConfigJSON for admin/status (no secrets).
func ExportTopupConfigJSON() json.RawMessage {
	type cfg struct {
		Enabled           bool                  `json:"enabled"`
		ProductID         string                `json:"product_id"`
		AccountID         string                `json:"account_id"`
		RequireTestKeys   bool                  `json:"require_test_keys"`
		Currencies        []string              `json:"currencies"`
		DefaultCurrency   string                `json:"default_currency"`
		Presets           map[string][]int      `json:"presets"`
		MinMaxMajor       map[string][2]float64 `json:"min_max_major"`
		MinMaxMinor       map[string][2]int64   `json:"min_max_minor"`
		FX                map[string]float64    `json:"fx_presentment_per_usd"`
		PublishableKeySet bool                  `json:"publishable_key_set"`
		SecretKeySet      bool                  `json:"secret_key_set"`
		WebhookSecretSet  bool                  `json:"webhook_secret_set"`
		SandboxIndicator  string                `json:"environment"`
	}
	presets := map[string][]int{}
	minmaxMajor := map[string][2]float64{}
	minmaxMinor := map[string][2]int64{}
	fx := map[string]float64{}
	for _, c := range SupportedTopupCurrencies() {
		presets[c] = GetTopupPresets(c)
		mi, ma := TopupMinMaxMinor(c)
		minmaxMinor[c] = [2]int64{mi, ma}
		minmaxMajor[c] = [2]float64{float64(mi) / 100, float64(ma) / 100}
		fx[c] = FXRatePresentmentPerUSD(c)
	}
	env := "disabled"
	if StripeRuntimeEnvironment == StripeRuntimeTest {
		env = "sandbox"
	} else if StripeRuntimeEnvironment == StripeRuntimeProduction {
		env = "live"
	}
	b, _ := common.Marshal(cfg{
		Enabled:           StripeTopupEnabled,
		ProductID:         StripeTopupProductID,
		AccountID:         StripeAccountID,
		RequireTestKeys:   StripeRequireTestKeys,
		Currencies:        SupportedTopupCurrencies(),
		DefaultCurrency:   DefaultBillingCurrency(),
		Presets:           presets,
		MinMaxMajor:       minmaxMajor,
		MinMaxMinor:       minmaxMinor,
		FX:                fx,
		PublishableKeySet: strings.TrimSpace(StripePublishableKey) != "",
		SecretKeySet:      strings.TrimSpace(StripeApiSecret) != "",
		WebhookSecretSet:  strings.TrimSpace(StripeWebhookSecret) != "",
		SandboxIndicator:  env,
	})
	return b
}
