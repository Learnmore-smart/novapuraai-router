package controller

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---- Shared types ----

type SubscriptionPlanDTO struct {
	Plan               model.SubscriptionPlan `json:"plan"`
	RecurringAvailable *bool                  `json:"recurring_available,omitempty"`
}

type BillingPreferenceRequest struct {
	BillingPreference string `json:"billing_preference"`
}

type SubscriptionBalancePayRequest struct {
	PlanId int `json:"plan_id"`
}

// ---- User APIs ----

func GetSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Where("enabled = ?", true).Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		p.NormalizeDefaults()
		if subscriptionPlanHasRecurringFields(&p) {
			if err := model.ValidatePurchasableSubscriptionPlan(&p, model.StripeRecurringPurchaseSource); err != nil {
				// A disabled or mismatched recurring contract is not a public
				// purchasable offer. Keep unrelated legacy plans visible.
				continue
			}
		}
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

	common.ApiSuccess(c, gin.H{
		"billing_preference": pref,
		"subscriptions":      activeSubscriptions, // all active subscriptions
		"all_subscriptions":  allSubscriptions,    // all subscriptions including expired
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
		var recurringAvailable *bool
		if subscriptionPlanHasRecurringFields(&p) {
			available := false
			config, configErr := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
			if configErr == nil && config.Enabled && p.Enabled && p.StripeSubscriptionEnabled && model.ValidateStripeSubscriptionPlan(&p, config, false) == nil {
				available = true
			}
			recurringAvailable = &available
		}
		result = append(result, SubscriptionPlanDTO{
			Plan:               p,
			RecurringAvailable: recurringAvailable,
		})
	}
	common.ApiSuccess(c, result)
}

type AdminUpsertSubscriptionPlanRequest struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

func subscriptionPlanHasRecurringFields(plan *model.SubscriptionPlan) bool {
	if plan == nil {
		return false
	}
	return plan.RecurringCode != nil || plan.StripeSubscriptionEnabled || strings.TrimSpace(plan.StripeSubscriptionModel) != "" ||
		strings.TrimSpace(plan.FounderStripePriceId) != "" || strings.TrimSpace(plan.StandardStripePriceId) != "" ||
		strings.TrimSpace(plan.StripeProductId) != "" || strings.TrimSpace(plan.StripeAccountId) != "" ||
		strings.TrimSpace(plan.StripePortalConfigurationId) != "" || strings.TrimSpace(plan.Code) == model.SandboxStripeSubscriptionPlanCode
}

