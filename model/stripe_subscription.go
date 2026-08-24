package model

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	// Recurring sandbox plans are platform-wide. Keep the legacy model name so
	// existing plan rows and checkout metadata can be normalized safely.
	SandboxStripeSubscriptionModel                    = ""
	LegacySandboxStripeSubscriptionModel              = "deepseek-v4-flash-0731"
	SandboxStripeSubscriptionAccountID                = "acct_1Tta8mPKe8UWYDw1"
	SandboxStripeSubscriptionProductID                = "prod_V7MVUgAzzCfH20"
	SandboxStripeSubscriptionProductName              = "NovaPuraAI DeepSeek V4 Flash Unlimited"
	SandboxStripeSubscriptionFounderPriceID           = "price_1U77mhPKe8UWYDw1ToMHT22i"
	SandboxStripeSubscriptionStandardPriceID          = "price_1U77miPKe8UWYDw1DEov9KXK"
	SandboxStripeSubscriptionFounderLookupKey         = "novapura_deepseek_v4_flash_founder_cny_monthly_v1"
	SandboxStripeSubscriptionStandardLookupKey        = "novapura_deepseek_v4_flash_standard_cny_monthly_v1"
	SandboxStripeSubscriptionPortalConfigurationID    = "bpc_1U77nfPKe8UWYDw1x4tvkHey"
	ProductionStripeSubscriptionAccountID             = "acct_1Tta8ePDiyyu7sie"
	ProductionStripeSubscriptionProductID             = "prod_V7eIrR5wouPVZG"
	ProductionStripeSubscriptionFounderPriceID        = "price_1U7P0BPDiyyu7sieXfQsXuEp"
	ProductionStripeSubscriptionStandardPriceID       = "price_1U7P0CPDiyyu7sieMXRzW3Sf"
	ProductionStripeSubscriptionPortalConfigurationID = "bpc_1U7P0CPDiyyu7sieCZtSReJp"
	SandboxStripeSubscriptionPlanCode                 = "novapura-deepseek-v4-flash-unlimited"
	SandboxStripeSubscriptionGroup                    = "deepseek-v4-flash-unlimited"
	SandboxStripeSubscriptionEnvironment              = "test"
	ProductionStripeSubscriptionEnvironment           = "production"
	SandboxStripeSubscriptionCurrency                 = "cny"
	SandboxStripeSubscriptionBillingInterval          = "month"
	StripeSubscriptionReservationPending              = "pending"
	StripeSubscriptionReservationActive               = "active"
	StripeSubscriptionReservationExpired              = "expired"
	StripeSubscriptionReservationReleased             = "released"
	StripeSubscriptionReservationReconciliation       = "reconciliation_required"
	StripeSubscriptionRemoteSessionPending            = "pending"
	StripeSubscriptionRemoteSessionCreated            = "created"
	StripeSubscriptionRemoteSessionExpired            = "expired"
	StripeSubscriptionRemoteSessionReconciliation     = "reconciliation_required"
	StripeSubscriptionTierFounder                     = "founder"
	StripeSubscriptionTierStandard                    = "standard"
	StripeSubscriptionStatusIncomplete                = "incomplete"
	StripeSubscriptionStatusActive                    = "active"
	StripeSubscriptionStatusPastDue                   = "past_due"
	StripeSubscriptionStatusCanceled                  = "canceled"
	StripeSubscriptionStatusUnpaid                    = "unpaid"
	StripeSubscriptionReservationTTL                  = 30 * time.Minute
	StripeSubscriptionGracePeriod                     = 72 * time.Hour
)

const (
	stripeSubscriptionEnabledEnv             = "STRIPE_SUBSCRIPTION_ENABLED"
	stripeSubscriptionAccountIDEnv           = "STRIPE_SUBSCRIPTION_ACCOUNT_ID"
	stripeSubscriptionProductIDEnv           = "STRIPE_SUBSCRIPTION_PRODUCT_ID"
	stripeSubscriptionFounderPriceIDEnv      = "STRIPE_SUBSCRIPTION_FOUNDER_PRICE_ID"
	stripeSubscriptionStandardPriceIDEnv     = "STRIPE_SUBSCRIPTION_STANDARD_PRICE_ID"
	stripeSubscriptionPortalConfigurationEnv = "STRIPE_SUBSCRIPTION_PORTAL_CONFIGURATION_ID"
)

var (
	ErrSubscriptionCapacityFull             = errors.New("subscription capacity full")
	ErrSubscriptionAlreadyActive            = errors.New("subscription already active")
	ErrSubscriptionAlreadyPending           = errors.New("subscription checkout already pending")
	ErrStripeSubscriptionDisabled           = errors.New("stripe subscription disabled")
	ErrStripeSubscriptionPlanInvalid        = errors.New("stripe subscription plan invalid")
	ErrStripeSubscriptionReservation        = errors.New("stripe subscription reservation invalid")
	ErrStripeSubscriptionReservationExpired = errors.New("stripe subscription reservation expired")
	ErrStripeSubscriptionFounderClaimUsed   = errors.New("stripe subscription founder claim already used")
	ErrStripeSubscriptionFounderSoldOut     = errors.New("stripe subscription founder claims exhausted")
	ErrStripeSubscriptionEnded              = errors.New("stripe subscription already ended")
	stripeSubscriptionSQLiteSeatMu          sync.Mutex
)

// StripeSubscriptionConfig contains non-secret, environment-specific subscription wiring.
// Enabled intentionally defaults to false; activation is an explicit runtime/config action.
type StripeSubscriptionConfig struct {
	Enabled               bool   `json:"enabled"`
	Environment           string `json:"environment"`
	AccountID             string `json:"account_id"`
	ProductID             string `json:"product_id"`
	ProductName           string `json:"product_name"`
	FounderPriceID        string `json:"founder_price_id"`
	StandardPriceID       string `json:"standard_price_id"`
	FounderLookupKey      string `json:"founder_lookup_key"`
	StandardLookupKey     string `json:"standard_lookup_key"`
	PortalConfigurationID string `json:"portal_configuration_id"`
	FounderAmountMinor    int64  `json:"founder_amount_minor"`
	StandardAmountMinor   int64  `json:"standard_amount_minor"`
	Currency              string `json:"currency"`
	BillingInterval       string `json:"billing_interval"`
	Model                 string `json:"model"`
	ModelScope            string `json:"model_scope"`
	MaxActiveSeats        int    `json:"max_active_seats"`
	FounderPurchaseLimit  int    `json:"founder_purchase_limit"`
	MaxActivePerUser      int    `json:"max_active_per_user"`
	PlanCode              string `json:"plan_code"`
	PlanTitle             string `json:"plan_title"`
	UpgradeGroup          string `json:"upgrade_group"`
}

func DefaultSandboxStripeSubscriptionConfig() StripeSubscriptionConfig {
	return StripeSubscriptionConfig{
		Enabled:               false,
		Environment:           SandboxStripeSubscriptionEnvironment,
		AccountID:             SandboxStripeSubscriptionAccountID,
		ProductID:             SandboxStripeSubscriptionProductID,
		ProductName:           SandboxStripeSubscriptionProductName,
		FounderPriceID:        SandboxStripeSubscriptionFounderPriceID,
		StandardPriceID:       SandboxStripeSubscriptionStandardPriceID,
		FounderLookupKey:      SandboxStripeSubscriptionFounderLookupKey,
		StandardLookupKey:     SandboxStripeSubscriptionStandardLookupKey,
		PortalConfigurationID: SandboxStripeSubscriptionPortalConfigurationID,
		FounderAmountMinor:    1999,
		StandardAmountMinor:   9999,
		Currency:              SandboxStripeSubscriptionCurrency,
		BillingInterval:       SandboxStripeSubscriptionBillingInterval,
		Model:                 SandboxStripeSubscriptionModel,
		ModelScope:            StripeSubscriptionModelScopeAll,
		MaxActiveSeats:        20,
		FounderPurchaseLimit:  20,
		MaxActivePerUser:      1,
		PlanCode:              SandboxStripeSubscriptionPlanCode,
		PlanTitle:             SandboxStripeSubscriptionProductName,
		UpgradeGroup:          SandboxStripeSubscriptionGroup,
	}
}

func stripeSubscriptionModelMatchesConfig(plan *SubscriptionPlan, config StripeSubscriptionConfig) bool {
	if plan == nil {
		return false
	}

	storedModel := strings.TrimSpace(plan.StripeSubscriptionModel)
	if strings.EqualFold(strings.TrimSpace(config.ModelScope), StripeSubscriptionModelScopeAll) ||
		(strings.TrimSpace(config.ModelScope) == "" && strings.TrimSpace(config.Model) == "") {
		return storedModel == "" || storedModel == LegacySandboxStripeSubscriptionModel
	}

	return storedModel == strings.TrimSpace(config.Model)
}

func DefaultProductionStripeSubscriptionConfig() StripeSubscriptionConfig {
	config := DefaultSandboxStripeSubscriptionConfig()
	config.Environment = ProductionStripeSubscriptionEnvironment
	config.AccountID = ProductionStripeSubscriptionAccountID
	config.ProductID = ProductionStripeSubscriptionProductID
	config.FounderPriceID = ProductionStripeSubscriptionFounderPriceID
	config.StandardPriceID = ProductionStripeSubscriptionStandardPriceID
	config.PortalConfigurationID = ProductionStripeSubscriptionPortalConfigurationID
	return config
}

// StripeSubscriptionEnvironmentForRuntime follows the existing GIN_MODE
// boundary used by the Stripe credential profile. Any non-release runtime is
// intentionally treated as the isolated test catalog.
func StripeSubscriptionEnvironmentForRuntime() string {
	if common.GetEnvOrDefaultString("GIN_MODE", "") == "release" {
		return ProductionStripeSubscriptionEnvironment
	}
	return SandboxStripeSubscriptionEnvironment
}

func applyStripeSubscriptionEnvironmentOverrides(config StripeSubscriptionConfig) (StripeSubscriptionConfig, error) {
	config.Enabled = common.GetEnvOrDefaultBool(stripeSubscriptionEnabledEnv, false)
	overrides := []struct {
		env      string
		expected string
	}{
		{stripeSubscriptionAccountIDEnv, config.AccountID},
		{stripeSubscriptionProductIDEnv, config.ProductID},
		{stripeSubscriptionFounderPriceIDEnv, config.FounderPriceID},
		{stripeSubscriptionStandardPriceIDEnv, config.StandardPriceID},
		{stripeSubscriptionPortalConfigurationEnv, config.PortalConfigurationID},
	}
	for _, override := range overrides {
		value, configured := common.LookupEnv(override.env)
		if !configured {
			continue
		}
		if value != override.expected {
			return StripeSubscriptionConfig{}, fmt.Errorf(
				"%w: %s override does not match the fixed %s Stripe subscription catalog",
				ErrStripeSubscriptionPlanInvalid,
				override.env,
				config.Environment,
			)
		}
	}
	return config, nil
}

