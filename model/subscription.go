package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Subscription duration units
const (
	SubscriptionDurationYear   = "year"
	SubscriptionDurationMonth  = "month"
	SubscriptionDurationDay    = "day"
	SubscriptionDurationHour   = "hour"
	SubscriptionDurationCustom = "custom"
)

// Subscription quota reset period
const (
	SubscriptionResetNever   = "never"
	SubscriptionResetDaily   = "daily"
	SubscriptionResetWeekly  = "weekly"
	SubscriptionResetMonthly = "monthly"
	SubscriptionResetCustom  = "custom"
)

// Subscription status values. Stored as strings in UserSubscription.Status.
const (
	SubscriptionStatusActive        = "active"         // auto-renew subscription, paid & current
	SubscriptionStatusPrepaidActive = "prepaid_active" // prepaid months, not auto-renewing
	SubscriptionStatusCanceling     = "canceling"      // auto-renew with cancel_at_period_end=true
	SubscriptionStatusCanceled      = "canceled"       // subscription ended after period
	SubscriptionStatusPastDue       = "past_due"       // renewal charge failed, grace period
	SubscriptionStatusPaymentFailed = "payment_failed" // renewal ultimately failed
	SubscriptionStatusExpired       = "expired"        // past EndTime (prepaid) or terminated
)

var (
	ErrSubscriptionOrderNotFound      = errors.New("subscription order not found")
	ErrSubscriptionOrderStatusInvalid = errors.New("subscription order status invalid")

	ErrSubscriptionCouponNotFound            = errors.New("coupon not found")
	ErrSubscriptionCouponDisabled            = errors.New("coupon is disabled")
	ErrSubscriptionCouponNotStarted          = errors.New("coupon is not yet active")
	ErrSubscriptionCouponExpired             = errors.New("coupon has expired")
	ErrSubscriptionCouponUsageCapReached     = errors.New("coupon usage cap reached")
	ErrSubscriptionCouponPerUserLimitReached = errors.New("coupon per-user limit reached")
	ErrSubscriptionCouponNewUserOnly         = errors.New("coupon is only for new subscribers")

	ErrSubscriptionCouponRedemptionNotReserved = errors.New("coupon redemption is not in reserved status")
	ErrSubscriptionCouponRedemptionNotIssued   = errors.New("coupon redemption is not in issued status")

	ErrInvalidSubscriptionStatusTransition = errors.New("invalid subscription status transition")
	ErrSubscriptionNotPrepaid              = errors.New("subscription is not prepaid")
)

// prepaidMonthSeconds is the canonical duration of one prepaid month used by
// ExtendPrepaidSubscriptionEndTime when stacking additional prepaid months on
// an existing prepaid subscription. It mirrors the month-multiplier semantics
// in calcPlanEndTime / CompleteSubscriptionOrderV2 (30 calendar days).
const prepaidMonthSeconds int64 = 30 * 24 * 3600

const (
	subscriptionPlanCacheNamespace     = "new-api:subscription_plan:v1"
	subscriptionPlanInfoCacheNamespace = "new-api:subscription_plan_info:v1"
)

var (
	subscriptionPlanCacheOnce     sync.Once
	subscriptionPlanInfoCacheOnce sync.Once

	subscriptionPlanCache     *cachex.HybridCache[SubscriptionPlan]
	subscriptionPlanInfoCache *cachex.HybridCache[SubscriptionPlanInfo]
)

func subscriptionPlanCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_CACHE_TTL", 300)
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	return time.Duration(ttlSeconds) * time.Second
}

func subscriptionPlanInfoCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_INFO_CACHE_TTL", 120)
	if ttlSeconds <= 0 {
		ttlSeconds = 120
	}
	return time.Duration(ttlSeconds) * time.Second
}

func subscriptionPlanCacheCapacity() int {
	capacity := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_CACHE_CAP", 5000)
	if capacity <= 0 {
		capacity = 5000
	}
	return capacity
}

func subscriptionPlanInfoCacheCapacity() int {
	capacity := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_INFO_CACHE_CAP", 10000)
	if capacity <= 0 {
		capacity = 10000
	}
	return capacity
}

