package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBillingCurrencyControllerTest(t *testing.T) *model.User {
	t.Helper()
	gin.SetMode(gin.TestMode)
	originalDB := model.DB
	originalRedisEnabled := common.RedisEnabled
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB = originalDB
		common.RedisEnabled = originalRedisEnabled
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(
		&model.Option{},
		&model.User{},
		&model.BalanceCreditLot{},
		&model.BalanceLedger{},
		&model.TopupPromotionCampaign{},
		&model.TopupPromoTier{},
		&model.TopupPromoRedemption{},
	))
	model.DB = db
	require.NoError(t, model.SeedLaunchTopupPromotion(db))
	user := &model.User{Username: "currency-user", Email: "currency@example.com", Quota: 500000}
	user.SetSetting(structToUserSetting(t, `{"billing_currency":"cad"}`))
	require.NoError(t, db.Create(user).Error)
	return user
}

func structToUserSetting(t *testing.T, raw string) (settingValue dto.UserSetting) {
	t.Helper()
	require.NoError(t, common.UnmarshalJsonStr(raw, &settingValue))
	return settingValue
}

func TestGetBillingTopupConfigPrefersSavedEnabledCurrency(t *testing.T) {
	user := setupBillingCurrencyControllerTest(t)
	original := setting.BillingCurrencyConfigJSON()
	t.Cleanup(func() { require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(original)) })

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/billing/top-up/config", nil)
	ctx.Request.Header.Set("CF-IPCountry", "CN")
	ctx.Request.Header.Set("Accept-Language", "zh-CN")
	ctx.Set("id", user.Id)

	GetBillingTopupConfig(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Data struct {
			SelectedCurrency string `json:"selected_currency"`
			DefaultCurrency  string `json:"default_currency"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "cad", response.Data.SelectedCurrency)
	assert.Equal(t, "cny", response.Data.DefaultCurrency)
}

func TestPutBillingCurrencyRejectsDisabledCurrencyAndPersistsEnabledCurrency(t *testing.T) {
	user := setupBillingCurrencyControllerTest(t)
	original := setting.BillingCurrencyConfigJSON()
	t.Cleanup(func() { require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(original)) })
	require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(`{"default_currency":"cny","currencies":{"cny":{"enabled":true,"fx_presentment_per_usd":7.3},"usd":{"enabled":false,"fx_presentment_per_usd":1},"cad":{"enabled":true,"fx_presentment_per_usd":1.37}}}`))

	invoke := func(currency string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/api/billing/currency", bytes.NewBufferString(`{"currency":"`+currency+`"}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Set("id", user.Id)
		PutBillingCurrency(ctx)
		return recorder
	}

	assert.Equal(t, http.StatusBadRequest, invoke("usd").Code)
	assert.Equal(t, http.StatusOK, invoke("cny").Code)

	var refreshed model.User
	require.NoError(t, model.DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, "cny", refreshed.GetSetting().BillingCurrency)
}

func TestAdminBillingCurrencyConfigPreservesServerOwnedReferenceRates(t *testing.T) {
	setupBillingCurrencyControllerTest(t)
	original := setting.BillingCurrencyConfigJSON()
	t.Cleanup(func() { require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(original)) })
	require.NoError(t, setting.UpdateBillingCurrencyConfigByJSON(`{"default_currency":"cny","auto_update_fx":false,"fx_source":"Bank of Canada","fx_updated_at":1752796800,"reference_fx_presentment_per_usd":{"cny":7.21,"usd":1,"cad":1.36},"currencies":{"cny":{"enabled":true,"fx_presentment_per_usd":7.3},"usd":{"enabled":true,"fx_presentment_per_usd":1},"cad":{"enabled":true,"fx_presentment_per_usd":1.4}}}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/admin/billing/currencies", bytes.NewBufferString(`{"default_currency":"cny","auto_update_fx":false,"reference_fx_presentment_per_usd":{"cny":999,"usd":1,"cad":999},"currencies":{"cny":{"enabled":true,"fx_presentment_per_usd":7.45},"usd":{"enabled":true,"fx_presentment_per_usd":1},"cad":{"enabled":true,"fx_presentment_per_usd":1.42}}}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	AdminBillingCurrencyConfig(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	config := setting.GetBillingCurrencyConfig()
	assert.Equal(t, 7.45, config.Currencies["cny"].FXPresentmentPerUSD)
	assert.Equal(t, 1.42, config.Currencies["cad"].FXPresentmentPerUSD)
	assert.Equal(t, map[string]float64{"cny": 7.21, "usd": 1, "cad": 1.36}, config.ReferenceCurrencies)
	assert.Equal(t, "Bank of Canada", config.FXSource)
	assert.Equal(t, int64(1752796800), config.FXUpdatedAt)
}
