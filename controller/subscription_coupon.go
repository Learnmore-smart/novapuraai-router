package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/coupon"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Admin coupon CRUD
// ---------------------------------------------------------------------------

// CreateSubscriptionCouponRequest is the DTO for POST /admin/coupons.
// PercentOff and DurationMonths are immutable after creation (they map 1:1 to
// the Stripe Coupon that is created at coupon-create time per requirement 4).
type CreateSubscriptionCouponRequest struct {
	Code           string `json:"code" binding:"required"`
	Name           string `json:"name" binding:"required"`
	PercentOff     int    `json:"percent_off" binding:"required,min=1,max=100"`
	DurationMonths int    `json:"duration_months" binding:"required,min=1"`
	Enabled        bool   `json:"enabled"`
	StartAt        int64  `json:"start_at"`
	EndAt          int64  `json:"end_at"`
	MaxRedemptions int    `json:"max_redemptions"`
	PerUserLimit   int    `json:"per_user_limit"`
	NewUserOnly    bool   `json:"new_user_only"`
}

// UpdateSubscriptionCouponRequest is the DTO for PUT /admin/coupons/:id. Only
// the mutable fields are present; Code / PercentOff / DurationMonths /
// StripeCouponId are immutable post-creation (changing them would desync the
// local row from the Stripe Coupon).
type UpdateSubscriptionCouponRequest struct {
	Name           *string `json:"name"`
	Enabled        *bool   `json:"enabled"`
	StartAt        *int64  `json:"start_at"`
	EndAt          *int64  `json:"end_at"`
	MaxRedemptions *int    `json:"max_redemptions"`
	PerUserLimit   *int    `json:"per_user_limit"`
	NewUserOnly    *bool   `json:"new_user_only"`
}

// validateCreateCouponRequest applies the canonical validation rules for a new
// coupon. Extracted from the HTTP handler so it can be unit-tested directly
// (the binding tags on CreateSubscriptionCouponRequest provide a first line of
// defense via Gin, but this function is the source of truth).
func validateCreateCouponRequest(req *CreateSubscriptionCouponRequest) error {
	if strings.TrimSpace(req.Code) == "" {
		return errors.New("code is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if req.PercentOff < 1 || req.PercentOff > 100 {
		return errors.New("percent_off must be between 1 and 100")
	}
	if req.DurationMonths < 1 {
		return errors.New("duration_months must be >= 1")
	}
	if req.StartAt > 0 && req.EndAt > 0 && req.EndAt <= req.StartAt {
		return errors.New("end_at must be greater than start_at")
	}
	if req.MaxRedemptions < 0 {
		return errors.New("max_redemptions must be >= 0")
	}
	if req.PerUserLimit < 0 {
		return errors.New("per_user_limit must be >= 0")
	}
	return nil
}

// AdminListSubscriptionCoupons lists coupons with simple pagination. Mirrors
// the AdminBillingTopupOrders pagination pattern (page / page_size).
func AdminListSubscriptionCoupons(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var coupons []model.SubscriptionCoupon
	var total int64
	if err := model.DB.Model(&model.SubscriptionCoupon{}).Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DB.Order("id desc").Limit(pageSize).Offset(offset).Find(&coupons).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items": coupons,
		"total": total,
	})
}

// AdminGetSubscriptionCoupon returns a single coupon plus its redemption
// stats (total redemptions and active redemptions).
func AdminGetSubscriptionCoupon(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var coupon model.SubscriptionCoupon
	if err := model.DB.First(&coupon, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "coupon not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	var totalRedemptions int64
	var activeRedemptions int64
	if err := model.DB.Model(&model.SubscriptionCouponRedemption{}).
		Where("coupon_id = ?", coupon.Id).Count(&totalRedemptions).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DB.Model(&model.SubscriptionCouponRedemption{}).
		Where("coupon_id = ? AND status IN ?", coupon.Id,
			[]string{model.CouponRedemptionStatusReserved, model.CouponRedemptionStatusIssued}).
		Count(&activeRedemptions).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"coupon":             coupon,
		"total_redemptions":  totalRedemptions,
		"active_redemptions": activeRedemptions,
	})
}

