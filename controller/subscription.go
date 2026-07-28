package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---- Shared types ----

type SubscriptionPlanDTO struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

type BillingPreferenceRequest struct {
	BillingPreference string `json:"billing_preference"`
}

type SubscriptionBalancePayRequest struct {
	PlanId int `json:"plan_id"`
}

// SubscriptionSelfDTO is the user-facing serialization of a UserSubscription.
// It embeds the full UserSubscription (so all existing fields are preserved)
// and adds NovaPura lifecycle / display fields consumed by the frontend.
type SubscriptionSelfDTO struct {
	model.UserSubscription
	// NovaPura linkage / lifecycle fields (mirror the model columns).
	StripeSubscriptionId string `json:"stripe_subscription_id"`
	StripeCustomerId     string `json:"stripe_customer_id"`
	CancelAtPeriodEnd    bool   `json:"cancel_at_period_end"`
	CouponId             *int   `json:"coupon_id"`
	CouponRedemptionId   *int   `json:"coupon_redemption_id"`
	BillingCycleAnchor   int64  `json:"billing_cycle_anchor"`
	// Display-only fields resolved from the plan / status.
	Currency        string `json:"currency"`
	NextRenewalDate int64  `json:"next_renewal_date"`
	IsAutoRenew     bool   `json:"is_auto_renew"`
	DisplayStatus   string `json:"display_status"`
}

// buildSubscriptionSelfDTO maps a UserSubscription + its plan to the DTO the
// /api/subscription/self endpoint returns. The plan is used to resolve the
// display currency; a nil plan falls back to USD.
func buildSubscriptionSelfDTO(sub *model.UserSubscription, plan *model.SubscriptionPlan) SubscriptionSelfDTO {
	dto := SubscriptionSelfDTO{
		UserSubscription:     *sub,
		StripeSubscriptionId: sub.StripeSubscriptionId,
		StripeCustomerId:     sub.StripeCustomerId,
		CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		CouponId:             sub.CouponId,
		CouponRedemptionId:   sub.CouponRedemptionId,
		BillingCycleAnchor:   sub.BillingCycleAnchor,
		Currency:             resolveSubscriptionDisplayCurrency(plan),
		NextRenewalDate:      sub.EndTime,
		IsAutoRenew:          isSubscriptionAutoRenewStatus(sub.Status),
		DisplayStatus:        subscriptionDisplayStatus(sub.Status),
	}
	return dto
}

// resolveSubscriptionDisplayCurrency picks the display currency for a plan.
// NovaPura plans carry dual Stripe price IDs (CNY/USD); when only one is
// configured that currency wins. Otherwise the plan's legacy Currency column
// is used, defaulting to USD.
func resolveSubscriptionDisplayCurrency(plan *model.SubscriptionPlan) string {
	if plan != nil {
		hasCNY := strings.TrimSpace(plan.StripePriceIdCNY) != ""
		hasUSD := strings.TrimSpace(plan.StripePriceIdUSD) != ""
		switch {
		case hasCNY && !hasUSD:
			return "CNY"
		case hasUSD && !hasCNY:
			return "USD"
		}
		if c := strings.ToUpper(strings.TrimSpace(plan.Currency)); c == "CNY" || c == "USD" {
			return c
		}
	}
	return "USD"
}

// isSubscriptionAutoRenewStatus returns true for the auto-renew lifecycle
// statuses (active / canceling / past_due / payment_failed). prepaid_active is
// the only non-auto-renew live status; terminal states (canceled / expired)
// return false since the distinction is no longer meaningful for display.
func isSubscriptionAutoRenewStatus(status string) bool {
	switch status {
	case model.SubscriptionStatusActive,
		model.SubscriptionStatusCanceling,
		model.SubscriptionStatusPastDue,
		model.SubscriptionStatusPaymentFailed:
		return true
	}
	return false
}

