package setting

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

const (
	BillingCurrencyCNY = "cny"
	BillingCurrencyUSD = "usd"
	BillingCurrencyCAD = "cad"
)

var billingCurrencyOrder = []string{
	BillingCurrencyCNY,
	BillingCurrencyUSD,
	BillingCurrencyCAD,
}

type BillingCurrencyDefinition struct {
	Enabled             bool    `json:"enabled"`
	FXPresentmentPerUSD float64 `json:"fx_presentment_per_usd"`
}

type BillingCurrencyConfig struct {
	DefaultCurrency     string                               `json:"default_currency"`
	AutoUpdateFX        bool                                 `json:"auto_update_fx"`
	FXSource            string                               `json:"fx_source,omitempty"`
	FXUpdatedAt         int64                                `json:"fx_updated_at,omitempty"`
	ReferenceCurrencies map[string]float64                   `json:"reference_fx_presentment_per_usd,omitempty"`
	Currencies          map[string]BillingCurrencyDefinition `json:"currencies"`
}

var (
	billingCurrencyMu     sync.RWMutex
	billingCurrencyConfig = BillingCurrencyConfig{
		DefaultCurrency: BillingCurrencyCNY,
		AutoUpdateFX:    true,
		ReferenceCurrencies: map[string]float64{
			BillingCurrencyCNY: 7.3,
			BillingCurrencyUSD: 1,
			BillingCurrencyCAD: 1.37,
		},
		Currencies: map[string]BillingCurrencyDefinition{
			BillingCurrencyCNY: {Enabled: true, FXPresentmentPerUSD: 7.3},
			BillingCurrencyUSD: {Enabled: true, FXPresentmentPerUSD: 1},
			BillingCurrencyCAD: {Enabled: true, FXPresentmentPerUSD: 1.37},
		},
	}
)

func IsSupportedBillingCurrency(currency string) bool {
	switch strings.ToLower(strings.TrimSpace(currency)) {
	case BillingCurrencyCNY, BillingCurrencyUSD, BillingCurrencyCAD:
		return true
	default:
		return false
	}
}

func AllBillingCurrencies() []string {
	return append([]string(nil), billingCurrencyOrder...)
}

func GetBillingCurrencyConfig() BillingCurrencyConfig {
	billingCurrencyMu.RLock()
	defer billingCurrencyMu.RUnlock()
	return cloneBillingCurrencyConfig(billingCurrencyConfig)
}

func cloneBillingCurrencyConfig(config BillingCurrencyConfig) BillingCurrencyConfig {
	cloned := BillingCurrencyConfig{
		DefaultCurrency:     config.DefaultCurrency,
		AutoUpdateFX:        config.AutoUpdateFX,
		FXSource:            config.FXSource,
		FXUpdatedAt:         config.FXUpdatedAt,
		ReferenceCurrencies: make(map[string]float64, len(config.ReferenceCurrencies)),
		Currencies:          make(map[string]BillingCurrencyDefinition, len(config.Currencies)),
	}
	for currency, rate := range config.ReferenceCurrencies {
		cloned.ReferenceCurrencies[currency] = rate
	}
	for currency, definition := range config.Currencies {
		cloned.Currencies[currency] = definition
	}
	return cloned
}

func BillingCurrencyConfigJSON() string {
	data, err := common.Marshal(GetBillingCurrencyConfig())
	if err != nil {
		common.SysError("marshal billing currency config: " + err.Error())
		return `{"default_currency":"cny","currencies":{}}`
	}
	return string(data)
}

func UpdateBillingCurrencyConfigByJSON(raw string) error {
	next, err := parseBillingCurrencyConfigJSON(raw)
	if err != nil {
		return err
	}

	billingCurrencyMu.Lock()
	billingCurrencyConfig = cloneBillingCurrencyConfig(next)
	billingCurrencyMu.Unlock()
	return nil
}

func ValidateBillingCurrencyConfigJSON(raw string) error {
	_, err := parseBillingCurrencyConfigJSON(raw)
	return err
}

func parseBillingCurrencyConfigJSON(raw string) (BillingCurrencyConfig, error) {
	var next BillingCurrencyConfig
	if err := common.UnmarshalJsonStr(raw, &next); err != nil {
		return BillingCurrencyConfig{}, fmt.Errorf("invalid billing currency config: %w", err)
	}
	var fields map[string]any
	if err := common.UnmarshalJsonStr(raw, &fields); err != nil {
		return BillingCurrencyConfig{}, fmt.Errorf("invalid billing currency config: %w", err)
	}
	if _, exists := fields["auto_update_fx"]; !exists {
		next.AutoUpdateFX = true
	}
	if err := normalizeAndValidateBillingCurrencyConfig(&next); err != nil {
		return BillingCurrencyConfig{}, err
	}
	return next, nil
}