// AdminCreateSubscriptionCoupon creates the Stripe Coupon via the Stripe API
// first, then inserts the local SubscriptionCoupon row referencing the Stripe
// Coupon ID. Per requirement 4 the discount must be applied at Stripe
// Checkout, so a local coupon without a Stripe Coupon is never created — a
// Stripe API failure returns 502 without writing anything locally.
func AdminCreateSubscriptionCoupon(c *gin.Context) {
	var req CreateSubscriptionCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if err := validateCreateCouponRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Code uniqueness check (case-sensitive — the spec defines Code as the
	// canonical user-typed string).
	var existing model.SubscriptionCoupon
	if err := model.DB.Where("code = ?", req.Code).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "coupon code already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		common.ApiError(c, err)
		return
	}

	// Stripe credential guards (mirror subscription_checkout_stripe.go).
	if !strings.HasPrefix(setting.StripeApiSecret, "sk_") && !strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Stripe 未配置或密钥无效"})
		return
	}

	// Create the Stripe Coupon first. Duration="repeating" + DurationInMonths
	// applies the percent_off discount for the next DurationInMonths billing
	// periods (matches the local DurationMonths semantics).
	stripe.Key = setting.StripeApiSecret
	params := &stripe.CouponParams{
		Name:             stripe.String(req.Name),
		PercentOff:       stripe.Float64(float64(req.PercentOff)),
		Duration:         stripe.String(string(stripe.CouponDurationRepeating)),
		DurationInMonths: stripe.Int64(int64(req.DurationMonths)),
	}
	stripeCoupon, err := coupon.New(params)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("admin create subscription coupon stripe api failed code=%s err=%q", req.Code, err.Error()))
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "Stripe Coupon 创建失败", "data": err.Error()})
		return
	}

	couponRow := &model.SubscriptionCoupon{
		Code:           req.Code,
		Name:           req.Name,
		StripeCouponId: stripeCoupon.ID,
		PercentOff:     req.PercentOff,
		DurationMonths: req.DurationMonths,
		Enabled:        req.Enabled,
		StartAt:        req.StartAt,
		EndAt:          req.EndAt,
		MaxRedemptions: req.MaxRedemptions,
		PerUserLimit:   req.PerUserLimit,
		NewUserOnly:    req.NewUserOnly,
		TimesRedeemed:  0,
	}
	if err := model.DB.Create(couponRow).Error; err != nil {
		// Best-effort: delete the Stripe Coupon so we don't leave an orphan.
		// A failure here is logged but does not block the error response.
		if _, delErr := coupon.Del(stripeCoupon.ID, nil); delErr != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("admin create subscription coupon rollback stripe delete failed stripe_id=%s err=%q", stripeCoupon.ID, delErr.Error()))
		}
		common.ApiError(c, err)
		return
	}

	recordManageAudit(c, "subscription.coupon_create", map[string]interface{}{
		"coupon_id":       couponRow.Id,
		"code":            couponRow.Code,
		"name":            couponRow.Name,
		"percent_off":     couponRow.PercentOff,
		"duration_months": couponRow.DurationMonths,
		"stripe_coupon_id": couponRow.StripeCouponId,
	})
	common.ApiSuccess(c, couponRow)
}