// subscriptionDisplayStatus maps a raw status to a frontend-friendly label.
func subscriptionDisplayStatus(status string) string {
	switch status {
	case model.SubscriptionStatusActive:
		return "Active"
	case model.SubscriptionStatusPrepaidActive:
		return "Active"
	case model.SubscriptionStatusCanceling:
		return "Canceling"
	case model.SubscriptionStatusCanceled:
		return "Canceled"
	case model.SubscriptionStatusPastDue:
		return "Past due"
	case model.SubscriptionStatusPaymentFailed:
		return "Payment failed"
	case model.SubscriptionStatusExpired:
		return "Expired"
	}
	return status
}

// ---- User APIs ----

// validateSubscriptionPlanFields validates the NovaPura Phase 1 fields on a
// SubscriptionPlan. It is called by both AdminCreateSubscriptionPlan and
// AdminUpdateSubscriptionPlan so the rules stay in one place. The legacy
// fields (Title, PriceAmount, Currency, etc.) are still validated inline in
// each handler; this helper only covers the new dual-currency / Stripe /
// auto-renew / prepaid fields.
func validateSubscriptionPlanFields(plan *model.SubscriptionPlan) error {
	if plan == nil {
		return errors.New("plan is nil")
	}
	// Dual-currency display prices: 0..9999 (matches legacy PriceAmount bounds).
	if plan.PriceAmountCNY < 0 || plan.PriceAmountCNY > 9999 {
		return errors.New("price_amount_cny must be between 0 and 9999")
	}
	if plan.PriceAmountUSD < 0 || plan.PriceAmountUSD > 9999 {
		return errors.New("price_amount_usd must be between 0 and 9999")
	}
	// Renewal prices (post-coupon-expiry): 0 means "same as PriceAmount".
	if plan.RenewalPriceCNY < 0 {
		return errors.New("renewal_price_cny must be >= 0")
	}
	if plan.RenewalPriceUSD < 0 {
		return errors.New("renewal_price_usd must be >= 0")
	}
	// Auto-renew requires at least one recurring Stripe price so the renewal
	// cycle can be driven by Stripe Subscription billing.
	if plan.AutoRenew {
		if strings.TrimSpace(plan.StripePriceIdCNY) == "" && strings.TrimSpace(plan.StripePriceIdUSD) == "" {
			return errors.New("auto_renew plans must set at least one of stripe_price_id_cny or stripe_price_id_usd")
		}
	}
	// PrepaidMonths: empty = no prepaid option. Non-empty = CSV of positive ints.
	if strings.TrimSpace(plan.PrepaidMonths) != "" {
		months, err := parsePrepaidMonthsCSV(plan.PrepaidMonths)
		if err != nil {
			return err
		}
		if len(months) == 0 {
			return errors.New("prepaid_months must contain at least one value")
		}
		// Prepaid mode uses dynamic price_data which needs a Stripe product.
		if strings.TrimSpace(plan.StripeProductId) == "" {
			return errors.New("stripe_product_id is required when prepaid_months is set")
		}
	}
	return nil
}

// parsePrepaidMonthsCSV parses a CSV like "1,3,6,12" into a slice of positive
// ints. Returns an error if any token is missing, non-integer, or <= 0.
func parsePrepaidMonthsCSV(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, errors.New("prepaid_months contains an empty value")
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("prepaid_months contains non-integer value %q", p)
		}
		if n <= 0 {
			return nil, fmt.Errorf("prepaid_months value must be > 0, got %d", n)
		}
		out = append(out, n)
	}
	return out, nil
}

func GetSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Where("enabled = ?", true).Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		p.NormalizeDefaults()
		result = append(result, SubscriptionPlanDTO{
			Plan: p,
		})
	}
	common.ApiSuccess(c, result)
}

