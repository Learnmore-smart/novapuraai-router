package common

import (
	"fmt"
	"math"
)

// CNY ↔ internal quota conversion — single source of truth for NovaPura campaigns / display.
//
// Convention:
//   1 USD of internal unit scale uses QuotaPerUnit (legacy New API).
//   CNY display uses operation_setting.USDExchangeRate at call sites when needed;
//   campaign amounts in design docs are stated in CNY yuan and converted here using
//   a fixed platform rate of QuotaPerCNYYuan (quota units per 1 CNY).
//
// Default: QuotaPerCNYYuan = QuotaPerUnit / USDExchangeRate_default(7.3) is NOT used
// because exchange rate is admin-configurable. Instead we define:
//   CNYYuanToQuota uses QuotaPerUnit * (yuan / USDExchangeRate) when rate is set,
//   or falls back to treating 1 CNY ≈ 1/7.3 USD for campaign math when rate is 0.

// DefaultUSDExchangeRate used only when operation_setting rate is unavailable at common layer.
const DefaultUSDExchangeRate = 7.3

// CNYYuanToQuota converts yuan (float for config convenience) to integer quota.
// Uses QuotaPerUnit and exchangeRate (CNY per 1 USD). Never returns negative.
func CNYYuanToQuota(yuan float64, exchangeRate float64) int {
	if yuan <= 0 {
		return 0
	}
	if exchangeRate <= 0 {
		exchangeRate = DefaultUSDExchangeRate
	}
	// yuan → USD → quota
	usd := yuan / exchangeRate
	q := QuotaFromFloat(usd * QuotaPerUnit)
	if q < 0 {
		return 0
	}
	return q
}

// QuotaToCNYYuan converts quota to yuan for display (may use float for UI only).
func QuotaToCNYYuan(quota int, exchangeRate float64) float64 {
	if quota <= 0 {
		return 0
	}
	if exchangeRate <= 0 {
		exchangeRate = DefaultUSDExchangeRate
	}
	usd := float64(quota) / QuotaPerUnit
	return usd * exchangeRate
}

// CNYYuanToQuotaStrict rejects NaN/Inf and non-finite results.
func CNYYuanToQuotaStrict(yuan float64, exchangeRate float64) (int, error) {
	if math.IsNaN(yuan) || math.IsInf(yuan, 0) {
		return 0, fmt.Errorf("invalid yuan amount")
	}
	if yuan < 0 {
		return 0, fmt.Errorf("yuan amount cannot be negative")
	}
	return CNYYuanToQuota(yuan, exchangeRate), nil
}
