package setting

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

// Stripe sandbox product for dynamic Checkout price_data (no per-amount Price objects).
// Created in acct_1Tta8mPKe8UWYDw1 (NovaPuraAI 沙盒).
var StripeTopupProductID = ""

// StripePublishableKey is optional (pk_test_…); never required for server credit path.
var StripePublishableKey = ""

// StripeAccountID expected webhook account (sandbox). Empty = skip account check.
var StripeAccountID = "acct_1Tta8mPKe8UWYDw1"

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
	StripePresetsCNY = []int{5, 10, 30, 50, 100, 200}
	StripePresetsUSD = []int{1, 5, 10, 20, 50, 100}
	StripePresetsCAD = []int{2, 5, 10, 20, 50, 100}
)

// Min/max presentment major units per currency (~¥5 / ~$1 and ~¥2000 / ~$2000).
var (
	StripeMinMajorCNY = 5
	StripeMaxMajorCNY = 2000
	StripeMinMajorUSD = 1
	StripeMaxMajorUSD = 2000
	StripeMinMajorCAD = 2
	StripeMaxMajorCAD = 2000
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
	return []string{"cny", "usd", "cad"}
}

func IsSupportedTopupCurrency(cur string) bool {
	switch strings.ToLower(strings.TrimSpace(cur)) {
	case "cny", "usd", "cad":
		return true
	default:
		return false
	}
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

func TopupMinMaxMajor(currency string) (min, max int) {
	switch strings.ToLower(strings.TrimSpace(currency)) {
	case "cny":
		return StripeMinMajorCNY, StripeMaxMajorCNY
	case "cad":
		return StripeMinMajorCAD, StripeMaxMajorCAD
	default:
		return StripeMinMajorUSD, StripeMaxMajorUSD
	}
}

// FXRatePresentmentPerUSD returns locked rate: presentment currency units per 1 USD.
func FXRatePresentmentPerUSD(currency string) float64 {
	switch strings.ToLower(strings.TrimSpace(currency)) {
	case "cny":
		if StripeFXCNYPerUSD <= 0 {
			return 7.3
		}
		return StripeFXCNYPerUSD
	case "cad":
		if StripeFXCADPerUSD <= 0 {
			return 1.37
		}
		return StripeFXCADPerUSD
	default:
		return 1.0
	}
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
	usd := float64(amountMinor) / 100.0 / fxPresentmentPerUSD
	return int64(common.QuotaFromFloat(usd * float64(MicroUSDPerUSD)))
}

// MicroUSDToQuota maps micro-USD into existing platform quota (QuotaPerUnit per USD).
func MicroUSDToQuota(microUSD int64) int {
	if microUSD <= 0 {
		return 0
	}
	// quota = microUSD / 1e6 * QuotaPerUnit
	q := float64(microUSD) / float64(MicroUSDPerUSD) * common.QuotaPerUnit
	return common.QuotaFromFloat(q)
}

// ExportTopupConfigJSON for admin/status (no secrets).
func ExportTopupConfigJSON() json.RawMessage {
	type cfg struct {
		Enabled          bool              `json:"enabled"`
		ProductID        string            `json:"product_id"`
		AccountID        string            `json:"account_id"`
		RequireTestKeys  bool              `json:"require_test_keys"`
		Currencies       []string          `json:"currencies"`
		Presets          map[string][]int  `json:"presets"`
		MinMax           map[string][2]int `json:"min_max_major"`
		FX               map[string]float64 `json:"fx_presentment_per_usd"`
		PublishableKeySet bool             `json:"publishable_key_set"`
		SecretKeySet     bool              `json:"secret_key_set"`
		WebhookSecretSet bool              `json:"webhook_secret_set"`
		SandboxIndicator string            `json:"environment"`
	}
	presets := map[string][]int{}
	minmax := map[string][2]int{}
	fx := map[string]float64{}
	for _, c := range SupportedTopupCurrencies() {
		presets[c] = GetTopupPresets(c)
		mi, ma := TopupMinMaxMajor(c)
		minmax[c] = [2]int{mi, ma}
		fx[c] = FXRatePresentmentPerUSD(c)
	}
	env := "sandbox"
	if strings.HasPrefix(StripeApiSecret, "sk_live") {
		env = "live"
	}
	b, _ := common.Marshal(cfg{
		Enabled:           StripeTopupEnabled,
		ProductID:         StripeTopupProductID,
		AccountID:         StripeAccountID,
		RequireTestKeys:   StripeRequireTestKeys,
		Currencies:        SupportedTopupCurrencies(),
		Presets:           presets,
		MinMax:            minmax,
		FX:                fx,
		PublishableKeySet: strings.TrimSpace(StripePublishableKey) != "",
		SecretKeySet:      strings.TrimSpace(StripeApiSecret) != "",
		WebhookSecretSet:  strings.TrimSpace(StripeWebhookSecret) != "",
		SandboxIndicator:  env,
	})
	return b
}
