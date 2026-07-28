package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupSubscriptionAdminTestDB initialises an in-memory SQLite DB with the
// tables the plan admin + covered-models endpoints touch. Mirrors
// setupSubscriptionCouponTestDB but adds SubscriptionPlanCoveredModel.
func setupSubscriptionAdminTestDB(t *testing.T) {
	t.Helper()
	initModelListColumnNames(t)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open("file:sub-admin-"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.RedisEnabled = originalRedisEnabled
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.SubscriptionPlan{},
		&model.SubscriptionPlanCoveredModel{},
		&model.UserSubscription{},
		&model.Log{},
	))
	model.DB = db
	model.LOG_DB = db
	// The subscription plan cache is a process-wide singleton — purge it so
	// GetSubscriptionPlanById always hits this DB on first lookup.
	model.PurgeSubscriptionPlanCache()
}

// decodeCoveredModels pulls the "models" field out of the data object returned
// by AdminGetPlanCoveredModels.
func decodeCoveredModels(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()
	_, _, data := decodeCouponEnvelope(t, recorder)
	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var resp struct {
		Models []string `json:"models"`
	}
	require.NoError(t, common.Unmarshal(respBytes, &resp))
	return resp.Models
}

// ---------------------------------------------------------------------------
// AdminSetPlanCoveredModels — PUT replaces the whole list
// ---------------------------------------------------------------------------