func GetSubscriptionSelf(c *gin.Context) {
	userId := c.GetInt("id")
	settingMap, _ := model.GetUserSetting(userId, false)
	pref := common.NormalizeBillingPreference(settingMap.BillingPreference)

	// Get all subscriptions (including expired)
	allSubscriptions, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		allSubscriptions = []model.SubscriptionSummary{}
	}

	// Get active subscriptions for backward compatibility
	activeSubscriptions, err := model.GetAllActiveUserSubscriptions(userId)
	if err != nil {
		activeSubscriptions = []model.SubscriptionSummary{}
	}

	// NovaPura Phase 3: derive the display DTO for the most-recent active
	// subscription (nil when the user has none) and resolve the user's default
	// subscription currency. Both fields are additive — existing consumers
	// that only read billing_preference/subscriptions/all_subscriptions keep
	// working unchanged.
	var currentSubscription *SubscriptionSelfDTO
	if len(activeSubscriptions) > 0 && activeSubscriptions[0].Subscription != nil {
		sub := activeSubscriptions[0].Subscription
		plan, planErr := model.GetSubscriptionPlanById(sub.PlanId)
		if planErr != nil {
			plan = nil
		}
		dto := buildSubscriptionSelfDTO(sub, plan)
		currentSubscription = &dto
	}
	defaultCurrency := service.ResolveSubscriptionCurrency(settingMap.BillingCurrency)

	common.ApiSuccess(c, gin.H{
		"billing_preference":   pref,
		"subscriptions":        activeSubscriptions, // all active subscriptions
		"all_subscriptions":    allSubscriptions,    // all subscriptions including expired
		"current_subscription": currentSubscription, // NovaPura display DTO (nullable)
		"default_currency":     defaultCurrency,     // NovaPura resolved currency (CNY/USD)
	})
}

func UpdateSubscriptionPreference(c *gin.Context) {
	userId := c.GetInt("id")
	var req BillingPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	pref := common.NormalizeBillingPreference(req.BillingPreference)

	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	current := user.GetSetting()
	current.BillingPreference = pref
	if err := model.UpdateUserSetting(user.Id, current); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"billing_preference": pref})
}

func SubscriptionRequestBalancePay(c *gin.Context) {
	userId := c.GetInt("id")
	var req SubscriptionBalancePayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if err := model.PurchaseSubscriptionWithBalance(userId, req.PlanId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin APIs ----

func AdminListSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		p.NormalizeDefaults()
		result = append(result, SubscriptionPlanDTO{
			Plan: p,
		})
	}
	common.ApiSuccess(c, result)
}

type AdminUpsertSubscriptionPlanRequest struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

func AdminCreateSubscriptionPlan(c *gin.Context) {
	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Plan.Id = 0
	if strings.TrimSpace(req.Plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if req.Plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if req.Plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	if req.Plan.Currency == "" {
		req.Plan.Currency = "USD"
	}
	if req.Plan.AllowBalancePay == nil {
		req.Plan.AllowBalancePay = common.GetPointer(true)
	}
	if req.Plan.AllowWalletOverflow == nil {
		req.Plan.AllowWalletOverflow = common.GetPointer(true)
	}
	if req.Plan.DurationUnit == "" {
		req.Plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if req.Plan.DurationValue <= 0 && req.Plan.DurationUnit != model.SubscriptionDurationCustom {
		req.Plan.DurationValue = 1
	}
	if req.Plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if req.Plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	req.Plan.UpgradeGroup = strings.TrimSpace(req.Plan.UpgradeGroup)
	if req.Plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	req.Plan.DowngradeGroup = strings.TrimSpace(req.Plan.DowngradeGroup)
	if req.Plan.DowngradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.DowngradeGroup]; !ok {
			common.ApiErrorMsg(c, "降级分组不存在")
			return
		}
	}
	req.Plan.QuotaResetPeriod = model.NormalizeResetPeriod(req.Plan.QuotaResetPeriod)
	if req.Plan.QuotaResetPeriod == model.SubscriptionResetCustom && req.Plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}
	if err := validateSubscriptionPlanFields(&req.Plan); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	err := model.DB.Create(&req.Plan).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(req.Plan.Id)
	common.ApiSuccess(c, req.Plan)
}