func StripeSubscriptionConfigForEnvironment(environment string) (StripeSubscriptionConfig, error) {
	var config StripeSubscriptionConfig
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case SandboxStripeSubscriptionEnvironment:
		config = DefaultSandboxStripeSubscriptionConfig()
	case ProductionStripeSubscriptionEnvironment:
		config = DefaultProductionStripeSubscriptionConfig()
	default:
		return StripeSubscriptionConfig{}, fmt.Errorf("%w: unsupported Stripe runtime environment %q", ErrStripeSubscriptionPlanInvalid, environment)
	}
	return applyStripeSubscriptionEnvironmentOverrides(config)
}

// StripeSubscriptionRuntimePreflight resolves the complete non-secret
// recurring runtime contract once for startup/migration. Callers that perform
// broad migrations should pass the returned config onward instead of reading
// the environment again after database mutations begin.
func StripeSubscriptionRuntimePreflight() (StripeSubscriptionConfig, error) {
	return StripeSubscriptionConfigForEnvironment(StripeSubscriptionEnvironmentForRuntime())
}

// ValidateSandboxStripeSubscriptionPlan retains the test-catalog compatibility
// entry point used by existing callers.
func ValidateSandboxStripeSubscriptionPlan(plan *SubscriptionPlan, requireEnabled bool) error {
	config, err := StripeSubscriptionConfigForEnvironment(SandboxStripeSubscriptionEnvironment)
	if err != nil {
		return err
	}
	return ValidateStripeSubscriptionPlan(plan, config, requireEnabled)
}

func ValidateStripeSubscriptionPlan(plan *SubscriptionPlan, config StripeSubscriptionConfig, requireEnabled bool) error {
	if requireEnabled && !config.Enabled {
		return ErrStripeSubscriptionDisabled
	}
	if plan == nil {
		return fmt.Errorf("%w: plan missing", ErrStripeSubscriptionPlanInvalid)
	}
	if strings.TrimSpace(plan.Code) != config.PlanCode || plan.RecurringCode == nil || strings.TrimSpace(*plan.RecurringCode) != config.PlanCode {
		return fmt.Errorf("%w: stable recurring plan code mismatch", ErrStripeSubscriptionPlanInvalid)
	}
	if requireEnabled && (!plan.Enabled || !plan.StripeSubscriptionEnabled) {
		return ErrStripeSubscriptionDisabled
	}
	if plan.PriceAmount != 19.99 || strings.ToLower(strings.TrimSpace(plan.Currency)) != config.Currency ||
		plan.DurationUnit != config.BillingInterval || plan.DurationValue != 1 ||
		!stripeSubscriptionModelMatchesConfig(plan, config) || strings.TrimSpace(plan.UpgradeGroup) != config.UpgradeGroup ||
		plan.MaxActiveSubscriptions != config.MaxActiveSeats || plan.FounderPurchaseLimit != config.FounderPurchaseLimit ||
		plan.MaxActivePerUser != config.MaxActivePerUser || plan.FounderStripePriceId != config.FounderPriceID ||
		plan.StandardStripePriceId != config.StandardPriceID || plan.FounderAmountMinor != config.FounderAmountMinor ||
		plan.StandardAmountMinor != config.StandardAmountMinor || strings.ToLower(strings.TrimSpace(plan.StripeCurrency)) != config.Currency ||
		plan.StripeProductId != config.ProductID || plan.StripeAccountId != config.AccountID ||
		plan.StripePortalConfigurationId != config.PortalConfigurationID || plan.TotalAmount != 0 ||
		plan.AllowBalancePay == nil || *plan.AllowBalancePay || plan.AllowWalletOverflow == nil || *plan.AllowWalletOverflow {
		return fmt.Errorf("%w: exact %s offer configuration required", ErrStripeSubscriptionPlanInvalid, config.Environment)
	}
	return nil
}

// GetFixedStripeSubscriptionPlan selects only the runtime contract's stable
// code pair. It never treats an unrelated enabled recurring row as the offer.
// requireEnabled=false is for existing lifecycle/diagnostic validation; it
// still performs the complete fixed-catalog contract check.
func GetFixedStripeSubscriptionPlan(config StripeSubscriptionConfig, requireEnabled bool) (*SubscriptionPlan, error) {
	if DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	var plan SubscriptionPlan
	if err := DB.Where("code = ? AND recurring_code = ?", config.PlanCode, config.PlanCode).First(&plan).Error; err != nil {
		return nil, err
	}
	plan.NormalizeDefaults()
	if err := ValidateStripeSubscriptionPlan(&plan, config, requireEnabled); err != nil {
		return nil, err
	}
	return &plan, nil
}

// EnsureSandboxStripeSubscriptionPlan retains the test-catalog compatibility
// entry point used by existing callers.
func EnsureSandboxStripeSubscriptionPlan() error {
	return EnsureStripeSubscriptionPlan(DefaultSandboxStripeSubscriptionConfig())
}

func EnsureStripeSubscriptionPlanForEnvironment(environment string) error {
	config, err := StripeSubscriptionConfigForEnvironment(environment)
	if err != nil {
		return err
	}
	return EnsureStripeSubscriptionPlan(config)
}

func EnsureStripeSubscriptionPlan(config StripeSubscriptionConfig) error {
	if DB == nil {
		return ErrStripeSubscriptionPlanInvalid
	}
	var invalidatedPlanID int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var plans []SubscriptionPlan
		if err := lockForUpdate(tx).
			Where("code = ? OR recurring_code = ?", config.PlanCode, config.PlanCode).
			Limit(2).Find(&plans).Error; err != nil {
			return err
		}
		if len(plans) > 1 {
			return fmt.Errorf("%w: duplicate %s subscription plan code %q", ErrStripeSubscriptionPlanInvalid, config.Environment, config.PlanCode)
		}
		if len(plans) == 1 {
			plan := &plans[0]
			plan.NormalizeDefaults()
			if strings.TrimSpace(plan.Code) != config.PlanCode ||
				(plan.RecurringCode != nil && strings.TrimSpace(*plan.RecurringCode) != config.PlanCode) ||
				!stripeSubscriptionModelMatchesConfig(plan, config) || plan.PriceAmount != 19.99 ||
				strings.ToLower(strings.TrimSpace(plan.Currency)) != config.Currency ||
				plan.DurationUnit != config.BillingInterval || plan.DurationValue != 1 ||
				plan.MaxActiveSubscriptions != config.MaxActiveSeats || plan.FounderPurchaseLimit != config.FounderPurchaseLimit ||
				plan.MaxActivePerUser != config.MaxActivePerUser || plan.FounderStripePriceId != config.FounderPriceID ||
				plan.StandardStripePriceId != config.StandardPriceID || plan.FounderAmountMinor != config.FounderAmountMinor ||
				plan.StandardAmountMinor != config.StandardAmountMinor || strings.ToLower(strings.TrimSpace(plan.StripeCurrency)) != config.Currency ||
				plan.StripeProductId != config.ProductID || plan.StripeAccountId != config.AccountID ||
				plan.StripePortalConfigurationId != config.PortalConfigurationID || strings.TrimSpace(plan.UpgradeGroup) != config.UpgradeGroup ||
				plan.AllowBalancePay == nil || *plan.AllowBalancePay || plan.TotalAmount != 0 || plan.AllowWalletOverflow == nil || *plan.AllowWalletOverflow {
				return fmt.Errorf("%w: %s subscription plan %q conflicts with fixed Stripe contract", ErrStripeSubscriptionPlanInvalid, config.Environment, config.PlanCode)
			}
			updates := map[string]any{}
			if strings.TrimSpace(plan.StripeSubscriptionModel) != strings.TrimSpace(config.Model) {
				updates["stripe_subscription_model"] = config.Model
			}
			if plan.RecurringCode == nil {
				updates["recurring_code"] = config.PlanCode
			}
			if plan.Enabled != config.Enabled {
				updates["enabled"] = config.Enabled
			}
			if plan.StripeSubscriptionEnabled != config.Enabled {
				updates["stripe_subscription_enabled"] = config.Enabled
			}
			if len(updates) == 0 {
				return nil
			}
			if err := tx.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Updates(updates).Error; err != nil {
				return err
			}
			invalidatedPlanID = plan.Id
			return nil
		}
		recurringCode := config.PlanCode
		plan := &SubscriptionPlan{
			Code:                        config.PlanCode,
			RecurringCode:               &recurringCode,
			Title:                       config.PlanTitle,
			Subtitle:                    "Unlimited platform-model tokens with fair-use protection.",
			PriceAmount:                 19.99,
			Currency:                    "CNY",
			DurationUnit:                SubscriptionDurationMonth,
			DurationValue:               1,
			Enabled:                     config.Enabled,
			AllowBalancePay:             common.GetPointer(false),
			AllowWalletOverflow:         common.GetPointer(false),
			StripeSubscriptionEnabled:   config.Enabled,
			StripeSubscriptionModel:     config.Model,
			MaxActiveSubscriptions:      config.MaxActiveSeats,
			FounderPurchaseLimit:        config.FounderPurchaseLimit,
			MaxActivePerUser:            config.MaxActivePerUser,
			FounderStripePriceId:        config.FounderPriceID,
			StandardStripePriceId:       config.StandardPriceID,
			FounderAmountMinor:          config.FounderAmountMinor,
			StandardAmountMinor:         config.StandardAmountMinor,
			StripeCurrency:              config.Currency,
			StripeProductId:             config.ProductID,
			StripeAccountId:             config.AccountID,
			StripePortalConfigurationId: config.PortalConfigurationID,
			UpgradeGroup:                config.UpgradeGroup,
			TotalAmount:                 0,
		}
		if err := tx.Create(plan).Error; err != nil {
			return err
		}
		// SubscriptionPlan.Enabled has a legacy `default:true` tag. GORM treats a
		// false zero value as omitted on INSERT, so persist both feature flags
		// explicitly in the same transaction before the row can become visible.
		if err := tx.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Updates(map[string]any{
			"enabled":                     config.Enabled,
			"stripe_subscription_enabled": config.Enabled,
			"allow_balance_pay":           false,
			"allow_wallet_overflow":       false,
		}).Error; err != nil {
			return err
		}
		invalidatedPlanID = plan.Id
		return nil
	})
	if err != nil {
		return err
	}
	if invalidatedPlanID > 0 {
		InvalidateSubscriptionPlanCache(invalidatedPlanID)
	}
	return nil
}

