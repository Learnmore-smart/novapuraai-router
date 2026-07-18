package service

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

const (
	bankOfCanadaDailyFXURL = "https://www.bankofcanada.ca/valet/observations/group/FX_RATES_DAILY/json?recent=10"
	billingFXRefreshPeriod = 6 * time.Hour
	billingFXFetchTimeout  = 15 * time.Second
)

type bankOfCanadaFXQuote struct {
	Value string `json:"v"`
}

type bankOfCanadaFXObservation struct {
	Date   string               `json:"d"`
	USDCAD *bankOfCanadaFXQuote `json:"FXUSDCAD"`
	CNYCAD *bankOfCanadaFXQuote `json:"FXCNYCAD"`
}

type bankOfCanadaFXResponse struct {
	Observations []bankOfCanadaFXObservation `json:"observations"`
}

func fetchBankOfCanadaBillingFXRates(ctx context.Context, client *http.Client, endpoint string) (map[string]float64, int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, 0, fmt.Errorf("Bank of Canada exchange-rate request returned HTTP %d", response.StatusCode)
	}

	var payload bankOfCanadaFXResponse
	if err := common.DecodeJson(io.LimitReader(response.Body, 1<<20), &payload); err != nil {
		return nil, 0, fmt.Errorf("decode Bank of Canada exchange rates: %w", err)
	}

	var newest *bankOfCanadaFXObservation
	var newestDate time.Time
	for index := range payload.Observations {
		observation := &payload.Observations[index]
		if observation.USDCAD == nil || observation.CNYCAD == nil {
			continue
		}
		published, parseErr := time.Parse("2006-01-02", observation.Date)
		if parseErr != nil {
			continue
		}
		if newest == nil || published.After(newestDate) {
			newest = observation
			newestDate = published
		}
	}
	if newest == nil {
		return nil, 0, fmt.Errorf("Bank of Canada exchange-rate response is missing a complete USD/CAD and CNY/CAD observation")
	}
	usdCAD, err := strconv.ParseFloat(newest.USDCAD.Value, 64)
	if err != nil || usdCAD <= 0 || math.IsNaN(usdCAD) || math.IsInf(usdCAD, 0) {
		return nil, 0, fmt.Errorf("invalid Bank of Canada USD/CAD exchange rate")
	}
	cnyCAD, err := strconv.ParseFloat(newest.CNYCAD.Value, 64)
	if err != nil || cnyCAD <= 0 || math.IsNaN(cnyCAD) || math.IsInf(cnyCAD, 0) {
		return nil, 0, fmt.Errorf("invalid Bank of Canada CNY/CAD exchange rate")
	}
	return map[string]float64{
		setting.BillingCurrencyUSD: 1,
		setting.BillingCurrencyCNY: usdCAD / cnyCAD,
		setting.BillingCurrencyCAD: usdCAD,
	}, newestDate.Unix(), nil
}

func refreshBillingFXRates(ctx context.Context) error {
	rates, publishedAt, err := fetchBankOfCanadaBillingFXRates(ctx, GetHttpClient(), bankOfCanadaDailyFXURL)
	if err != nil {
		return err
	}
	raw, err := setting.ApplyBillingCurrencyFXRates(rates, "Bank of Canada", publishedAt)
	if err != nil {
		return err
	}
	return model.UpdateOption("BillingCurrencyConfig", raw)
}

func StartBillingFXRefreshTask() {
	if !common.IsMasterNode {
		return
	}
	go func() {
		refresh := func() {
			ctx, cancel := context.WithTimeout(context.Background(), billingFXFetchTimeout)
			defer cancel()
			if err := refreshBillingFXRates(ctx); err != nil {
				common.SysError("refresh billing exchange rates: " + err.Error())
			}
		}
		refresh()
		ticker := time.NewTicker(billingFXRefreshPeriod)
		defer ticker.Stop()
		for range ticker.C {
			refresh()
		}
	}()
}