func ApplyBillingCurrencyFXRates(rates map[string]float64, source string, updatedAt int64) (string, error) {
	config := GetBillingCurrencyConfig()
	config.ReferenceCurrencies = make(map[string]float64, len(billingCurrencyOrder))
	for _, currency := range billingCurrencyOrder {
		rate, ok := rates[currency]
		if !ok || rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
			return "", fmt.Errorf("%s exchange rate must be a finite positive number", strings.ToUpper(currency))
		}
		if currency == BillingCurrencyUSD && math.Abs(rate-1) > 0.00000001 {
			return "", fmt.Errorf("USD exchange rate must equal 1")
		}
		config.ReferenceCurrencies[currency] = rate
		if config.AutoUpdateFX {
			definition := config.Currencies[currency]
			definition.FXPresentmentPerUSD = rate
			config.Currencies[currency] = definition
		}
	}
	config.FXSource = strings.TrimSpace(source)
	if config.FXSource == "" {
		return "", fmt.Errorf("exchange rate source is required")
	}
	if updatedAt <= 0 {
		return "", fmt.Errorf("exchange rate update time is required")
	}
	config.FXUpdatedAt = updatedAt
	data, err := common.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeAndValidateBillingCurrencyConfig(config *BillingCurrencyConfig) error {
	if config == nil {
		return fmt.Errorf("billing currency config is required")
	}
	config.DefaultCurrency = strings.ToLower(strings.TrimSpace(config.DefaultCurrency))
	if !IsSupportedBillingCurrency(config.DefaultCurrency) {
		return fmt.Errorf("unsupported default currency %q", config.DefaultCurrency)
	}
	if config.Currencies == nil {
		return fmt.Errorf("at least one billing currency is required")
	}

	normalized := make(map[string]BillingCurrencyDefinition, len(config.Currencies))
	for rawCurrency, definition := range config.Currencies {
		currency := strings.ToLower(strings.TrimSpace(rawCurrency))
		if !IsSupportedBillingCurrency(currency) {
			return fmt.Errorf("unsupported currency %q", rawCurrency)
		}
		if definition.FXPresentmentPerUSD <= 0 || math.IsNaN(definition.FXPresentmentPerUSD) || math.IsInf(definition.FXPresentmentPerUSD, 0) {
			return fmt.Errorf("%s exchange rate must be a finite positive number", strings.ToUpper(currency))
		}
		if currency == BillingCurrencyUSD && math.Abs(definition.FXPresentmentPerUSD-1) > 0.00000001 {
			return fmt.Errorf("USD exchange rate must equal 1")
		}
		normalized[currency] = definition
	}
	for _, currency := range billingCurrencyOrder {
		if _, ok := normalized[currency]; ok {
			continue
		}
		fallback := billingCurrencyConfig.Currencies[currency]
		fallback.Enabled = false
		normalized[currency] = fallback
	}
	defaultDefinition := normalized[config.DefaultCurrency]
	if !defaultDefinition.Enabled {
		return fmt.Errorf("default currency must be enabled")
	}
	config.Currencies = normalized
	if config.ReferenceCurrencies == nil {
		config.ReferenceCurrencies = make(map[string]float64, len(billingCurrencyOrder))
	}
	for _, currency := range billingCurrencyOrder {
		rate := config.ReferenceCurrencies[currency]
		if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
			config.ReferenceCurrencies[currency] = normalized[currency].FXPresentmentPerUSD
		}
	}
	return nil
}

func EnabledBillingCurrencies() []string {
	config := GetBillingCurrencyConfig()
	enabled := make([]string, 0, len(billingCurrencyOrder))
	for _, currency := range billingCurrencyOrder {
		if config.Currencies[currency].Enabled {
			enabled = append(enabled, currency)
		}
	}
	return enabled
}

func IsBillingCurrencyEnabled(currency string) bool {
	currency = strings.ToLower(strings.TrimSpace(currency))
	if !IsSupportedBillingCurrency(currency) {
		return false
	}
	config := GetBillingCurrencyConfig()
	return config.Currencies[currency].Enabled
}

func DefaultBillingCurrency() string {
	return GetBillingCurrencyConfig().DefaultCurrency
}

func BillingCurrencyFXRate(currency string) float64 {
	currency = strings.ToLower(strings.TrimSpace(currency))
	config := GetBillingCurrencyConfig()
	if definition, ok := config.Currencies[currency]; ok && definition.FXPresentmentPerUSD > 0 {
		return definition.FXPresentmentPerUSD
	}
	return 0
}

func SetBillingCurrencyFXRate(currency string, rate float64) error {
	currency = strings.ToLower(strings.TrimSpace(currency))
	config := GetBillingCurrencyConfig()
	definition, ok := config.Currencies[currency]
	if !ok {
		return fmt.Errorf("unsupported currency %q", currency)
	}
	definition.FXPresentmentPerUSD = rate
	config.Currencies[currency] = definition
	data, err := common.Marshal(config)
	if err != nil {
		return err
	}
	return UpdateBillingCurrencyConfigByJSON(string(data))
}

func ResolveBillingCurrency(saved, country, locale string) string {
	saved = strings.ToLower(strings.TrimSpace(saved))
	if IsBillingCurrencyEnabled(saved) {
		return saved
	}

	country = strings.ToUpper(strings.TrimSpace(country))
	locale = strings.ToLower(strings.TrimSpace(locale))
	suggested := ""
	switch {
	case country == "CN" || strings.HasPrefix(locale, "zh"):
		suggested = BillingCurrencyCNY
	case country == "CA" || strings.HasPrefix(locale, "en-ca") || strings.HasPrefix(locale, "fr-ca"):
		suggested = BillingCurrencyCAD
	case country == "US" || strings.HasPrefix(locale, "en-us"):
		suggested = BillingCurrencyUSD
	}
	if IsBillingCurrencyEnabled(suggested) {
		return suggested
	}

	config := GetBillingCurrencyConfig()
	if config.Currencies[config.DefaultCurrency].Enabled {
		return config.DefaultCurrency
	}
	for _, currency := range billingCurrencyOrder {
		if config.Currencies[currency].Enabled {
			return currency
		}
	}
	return BillingCurrencyCNY
}