// StripeSubscriptionReservation tracks a single subscriber account's pending or active seat.
// The row is retained after release/expiry so checkout and founder-claim history is auditable.
type StripeSubscriptionReservation struct {
	Id                   int64  `json:"id" gorm:"primaryKey"`
	PlanId               int    `json:"plan_id" gorm:"not null;index:idx_stripe_sub_reservation_plan_status"`
	UserId               int    `json:"user_id" gorm:"not null;index:idx_stripe_sub_reservation_user_status"`
	ActiveUserId         *int   `json:"-" gorm:"uniqueIndex:idx_stripe_sub_reservation_active_user"`
	ReferenceId          string `json:"reference_id" gorm:"type:varchar(128);not null;uniqueIndex"`
	IdempotencyKey       string `json:"-" gorm:"type:varchar(128);uniqueIndex"`
	CheckoutSessionId    string `json:"checkout_session_id" gorm:"type:varchar(128);index"`
	CheckoutURL          string `json:"checkout_url,omitempty" gorm:"type:varchar(512)"`
	RemoteSessionStatus  string `json:"remote_session_status" gorm:"type:varchar(32);index"`
	RemoteSessionError   string `json:"-" gorm:"type:varchar(255)"`
	StripeCustomerId     string `json:"stripe_customer_id" gorm:"type:varchar(128);index"`
	StripeSubscriptionId string `json:"stripe_subscription_id" gorm:"type:varchar(128);index"`
	StripePriceId        string `json:"stripe_price_id" gorm:"type:varchar(128)"`
	Tier                 string `json:"tier" gorm:"type:varchar(16);not null"`
	Status               string `json:"status" gorm:"type:varchar(16);not null;index:idx_stripe_sub_reservation_plan_status;index:idx_stripe_sub_reservation_user_status"`
	ExpiresAt            int64  `json:"expires_at" gorm:"bigint;index"`
	ActivatedAt          int64  `json:"activated_at" gorm:"bigint"`
	ReleasedAt           int64  `json:"released_at" gorm:"bigint"`
	FounderClaimId       int64  `json:"founder_claim_id" gorm:"bigint;index"`
	CreatedAt            int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt            int64  `json:"updated_at" gorm:"bigint"`
}

func (StripeSubscriptionReservation) TableName() string {
	return "stripe_subscription_reservations"
}

func stripeSubscriptionReservationHoldsUserSeat(status string) bool {
	switch strings.TrimSpace(status) {
	case StripeSubscriptionReservationPending, StripeSubscriptionReservationActive, StripeSubscriptionReservationReconciliation:
		return true
	default:
		return false
	}
}

func (r *StripeSubscriptionReservation) BeforeCreate(tx *gorm.DB) error {
	if stripeSubscriptionReservationHoldsUserSeat(r.Status) {
		userID := r.UserId
		r.ActiveUserId = &userID
	} else {
		r.ActiveUserId = nil
	}
	if strings.TrimSpace(r.IdempotencyKey) == "" && strings.TrimSpace(r.ReferenceId) != "" {
		r.IdempotencyKey = "novapura_sub_checkout_" + strings.TrimSpace(r.ReferenceId)
	}
	if strings.TrimSpace(r.RemoteSessionStatus) == "" {
		r.RemoteSessionStatus = StripeSubscriptionRemoteSessionPending
	}
	now := common.GetTimestamp()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	if r.UpdatedAt == 0 {
		r.UpdatedAt = now
	}
	return nil
}

func (r *StripeSubscriptionReservation) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

// ensureStripeSubscriptionReservationActiveUsers repairs the derived nullable
// unique seat key after schema migration. Multiple live reservations for one
// account are ambiguous billing state, so startup fails closed instead of
// choosing one silently.
func ensureStripeSubscriptionReservationActiveUsers() error {
	if DB == nil {
		return ErrStripeSubscriptionReservation
	}
	now := common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		var live []StripeSubscriptionReservation
		if err := tx.Where(
			"status = ? OR status = ? OR (status = ? AND (expires_at = 0 OR expires_at > ?))",
			StripeSubscriptionReservationActive,
			StripeSubscriptionReservationReconciliation,
			StripeSubscriptionReservationPending,
			now,
		).Order("id ASC").Find(&live).Error; err != nil {
			return err
		}
		seen := make(map[int]int64, len(live))
		for _, reservation := range live {
			if previousID, exists := seen[reservation.UserId]; exists {
				return fmt.Errorf("%w: user %d has live reservations %d and %d", ErrStripeSubscriptionReservation, reservation.UserId, previousID, reservation.Id)
			}
			seen[reservation.UserId] = reservation.Id
		}
		if err := tx.Model(&StripeSubscriptionReservation{}).
			Where("active_user_id IS NOT NULL").
			Update("active_user_id", nil).Error; err != nil {
			return err
		}
		for _, reservation := range live {
			if err := tx.Model(&StripeSubscriptionReservation{}).
				Where("id = ?", reservation.Id).
				Update("active_user_id", reservation.UserId).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// StripeSubscriptionFounderClaim is permanent evidence that a distinct account consumed a founder claim.
type StripeSubscriptionFounderClaim struct {
	Id                 int64 `json:"id" gorm:"primaryKey"`
	PlanId             int   `json:"plan_id" gorm:"not null;index:idx_stripe_sub_founder_claim_user,unique"`
	UserId             int   `json:"user_id" gorm:"not null;index:idx_stripe_sub_founder_claim_user,unique"`
	FirstReservationId int64 `json:"first_reservation_id" gorm:"bigint;index"`
	ClaimedAt          int64 `json:"claimed_at" gorm:"bigint"`
	CreatedAt          int64 `json:"created_at" gorm:"bigint"`
}

func (StripeSubscriptionFounderClaim) TableName() string {
	return "stripe_subscription_founder_claims"
}

func (c *StripeSubscriptionFounderClaim) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if c.ClaimedAt == 0 {
		c.ClaimedAt = now
	}
	if c.CreatedAt == 0 {
		c.CreatedAt = now
	}
	return nil
}

// StripeSubscription links Stripe's recurring object to one local entitlement.
type StripeSubscription struct {
	Id                      int64  `json:"id" gorm:"primaryKey"`
	PlanId                  int    `json:"plan_id" gorm:"not null;index"`
	UserId                  int    `json:"user_id" gorm:"not null;index"`
	ReservationId           int64  `json:"reservation_id" gorm:"bigint;uniqueIndex"`
	StripeCustomerId        string `json:"stripe_customer_id" gorm:"type:varchar(128);not null;index"`
	StripeSubscriptionId    string `json:"stripe_subscription_id" gorm:"type:varchar(128);not null;uniqueIndex"`
	StripeCheckoutSessionId string `json:"stripe_checkout_session_id" gorm:"type:varchar(128);index"`
	StripePriceId           string `json:"stripe_price_id" gorm:"type:varchar(128);not null"`
	UserSubscriptionId      int    `json:"user_subscription_id" gorm:"index"`
	Tier                    string `json:"tier" gorm:"type:varchar(16);not null"`
	Status                  string `json:"status" gorm:"type:varchar(32);not null;index"`
	CancelAtPeriodEnd       bool   `json:"cancel_at_period_end"`
	CurrentPeriodStart      int64  `json:"current_period_start" gorm:"bigint"`
	CurrentPeriodEnd        int64  `json:"current_period_end" gorm:"bigint;index"`
	GraceUntil              int64  `json:"grace_until" gorm:"bigint;index"`
	LatestInvoiceId         string `json:"latest_invoice_id" gorm:"type:varchar(128);index"`
	FailureReason           string `json:"failure_reason" gorm:"type:varchar(255)"`
	EndedAt                 int64  `json:"ended_at" gorm:"bigint"`
	CreatedAt               int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt               int64  `json:"updated_at" gorm:"bigint"`
}

func (StripeSubscription) TableName() string {
	return "stripe_subscriptions"
}

func (s *StripeSubscription) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if s.CreatedAt == 0 {
		s.CreatedAt = now
	}
	if s.UpdatedAt == 0 {
		s.UpdatedAt = now
	}
	return nil
}

func (s *StripeSubscription) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = common.GetTimestamp()
	return nil
}

// StripeSubscriptionInvoice is the invoice-level idempotency ledger for recurring payments.
type StripeSubscriptionInvoice struct {
	Id                   int64  `json:"id" gorm:"primaryKey"`
	PlanId               int    `json:"plan_id" gorm:"not null;index"`
	UserId               int    `json:"user_id" gorm:"not null;index"`
	StripeSubscriptionId string `json:"stripe_subscription_id" gorm:"type:varchar(128);not null;index:idx_stripe_sub_invoice_key,unique"`
	StripeInvoiceId      string `json:"stripe_invoice_id" gorm:"type:varchar(128);not null;index:idx_stripe_sub_invoice_key,unique"`
	EventId              string `json:"event_id" gorm:"type:varchar(128);uniqueIndex"`
	PeriodStart          int64  `json:"period_start" gorm:"bigint"`
	PeriodEnd            int64  `json:"period_end" gorm:"bigint"`
	AmountPaidMinor      int64  `json:"amount_paid_minor" gorm:"bigint"`
	Currency             string `json:"currency" gorm:"type:varchar(8)"`
	Status               string `json:"status" gorm:"type:varchar(32);index"`
	AppliedAt            int64  `json:"applied_at" gorm:"bigint"`
	CreatedAt            int64  `json:"created_at" gorm:"bigint"`
}

func (StripeSubscriptionInvoice) TableName() string {
	return "stripe_subscription_invoices"
}

func (i *StripeSubscriptionInvoice) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if i.AppliedAt == 0 {
		i.AppliedAt = now
	}
	if i.CreatedAt == 0 {
		i.CreatedAt = now
	}
	return nil
}

type StripeSubscriptionInvoiceInput struct {
	PlanID               int
	UserID               int
	StripeSubscriptionID string
	StripeInvoiceID      string
	EventID              string
	PeriodStart          int64
	PeriodEnd            int64
	AmountPaidMinor      int64
	Currency             string
	Status               string
}

func recurringPlanTx(tx *gorm.DB, planID int) (*SubscriptionPlan, error) {
	if tx == nil || planID <= 0 {
		return nil, ErrStripeSubscriptionPlanInvalid
	}
	var plan SubscriptionPlan
	if err := lockForUpdate(tx).Where("id = ?", planID).First(&plan).Error; err != nil {
		return nil, err
	}
	plan.NormalizeDefaults()
	if !plan.Enabled || !plan.StripeSubscriptionEnabled {
		return nil, ErrStripeSubscriptionDisabled
	}
	if plan.MaxActiveSubscriptions <= 0 || plan.MaxActivePerUser <= 0 {
		return nil, ErrStripeSubscriptionDisabled
	}
	return &plan, nil
}