func AdminUpdateSubscriptionPlan(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.Plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if req.Plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if req.Plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	req.Plan.Id = id
	if req.Plan.Currency == "" {
		req.Plan.Currency = "USD"
	}
	if req.Plan.DurationUnit == "" {
		req.Plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if req.Plan.DurationValue <= 0 && req.Plan.DurationUnit != model.SubscriptionDurationCustom {
		req.Plan.DurationValue = 1
	}
	if req.Plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if req.Plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	req.Plan.UpgradeGroup = strings.TrimSpace(req.Plan.UpgradeGroup)
	if req.Plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	req.Plan.DowngradeGroup = strings.TrimSpace(req.Plan.DowngradeGroup)
	if req.Plan.DowngradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.DowngradeGroup]; !ok {
			common.ApiErrorMsg(c, "降级分组不存在")
			return
		}
	}
	req.Plan.QuotaResetPeriod = model.NormalizeResetPeriod(req.Plan.QuotaResetPeriod)
	if req.Plan.QuotaResetPeriod == model.SubscriptionResetCustom && req.Plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}
	if err := validateSubscriptionPlanFields(&req.Plan); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		// update plan (allow zero values updates with map)
		updateMap := map[string]interface{}{
			"title":                      req.Plan.Title,
			"subtitle":                   req.Plan.Subtitle,
			"price_amount":               req.Plan.PriceAmount,
			"currency":                   req.Plan.Currency,
			"duration_unit":              req.Plan.DurationUnit,
			"duration_value":             req.Plan.DurationValue,
			"custom_seconds":             req.Plan.CustomSeconds,
			"enabled":                    req.Plan.Enabled,
			"sort_order":                 req.Plan.SortOrder,
			"stripe_price_id":            req.Plan.StripePriceId,
			"creem_product_id":           req.Plan.CreemProductId,
			"waffo_pancake_product_id":   req.Plan.WaffoPancakeProductId,
			"max_purchase_per_user":      req.Plan.MaxPurchasePerUser,
			"total_amount":               req.Plan.TotalAmount,
			"upgrade_group":              req.Plan.UpgradeGroup,
			"downgrade_group":            req.Plan.DowngradeGroup,
			"quota_reset_period":         req.Plan.QuotaResetPeriod,
			"quota_reset_custom_seconds": req.Plan.QuotaResetCustomSeconds,
			// NovaPura Phase 1 fields: dual-currency prices, Stripe product/price
			// bindings, auto-renew flag, prepaid-months CSV, and renewal prices.
			"price_amount_cny":           req.Plan.PriceAmountCNY,
			"price_amount_usd":           req.Plan.PriceAmountUSD,
			"stripe_price_id_cny":        req.Plan.StripePriceIdCNY,
			"stripe_price_id_usd":        req.Plan.StripePriceIdUSD,
			"stripe_product_id":          req.Plan.StripeProductId,
			"auto_renew":                 req.Plan.AutoRenew,
			"prepaid_months":             req.Plan.PrepaidMonths,
			"renewal_price_cny":          req.Plan.RenewalPriceCNY,
			"renewal_price_usd":          req.Plan.RenewalPriceUSD,
			"updated_at":                 common.GetTimestamp(),
		}
		if req.Plan.AllowBalancePay != nil {
			updateMap["allow_balance_pay"] = *req.Plan.AllowBalancePay
		}
		if req.Plan.AllowWalletOverflow != nil {
			updateMap["allow_wallet_overflow"] = *req.Plan.AllowWalletOverflow
		}
		if err := tx.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminUpdateSubscriptionPlanStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

func AdminUpdateSubscriptionPlanStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpdateSubscriptionPlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Update("enabled", *req.Enabled).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminBindSubscriptionRequest struct {
	UserId int `json:"user_id"`
	PlanId int `json:"plan_id"`
}

