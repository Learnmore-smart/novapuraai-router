package controller

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/stripesubscription"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v85"
	"gorm.io/gorm"
)

type controllerStripeSubscriptionGateway struct {
	portalParams *stripe.BillingPortalSessionParams
}

func clearStripeSubscriptionControllerEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"STRIPE_SUBSCRIPTION_ENABLED",
		"STRIPE_SUBSCRIPTION_ACCOUNT_ID",
		"STRIPE_SUBSCRIPTION_PRODUCT_ID",
		"STRIPE_SUBSCRIPTION_FOUNDER_PRICE_ID",
		"STRIPE_SUBSCRIPTION_STANDARD_PRICE_ID",
		"STRIPE_SUBSCRIPTION_PORTAL_CONFIGURATION_ID",
	} {
		original, wasSet := os.LookupEnv(key)
		require.NoError(t, os.Unsetenv(key))
		key := key
		t.Cleanup(func() {
			if wasSet {
				_ = os.Setenv(key, original)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}

func (f *controllerStripeSubscriptionGateway) CreateCheckoutSession(context.Context, *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	return &stripe.CheckoutSession{ID: "cs_controller", URL: "https://checkout.example/controller"}, nil
}

func (f *controllerStripeSubscriptionGateway) ExpireCheckoutSession(context.Context, string) error {
	return nil
}

func (f *controllerStripeSubscriptionGateway) CreatePortalSession(_ context.Context, params *stripe.BillingPortalSessionParams) (*stripe.BillingPortalSession, error) {
	f.portalParams = params
	return &stripe.BillingPortalSession{URL: "https://billing.example/controller"}, nil
}

func setupStripeSubscriptionControllerDB(t *testing.T) (*gorm.DB, *model.SubscriptionPlan, *model.User) {
	t.Helper()
	clearStripeSubscriptionControllerEnv(t)
	originalDB := model.DB
	originalDatabaseType := common.MainDatabaseType()
	originalAPISecret := setting.StripeApiSecret
	originalWebhookSecret := setting.StripeWebhookSecret
	originalAccountID := setting.StripeAccountID
	originalRuntime := setting.StripeRuntimeEnvironment
	originalRequireTest := setting.StripeRequireTestKeys
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:stripe-subscription-controller-%s-%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	model.DB = db
	setting.StripeApiSecret = "sk_test_subscription_controller"
	setting.StripeWebhookSecret = "whsec_subscription_controller"
	setting.StripeAccountID = model.SandboxStripeSubscriptionAccountID
	setting.StripeRuntimeEnvironment = setting.StripeRuntimeTest
	setting.StripeRequireTestKeys = true
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "true")
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.StripeSubscriptionReservation{},
		&model.StripeSubscriptionFounderClaim{},
		&model.StripeSubscription{},
		&model.StripeSubscriptionInvoice{},
		&model.StripeWebhookEvent{},
	))
	recurringCode := model.SandboxStripeSubscriptionPlanCode
	plan := &model.SubscriptionPlan{
		Code:                        model.SandboxStripeSubscriptionPlanCode,
		RecurringCode:               &recurringCode,
		Title:                       "NovaPuraAI DeepSeek V4 Flash Unlimited",
		PriceAmount:                 19.99,
		Currency:                    "CNY",
		DurationUnit:                model.SubscriptionDurationMonth,
		DurationValue:               1,
		Enabled:                     true,
		StripeSubscriptionEnabled:   true,
		StripeSubscriptionModel:     model.SandboxStripeSubscriptionModel,
		UpgradeGroup:                model.SandboxStripeSubscriptionGroup,
		MaxActiveSubscriptions:      20,
		FounderPurchaseLimit:        20,
		MaxActivePerUser:            1,
		FounderStripePriceId:        model.SandboxStripeSubscriptionFounderPriceID,
		StandardStripePriceId:       model.SandboxStripeSubscriptionStandardPriceID,
		StripeProductId:             model.SandboxStripeSubscriptionProductID,
		StripeAccountId:             model.SandboxStripeSubscriptionAccountID,
		StripePortalConfigurationId: model.SandboxStripeSubscriptionPortalConfigurationID,
		FounderAmountMinor:          1999,
		StandardAmountMinor:         9999,
		StripeCurrency:              model.SandboxStripeSubscriptionCurrency,
		TotalAmount:                 0,
		AllowBalancePay:             common.GetPointer(false),
		AllowWalletOverflow:         common.GetPointer(false),
	}
	require.NoError(t, db.Create(plan).Error)
	model.InvalidateSubscriptionPlanCache(plan.Id)
	user := &model.User{Username: "stripe-subscription-controller-user", Email: "controller@example.com", StripeCustomer: "cus_controller"}
	require.NoError(t, db.Create(user).Error)
	t.Cleanup(func() {
		model.DB = originalDB
		common.SetMainDatabaseType(originalDatabaseType)
		setting.StripeApiSecret = originalAPISecret
		setting.StripeWebhookSecret = originalWebhookSecret
		setting.StripeAccountID = originalAccountID
		setting.StripeRuntimeEnvironment = originalRuntime
		setting.StripeRequireTestKeys = originalRequireTest
		_ = sqlDB.Close()
	})
	return db, plan, user
}