// TestAdminSetPlanCoveredModels verifies that PUT replaces the entire
// covered-model list: a first PUT with 3 models persists all 3, and a second
// PUT with 2 different models replaces the first 3 (the dropped model is gone).
func TestAdminSetPlanCoveredModels(t *testing.T) {
	setupSubscriptionAdminTestDB(t)

	plan := &model.SubscriptionPlan{Title: "Set Models Plan", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, model.DB.Create(plan).Error)

	// First PUT: 3 models.
	ctx, recorder := newCouponJSONRequest(t, http.MethodPut, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models",
		AdminSetPlanCoveredModelsRequest{Models: []string{"gpt-4", "claude-3-opus", "gemini-pro"}}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminSetPlanCoveredModels(ctx)

	success, _, _ := decodeCouponEnvelope(t, recorder)
	require.True(t, success)

	// GET: all 3 stored.
	ctx, recorder = newCouponJSONRequest(t, http.MethodGet, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models", nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminGetPlanCoveredModels(ctx)
	models := decodeCoveredModels(t, recorder)
	assert.ElementsMatch(t, []string{"gpt-4", "claude-3-opus", "gemini-pro"}, models)

	// Second PUT: 2 different models — the first 3 must be fully replaced.
	ctx, recorder = newCouponJSONRequest(t, http.MethodPut, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models",
		AdminSetPlanCoveredModelsRequest{Models: []string{"gpt-4", "claude-3-opus"}}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminSetPlanCoveredModels(ctx)
	success, _, _ = decodeCouponEnvelope(t, recorder)
	require.True(t, success)

	ctx, recorder = newCouponJSONRequest(t, http.MethodGet, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models", nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminGetPlanCoveredModels(ctx)
	models = decodeCoveredModels(t, recorder)
	assert.ElementsMatch(t, []string{"gpt-4", "claude-3-opus"}, models, "second PUT must replace the first list, not append")
}

// ---------------------------------------------------------------------------
// AdminAddPlanCoveredModel — POST appends one (idempotent on duplicates)
// ---------------------------------------------------------------------------

// TestAdminAddPlanCoveredModel verifies that POST appends a single model to the
// covered list, and that posting the same model_id again is a no-op (the list
// stays the same length — no duplicate rows).
func TestAdminAddPlanCoveredModel(t *testing.T) {
	setupSubscriptionAdminTestDB(t)

	plan := &model.SubscriptionPlan{Title: "Add Model Plan", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, model.DB.Create(plan).Error)

	// Seed 2 models via PUT.
	ctx, _ := newCouponJSONRequest(t, http.MethodPut, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models",
		AdminSetPlanCoveredModelsRequest{Models: []string{"gpt-4", "claude-3-opus"}}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminSetPlanCoveredModels(ctx)

	// POST a 3rd model.
	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models",
		AdminAddPlanCoveredModelRequest{ModelId: "gemini-pro"}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminAddPlanCoveredModel(ctx)
	success, _, _ := decodeCouponEnvelope(t, recorder)
	require.True(t, success)

	// GET: 3 models.
	ctx, recorder = newCouponJSONRequest(t, http.MethodGet, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models", nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminGetPlanCoveredModels(ctx)
	models := decodeCoveredModels(t, recorder)
	assert.ElementsMatch(t, []string{"gpt-4", "claude-3-opus", "gemini-pro"}, models)

	// POST a duplicate — must stay at 3.
	ctx, recorder = newCouponJSONRequest(t, http.MethodPost, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models",
		AdminAddPlanCoveredModelRequest{ModelId: "gpt-4"}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminAddPlanCoveredModel(ctx)
	success, _, _ = decodeCouponEnvelope(t, recorder)
	require.True(t, success, "duplicate add must be a no-op success, not an error")

	ctx, recorder = newCouponJSONRequest(t, http.MethodGet, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models", nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminGetPlanCoveredModels(ctx)
	models = decodeCoveredModels(t, recorder)
	assert.Len(t, models, 3, "duplicate add must not produce a duplicate row")
	assert.ElementsMatch(t, []string{"gpt-4", "claude-3-opus", "gemini-pro"}, models)
}

// ---------------------------------------------------------------------------
// AdminRemovePlanCoveredModel — DELETE removes one
// ---------------------------------------------------------------------------

// TestAdminRemovePlanCoveredModel verifies that DELETE /:model_id removes the
// named model from the covered list while leaving the others intact.
func TestAdminRemovePlanCoveredModel(t *testing.T) {
	setupSubscriptionAdminTestDB(t)

	plan := &model.SubscriptionPlan{Title: "Remove Model Plan", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1}
	require.NoError(t, model.DB.Create(plan).Error)

	// Seed 3 models.
	ctx, _ := newCouponJSONRequest(t, http.MethodPut, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models",
		AdminSetPlanCoveredModelsRequest{Models: []string{"gpt-4", "claude-3-opus", "gemini-pro"}}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminSetPlanCoveredModels(ctx)

	// DELETE claude-3-opus.
	ctx, recorder := newCouponJSONRequest(t, http.MethodDelete, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models/claude-3-opus", nil, 1)
	ctx.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(plan.Id)},
		{Key: "model_id", Value: "claude-3-opus"},
	}
	AdminRemovePlanCoveredModel(ctx)
	success, _, _ := decodeCouponEnvelope(t, recorder)
	require.True(t, success)

	// GET: 2 models remain (gpt-4, gemini-pro).
	ctx, recorder = newCouponJSONRequest(t, http.MethodGet, "/api/subscription/admin/plans/"+strconv.Itoa(plan.Id)+"/models", nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(plan.Id)}}
	AdminGetPlanCoveredModels(ctx)
	models := decodeCoveredModels(t, recorder)
	assert.ElementsMatch(t, []string{"gpt-4", "gemini-pro"}, models, "only the deleted model should be gone")
}

// ---------------------------------------------------------------------------
// validateSubscriptionPlanFields — pure logic table test
// ---------------------------------------------------------------------------

// TestValidateSubscriptionPlanFields covers every validation branch via
// explicit inputs. Each case mutates a valid base plan and asserts whether
// validation passes or fails with the expected error substring.
func TestValidateSubscriptionPlanFields(t *testing.T) {
	base := func() model.SubscriptionPlan {
		return model.SubscriptionPlan{
			Title:          "Validate Plan",
			PriceAmountCNY: 10.0,
			PriceAmountUSD: 2.0,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*model.SubscriptionPlan)
		wantErr string
	}{
		{
			name:    "auto_renew with no stripe price ids rejected",
			mutate:  func(p *model.SubscriptionPlan) { p.AutoRenew = true },
			wantErr: "auto_renew plans must set at least one of stripe_price_id_cny or stripe_price_id_usd",
		},
		{
			name: "auto_renew with stripe_price_id_cny ok",
			mutate: func(p *model.SubscriptionPlan) {
				p.AutoRenew = true
				p.StripePriceIdCNY = "price_cny_xxx"
			},
			wantErr: "",
		},
		{
			name: "auto_renew with stripe_price_id_usd ok",
			mutate: func(p *model.SubscriptionPlan) {
				p.AutoRenew = true
				p.StripePriceIdUSD = "price_usd_xxx"
			},
			wantErr: "",
		},
		{
			name: "prepaid_months valid csv ok",
			mutate: func(p *model.SubscriptionPlan) {
				p.PrepaidMonths = "1,3,6,12"
				p.StripeProductId = "prod_xxx"
			},
			wantErr: "",
		},
		{
			name: "prepaid_months with zero rejected",
			mutate: func(p *model.SubscriptionPlan) {
				p.PrepaidMonths = "0,3"
				p.StripeProductId = "prod_xxx"
			},
			wantErr: "prepaid_months value must be > 0",
		},
		{
			name: "prepaid_months non-integer rejected",
			mutate: func(p *model.SubscriptionPlan) {
				p.PrepaidMonths = "abc"
				p.StripeProductId = "prod_xxx"
			},
			wantErr: "prepaid_months contains non-integer value",
		},
		{
			name: "prepaid_months without stripe_product_id rejected",
			mutate: func(p *model.SubscriptionPlan) {
				p.PrepaidMonths = "1,3"
			},
			wantErr: "stripe_product_id is required when prepaid_months is set",
		},
		{
			name: "price_amount_cny above 9999 rejected",
			mutate: func(p *model.SubscriptionPlan) {
				p.PriceAmountCNY = 10000
			},
			wantErr: "price_amount_cny must be between 0 and 9999",
		},
		{
			name: "price_amount_cny below 0 rejected",
			mutate: func(p *model.SubscriptionPlan) {
				p.PriceAmountCNY = -1
			},
			wantErr: "price_amount_cny must be between 0 and 9999",
		},
		{
			name: "price_amount_usd above 9999 rejected",
			mutate: func(p *model.SubscriptionPlan) {
				p.PriceAmountUSD = 10000
			},
			wantErr: "price_amount_usd must be between 0 and 9999",
		},
		{
			name: "renewal_price_cny below 0 rejected",
			mutate: func(p *model.SubscriptionPlan) {
				p.RenewalPriceCNY = -0.01
			},
			wantErr: "renewal_price_cny must be >= 0",
		},
		{
			name: "renewal_price_usd below 0 rejected",
			mutate: func(p *model.SubscriptionPlan) {
				p.RenewalPriceUSD = -1
			},
			wantErr: "renewal_price_usd must be >= 0",
		},
		{
			name:    "valid base plan ok",
			mutate:  func(*model.SubscriptionPlan) {},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := base()
			tt.mutate(&plan)
			err := validateSubscriptionPlanFields(&plan)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AdminCreateSubscriptionPlan — accepts and persists the NovaPura Phase 1 fields
// ---------------------------------------------------------------------------

// TestAdminCreatePlan_AcceptsNewFields verifies that the create endpoint
// accepts a plan carrying every NovaPura Phase 1 field (dual-currency prices,
// Stripe price/product IDs, auto-renew flag, prepaid-months CSV, renewal
// prices) and persists them. The response carries the created plan, so we
// decode and assert each field round-trips through the DB.
func TestAdminCreatePlan_AcceptsNewFields(t *testing.T) {
	setupSubscriptionAdminTestDB(t)

	plan := model.SubscriptionPlan{
		Title:            "Nova Unlimited",
		PriceAmountCNY:   99.00,
		PriceAmountUSD:   14.99,
		StripePriceIdCNY: "price_cny_nova",
		StripePriceIdUSD: "price_usd_nova",
		StripeProductId:  "prod_nova",
		AutoRenew:        true,
		PrepaidMonths:    "1,3,6,12",
		RenewalPriceCNY:  129.00,
		RenewalPriceUSD:  19.99,
		DurationUnit:     model.SubscriptionDurationMonth,
		DurationValue:    1,
	}

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/admin/plans",
		AdminUpsertSubscriptionPlanRequest{Plan: plan}, 1)
	AdminCreateSubscriptionPlan(ctx)

	success, _, data := decodeCouponEnvelope(t, recorder)
	require.True(t, success, "valid plan with all new fields must be created")

	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var created model.SubscriptionPlan
	require.NoError(t, common.Unmarshal(respBytes, &created))
	require.NotZero(t, created.Id, "created plan must have a non-zero ID")

	assert.Equal(t, "Nova Unlimited", created.Title)
	assert.InDelta(t, 99.00, created.PriceAmountCNY, 0.0001)
	assert.InDelta(t, 14.99, created.PriceAmountUSD, 0.0001)
	assert.Equal(t, "price_cny_nova", created.StripePriceIdCNY)
	assert.Equal(t, "price_usd_nova", created.StripePriceIdUSD)
	assert.Equal(t, "prod_nova", created.StripeProductId)
	assert.True(t, created.AutoRenew)
	assert.Equal(t, "1,3,6,12", created.PrepaidMonths)
	assert.InDelta(t, 129.00, created.RenewalPriceCNY, 0.0001)
	assert.InDelta(t, 19.99, created.RenewalPriceUSD, 0.0001)

	// Confirm persistence by re-reading from the DB (bypassing the cache so
	// we see the actual row, not a cached copy).
	model.InvalidateSubscriptionPlanCache(created.Id)
	var refreshed model.SubscriptionPlan
	require.NoError(t, model.DB.First(&refreshed, created.Id).Error)
	assert.Equal(t, created.StripePriceIdCNY, refreshed.StripePriceIdCNY)
	assert.Equal(t, created.StripePriceIdUSD, refreshed.StripePriceIdUSD)
	assert.Equal(t, created.StripeProductId, refreshed.StripeProductId)
	assert.True(t, refreshed.AutoRenew)
	assert.Equal(t, "1,3,6,12", refreshed.PrepaidMonths)
	assert.InDelta(t, 129.00, refreshed.RenewalPriceCNY, 0.0001)
	assert.InDelta(t, 19.99, refreshed.RenewalPriceUSD, 0.0001)
}

// ---------------------------------------------------------------------------
// AdminCreateSubscriptionPlan — rejects invalid NovaPura field combinations
// ---------------------------------------------------------------------------

// TestAdminCreatePlan_RejectsAutoRenewWithoutStripePrice verifies the
// validation helper is actually wired into the create handler: an auto_renew
// plan with no Stripe price IDs must be rejected with success=false.
func TestAdminCreatePlan_RejectsAutoRenewWithoutStripePrice(t *testing.T) {
	setupSubscriptionAdminTestDB(t)

	plan := model.SubscriptionPlan{
		Title:         "Bad AutoRenew",
		AutoRenew:     true,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/admin/plans",
		AdminUpsertSubscriptionPlanRequest{Plan: plan}, 1)
	AdminCreateSubscriptionPlan(ctx)

	success, msg, _ := decodeCouponEnvelope(t, recorder)
	require.False(t, success)
	assert.Contains(t, msg, "auto_renew")

	// No plan row must have been created.
	var count int64
	require.NoError(t, model.DB.Model(&model.SubscriptionPlan{}).Where("title = ?", plan.Title).Count(&count).Error)
	assert.EqualValues(t, 0, count)
}