func getSubscriptionPlanCache() *cachex.HybridCache[SubscriptionPlan] {
	subscriptionPlanCacheOnce.Do(func() {
		ttl := subscriptionPlanCacheTTL()
		subscriptionPlanCache = cachex.NewHybridCache[SubscriptionPlan](cachex.HybridCacheConfig[SubscriptionPlan]{
			Namespace: cachex.Namespace(subscriptionPlanCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[SubscriptionPlan]{},
			Memory: func() *hot.HotCache[string, SubscriptionPlan] {
				return hot.NewHotCache[string, SubscriptionPlan](hot.LRU, subscriptionPlanCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return subscriptionPlanCache
}

func getSubscriptionPlanInfoCache() *cachex.HybridCache[SubscriptionPlanInfo] {
	subscriptionPlanInfoCacheOnce.Do(func() {
		ttl := subscriptionPlanInfoCacheTTL()
		subscriptionPlanInfoCache = cachex.NewHybridCache[SubscriptionPlanInfo](cachex.HybridCacheConfig[SubscriptionPlanInfo]{
			Namespace: cachex.Namespace(subscriptionPlanInfoCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[SubscriptionPlanInfo]{},
			Memory: func() *hot.HotCache[string, SubscriptionPlanInfo] {
				return hot.NewHotCache[string, SubscriptionPlanInfo](hot.LRU, subscriptionPlanInfoCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return subscriptionPlanInfoCache
}

func subscriptionPlanCacheKey(id int) string {
	if id <= 0 {
		return ""
	}
	return strconv.Itoa(id)
}

func InvalidateSubscriptionPlanCache(planId int) {
	if planId <= 0 {
		return
	}
	cache := getSubscriptionPlanCache()
	_, _ = cache.DeleteMany([]string{subscriptionPlanCacheKey(planId)})
	infoCache := getSubscriptionPlanInfoCache()
	_ = infoCache.Purge()
}

// PurgeSubscriptionPlanCache empties the entire subscription plan cache
// (both the plan cache and the plan-info cache). Used by tests that swap
// the underlying database: the cache is a process-wide singleton, so
// without purging, a plan cached under one test's DB can mask a different
// plan row with the same ID created in a later test's DB.
func PurgeSubscriptionPlanCache() {
	_ = getSubscriptionPlanCache().Purge()
	_ = getSubscriptionPlanInfoCache().Purge()
}

// Subscription plan
type SubscriptionPlan struct {
	Id int `json:"id"`

	Title    string `json:"title" gorm:"type:varchar(128);not null"`
	Subtitle string `json:"subtitle" gorm:"type:varchar(255);default:''"`

	// Display money amount (follow existing code style: float64 for money)
	PriceAmount float64 `json:"price_amount" gorm:"type:decimal(10,6);not null;default:0"`
	Currency    string  `json:"currency" gorm:"type:varchar(8);not null;default:'USD'"`

	DurationUnit  string `json:"duration_unit" gorm:"type:varchar(16);not null;default:'month'"`
	DurationValue int    `json:"duration_value" gorm:"type:int;not null;default:1"`
	CustomSeconds int64  `json:"custom_seconds" gorm:"type:bigint;not null;default:0"`

	Enabled   bool `json:"enabled" gorm:"default:true"`
	SortOrder int  `json:"sort_order" gorm:"type:int;default:0"`

	AllowBalancePay *bool `json:"allow_balance_pay"`

	// Allow falling back to wallet balance after subscription quota is exhausted (empty = true)
	AllowWalletOverflow *bool `json:"allow_wallet_overflow"`

	StripePriceId         string `json:"stripe_price_id" gorm:"type:varchar(128);default:''"`
	CreemProductId        string `json:"creem_product_id" gorm:"type:varchar(128);default:''"`
	WaffoPancakeProductId string `json:"waffo_pancake_product_id" gorm:"type:varchar(128);default:''"`

	// Max purchases per user (0 = unlimited)
	MaxPurchasePerUser int `json:"max_purchase_per_user" gorm:"type:int;default:0"`

	// Upgrade user group after purchase (empty = no change)
	UpgradeGroup string `json:"upgrade_group" gorm:"type:varchar(64);default:''"`

	// Downgrade user group on expiry (empty = revert to the group held before purchase)
	DowngradeGroup string `json:"downgrade_group" gorm:"type:varchar(64);default:''"`

	// Total quota (amount in quota units, 0 = unlimited)
	TotalAmount int64 `json:"total_amount" gorm:"type:bigint;not null;default:0"`

	// Quota reset period for plan
	QuotaResetPeriod        string `json:"quota_reset_period" gorm:"type:varchar(16);default:'never'"`
	QuotaResetCustomSeconds int64  `json:"quota_reset_custom_seconds" gorm:"type:bigint;default:0"`

	// NovaPura subscription: dual-currency prices and Stripe product/price bindings.
	PriceAmountCNY   float64 `json:"price_amount_cny" gorm:"type:decimal(10,6);not null;default:0"`
	PriceAmountUSD   float64 `json:"price_amount_usd" gorm:"type:decimal(10,6);not null;default:0"`
	StripePriceIdCNY string  `json:"stripe_price_id_cny" gorm:"type:varchar(128);default:''"`
	StripePriceIdUSD string  `json:"stripe_price_id_usd" gorm:"type:varchar(128);default:''"`
	StripeProductId  string  `json:"stripe_product_id" gorm:"type:varchar(128);default:''"`
	AutoRenew        bool    `json:"auto_renew" gorm:"default:false"`
	PrepaidMonths    string  `json:"prepaid_months" gorm:"type:varchar(64);default:'1,3,6,12'"`
	RenewalPriceCNY  float64 `json:"renewal_price_cny" gorm:"type:decimal(10,6);not null;default:0"`
	RenewalPriceUSD  float64 `json:"renewal_price_usd" gorm:"type:decimal(10,6);not null;default:0"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (p *SubscriptionPlan) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *SubscriptionPlan) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

func (p *SubscriptionPlan) NormalizeDefaults() {
	if p.AllowBalancePay == nil {
		p.AllowBalancePay = common.GetPointer(true)
	}
	if p.AllowWalletOverflow == nil {
		p.AllowWalletOverflow = common.GetPointer(true)
	}
}

// SubscriptionPlanCoveredModel maps a plan to a canonical model ID that the
// plan covers (unlimited use at zero balance cost while the subscription is active).
type SubscriptionPlanCoveredModel struct {
	Id      int    `json:"id" gorm:"primaryKey"`
	PlanId  int    `json:"plan_id" gorm:"index;uniqueIndex:idx_plan_covered_model,priority:1"`
	ModelId string `json:"model_id" gorm:"type:varchar(128);uniqueIndex:idx_plan_covered_model,priority:2"` // canonical model ID
	Enabled bool   `json:"enabled" gorm:"default:false"`
	CreatedAt int64 `json:"created_at" gorm:"bigint"`
}

func (SubscriptionPlanCoveredModel) TableName() string { return "subscription_plan_covered_models" }

// SubscriptionCoupon is a discount code applicable to subscription purchases.
// Each coupon maps to a Stripe Coupon (StripeCouponId) so the discount is
// applied at Stripe Checkout, not just frontend-computed.
type SubscriptionCoupon struct {
	Id             int64  `json:"id" gorm:"primaryKey"`
	Code           string `json:"code" gorm:"type:varchar(64);uniqueIndex;not null"`
	Name           string `json:"name" gorm:"type:varchar(128);not null"`
	StripeCouponId string `json:"stripe_coupon_id" gorm:"type:varchar(128);not null"`
	PercentOff     int    `json:"percent_off" gorm:"not null;default:0"`     // 0-100
	DurationMonths int    `json:"duration_months" gorm:"not null;default:1"` // Stripe duration_in_months
	Enabled        bool   `json:"enabled" gorm:"default:false"`
	StartAt        int64  `json:"start_at" gorm:"default:0"`
	EndAt          int64  `json:"end_at" gorm:"default:0"`
	MaxRedemptions int    `json:"max_redemptions" gorm:"default:0"`  // 0 = unlimited
	PerUserLimit   int    `json:"per_user_limit" gorm:"default:0"`   // 0 = unlimited
	NewUserOnly    bool   `json:"new_user_only" gorm:"default:false"`
	TimesRedeemed  int    `json:"times_redeemed" gorm:"default:0"`
	CreatedAt      int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (SubscriptionCoupon) TableName() string { return "subscription_coupons" }

// SubscriptionCouponRedemption tracks per-order coupon reservation/issue/release/reversal.
// Lifecycle: reserved -> issued (on payment success) | released (on checkout expire/cancel);
// issued -> reversed (on refund).
const (
	CouponRedemptionStatusReserved = "reserved"
	CouponRedemptionStatusIssued   = "issued"
	CouponRedemptionStatusReleased = "released"
	CouponRedemptionStatusReversed = "reversed"
)

type SubscriptionCouponRedemption struct {
	Id             int64  `json:"id" gorm:"primaryKey"`
	OrderId        string `json:"order_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	CouponId       int64  `json:"coupon_id" gorm:"index;not null"`
	UserId         int    `json:"user_id" gorm:"index;not null"`
	PlanId         int    `json:"plan_id" gorm:"index;not null"`
	Status         string `json:"status" gorm:"type:varchar(16);index;not null"`
	PercentOff     int    `json:"percent_off" gorm:"not null;default:0"`
	OriginalAmount int64  `json:"original_amount" gorm:"type:bigint;not null;default:0"` // minor units
	DiscountAmount int64  `json:"discount_amount" gorm:"type:bigint;not null;default:0"` // minor units
	FinalAmount    int64  `json:"final_amount" gorm:"type:bigint;not null;default:0"`    // minor units
	Currency       string `json:"currency" gorm:"type:varchar(8);not null;default:'USD'"`
	CreatedAt      int64  `json:"created_at" gorm:"not null"`
	UpdatedAt      int64  `json:"updated_at" gorm:"not null"`
	IssuedAt       int64  `json:"issued_at" gorm:"not null;default:0"`
	ReleasedAt     int64  `json:"released_at" gorm:"not null;default:0"`
	ReversedAt     int64  `json:"reversed_at" gorm:"not null;default:0"`
}

func (SubscriptionCouponRedemption) TableName() string { return "subscription_coupon_redemptions" }

// Subscription order (payment -> webhook -> create UserSubscription)
type SubscriptionOrder struct {
	Id     int     `json:"id"`
	UserId int     `json:"user_id" gorm:"index"`
	PlanId int     `json:"plan_id" gorm:"index"`
	Money  float64 `json:"money"`

	TradeNo         string `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	Status          string `json:"status"`
	CreateTime      int64  `json:"create_time"`
	CompleteTime    int64  `json:"complete_time"`

	ProviderPayload string `json:"provider_payload" gorm:"type:text"`

	// NovaPura subscription: Stripe checkout / coupon settlement fields.
	StripeSubscriptionId    string `json:"stripe_subscription_id" gorm:"type:varchar(128);default:''"`
	StripeCheckoutSessionId string `json:"stripe_checkout_session_id" gorm:"type:varchar(128);default:''"`
	Currency                string `json:"currency" gorm:"type:varchar(8);default:''"`
	PrepaidMonths           int    `json:"prepaid_months" gorm:"default:0"`
	CouponRedemptionId      *int   `json:"coupon_redemption_id" gorm:"index"`
	OriginalAmount          int64  `json:"original_amount" gorm:"type:bigint;default:0"`
	DiscountAmount          int64  `json:"discount_amount" gorm:"type:bigint;default:0"`
	FinalAmount             int64  `json:"final_amount" gorm:"type:bigint;default:0"`

	// NovaPura Phase 10: Mode ("auto_renew" or "prepaid") and the Stripe
	// Checkout Session URL captured at creation time. The URL is reused by
	// the recent-pending-order dedup path so a rapid double-click returns the
	// existing session instead of creating a second Stripe session.
	Mode         string `json:"mode" gorm:"type:varchar(16);default:''"`
	CheckoutUrl  string `json:"checkout_url" gorm:"type:text;default:''"`
}

func (o *SubscriptionOrder) Insert() error {
	if o.CreateTime == 0 {
		o.CreateTime = common.GetTimestamp()
	}
	return DB.Create(o).Error
}

func (o *SubscriptionOrder) Update() error {
	return DB.Save(o).Error
}

func GetSubscriptionOrderByTradeNo(tradeNo string) *SubscriptionOrder {
	if tradeNo == "" {
		return nil
	}
	var order SubscriptionOrder
	if err := DB.Where("trade_no = ?", tradeNo).First(&order).Error; err != nil {
		return nil
	}
	return &order
}

// User subscription instance
type UserSubscription struct {
	Id     int `json:"id"`
	UserId int `json:"user_id" gorm:"index;index:idx_user_sub_active,priority:1"`
	PlanId int `json:"plan_id" gorm:"index"`

	AmountTotal int64 `json:"amount_total" gorm:"type:bigint;not null;default:0"`
	AmountUsed  int64 `json:"amount_used" gorm:"type:bigint;not null;default:0"`

	StartTime int64  `json:"start_time" gorm:"bigint"`
	EndTime   int64  `json:"end_time" gorm:"bigint;index;index:idx_user_sub_active,priority:3"`
	Status    string `json:"status" gorm:"type:varchar(32);index;index:idx_user_sub_active,priority:2"` // active/expired/cancelled

	Source string `json:"source" gorm:"type:varchar(32);default:'order'"` // order/admin

	LastResetTime int64 `json:"last_reset_time" gorm:"type:bigint;default:0"`
	NextResetTime int64 `json:"next_reset_time" gorm:"type:bigint;default:0;index"`

	UpgradeGroup  string `json:"upgrade_group" gorm:"type:varchar(64);default:''"`
	PrevUserGroup string `json:"prev_user_group" gorm:"type:varchar(64);default:''"`

	// Downgrade target group on expiry (snapshot from plan; empty = revert to PrevUserGroup)
	DowngradeGroup string `json:"downgrade_group" gorm:"type:varchar(64);default:''"`

	// Whether wallet fallback is allowed after this subscription's quota is exhausted (snapshot from plan)
	AllowWalletOverflow bool `json:"allow_wallet_overflow"`

	// NovaPura subscription: Stripe linkage and coupon settlement fields.
	StripeSubscriptionId string `json:"stripe_subscription_id" gorm:"type:varchar(128);default:'';index"`
	StripeCustomerId     string `json:"stripe_customer_id" gorm:"type:varchar(128);default:''"`
	BillingCycleAnchor   int64  `json:"billing_cycle_anchor" gorm:"bigint;default:0"`
	CancelAtPeriodEnd    bool   `json:"cancel_at_period_end" gorm:"default:false"`
	CouponId             *int   `json:"coupon_id" gorm:"index"`
	CouponRedemptionId   *int   `json:"coupon_redemption_id" gorm:"index"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (s *UserSubscription) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

func (s *UserSubscription) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = common.GetTimestamp()
	return nil
}

type SubscriptionSummary struct {
	Subscription *UserSubscription `json:"subscription"`
}

type SubscriptionResetResult struct {
	PlanId           int    `json:"plan_id"`
	MatchedCount     int    `json:"matched_count"`
	ResetCount       int    `json:"reset_count"`
	UserCount        int    `json:"user_count"`
	AdvanceResetTime bool   `json:"advance_reset_time"`
	PlanTitle        string `json:"-"`
	AffectedUserIds  []int  `json:"-"`
}

func calcPlanEndTime(start time.Time, plan *SubscriptionPlan) (int64, error) {
	if plan == nil {
		return 0, errors.New("plan is nil")
	}
	if plan.DurationValue <= 0 && plan.DurationUnit != SubscriptionDurationCustom {
		return 0, errors.New("duration_value must be > 0")
	}
	switch plan.DurationUnit {
	case SubscriptionDurationYear:
		return start.AddDate(plan.DurationValue, 0, 0).Unix(), nil
	case SubscriptionDurationMonth:
		return start.AddDate(0, plan.DurationValue, 0).Unix(), nil
	case SubscriptionDurationDay:
		return start.Add(time.Duration(plan.DurationValue) * 24 * time.Hour).Unix(), nil
	case SubscriptionDurationHour:
		return start.Add(time.Duration(plan.DurationValue) * time.Hour).Unix(), nil
	case SubscriptionDurationCustom:
		if plan.CustomSeconds <= 0 {
			return 0, errors.New("custom_seconds must be > 0")
		}
		return start.Add(time.Duration(plan.CustomSeconds) * time.Second).Unix(), nil
	default:
		return 0, fmt.Errorf("invalid duration_unit: %s", plan.DurationUnit)
	}
}

func NormalizeResetPeriod(period string) string {
	switch strings.TrimSpace(period) {
	case SubscriptionResetDaily, SubscriptionResetWeekly, SubscriptionResetMonthly, SubscriptionResetCustom:
		return strings.TrimSpace(period)
	default:
		return SubscriptionResetNever
	}
}

func calcNextResetTime(base time.Time, plan *SubscriptionPlan, endUnix int64) int64 {
	if plan == nil {
		return 0
	}
	period := NormalizeResetPeriod(plan.QuotaResetPeriod)
	if period == SubscriptionResetNever {
		return 0
	}
	var next time.Time
	switch period {
	case SubscriptionResetDaily:
		next = time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).
			AddDate(0, 0, 1)
	case SubscriptionResetWeekly:
		// Align to next Monday 00:00
		weekday := int(base.Weekday()) // Sunday=0
		// Convert to Monday=1..Sunday=7
		if weekday == 0 {
			weekday = 7
		}
		daysUntil := 8 - weekday
		next = time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).
			AddDate(0, 0, daysUntil)
	case SubscriptionResetMonthly:
		// Align to first day of next month 00:00
		next = time.Date(base.Year(), base.Month(), 1, 0, 0, 0, 0, base.Location()).
			AddDate(0, 1, 0)
	case SubscriptionResetCustom:
		if plan.QuotaResetCustomSeconds <= 0 {
			return 0
		}
		next = base.Add(time.Duration(plan.QuotaResetCustomSeconds) * time.Second)
	default:
		return 0
	}
	if endUnix > 0 && next.Unix() > endUnix {
		return 0
	}
	return next.Unix()
}

func GetSubscriptionPlanById(id int) (*SubscriptionPlan, error) {
	return getSubscriptionPlanByIdTx(nil, id)
}

func getSubscriptionPlanByIdTx(tx *gorm.DB, id int) (*SubscriptionPlan, error) {
	if id <= 0 {
		return nil, errors.New("invalid plan id")
	}
	key := subscriptionPlanCacheKey(id)
	if key != "" {
		if cached, found, err := getSubscriptionPlanCache().Get(key); err == nil && found {
			cached.NormalizeDefaults()
			return &cached, nil
		}
	}
	var plan SubscriptionPlan
	query := DB
	if tx != nil {
		query = tx
	}
	if err := query.Where("id = ?", id).First(&plan).Error; err != nil {
		return nil, err
	}
	plan.NormalizeDefaults()
	_ = getSubscriptionPlanCache().SetWithTTL(key, plan, subscriptionPlanCacheTTL())
	return &plan, nil
}

func CountUserSubscriptionsByPlan(userId int, planId int) (int64, error) {
	if userId <= 0 || planId <= 0 {
		return 0, errors.New("invalid userId or planId")
	}
	var count int64
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND plan_id = ?", userId, planId).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func getUserGroupByIdTx(tx *gorm.DB, userId int) (string, error) {
	if userId <= 0 {
		return "", errors.New("invalid userId")
	}
	if tx == nil {
		tx = DB
	}
	var group string
	if err := tx.Model(&User{}).Where("id = ?", userId).Select(commonGroupCol).Find(&group).Error; err != nil {
		return "", err
	}
	return group, nil
}

func downgradeUserGroupForSubscriptionTx(tx *gorm.DB, sub *UserSubscription, now int64) (string, error) {
	if tx == nil || sub == nil {
		return "", errors.New("invalid downgrade args")
	}
	downgradeGroup := strings.TrimSpace(sub.DowngradeGroup)
	upgradeGroup := strings.TrimSpace(sub.UpgradeGroup)
	// Nothing to do if neither an explicit downgrade target nor an upgrade snapshot exists.
	if downgradeGroup == "" && upgradeGroup == "" {
		return "", nil
	}
	currentGroup, err := getUserGroupByIdTx(tx, sub.UserId)
	if err != nil {
		return "", err
	}
	// If another active upgraded subscription exists, keep the current group.
	var activeSub UserSubscription
	activeQuery := tx.Where("user_id = ? AND status = ? AND end_time > ? AND id <> ? AND upgrade_group <> ''",
		sub.UserId, "active", now, sub.Id).
		Order("end_time desc, id desc").
		Limit(1).
		Find(&activeSub)
	if activeQuery.Error == nil && activeQuery.RowsAffected > 0 {
		return "", nil
	}
	// Determine the downgrade target: an explicit downgrade group takes precedence,
	// otherwise revert to the group held before purchase (legacy behavior).
	target := downgradeGroup
	if target == "" {
		// Legacy behavior: only revert when the subscription actually elevated the user.
		if currentGroup != upgradeGroup {
			return "", nil
		}
		target = strings.TrimSpace(sub.PrevUserGroup)
	}
	if target == "" || target == currentGroup {
		return "", nil
	}
	if err := tx.Model(&User{}).Where("id = ?", sub.UserId).
		Update("group", target).Error; err != nil {
		return "", err
	}
	return target, nil
}

func CreateUserSubscriptionFromPlanTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, source string) (*UserSubscription, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	if plan == nil || plan.Id == 0 {
		return nil, errors.New("invalid plan")
	}
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	if plan.MaxPurchasePerUser > 0 {
		var count int64
		if err := tx.Model(&UserSubscription{}).
			Where("user_id = ? AND plan_id = ?", userId, plan.Id).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			return nil, errors.New("已达到该套餐购买上限")
		}
	}
	nowUnix := GetDBTimestamp()
	now := time.Unix(nowUnix, 0)
	endUnix, err := calcPlanEndTime(now, plan)
	if err != nil {
		return nil, err
	}
	resetBase := now
	nextReset := calcNextResetTime(resetBase, plan, endUnix)
	lastReset := int64(0)
	if nextReset > 0 {
		lastReset = now.Unix()
	}
	upgradeGroup := strings.TrimSpace(plan.UpgradeGroup)
	prevGroup := ""
	if upgradeGroup != "" {
		currentGroup, err := getUserGroupByIdTx(tx, userId)
		if err != nil {
			return nil, err
		}
		if currentGroup != upgradeGroup {
			prevGroup = currentGroup
			if err := tx.Model(&User{}).Where("id = ?", userId).
				Update("group", upgradeGroup).Error; err != nil {
				return nil, err
			}
		}
	}
	allowWalletOverflow := true
	if plan.AllowWalletOverflow != nil {
		allowWalletOverflow = *plan.AllowWalletOverflow
	}
	// Legacy creation path always produces an auto-renew-style "active" sub.
	// The NovaPura v2 prepaid path (CompleteSubscriptionOrderV2) sets
	// prepaid_active instead; Phase 3 does not change creation semantics.
	sub := &UserSubscription{
		UserId:              userId,
		PlanId:              plan.Id,
		AmountTotal:         plan.TotalAmount,
		AmountUsed:          0,
		StartTime:           now.Unix(),
		EndTime:             endUnix,
		Status:              SubscriptionStatusActive,
		Source:              source,
		LastResetTime:       lastReset,
		NextResetTime:       nextReset,
		UpgradeGroup:        upgradeGroup,
		PrevUserGroup:       prevGroup,
		DowngradeGroup:      strings.TrimSpace(plan.DowngradeGroup),
		AllowWalletOverflow: allowWalletOverflow,
		CreatedAt:           common.GetTimestamp(),
		UpdatedAt:           common.GetTimestamp(),
	}
	if err := tx.Create(sub).Error; err != nil {
		return nil, err
	}
	return sub, nil
}

// Complete a subscription order (idempotent). Creates a UserSubscription snapshot from the plan.
// expectedPaymentProvider guards against cross-gateway callback attacks (empty skips the check).
// actualPaymentMethod updates the order's PaymentMethod to reflect the real payment type used (empty skips update).
func CompleteSubscriptionOrder(tradeNo string, providerPayload string, expectedPaymentProvider string, actualPaymentMethod string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}
	var logUserId int
	var logPlanTitle string
	var logMoney float64
	var logPaymentMethod string
	var upgradeGroup string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var order SubscriptionOrder
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrSubscriptionOrderNotFound
		}
		if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if order.Status == common.TopUpStatusSuccess {
			return nil
		}
		if order.Status != common.TopUpStatusPending {
			return ErrSubscriptionOrderStatusInvalid
		}
		plan, err := GetSubscriptionPlanById(order.PlanId)
		if err != nil {
			return err
		}
		if !plan.Enabled {
			// still allow completion for already purchased orders
		}
		upgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
		_, err = CreateUserSubscriptionFromPlanTx(tx, order.UserId, plan, "order")
		if err != nil {
			return err
		}
		if err := upsertSubscriptionTopUpTx(tx, &order); err != nil {
			return err
		}
		order.Status = common.TopUpStatusSuccess
		order.CompleteTime = common.GetTimestamp()
		if providerPayload != "" {
			order.ProviderPayload = providerPayload
		}
		if actualPaymentMethod != "" && order.PaymentMethod != actualPaymentMethod {
			order.PaymentMethod = actualPaymentMethod
		}
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		logUserId = order.UserId
		logPlanTitle = plan.Title
		logMoney = order.Money
		logPaymentMethod = order.PaymentMethod
		return nil
	})
	if err != nil {
		return err
	}
	if upgradeGroup != "" && logUserId > 0 {
		_ = UpdateUserGroupCache(logUserId, upgradeGroup)
	}
	if logUserId > 0 {
		InvalidateUserSubscriptionCoverageCache(logUserId)
		msg := fmt.Sprintf("订阅购买成功，套餐: %s，支付金额: %.2f，支付方式: %s", logPlanTitle, logMoney, logPaymentMethod)
		RecordLog(logUserId, LogTypeTopup, msg)
	}
	return nil
}

func upsertSubscriptionTopUpTx(tx *gorm.DB, order *SubscriptionOrder) error {
	if tx == nil || order == nil {
		return errors.New("invalid subscription order")
	}
	now := common.GetTimestamp()
	var topup TopUp
	if err := tx.Where("trade_no = ?", order.TradeNo).First(&topup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			topup = TopUp{
				UserId:        order.UserId,
				Amount:        0,
				Money:         order.Money,
				TradeNo:       order.TradeNo,
				PaymentMethod: order.PaymentMethod,
				CreateTime:    order.CreateTime,
				CompleteTime:  now,
				Status:        common.TopUpStatusSuccess,
			}
			return tx.Create(&topup).Error
		}
		return err
	}
	topup.Money = order.Money
	if topup.PaymentMethod == "" {
		topup.PaymentMethod = order.PaymentMethod
	} else if topup.PaymentMethod != order.PaymentMethod {
		return ErrPaymentMethodMismatch
	}
	if topup.CreateTime == 0 {
		topup.CreateTime = order.CreateTime
	}
	topup.CompleteTime = now
	topup.Status = common.TopUpStatusSuccess
	return tx.Save(&topup).Error
}

func ExpireSubscriptionOrder(tradeNo string, expectedPaymentProvider string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var order SubscriptionOrder
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrSubscriptionOrderNotFound
		}
		if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if order.Status != common.TopUpStatusPending {
			return nil
		}
		order.Status = common.TopUpStatusExpired
		order.CompleteTime = common.GetTimestamp()
		return tx.Save(&order).Error
	})
}

// Admin bind (no payment). Creates a UserSubscription from a plan.
func AdminBindSubscription(userId int, planId int, sourceNote string) (string, error) {
	if userId <= 0 || planId <= 0 {
		return "", errors.New("invalid userId or planId")
	}
	plan, err := GetSubscriptionPlanById(planId)
	if err != nil {
		return "", err
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		_, err := CreateUserSubscriptionFromPlanTx(tx, userId, plan, "admin")
		return err
	})
	if err != nil {
		return "", err
	}
	InvalidateUserSubscriptionCoverageCache(userId)
	if strings.TrimSpace(plan.UpgradeGroup) != "" {
		_ = UpdateUserGroupCache(userId, plan.UpgradeGroup)
		return fmt.Sprintf("用户分组将升级到 %s", plan.UpgradeGroup), nil
	}
	return "", nil
}

func calcSubscriptionBalanceQuota(priceAmount float64) (int, error) {
	if priceAmount <= 0 {
		return 0, nil
	}
	if common.QuotaPerUnit <= 0 {
		return 0, errors.New("额度单位配置错误")
	}
	quota := decimal.NewFromFloat(priceAmount).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Ceil().
		IntPart()
	return int(quota), nil
}

// PurchaseSubscriptionWithBalance creates a subscription by deducting the user's wallet quota.
func PurchaseSubscriptionWithBalance(userId int, planId int) error {
	if userId <= 0 || planId <= 0 {
		return errors.New("invalid userId or planId")
	}

	var logPlanTitle string
	var logMoney float64
	var chargedQuota int
	var upgradeGroup string
	err := DB.Transaction(func(tx *gorm.DB) error {
		plan, err := getSubscriptionPlanByIdTx(tx, planId)
		if err != nil {
			return err
		}
		if !plan.Enabled {
			return errors.New("套餐未启用")
		}
		if plan.PriceAmount < 0 {
			return errors.New("套餐价格不能为负数")
		}
		if plan.AllowBalancePay != nil && !*plan.AllowBalancePay {
			return errors.New("该套餐不允许使用余额兑换")
		}

		requiredQuota, err := calcSubscriptionBalanceQuota(plan.PriceAmount)
		if err != nil {
			return err
		}

		var user User
		if err := lockForUpdate(tx).Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}
		if requiredQuota > 0 && user.Quota < requiredQuota {
			return errors.New("余额不足")
		}
		if requiredQuota > 0 {
			if err := tx.Model(&User{}).Where("id = ?", userId).
				Update("quota", gorm.Expr("quota - ?", requiredQuota)).Error; err != nil {
				return err
			}
		}

		if _, err := CreateUserSubscriptionFromPlanTx(tx, userId, plan, PaymentMethodBalance); err != nil {
			return err
		}

		now := common.GetTimestamp()
		tradeNo := fmt.Sprintf("SUBBALUSR%dNO%s%d", userId, common.GetRandomString(6), time.Now().UnixNano())
		order := &SubscriptionOrder{
			UserId:          userId,
			PlanId:          plan.Id,
			Money:           plan.PriceAmount,
			TradeNo:         tradeNo,
			PaymentMethod:   PaymentMethodBalance,
			PaymentProvider: PaymentProviderBalance,
			Status:          common.TopUpStatusSuccess,
			CreateTime:      now,
			CompleteTime:    now,
			ProviderPayload: fmt.Sprintf("charged_quota=%d", requiredQuota),
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		logPlanTitle = plan.Title
		logMoney = plan.PriceAmount
		chargedQuota = requiredQuota
		upgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
		return nil
	})
	if err != nil {
		return err
	}

	if chargedQuota > 0 {
		if err := cacheDecrUserQuota(userId, int64(chargedQuota)); err != nil {
			common.SysLog("failed to decrease user quota cache after subscription balance purchase: " + err.Error())
		}
	}
	InvalidateUserSubscriptionCoverageCache(userId)
	if upgradeGroup != "" {
		_ = UpdateUserGroupCache(userId, upgradeGroup)
	}
	msg := fmt.Sprintf("使用余额购买订阅成功，套餐: %s，支付金额: %.2f，扣除额度: %d", logPlanTitle, logMoney, chargedQuota)
	RecordLog(userId, LogTypeTopup, msg)
	return nil
}

// GetAllActiveUserSubscriptions returns all active subscriptions for a user.
func GetAllActiveUserSubscriptions(userId int) ([]SubscriptionSummary, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var subs []UserSubscription
	err := DB.Where("user_id = ? AND status = ? AND end_time > ?", userId, "active", now).
		Order("end_time desc, id desc").
		Find(&subs).Error
	if err != nil {
		return nil, err
	}
	return buildSubscriptionSummaries(subs), nil
}

// HasActiveUserSubscription returns whether the user has any active subscription.
// This is a lightweight existence check to avoid heavy pre-consume transactions.
func HasActiveUserSubscription(userId int) (bool, error) {
	if userId <= 0 {
		return false, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var count int64
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND status = ? AND end_time > ?", userId, "active", now).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// UserActiveSubscriptionsAllowWalletOverflow returns whether wallet balance may be used
// after the user's subscription quota is exhausted. A single active subscription that
// disallows wallet overflow (allow_wallet_overflow = false) blocks the fallback.
func UserActiveSubscriptionsAllowWalletOverflow(userId int) (bool, error) {
	if userId <= 0 {
		return false, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var strictCount int64
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND status = ? AND end_time > ? AND allow_wallet_overflow = ?",
			userId, "active", now, false).
		Count(&strictCount).Error; err != nil {
		return false, err
	}
	return strictCount == 0, nil
}

// GetAllUserSubscriptions returns all subscriptions (active and expired) for a user.
func GetAllUserSubscriptions(userId int) ([]SubscriptionSummary, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	var subs []UserSubscription
	err := DB.Where("user_id = ?", userId).
		Order("end_time desc, id desc").
		Find(&subs).Error
	if err != nil {
		return nil, err
	}
	return buildSubscriptionSummaries(subs), nil
}

func buildSubscriptionSummaries(subs []UserSubscription) []SubscriptionSummary {
	if len(subs) == 0 {
		return []SubscriptionSummary{}
	}
	result := make([]SubscriptionSummary, 0, len(subs))
	for _, sub := range subs {
		subCopy := sub
		result = append(result, SubscriptionSummary{
			Subscription: &subCopy,
		})
	}
	return result
}

// AdminInvalidateUserSubscription marks a user subscription as cancelled and ends it immediately.
func AdminInvalidateUserSubscription(userSubscriptionId int) (string, error) {
	if userSubscriptionId <= 0 {
		return "", errors.New("invalid userSubscriptionId")
	}
	now := common.GetTimestamp()
	cacheGroup := ""
	downgradeGroup := ""
	var userId int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).
			Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
			return err
		}
		userId = sub.UserId
		// NOTE: this writes the British spelling "cancelled" for back-compat
		// with existing rows/admin tooling. New code should use the American
		// SubscriptionStatusCanceled ("canceled") constant instead. We do NOT
		// migrate existing rows here (Phase 3 is back-compat only).
		if err := tx.Model(&sub).Updates(map[string]interface{}{
			"status":     "cancelled",
			"end_time":   now,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		target, err := downgradeUserGroupForSubscriptionTx(tx, &sub, now)
		if err != nil {
			return err
		}
		if target != "" {
			cacheGroup = target
			downgradeGroup = target
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if cacheGroup != "" && userId > 0 {
		_ = UpdateUserGroupCache(userId, cacheGroup)
	}
	if userId > 0 {
		InvalidateUserSubscriptionCoverageCache(userId)
	}
	if downgradeGroup != "" {
		return fmt.Sprintf("用户分组将回退到 %s", downgradeGroup), nil
	}
	return "", nil
}

// AdminDeleteUserSubscription hard-deletes a user subscription.
func AdminDeleteUserSubscription(userSubscriptionId int) (string, error) {
	if userSubscriptionId <= 0 {
		return "", errors.New("invalid userSubscriptionId")
	}
	now := common.GetTimestamp()
	cacheGroup := ""
	downgradeGroup := ""
	var userId int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).
			Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
			return err
		}
		userId = sub.UserId
		target, err := downgradeUserGroupForSubscriptionTx(tx, &sub, now)
		if err != nil {
			return err
		}
		if target != "" {
			cacheGroup = target
			downgradeGroup = target
		}
		if err := tx.Where("id = ?", userSubscriptionId).Delete(&UserSubscription{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if cacheGroup != "" && userId > 0 {
		_ = UpdateUserGroupCache(userId, cacheGroup)
	}
	if userId > 0 {
		InvalidateUserSubscriptionCoverageCache(userId)
	}
	if downgradeGroup != "" {
		return fmt.Sprintf("用户分组将回退到 %s", downgradeGroup), nil
	}
	return "", nil
}

func resetUserSubscriptionTx(tx *gorm.DB, sub *UserSubscription, plan *SubscriptionPlan, now int64, advanceResetTime bool) error {
	if tx == nil || sub == nil || plan == nil {
		return errors.New("invalid reset args")
	}
	sub.AmountUsed = 0
	if advanceResetTime {
		nextReset := calcNextResetTime(time.Unix(now, 0), plan, sub.EndTime)
		sub.NextResetTime = nextReset
		if nextReset > 0 {
			sub.LastResetTime = now
		} else {
			sub.LastResetTime = 0
		}
	}
	return tx.Save(sub).Error
}

func buildSubscriptionResetResult(plan *SubscriptionPlan, subs []UserSubscription, advanceResetTime bool) *SubscriptionResetResult {
	userIds := make([]int, 0, len(subs))
	seenUsers := make(map[int]struct{}, len(subs))
	for _, sub := range subs {
		if _, ok := seenUsers[sub.UserId]; ok {
			continue
		}
		seenUsers[sub.UserId] = struct{}{}
		userIds = append(userIds, sub.UserId)
	}
	return &SubscriptionResetResult{
		PlanId:           plan.Id,
		MatchedCount:     len(subs),
		ResetCount:       len(subs),
		UserCount:        len(userIds),
		AdvanceResetTime: advanceResetTime,
		PlanTitle:        plan.Title,
		AffectedUserIds:  userIds,
	}
}

func adminResetUserSubscriptionsByPlanTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, now int64, advanceResetTime bool) (*SubscriptionResetResult, error) {
	if tx == nil || plan == nil {
		return nil, errors.New("invalid reset args")
	}
	var subs []UserSubscription
	if err := lockForUpdate(tx).
		Where("user_id = ? AND plan_id = ? AND status = ? AND end_time > ?", userId, plan.Id, "active", now).
		Order("end_time asc, id asc").
		Find(&subs).Error; err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, errors.New("该用户没有有效的此套餐订阅")
	}
	for i := range subs {
		if err := resetUserSubscriptionTx(tx, &subs[i], plan, now, advanceResetTime); err != nil {
			return nil, err
		}
	}
	return buildSubscriptionResetResult(plan, subs, advanceResetTime), nil
}

func adminResetPlanSubscriptionsTx(tx *gorm.DB, plan *SubscriptionPlan, now int64, advanceResetTime bool) (*SubscriptionResetResult, error) {
	if tx == nil || plan == nil {
		return nil, errors.New("invalid reset args")
	}
	var subs []UserSubscription
	if err := lockForUpdate(tx).
		Where("plan_id = ? AND status = ? AND end_time > ?", plan.Id, "active", now).
		Order("user_id asc, end_time asc, id asc").
		Find(&subs).Error; err != nil {
		return nil, err
	}
	for i := range subs {
		if err := resetUserSubscriptionTx(tx, &subs[i], plan, now, advanceResetTime); err != nil {
			return nil, err
		}
	}
	return buildSubscriptionResetResult(plan, subs, advanceResetTime), nil
}

func AdminResetUserSubscriptionsByPlan(userId int, planId int, advanceResetTime bool) (*SubscriptionResetResult, error) {
	if userId <= 0 || planId <= 0 {
		return nil, errors.New("invalid userId or planId")
	}
	var result *SubscriptionResetResult
	now := GetDBTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		plan, err := getSubscriptionPlanByIdTx(tx, planId)
		if err != nil {
			return err
		}
		result, err = adminResetUserSubscriptionsByPlanTx(tx, userId, plan, now, advanceResetTime)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func AdminResetPlanSubscriptions(planId int, advanceResetTime bool) (*SubscriptionResetResult, error) {
	if planId <= 0 {
		return nil, errors.New("invalid planId")
	}
	var result *SubscriptionResetResult
	now := GetDBTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		plan, err := getSubscriptionPlanByIdTx(tx, planId)
		if err != nil {
			return err
		}
		result, err = adminResetPlanSubscriptionsTx(tx, plan, now, advanceResetTime)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type SubscriptionPreConsumeResult struct {
	UserSubscriptionId int
	PreConsumed        int64
	AmountTotal        int64
	AmountUsedBefore   int64
	AmountUsedAfter    int64
}

// ExpireDueSubscriptions marks expired subscriptions and handles group downgrade.
func ExpireDueSubscriptions(limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	now := GetDBTimestamp()
	var subs []UserSubscription
	if err := DB.Where("status = ? AND end_time > 0 AND end_time <= ?", SubscriptionStatusActive, now).
		Order("end_time asc, id asc").
		Limit(limit).
		Find(&subs).Error; err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}
	expiredCount := 0
	userIds := make(map[int]struct{}, len(subs))
	for _, sub := range subs {
		if sub.UserId > 0 {
			userIds[sub.UserId] = struct{}{}
		}
	}
	for userId := range userIds {
		cacheGroup := ""
		err := DB.Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&UserSubscription{}).
				Where("user_id = ? AND status = ? AND end_time > 0 AND end_time <= ?", userId, SubscriptionStatusActive, now).
				Updates(map[string]interface{}{
					"status":     SubscriptionStatusExpired,
					"updated_at": common.GetTimestamp(),
				})
			if res.Error != nil {
				return res.Error
			}
			expiredCount += int(res.RowsAffected)

			// If there's an active upgraded subscription, keep current group.
			var activeSub UserSubscription
			activeQuery := tx.Where("user_id = ? AND status = ? AND end_time > ? AND upgrade_group <> ''",
				userId, SubscriptionStatusActive, now).
				Order("end_time desc, id desc").
				Limit(1).
				Find(&activeSub)
			if activeQuery.Error == nil && activeQuery.RowsAffected > 0 {
				return nil
			}

			// Find the most recently expired subscription that defines a group transition
			// (an explicit downgrade target or an upgrade snapshot to revert).
			var lastExpired UserSubscription
			expiredQuery := tx.Where("user_id = ? AND status = ? AND (downgrade_group <> '' OR upgrade_group <> '')",
				userId, SubscriptionStatusExpired).
				Order("end_time desc, id desc").
				Limit(1).
				Find(&lastExpired)
			if expiredQuery.Error != nil || expiredQuery.RowsAffected == 0 {
				return nil
			}
			currentGroup, err := getUserGroupByIdTx(tx, userId)
			if err != nil {
				return err
			}
			// An explicit downgrade group takes precedence; otherwise revert to the
			// group held before purchase (legacy behavior, only when the subscription
			// actually elevated the user).
			target := strings.TrimSpace(lastExpired.DowngradeGroup)
			if target == "" {
				upgradeGroup := strings.TrimSpace(lastExpired.UpgradeGroup)
				prevGroup := strings.TrimSpace(lastExpired.PrevUserGroup)
				if upgradeGroup == "" || prevGroup == "" {
					return nil
				}
				if currentGroup != upgradeGroup {
					return nil
				}
				target = prevGroup
			}
			if target == "" || target == currentGroup {
				return nil
			}
			if err := tx.Model(&User{}).Where("id = ?", userId).
				Update("group", target).Error; err != nil {
				return err
			}
			cacheGroup = target
			return nil
		})
		if err != nil {
			return expiredCount, err
		}
		if cacheGroup != "" {
			_ = UpdateUserGroupCache(userId, cacheGroup)
		}
		// Coverage cache may have cached "covered" results that are now stale
		// because the user's active subscription set shrank.
		InvalidateUserSubscriptionCoverageCache(userId)
	}
	return expiredCount, nil
}