func AdminBindSubscription(c *gin.Context) {
	var req AdminBindSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserId <= 0 || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	msg, err := model.AdminBindSubscription(req.UserId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin: user subscription management ----

func AdminListUserSubscriptions(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	subs, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, subs)
}

type AdminCreateUserSubscriptionRequest struct {
	PlanId int `json:"plan_id"`
}

type AdminResetSubscriptionRequest struct {
	PlanId           int   `json:"plan_id"`
	AdvanceResetTime *bool `json:"advance_reset_time"`
}

func resolveAdvanceResetTime(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func recordSubscriptionResetUserLogs(result *model.SubscriptionResetResult, adminInfo map[string]interface{}) {
	if result == nil || result.ResetCount == 0 {
		return
	}
	content := fmt.Sprintf("管理员重置订阅套餐 %s（ID: %d）额度", result.PlanTitle, result.PlanId)
	for _, userId := range result.AffectedUserIds {
		model.RecordLogWithAdminInfo(userId, model.LogTypeManage, content, adminInfo)
	}
}

// AdminCreateUserSubscription creates a new user subscription from a plan (no payment).
func AdminCreateUserSubscription(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	var req AdminCreateUserSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	msg, err := model.AdminBindSubscription(userId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminResetUserSubscriptionsByPlan(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	var req AdminResetSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	advanceResetTime := resolveAdvanceResetTime(req.AdvanceResetTime)
	result, err := model.AdminResetUserSubscriptionsByPlan(userId, req.PlanId, advanceResetTime)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordSubscriptionResetUserLogs(result, auditOperatorInfo(c))
	recordManageAuditFor(c, userId, "subscription.user_plan_reset", map[string]interface{}{
		"target_user_id":     userId,
		"plan_id":            result.PlanId,
		"plan_title":         result.PlanTitle,
		"reset_count":        result.ResetCount,
		"user_count":         result.UserCount,
		"advance_reset_time": result.AdvanceResetTime,
	})
	common.ApiSuccess(c, result)
}

func AdminResetPlanSubscriptions(c *gin.Context) {
	planId, _ := strconv.Atoi(c.Param("id"))
	if planId <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminResetSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	advanceResetTime := resolveAdvanceResetTime(req.AdvanceResetTime)
	result, err := model.AdminResetPlanSubscriptions(planId, advanceResetTime)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordSubscriptionResetUserLogs(result, auditOperatorInfo(c))
	common.SysLog(fmt.Sprintf("admin reset subscription plan %d quota: reset_count=%d user_count=%d advance_reset_time=%t",
		result.PlanId, result.ResetCount, result.UserCount, result.AdvanceResetTime))
	recordManageAudit(c, "subscription.plan_reset", map[string]interface{}{
		"plan_id":            result.PlanId,
		"plan_title":         result.PlanTitle,
		"reset_count":        result.ResetCount,
		"user_count":         result.UserCount,
		"advance_reset_time": result.AdvanceResetTime,
	})
	common.ApiSuccess(c, result)
}

// AdminInvalidateUserSubscription cancels a user subscription immediately.
func AdminInvalidateUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminInvalidateUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminDeleteUserSubscription hard-deletes a user subscription.
func AdminDeleteUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminDeleteUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin: plan covered-models CRUD ----
//
// These endpoints manage the subscription_plan_covered_models table for a
// plan. The covered-model list is the source of truth for which canonical
// model IDs a subscription to the plan can use at zero balance cost
// (Phase 4's UserHasSubscriptionCoveringModel). Per requirement 9 the
// frontend never decides whether a specific model is free — it only shows
// "unlimited access to included models" — so the public plan-list endpoint
// does NOT expose this list. Only admin endpoints expose the per-model list.

// AdminGetPlanCoveredModels returns the canonical model IDs covered by a plan.
// Wired to GET /api/subscription/admin/plans/:id/models.
func AdminGetPlanCoveredModels(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	if _, err := model.GetSubscriptionPlanById(id); err != nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	models, err := model.GetPlanCoveredModels(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"models": models})
}

// AdminSetPlanCoveredModelsRequest is the DTO for PUT /admin/plans/:id/models.
type AdminSetPlanCoveredModelsRequest struct {
	Models []string `json:"models"`
}

// AdminSetPlanCoveredModels replaces the covered-model list for a plan.
// Wired to PUT /api/subscription/admin/plans/:id/models. Empty/whitespace
// entries and duplicates are stripped before storage. Existence in the
// abilities table is NOT enforced — admins may pre-configure coverage for
// models that have not been added to a channel yet.
func AdminSetPlanCoveredModels(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	if _, err := model.GetSubscriptionPlanById(id); err != nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	var req AdminSetPlanCoveredModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	seen := make(map[string]bool, len(req.Models))
	cleaned := make([]string, 0, len(req.Models))
	for _, m := range req.Models {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		cleaned = append(cleaned, m)
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		return model.SetPlanCoveredModels(tx, id, cleaned)
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidatePlanCoveredModelsCache()
	recordManageAudit(c, "subscription.plan_covered_models_set", map[string]interface{}{
		"plan_id":     id,
		"model_count": len(cleaned),
		"models":      cleaned,
	})
	common.ApiSuccess(c, gin.H{"message": "success"})
}

// AdminAddPlanCoveredModelRequest is the DTO for POST /admin/plans/:id/models.
type AdminAddPlanCoveredModelRequest struct {
	ModelId string `json:"model_id"`
}

// AdminAddPlanCoveredModel appends a single model to the covered list. This
// is a convenience endpoint — the PUT replaces the whole list, this appends
// one. Idempotent: appending an already-covered model is a no-op success.
// Wired to POST /api/subscription/admin/plans/:id/models.
func AdminAddPlanCoveredModel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	if _, err := model.GetSubscriptionPlanById(id); err != nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	var req AdminAddPlanCoveredModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.ModelId = strings.TrimSpace(req.ModelId)
	if req.ModelId == "" {
		common.ApiErrorMsg(c, "model_id 不能为空")
		return
	}
	existing, err := model.GetPlanCoveredModels(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, m := range existing {
		if m == req.ModelId {
			common.ApiSuccess(c, gin.H{"message": "already covered"})
			return
		}
	}
	updated := append(existing, req.ModelId)
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		return model.SetPlanCoveredModels(tx, id, updated)
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidatePlanCoveredModelsCache()
	recordManageAudit(c, "subscription.plan_covered_model_add", map[string]interface{}{
		"plan_id":  id,
		"model_id": req.ModelId,
	})
	common.ApiSuccess(c, gin.H{"message": "success"})
}

// AdminRemovePlanCoveredModel removes a single model from the covered list.
// Wired to DELETE /api/subscription/admin/plans/:id/models/:model_id.
// Idempotent: removing a model that was not covered is a no-op success.
func AdminRemovePlanCoveredModel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	modelId := strings.TrimSpace(c.Param("model_id"))
	if modelId == "" {
		common.ApiErrorMsg(c, "无效的model_id")
		return
	}
	if _, err := model.GetSubscriptionPlanById(id); err != nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	existing, err := model.GetPlanCoveredModels(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	filtered := make([]string, 0, len(existing))
	removed := false
	for _, m := range existing {
		if m == modelId {
			removed = true
			continue
		}
		filtered = append(filtered, m)
	}
	if !removed {
		common.ApiSuccess(c, gin.H{"message": "not covered"})
		return
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		return model.SetPlanCoveredModels(tx, id, filtered)
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidatePlanCoveredModelsCache()
	recordManageAudit(c, "subscription.plan_covered_model_remove", map[string]interface{}{
		"plan_id":  id,
		"model_id": modelId,
	})
	common.ApiSuccess(c, gin.H{"message": "success"})
}

// ---- Admin: model-list endpoint for the covered-model picker ----
//
// The admin UI's covered-model picker needs a list of canonical model IDs to
// choose from. GET /api/pricing returns Pricing structs (with model_name) but
// filters by the caller's usable groups, so an admin would not see ALL
// configured models. This endpoint returns the distinct enabled model names
// from the abilities table (the same source GetEnabledModels uses), with an
// optional ?search= substring filter for large model lists.

// AdminListModelsForSubscription returns the canonical model IDs the system
// knows about, for the admin UI's covered-model picker. Supports an optional
// ?search= query to filter by substring (case-insensitive).
// Wired to GET /api/subscription/admin/models.
func AdminListModelsForSubscription(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	models := model.GetEnabledModels()
	if search == "" {
		common.ApiSuccess(c, gin.H{"data": models})
		return
	}
	needle := strings.ToLower(search)
	filtered := make([]string, 0, len(models))
	for _, m := range models {
		if strings.Contains(strings.ToLower(m), needle) {
			filtered = append(filtered, m)
		}
	}
	common.ApiSuccess(c, gin.H{"data": filtered})
}
