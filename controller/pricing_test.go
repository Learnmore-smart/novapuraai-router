package controller

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestApplyBillingCurrencyPricesUsesCanonicalPricingAndFX(t *testing.T) {
	pricing := []model.Pricing{
		{ModelName: "token-model", ModelRatio: 1.5, CompletionRatio: 3},
		{ModelName: "fixed-model", QuotaType: 1, ModelPrice: 0.25},
	}

	applyBillingCurrencyPrices(pricing, "cad", 1.4)

	assert.Equal(t, "cad", pricing[0].BillingCurrency)
	assert.Equal(t, 1.4, pricing[0].BillingFXRate)
	require.NotNil(t, pricing[0].BillingInputPerMillion)
	require.NotNil(t, pricing[0].BillingOutputPerMillion)
	assert.InDelta(t, 4.2, *pricing[0].BillingInputPerMillion, 0.000001)
	assert.InDelta(t, 12.6, *pricing[0].BillingOutputPerMillion, 0.000001)
	require.NotNil(t, pricing[1].BillingPerRequest)
	assert.InDelta(t, 0.35, *pricing[1].BillingPerRequest, 0.000001)
}

func TestApplyBillingCurrencyPricesSkipsUnsetModels(t *testing.T) {
	pricing := []model.Pricing{
		{ModelName: "unset-model", PriceUnset: true, ModelRatio: 37.5, CompletionRatio: 1},
	}

	applyBillingCurrencyPrices(pricing, "cny", 6.722868)

	assert.Equal(t, "cny", pricing[0].BillingCurrency)
	assert.Nil(t, pricing[0].BillingInputPerMillion)
	assert.Nil(t, pricing[0].BillingOutputPerMillion)
	assert.Nil(t, pricing[0].BillingPerRequest)
}

func TestCanonicalDeepSeekPricingUsesThreeToOneOutputBeforeFX(t *testing.T) {
	ratio_setting.InitRatioSettings()
	ratio, found, _ := ratio_setting.GetModelRatio("deepseek-v4-flash-0731")
	require.True(t, found)
	assert.InDelta(t, 0.11, ratio, 0.000001)
	assert.InDelta(t, 3.0, ratio_setting.GetCompletionRatio("deepseek-v4-flash-0731"), 0.000001)

	pricing := []model.Pricing{{
		ModelName:       "deepseek-v4-flash-0731",
		ModelRatio:      ratio,
		CompletionRatio: ratio_setting.GetCompletionRatio("deepseek-v4-flash-0731"),
	}}
	applyBillingCurrencyPrices(pricing, "usd", 1)

	require.NotNil(t, pricing[0].BillingInputPerMillion)
	require.NotNil(t, pricing[0].BillingOutputPerMillion)
	assert.InDelta(t, 0.22, *pricing[0].BillingInputPerMillion, 0.000001)
	assert.InDelta(t, 0.66, *pricing[0].BillingOutputPerMillion, 0.000001)
}