// SubscriptionPreConsumeRecord stores idempotent pre-consume operations per request.
type SubscriptionPreConsumeRecord struct {
	Id                 int    `json:"id"`
	RequestId          string `json:"request_id" gorm:"type:varchar(64);uniqueIndex"`
	UserId             int    `json:"user_id" gorm:"index"`
	UserSubscriptionId int    `json:"user_subscription_id" gorm:"index"`
	PreConsumed        int64  `json:"pre_consumed" gorm:"type:bigint;not null;default:0"`
	Status             string `json:"status" gorm:"type:varchar(32);index"` // consumed/refunded
	CreatedAt          int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt          int64  `json:"updated_at" gorm:"bigint;index"`
}

func (r *SubscriptionPreConsumeRecord) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *SubscriptionPreConsumeRecord) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func maybeResetUserSubscriptionWithPlanTx(tx *gorm.DB, sub *UserSubscription, plan *SubscriptionPlan, now int64) error {
	if tx == nil || sub == nil || plan == nil {
		return errors.New("invalid reset args")
	}
	if sub.NextResetTime > 0 && sub.NextResetTime > now {
		return nil
	}
	if NormalizeResetPeriod(plan.QuotaResetPeriod) == SubscriptionResetNever {
		return nil
	}
	baseUnix := sub.LastResetTime
	if baseUnix <= 0 {
		baseUnix = sub.StartTime
	}
	base := time.Unix(baseUnix, 0)
	next := calcNextResetTime(base, plan, sub.EndTime)
	advanced := false
	for next > 0 && next <= now {
		advanced = true
		base = time.Unix(next, 0)
		next = calcNextResetTime(base, plan, sub.EndTime)
	}
	if !advanced {
		if sub.NextResetTime == 0 && next > 0 {
			sub.NextResetTime = next
			sub.LastResetTime = base.Unix()
			return tx.Save(sub).Error
		}
		return nil
	}
	sub.AmountUsed = 0
	sub.LastResetTime = base.Unix()
	sub.NextResetTime = next
	return tx.Save(sub).Error
}