// AdminUpdateSubscriptionCoupon updates the mutable fields of a coupon. Code,
// PercentOff, DurationMonths and StripeCouponId are immutable (changing them
// would desync from the Stripe Coupon created at coupon-create time).
func AdminUpdateSubscriptionCoupon(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req UpdateSubscriptionCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	// Validate the request fields that have constraints.
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "name cannot be empty"})
		return
	}
	if req.MaxRedemptions != nil && *req.MaxRedemptions < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "max_redemptions must be >= 0"})
		return
	}
	if req.PerUserLimit != nil && *req.PerUserLimit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "per_user_limit must be >= 0"})
		return
	}
	// end_at > start_at when both are being set in this request OR when only
	// one is being set, compare against the existing row's value.
	var current model.SubscriptionCoupon
	if err := model.DB.First(&current, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "coupon not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	startAt := current.StartAt
	endAt := current.EndAt
	if req.StartAt != nil {
		startAt = *req.StartAt
	}
	if req.EndAt != nil {
		endAt = *req.EndAt
	}
	if startAt > 0 && endAt > 0 && endAt <= startAt {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "end_at must be greater than start_at"})
		return
	}

	updateMap := map[string]interface{}{
		"updated_at": common.GetTimestamp(),
	}
	if req.Name != nil {
		updateMap["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Enabled != nil {
		updateMap["enabled"] = *req.Enabled
	}
	if req.StartAt != nil {
		updateMap["start_at"] = *req.StartAt
	}
	if req.EndAt != nil {
		updateMap["end_at"] = *req.EndAt
	}
	if req.MaxRedemptions != nil {
		updateMap["max_redemptions"] = *req.MaxRedemptions
	}
	if req.PerUserLimit != nil {
		updateMap["per_user_limit"] = *req.PerUserLimit
	}
	if req.NewUserOnly != nil {
		updateMap["new_user_only"] = *req.NewUserOnly
	}

	if err := model.DB.Model(&model.SubscriptionCoupon{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	recordManageAudit(c, "subscription.coupon_update", map[string]interface{}{
		"coupon_id": id,
		"fields":    updateMap,
	})
	common.ApiSuccess(c, nil)
}

// AdminDeleteSubscriptionCoupon deletes a coupon. If any reserved/issued
// redemptions exist (active discounts), the coupon is disabled instead of
// hard-deleted to preserve audit history. When no redemptions exist, the
// coupon is hard-deleted locally and the corresponding Stripe Coupon is
// deleted via the Stripe API.
func AdminDeleteSubscriptionCoupon(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}

	var couponRow model.SubscriptionCoupon
	if err := model.DB.First(&couponRow, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "coupon not found")
			return
		}
		common.ApiError(c, err)
		return
	}

	// Active redemptions (reserved/issued) block hard-delete — disable instead.
	var activeCount int64
	if err := model.DB.Model(&model.SubscriptionCouponRedemption{}).
		Where("coupon_id = ? AND status IN ?", couponRow.Id,
			[]string{model.CouponRedemptionStatusReserved, model.CouponRedemptionStatusIssued}).
		Count(&activeCount).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if activeCount > 0 {
		if err := model.DB.Model(&model.SubscriptionCoupon{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"enabled":    false,
				"updated_at": common.GetTimestamp(),
			}).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		recordManageAudit(c, "subscription.coupon_delete", map[string]interface{}{
			"coupon_id":    id,
			"hard_delete":  false,
			"active_count": activeCount,
			"reason":       "active redemptions exist; coupon disabled instead",
		})
		common.ApiSuccess(c, gin.H{
			"hard_deleted": false,
			"message":      "coupon has active redemptions; disabled instead of hard-deleted",
		})
		return
	}

	// No active redemptions — hard-delete locally, then best-effort delete the
	// Stripe Coupon. A Stripe API failure is logged but does not roll back the
	// local delete (the coupon is no longer usable locally and the orphaned
	// Stripe Coupon can be cleaned up separately).
	if err := model.DB.Where("id = ?", id).Delete(&model.SubscriptionCoupon{}).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	stripeErr := ""
	if strings.HasPrefix(setting.StripeApiSecret, "sk_") || strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		stripe.Key = setting.StripeApiSecret
		if _, err := coupon.Del(couponRow.StripeCouponId, nil); err != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("admin delete subscription coupon stripe api failed coupon_id=%d stripe_id=%s err=%q", id, couponRow.StripeCouponId, err.Error()))
			stripeErr = err.Error()
		}
	}
	recordManageAudit(c, "subscription.coupon_delete", map[string]interface{}{
		"coupon_id":       id,
		"hard_delete":     true,
		"stripe_coupon_id": couponRow.StripeCouponId,
		"stripe_error":    stripeErr,
	})
	common.ApiSuccess(c, gin.H{
		"hard_deleted": true,
		"stripe_error": stripeErr,
	})
}

// ---------------------------------------------------------------------------
// Public coupon validate endpoint
// ---------------------------------------------------------------------------

// ValidateCouponRequest is the DTO for POST /api/subscription/coupons/validate.
// The endpoint is read-only: it does NOT reserve the coupon, increment
// TimesRedeemed, or create a redemption row. Reservation happens at checkout.
type ValidateCouponRequest struct {
	Code          string `json:"code" binding:"required"`
	PlanId        int    `json:"plan_id" binding:"required"`
	Mode          string `json:"mode" binding:"required"`     // "auto_renew" or "prepaid"
	Currency      string `json:"currency" binding:"required"` // "CNY" or "USD"
	PrepaidMonths int    `json:"prepaid_months"`              // only for prepaid mode
}

