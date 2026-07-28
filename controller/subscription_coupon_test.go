package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

// setupSubscriptionCouponTestDB initialises an in-memory SQLite DB with the
// tables the coupon admin CRUD + validate endpoints touch. Mirrors
// setupSubscriptionStripeTestDB (initModelListColumnNames is required so
// commonGroupCol etc. are populated for the audit log writer).
func setupSubscriptionCouponTestDB(t *testing.T) {
	t.Helper()
	initModelListColumnNames(t)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	originalStripeSecret := setting.StripeApiSecret
	common.RedisEnabled = false
	setting.StripeApiSecret = ""
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:sub-coupon-%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.RedisEnabled = originalRedisEnabled
		setting.StripeApiSecret = originalStripeSecret
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.SubscriptionPlan{},
		&model.SubscriptionOrder{},
		&model.UserSubscription{},
		&model.SubscriptionCoupon{},
		&model.SubscriptionCouponRedemption{},
		&model.Log{},
	))
	model.DB = db
	model.LOG_DB = db
	// The subscription plan cache is a process-wide singleton, so a plan
	// cached under a previous test's DB can mask a different plan row with
	// the same ID in this DB. Purge it so GetSubscriptionPlanById always
	// hits this DB on first lookup.
	model.PurgeSubscriptionPlanCache()
}

// newCouponJSONRequest builds a gin.Context whose request body is the JSON
// encoding of payload, ready to be passed to a coupon admin handler. The
// context's id is set to adminId so the audit log writer can attribute the
// operation.
func newCouponJSONRequest(t *testing.T, method, path string, payload any, adminId int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	body, err := common.Marshal(payload)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if adminId > 0 {
		ctx.Set("id", adminId)
		ctx.Set("username", "coupon-admin")
		ctx.Set("role", common.RoleAdminUser)
	}
	return ctx, recorder
}