// IsValidSubscriptionStatus returns true if s is a recognized subscription status value.
func IsValidSubscriptionStatus(s string) bool {
	switch s {
	case SubscriptionStatusActive,
		SubscriptionStatusPrepaidActive,
		SubscriptionStatusCanceling,
		SubscriptionStatusCanceled,
		SubscriptionStatusPastDue,
		SubscriptionStatusPaymentFailed,
		SubscriptionStatusExpired:
		return true
	}
	return false
}

// allowedSubscriptionTransitions codifies the legal status state machine.
// Terminal states (canceled, expired) have no outgoing transitions. This map
// is the single source of truth for TransitionSubscriptionStatus; callers must
// not bypass it with direct Status writes for lifecycle transitions.
//
// NOTE: group-downgrade logic for terminal transitions (Canceled/Expired) is
// handled by downgradeUserGroupForSubscriptionTx / CancelUserSubscriptionFromStripe
// / ExpireDueSubscriptions — TransitionSubscriptionStatus ONLY updates the
// Status field. The caller decides when (and whether) to downgrade.
var allowedSubscriptionTransitions = map[string][]string{
	SubscriptionStatusActive: {
		SubscriptionStatusCanceling,
		SubscriptionStatusPastDue,
		SubscriptionStatusPaymentFailed,
		SubscriptionStatusCanceled,
		SubscriptionStatusExpired,
	},
	SubscriptionStatusPrepaidActive: {
		SubscriptionStatusExpired,
		SubscriptionStatusCanceled,
	},
	SubscriptionStatusCanceling: {
		SubscriptionStatusActive,
		SubscriptionStatusCanceled,
		SubscriptionStatusPastDue,
		SubscriptionStatusExpired,
	},
	SubscriptionStatusPastDue: {
		SubscriptionStatusActive,
		SubscriptionStatusPaymentFailed,
		SubscriptionStatusCanceled,
	},
	SubscriptionStatusPaymentFailed: {
		SubscriptionStatusCanceled,
		SubscriptionStatusExpired,
	},
	SubscriptionStatusCanceled: {},
	SubscriptionStatusExpired:  {},
}

// isAllowedSubscriptionTransition returns true when moving from -> to is a
// legal state-machine transition. old == new is always allowed (no-op).
func isAllowedSubscriptionTransition(old, new string) bool {
	if old == new {
		return true
	}
	for _, s := range allowedSubscriptionTransitions[old] {
		if s == new {
			return true
		}
	}
	return false
}

