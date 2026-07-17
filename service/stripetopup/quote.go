package stripetopup

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

// QuoteRequest is the client-facing quote/checkout request (amount is major units).
type QuoteRequest struct {
	Currency          string `json:"currency"`
	AmountMajor       int    `json:"amount_major"`
	PreferredLocale   string `json:"preferred_locale,omitempty"`
	CountryHint       string `json:"country_hint,omitempty"`
}

// QuoteResult is always recalculated server-side.
type QuoteResult struct {
	Currency               string  `json:"currency"`
	AmountMajor            int     `json:"amount_major"`
	AmountMinor            int64   `json:"amount_minor"`
	FxRateSnapshot         float64 `json:"fx_rate_snapshot"`
	PaidCreditMicroUSD     int64   `json:"paid_credit_micro_usd"`
	PromoCreditMicroUSD    int64   `json:"promo_credit_micro_usd"`
	TotalCreditMicroUSD    int64   `json:"total_credit_micro_usd"`
	PaidQuota              int     `json:"paid_quota"`
	PromoQuota             int     `json:"promo_quota"`
	TotalQuota             int     `json:"total_quota"`
	PromotionSnapshotJSON  string  `json:"promotion_snapshot_json"`
	PromotionTierID        int     `json:"promotion_tier_id"`
	PromotionName          string  `json:"promotion_name,omitempty"`
	DisplayLabel           string  `json:"display_label"`
}

// DetectDefaultCurrency uses country/locale hints; never mandatory for client.
func DetectDefaultCurrency(country, locale string) string {
	c := strings.ToUpper(strings.TrimSpace(country))
	l := strings.ToLower(strings.TrimSpace(locale))
	if c == "CN" || strings.HasPrefix(l, "zh") {
		return "cny"
	}
	if c == "CA" {
		return "cad"
	}
	return "usd"
}

// BuildQuote validates amount/currency and computes locked conversion + promo.
func BuildQuote(userId int, req QuoteRequest) (*QuoteResult, error) {
	cur := strings.ToLower(strings.TrimSpace(req.Currency))
	if !setting.IsSupportedTopupCurrency(cur) {
		return nil, fmt.Errorf("unsupported currency")
	}
	minM, maxM := setting.TopupMinMaxMajor(cur)
	if req.AmountMajor < minM {
		return nil, fmt.Errorf("amount below minimum (%d)", minM)
	}
	if req.AmountMajor > maxM {
		return nil, fmt.Errorf("amount above maximum (%d)", maxM)
	}

	fx := setting.FXRatePresentmentPerUSD(cur)
	minor := int64(req.AmountMajor) * 100
	paidMicro := setting.PresentmentMinorToMicroUSD(cur, minor, fx)
	if paidMicro <= 0 {
		return nil, fmt.Errorf("invalid converted amount")
	}
	paidQuota := setting.MicroUSDToQuota(paidMicro)
	if paidQuota <= 0 {
		return nil, fmt.Errorf("invalid paid quota")
	}

	promo, err := model.CalculateTopupPromo(userId, cur, req.AmountMajor, paidMicro, paidQuota)
	if err != nil {
		promo = &model.PromoCalcResult{SnapshotJSON: `{"applied":false}`}
	}
	promoMicro := promo.PromoCreditMicroUSD
	promoQuota := promo.PromoQuota
	if promoQuota <= 0 && promoMicro > 0 {
		promoQuota = setting.MicroUSDToQuota(promoMicro)
	}

	label := formatDisplay(cur, req.AmountMajor)
	return &QuoteResult{
		Currency:              cur,
		AmountMajor:           req.AmountMajor,
		AmountMinor:           minor,
		FxRateSnapshot:        fx,
		PaidCreditMicroUSD:    paidMicro,
		PromoCreditMicroUSD:   promoMicro,
		TotalCreditMicroUSD:   paidMicro + promoMicro,
		PaidQuota:             paidQuota,
		PromoQuota:            promoQuota,
		TotalQuota:            paidQuota + promoQuota,
		PromotionSnapshotJSON: promo.SnapshotJSON,
		PromotionTierID:       promo.TierID,
		PromotionName:         promo.TierName,
		DisplayLabel:          label,
	}, nil
}

func formatDisplay(currency string, major int) string {
	switch currency {
	case "cny":
		return fmt.Sprintf("¥%d", major)
	case "cad":
		return fmt.Sprintf("C$%d", major)
	default:
		return fmt.Sprintf("$%d", major)
	}
}