func prepareAdminSubscriptionPlan(plan *model.SubscriptionPlan, existing *model.SubscriptionPlan) (bool, error) {
	if plan == nil {
		return false, model.ErrStripeSubscriptionPlanInvalid
	}
	recurring := subscriptionPlanHasRecurringFields(plan)
	if existing != nil && existing.RecurringCode != nil {
		recurring = true
		if strings.TrimSpace(plan.Code) == "" {
			plan.Code = existing.Code
		}
	}
	if !recurring {
		return false, nil
	}
	config, err := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
	if err != nil {
		return true, err
	}
	if !config.Enabled && (plan.Enabled || plan.StripeSubscriptionEnabled) {
		return true, model.ErrStripeSubscriptionDisabled
	}
	if strings.TrimSpace(plan.Code) == "" && existing != nil {
		plan.Code = existing.Code
	}
	if strings.TrimSpace(plan.Code) == "" {
		plan.Code = config.PlanCode
	}
	recurringCode := config.PlanCode
	plan.RecurringCode = &recurringCode
	if plan.Enabled && !plan.StripeSubscriptionEnabled {
		return true, model.ErrStripeSubscriptionDisabled
	}
	return true, model.ValidateStripeSubscriptionPlan(plan, config, plan.Enabled && plan.StripeSubscriptionEnabled)
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
		if !(subscriptionPlanHasRecurringFields(&req.Plan) && req.Plan.UpgradeGroup == model.SandboxStripeSubscriptionGroup) {
			if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
				common.ApiErrorMsg(c, "升级分组不存在")
				return
			}
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
	if subscriptionPlanHasRecurringFields(&req.Plan) {
		// Recurring admin requests are validated against the fixed CNY catalog;
		// do this before prepareAdminSubscriptionPlan so omitted currency cannot
		// inherit the legacy USD default and fail an otherwise valid upsert.
		config, err := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		req.Plan.Currency = config.Currency
	}
	recurring, err := prepareAdminSubscriptionPlan(&req.Plan, nil)
	if err != nil {
		common.ApiError(c, err)
		return
	} else if recurring {
		config, _ := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
		req.Plan.Currency = config.Currency
	}
	requestedEnabled := req.Plan.Enabled
	requestedStripeSubscriptionEnabled := req.Plan.StripeSubscriptionEnabled
	if recurring {
		err = model.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&req.Plan).Error; err != nil {
				return err
			}
			// GORM may omit false zero values because of the legacy default
			// tag. Keep creation and the explicit recurring flags in one
			// transaction so no partially enabled fixed plan is observable.
			return tx.Model(&model.SubscriptionPlan{}).Where("id = ?", req.Plan.Id).Updates(map[string]any{
				"enabled":                     requestedEnabled,
				"stripe_subscription_enabled": requestedStripeSubscriptionEnabled,
			}).Error
		})
	} else {
		err = model.DB.Create(&req.Plan).Error
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if recurring {
		req.Plan.Enabled = requestedEnabled
		req.Plan.StripeSubscriptionEnabled = requestedStripeSubscriptionEnabled
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
		if !(subscriptionPlanHasRecurringFields(&req.Plan) && req.Plan.UpgradeGroup == model.SandboxStripeSubscriptionGroup) {
			if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
				common.ApiErrorMsg(c, "升级分组不存在")
				return
			}
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
	var existingPlan model.SubscriptionPlan
	if err := model.DB.First(&existingPlan, id).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if subscriptionPlanHasRecurringFields(&req.Plan) || existingPlan.RecurringCode != nil {
		// Existing recurring rows may be updated with a sparse admin payload;
		// the catalog validator still receives its authoritative CNY currency.
		config, err := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		req.Plan.Currency = config.Currency
	}
	recurring, err := prepareAdminSubscriptionPlan(&req.Plan, &existingPlan)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if recurring {
		config, _ := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
		req.Plan.Currency = config.Currency
	}

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		// update plan (allow zero values updates with map)
		updateMap := map[string]interface{}{
			"code":                       req.Plan.Code,
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
			"updated_at":                 common.GetTimestamp(),
		}
		if recurring {
			updateMap["recurring_code"] = *req.Plan.RecurringCode
			updateMap["stripe_subscription_enabled"] = req.Plan.StripeSubscriptionEnabled
			updateMap["stripe_subscription_model"] = req.Plan.StripeSubscriptionModel
			updateMap["max_active_subscriptions"] = req.Plan.MaxActiveSubscriptions
			updateMap["founder_purchase_limit"] = req.Plan.FounderPurchaseLimit
			updateMap["max_active_per_user"] = req.Plan.MaxActivePerUser
			updateMap["founder_stripe_price_id"] = req.Plan.FounderStripePriceId
			updateMap["standard_stripe_price_id"] = req.Plan.StandardStripePriceId
			updateMap["founder_amount_minor"] = req.Plan.FounderAmountMinor
			updateMap["standard_amount_minor"] = req.Plan.StandardAmountMinor
			updateMap["stripe_currency"] = req.Plan.StripeCurrency
			updateMap["stripe_product_id"] = req.Plan.StripeProductId
			updateMap["stripe_account_id"] = req.Plan.StripeAccountId
			updateMap["stripe_portal_configuration_id"] = req.Plan.StripePortalConfigurationId
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
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var plan model.SubscriptionPlan
		if err := model.LockForUpdate(tx).Where("id = ?", id).First(&plan).Error; err != nil {
			return err
		}
		recurring := subscriptionPlanHasRecurringFields(&plan)
		if *req.Enabled && recurring {
			plan.Enabled = true
			plan.StripeSubscriptionEnabled = true
			if _, err := prepareAdminSubscriptionPlan(&plan, &plan); err != nil {
				return err
			}
		} else if !*req.Enabled && recurring {
			// Disabling the recurring offer turns off both gates so a later
			// maintenance path cannot mistake a disabled row for an enabled
			// recurring catalog entry.
			plan.StripeSubscriptionEnabled = false
		}
		updates := map[string]any{
			"enabled":                     *req.Enabled,
			"stripe_subscription_enabled": plan.StripeSubscriptionEnabled,
		}
		if recurring && plan.RecurringCode != nil {
			updates["recurring_code"] = *plan.RecurringCode
		}
		return tx.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(updates).Error
	})
	if err != nil {
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