// TransitionSubscriptionStatus validates and applies a status transition to a
// UserSubscription inside the given transaction. It loads the row with
// lockForUpdate, validates newStatus via IsValidSubscriptionStatus, enforces
// the allowedSubscriptionTransitions state machine, and updates Status +
// UpdatedAt. A no-op (old == new) returns nil without writing.
//
// This function ONLY updates the Status field. Group-downgrade logic for
// terminal transitions (Canceled/Expired) is the caller's responsibility —
// the existing expiry task and CancelUserSubscriptionFromStripe already do it.
func TransitionSubscriptionStatus(tx *gorm.DB, subId int, newStatus string) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if subId <= 0 {
		return errors.New("invalid subId")
	}
	if !IsValidSubscriptionStatus(newStatus) {
		return fmt.Errorf("invalid subscription status: %s", newStatus)
	}
	var sub UserSubscription
	if err := lockForUpdate(tx).Where("id = ?", subId).First(&sub).Error; err != nil {
		return err
	}
	if sub.Status == newStatus {
		return nil
	}
	if !isAllowedSubscriptionTransition(sub.Status, newStatus) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidSubscriptionStatusTransition, sub.Status, newStatus)
	}
	now := common.GetTimestamp()
	return tx.Model(&sub).Updates(map[string]interface{}{
		"status":     newStatus,
		"updated_at": now,
	}).Error
}

// ExtendPrepaidSubscriptionEndTime stacks additional prepaid months onto an
// existing prepaid subscription. The new EndTime is computed as
// max(now, current EndTime) + additionalMonths months, so a still-active
// prepaid subscription extends from its current end (not from now), while an
// already-expired-but-still-prepaid_active row (a race) extends from now.
//
// Returns ErrSubscriptionNotPrepaid when the subscription is not in the
// prepaid_active status. Phase 10 (prepaid top-up stacking) calls this;
// Phase 3 only provides the function.
func ExtendPrepaidSubscriptionEndTime(subId int, additionalMonths int) error {
	if subId <= 0 {
		return errors.New("invalid subId")
	}
	if additionalMonths <= 0 {
		return errors.New("additionalMonths must be > 0")
	}
	var userId int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).Where("id = ?", subId).First(&sub).Error; err != nil {
			return err
		}
		userId = sub.UserId
		if sub.Status != SubscriptionStatusPrepaidActive {
			return ErrSubscriptionNotPrepaid
		}
		now := common.GetTimestamp()
		baseTime := now
		if sub.EndTime > baseTime {
			baseTime = sub.EndTime
		}
		newEnd := baseTime + int64(additionalMonths)*prepaidMonthSeconds
		return tx.Model(&sub).Updates(map[string]interface{}{
			"end_time":   newEnd,
			"updated_at": now,
		}).Error
	})
	if err != nil {
		return err
	}
	if userId > 0 {
		InvalidateUserSubscriptionCoverageCache(userId)
	}
	return nil
}

// GetPlanCoveredModels returns the canonical model IDs covered by a plan.
func GetPlanCoveredModels(planId int) ([]string, error) {
	var rows []SubscriptionPlanCoveredModel
	err := DB.Where("plan_id = ? AND enabled = ?", planId, true).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	models := make([]string, 0, len(rows))
	for _, r := range rows {
		models = append(models, r.ModelId)
	}
	return models, nil
}