// recurringLifecyclePlanTx reads the selected plan under the caller's
// transaction lock and validates the fixed catalog contract without applying
// the new-sale gate. A reservation or Stripe subscription is already the
// durable association that authorizes settlement and lifecycle processing;
// disabling new sales must not strand that existing entitlement. The runtime
// catalog/account/environment checks remain mandatory.
func recurringLifecyclePlanTx(tx *gorm.DB, planID int) (*SubscriptionPlan, error) {
	if tx == nil || planID <= 0 {
		return nil, ErrStripeSubscriptionPlanInvalid
	}
	var plan SubscriptionPlan
	if err := lockForUpdate(tx).Where("id = ?", planID).First(&plan).Error; err != nil {
		return nil, err
	}
	plan.NormalizeDefaults()
	config, err := StripeSubscriptionRuntimePreflight()
	if err != nil {
		return nil, err
	}
	if err := ValidateStripeSubscriptionPlan(&plan, config, false); err != nil {
		return nil, err
	}
	return &plan, nil
}

// GetStripeSubscriptionPlan loads a recurring plan without using the legacy
// subscription-plan cache. Recurring checkout and webhook validation must see
// the current database values for the exact Stripe objects they authorize.
func GetStripeSubscriptionPlan(planID int) (*SubscriptionPlan, error) {
	if DB == nil || planID <= 0 {
		return nil, ErrStripeSubscriptionPlanInvalid
	}
	var plan SubscriptionPlan
	if err := DB.Where("id = ?", planID).First(&plan).Error; err != nil {
		return nil, err
	}
	plan.NormalizeDefaults()
	return &plan, nil
}

// HasEnabledStripeSubscriptionPlan is intentionally narrow: it is used only
// to decide whether the shared Stripe webhook endpoint may accept recurring
// events. Detailed sandbox validation still happens before every operation.
func HasEnabledStripeSubscriptionPlan() bool {
	enabled, err := EnabledStripeSubscriptionPlan()
	return err == nil && enabled
}

// EnabledStripeSubscriptionPlan preserves the distinction between a disabled
// feature and a primary-database failure so webhook callers can return a
// retryable 5xx instead of acknowledging an event while state is unavailable.
func EnabledStripeSubscriptionPlan() (bool, error) {
	if DB == nil {
		return false, gorm.ErrInvalidDB
	}
	config, err := StripeSubscriptionRuntimePreflight()
	if err != nil {
		return false, err
	}
	if !config.Enabled {
		return false, nil
	}
	_, err = GetFixedStripeSubscriptionPlan(config, true)
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrStripeSubscriptionDisabled) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// HasStripeSubscriptionLifecyclePlan reports whether an existing fixed
// recurring contract can still reconcile Stripe lifecycle events. It ignores
// the new-sale gate but never skips fixed-catalog validation.
func HasStripeSubscriptionLifecyclePlan() (bool, error) {
	if DB == nil {
		return false, gorm.ErrInvalidDB
	}
	config, err := StripeSubscriptionRuntimePreflight()
	if err != nil {
		return false, err
	}
	_, err = GetFixedStripeSubscriptionPlan(config, false)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func expireStripeSubscriptionReservationsTx(tx *gorm.DB, now int64) (int64, error) {
	if tx == nil {
		return 0, ErrStripeSubscriptionReservation
	}
	result := tx.Model(&StripeSubscriptionReservation{}).
		Where("status = ? AND expires_at > 0 AND expires_at <= ?", StripeSubscriptionReservationPending, now).
		Updates(map[string]any{
			"status":         StripeSubscriptionReservationExpired,
			"active_user_id": nil,
			"released_at":    now,
			"updated_at":     now,
		})
	return result.RowsAffected, result.Error
}

func ExpireStripeSubscriptionReservations(now int64) (int64, error) {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	var expired int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		expired, err = expireStripeSubscriptionReservationsTx(tx, now)
		return err
	})
	return expired, err
}

func ReserveStripeSubscriptionSeat(planID int, userID int, referenceID string, now int64) (*StripeSubscriptionReservation, error) {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	// SQLite skips SELECT FOR UPDATE and deferred transactions can otherwise
	// deadlock while upgrading concurrent capacity reads to reservation writes.
	// Hold the mutex through commit; MySQL/PostgreSQL use the row-lock order in
	// ReserveStripeSubscriptionSeatTx and remain fully database-coordinated.
	if common.MainDatabaseType() == common.DatabaseTypeSQLite {
		stripeSubscriptionSQLiteSeatMu.Lock()
		defer stripeSubscriptionSQLiteSeatMu.Unlock()
	}
	var reservation *StripeSubscriptionReservation
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		reservation, err = ReserveStripeSubscriptionSeatTx(tx, planID, userID, referenceID, now)
		return err
	})
	return reservation, err
}

