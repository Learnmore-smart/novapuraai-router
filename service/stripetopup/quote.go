package stripetopup

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/shopspring/decimal"
)

// QuoteRequest selects an exact server-configured tier. AmountMajor remains
// accepted only as a legacy lookup key and never overrides a supplied TierID.
type QuoteRequest struct {
	Currency        string `json:"currency"`
	TierID          int    `json:"tier_id"`
	AmountMinor     int64  `json:"amount_minor,omitempty"`
	AmountMajor     int    `json:"amount_major,omitempty"`
	PreferredLocale string `json:"preferred_locale,omitempty"`
	CountryHint     string `json:"country_hint,omitempty"`
}

// QuoteResult is always recalculated server-side from the exact tier.
type QuoteResult struct {
	TierID                 int     `json:"tier_id"`
	Currency               string  `json:"currency"`
	AmountMajor            int     `json:"amount_major"`
	AmountMinor            int64   `json:"amount_minor"`
	PaidCreditAmountMinor  int64   `json:"paid_credit_amount_minor"`
	PromoCreditAmountMinor int64   `json:"promo_credit_amount_minor"`
	TotalCreditAmountMinor int64   `json:"total_credit_amount_minor"`
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
	PromoExpiryDays        int     `json:"promo_expiry_days"`
	Recommended            bool    `json:"recommended"`
	PaymentDisplay         string  `json:"payment_display"`
	BonusDisplay           string  `json:"bonus_display"`
	TotalDisplay           string  `json:"total_display"`
	DisplayLabel           string  `json:"display_label"`
}

type TopupOffer struct {
	TierID                 int    `json:"tier_id"`
	Code                   string `json:"code"`
	Name                   string `json:"name"`
	Currency               string `json:"currency"`
	PaymentAmountMinor     int64  `json:"payment_amount_minor"`
	BonusAmountMinor       int64  `json:"bonus_amount_minor"`
	TotalCreditAmountMinor int64  `json:"total_credit_amount_minor"`
	PaymentDisplay         string `json:"payment_display"`
	BonusDisplay           string `json:"bonus_display"`
	TotalDisplay           string `json:"total_display"`
	Available              bool   `json:"available"`
	UnavailableReason      string `json:"unavailable_reason,omitempty"`
	Recommended            bool   `json:"recommended"`
	PromoExpiryDays        int    `json:"promo_expiry_days"`
	StartAt                int64  `json:"start_at"`
	EndAt                  int64  `json:"end_at"`
}

type TopupOfferCatalog struct {
	Currency       string                        `json:"currency"`
	Campaign       *model.TopupPromotionCampaign `json:"campaign,omitempty"`
	CampaignActive bool                          `json:"campaign_active"`
	Repeatable     bool                          `json:"repeatable"`
	Offers         []TopupOffer                  `json:"offers"`
}

func DetectDefaultCurrency(country, locale string) string {
	return setting.ResolveBillingCurrency("", country, locale)
}

func BuildQuote(userID int, req QuoteRequest) (*QuoteResult, error) {
	currency := strings.ToLower(strings.TrimSpace(req.Currency))
	if !setting.IsSupportedTopupCurrency(currency) {
		return nil, fmt.Errorf("unsupported currency")
	}

	now := time.Now().Unix()
	var tier *model.TopupPromoTier
	var err error
	var paymentMinor int64
	if req.TierID > 0 {
		tier, _, err = model.GetExactTopupPromoTier(req.TierID, currency, now)
		if err == nil {
			paymentMinor = tier.PaymentAmountMinor
		}
	} else if req.AmountMinor > 0 {
		paymentMinor = req.AmountMinor
	} else if req.AmountMajor > 0 && int64(req.AmountMajor) <= math.MaxInt64/100 {
		paymentMinor = int64(req.AmountMajor) * 100
	} else {
		err = model.ErrTopupPromotionUnavailable
	}
	if err != nil {
		return nil, err
	}

	minMinor, maxMinor := setting.TopupMinMaxMinor(currency)
	if paymentMinor < minMinor || paymentMinor > maxMinor {
		return nil, fmt.Errorf("top-up payment amount is outside configured limits")
	}

	fx := setting.FXRatePresentmentPerUSD(currency)
	paidMicro := setting.PresentmentMinorToMicroUSD(currency, paymentMinor, fx)
	if paidMicro <= 0 {
		return nil, fmt.Errorf("invalid converted credit amount")
	}
	paidQuota := setting.MicroUSDToQuota(paidMicro)
	if paidQuota <= 0 {
		return nil, fmt.Errorf("invalid top-up quota")
	}

	totalMinor := paymentMinor
	snapshotData := map[string]any{
		"applied":                   false,
		"currency":                  currency,
		"payment_amount_minor":      paymentMinor,
		"bonus_amount_minor":        int64(0),
		"total_credit_amount_minor": totalMinor,
		"fx_presentment_per_usd":    fx,
	}
	snapshot, err := common.Marshal(snapshotData)
	if err != nil {
		return nil, fmt.Errorf("snapshot promotion: %w", err)
	}

	return &QuoteResult{
		TierID:                 0,
		Currency:               currency,
		AmountMajor:            int(paymentMinor / 100),
		AmountMinor:            paymentMinor,
		PaidCreditAmountMinor:  paymentMinor,
		PromoCreditAmountMinor: 0,
		TotalCreditAmountMinor: totalMinor,
		FxRateSnapshot:         fx,
		PaidCreditMicroUSD:     paidMicro,
		PromoCreditMicroUSD:    0,
		TotalCreditMicroUSD:    paidMicro,
		PaidQuota:              paidQuota,
		PromoQuota:             0,
		TotalQuota:             paidQuota,
		PromotionSnapshotJSON:  string(snapshot),
		PromotionTierID:        0,
		PromoExpiryDays:        0,
		Recommended:            false,
		PaymentDisplay:         FormatMinor(currency, paymentMinor),
		BonusDisplay:           FormatMinor(currency, 0),
		TotalDisplay:           FormatMinor(currency, totalMinor),
		DisplayLabel:           FormatMinor(currency, paymentMinor),
	}, nil
}