// SetPlanCoveredModels replaces the covered-model list for a plan within a transaction.
// modelIds must be canonical model IDs (post-alias-resolution). Duplicates are ignored.
//
// Callers MUST call InvalidatePlanCoveredModelsCache() after the transaction commits
// so the subscription-coverage lookup cache does not serve stale results.
func SetPlanCoveredModels(tx *gorm.DB, planId int, modelIds []string) error {
	if err := tx.Where("plan_id = ?", planId).Delete(&SubscriptionPlanCoveredModel{}).Error; err != nil {
		return err
	}
	seen := make(map[string]bool, len(modelIds))
	now := common.GetTimestamp()
	for _, m := range modelIds {
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		if err := tx.Create(&SubscriptionPlanCoveredModel{
			PlanId:    planId,
			ModelId:   m,
			Enabled:   true,
			CreatedAt: now,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Subscription-covered model lookup (Phase 4: subscription-covered free models)
// ---------------------------------------------------------------------------

const subscriptionCoverageCacheNamespace = "new-api:subscription_coverage:v1"

var (
	subscriptionCoverageCacheOnce sync.Once
	subscriptionCoverageCache     *cachex.HybridCache[SubscriptionCoverage]
)

// SubscriptionCoverage is the cached result of a UserHasSubscriptionCoveringModel
// lookup. Cached even when Covered=false to avoid hitting the DB on every
// request for non-covered models (the common case).
type SubscriptionCoverage struct {
	Covered      bool              `json:"covered"`
	Subscription *UserSubscription `json:"subscription,omitempty"`
	Plan         *SubscriptionPlan `json:"plan,omitempty"`
}

func subscriptionCoverageCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("SUBSCRIPTION_COVERAGE_CACHE_TTL", 60)
	if ttlSeconds <= 0 {
		ttlSeconds = 60
	}
	return time.Duration(ttlSeconds) * time.Second
}

func subscriptionCoverageCacheCapacity() int {
	capacity := common.GetEnvOrDefault("SUBSCRIPTION_COVERAGE_CACHE_CAP", 50000)
	if capacity <= 0 {
		capacity = 50000
	}
	return capacity
}

func getSubscriptionCoverageCache() *cachex.HybridCache[SubscriptionCoverage] {
	subscriptionCoverageCacheOnce.Do(func() {
		ttl := subscriptionCoverageCacheTTL()
		subscriptionCoverageCache = cachex.NewHybridCache[SubscriptionCoverage](cachex.HybridCacheConfig[SubscriptionCoverage]{
			Namespace: cachex.Namespace(subscriptionCoverageCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[SubscriptionCoverage]{},
			Memory: func() *hot.HotCache[string, SubscriptionCoverage] {
				return hot.NewHotCache[string, SubscriptionCoverage](hot.LRU, subscriptionCoverageCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return subscriptionCoverageCache
}

func subscriptionCoverageCacheKey(userId int, modelName string) string {
	if userId <= 0 || modelName == "" {
		return ""
	}
	// Format: u:<userId>:m:<modelName>. modelName is the canonical model ID
	// (post-alias-resolution) and is used verbatim — invalidation is by prefix
	// (u:<userId>:) so embedded characters do not affect correctness.
	return fmt.Sprintf("u:%d:m:%s", userId, modelName)
}

// InvalidateUserSubscriptionCoverageCache invalidates all cached coverage
// lookups for a user. Call this whenever the user's subscription set changes
// (purchase, cancel, expire, status transition, prepaid extension, etc.).
func InvalidateUserSubscriptionCoverageCache(userId int) {
	if userId <= 0 {
		return
	}
	prefix := fmt.Sprintf("u:%d:m:", userId)
	_, _ = getSubscriptionCoverageCache().DeleteByPrefix(prefix)
}

// InvalidatePlanCoveredModelsCache invalidates all cached coverage lookups
// when a plan's covered-model list changes. Since the cache is keyed by
// (user, model) and not by plan, we purge the entire cache — plan changes
// are rare admin operations and the cache repopulates quickly.
func InvalidatePlanCoveredModelsCache() {
	_ = getSubscriptionCoverageCache().Purge()
}

// UserHasSubscriptionCoveringModel checks whether the user has an active
// subscription whose plan covers the given canonical model name. Returns the
// covering subscription (and its plan) if so. The lookup is cached: both
// positive and negative results are stored for subscriptionCoverageCacheTTL().
//
// modelName must be the CANONICAL model ID (post-alias-resolution), i.e.
// relayInfo.OriginModelName. Coverage is matched against the
// subscription_plan_covered_models table (enabled rows only).
//
// Selection policy: among the user's active subscriptions whose plan covers
// the model, the one with the earliest end_time is returned (matches the
// PreConsumeUserSubscription selection policy — consume the soonest-expiring
// sub first). On error the caller should fall through to the normal billing
// path rather than blocking the request.
func UserHasSubscriptionCoveringModel(userId int, modelName string) (covered bool, sub *UserSubscription, plan *SubscriptionPlan, err error) {
	if userId <= 0 || strings.TrimSpace(modelName) == "" {
		return false, nil, nil, nil
	}
	key := subscriptionCoverageCacheKey(userId, modelName)
	if key != "" {
		if cached, found, cacheErr := getSubscriptionCoverageCache().Get(key); cacheErr == nil && found {
			return cached.Covered, cached.Subscription, cached.Plan, nil
		}
	}
	now := common.GetTimestamp()
	var subs []UserSubscription
	if err = DB.Where("user_id = ? AND status = ? AND end_time > ?", userId, SubscriptionStatusActive, now).
		Order("end_time asc, id asc").
		Find(&subs).Error; err != nil {
		return false, nil, nil, err
	}
	var matchedSub *UserSubscription
	var matchedPlan *SubscriptionPlan
	for i := range subs {
		s := subs[i]
		var count int64
		if qErr := DB.Model(&SubscriptionPlanCoveredModel{}).
			Where("plan_id = ? AND model_id = ? AND enabled = ?", s.PlanId, modelName, true).
			Count(&count).Error; qErr != nil {
			return false, nil, nil, qErr
		}
		if count == 0 {
			continue
		}
		p, pErr := GetSubscriptionPlanById(s.PlanId)
		if pErr != nil {
			return false, nil, nil, pErr
		}
		matchedSub = &s
		matchedPlan = p
		break
	}
	result := SubscriptionCoverage{
		Covered:      matchedSub != nil,
		Subscription: matchedSub,
		Plan:         matchedPlan,
	}
	if key != "" {
		_ = getSubscriptionCoverageCache().SetWithTTL(key, result, subscriptionCoverageCacheTTL())
	}
	return result.Covered, result.Subscription, result.Plan, nil
}

// ValidateSubscriptionCoupon validates a coupon code for a user without reserving it.
// Returns the coupon and nil error if valid; nil and a descriptive error otherwise.
// This is a read-only validation — does NOT increment TimesRedeemed or create a redemption.
func ValidateSubscriptionCoupon(code string, userId int) (*SubscriptionCoupon, error) {
	var coupon SubscriptionCoupon
	err := DB.Where("code = ?", code).First(&coupon).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionCouponNotFound
		}
		return nil, err
	}
	if !coupon.Enabled {
		return nil, ErrSubscriptionCouponDisabled
	}
	now := common.GetTimestamp()
	if coupon.StartAt > 0 && now < coupon.StartAt {
		return nil, ErrSubscriptionCouponNotStarted
	}
	if coupon.EndAt > 0 && now > coupon.EndAt {
		return nil, ErrSubscriptionCouponExpired
	}
	if coupon.MaxRedemptions > 0 && coupon.TimesRedeemed >= coupon.MaxRedemptions {
		return nil, ErrSubscriptionCouponUsageCapReached
	}
	if coupon.PerUserLimit > 0 {
		var count int64
		err = DB.Model(&SubscriptionCouponRedemption{}).
			Where("coupon_id = ? AND user_id = ? AND status IN ?", coupon.Id, userId,
				[]string{CouponRedemptionStatusReserved, CouponRedemptionStatusIssued}).
			Count(&count).Error
		if err != nil {
			return nil, err
		}
		if count >= int64(coupon.PerUserLimit) {
			return nil, ErrSubscriptionCouponPerUserLimitReached
		}
	}
	if coupon.NewUserOnly {
		var existingCount int64
		err = DB.Model(&UserSubscription{}).
			Where("user_id = ? AND status IN ?", userId,
				[]string{SubscriptionStatusActive, SubscriptionStatusPrepaidActive, SubscriptionStatusCanceling}).
			Count(&existingCount).Error
		if err != nil {
			return nil, err
		}
		if existingCount > 0 {
			return nil, ErrSubscriptionCouponNewUserOnly
		}
	}
	return &coupon, nil
}

// PreConsumeUserSubscription pre-consumes from any active subscription total quota.
func PreConsumeUserSubscription(requestId string, userId int, modelName string, quotaType int, amount int64) (*SubscriptionPreConsumeResult, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	if strings.TrimSpace(requestId) == "" {
		return nil, errors.New("requestId is empty")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}
	now := GetDBTimestamp()

	returnValue := &SubscriptionPreConsumeResult{}

	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing SubscriptionPreConsumeRecord
		query := tx.Where("request_id = ?", requestId).Limit(1).Find(&existing)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected > 0 {
			if existing.Status == "refunded" {
				return errors.New("subscription pre-consume already refunded")
			}
			var sub UserSubscription
			if err := tx.Where("id = ?", existing.UserSubscriptionId).First(&sub).Error; err != nil {
				return err
			}
			returnValue.UserSubscriptionId = sub.Id
			returnValue.PreConsumed = existing.PreConsumed
			returnValue.AmountTotal = sub.AmountTotal
			returnValue.AmountUsedBefore = sub.AmountUsed
			returnValue.AmountUsedAfter = sub.AmountUsed
			return nil
		}

		var subs []UserSubscription
		if err := lockForUpdate(tx).
			Where("user_id = ? AND status = ? AND end_time > ?", userId, "active", now).
			Order("end_time asc, id asc").
			Find(&subs).Error; err != nil {
			return errors.New("no active subscription")
		}
		if len(subs) == 0 {
			return errors.New("no active subscription")
		}
		for _, candidate := range subs {
			sub := candidate
			plan, err := getSubscriptionPlanByIdTx(tx, sub.PlanId)
			if err != nil {
				return err
			}
			if err := maybeResetUserSubscriptionWithPlanTx(tx, &sub, plan, now); err != nil {
				return err
			}
			usedBefore := sub.AmountUsed
			if sub.AmountTotal > 0 {
				remain := sub.AmountTotal - usedBefore
				if remain < amount {
					continue
				}
			}
			record := &SubscriptionPreConsumeRecord{
				RequestId:          requestId,
				UserId:             userId,
				UserSubscriptionId: sub.Id,
				PreConsumed:        amount,
				Status:             "consumed",
			}
			if err := tx.Create(record).Error; err != nil {
				var dup SubscriptionPreConsumeRecord
				if err2 := tx.Where("request_id = ?", requestId).First(&dup).Error; err2 == nil {
					if dup.Status == "refunded" {
						return errors.New("subscription pre-consume already refunded")
					}
					returnValue.UserSubscriptionId = sub.Id
					returnValue.PreConsumed = dup.PreConsumed
					returnValue.AmountTotal = sub.AmountTotal
					returnValue.AmountUsedBefore = sub.AmountUsed
					returnValue.AmountUsedAfter = sub.AmountUsed
					return nil
				}
				return err
			}
			sub.AmountUsed += amount
			if err := tx.Save(&sub).Error; err != nil {
				return err
			}
			returnValue.UserSubscriptionId = sub.Id
			returnValue.PreConsumed = amount
			returnValue.AmountTotal = sub.AmountTotal
			returnValue.AmountUsedBefore = usedBefore
			returnValue.AmountUsedAfter = sub.AmountUsed
			return nil
		}
		return fmt.Errorf("subscription quota insufficient, need=%d", amount)
	})
	if err != nil {
		return nil, err
	}
	return returnValue, nil
}

// RefundSubscriptionPreConsume is idempotent and refunds pre-consumed subscription quota by requestId.
func RefundSubscriptionPreConsume(requestId string) error {
	if strings.TrimSpace(requestId) == "" {
		return errors.New("requestId is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var record SubscriptionPreConsumeRecord
		if err := lockForUpdate(tx).
			Where("request_id = ?", requestId).First(&record).Error; err != nil {
			return err
		}
		if record.Status == "refunded" {
			return nil
		}
		if record.PreConsumed <= 0 {
			record.Status = "refunded"
			return tx.Save(&record).Error
		}
		if err := PostConsumeUserSubscriptionDelta(record.UserSubscriptionId, -record.PreConsumed); err != nil {
			return err
		}
		record.Status = "refunded"
		return tx.Save(&record).Error
	})
}

// ResetDueSubscriptions resets subscriptions whose next_reset_time has passed.
func ResetDueSubscriptions(limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	now := GetDBTimestamp()
	var subs []UserSubscription
	if err := DB.Where("next_reset_time > 0 AND next_reset_time <= ? AND status = ?", now, "active").
		Order("next_reset_time asc").
		Limit(limit).
		Find(&subs).Error; err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}
	resetCount := 0
	for _, sub := range subs {
		subCopy := sub
		plan, err := getSubscriptionPlanByIdTx(nil, sub.PlanId)
		if err != nil || plan == nil {
			continue
		}
		err = DB.Transaction(func(tx *gorm.DB) error {
			var locked UserSubscription
			if err := lockForUpdate(tx).
				Where("id = ? AND next_reset_time > 0 AND next_reset_time <= ?", subCopy.Id, now).
				First(&locked).Error; err != nil {
				return nil
			}
			if err := maybeResetUserSubscriptionWithPlanTx(tx, &locked, plan, now); err != nil {
				return err
			}
			resetCount++
			return nil
		})
		if err != nil {
			return resetCount, err
		}
	}
	return resetCount, nil
}

// CleanupSubscriptionPreConsumeRecords removes old idempotency records to keep table small.
func CleanupSubscriptionPreConsumeRecords(olderThanSeconds int64) (int64, error) {
	if olderThanSeconds <= 0 {
		olderThanSeconds = 7 * 24 * 3600
	}
	cutoff := GetDBTimestamp() - olderThanSeconds
	res := DB.Where("updated_at < ?", cutoff).Delete(&SubscriptionPreConsumeRecord{})
	return res.RowsAffected, res.Error
}

type SubscriptionPlanInfo struct {
	PlanId    int
	PlanTitle string
}

func GetSubscriptionPlanInfoByUserSubscriptionId(userSubscriptionId int) (*SubscriptionPlanInfo, error) {
	if userSubscriptionId <= 0 {
		return nil, errors.New("invalid userSubscriptionId")
	}
	cacheKey := fmt.Sprintf("sub:%d", userSubscriptionId)
	if cached, found, err := getSubscriptionPlanInfoCache().Get(cacheKey); err == nil && found {
		return &cached, nil
	}
	var sub UserSubscription
	if err := DB.Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
		return nil, err
	}
	plan, err := getSubscriptionPlanByIdTx(nil, sub.PlanId)
	if err != nil {
		return nil, err
	}
	info := &SubscriptionPlanInfo{
		PlanId:    sub.PlanId,
		PlanTitle: plan.Title,
	}
	_ = getSubscriptionPlanInfoCache().SetWithTTL(cacheKey, *info, subscriptionPlanInfoCacheTTL())
	return info, nil
}

// Update subscription used amount by delta (positive consume more, negative refund).
func PostConsumeUserSubscriptionDelta(userSubscriptionId int, delta int64) error {
	if userSubscriptionId <= 0 {
		return errors.New("invalid userSubscriptionId")
	}
	if delta == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).
			Where("id = ?", userSubscriptionId).
			First(&sub).Error; err != nil {
			return err
		}
		newUsed := sub.AmountUsed + delta
		if newUsed < 0 {
			newUsed = 0
		}
		if sub.AmountTotal > 0 && newUsed > sub.AmountTotal {
			return fmt.Errorf("subscription used exceeds total, used=%d total=%d", newUsed, sub.AmountTotal)
		}
		sub.AmountUsed = newUsed
		return tx.Save(&sub).Error
	})
}

// ReserveSubscriptionCouponWithTx locks the coupon row, re-validates its
// eligibility inside the transaction, and inserts a SubscriptionCouponRedemption
// row in the "reserved" state. Idempotent: if a redemption with the same
// OrderId already exists and is reserved/issued, the existing row is returned
// without creating a new one or bumping TimesRedeemed. Mirrors the structure
// of ReserveTopupPromotionWithTx.
func ReserveSubscriptionCouponWithTx(tx *gorm.DB, couponId int64, userId int, planId int, orderId string, percentOff int, originalAmount, discountAmount, finalAmount int64, currency string) (*SubscriptionCouponRedemption, error) {
	if tx == nil || couponId <= 0 || userId <= 0 || planId <= 0 || orderId == "" {
		return nil, errors.New("invalid subscription coupon reservation")
	}
	// Idempotency: an existing reserved/issued redemption for this order is returned as-is.
	var existing SubscriptionCouponRedemption
	if err := tx.Where("order_id = ?", orderId).First(&existing).Error; err == nil {
		if existing.CouponId == couponId && existing.UserId == userId {
			return &existing, nil
		}
		return nil, errors.New("subscription coupon order already reserved with different terms")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Lock the coupon row and re-validate every eligibility check inside the tx.
	var coupon SubscriptionCoupon
	if err := lockForUpdate(tx).First(&coupon, couponId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionCouponNotFound
		}
		return nil, err
	}
	if !coupon.Enabled {
		return nil, ErrSubscriptionCouponDisabled
	}
	now := common.GetTimestamp()
	if coupon.StartAt > 0 && now < coupon.StartAt {
		return nil, ErrSubscriptionCouponNotStarted
	}
	if coupon.EndAt > 0 && now > coupon.EndAt {
		return nil, ErrSubscriptionCouponExpired
	}
	if coupon.MaxRedemptions > 0 && coupon.TimesRedeemed >= coupon.MaxRedemptions {
		return nil, ErrSubscriptionCouponUsageCapReached
	}
	if coupon.PerUserLimit > 0 {
		var used int64
		activeStatuses := []string{CouponRedemptionStatusReserved, CouponRedemptionStatusIssued}
		if err := tx.Model(&SubscriptionCouponRedemption{}).
			Where("coupon_id = ? AND user_id = ? AND status IN ?", coupon.Id, userId, activeStatuses).
			Count(&used).Error; err != nil {
			return nil, err
		}
		if used >= int64(coupon.PerUserLimit) {
			return nil, ErrSubscriptionCouponPerUserLimitReached
		}
	}

	redemption := &SubscriptionCouponRedemption{
		OrderId:        orderId,
		CouponId:       coupon.Id,
		UserId:         userId,
		PlanId:         planId,
		Status:         CouponRedemptionStatusReserved,
		PercentOff:     percentOff,
		OriginalAmount: originalAmount,
		DiscountAmount: discountAmount,
		FinalAmount:    finalAmount,
		Currency:       currency,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := tx.Create(redemption).Error; err != nil {
		return nil, err
	}
	coupon.TimesRedeemed += 1
	if err := tx.Save(&coupon).Error; err != nil {
		return nil, err
	}
	return redemption, nil
}

// ReleaseSubscriptionCouponRedemption marks a reserved redemption as released.
// Used when a checkout Session fails / expires before payment is captured.
// Issued redemptions cannot be released (they must be reversed).
//
// Thin wrapper around ReleaseSubscriptionCouponWithTx so the TimesRedeemed
// decrement is consistent across the checkout-failure and session-expired paths.
func ReleaseSubscriptionCouponRedemption(orderId string) error {
	if orderId == "" {
		return errors.New("orderId is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseSubscriptionCouponWithTx(tx, orderId)
	})
}

// IssueSubscriptionCouponWithTx transitions a reserved redemption to issued
// after the corresponding payment is captured. Idempotent: a redemption that is
// already issued (or absent) is a no-op. Mirrors IssueTopupPromotionWithTx.
func IssueSubscriptionCouponWithTx(tx *gorm.DB, orderId string) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if orderId == "" {
		return errors.New("orderId is empty")
	}
	var redemption SubscriptionCouponRedemption
	if err := lockForUpdate(tx).Where("order_id = ?", orderId).First(&redemption).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if redemption.Status == CouponRedemptionStatusIssued {
		return nil
	}
	if redemption.Status != CouponRedemptionStatusReserved {
		return ErrSubscriptionCouponRedemptionNotReserved
	}
	now := common.GetTimestamp()
	redemption.Status = CouponRedemptionStatusIssued
	redemption.IssuedAt = now
	redemption.UpdatedAt = now
	return tx.Save(&redemption).Error
}

// ReleaseSubscriptionCouponWithTx transitions a reserved redemption to
// released and decrements the coupon's TimesRedeemed counter (the reservation
// never led to issuance, so it must not count as a redemption). Idempotent: a
// redemption already released/reversed (or absent) is a no-op. Mirrors
// ReleaseTopupPromotionWithTx.
func ReleaseSubscriptionCouponWithTx(tx *gorm.DB, orderId string) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if orderId == "" {
		return errors.New("orderId is empty")
	}
	var redemption SubscriptionCouponRedemption
	if err := lockForUpdate(tx).Where("order_id = ?", orderId).First(&redemption).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if redemption.Status == CouponRedemptionStatusReleased || redemption.Status == CouponRedemptionStatusReversed {
		return nil
	}
	if redemption.Status != CouponRedemptionStatusReserved {
		return ErrSubscriptionCouponRedemptionNotReserved
	}
	// Lock the coupon row and decrement TimesRedeemed (guarded against going
	// negative — a reservation that never issued should not count).
	var coupon SubscriptionCoupon
	if err := lockForUpdate(tx).First(&coupon, redemption.CouponId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSubscriptionCouponNotFound
		}
		return err
	}
	if coupon.TimesRedeemed > 0 {
		coupon.TimesRedeemed -= 1
	}
	now := common.GetTimestamp()
	coupon.UpdatedAt = now
	if err := tx.Save(&coupon).Error; err != nil {
		return err
	}
	redemption.Status = CouponRedemptionStatusReleased
	redemption.ReleasedAt = now
	redemption.UpdatedAt = now
	return tx.Save(&redemption).Error
}

// ReverseSubscriptionCouponWithTx transitions an issued redemption to
// reversed (e.g. after a refund). Idempotent: a redemption already reversed (or
// absent) is a no-op. TimesRedeemed is NOT decremented — the redemption did
// happen, it is just being reversed for accounting. reason is recorded in the
// caller's audit log; this function does not persist it on the redemption row.
// Mirrors ReverseTopupPromotionWithTx.
func ReverseSubscriptionCouponWithTx(tx *gorm.DB, orderId string, reason string) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if orderId == "" {
		return errors.New("orderId is empty")
	}
	var redemption SubscriptionCouponRedemption
	if err := lockForUpdate(tx).Where("order_id = ?", orderId).First(&redemption).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if redemption.Status == CouponRedemptionStatusReversed {
		return nil
	}
	if redemption.Status != CouponRedemptionStatusIssued {
		return ErrSubscriptionCouponRedemptionNotIssued
	}
	now := common.GetTimestamp()
	redemption.Status = CouponRedemptionStatusReversed
	redemption.ReversedAt = now
	redemption.UpdatedAt = now
	return tx.Save(&redemption).Error
}