func ReserveStripeSubscriptionSeatTx(tx *gorm.DB, planID int, userID int, referenceID string, now int64) (*StripeSubscriptionReservation, error) {
	if tx == nil || planID <= 0 || userID <= 0 || strings.TrimSpace(referenceID) == "" {
		return nil, ErrStripeSubscriptionReservation
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	var existingRecurring StripeSubscription
	err := lockForUpdate(tx).
		Where("user_id = ? AND status IN ?", userID, []string{
			StripeSubscriptionStatusIncomplete,
			StripeSubscriptionStatusActive,
			StripeSubscriptionStatusPastDue,
			StripeSubscriptionStatusUnpaid,
		}).Order("id DESC").First(&existingRecurring).Error
	if err == nil {
		return nil, ErrSubscriptionAlreadyActive
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// The derived active_user_id key is the database-level guard against two
	// concurrent Checkout reservations for one account. Reclaim an expired key
	// here as well as in the webhook/cleanup paths so a delayed or lost
	// checkout.session.expired event cannot strand the user's only slot.
	var heldUserSeat StripeSubscriptionReservation
	err = lockForUpdate(tx).
		Where("active_user_id = ?", userID).
		Order("id DESC").First(&heldUserSeat).Error
	if err == nil {
		switch heldUserSeat.Status {
		case StripeSubscriptionReservationPending:
			if heldUserSeat.ExpiresAt == 0 || heldUserSeat.ExpiresAt > now {
				return nil, ErrSubscriptionAlreadyPending
			}
			if err := tx.Model(&heldUserSeat).Updates(map[string]any{
				"status":         StripeSubscriptionReservationExpired,
				"active_user_id": nil,
				"released_at":    now,
				"updated_at":     now,
			}).Error; err != nil {
				return nil, err
			}
		case StripeSubscriptionReservationActive:
			return nil, ErrSubscriptionAlreadyActive
		case StripeSubscriptionReservationReconciliation:
			return nil, ErrSubscriptionAlreadyPending
		case StripeSubscriptionReservationReleased, StripeSubscriptionReservationExpired:
			if err := tx.Model(&heldUserSeat).Updates(map[string]any{
				"active_user_id": nil,
				"updated_at":     now,
			}).Error; err != nil {
				return nil, err
			}
		default:
			return nil, ErrStripeSubscriptionReservation
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var existing StripeSubscriptionReservation
	err = lockForUpdate(tx).
		Where("user_id = ? AND (status = ? OR status = ? OR (status = ? AND (expires_at = 0 OR expires_at > ?)))",
			userID,
			StripeSubscriptionReservationActive,
			StripeSubscriptionReservationReconciliation,
			StripeSubscriptionReservationPending,
			now,
		).
		Order("id DESC").First(&existing).Error
	if err == nil {
		if existing.Status == StripeSubscriptionReservationActive {
			return nil, ErrSubscriptionAlreadyActive
		}
		return nil, ErrSubscriptionAlreadyPending
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Every recurring checkout/lifecycle transaction acquires rows in the same
	// order: StripeSubscription, reservation, then plan. The plan lock still
	// serializes capacity and Founder-tier allocation. Expired pending rows are
	// excluded by timestamp and cleaned independently, so cleanup never inverts
	// the lifecycle lock order by touching reservations after the plan row.
	plan, err := recurringPlanTx(tx, planID)
	if err != nil {
		return nil, err
	}

	var activeCount int64
	if err := tx.Model(&StripeSubscriptionReservation{}).
		Where("plan_id = ? AND status = ?", planID, StripeSubscriptionReservationActive).
		Count(&activeCount).Error; err != nil {
		return nil, err
	}
	var pendingCount int64
	if err := tx.Model(&StripeSubscriptionReservation{}).
		Where("plan_id = ? AND ((status = ? AND (expires_at = 0 OR expires_at > ?)) OR status = ?)", planID, StripeSubscriptionReservationPending, now, StripeSubscriptionReservationReconciliation).
		Count(&pendingCount).Error; err != nil {
		return nil, err
	}
	if activeCount+pendingCount >= int64(plan.MaxActiveSubscriptions) {
		return nil, ErrSubscriptionCapacityFull
	}

	// A founder claim is permanent, but a pending checkout must not consume it.
	// Pending founder-tier reservations temporarily hold an available founder
	// slot so concurrent checkouts do not all receive the founder price.
	var existingClaim StripeSubscriptionFounderClaim
	claimErr := lockForUpdate(tx).
		Where("plan_id = ? AND user_id = ?", planID, userID).
		First(&existingClaim).Error
	if claimErr != nil && !errors.Is(claimErr, gorm.ErrRecordNotFound) {
		return nil, claimErr
	}
	var founderTierReservations int64
	if err := tx.Model(&StripeSubscriptionReservation{}).
		Where("plan_id = ? AND tier = ? AND ((status = ? AND (expires_at = 0 OR expires_at > ?)) OR status = ?)", planID, StripeSubscriptionTierFounder, StripeSubscriptionReservationPending, now, StripeSubscriptionReservationReconciliation).
		Count(&founderTierReservations).Error; err != nil {
		return nil, err
	}

	reservation := &StripeSubscriptionReservation{
		PlanId:      planID,
		UserId:      userID,
		ReferenceId: strings.TrimSpace(referenceID),
		Tier:        StripeSubscriptionTierStandard,
		Status:      StripeSubscriptionReservationPending,
		ExpiresAt:   now + int64(StripeSubscriptionReservationTTL/time.Second),
	}
	if errors.Is(claimErr, gorm.ErrRecordNotFound) && plan.FounderPurchaseLimit > 0 {
		var claims int64
		if err := tx.Model(&StripeSubscriptionFounderClaim{}).
			Where("plan_id = ?", planID).
			Count(&claims).Error; err != nil {
			return nil, err
		}
		if claims+founderTierReservations < int64(plan.FounderPurchaseLimit) {
			reservation.Tier = StripeSubscriptionTierFounder
		}
	}
	if err := tx.Create(reservation).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyError(err) {
			return nil, ErrSubscriptionAlreadyPending
		}
		return nil, err
	}
	return reservation, nil
}

func ActivateStripeSubscriptionReservation(reservationID int64, checkoutSessionID string, customerID string, subscriptionID string, priceID string, now int64) (*StripeSubscriptionReservation, error) {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	var reservation *StripeSubscriptionReservation
	err := DB.Transaction(func(tx *gorm.DB) error {
		existing, err := lockStripeSubscriptionRowsTx(tx, subscriptionID, reservationID)
		if err != nil {
			return err
		}
		if existing != nil && (existing.ReservationId != reservationID || existing.StripeSubscriptionId != strings.TrimSpace(subscriptionID)) {
			return ErrStripeSubscriptionPlanInvalid
		}
		reservation, err = ActivateStripeSubscriptionReservationTx(tx, reservationID, checkoutSessionID, customerID, subscriptionID, priceID, now)
		return err
	})
	return reservation, err
}

func ActivateStripeSubscriptionReservationTx(tx *gorm.DB, reservationID int64, checkoutSessionID string, customerID string, subscriptionID string, priceID string, now int64) (*StripeSubscriptionReservation, error) {
	if tx == nil || reservationID <= 0 || strings.TrimSpace(subscriptionID) == "" || strings.TrimSpace(priceID) == "" {
		return nil, ErrStripeSubscriptionReservation
	}
	var reservation StripeSubscriptionReservation
	if err := lockForUpdate(tx).Where("id = ?", reservationID).First(&reservation).Error; err != nil {
		return nil, err
	}
	if reservation.Status == StripeSubscriptionReservationActive {
		if reservation.StripeSubscriptionId == subscriptionID {
			return &reservation, nil
		}
		return nil, ErrSubscriptionAlreadyActive
	}
	if reservation.Status != StripeSubscriptionReservationPending && reservation.Status != StripeSubscriptionReservationReconciliation {
		return nil, ErrStripeSubscriptionReservation
	}
	if reservation.Status == StripeSubscriptionReservationPending && reservation.ExpiresAt > 0 && reservation.ExpiresAt <= now {
		if err := tx.Model(&reservation).Updates(map[string]any{
			"status":         StripeSubscriptionReservationExpired,
			"active_user_id": nil,
			"released_at":    now,
			"updated_at":     now,
		}).Error; err != nil {
			return nil, err
		}
		return nil, ErrStripeSubscriptionReservationExpired
	}
	plan, err := recurringLifecyclePlanTx(tx, reservation.PlanId)
	if err != nil {
		return nil, err
	}
	expectedPriceID := plan.StandardStripePriceId
	if reservation.Tier == StripeSubscriptionTierFounder {
		expectedPriceID = plan.FounderStripePriceId
	}
	if priceID != expectedPriceID {
		// A reservation that was issued as Standard cannot be upgraded by
		// changing the Checkout payload. Preserve the stable sold-out error
		// when the attempted Founder price is rejected because all permanent
		// Founder claims have already been consumed.
		if priceID == plan.FounderStripePriceId && reservation.Tier != StripeSubscriptionTierFounder && plan.FounderPurchaseLimit > 0 {
			var claims int64
			if err := tx.Model(&StripeSubscriptionFounderClaim{}).
				Where("plan_id = ?", reservation.PlanId).Count(&claims).Error; err != nil {
				return nil, err
			}
			if claims >= int64(plan.FounderPurchaseLimit) {
				return nil, ErrStripeSubscriptionFounderSoldOut
			}
		}
		return nil, ErrStripeSubscriptionPlanInvalid
	}

	tier := StripeSubscriptionTierStandard
	var claim StripeSubscriptionFounderClaim
	claimErr := lockForUpdate(tx).Where("plan_id = ? AND user_id = ?", reservation.PlanId, reservation.UserId).First(&claim).Error
	if claimErr == nil && priceID == plan.FounderStripePriceId {
		return nil, ErrStripeSubscriptionFounderClaimUsed
	}
	if errors.Is(claimErr, gorm.ErrRecordNotFound) && priceID == plan.FounderStripePriceId && plan.FounderPurchaseLimit > 0 {
		var claims int64
		if err := tx.Model(&StripeSubscriptionFounderClaim{}).Where("plan_id = ?", reservation.PlanId).Count(&claims).Error; err != nil {
			return nil, err
		}
		if claims < int64(plan.FounderPurchaseLimit) {
			claim = StripeSubscriptionFounderClaim{
				PlanId:             reservation.PlanId,
				UserId:             reservation.UserId,
				FirstReservationId: reservation.Id,
			}
			if err := tx.Create(&claim).Error; err != nil {
				if !(errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyError(err)) {
					return nil, err
				}
				if err := lockForUpdate(tx).Where("plan_id = ? AND user_id = ?", reservation.PlanId, reservation.UserId).First(&claim).Error; err != nil {
					return nil, err
				}
			}
		} else {
			return nil, ErrStripeSubscriptionFounderSoldOut
		}
	} else if claimErr != nil && !errors.Is(claimErr, gorm.ErrRecordNotFound) {
		return nil, claimErr
	}
	if claim.Id > 0 && priceID == plan.FounderStripePriceId {
		tier = StripeSubscriptionTierFounder
	}

	updates := map[string]any{
		"checkout_session_id":    strings.TrimSpace(checkoutSessionID),
		"stripe_subscription_id": strings.TrimSpace(subscriptionID),
		"stripe_price_id":        strings.TrimSpace(priceID),
		"active_user_id":         reservation.UserId,
		"tier":                   tier,
		"status":                 StripeSubscriptionReservationActive,
		"expires_at":             0,
		"activated_at":           now,
		"founder_claim_id":       claim.Id,
		"updated_at":             now,
	}
	if strings.TrimSpace(customerID) != "" {
		updates["stripe_customer_id"] = strings.TrimSpace(customerID)
	}
	if err := tx.Model(&reservation).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := tx.First(&reservation, reservation.Id).Error; err != nil {
		return nil, err
	}
	return &reservation, nil
}

func ReleaseStripeSubscriptionReservation(reservationID int64, now int64) error {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseStripeSubscriptionReservationTx(tx, reservationID, now)
	})
}

// ReleasePendingStripeSubscriptionReservation handles Checkout abandonment.
// It releases only pending/reconciliation capacity and cannot revoke a seat
// that a concurrent paid activation already made active.
func ReleasePendingStripeSubscriptionReservation(reservationID int64, now int64) error {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if reservationID <= 0 {
			return ErrStripeSubscriptionReservation
		}
		var reservation StripeSubscriptionReservation
		if err := lockForUpdate(tx).Where("id = ?", reservationID).First(&reservation).Error; err != nil {
			return err
		}
		switch reservation.Status {
		case StripeSubscriptionReservationActive, StripeSubscriptionReservationReleased, StripeSubscriptionReservationExpired:
			return nil
		case StripeSubscriptionReservationPending, StripeSubscriptionReservationReconciliation:
			return tx.Model(&reservation).Updates(map[string]any{
				"status":         StripeSubscriptionReservationReleased,
				"active_user_id": nil,
				"released_at":    now,
				"updated_at":     now,
			}).Error
		default:
			return ErrStripeSubscriptionReservation
		}
	})
}

func ReleaseStripeSubscriptionReservationTx(tx *gorm.DB, reservationID int64, now int64) error {
	if tx == nil || reservationID <= 0 {
		return ErrStripeSubscriptionReservation
	}
	var reservation StripeSubscriptionReservation
	if err := lockForUpdate(tx).Where("id = ?", reservationID).First(&reservation).Error; err != nil {
		return err
	}
	if reservation.Status == StripeSubscriptionReservationReleased || reservation.Status == StripeSubscriptionReservationExpired {
		return nil
	}
	if err := tx.Model(&reservation).Updates(map[string]any{
		"status":         StripeSubscriptionReservationReleased,
		"active_user_id": nil,
		"released_at":    now,
		"updated_at":     now,
	}).Error; err != nil {
		return err
	}
	return nil
}

func RecordStripeSubscriptionInvoice(input StripeSubscriptionInvoiceInput) (bool, error) {
	var inserted bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		inserted, err = RecordStripeSubscriptionInvoiceTx(tx, input)
		return err
	})
	return inserted, err
}