func requestSubscriptionStripePay(t *testing.T, userID int, planID int) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", userID)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/stripe/pay", bytes.NewBufferString(fmt.Sprintf(`{"plan_id":%d}`, planID)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	SubscriptionRequestStripePay(ctx)
	return recorder
}

func decodeControllerJSON(t *testing.T, recorder *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestRecurringStripePayMapsCapacityAndDuplicateToStableHTTP409Codes(t *testing.T) {
	db, plan, user := setupStripeSubscriptionControllerDB(t)
	now := time.Now().Unix()
	for id := 1; id <= 20; id++ {
		other := &model.User{Username: fmt.Sprintf("controller-seat-%d", id), AffCode: fmt.Sprintf("controller-seat-aff-%d", id)}
		require.NoError(t, db.Create(other).Error)
		_, err := model.ReserveStripeSubscriptionSeat(plan.Id, other.Id, fmt.Sprintf("controller-capacity-%d", id), now+int64(id))
		require.NoError(t, err)
	}
	capacityRecorder := requestSubscriptionStripePay(t, user.Id, plan.Id)
	assert.Equal(t, http.StatusConflict, capacityRecorder.Code)
	assert.Equal(t, "subscription_capacity_full", decodeControllerJSON(t, capacityRecorder)["code"])

	db2, plan2, user2 := setupStripeSubscriptionControllerDB(t)
	_, err := model.ReserveStripeSubscriptionSeat(plan2.Id, user2.Id, "controller-pending", time.Now().Unix())
	require.NoError(t, err)
	duplicateRecorder := requestSubscriptionStripePay(t, user2.Id, plan2.Id)
	assert.Equal(t, http.StatusConflict, duplicateRecorder.Code)
	assert.Equal(t, "subscription_already_pending", decodeControllerJSON(t, duplicateRecorder)["code"])
	assert.NotNil(t, db2)
}

func TestRecurringFounderSoldOutUsesPublicCapacityCode(t *testing.T) {
	setupStripeSubscriptionControllerDB(t)
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	stripeSubscriptionHTTPError(ctx, model.ErrStripeSubscriptionFounderSoldOut)
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.Equal(t, "subscription_capacity_full", decodeControllerJSON(t, recorder)["code"])
}

func TestPrepareAdminSubscriptionPlanUsesActiveProductionCatalog(t *testing.T) {
	_, sandboxPlan, _ := setupStripeSubscriptionControllerDB(t)
	production := model.DefaultProductionStripeSubscriptionConfig()
	plan := *sandboxPlan
	plan.FounderStripePriceId = production.FounderPriceID
	plan.StandardStripePriceId = production.StandardPriceID
	plan.StripeProductId = production.ProductID
	plan.StripeAccountId = production.AccountID
	plan.StripePortalConfigurationId = production.PortalConfigurationID
	setting.StripeRuntimeEnvironment = setting.StripeRuntimeProduction
	setting.StripeRequireTestKeys = false
	setting.StripeApiSecret = "sk_live_subscription_controller"
	setting.StripeAccountID = production.AccountID

	recurring, err := prepareAdminSubscriptionPlan(&plan, sandboxPlan)
	require.NoError(t, err)
	assert.True(t, recurring)
	assert.Equal(t, production.PlanCode, plan.Code)
	require.NotNil(t, plan.RecurringCode)
	assert.Equal(t, production.PlanCode, *plan.RecurringCode)

	plan.FounderStripePriceId = model.SandboxStripeSubscriptionFounderPriceID
	_, err = prepareAdminSubscriptionPlan(&plan, sandboxPlan)
	assert.ErrorIs(t, err, model.ErrStripeSubscriptionPlanInvalid)
}

func TestRootAdminCanCreateUpdateAndEnableSandboxRecurringPlan(t *testing.T) {
	db, seedPlan, _ := setupStripeSubscriptionControllerDB(t)
	// The fixture seeds the canonical row for most controller tests; remove it
	// here so this test exercises the root-admin create path without violating
	// the structural recurring-code uniqueness invariant.
	require.NoError(t, db.Delete(seedPlan).Error)
	plan := model.SubscriptionPlan{
		Code:        model.SandboxStripeSubscriptionPlanCode,
		Title:       "Sandbox recurring admin plan",
		PriceAmount: 19.99,
		// Omit currency to verify the recurring admin path applies the fixed
		// CNY catalog before exact validation instead of inheriting USD.
		Currency:                    "",
		DurationUnit:                model.SubscriptionDurationMonth,
		DurationValue:               1,
		Enabled:                     false,
		AllowBalancePay:             common.GetPointer(false),
		AllowWalletOverflow:         common.GetPointer(false),
		StripeSubscriptionEnabled:   false,
		StripeSubscriptionModel:     model.SandboxStripeSubscriptionModel,
		MaxActiveSubscriptions:      20,
		FounderPurchaseLimit:        20,
		MaxActivePerUser:            1,
		FounderStripePriceId:        model.SandboxStripeSubscriptionFounderPriceID,
		StandardStripePriceId:       model.SandboxStripeSubscriptionStandardPriceID,
		FounderAmountMinor:          1999,
		StandardAmountMinor:         9999,
		StripeCurrency:              model.SandboxStripeSubscriptionCurrency,
		StripeProductId:             model.SandboxStripeSubscriptionProductID,
		StripeAccountId:             model.SandboxStripeSubscriptionAccountID,
		StripePortalConfigurationId: model.SandboxStripeSubscriptionPortalConfigurationID,
		UpgradeGroup:                model.SandboxStripeSubscriptionGroup,
		TotalAmount:                 0,
	}
	createBody, err := common.Marshal(map[string]any{"plan": plan})
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	createRecorder := httptest.NewRecorder()
	createContext, _ := gin.CreateTestContext(createRecorder)
	createContext.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/admin/plans", bytes.NewReader(createBody))
	createContext.Request.Header.Set("Content-Type", "application/json")
	AdminCreateSubscriptionPlan(createContext)
	assert.Equal(t, http.StatusOK, createRecorder.Code)

	var created model.SubscriptionPlan
	require.NoError(t, db.Where("title = ?", plan.Title).First(&created).Error)
	assert.Equal(t, model.SandboxStripeSubscriptionCurrency, created.Currency)
	require.NotNil(t, created.RecurringCode)
	assert.Equal(t, model.SandboxStripeSubscriptionPlanCode, *created.RecurringCode)
	assert.False(t, created.Enabled)
	assert.False(t, created.StripeSubscriptionEnabled)

	created.Subtitle = "updated recurring subtitle"
	created.Currency = ""
	updateBody, err := common.Marshal(map[string]any{"plan": created})
	require.NoError(t, err)
	updateRecorder := httptest.NewRecorder()
	updateContext, _ := gin.CreateTestContext(updateRecorder)
	updateContext.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", created.Id)}}
	updateContext.Request = httptest.NewRequest(http.MethodPut, "/api/subscription/admin/plans/"+fmt.Sprintf("%d", created.Id), bytes.NewReader(updateBody))
	updateContext.Request.Header.Set("Content-Type", "application/json")
	AdminUpdateSubscriptionPlan(updateContext)
	assert.Equal(t, http.StatusOK, updateRecorder.Code)

	enableRecorder := httptest.NewRecorder()
	enableContext, _ := gin.CreateTestContext(enableRecorder)
	enableContext.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", created.Id)}}
	enableContext.Request = httptest.NewRequest(http.MethodPatch, "/api/subscription/admin/plans/"+fmt.Sprintf("%d", created.Id), bytes.NewBufferString(`{"enabled":true}`))
	enableContext.Request.Header.Set("Content-Type", "application/json")
	AdminUpdateSubscriptionPlanStatus(enableContext)
	assert.Equal(t, http.StatusOK, enableRecorder.Code)

	require.NoError(t, db.First(&created, created.Id).Error)
	assert.True(t, created.Enabled)
	assert.True(t, created.StripeSubscriptionEnabled)
	assert.Equal(t, "updated recurring subtitle", created.Subtitle)
}

func TestRootAdminCannotEnableRecurringPlanWhenEnvGateIsFalse(t *testing.T) {
	db, plan, _ := setupStripeSubscriptionControllerDB(t)
	require.NoError(t, db.Model(&model.SubscriptionPlan{}).Where("id = ?", plan.Id).Updates(map[string]any{
		"enabled":                     false,
		"stripe_subscription_enabled": false,
	}).Error)
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "false")

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", plan.Id)}}
	ctx.Request = httptest.NewRequest(http.MethodPatch, "/api/subscription/admin/plans/"+fmt.Sprintf("%d", plan.Id), bytes.NewBufferString(`{"enabled":true}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	AdminUpdateSubscriptionPlanStatus(ctx)
	assert.Equal(t, http.StatusOK, recorder.Code)
	payload := decodeControllerJSON(t, recorder)
	assert.Equal(t, false, payload["success"])
	assert.Equal(t, model.ErrStripeSubscriptionDisabled.Error(), payload["message"])

	var persisted model.SubscriptionPlan
	require.NoError(t, db.First(&persisted, plan.Id).Error)
	assert.False(t, persisted.Enabled)
	assert.False(t, persisted.StripeSubscriptionEnabled)

	persisted.Enabled = true
	persisted.StripeSubscriptionEnabled = true
	updateBody, err := common.Marshal(map[string]any{"plan": persisted})
	require.NoError(t, err)
	updateRecorder := httptest.NewRecorder()
	updateContext, _ := gin.CreateTestContext(updateRecorder)
	updateContext.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", plan.Id)}}
	updateContext.Request = httptest.NewRequest(http.MethodPut, "/api/subscription/admin/plans/"+fmt.Sprintf("%d", plan.Id), bytes.NewReader(updateBody))
	updateContext.Request.Header.Set("Content-Type", "application/json")
	AdminUpdateSubscriptionPlan(updateContext)
	assert.Equal(t, http.StatusOK, updateRecorder.Code)
	updatePayload := decodeControllerJSON(t, updateRecorder)
	assert.Equal(t, false, updatePayload["success"])
	assert.Equal(t, model.ErrStripeSubscriptionDisabled.Error(), updatePayload["message"])

	require.NoError(t, db.First(&persisted, plan.Id).Error)
	assert.False(t, persisted.Enabled)
	assert.False(t, persisted.StripeSubscriptionEnabled)
}

func TestRootAdminRecurringCreateRollsBackWhenAtomicFlagSyncFails(t *testing.T) {
	db, seedPlan, _ := setupStripeSubscriptionControllerDB(t)
	require.NoError(t, db.Delete(seedPlan).Error)
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "true")
	require.NoError(t, db.Exec("CREATE TRIGGER admin_recurring_sync_failure BEFORE UPDATE ON subscription_plans BEGIN SELECT RAISE(ABORT, 'admin sync failure'); END").Error)

	plan := *seedPlan
	plan.Id = 0
	plan.Title = "atomic recurring create"
	body, err := common.Marshal(map[string]any{"plan": plan})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/admin/plans", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	AdminCreateSubscriptionPlan(ctx)

	payload := decodeControllerJSON(t, recorder)
	assert.Equal(t, false, payload["success"])
	var count int64
	require.NoError(t, db.Model(&model.SubscriptionPlan{}).Where("title = ?", plan.Title).Count(&count).Error)
	assert.Zero(t, count)
}

func TestStripeSubscriptionOfferReturnsFairUseAndUserDuplicateState(t *testing.T) {
	_, plan, user := setupStripeSubscriptionControllerDB(t)
	reservation, err := model.ReserveStripeSubscriptionSeat(plan.Id, user.Id, "controller-offer-pending", time.Now().Unix())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = model.ReleaseStripeSubscriptionReservation(reservation.Id, time.Now().Unix())
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/subscription/stripe/offer", nil)
	GetStripeSubscriptionOffer(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	payload := decodeControllerJSON(t, recorder)
	data, ok := payload["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, false, data["checkout_allowed"])
	assert.Equal(t, true, data["already_pending"])
	fairUse, ok := data["fair_use"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(10), fairUse["peak_concurrency"])
	assert.Equal(t, float64(600), fairUse["rolling_window_seconds"])
	assert.Equal(t, float64(1800), fairUse["concurrent_seconds_budget"])
	assert.Equal(t, float64(600), fairUse["successful_requests"])
	assert.Equal(t, float64(750), fairUse["admitted_requests"])
	assert.Equal(t, float64(15), fairUse["heartbeat_interval_seconds"])
	assert.Equal(t, float64(45), fairUse["stale_lease_recovery_seconds"])
	assert.Equal(t, stripesubscription.NoResaleCopyIdentifier, fairUse["no_resale_copy_identifier"])
}

func TestStripeSubscriptionPortalIgnoresClientOwnershipAndReturnURL(t *testing.T) {
	db, plan, user := setupStripeSubscriptionControllerDB(t)
	require.NoError(t, db.Create(&model.StripeSubscription{
		PlanId:               plan.Id,
		UserId:               user.Id,
		StripeCustomerId:     "cus_controller",
		StripeSubscriptionId: "sub_controller",
		StripePriceId:        plan.FounderStripePriceId,
		Tier:                 model.StripeSubscriptionTierFounder,
		Status:               model.StripeSubscriptionStatusActive,
	}).Error)
	fake := &controllerStripeSubscriptionGateway{}
	restore := stripesubscription.SetGatewayForTest(fake)
	t.Cleanup(restore)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/stripe/portal", bytes.NewBufferString(`{"customer":"cus_attacker","return_url":"https://attacker.example"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	PostStripeSubscriptionPortal(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, fake.portalParams)
	assert.Equal(t, "cus_controller", *fake.portalParams.Customer)
	assert.Equal(t, model.SandboxStripeSubscriptionPortalConfigurationID, *fake.portalParams.Configuration)
	assert.NotEqual(t, "https://attacker.example", *fake.portalParams.ReturnURL)
	assert.Contains(t, recorder.Body.String(), "https://billing.example/controller")
}

func TestSubscriptionPlanListHidesRecurringOfferWhenRuntimeGateIsFalse(t *testing.T) {
	db, plan, _ := setupStripeSubscriptionControllerDB(t)
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "false")
	legacy := &model.SubscriptionPlan{
		Code:          "legacy-visible-plan",
		Title:         "Legacy visible plan",
		PriceAmount:   5,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
	}
	require.NoError(t, db.Create(legacy).Error)
	require.NoError(t, db.Model(&model.SubscriptionPlan{}).Where("id = ?", plan.Id).Update("enabled", true).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/subscription/plans", nil)
	GetSubscriptionPlans(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	payload := decodeControllerJSON(t, recorder)
	data, ok := payload["data"].([]interface{})
	require.True(t, ok)
	require.Len(t, data, 1)
	entry, ok := data[0].(map[string]interface{})
	require.True(t, ok)
	planData, ok := entry["plan"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, legacy.Code, planData["code"])
}

func TestLegacyEpayPreflightRejectsRecurringBeforePaymentValidation(t *testing.T) {
	_, plan, user := setupStripeSubscriptionControllerDB(t)
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "false")

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/epay/pay", bytes.NewBufferString(fmt.Sprintf(`{"plan_id":%d,"payment_method":"not-configured"}`, plan.Id)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	SubscriptionRequestEpay(ctx)

	payload := decodeControllerJSON(t, recorder)
	assert.Equal(t, false, payload["success"])
	assert.Equal(t, model.ErrStripeSubscriptionDisabled.Error(), payload["message"])
}

func TestLegacyStripePreflightRejectsMixedRecurringStateBeforeSecretValidation(t *testing.T) {
	_, plan, user := setupStripeSubscriptionControllerDB(t)
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "false")
	plan.StripeSubscriptionEnabled = false
	plan.StripePriceId = "legacy-price-placeholder"
	require.NoError(t, model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", plan.Id).Updates(map[string]any{
		"stripe_subscription_enabled": false,
		"stripe_price_id":             plan.StripePriceId,
	}).Error)
	originalSecret := setting.StripeApiSecret
	setting.StripeApiSecret = "not-configured"
	t.Cleanup(func() { setting.StripeApiSecret = originalSecret })

	recorder := requestSubscriptionStripePay(t, user.Id, plan.Id)
	payload := decodeControllerJSON(t, recorder)
	assert.Equal(t, false, payload["success"])
	assert.Equal(t, model.ErrStripeSubscriptionDisabled.Error(), payload["message"])
}