// MarkSubscriptionOrderStatus transitions a SubscriptionOrder's status with
// from/to guards. Mirrors MarkStripeTopupOrderStatus. fromStatus="*" matches
// any non-terminal status. Returns ErrSubscriptionOrderNotFound when the
// order doesn't exist.
func MarkSubscriptionOrderStatus(tradeNo, fromStatus, toStatus, reason string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var order SubscriptionOrder
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrSubscriptionOrderNotFound
		}
		if fromStatus != "" && fromStatus != "*" && order.Status != fromStatus {
			if order.Status == toStatus {
				return nil
			}
			return ErrSubscriptionOrderStatusInvalid
		}
		if order.Status == common.TopUpStatusSuccess {
			// never silently downgrade a paid order
			return ErrSubscriptionOrderStatusInvalid
		}
		order.Status = toStatus
		if reason != "" {
			// Persist the failure reason into ProviderPayload for diagnosis
			// (subscription orders do not have a dedicated failure_reason column).
			order.ProviderPayload = reason
		}
		order.CompleteTime = common.GetTimestamp()
		return tx.Save(&order).Error
	})
}

// GetActiveUserSubscriptionByStripeId returns the UserSubscription linked to
// a Stripe Subscription ID, regardless of status. Returns nil, nil when no
// such subscription exists (the webhook handler treats this as "not our
// subscription" and returns nil).
func FindUserSubscriptionByStripeId(stripeSubId string) (*UserSubscription, error) {
	if stripeSubId == "" {
		return nil, nil
	}
	var sub UserSubscription
	if err := DB.Where("stripe_subscription_id = ?", stripeSubId).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

// HasActiveAutoRenewSubscriptionForPlan returns true if the user already has
// an active auto-renew subscription for the given plan (used as a basic
// duplicate-prevention guard at checkout time).
func HasActiveAutoRenewSubscriptionForPlan(userId int, planId int) (bool, error) {
	if userId <= 0 || planId <= 0 {
		return false, errors.New("invalid userId or planId")
	}
	var count int64
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND plan_id = ? AND status IN ?",
			userId, planId,
			[]string{SubscriptionStatusActive, SubscriptionStatusCanceling, SubscriptionStatusPastDue}).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasActiveAutoRenewSubscription returns true if the user has ANY active
// auto-renew subscription (Status in active, canceling, past_due) for ANY
// plan. A user should only have ONE auto-renew subscription at a time, so this
// is the hardened Phase 10 duplicate-prevention guard (replaces the per-plan
// check, which allowed stacking auto-renew subscriptions across plans).
//
// Policy (Phase 10): even a `canceling` subscription blocks a new auto-renew
// checkout. The user must let the existing subscription expire
// (cancel_at_period_end) or contact support to switch plans early. This is the
// safer default — allowing a new auto-renew subscription while a previous one
// is still "canceling" can lead to double-charges if the user later reactivates
// the old one or if a webhook race delays the cancellation.
func HasActiveAutoRenewSubscription(userId int) (bool, error) {
	if userId <= 0 {
		return false, errors.New("invalid userId")
	}
	var count int64
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND status IN ?",
			userId,
			[]string{SubscriptionStatusActive, SubscriptionStatusCanceling, SubscriptionStatusPastDue}).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindRecentPendingSubscriptionOrder returns the most-recent pending
// SubscriptionOrder for the given user+plan+mode created within the last
// withinSeconds seconds. Returns (nil, nil) when no such order exists. Used by
// the checkout controller to deduplicate rapid double-clicks: if a pending
// order from the same user+plan+mode exists within the dedup window, the
// checkout handler can return the existing Stripe session URL (when available)
// or a 409 "checkout in progress" instead of creating a second order and a
// second Stripe Checkout Session.
func FindRecentPendingSubscriptionOrder(userId, planId int, mode string, withinSeconds int64) (*SubscriptionOrder, error) {
	if userId <= 0 || planId <= 0 || mode == "" || withinSeconds <= 0 {
		return nil, nil
	}
	cutoff := common.GetTimestamp() - withinSeconds
	var order SubscriptionOrder
	err := DB.Where("user_id = ? AND plan_id = ? AND mode = ? AND status = ? AND create_time >= ?",
		userId, planId, mode, common.TopUpStatusPending, cutoff).
		Order("create_time desc, id desc").
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// CompleteSubscriptionOrderV2 supersedes CompleteSubscriptionOrder for the
// NovaPura v2 flow. It is idempotent (calling twice with the same tradeNo
// returns nil the second time), creates a UserSubscription snapshot from the
// plan, links the Stripe Subscription ID, marks the order Success, and issues
// any reserved coupon redemption (status reserved -> issued).
//
// mode is "auto_renew" or "prepaid"; for prepaid the EndTime is multiplied
// by order.PrepaidMonths. For auto_renew the EndTime is set as a placeholder
// estimate (the first invoice.paid webhook overwrites it with the real
// period end).
func CompleteSubscriptionOrderV2(tradeNo string, providerPayload string, stripeSubscriptionId string, stripeCustomerId string, mode string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	var logUserId int
	var logPlanTitle string
	var logMoney float64
	var logPaymentMethod string
	var upgradeGroup string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var order SubscriptionOrder
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrSubscriptionOrderNotFound
		}
		if order.PaymentProvider != PaymentProviderStripe {
			return ErrPaymentMethodMismatch
		}
		if order.Status == common.TopUpStatusSuccess {
			return nil
		}
		if order.Status != common.TopUpStatusPending {
			return ErrSubscriptionOrderStatusInvalid
		}
		plan, err := getSubscriptionPlanByIdTx(tx, order.PlanId)
		if err != nil {
			return err
		}

		// Compute the subscription EndTime. Auto-renew uses an initial estimate
		// (the first invoice.paid handler will overwrite it with the real
		// period end from Stripe). Prepaid multiplies the plan duration by the
		// number of prepaid months. We do NOT modify calcPlanEndTime's signature
		// (other callers depend on it); instead we compute prepaid EndTime inline
		// using the same month-multiplier semantics.
		nowUnix := GetDBTimestamp()
		now := time.Unix(nowUnix, 0)
		var endUnix int64
		if mode == "prepaid" && order.PrepaidMonths > 1 {
			months := plan.DurationValue * order.PrepaidMonths
			switch plan.DurationUnit {
			case SubscriptionDurationYear:
				endUnix = now.AddDate(months, 0, 0).Unix()
			case SubscriptionDurationMonth:
				endUnix = now.AddDate(0, months, 0).Unix()
			case SubscriptionDurationDay:
				endUnix = now.Add(time.Duration(months) * 24 * time.Hour).Unix()
			case SubscriptionDurationHour:
				endUnix = now.Add(time.Duration(months) * time.Hour).Unix()
			case SubscriptionDurationCustom:
				if plan.CustomSeconds > 0 {
					endUnix = now.Add(time.Duration(plan.CustomSeconds*int64(order.PrepaidMonths)) * time.Second).Unix()
				} else {
					endUnix = now.AddDate(0, months, 0).Unix()
				}
			default:
				endUnix = now.AddDate(0, months, 0).Unix()
			}
		} else {
			endUnix, err = calcPlanEndTime(now, plan)
			if err != nil {
				return err
			}
		}

		resetBase := now
		nextReset := calcNextResetTime(resetBase, plan, endUnix)
		lastReset := int64(0)
		if nextReset > 0 {
			lastReset = now.Unix()
		}

		// Snapshot the user's pre-purchase group, then upgrade if the plan defines one.
		upgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
		prevGroup := ""
		if upgradeGroup != "" {
			currentGroup, err := getUserGroupByIdTx(tx, order.UserId)
			if err != nil {
				return err
			}
			if currentGroup != upgradeGroup {
				prevGroup = currentGroup
				if err := tx.Model(&User{}).Where("id = ?", order.UserId).
					Update("group", upgradeGroup).Error; err != nil {
					return err
				}
			}
		}
		allowWalletOverflow := true
		if plan.AllowWalletOverflow != nil {
			allowWalletOverflow = *plan.AllowWalletOverflow
		}

		// Determine the active status from the mode.
		status := SubscriptionStatusActive
		if mode == "prepaid" {
			status = SubscriptionStatusPrepaidActive
		}

		sub := &UserSubscription{
			UserId:               order.UserId,
			PlanId:               plan.Id,
			AmountTotal:          plan.TotalAmount,
			AmountUsed:           0,
			StartTime:            now.Unix(),
			EndTime:              endUnix,
			Status:               status,
			Source:               "order",
			LastResetTime:        lastReset,
			NextResetTime:        nextReset,
			UpgradeGroup:         upgradeGroup,
			PrevUserGroup:        prevGroup,
			DowngradeGroup:       strings.TrimSpace(plan.DowngradeGroup),
			AllowWalletOverflow:  allowWalletOverflow,
			StripeSubscriptionId: stripeSubscriptionId,
			StripeCustomerId:     stripeCustomerId,
			CancelAtPeriodEnd:    false,
			CreatedAt:            common.GetTimestamp(),
			UpdatedAt:            common.GetTimestamp(),
		}
		if order.CouponRedemptionId != nil {
			sub.CouponRedemptionId = order.CouponRedemptionId
			// Look up the coupon_id from the redemption row so the snapshot is complete.
			var rd SubscriptionCouponRedemption
			if err := tx.Where("order_id = ?", tradeNo).First(&rd).Error; err == nil {
				couponId := int(rd.CouponId)
				sub.CouponId = &couponId
			}
		}
		if err := tx.Create(sub).Error; err != nil {
			return err
		}

		// Issue any reserved coupon redemption (reserved -> issued). Delegates
		// to IssueSubscriptionCouponWithTx so the issue transition (including
		// lockForUpdate and idempotency) has a single source of truth.
		if err := IssueSubscriptionCouponWithTx(tx, tradeNo); err != nil {
			return err
		}

		if err := upsertSubscriptionTopUpTx(tx, &order); err != nil {
			return err
		}
		order.Status = common.TopUpStatusSuccess
		order.CompleteTime = common.GetTimestamp()
		if providerPayload != "" {
			order.ProviderPayload = providerPayload
		}
		order.StripeSubscriptionId = stripeSubscriptionId
		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		// Settle affiliate commission on the actual paid amount (post-coupon).
		// The commission is settled inside the same transaction so the order's
		// own idempotency protects commission idempotency too. A failure here
		// must NOT block the subscription activation — log and continue.
		// ConvertAmountToUSDCents safe-fails to 0 on unsupported currency or
		// NaN/Inf input, so an unconvertible amount just skips commission.
		if order.FinalAmount > 0 {
			paidCentsUSD := ConvertAmountToUSDCents(float64(order.FinalAmount)/100.0, order.Currency)
			if paidCentsUSD > 0 {
				if _, settleErr := SettleRechargeCommission(tx, order.UserId, order.TradeNo, paidCentsUSD, "USD"); settleErr != nil {
					common.SysError(fmt.Sprintf("subscription commission settle failed order=%s err=%q", order.TradeNo, settleErr.Error()))
				}
			}
		}

		logUserId = order.UserId
		logPlanTitle = plan.Title
		logMoney = order.Money
		logPaymentMethod = order.PaymentMethod
		return nil
	})
	if err != nil {
		return err
	}
	if upgradeGroup != "" && logUserId > 0 {
		_ = UpdateUserGroupCache(logUserId, upgradeGroup)
	}
	if logUserId > 0 {
		InvalidateUserSubscriptionCoverageCache(logUserId)
		msg := fmt.Sprintf("订阅购买成功，套餐: %s，支付金额: %.2f，支付方式: %s", logPlanTitle, logMoney, logPaymentMethod)
		RecordLog(logUserId, LogTypeTopup, msg)
	}
	return nil
}