// ListTopupOffers includes unavailable exact tiers so clients can render the
// configured catalog and explain why a card cannot currently be purchased.
func ListTopupOffers(userID int, currency string) (*TopupOfferCatalog, error) {
	currency = strings.ToLower(strings.TrimSpace(currency))
	if !setting.IsSupportedBillingCurrency(currency) {
		return nil, fmt.Errorf("unsupported currency")
	}
	presets := setting.GetTopupPresets(currency)
	catalog := &TopupOfferCatalog{
		Currency: currency,
		Offers:   make([]TopupOffer, 0, len(presets)),
	}
	for _, amountMajor := range presets {
		paymentMinor := int64(amountMajor) * 100
		quote, quoteErr := BuildQuote(userID, QuoteRequest{Currency: currency, AmountMinor: paymentMinor})
		offer := TopupOffer{
			Code:               fmt.Sprintf("preset-%s-%d", currency, amountMajor),
			Name:               FormatMinor(currency, paymentMinor),
			Currency:           currency,
			PaymentAmountMinor: paymentMinor,
			PaymentDisplay:     FormatMinor(currency, paymentMinor),
			BonusDisplay:       FormatMinor(currency, 0),
			TotalDisplay:       FormatMinor(currency, paymentMinor),
		}

		if quoteErr == nil {
			offer.Available = true
			offer.TierID = quote.TierID
			offer.BonusAmountMinor = quote.PromoCreditAmountMinor
			offer.TotalCreditAmountMinor = quote.TotalCreditAmountMinor
			offer.PaymentDisplay = quote.PaymentDisplay
			offer.BonusDisplay = quote.BonusDisplay
			offer.TotalDisplay = quote.TotalDisplay
			offer.Recommended = quote.Recommended
			offer.PromoExpiryDays = quote.PromoExpiryDays
		} else {
			switch {
			case !setting.IsBillingCurrencyEnabled(currency):
				offer.UnavailableReason = "currency_unavailable"
			default:
				offer.UnavailableReason = "tier_unavailable"
			}
		}
		catalog.Offers = append(catalog.Offers, offer)
	}
	return catalog, nil
}

func FormatMinor(currency string, minor int64) string {
	whole := minor / 100
	fraction := minor % 100
	if currency == setting.BillingCurrencyCNY {
		return fmt.Sprintf("\u00a5%d.%02d", whole, fraction)
	}
	switch currency {
	case setting.BillingCurrencyCAD:
		return fmt.Sprintf("C$%d.%02d", whole, fraction)
	default:
		return fmt.Sprintf("$%d.%02d", whole, fraction)
	}
}

func QuotaToPresentmentMinor(currency string, quota int) int64 {
	if quota <= 0 || common.QuotaPerUnit <= 0 {
		return 0
	}
	minor := decimal.NewFromInt(int64(quota)).
		Div(decimal.NewFromFloat(common.QuotaPerUnit)).
		Mul(decimal.NewFromFloat(setting.FXRatePresentmentPerUSD(currency))).
		Mul(decimal.NewFromInt(100))
	if minor.IsNegative() || minor.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0
	}
	return minor.Round(0).IntPart()
}