func TestGetPricingHandlerReturnsCanonicalDeepSeekPricesAndOmitsMalformedIdentity(t *testing.T) {
	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	originalCompletionRatio := ratio_setting.CompletionRatio2JSONString()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:pricing-handler-%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	common.MemoryCacheEnabled = true
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Model{}, &model.Vendor{}))

	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		_ = ratio_setting.UpdateModelRatioByJSONString(originalModelRatio)
		_ = ratio_setting.UpdateCompletionRatioByJSONString(originalCompletionRatio)
		model.InvalidatePricingCache()
	})

	canonical := common.CanonicalDeepSeekV4Flash0731
	malformed := canonical + `"`
	require.NoError(t, db.Create(&model.Channel{
		Id:     9201,
		Type:   constant.ChannelTypeOpenAI,
		Key:    "pricing-handler-key",
		Name:   "pricing-handler-channel",
		Status: common.ChannelStatusEnabled,
		Models: canonical,
		Group:  "default",
	}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: canonical, ChannelId: 9201, Enabled: true}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: malformed, ChannelId: 9201, Enabled: true}).Error)
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"deepseek-v4-flash-0731":0.11}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"deepseek-v4-flash-0731":3}`))
	model.InvalidatePricingCache()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/api/pricing?currency=usd", nil)
	GetPricing(ctx)

	var response struct {
		Success bool            `json:"success"`
		Data    []model.Pricing `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var deepSeek *model.Pricing
	for i := range response.Data {
		assert.NotEqual(t, malformed, response.Data[i].ModelName)
		if response.Data[i].ModelName == canonical {
			deepSeek = &response.Data[i]
		}
	}
	require.NotNil(t, deepSeek)
	require.NotNil(t, deepSeek.BillingInputPerMillion)
	require.NotNil(t, deepSeek.BillingOutputPerMillion)
	assert.InDelta(t, 0.22, *deepSeek.BillingInputPerMillion, 0.000001)
	assert.InDelta(t, 0.66, *deepSeek.BillingOutputPerMillion, 0.000001)
	assert.InDelta(t, 3,
		(*deepSeek.BillingOutputPerMillion)/(*deepSeek.BillingInputPerMillion),
		0.000001,
	)
}

func TestGetPricingHandlerMarksUnsetModelsWithoutDefaultRatio(t *testing.T) {
	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	originalCompletionRatio := ratio_setting.CompletionRatio2JSONString()
	originalDiscount := ratio_setting.ModelDiscount2JSONString()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:pricing-unset-%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	common.MemoryCacheEnabled = true
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Model{}, &model.Vendor{}))

	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		_ = ratio_setting.UpdateModelRatioByJSONString(originalModelRatio)
		_ = ratio_setting.UpdateCompletionRatioByJSONString(originalCompletionRatio)
		_ = ratio_setting.UpdateModelDiscountByJSONString(originalDiscount)
		model.InvalidatePricingCache()
	})

	unsetModel := "adept/fuyu-8b"
	pricedModel := "deepseek-v4-flash-0731"
	require.NoError(t, db.Create(&model.Channel{
		Id:     9202,
		Type:   constant.ChannelTypeOpenAI,
		Key:    "pricing-unset-key",
		Name:   "pricing-unset-channel",
		Status: common.ChannelStatusEnabled,
		Models: pricedModel + "," + unsetModel,
		Group:  "default",
	}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: pricedModel, ChannelId: 9202, Enabled: true}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: unsetModel, ChannelId: 9202, Enabled: true}).Error)
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"deepseek-v4-flash-0731":0.11}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"deepseek-v4-flash-0731":3}`))
	require.NoError(t, ratio_setting.UpdateModelDiscountByJSONString(`{"*":0.25}`))
	model.InvalidatePricingCache()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/api/pricing?currency=cny", nil)
	GetPricing(ctx)

	var response struct {
		Success bool            `json:"success"`
		Data    []model.Pricing `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var unset *model.Pricing
	var priced *model.Pricing
	for i := range response.Data {
		switch response.Data[i].ModelName {
		case unsetModel:
			unset = &response.Data[i]
		case pricedModel:
			priced = &response.Data[i]
		}
	}
	require.NotNil(t, unset)
	require.NotNil(t, priced)
	assert.True(t, unset.PriceUnset)
	assert.Equal(t, 0.0, unset.ModelRatio)
	assert.Nil(t, unset.BillingInputPerMillion)
	assert.Nil(t, unset.Discount)
	assert.False(t, priced.PriceUnset)
	require.NotNil(t, priced.BillingInputPerMillion)
	assert.InDelta(t, 0.11*2*priced.BillingFXRate, *priced.BillingInputPerMillion, 0.000001)
	require.NotNil(t, priced.Discount)
	assert.InDelta(t, 0.25, *priced.Discount, 0.000001)
}