func TestAdminListMarksNonFixedRecurringRowsUnavailable(t *testing.T) {
	db, plan, _ := setupStripeSubscriptionControllerDB(t)
	otherCode := "admin-other-recurring"
	other := *plan
	other.Id = 0
	other.Code = otherCode
	other.RecurringCode = &otherCode
	other.SortOrder = -1
	require.NoError(t, db.Create(&other).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/subscription/admin/plans", nil)
	AdminListSubscriptionPlans(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	payload := decodeControllerJSON(t, recorder)
	data, ok := payload["data"].([]interface{})
	require.True(t, ok)
	availability := make(map[string]bool)
	for _, raw := range data {
		entry, ok := raw.(map[string]interface{})
		require.True(t, ok)
		planData, ok := entry["plan"].(map[string]interface{})
		require.True(t, ok)
		if value, ok := entry["recurring_available"].(bool); ok {
			availability[planData["code"].(string)] = value
		}
	}
	assert.True(t, availability[plan.Code])
	assert.False(t, availability[other.Code])
}

func TestEpayPreflightRejectsStaleCachedRecurringPlanBeforePayment(t *testing.T) {
	db, plan, user := setupStripeSubscriptionControllerDB(t)
	_, err := model.GetSubscriptionPlanById(plan.Id)
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.SubscriptionPlan{}).Where("id = ?", plan.Id).Updates(map[string]any{
		"enabled":                     false,
		"stripe_subscription_enabled": false,
	}).Error)
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "true")

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/epay/pay", bytes.NewBufferString(fmt.Sprintf(`{"plan_id":%d,"payment_method":"not-configured"}`, plan.Id)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	SubscriptionRequestEpay(ctx)

	payload := decodeControllerJSON(t, recorder)
	assert.Equal(t, false, payload["success"])
	assert.Contains(t, fmt.Sprint(payload["message"]), model.ErrStripeSubscriptionPlanInvalid.Error())
}
