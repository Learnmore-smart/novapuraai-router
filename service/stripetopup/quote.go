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
	var campaign *model.TopupPromotionCampaign
	var err error
	var paymentMinor int64
	if req.TierID > 0 {
		tier, campaign, err = model.GetExactTopupPromoTier(req.TierID, currency, now)
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

	if req.TierID == 0 {
		tier, campaign, err = model.FindTopupPromoBand(currency, paymentMinor, now)
		if err != nil {
			tier, campaign, err = model.FindExactTopupPromoTier(currency, paymentMinor, now)
		}
		if err != nil {
			tier = nil
			campaign = nil
		}
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

	promoMinor := int64(0)
	promoMicro := int64(0)
	promoQuota := 0
	expiryDays := 0
	promotionTierID := 0
	promotionName := ""
	recommended := false
	if tier != nil && campaign != nil {
		if tier.PaymentAmountMinor > 0 {
			if tier.BonusAmountMinor <= 0 || tier.PaymentAmountMinor > math.MaxInt64-tier.BonusAmountMinor || tier.TotalCreditAmountMinor != tier.PaymentAmountMinor+tier.BonusAmountMinor {
				return nil, fmt.Errorf("invalid top-up tier totals")
			}
			promoMinor = tier.BonusAmountMinor
		} else {
			if tier.PercentBonusBps <= 0 || paymentMinor > math.MaxInt64/int64(tier.PercentBonusBps) {
				return nil, fmt.Errorf("invalid top-up promotion multiplier")
			}
			promoMinor = paymentMinor * int64(tier.PercentBonusBps) / 10000
		}
		promoMicro = setting.PresentmentMinorToMicroUSD(currency, promoMinor, fx)
		promoQuota = setting.MicroUSDToQuota(promoMicro)
		if promoMicro <= 0 || promoQuota <= 0 || paidMicro > math.MaxInt64-promoMicro || paidQuota > common.MaxQuota-promoQuota {
			return nil, fmt.Errorf("invalid converted promotional credit amount")
		}
		if capacityErr := model.CheckTopupPromotionCapacity(userID, tier, campaign, promoMicro); capacityErr != nil {
			if req.TierID > 0 {
				return nil, capacityErr
			}
			tier = nil
			campaign = nil
			promoMinor = 0
			promoMicro = 0
			promoQuota = 0
		} else {
			expiryDays = tier.PromoExpiryDays
			if expiryDays == 0 {
				expiryDays = campaign.DefaultPromoExpiryDays
			}
			promotionTierID = tier.Id
			promotionName = tier.Name
			recommended = tier.Recommended
		}
	}

	if paymentMinor > math.MaxInt64-promoMinor {
		return nil, fmt.Errorf("invalid top-up total")
	}
	totalMinor := paymentMinor + promoMinor
	snapshotData := map[string]any{
		"applied":                   tier != nil && campaign != nil,
		"currency":                  currency,
		"payment_amount_minor":      paymentMinor,
		"bonus_amount_minor":        promoMinor,
		"total_credit_amount_minor": totalMinor,
		"fx_presentment_per_usd":    fx,
	}
	if tier != nil && campaign != nil {
		snapshotData["campaign_id"] = campaign.Id
		snapshotData["campaign_name"] = campaign.Name
		snapshotData["tier_id"] = tier.Id
		snapshotData["tier_code"] = tier.Code
		snapshotData["tier_name"] = tier.Name
		snapshotData["percent_bonus_bps"] = tier.PercentBonusBps
		snapshotData["promo_expiry_days"] = expiryDays
		snapshotData["campaign_per_user_limit"] = campaign.PerUserLimit
		snapshotData["tier_per_user_limit"] = tier.PerUserLimit
	}
	snapshot, err := common.Marshal(snapshotData)
	if err != nil {
		return nil, fmt.Errorf("snapshot promotion: %w", err)
	}

	return &QuoteResult{
		TierID:                 promotionTierID,
		Currency:               currency,
		AmountMajor:            int(paymentMinor / 100),
		AmountMinor:            paymentMinor,
		PaidCreditAmountMinor:  paymentMinor,
		PromoCreditAmountMinor: promoMinor,
		TotalCreditAmountMinor: totalMinor,
		FxRateSnapshot:         fx,
		PaidCreditMicroUSD:     paidMicro,
		PromoCreditMicroUSD:    promoMicro,
		TotalCreditMicroUSD:    paidMicro + promoMicro,
		PaidQuota:              paidQuota,
		PromoQuota:             promoQuota,
		TotalQuota:             paidQuota + promoQuota,
		PromotionSnapshotJSON:  string(snapshot),
		PromotionTierID:        promotionTierID,
		PromotionName:          promotionName,
		PromoExpiryDays:        expiryDays,
		Recommended:            recommended,
		PaymentDisplay:         FormatMinor(currency, paymentMinor),
		BonusDisplay:           FormatMinor(currency, promoMinor),
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
	campaign, campaignErr := model.GetTopupPromotionCampaign()
	now := time.Now().Unix()
	catalog := &TopupOfferCatalog{
		Currency:       currency,
		Campaign:       campaign,
		CampaignActive: campaignErr == nil && campaign.ActiveAt(now),
		Repeatable:     campaignErr == nil && campaign.PerUserLimit == 0,
		Offers:         make([]TopupOffer, 0, len(presets)),
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