func RecordStripeSubscriptionInvoiceTx(tx *gorm.DB, input StripeSubscriptionInvoiceInput) (bool, error) {
	if tx == nil || input.PlanID <= 0 || input.UserID <= 0 || strings.TrimSpace(input.StripeSubscriptionID) == "" || strings.TrimSpace(input.StripeInvoiceID) == "" {
		return false, ErrStripeSubscriptionReservation
	}
	var existing StripeSubscriptionInvoice
	err := lockForUpdate(tx).Where("stripe_subscription_id = ? AND stripe_invoice_id = ?", input.StripeSubscriptionID, input.StripeInvoiceID).First(&existing).Error
	if err == nil {
		if strings.EqualFold(existing.Status, "payment_failed") && strings.EqualFold(input.Status, "paid") {
			updates := map[string]any{
				"event_id":          input.EventID,
				"period_start":      input.PeriodStart,
				"period_end":        input.PeriodEnd,
				"amount_paid_minor": input.AmountPaidMinor,
				"currency":          strings.ToLower(strings.TrimSpace(input.Currency)),
				"status":            input.Status,
				"applied_at":        common.GetTimestamp(),
			}
			if err := tx.Model(&existing).Updates(updates).Error; err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	if input.EventID != "" {
		err = lockForUpdate(tx).Where("event_id = ?", input.EventID).First(&existing).Error
		if err == nil {
			return false, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
	}
	invoice := &StripeSubscriptionInvoice{
		PlanId:               input.PlanID,
		UserId:               input.UserID,
		StripeSubscriptionId: input.StripeSubscriptionID,
		StripeInvoiceId:      input.StripeInvoiceID,
		EventId:              input.EventID,
		PeriodStart:          input.PeriodStart,
		PeriodEnd:            input.PeriodEnd,
		AmountPaidMinor:      input.AmountPaidMinor,
		Currency:             strings.ToLower(strings.TrimSpace(input.Currency)),
		Status:               input.Status,
	}
	if err := tx.Create(invoice).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SetStripeSubscriptionCheckoutSessionDetails attaches the remote hosted
// session and marks it created. It is separate from the legacy wrapper so the
// service can persist the URL needed to resume a retry without another Stripe
// session.
func SetStripeSubscriptionCheckoutSessionDetails(reservationID int64, checkoutSessionID string, checkoutURL string, customerID string) error {
	if DB == nil || strings.TrimSpace(checkoutSessionID) == "" {
		return ErrStripeSubscriptionReservation
	}
	now := common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		var reservation StripeSubscriptionReservation
		if err := lockForUpdate(tx).Where("id = ?", reservationID).First(&reservation).Error; err != nil {
			return err
		}
		if reservation.Status != StripeSubscriptionReservationPending && reservation.Status != StripeSubscriptionReservationReconciliation {
			if reservation.CheckoutSessionId == checkoutSessionID {
				return nil
			}
			return ErrStripeSubscriptionReservation
		}
		if reservation.ExpiresAt > 0 && reservation.ExpiresAt <= now && reservation.Status == StripeSubscriptionReservationPending {
			if err := tx.Model(&reservation).Updates(map[string]any{
				"status":         StripeSubscriptionReservationExpired,
				"active_user_id": nil,
				"released_at":    now,
				"updated_at":     now,
			}).Error; err != nil {
				return err
			}
			return ErrStripeSubscriptionReservationExpired
		}
		updates := map[string]any{
			"checkout_session_id":   strings.TrimSpace(checkoutSessionID),
			"checkout_url":          strings.TrimSpace(checkoutURL),
			"remote_session_status": StripeSubscriptionRemoteSessionCreated,
			"remote_session_error":  "",
			"updated_at":            now,
		}
		if strings.TrimSpace(customerID) != "" {
			updates["stripe_customer_id"] = strings.TrimSpace(customerID)
		}
		return tx.Model(&reservation).Updates(updates).Error
	})
}

func MarkStripeSubscriptionCheckoutReconciliation(reservationID int64, sessionID string, checkoutURL string, reason string, now int64) error {
	if DB == nil || reservationID <= 0 {
		return ErrStripeSubscriptionReservation
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	var outcomeErr error
	err := DB.Transaction(func(tx *gorm.DB) error {
		var reservation StripeSubscriptionReservation
		if err := lockForUpdate(tx).Where("id = ?", reservationID).First(&reservation).Error; err != nil {
			return err
		}
		if reservation.Status == StripeSubscriptionReservationPending &&
			reservation.ExpiresAt > 0 && reservation.ExpiresAt <= now {
			if err := tx.Model(&reservation).Updates(map[string]any{
				"status":                StripeSubscriptionReservationExpired,
				"active_user_id":        nil,
				"remote_session_status": StripeSubscriptionRemoteSessionExpired,
				"remote_session_error":  strings.TrimSpace(reason),
				"released_at":           now,
				"updated_at":            now,
			}).Error; err != nil {
				return err
			}
			outcomeErr = ErrStripeSubscriptionReservationExpired
			return nil
		}
		if reservation.Status == StripeSubscriptionReservationActive ||
			reservation.Status == StripeSubscriptionReservationReleased ||
			reservation.Status == StripeSubscriptionReservationExpired {
			// A late remote response must never resurrect a seat that a paid
			// lifecycle or expiry path has already finalized.
			return nil
		}
		updates := map[string]any{
			"status":                StripeSubscriptionReservationReconciliation,
			"active_user_id":        reservation.UserId,
			"remote_session_status": StripeSubscriptionRemoteSessionReconciliation,
			"remote_session_error":  strings.TrimSpace(reason),
			"updated_at":            now,
		}
		if strings.TrimSpace(sessionID) != "" {
			updates["checkout_session_id"] = strings.TrimSpace(sessionID)
		}
		if strings.TrimSpace(checkoutURL) != "" {
			updates["checkout_url"] = strings.TrimSpace(checkoutURL)
		}
		return tx.Model(&reservation).Updates(updates).Error
	})
	if err != nil {
		return err
	}
	return outcomeErr
}

func SetStripeSubscriptionCheckoutSession(reservationID int64, checkoutSessionID string, customerID string) error {
	return SetStripeSubscriptionCheckoutSessionDetails(reservationID, checkoutSessionID, "", customerID)
}

// EnsureStripeSubscriptionReservationIdempotencyKey backfills the stable key
// for reservations created before Checkout idempotency was introduced. The
// row lock makes the read/repair atomic across concurrent retry requests.
func EnsureStripeSubscriptionReservationIdempotencyKey(reservationID int64) (string, error) {
	if DB == nil || reservationID <= 0 {
		return "", ErrStripeSubscriptionReservation
	}
	var key string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var reservation StripeSubscriptionReservation
		if err := lockForUpdate(tx).Where("id = ?", reservationID).First(&reservation).Error; err != nil {
			return err
		}
		key = strings.TrimSpace(reservation.IdempotencyKey)
		if key == "" {
			if strings.TrimSpace(reservation.ReferenceId) == "" {
				return ErrStripeSubscriptionReservation
			}
			key = "novapura_sub_checkout_" + strings.TrimSpace(reservation.ReferenceId)
			if err := tx.Model(&reservation).Updates(map[string]any{
				"idempotency_key": key,
				"updated_at":      common.GetTimestamp(),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return key, err
}

func GetStripeSubscriptionReservationByReference(referenceID string) (*StripeSubscriptionReservation, error) {
	if DB == nil || strings.TrimSpace(referenceID) == "" {
		return nil, ErrStripeSubscriptionReservation
	}
	var reservation StripeSubscriptionReservation
	if err := DB.Where("reference_id = ?", strings.TrimSpace(referenceID)).First(&reservation).Error; err != nil {
		return nil, err
	}
	return &reservation, nil
}

func GetStripeSubscriptionReservationByCheckoutSession(sessionID string) (*StripeSubscriptionReservation, error) {
	if DB == nil || strings.TrimSpace(sessionID) == "" {
		return nil, ErrStripeSubscriptionReservation
	}
	var reservation StripeSubscriptionReservation
	if err := DB.Where("checkout_session_id = ?", strings.TrimSpace(sessionID)).First(&reservation).Error; err != nil {
		return nil, err
	}
	return &reservation, nil
}

func HasStripeSubscriptionFounderClaim(planID int, userID int) (bool, error) {
	if DB == nil || planID <= 0 || userID <= 0 {
		return false, ErrStripeSubscriptionReservation
	}
	var count int64
	if err := DB.Model(&StripeSubscriptionFounderClaim{}).Where("plan_id = ? AND user_id = ?", planID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetStripeSubscriptionByStripeID(stripeSubscriptionID string) (*StripeSubscription, error) {
	if DB == nil || strings.TrimSpace(stripeSubscriptionID) == "" {
		return nil, ErrStripeSubscriptionReservation
	}
	var subscription StripeSubscription
	if err := DB.Where("stripe_subscription_id = ?", strings.TrimSpace(stripeSubscriptionID)).First(&subscription).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

func GetLatestStripeSubscriptionForUser(userID int) (*StripeSubscription, error) {
	if DB == nil || userID <= 0 {
		return nil, ErrStripeSubscriptionReservation
	}
	var subscription StripeSubscription
	if err := DB.Where("user_id = ?", userID).Order("id DESC").First(&subscription).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

// lockStripeSubscriptionRowsTx establishes the first stage of the recurring
// lifecycle lock order. Both unique identities are checked before callers lock
// the reservation and plan rows, and conflicting identities fail closed.
func lockStripeSubscriptionRowsTx(tx *gorm.DB, stripeSubscriptionID string, reservationID int64) (*StripeSubscription, error) {
	if tx == nil {
		return nil, ErrStripeSubscriptionReservation
	}
	stripeSubscriptionID = strings.TrimSpace(stripeSubscriptionID)
	var byStripeID *StripeSubscription
	if stripeSubscriptionID != "" {
		var subscription StripeSubscription
		err := lockForUpdate(tx).
			Where("stripe_subscription_id = ?", stripeSubscriptionID).
			First(&subscription).Error
		if err == nil {
			byStripeID = &subscription
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	var byReservationID *StripeSubscription
	if reservationID > 0 {
		var subscription StripeSubscription
		err := lockForUpdate(tx).
			Where("reservation_id = ?", reservationID).
			First(&subscription).Error
		if err == nil {
			byReservationID = &subscription
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if byStripeID != nil && byReservationID != nil && byStripeID.Id != byReservationID.Id {
		return nil, ErrStripeSubscriptionPlanInvalid
	}
	if byStripeID != nil {
		return byStripeID, nil
	}
	return byReservationID, nil
}

type StripeSubscriptionBindingInput struct {
	ReservationID        int64
	CheckoutSessionID    string
	CustomerID           string
	StripeSubscriptionID string
	StripePriceID        string
}

// BindStripeSubscriptionCheckout records Stripe's subscription object without
// granting the local entitlement. This is required when Checkout completes
// with unpaid/asynchronous payment and also gives invoice.paid a durable local
// anchor when Stripe delivers the invoice event first.
func BindStripeSubscriptionCheckout(input StripeSubscriptionBindingInput) (*StripeSubscription, error) {
	if DB == nil || input.ReservationID <= 0 || strings.TrimSpace(input.StripeSubscriptionID) == "" || strings.TrimSpace(input.StripePriceID) == "" {
		return nil, ErrStripeSubscriptionReservation
	}
	var result *StripeSubscription
	err := DB.Transaction(func(tx *gorm.DB) error {
		existing, err := lockStripeSubscriptionRowsTx(tx, input.StripeSubscriptionID, input.ReservationID)
		if err != nil {
			return err
		}
		var reservation StripeSubscriptionReservation
		if err := lockForUpdate(tx).Where("id = ?", input.ReservationID).First(&reservation).Error; err != nil {
			return err
		}
		if reservation.Status != StripeSubscriptionReservationPending && reservation.Status != StripeSubscriptionReservationReconciliation && reservation.Status != StripeSubscriptionReservationActive {
			return ErrStripeSubscriptionReservation
		}
		plan, err := recurringLifecyclePlanTx(tx, reservation.PlanId)
		if err != nil {
			return err
		}
		expectedPriceID := plan.StandardStripePriceId
		if reservation.Tier == StripeSubscriptionTierFounder {
			expectedPriceID = plan.FounderStripePriceId
		}
		if strings.TrimSpace(input.StripePriceID) != expectedPriceID {
			return ErrStripeSubscriptionPlanInvalid
		}
		if reservation.StripeSubscriptionId != "" && reservation.StripeSubscriptionId != strings.TrimSpace(input.StripeSubscriptionID) {
			return ErrStripeSubscriptionPlanInvalid
		}
		if reservation.StripeCustomerId != "" && strings.TrimSpace(input.CustomerID) != "" && reservation.StripeCustomerId != strings.TrimSpace(input.CustomerID) {
			return ErrStripeSubscriptionPlanInvalid
		}
		reservationUpdates := map[string]any{
			"checkout_session_id":    strings.TrimSpace(input.CheckoutSessionID),
			"stripe_subscription_id": strings.TrimSpace(input.StripeSubscriptionID),
			"stripe_price_id":        strings.TrimSpace(input.StripePriceID),
			"remote_session_status":  StripeSubscriptionRemoteSessionCreated,
			"remote_session_error":   "",
			"updated_at":             common.GetTimestamp(),
		}
		if strings.TrimSpace(input.CustomerID) != "" {
			reservationUpdates["stripe_customer_id"] = strings.TrimSpace(input.CustomerID)
		}
		if err := tx.Model(&reservation).Updates(reservationUpdates).Error; err != nil {
			return err
		}

		if existing != nil {
			if existing.PlanId != reservation.PlanId || existing.UserId != reservation.UserId || existing.ReservationId != reservation.Id || existing.StripePriceId != strings.TrimSpace(input.StripePriceID) {
				return ErrStripeSubscriptionPlanInvalid
			}
			updates := map[string]any{}
			if strings.TrimSpace(input.CheckoutSessionID) != "" && existing.StripeCheckoutSessionId == "" {
				updates["stripe_checkout_session_id"] = strings.TrimSpace(input.CheckoutSessionID)
			}
			if strings.TrimSpace(input.CustomerID) != "" && existing.StripeCustomerId == "" {
				updates["stripe_customer_id"] = strings.TrimSpace(input.CustomerID)
			}
			if len(updates) > 0 {
				if err := tx.Model(existing).Updates(updates).Error; err != nil {
					return err
				}
			}
			result = existing
			return nil
		}
		customerID := strings.TrimSpace(input.CustomerID)
		if customerID == "" {
			customerID = strings.TrimSpace(reservation.StripeCustomerId)
		}
		checkoutSessionID := strings.TrimSpace(input.CheckoutSessionID)
		if checkoutSessionID == "" {
			checkoutSessionID = strings.TrimSpace(reservation.CheckoutSessionId)
		}
		created := &StripeSubscription{
			PlanId:                  reservation.PlanId,
			UserId:                  reservation.UserId,
			ReservationId:           reservation.Id,
			StripeCustomerId:        customerID,
			StripeSubscriptionId:    strings.TrimSpace(input.StripeSubscriptionID),
			StripeCheckoutSessionId: checkoutSessionID,
			StripePriceId:           strings.TrimSpace(input.StripePriceID),
			Tier:                    reservation.Tier,
			Status:                  StripeSubscriptionStatusIncomplete,
		}
		if err := tx.Create(created).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyError(err) {
				return ErrSubscriptionAlreadyActive
			}
			return err
		}
		result = created
		return nil
	})
	return result, err
}

type StripeSubscriptionActivationInput struct {
	ReservationID        int64
	CheckoutSessionID    string
	CustomerID           string
	StripeSubscriptionID string
	StripePriceID        string
	PeriodStart          int64
	PeriodEnd            int64
}

func ensureStripeSubscriptionEntitlementTx(tx *gorm.DB, subscription *StripeSubscription, reservation *StripeSubscriptionReservation, plan *SubscriptionPlan, periodStart int64, periodEnd int64, now int64) error {
	if tx == nil || subscription == nil || reservation == nil || plan == nil {
		return ErrStripeSubscriptionReservation
	}
	if subscription.UserSubscriptionId <= 0 {
		entitlement, err := CreateUserSubscriptionFromPlanAtTx(tx, reservation.UserId, plan, StripeRecurringLifecycleSource, now)
		if err != nil {
			return err
		}
		subscription.UserSubscriptionId = entitlement.Id
		if err := tx.Model(subscription).Updates(map[string]any{
			"user_subscription_id": entitlement.Id,
			"status":               StripeSubscriptionStatusActive,
			"updated_at":           now,
		}).Error; err != nil {
			return err
		}
	}
	return restoreStripeSubscriptionEntitlementTx(tx, subscription.UserSubscriptionId, periodStart, periodEnd, now)
}

// ActivateStripeSubscriptionWithEntitlement atomically turns a paid
// Checkout reservation into exactly one local recurring subscription and one
// UserSubscription entitlement. Replayed completed events are harmless due
// to the Stripe subscription and reservation unique keys.
func ActivateStripeSubscriptionWithEntitlement(input StripeSubscriptionActivationInput) (*StripeSubscription, error) {
	if DB == nil || input.ReservationID <= 0 || strings.TrimSpace(input.StripeSubscriptionID) == "" || strings.TrimSpace(input.StripePriceID) == "" {
		return nil, ErrStripeSubscriptionReservation
	}
	now := common.GetTimestamp()
	var recurring *StripeSubscription
	var cacheUserID int
	var cacheGroup string
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Recurring lifecycle transactions always lock StripeSubscription before
		// reservation and plan rows. This matches reservation, binding, and
		// invoice paths and prevents out-of-order delivery from deadlocking.
		existing, err := lockStripeSubscriptionRowsTx(tx, input.StripeSubscriptionID, input.ReservationID)
		if err != nil {
			return err
		}

		reservation, err := ActivateStripeSubscriptionReservationTx(tx, input.ReservationID, input.CheckoutSessionID, input.CustomerID, input.StripeSubscriptionID, input.StripePriceID, now)
		if err != nil {
			return err
		}
		cacheUserID = reservation.UserId
		plan, err := recurringLifecyclePlanTx(tx, reservation.PlanId)
		if err != nil {
			return err
		}
		cacheGroup = strings.TrimSpace(plan.UpgradeGroup)

		if existing != nil {
			if existing.StripeSubscriptionId != input.StripeSubscriptionID || existing.UserId != reservation.UserId || existing.PlanId != reservation.PlanId || existing.ReservationId != reservation.Id {
				return ErrStripeSubscriptionPlanInvalid
			}
			updates := map[string]any{
				"stripe_checkout_session_id": strings.TrimSpace(input.CheckoutSessionID),
				"stripe_price_id":            strings.TrimSpace(input.StripePriceID),
				"tier":                       reservation.Tier,
				"status":                     StripeSubscriptionStatusActive,
				"grace_until":                0,
				"failure_reason":             "",
				"ended_at":                   0,
				"updated_at":                 now,
			}
			if strings.TrimSpace(input.CustomerID) != "" {
				updates["stripe_customer_id"] = strings.TrimSpace(input.CustomerID)
			}
			if input.PeriodStart > 0 {
				updates["current_period_start"] = input.PeriodStart
			}
			if input.PeriodEnd > 0 {
				updates["current_period_end"] = input.PeriodEnd
			}
			if err := tx.Model(existing).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.First(existing, existing.Id).Error; err != nil {
				return err
			}
			recurring = existing
			return ensureStripeSubscriptionEntitlementTx(tx, existing, reservation, plan, input.PeriodStart, input.PeriodEnd, now)
		}

		entitlement, err := CreateUserSubscriptionFromPlanAtTx(tx, reservation.UserId, plan, StripeRecurringLifecycleSource, now)
		if err != nil {
			return err
		}
		entitlementUpdates := map[string]any{"status": "active", "updated_at": now}
		if input.PeriodStart > 0 {
			entitlementUpdates["start_time"] = input.PeriodStart
		}
		if input.PeriodEnd > 0 {
			entitlementUpdates["end_time"] = input.PeriodEnd
		}
		if err := tx.Model(entitlement).Updates(entitlementUpdates).Error; err != nil {
			return err
		}
		created := &StripeSubscription{
			PlanId:                  reservation.PlanId,
			UserId:                  reservation.UserId,
			ReservationId:           reservation.Id,
			StripeCustomerId:        strings.TrimSpace(input.CustomerID),
			StripeSubscriptionId:    strings.TrimSpace(input.StripeSubscriptionID),
			StripeCheckoutSessionId: strings.TrimSpace(input.CheckoutSessionID),
			StripePriceId:           strings.TrimSpace(input.StripePriceID),
			UserSubscriptionId:      entitlement.Id,
			Tier:                    reservation.Tier,
			Status:                  StripeSubscriptionStatusActive,
			CurrentPeriodStart:      input.PeriodStart,
			CurrentPeriodEnd:        input.PeriodEnd,
		}
		if err := tx.Create(created).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyError(err) {
				return ErrSubscriptionAlreadyActive
			}
			return err
		}
		recurring = created
		return nil
	})
	if err == nil && cacheUserID > 0 && cacheGroup != "" && common.RedisEnabled && common.RDB != nil {
		_ = UpdateUserGroupCache(cacheUserID, cacheGroup)
	}
	return recurring, err
}

func ApplyStripeSubscriptionPaid(input StripeSubscriptionInvoiceInput) (bool, error) {
	if DB == nil {
		return false, ErrStripeSubscriptionReservation
	}
	var applied bool
	var cacheUserID int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var subscription StripeSubscription
		if err := lockForUpdate(tx).Where("stripe_subscription_id = ?", input.StripeSubscriptionID).First(&subscription).Error; err != nil {
			return err
		}
		cacheUserID = subscription.UserId
		if input.PlanID <= 0 {
			input.PlanID = subscription.PlanId
		}
		if input.UserID <= 0 {
			input.UserID = subscription.UserId
		}
		if input.PlanID != subscription.PlanId || input.UserID != subscription.UserId {
			return ErrStripeSubscriptionPlanInvalid
		}
		if subscription.Status == StripeSubscriptionStatusCanceled || subscription.EndedAt > 0 {
			return ErrStripeSubscriptionEnded
		}
		if subscription.UserSubscriptionId <= 0 {
			if subscription.ReservationId <= 0 {
				return ErrStripeSubscriptionReservation
			}
			var reservation StripeSubscriptionReservation
			if err := lockForUpdate(tx).Where("id = ?", subscription.ReservationId).First(&reservation).Error; err != nil {
				return err
			}
			if reservation.Status != StripeSubscriptionReservationActive {
				if _, err := ActivateStripeSubscriptionReservationTx(tx, reservation.Id, reservation.CheckoutSessionId, reservation.StripeCustomerId, subscription.StripeSubscriptionId, subscription.StripePriceId, common.GetTimestamp()); err != nil {
					return err
				}
				if err := lockForUpdate(tx).Where("id = ?", reservation.Id).First(&reservation).Error; err != nil {
					return err
				}
			}
			plan, err := recurringLifecyclePlanTx(tx, reservation.PlanId)
			if err != nil {
				return err
			}
			if err := ensureStripeSubscriptionEntitlementTx(tx, &subscription, &reservation, plan, input.PeriodStart, input.PeriodEnd, common.GetTimestamp()); err != nil {
				return err
			}
		}
		inserted, err := RecordStripeSubscriptionInvoiceTx(tx, input)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		now := common.GetTimestamp()
		updates := map[string]any{
			"status":            StripeSubscriptionStatusActive,
			"grace_until":       0,
			"failure_reason":    "",
			"latest_invoice_id": strings.TrimSpace(input.StripeInvoiceID),
			"updated_at":        now,
		}
		if input.PeriodStart > 0 {
			updates["current_period_start"] = input.PeriodStart
		}
		if input.PeriodEnd > 0 {
			updates["current_period_end"] = input.PeriodEnd
		}
		if err := tx.Model(&subscription).Updates(updates).Error; err != nil {
			return err
		}
		if err := restoreStripeSubscriptionEntitlementTx(tx, subscription.UserSubscriptionId, input.PeriodStart, input.PeriodEnd, now); err != nil {
			return err
		}
		applied = true
		return nil
	})
	if err == nil && applied {
		syncStripeSubscriptionUserGroupCache(cacheUserID)
	}
	return applied, err
}

func extendStripeSubscriptionGraceEntitlementTx(tx *gorm.DB, entitlementID int, graceUntil int64) error {
	if tx == nil || entitlementID <= 0 || graceUntil <= 0 {
		return nil
	}
	var entitlement UserSubscription
	if err := lockForUpdate(tx).Where("id = ?", entitlementID).First(&entitlement).Error; err != nil {
		return err
	}
	// A failed payment keeps the same entitlement usable through the explicit
	// grace boundary. Do not resurrect an entitlement that another lifecycle
	// path has already expired.
	if entitlement.Status != "active" || entitlement.EndTime >= graceUntil {
		return nil
	}
	return tx.Model(&entitlement).Updates(map[string]any{
		"end_time":   graceUntil,
		"updated_at": common.GetTimestamp(),
	}).Error
}

func MarkStripeSubscriptionPaymentFailed(stripeSubscriptionID string, reason string, now int64) error {
	if DB == nil || strings.TrimSpace(stripeSubscriptionID) == "" {
		return ErrStripeSubscriptionReservation
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var subscription StripeSubscription
		if err := lockForUpdate(tx).Where("stripe_subscription_id = ?", stripeSubscriptionID).First(&subscription).Error; err != nil {
			return err
		}
		if subscription.Status == StripeSubscriptionStatusCanceled || subscription.EndedAt > 0 {
			return ErrStripeSubscriptionEnded
		}
		graceUntil := now + int64(StripeSubscriptionGracePeriod/time.Second)
		if err := tx.Model(&subscription).Updates(map[string]any{
			"status":         StripeSubscriptionStatusPastDue,
			"grace_until":    graceUntil,
			"failure_reason": strings.TrimSpace(reason),
			"updated_at":     now,
		}).Error; err != nil {
			return err
		}
		return extendStripeSubscriptionGraceEntitlementTx(tx, subscription.UserSubscriptionId, graceUntil)
	})
}

// MarkStripeSubscriptionPaymentFailedWithInvoice records the invoice attempt
// before changing entitlement state. A later delivery with a different event
// ID for the same failed invoice is therefore a no-op, while a later paid
// event can transition that invoice to paid and restore the entitlement.
func MarkStripeSubscriptionPaymentFailedWithInvoice(input StripeSubscriptionInvoiceInput, reason string, now int64) (bool, error) {
	if DB == nil || strings.TrimSpace(input.StripeSubscriptionID) == "" || strings.TrimSpace(input.StripeInvoiceID) == "" {
		return false, ErrStripeSubscriptionReservation
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	var applied bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var subscription StripeSubscription
		if err := lockForUpdate(tx).Where("stripe_subscription_id = ?", input.StripeSubscriptionID).First(&subscription).Error; err != nil {
			return err
		}
		if input.PlanID <= 0 {
			input.PlanID = subscription.PlanId
		}
		if input.UserID <= 0 {
			input.UserID = subscription.UserId
		}
		if input.PlanID != subscription.PlanId || input.UserID != subscription.UserId {
			return ErrStripeSubscriptionPlanInvalid
		}
		if subscription.Status == StripeSubscriptionStatusCanceled || subscription.EndedAt > 0 {
			return ErrStripeSubscriptionEnded
		}
		inserted, err := RecordStripeSubscriptionInvoiceTx(tx, input)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		graceUntil := now + int64(StripeSubscriptionGracePeriod/time.Second)
		if err := tx.Model(&subscription).Updates(map[string]any{
			"status":            StripeSubscriptionStatusPastDue,
			"grace_until":       graceUntil,
			"latest_invoice_id": strings.TrimSpace(input.StripeInvoiceID),
			"failure_reason":    strings.TrimSpace(reason),
			"updated_at":        now,
		}).Error; err != nil {
			return err
		}
		if err := extendStripeSubscriptionGraceEntitlementTx(tx, subscription.UserSubscriptionId, graceUntil); err != nil {
			return err
		}
		applied = true
		return nil
	})
	return applied, err
}

func UpdateStripeSubscriptionState(stripeSubscriptionID string, status string, cancelAtPeriodEnd bool, periodStart int64, periodEnd int64, now int64) error {
	if DB == nil || strings.TrimSpace(stripeSubscriptionID) == "" {
		return ErrStripeSubscriptionReservation
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	var cacheUserID int
	var shouldSyncCache bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var subscription StripeSubscription
		if err := lockForUpdate(tx).Where("stripe_subscription_id = ?", stripeSubscriptionID).First(&subscription).Error; err != nil {
			return err
		}
		cacheUserID = subscription.UserId
		if subscription.Status == StripeSubscriptionStatusCanceled || subscription.EndedAt > 0 {
			return ErrStripeSubscriptionEnded
		}
		updates := map[string]any{
			"status":               strings.TrimSpace(status),
			"cancel_at_period_end": cancelAtPeriodEnd,
			"updated_at":           now,
		}
		graceUntil := subscription.GraceUntil
		if periodStart > 0 {
			updates["current_period_start"] = periodStart
		}
		if periodEnd > 0 {
			updates["current_period_end"] = periodEnd
		}
		if status == StripeSubscriptionStatusActive {
			updates["grace_until"] = 0
			updates["failure_reason"] = ""
		} else if status == StripeSubscriptionStatusPastDue && graceUntil <= now {
			graceUntil = now + int64(StripeSubscriptionGracePeriod/time.Second)
			updates["grace_until"] = graceUntil
		}
		if err := tx.Model(&subscription).Updates(updates).Error; err != nil {
			return err
		}
		if status == StripeSubscriptionStatusCanceled {
			shouldSyncCache = true
			return endStripeSubscriptionTx(tx, &subscription, now)
		}
		if status == StripeSubscriptionStatusActive {
			shouldSyncCache = true
			return restoreStripeSubscriptionEntitlementTx(tx, subscription.UserSubscriptionId, periodStart, periodEnd, now)
		}
		if status == StripeSubscriptionStatusPastDue {
			return extendStripeSubscriptionGraceEntitlementTx(tx, subscription.UserSubscriptionId, graceUntil)
		}
		if status == StripeSubscriptionStatusUnpaid && subscription.GraceUntil > 0 && subscription.GraceUntil <= now {
			shouldSyncCache = true
			return endStripeSubscriptionTx(tx, &subscription, now)
		}
		return nil
	})
	if err == nil && shouldSyncCache {
		syncStripeSubscriptionUserGroupCache(cacheUserID)
	}
	return err
}

func EndStripeSubscription(stripeSubscriptionID string, now int64) error {
	if DB == nil || strings.TrimSpace(stripeSubscriptionID) == "" {
		return ErrStripeSubscriptionReservation
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	var cacheUserID int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var subscription StripeSubscription
		if err := lockForUpdate(tx).Where("stripe_subscription_id = ?", stripeSubscriptionID).First(&subscription).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		cacheUserID = subscription.UserId
		return endStripeSubscriptionTx(tx, &subscription, now)
	})
	if err == nil {
		syncStripeSubscriptionUserGroupCache(cacheUserID)
	}
	return err
}

// ExpireDueStripeSubscriptions is the recurring lifecycle worker's terminal
// grace transition. Every candidate is re-read under the subscription row lock
// and rechecks status/grace so an invoice.paid transaction that wins the race
// leaves the entitlement active. Re-running the function is harmless.
func ExpireDueStripeSubscriptions(now int64, limit int) (int64, error) {
	if DB == nil {
		return 0, ErrStripeSubscriptionReservation
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	if limit <= 0 {
		limit = 200
	}
	var ids []int64
	if err := DB.Model(&StripeSubscription{}).
		Where("status IN ? AND grace_until > 0 AND grace_until <= ?", []string{StripeSubscriptionStatusPastDue, StripeSubscriptionStatusUnpaid}, now).
		Order("grace_until asc, id asc").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	var ended int64
	for _, id := range ids {
		var userID int
		err := DB.Transaction(func(tx *gorm.DB) error {
			var subscription StripeSubscription
			if err := lockForUpdate(tx).Where("id = ?", id).First(&subscription).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			if (subscription.Status != StripeSubscriptionStatusPastDue && subscription.Status != StripeSubscriptionStatusUnpaid) || subscription.GraceUntil <= 0 || subscription.GraceUntil > now {
				return nil
			}
			userID = subscription.UserId
			if err := endStripeSubscriptionTx(tx, &subscription, now); err != nil {
				return err
			}
			ended++
			return nil
		})
		if err != nil {
			return ended, err
		}
		if userID > 0 {
			syncStripeSubscriptionUserGroupCache(userID)
		}
	}
	return ended, nil
}

func restoreStripeSubscriptionEntitlementTx(tx *gorm.DB, entitlementID int, periodStart int64, periodEnd int64, now int64) error {
	if entitlementID <= 0 {
		return nil
	}
	var entitlement UserSubscription
	if err := lockForUpdate(tx).Where("id = ?", entitlementID).First(&entitlement).Error; err != nil {
		return err
	}
	updates := map[string]any{"status": "active", "updated_at": now}
	if periodStart > 0 {
		updates["start_time"] = periodStart
	}
	if periodEnd > 0 {
		updates["end_time"] = periodEnd
	}
	if err := tx.Model(&entitlement).Updates(updates).Error; err != nil {
		return err
	}
	if upgradeGroup := strings.TrimSpace(entitlement.UpgradeGroup); upgradeGroup != "" {
		if err := tx.Model(&User{}).Where("id = ?", entitlement.UserId).Update("group", upgradeGroup).Error; err != nil {
			return err
		}
	}
	return nil
}

func endStripeSubscriptionTx(tx *gorm.DB, subscription *StripeSubscription, now int64) error {
	if tx == nil || subscription == nil {
		return ErrStripeSubscriptionReservation
	}
	if subscription.Status != StripeSubscriptionStatusCanceled || subscription.EndedAt == 0 {
		if err := tx.Model(subscription).Updates(map[string]any{
			"status":      StripeSubscriptionStatusCanceled,
			"ended_at":    now,
			"grace_until": 0,
			"updated_at":  now,
		}).Error; err != nil {
			return err
		}
	}
	if subscription.ReservationId > 0 {
		if err := ReleaseStripeSubscriptionReservationTx(tx, subscription.ReservationId, now); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	if subscription.UserSubscriptionId <= 0 {
		return nil
	}
	var entitlement UserSubscription
	if err := lockForUpdate(tx).Where("id = ?", subscription.UserSubscriptionId).First(&entitlement).Error; err != nil {
		return err
	}
	if entitlement.Status != "expired" {
		if err := tx.Model(&entitlement).Updates(map[string]any{
			"status":     "expired",
			"end_time":   now,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		if _, err := downgradeUserGroupForSubscriptionTx(tx, &entitlement, now); err != nil {
			return err
		}
	}
	return nil
}

func syncStripeSubscriptionUserGroupCache(userID int) {
	if userID <= 0 || !common.RedisEnabled || common.RDB == nil {
		return
	}
	user, err := GetUserById(userID, true)
	if err != nil || user == nil {
		return
	}
	_ = UpdateUserGroupCache(userID, user.Group)
}
