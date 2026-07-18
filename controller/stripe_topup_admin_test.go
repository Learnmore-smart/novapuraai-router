package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTopupAdminControllerTest(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	originalConfig := setting.BillingCurrencyConfigJSON()
	common.OptionMapRWMutex.Lock()
	optionMapWasNil := common.OptionMap == nil
	if optionMapWasNil {
		common.OptionMap = make(map[string]string)
	}
	originalCurrencyOption, hadCurrencyOption := common.OptionMap["BillingCurrencyConfig"]
	common.OptionMapRWMutex.Unlock()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:topup-admin-%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB = originalDB
		require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(originalConfig))
		common.OptionMapRWMutex.Lock()
		if optionMapWasNil {
			common.OptionMap = nil
		} else {
			if hadCurrencyOption {
				common.OptionMap["BillingCurrencyConfig"] = originalCurrencyOption
			} else {
				delete(common.OptionMap, "BillingCurrencyConfig")
			}
		}
		common.OptionMapRWMutex.Unlock()
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(
		&model.Option{},
		&model.StripeTopupOrder{},
		&model.TopupPromotionCampaign{},
		&model.TopupPromoTier{},
		&model.TopupPromoRedemption{},
	))
	model.DB = db
	require.NoError(t, model.SeedLaunchTopupPromotion(db))
}

func TestAdminBillingCurrencyChangeDoesNotRewriteHistoricalOrder(t *testing.T) {
	setupTopupAdminControllerTest(t)
	order := &model.StripeTopupOrder{
		OrderID:                "historical-cny-order",
		UserId:                 11,
		Status:                 model.StripeOrderCredited,
		PresentmentCurrency:    "cny",
		PresentmentAmountMinor: 1000,
		PaidCreditAmountMinor:  1000,
		PromoCreditAmountMinor: 2000,
		TotalCreditAmountMinor: 3000,
		PaidCreditMicroUSD:     1_000_000,
		PromoCreditMicroUSD:    2_000_000,
		TotalCreditMicroUSD:    3_000_000,
		PromotionSnapshotJSON:  `{"campaign":"launch"}`,
	}
	require.NoError(t, model.DB.Create(order).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/billing/admin/currencies", bytes.NewBufferString(`{
		"default_currency":"usd",
		"currencies":{
			"cny":{"enabled":true,"fx_presentment_per_usd":7.3},
			"usd":{"enabled":true,"fx_presentment_per_usd":1},
			"cad":{"enabled":false,"fx_presentment_per_usd":1.37}
		}
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	AdminBillingCurrencyConfig(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Equal(t, "usd", setting.DefaultBillingCurrency())
	var unchanged model.StripeTopupOrder
	require.NoError(t, model.DB.Where("order_id = ?", order.OrderID).First(&unchanged).Error)
	assert.Equal(t, "cny", unchanged.PresentmentCurrency)
	assert.Equal(t, int64(1000), unchanged.PresentmentAmountMinor)
	assert.Equal(t, int64(2000), unchanged.PromoCreditAmountMinor)
	assert.Equal(t, order.PromotionSnapshotJSON, unchanged.PromotionSnapshotJSON)
}

func TestAdminBillingTopupCampaignPreservesIssuedCounters(t *testing.T) {
	setupTopupAdminControllerTest(t)
	require.NoError(t, model.DB.Model(&model.TopupPromotionCampaign{}).Where("id = ?", 1).Updates(map[string]any{
		"reserved_promo_micro_usd": 100,
		"issued_promo_micro_usd":   200,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/billing/admin/top-up/campaign", bytes.NewBufferString(`{
		"name":"Launch updated",
		"enabled":true,
		"start_at":0,
		"end_at":0,
		"global_budget_micro_usd":1000000,
		"reserved_promo_micro_usd":999999,
		"issued_promo_micro_usd":999999,
		"per_user_limit":0,
		"default_promo_expiry_days":30
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	AdminBillingTopupCampaign(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	campaign, err := model.GetTopupPromotionCampaign()
	require.NoError(t, err)
	assert.Equal(t, "Launch updated", campaign.Name)
	assert.Equal(t, int64(100), campaign.ReservedPromoMicroUSD)
	assert.Equal(t, int64(200), campaign.IssuedPromoMicroUSD)
}

func TestAdminBillingTopupPreviewReturnsLaunchCards(t *testing.T) {
	setupTopupAdminControllerTest(t)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/billing/admin/top-up/preview?currency=cny", nil)
	AdminBillingTopupPreview(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Offers []struct {
				TierID    int    `json:"tier_id"`
				Available bool   `json:"available"`
				Payment   string `json:"payment_display"`
				Bonus     string `json:"bonus_display"`
				Total     string `json:"total_display"`
			} `json:"offers"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data.Offers, 6)
	assert.True(t, response.Data.Offers[0].Available)
	assert.Equal(t, "¥10.00", response.Data.Offers[0].Payment)
	assert.Equal(t, "¥20.00", response.Data.Offers[0].Bonus)
	assert.Equal(t, "¥30.00", response.Data.Offers[0].Total)
}