// decodeCouponEnvelope decodes a standard {success,message,data} response.
// Returns the success flag, message, and the raw data field for further
// decoding by the caller.
func decodeCouponEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) (bool, string, any) {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code, "handler must respond with HTTP 200 (project convention); got body=%s", recorder.Body.String())
	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    any    `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &envelope))
	return envelope.Success, envelope.Message, envelope.Data
}

// ---------------------------------------------------------------------------
// validateCreateCouponRequest — pure logic table test
// ---------------------------------------------------------------------------

// TestValidateCreateCouponRequest covers every validation branch via explicit
// inputs. This is the source-of-truth validator (Gin's binding tags are only
// a first line of defense), so each rule must be enforced here.
func TestValidateCreateCouponRequest(t *testing.T) {
	base := func() CreateSubscriptionCouponRequest {
		return CreateSubscriptionCouponRequest{
			Code:           "SAVE20",
			Name:           "20% Off",
			PercentOff:     20,
			DurationMonths: 3,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*CreateSubscriptionCouponRequest)
		wantErr string
	}{
		{"happy path valid", func(*CreateSubscriptionCouponRequest) {}, ""},
		{"empty code rejected", func(r *CreateSubscriptionCouponRequest) { r.Code = "   " }, "code is required"},
		{"empty name rejected", func(r *CreateSubscriptionCouponRequest) { r.Name = "" }, "name is required"},
		{"percent_off below 1 rejected", func(r *CreateSubscriptionCouponRequest) { r.PercentOff = 0 }, "percent_off must be between 1 and 100"},
		{"percent_off above 100 rejected", func(r *CreateSubscriptionCouponRequest) { r.PercentOff = 101 }, "percent_off must be between 1 and 100"},
		{"duration_months below 1 rejected", func(r *CreateSubscriptionCouponRequest) { r.DurationMonths = 0 }, "duration_months must be >= 1"},
		{"end_at <= start_at rejected", func(r *CreateSubscriptionCouponRequest) { r.StartAt = 1000; r.EndAt = 1000 }, "end_at must be greater than start_at"},
		{"end_at before start_at rejected", func(r *CreateSubscriptionCouponRequest) { r.StartAt = 2000; r.EndAt = 1000 }, "end_at must be greater than start_at"},
		{"negative max_redemptions rejected", func(r *CreateSubscriptionCouponRequest) { r.MaxRedemptions = -1 }, "max_redemptions must be >= 0"},
		{"negative per_user_limit rejected", func(r *CreateSubscriptionCouponRequest) { r.PerUserLimit = -1 }, "per_user_limit must be >= 0"},
		{"zero max_redemptions allowed (unlimited)", func(r *CreateSubscriptionCouponRequest) { r.MaxRedemptions = 0 }, ""},
		{"start_at without end_at allowed", func(r *CreateSubscriptionCouponRequest) { r.StartAt = 1000; r.EndAt = 0 }, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base()
			tt.mutate(&req)
			err := validateCreateCouponRequest(&req)
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
// ValidateSubscriptionCouponForUser — public read-only validate endpoint
// ---------------------------------------------------------------------------

// TestValidateSubscriptionCouponForUser_ValidAutoRenew verifies the happy path
// for auto_renew mode: a valid coupon returns a breakdown whose math matches
// the plan price × 1 month × percent_off, with next_renewal_price equal to
// the discounted final amount (the discount applies for DurationMonths).
func TestValidateSubscriptionCouponForUser_ValidAutoRenew(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	user := &model.User{Username: "validate-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{
		Title:         "Pro",
		Enabled:       true,
		PriceAmountUSD: 19.99,
		PriceAmountCNY: 149.00,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	coupon := &model.SubscriptionCoupon{
		Code: "SAVE20", Name: "20% Off", StripeCouponId: "stripe_save20",
		PercentOff: 20, DurationMonths: 3, Enabled: true,
	}
	require.NoError(t, model.DB.Create(coupon).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/coupons/validate",
		ValidateCouponRequest{Code: "SAVE20", PlanId: plan.Id, Mode: "auto_renew", Currency: "USD"}, user.Id)
	ValidateSubscriptionCouponForUser(ctx)

	success, msg, data := decodeCouponEnvelope(t, recorder)
	require.True(t, success, "valid coupon must return success=true; msg=%s", msg)
	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var resp ValidateCouponResponse
	require.NoError(t, common.Unmarshal(respBytes, &resp))

	assert.True(t, resp.Valid)
	assert.Equal(t, "20% Off", resp.CouponName)
	assert.Equal(t, 20, resp.PercentOff)
	assert.Equal(t, 3, resp.DurationMonths)
	assert.Equal(t, "USD", resp.Currency)
	// original = 19.99 × 1 month; discount = 19.99 × 20 / 100 = 3.998 -> 4.00; final = 15.99
	assert.InDelta(t, 19.99, resp.OriginalPrice, 0.01)
	assert.InDelta(t, 4.00, resp.DiscountAmount, 0.01)
	assert.InDelta(t, 15.99, resp.FinalAmount, 0.01)
	// auto_renew: discount applies for DurationMonths, so next renewal is the discounted price.
	assert.InDelta(t, resp.FinalAmount, resp.NextRenewalPrice, 0.01)
}

// TestValidateSubscriptionCouponForUser_ValidPrepaid verifies the prepaid
// happy path: the discount is computed against original × prepaid_months, and
// next_renewal_price reports the original (one-time discount, no renewal).
func TestValidateSubscriptionCouponForUser_ValidPrepaid(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	user := &model.User{Username: "prepaid-validate-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{
		Title:         "Prepaid",
		Enabled:       true,
		PriceAmountCNY: 100.00,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	coupon := &model.SubscriptionCoupon{
		Code: "CNY50", Name: "CNY 50% Off", StripeCouponId: "stripe_cny50",
		PercentOff: 50, DurationMonths: 1, Enabled: true,
	}
	require.NoError(t, model.DB.Create(coupon).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/coupons/validate",
		ValidateCouponRequest{Code: "CNY50", PlanId: plan.Id, Mode: "prepaid", Currency: "CNY", PrepaidMonths: 6}, user.Id)
	ValidateSubscriptionCouponForUser(ctx)

	success, _, data := decodeCouponEnvelope(t, recorder)
	require.True(t, success)
	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var resp ValidateCouponResponse
	require.NoError(t, common.Unmarshal(respBytes, &resp))

	assert.True(t, resp.Valid)
	// original = 100 × 6 = 600; discount = 600 × 50% = 300; final = 300
	assert.InDelta(t, 600.00, resp.OriginalPrice, 0.01)
	assert.InDelta(t, 300.00, resp.DiscountAmount, 0.01)
	assert.InDelta(t, 300.00, resp.FinalAmount, 0.01)
	// prepaid: one-time discount, no renewal — next_renewal_price reports original.
	assert.InDelta(t, 600.00, resp.NextRenewalPrice, 0.01)
}

// TestValidateSubscriptionCouponForUser_InvalidCouponReturnsValidFalse
// verifies the read-only contract: an invalid coupon is a validation RESULT,
// not a server error. The endpoint returns HTTP 200 with success=true and
// valid=false (so the frontend can render the reason inline).
func TestValidateSubscriptionCouponForUser_InvalidCouponReturnsValidFalse(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	user := &model.User{Username: "invalid-coupon-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{
		Title:         "Pro",
		Enabled:       true,
		PriceAmountUSD: 19.99,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/coupons/validate",
		ValidateCouponRequest{Code: "DOESNOTEXIST", PlanId: plan.Id, Mode: "auto_renew", Currency: "USD"}, user.Id)
	ValidateSubscriptionCouponForUser(ctx)

	success, _, data := decodeCouponEnvelope(t, recorder)
	require.True(t, success, "invalid coupon is a validation result, not a server error")
	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var resp ValidateCouponResponse
	require.NoError(t, common.Unmarshal(respBytes, &resp))
	assert.False(t, resp.Valid)
	assert.NotEmpty(t, resp.Reason, "reason must be populated for an invalid coupon")
}

// TestValidateSubscriptionCouponForUser_BadModeRejected verifies that an
// unrecognized mode is rejected with HTTP 400 (not the soft valid=false path)
// because it indicates a malformed client request, not a coupon validation
// failure.
func TestValidateSubscriptionCouponForUser_BadModeRejected(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	user := &model.User{Username: "bad-mode-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{Title: "Pro", Enabled: true, PriceAmountUSD: 19.99}
	require.NoError(t, model.DB.Create(plan).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/coupons/validate",
		ValidateCouponRequest{Code: "X", PlanId: plan.Id, Mode: "invalid", Currency: "USD"}, user.Id)
	ValidateSubscriptionCouponForUser(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestValidateSubscriptionCouponForUser_PrepaidMonthsRequired verifies that
// prepaid mode requires prepaid_months > 0 (a prepaid checkout for 0 months
// is nonsensical and must be rejected with HTTP 400).
func TestValidateSubscriptionCouponForUser_PrepaidMonthsRequired(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	user := &model.User{Username: "prepaid-months-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{Title: "Pro", Enabled: true, PriceAmountUSD: 19.99}
	require.NoError(t, model.DB.Create(plan).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/coupons/validate",
		ValidateCouponRequest{Code: "X", PlanId: plan.Id, Mode: "prepaid", Currency: "USD", PrepaidMonths: 0}, user.Id)
	ValidateSubscriptionCouponForUser(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestValidateSubscriptionCouponForUser_BadCurrencyRejected verifies that an
// unrecognized currency is rejected with HTTP 400.
func TestValidateSubscriptionCouponForUser_BadCurrencyRejected(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	user := &model.User{Username: "bad-currency-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{Title: "Pro", Enabled: true, PriceAmountUSD: 19.99}
	require.NoError(t, model.DB.Create(plan).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/coupons/validate",
		ValidateCouponRequest{Code: "X", PlanId: plan.Id, Mode: "auto_renew", Currency: "EUR"}, user.Id)
	ValidateSubscriptionCouponForUser(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestValidateSubscriptionCouponForUser_DisabledPlanRejected verifies that a
// disabled plan short-circuits with a generic error (the validate math must
// never run against a plan the user cannot actually purchase).
func TestValidateSubscriptionCouponForUser_DisabledPlanRejected(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	user := &model.User{Username: "disabled-plan-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{Title: "Disabled", PriceAmountUSD: 19.99}
	require.NoError(t, model.DB.Create(plan).Error)
	// SubscriptionPlan.Enabled has gorm:"default:true", so GORM stores the
	// zero value false as true on insert. Force the disabled state with an
	// explicit column update so the controller's !plan.Enabled branch fires.
	require.NoError(t, model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", plan.Id).Update("enabled", false).Error)
	model.InvalidateSubscriptionPlanCache(plan.Id)

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/coupons/validate",
		ValidateCouponRequest{Code: "X", PlanId: plan.Id, Mode: "auto_renew", Currency: "USD"}, user.Id)
	ValidateSubscriptionCouponForUser(ctx)

	success, msg, _ := decodeCouponEnvelope(t, recorder)
	require.False(t, success)
	assert.Contains(t, msg, "套餐")
}

// ---------------------------------------------------------------------------
// AdminListSubscriptionCoupons
// ---------------------------------------------------------------------------

// TestAdminListSubscriptionCoupons_Pagination verifies the list endpoint
// returns coupons newest-first and respects page / page_size. The total field
// must reflect the full row count, not just the returned page.
func TestAdminListSubscriptionCoupons_Pagination(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	// Insert 3 coupons. IDs are assigned in insertion order (1, 2, 3); the
	// list endpoint orders by id desc, so the first page returns 3, 2.
	for i, code := range []string{"C1", "C2", "C3"} {
		require.NoError(t, model.DB.Create(&model.SubscriptionCoupon{
			Code: code, Name: code, StripeCouponId: "stripe_" + code,
			PercentOff: 10 + i, DurationMonths: 1, Enabled: true,
		}).Error)
	}

	ctx, recorder := newCouponJSONRequest(t, http.MethodGet, "/api/subscription/admin/coupons?page=1&page_size=2", nil, 1)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/subscription/admin/coupons?page=1&page_size=2", nil)
	ctx.Set("id", 1)
	AdminListSubscriptionCoupons(ctx)

	success, _, data := decodeCouponEnvelope(t, recorder)
	require.True(t, success)
	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var listResp struct {
		Items []model.SubscriptionCoupon `json:"items"`
		Total int64                      `json:"total"`
	}
	require.NoError(t, common.Unmarshal(respBytes, &listResp))
	assert.EqualValues(t, 3, listResp.Total)
	require.Len(t, listResp.Items, 2)
	// Newest first: C3 (id=3) then C2 (id=2).
	assert.Equal(t, "C3", listResp.Items[0].Code)
	assert.Equal(t, "C2", listResp.Items[1].Code)
}

// ---------------------------------------------------------------------------
// AdminGetSubscriptionCoupon
// ---------------------------------------------------------------------------

// TestAdminGetSubscriptionCoupon_IncludesRedemptionStats verifies the get
// endpoint returns the coupon plus redemption counts (total + active). Active
// = reserved + issued (released / reversed do not count as active).
func TestAdminGetSubscriptionCoupon_IncludesRedemptionStats(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	coupon := &model.SubscriptionCoupon{
		Code: "STATS", Name: "Stats", StripeCouponId: "stripe_stats",
		PercentOff: 10, DurationMonths: 1, Enabled: true,
	}
	require.NoError(t, model.DB.Create(coupon).Error)

	now := common.GetTimestamp()
	redemptions := []model.SubscriptionCouponRedemption{
		{OrderId: "o1", CouponId: coupon.Id, UserId: 1, PlanId: 1, Status: model.CouponRedemptionStatusReserved, PercentOff: 10, Currency: "USD", CreatedAt: now, UpdatedAt: now},
		{OrderId: "o2", CouponId: coupon.Id, UserId: 2, PlanId: 1, Status: model.CouponRedemptionStatusIssued, PercentOff: 10, Currency: "USD", CreatedAt: now, UpdatedAt: now},
		{OrderId: "o3", CouponId: coupon.Id, UserId: 3, PlanId: 1, Status: model.CouponRedemptionStatusReleased, PercentOff: 10, Currency: "USD", CreatedAt: now, UpdatedAt: now},
		{OrderId: "o4", CouponId: coupon.Id, UserId: 4, PlanId: 1, Status: model.CouponRedemptionStatusReversed, PercentOff: 10, Currency: "USD", CreatedAt: now, UpdatedAt: now},
	}
	for i := range redemptions {
		require.NoError(t, model.DB.Create(&redemptions[i]).Error)
	}

	ctx, recorder := newCouponJSONRequest(t, http.MethodGet, "/api/subscription/admin/coupons/"+strconv.Itoa(int(coupon.Id)), nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(coupon.Id))}}
	ctx.Set("id", 1)
	AdminGetSubscriptionCoupon(ctx)

	success, _, data := decodeCouponEnvelope(t, recorder)
	require.True(t, success)
	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var getResp struct {
		Coupon             model.SubscriptionCoupon `json:"coupon"`
		TotalRedemptions   int64                     `json:"total_redemptions"`
		ActiveRedemptions  int64                     `json:"active_redemptions"`
	}
	require.NoError(t, common.Unmarshal(respBytes, &getResp))
	assert.Equal(t, coupon.Code, getResp.Coupon.Code)
	assert.EqualValues(t, 4, getResp.TotalRedemptions)
	assert.EqualValues(t, 2, getResp.ActiveRedemptions, "active = reserved + issued only")
}

// TestAdminGetSubscriptionCoupon_NotFound verifies a missing coupon returns
// success=false with a not-found message (HTTP 200 per project convention).
func TestAdminGetSubscriptionCoupon_NotFound(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	ctx, recorder := newCouponJSONRequest(t, http.MethodGet, "/api/subscription/admin/coupons/9999", nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: "9999"}}
	ctx.Set("id", 1)
	AdminGetSubscriptionCoupon(ctx)

	success, msg, _ := decodeCouponEnvelope(t, recorder)
	require.False(t, success)
	assert.Contains(t, msg, "not found")
}

// ---------------------------------------------------------------------------
// AdminUpdateSubscriptionCoupon
// ---------------------------------------------------------------------------

// TestAdminUpdateSubscriptionCoupon_UpdatesMutableFields verifies that the
// mutable fields (name, enabled, max_redemptions, per_user_limit,
// new_user_only, start_at, end_at) are updated and the immutable fields
// (code, percent_off, duration_months, stripe_coupon_id) are left untouched.
func TestAdminUpdateSubscriptionCoupon_UpdatesMutableFields(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	coupon := &model.SubscriptionCoupon{
		Code: "UPD", Name: "Original", StripeCouponId: "stripe_upd",
		PercentOff: 10, DurationMonths: 1, Enabled: true,
		MaxRedemptions: 100, PerUserLimit: 1, NewUserOnly: false,
	}
	require.NoError(t, model.DB.Create(coupon).Error)

	enabled := false
	maxRed := 50
	perUser := 5
	newUser := true
	ctx, recorder := newCouponJSONRequest(t, http.MethodPut, "/api/subscription/admin/coupons/"+strconv.Itoa(int(coupon.Id)),
		UpdateSubscriptionCouponRequest{
			Name:           strPtr("Updated"),
			Enabled:        &enabled,
			MaxRedemptions: &maxRed,
			PerUserLimit:   &perUser,
			NewUserOnly:    &newUser,
		}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(coupon.Id))}}
	ctx.Set("id", 1)
	AdminUpdateSubscriptionCoupon(ctx)

	success, _, _ := decodeCouponEnvelope(t, recorder)
	require.True(t, success)

	var refreshed model.SubscriptionCoupon
	require.NoError(t, model.DB.First(&refreshed, coupon.Id).Error)
	assert.Equal(t, "Updated", refreshed.Name)
	assert.False(t, refreshed.Enabled)
	assert.Equal(t, 50, refreshed.MaxRedemptions)
	assert.Equal(t, 5, refreshed.PerUserLimit)
	assert.True(t, refreshed.NewUserOnly)
	// Immutable fields must be untouched.
	assert.Equal(t, "UPD", refreshed.Code)
	assert.Equal(t, 10, refreshed.PercentOff)
	assert.Equal(t, 1, refreshed.DurationMonths)
	assert.Equal(t, "stripe_upd", refreshed.StripeCouponId)
}

// TestAdminUpdateSubscriptionCoupon_EndAtMustExceedStartAt verifies the
// cross-field validation: when only end_at is being updated, it is compared
// against the existing row's start_at; an end_at <= start_at is rejected.
func TestAdminUpdateSubscriptionCoupon_EndAtMustExceedStartAt(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	coupon := &model.SubscriptionCoupon{
		Code: "UPD2", Name: "Dates", StripeCouponId: "stripe_upd2",
		PercentOff: 10, DurationMonths: 1, Enabled: true,
		StartAt: 10000,
	}
	require.NoError(t, model.DB.Create(coupon).Error)

	endAt := int64(5000) // before StartAt
	ctx, recorder := newCouponJSONRequest(t, http.MethodPut, "/api/subscription/admin/coupons/"+strconv.Itoa(int(coupon.Id)),
		UpdateSubscriptionCouponRequest{EndAt: &endAt}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(coupon.Id))}}
	ctx.Set("id", 1)
	AdminUpdateSubscriptionCoupon(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)

	// The coupon row must be unchanged.
	var refreshed model.SubscriptionCoupon
	require.NoError(t, model.DB.First(&refreshed, coupon.Id).Error)
	assert.Equal(t, int64(10000), refreshed.StartAt)
	assert.Equal(t, int64(0), refreshed.EndAt, "rejected update must not persist end_at")
}

// TestAdminUpdateSubscriptionCoupon_NotFound verifies a missing coupon is
// rejected with success=false.
func TestAdminUpdateSubscriptionCoupon_NotFound(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	ctx, recorder := newCouponJSONRequest(t, http.MethodPut, "/api/subscription/admin/coupons/9999",
		UpdateSubscriptionCouponRequest{}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: "9999"}}
	ctx.Set("id", 1)
	AdminUpdateSubscriptionCoupon(ctx)

	success, msg, _ := decodeCouponEnvelope(t, recorder)
	require.False(t, success)
	assert.Contains(t, msg, "not found")
}

// ---------------------------------------------------------------------------
// AdminDeleteSubscriptionCoupon
// ---------------------------------------------------------------------------

// TestAdminDeleteSubscriptionCoupon_SoftDeletesWhenActiveRedemptionsExist
// verifies that a coupon with active (reserved/issued) redemptions is disabled
// instead of hard-deleted, so the audit history is preserved. The response
// reports hard_deleted=false.
func TestAdminDeleteSubscriptionCoupon_SoftDeletesWhenActiveRedemptionsExist(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	coupon := &model.SubscriptionCoupon{
		Code: "SOFT", Name: "Soft", StripeCouponId: "stripe_soft",
		PercentOff: 10, DurationMonths: 1, Enabled: true,
	}
	require.NoError(t, model.DB.Create(coupon).Error)

	now := common.GetTimestamp()
	require.NoError(t, model.DB.Create(&model.SubscriptionCouponRedemption{
		OrderId: "o-soft", CouponId: coupon.Id, UserId: 1, PlanId: 1,
		Status: model.CouponRedemptionStatusIssued, PercentOff: 10, Currency: "USD",
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodDelete, "/api/subscription/admin/coupons/"+strconv.Itoa(int(coupon.Id)), nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(coupon.Id))}}
	ctx.Set("id", 1)
	AdminDeleteSubscriptionCoupon(ctx)

	success, _, data := decodeCouponEnvelope(t, recorder)
	require.True(t, success)
	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var delResp struct {
		HardDeleted bool   `json:"hard_deleted"`
		Message     string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(respBytes, &delResp))
	assert.False(t, delResp.HardDeleted)

	// The row must still exist, now disabled.
	var refreshed model.SubscriptionCoupon
	require.NoError(t, model.DB.First(&refreshed, coupon.Id).Error)
	assert.False(t, refreshed.Enabled, "soft-delete must disable the coupon")
}

// TestAdminDeleteSubscriptionCoupon_HardDeletesWhenNoActiveRedemptions
// verifies that a coupon with no active redemptions is hard-deleted locally.
// The Stripe API delete is best-effort and skipped when StripeApiSecret is
// not configured (the test DB leaves it empty), so the local delete must
// still succeed and report hard_deleted=true with an empty stripe_error.
func TestAdminDeleteSubscriptionCoupon_HardDeletesWhenNoActiveRedemptions(t *testing.T) {
	setupSubscriptionCouponTestDB(t)
	// StripeApiSecret is empty (set by setupSubscriptionCouponTestDB), so the
	// Stripe API delete branch is skipped — the local delete proceeds.

	coupon := &model.SubscriptionCoupon{
		Code: "HARD", Name: "Hard", StripeCouponId: "stripe_hard",
		PercentOff: 10, DurationMonths: 1, Enabled: true,
	}
	require.NoError(t, model.DB.Create(coupon).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodDelete, "/api/subscription/admin/coupons/"+strconv.Itoa(int(coupon.Id)), nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(coupon.Id))}}
	ctx.Set("id", 1)
	AdminDeleteSubscriptionCoupon(ctx)

	success, _, data := decodeCouponEnvelope(t, recorder)
	require.True(t, success)
	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var delResp struct {
		HardDeleted bool   `json:"hard_deleted"`
		StripeError string `json:"stripe_error"`
	}
	require.NoError(t, common.Unmarshal(respBytes, &delResp))
	assert.True(t, delResp.HardDeleted)
	assert.Empty(t, delResp.StripeError, "no Stripe secret => no Stripe API call => empty stripe_error")

	// The row must no longer exist.
	var count int64
	require.NoError(t, model.DB.Model(&model.SubscriptionCoupon{}).Where("id = ?", coupon.Id).Count(&count).Error)
	assert.EqualValues(t, 0, count)
}

// TestAdminDeleteSubscriptionCoupon_HardDeletesWithReleasedRedemptionsOnly
// verifies that released/reversed redemptions (which are NOT active) do not
// block a hard delete. Only reserved/issued redemptions force the soft-delete
// path.
func TestAdminDeleteSubscriptionCoupon_HardDeletesWithReleasedRedemptionsOnly(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	coupon := &model.SubscriptionCoupon{
		Code: "HARD2", Name: "Hard2", StripeCouponId: "stripe_hard2",
		PercentOff: 10, DurationMonths: 1, Enabled: true,
	}
	require.NoError(t, model.DB.Create(coupon).Error)

	now := common.GetTimestamp()
	require.NoError(t, model.DB.Create(&model.SubscriptionCouponRedemption{
		OrderId: "o-released", CouponId: coupon.Id, UserId: 1, PlanId: 1,
		Status: model.CouponRedemptionStatusReleased, PercentOff: 10, Currency: "USD",
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodDelete, "/api/subscription/admin/coupons/"+strconv.Itoa(int(coupon.Id)), nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(coupon.Id))}}
	ctx.Set("id", 1)
	AdminDeleteSubscriptionCoupon(ctx)

	success, _, data := decodeCouponEnvelope(t, recorder)
	require.True(t, success)
	respBytes, err := common.Marshal(data)
	require.NoError(t, err)
	var delResp struct {
		HardDeleted bool `json:"hard_deleted"`
	}
	require.NoError(t, common.Unmarshal(respBytes, &delResp))
	assert.True(t, delResp.HardDeleted, "released redemptions must not block hard delete")

	var count int64
	require.NoError(t, model.DB.Model(&model.SubscriptionCoupon{}).Where("id = ?", coupon.Id).Count(&count).Error)
	assert.EqualValues(t, 0, count)
}

// TestAdminDeleteSubscriptionCoupon_NotFound verifies a missing coupon is
// rejected with success=false.
func TestAdminDeleteSubscriptionCoupon_NotFound(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	ctx, recorder := newCouponJSONRequest(t, http.MethodDelete, "/api/subscription/admin/coupons/9999", nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: "9999"}}
	ctx.Set("id", 1)
	AdminDeleteSubscriptionCoupon(ctx)

	success, msg, _ := decodeCouponEnvelope(t, recorder)
	require.False(t, success)
	assert.Contains(t, msg, "not found")
}

// ---------------------------------------------------------------------------
// AdminCreateSubscriptionCoupon — validation + Stripe-secret guard (no real
// Stripe API calls; the Stripe API branch is unreachable without a valid
// sk_/rk_ secret).
// ---------------------------------------------------------------------------

// TestAdminCreateSubscriptionCoupon_RejectsInvalidInput verifies the
// validation layer rejects bad input before touching the Stripe API or the DB.
// Per project convention, binding-tag violations are reported as HTTP 200 with
// success=false (the same envelope used by common.ApiErrorMsg), so the test
// asserts on the success flag rather than the HTTP status code.
func TestAdminCreateSubscriptionCoupon_RejectsInvalidInput(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	tests := []struct {
		name string
		req  CreateSubscriptionCouponRequest
	}{
		{
			name: "percent_off out of range",
			req:  CreateSubscriptionCouponRequest{Code: "BAD", Name: "Bad", PercentOff: 150, DurationMonths: 1},
		},
		{
			name: "duration_months below 1",
			req:  CreateSubscriptionCouponRequest{Code: "BAD", Name: "Bad", PercentOff: 10, DurationMonths: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/admin/coupons", tt.req, 1)
			AdminCreateSubscriptionCoupon(ctx)
			success, _, _ := decodeCouponEnvelope(t, recorder)
			require.False(t, success, "invalid input must be rejected (success=false)")

			// No coupon row must be created.
			var count int64
			require.NoError(t, model.DB.Model(&model.SubscriptionCoupon{}).Where("code = ?", tt.req.Code).Count(&count).Error)
			assert.EqualValues(t, 0, count)
		})
	}
}

// TestAdminCreateSubscriptionCoupon_RejectsDuplicateCode verifies that a
// duplicate code is rejected with HTTP 409 (Conflict) before any Stripe API
// call. This guards the "Stripe Coupon created then local insert fails"
// cleanup path by never reaching the Stripe API for a duplicate.
func TestAdminCreateSubscriptionCoupon_RejectsDuplicateCode(t *testing.T) {
	setupSubscriptionCouponTestDB(t)

	existing := &model.SubscriptionCoupon{
		Code: "DUP", Name: "Dup", StripeCouponId: "stripe_dup",
		PercentOff: 10, DurationMonths: 1, Enabled: true,
	}
	require.NoError(t, model.DB.Create(existing).Error)

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/admin/coupons",
		CreateSubscriptionCouponRequest{Code: "DUP", Name: "Dup2", PercentOff: 20, DurationMonths: 2}, 1)
	AdminCreateSubscriptionCoupon(ctx)

	require.Equal(t, http.StatusConflict, recorder.Code)

	// The existing row must be untouched.
	var refreshed model.SubscriptionCoupon
	require.NoError(t, model.DB.First(&refreshed, existing.Id).Error)
	assert.Equal(t, "Dup", refreshed.Name)
	assert.Equal(t, 10, refreshed.PercentOff)
}

// TestAdminCreateSubscriptionCoupon_RejectsWhenStripeNotConfigured verifies
// that when StripeApiSecret is not a valid sk_/rk_ key, the endpoint returns
// HTTP 503 without creating a local coupon row. This is the guard that
// prevents creating a local coupon without a corresponding Stripe Coupon
// (requirement 4: the discount must be applied at Stripe Checkout).
func TestAdminCreateSubscriptionCoupon_RejectsWhenStripeNotConfigured(t *testing.T) {
	setupSubscriptionCouponTestDB(t)
	// StripeApiSecret is "" (set by setupSubscriptionCouponTestDB).

	ctx, recorder := newCouponJSONRequest(t, http.MethodPost, "/api/subscription/admin/coupons",
		CreateSubscriptionCouponRequest{Code: "NOSTRIPE", Name: "NoStripe", PercentOff: 10, DurationMonths: 1}, 1)
	AdminCreateSubscriptionCoupon(ctx)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)

	var count int64
	require.NoError(t, model.DB.Model(&model.SubscriptionCoupon{}).Where("code = ?", "NOSTRIPE").Count(&count).Error)
	assert.EqualValues(t, 0, count, "no local coupon row must be created without a Stripe Coupon")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// strPtr returns a pointer to the given string. Used to populate the
// optional *string fields of UpdateSubscriptionCouponRequest.
func strPtr(s string) *string { return &s }
