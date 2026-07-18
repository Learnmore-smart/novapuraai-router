package controller

import "github.com/QuantumNous/new-api/model"

func applyBillingCurrencyPrices(pricing []model.Pricing, currency string, fxRate float64) {
	for i := range pricing {
		item := &pricing[i]
		item.BillingCurrency = currency
		item.BillingFXRate = fxRate
		if item.QuotaType == 1 {
			value := item.ModelPrice * fxRate
			item.BillingPerRequest = &value
			continue
		}
		input := item.ModelRatio * 2 * fxRate
		output := input * item.CompletionRatio
		item.BillingInputPerMillion = &input
		item.BillingOutputPerMillion = &output
	}
}