// CompletePrepaidSubscriptionOrderOrExtend completes a prepaid order. If the
// user already has a prepaid_active subscription for the same plan, it extends
// that subscription's EndTime by prepaidMonths (stacking). Otherwise it creates
// a new UserSubscription with status prepaid_active. Idempotent on the order's
// TradeNo: a second call with an already-Success order returns nil without
// extending again.
//
// Prepaid stacking rules (Phase 10):
//   - Prepaid is ALLOWED on top of an existing auto-renew subscription
//     (they coexist as separate UserSubscription rows).
//   - Prepaid on top of an existing prepaid_active subscription for the SAME
//     plan extends that subscription's EndTime (no new row created).
//   - Prepaid for a plan with no existing prepaid_active subscription creates
//     a new row, even if the user has a prepaid_active subscription for a
//     DIFFERENT plan (multi-plan prepaid coexistence is allowed).
//
// The EndTime extension uses the same max(now, current EndTime) + months*30days
// semantics as ExtendPrepaidSubscriptionEndTime. The extension is inlined here
// (rather than calling ExtendPrepaidSubscriptionEndTime) so it runs inside this
// transaction with the row already locked.
func CompletePrepaidSubscriptionOrderOrExtend(tradeNo string, providerPayload string, prepaidMonths int, stripeSubscriptionId string, stripeCustomerId string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	if prepaidMonths <= 0 {
		return errors.New("prepaidMonths must be > 0")
	}
	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	var logUserId int
	var logPlanTitle string
	var logMoney float64
	var logPaymentMethod string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var order SubscriptionOrder
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrSubscriptionOrderNotFound
		}
		if order.PaymentProvider != PaymentProviderStripe {
			return ErrPaymentMethodMismatch
		}
		if order.Status == common.TopUpStatusSuccess {
			return nil
		}
		if order.Status != common.TopUpStatusPending {
			return ErrSubscriptionOrderStatusInvalid
		}
		plan, err := getSubscriptionPlanByIdTx(tx, order.PlanId)
		if err != nil {
			return err
		}

		// Look for an existing prepaid_active subscription for the same
		// user+plan. If found, extend its EndTime; otherwise create a new one.
		var existing UserSubscription
		findErr := tx.Where("user_id = ? AND plan_id = ? AND status = ?",
			order.UserId, order.PlanId, SubscriptionStatusPrepaidActive).
			Order("end_time desc, id desc").
			First(&existing).Error
		if findErr == nil {
			now := common.GetTimestamp()
			baseTime := now
			if existing.EndTime > baseTime {
				baseTime = existing.EndTime
			}
			newEnd := baseTime + int64(prepaidMonths)*prepaidMonthSeconds
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"end_time":   newEnd,
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			// No existing prepaid sub for this plan — create a new one. Reuse
			// the EndTime computation from CompleteSubscriptionOrderV2.
			nowUnix := GetDBTimestamp()
			now := time.Unix(nowUnix, 0)
			months := plan.DurationValue * prepaidMonths
			var endUnix int64
			switch plan.DurationUnit {
			case SubscriptionDurationYear:
				endUnix = now.AddDate(months, 0, 0).Unix()
			case SubscriptionDurationDay:
				endUnix = now.Add(time.Duration(months) * 24 * time.Hour).Unix()
			case SubscriptionDurationHour:
				endUnix = now.Add(time.Duration(months) * time.Hour).Unix()
			case SubscriptionDurationCustom:
				if plan.CustomSeconds > 0 {
					endUnix = now.Add(time.Duration(plan.CustomSeconds*int64(prepaidMonths)) * time.Second).Unix()
				} else {
					endUnix = now.AddDate(0, months, 0).Unix()
				}
			default:
				endUnix = now.AddDate(0, months, 0).Unix()
			}
			resetBase := now
			nextReset := calcNextResetTime(resetBase, plan, endUnix)
			lastReset := int64(0)
			if nextReset > 0 {
				lastReset = now.Unix()
			}
			upgradeGroup := strings.TrimSpace(plan.UpgradeGroup)
			prevGroup := ""
			if upgradeGroup != "" {
				currentGroup, gerr := getUserGroupByIdTx(tx, order.UserId)
				if gerr != nil {
					return gerr
				}
				if currentGroup != upgradeGroup {
					prevGroup = currentGroup
					if err := tx.Model(&User{}).Where("id = ?", order.UserId).
						Update("group", upgradeGroup).Error; err != nil {
						return err
					}
				}
			}
			allowWalletOverflow := true
			if plan.AllowWalletOverflow != nil {
				allowWalletOverflow = *plan.AllowWalletOverflow
			}
			sub := &UserSubscription{
				UserId:              order.UserId,
				PlanId:              plan.Id,
				AmountTotal:         plan.TotalAmount,
				AmountUsed:          0,
				StartTime:           now.Unix(),
				EndTime:             endUnix,
				Status:              SubscriptionStatusPrepaidActive,
				Source:              "order",
				LastResetTime:       lastReset,
				NextResetTime:       nextReset,
				UpgradeGroup:        upgradeGroup,
				PrevUserGroup:       prevGroup,
				DowngradeGroup:      strings.TrimSpace(plan.DowngradeGroup),
				AllowWalletOverflow: allowWalletOverflow,
				CancelAtPeriodEnd:   false,
				// For prepaid mode the linkage key is the Stripe PaymentIntent ID
				// (there is no Stripe Subscription). The stacking path does NOT
				// overwrite this — the existing sub keeps its original linkage.
				StripeSubscriptionId: stripeSubscriptionId,
				StripeCustomerId:     stripeCustomerId,
				CreatedAt:            common.GetTimestamp(),
				UpdatedAt:            common.GetTimestamp(),
			}
			if order.CouponRedemptionId != nil {
				sub.CouponRedemptionId = order.CouponRedemptionId
				var rd SubscriptionCouponRedemption
				if err := tx.Where("order_id = ?", tradeNo).First(&rd).Error; err == nil {
					couponId := int(rd.CouponId)
					sub.CouponId = &couponId
				}
			}
			if err := tx.Create(sub).Error; err != nil {
				return err
			}
		} else {
			return findErr
		}

		// Issue any reserved coupon redemption (reserved -> issued).
		if err := IssueSubscriptionCouponWithTx(tx, tradeNo); err != nil {
			return err
		}
		if err := upsertSubscriptionTopUpTx(tx, &order); err != nil {
			return err
		}
		order.Status = common.TopUpStatusSuccess
		order.CompleteTime = common.GetTimestamp()
		if providerPayload != "" {
			order.ProviderPayload = providerPayload
		}
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		// Settle affiliate commission on the actual paid amount (post-coupon).
		// Mirrors CompleteSubscriptionOrderV2: a settlement failure must not
		// block the subscription activation (log and continue).
		if order.FinalAmount > 0 {
			paidCentsUSD := ConvertAmountToUSDCents(float64(order.FinalAmount)/100.0, order.Currency)
			if paidCentsUSD > 0 {
				if _, settleErr := SettleRechargeCommission(tx, order.UserId, order.TradeNo, paidCentsUSD, "USD"); settleErr != nil {
					common.SysError(fmt.Sprintf("subscription commission settle failed order=%s err=%q", order.TradeNo, settleErr.Error()))
				}
			}
		}
		logUserId = order.UserId
		logPlanTitle = plan.Title
		logMoney = order.Money
		logPaymentMethod = order.PaymentMethod
		return nil
	})
	if err != nil {
		return err
	}
	if logUserId > 0 {
		InvalidateUserSubscriptionCoverageCache(logUserId)
		msg := fmt.Sprintf("订阅购买成功，套餐: %s，支付金额: %.2f，支付方式: %s", logPlanTitle, logMoney, logPaymentMethod)
		RecordLog(logUserId, LogTypeTopup, msg)
	}
	return nil
}

// ReverseSubscriptionCouponRedemptionsForSubscription flips every issued
// redemption tied to a UserSubscription to "reversed". Called when the
// underlying Stripe subscription is deleted (refund / cancellation). Returns
// the number of redemptions reversed.
func ReverseSubscriptionCouponRedemptionsForSubscription(userSubscriptionId int) (int, error) {
	if userSubscriptionId <= 0 {
		return 0, errors.New("invalid userSubscriptionId")
	}
	reversed := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		var subs []UserSubscription
		if err := tx.Where("id = ?", userSubscriptionId).Find(&subs).Error; err != nil {
			return err
		}
		if len(subs) == 0 {
			return nil
		}
		sub := subs[0]
		if sub.CouponRedemptionId == nil {
			return nil
		}
		var redemption SubscriptionCouponRedemption
		if err := lockForUpdate(tx).First(&redemption, *sub.CouponRedemptionId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if redemption.Status != CouponRedemptionStatusIssued {
			return nil
		}
		now := common.GetTimestamp()
		redemption.Status = CouponRedemptionStatusReversed
		redemption.ReversedAt = now
		redemption.UpdatedAt = now
		if err := tx.Save(&redemption).Error; err != nil {
			return err
		}
		reversed = 1
		return nil
	})
	if err != nil {
		return 0, err
	}
	return reversed, nil
}

// RenewUserSubscriptionFromStripe extends a subscription after a successful
// recurring invoice payment. It updates EndTime to the Stripe period end,
// resets AmountUsed, and recalculates the next reset time. Idempotent: if the
// subscription's EndTime is already at or beyond periodEnd, it is a no-op
// (handles Stripe retrying the same invoice.paid event).
func RenewUserSubscriptionFromStripe(stripeSubId string, periodEnd int64) error {
	if stripeSubId == "" || periodEnd <= 0 {
		return errors.New("invalid renewal args")
	}
	now := common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).Where("stripe_subscription_id = ?", stripeSubId).First(&sub).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		// Idempotency: if EndTime is already at or beyond periodEnd, skip.
		if sub.EndTime >= periodEnd {
			return nil
		}
		sub.EndTime = periodEnd
		sub.AmountUsed = 0
		sub.LastResetTime = now
		plan, err := getSubscriptionPlanByIdTx(tx, sub.PlanId)
		if err != nil {
			return err
		}
		sub.NextResetTime = calcNextResetTime(time.Unix(now, 0), plan, periodEnd)
		if sub.Status == SubscriptionStatusPastDue || sub.Status == SubscriptionStatusPaymentFailed {
			sub.Status = SubscriptionStatusActive
		}
		return tx.Save(&sub).Error
	})
}

// SetUserSubscriptionStatusFromStripe updates a subscription's status and
// cancel_at_period_end flag based on a customer.subscription.updated event.
// Maps Stripe subscription statuses to local statuses. Idempotent.
func SetUserSubscriptionStatusFromStripe(stripeSubId string, stripeStatus string, cancelAtPeriodEnd bool, periodEnd int64) error {
	if stripeSubId == "" {
		return errors.New("invalid stripeSubId")
	}
	localStatus := mapStripeSubscriptionStatus(stripeStatus, cancelAtPeriodEnd)
	if localStatus == "" {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).Where("stripe_subscription_id = ?", stripeSubId).First(&sub).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		updates := map[string]interface{}{
			"status":               localStatus,
			"cancel_at_period_end": cancelAtPeriodEnd,
			"updated_at":           common.GetTimestamp(),
		}
		if periodEnd > 0 && sub.EndTime < periodEnd {
			updates["end_time"] = periodEnd
		}
		return tx.Model(&sub).Updates(updates).Error
	})
}

// MarkUserSubscriptionPastDue sets a subscription's status to past_due after
// a renewal invoice payment fails. Idempotent.
func MarkUserSubscriptionPastDue(stripeSubId string) error {
	if stripeSubId == "" {
		return errors.New("invalid stripeSubId")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).Where("stripe_subscription_id = ?", stripeSubId).First(&sub).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if sub.Status == SubscriptionStatusCanceled || sub.Status == SubscriptionStatusExpired {
			return nil
		}
		return tx.Model(&sub).Updates(map[string]interface{}{
			"status":     SubscriptionStatusPastDue,
			"updated_at": common.GetTimestamp(),
		}).Error
	})
}

// CancelUserSubscriptionFromStripe marks a subscription as canceled after the
// underlying Stripe subscription is deleted. It sets end_time = now, downgrades
// the user group (if applicable), and reverses coupon redemptions. Idempotent.
func CancelUserSubscriptionFromStripe(stripeSubId string) error {
	if stripeSubId == "" {
		return errors.New("invalid stripeSubId")
	}
	now := common.GetTimestamp()
	var userId int
	var subId int
	var cacheGroup string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).Where("stripe_subscription_id = ?", stripeSubId).First(&sub).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if sub.Status == SubscriptionStatusCanceled {
			return nil
		}
		userId = sub.UserId
		subId = sub.Id
		if err := tx.Model(&sub).Updates(map[string]interface{}{
			"status":     SubscriptionStatusCanceled,
			"end_time":   now,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		target, err := downgradeUserGroupForSubscriptionTx(tx, &sub, now)
		if err != nil {
			return err
		}
		if target != "" {
			cacheGroup = target
		}
		return nil
	})
	if err != nil {
		return err
	}
	if cacheGroup != "" && userId > 0 {
		_ = UpdateUserGroupCache(userId, cacheGroup)
	}
	if subId > 0 {
		_, _ = ReverseSubscriptionCouponRedemptionsForSubscription(subId)
	}
	return nil
}

// mapStripeSubscriptionStatus converts a Stripe subscription status to the
// local status vocabulary. Returns "" when the Stripe status should not
// trigger a local update (e.g. "incomplete" is transitional).
func mapStripeSubscriptionStatus(stripeStatus string, cancelAtPeriodEnd bool) string {
	switch stripeStatus {
	case "active":
		if cancelAtPeriodEnd {
			return SubscriptionStatusCanceling
		}
		return SubscriptionStatusActive
	case "past_due":
		return SubscriptionStatusPastDue
	case "canceled", "unpaid":
		return SubscriptionStatusCanceled
	case "trialing":
		return SubscriptionStatusActive
	default:
		return ""
	}
}

// UpdateUserStripeCustomer persists the Stripe Customer ID on the user record
// when it is first seen from a checkout.session.completed or invoice event.
// Idempotent: no-op if the user already has the same or non-empty customer ID.
func UpdateUserStripeCustomer(userId int, customerId string) error {
	if userId <= 0 || customerId == "" {
		return nil
	}
	return DB.Model(&User{}).Where("id = ? AND (stripe_customer = '' OR stripe_customer IS NULL)", userId).
		Update("stripe_customer", customerId).Error
}