// ValidateCouponResponse is the breakdown the frontend confirmation modal
// renders (requirement 4). All money fields are major-unit display values
// rounded to 2 decimal places.
type ValidateCouponResponse struct {
	Valid            bool    `json:"valid"`
	Reason           string  `json:"reason"`            // empty if valid
	CouponName       string  `json:"coupon_name"`
	PercentOff       int     `json:"percent_off"`
	DurationMonths   int     `json:"duration_months"`
	OriginalPrice    float64 `json:"original_price"`    // major units, in the requested currency
	DiscountAmount   float64 `json:"discount_amount"`   // major units
	FinalAmount      float64 `json:"final_amount"`      // major units
	NextRenewalPrice float64 `json:"next_renewal_price"` // major units; for auto_renew = finalAmount (discount applies for DurationMonths), for prepaid = original (one-time discount)
	Currency         string  `json:"currency"`
}

// ValidateSubscriptionCouponForUser validates a coupon code against a plan and
// returns the price breakdown the frontend needs to render the confirmation
// modal. Read-only: does not reserve the coupon or create any redemption.
//
// Wired to POST /api/subscription/coupons/validate under middleware.UserAuth().
func ValidateSubscriptionCouponForUser(c *gin.Context) {
	var req ValidateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.Mode = strings.TrimSpace(req.Mode)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Mode != subscriptionCheckoutModeAutoRenew && req.Mode != subscriptionCheckoutModePrepaid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid mode; must be auto_renew or prepaid"})
		return
	}
	if req.Currency != "CNY" && req.Currency != "USD" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid currency; must be CNY or USD"})
		return
	}
	if req.Mode == subscriptionCheckoutModePrepaid && req.PrepaidMonths <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "prepaid_months must be > 0 for prepaid mode"})
		return
	}

	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil || plan == nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}

	// Read-only coupon validation. An invalid coupon is a validation result,
	// not a server error: return HTTP 200 with valid=false.
	couponRow, err := model.ValidateSubscriptionCoupon(req.Code, userId)
	if err != nil {
		common.ApiSuccess(c, ValidateCouponResponse{
			Valid:    false,
			Reason:   err.Error(),
			Currency: req.Currency,
		})
		return
	}

	// Compute the breakdown using decimal arithmetic for safety (the values
	// are display-only, but going through decimal avoids float drift and keeps
	// the rounding deterministic at 2 decimal places).
	original := resolvePriceAmount(plan, req.Currency)
	months := 1
	if req.Mode == subscriptionCheckoutModePrepaid {
		months = req.PrepaidMonths
	}
	originalDecimal := decimal.NewFromFloat(original).Mul(decimal.NewFromInt(int64(months)))
	discountDecimal := originalDecimal.
		Mul(decimal.NewFromInt(int64(couponRow.PercentOff))).
		Div(decimal.NewFromInt(100))
	finalDecimal := originalDecimal.Sub(discountDecimal)
	if finalDecimal.IsNegative() {
		finalDecimal = decimal.Zero
	}

	// nextRenewalPrice: for auto_renew the discount applies for DurationMonths
	// renewals, so the next renewal is the discounted price. For prepaid there
	// is no renewal (one-time charge), so the field reports the original price
	// for clarity.
	nextRenewal := finalDecimal
	if req.Mode == subscriptionCheckoutModePrepaid {
		nextRenewal = originalDecimal
	}

	resp := ValidateCouponResponse{
		Valid:            true,
		CouponName:       couponRow.Name,
		PercentOff:       couponRow.PercentOff,
		DurationMonths:   couponRow.DurationMonths,
		OriginalPrice:    originalDecimal.Round(2).InexactFloat64(),
		DiscountAmount:   discountDecimal.Round(2).InexactFloat64(),
		FinalAmount:      finalDecimal.Round(2).InexactFloat64(),
		NextRenewalPrice: nextRenewal.Round(2).InexactFloat64(),
		Currency:         req.Currency,
	}
	common.ApiSuccess(c, resp)
}
