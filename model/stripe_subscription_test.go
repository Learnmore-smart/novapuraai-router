package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	stripeSubscriptionEnabledEnvTestKey       = "STRIPE_SUBSCRIPTION_ENABLED"
	stripeSubscriptionAccountEnvTestKey       = "STRIPE_SUBSCRIPTION_ACCOUNT_ID"
	stripeSubscriptionProductEnvTestKey       = "STRIPE_SUBSCRIPTION_PRODUCT_ID"
	stripeSubscriptionFounderPriceEnvTestKey  = "STRIPE_SUBSCRIPTION_FOUNDER_PRICE_ID"
	stripeSubscriptionStandardPriceEnvTestKey = "STRIPE_SUBSCRIPTION_STANDARD_PRICE_ID"
	stripeSubscriptionPortalEnvTestKey        = "STRIPE_SUBSCRIPTION_PORTAL_CONFIGURATION_ID"
)

type legacyStripeSubscriptionReservationCheckoutURL struct {
	Id          int64  `gorm:"primaryKey"`
	CheckoutURL string `gorm:"column:checkout_url;type:varchar(512)"`
}

func (legacyStripeSubscriptionReservationCheckoutURL) TableName() string {
	return "stripe_subscription_reservations"
}

func clearStripeSubscriptionEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		stripeSubscriptionEnabledEnvTestKey,
		stripeSubscriptionAccountEnvTestKey,
		stripeSubscriptionProductEnvTestKey,
		stripeSubscriptionFounderPriceEnvTestKey,
		stripeSubscriptionStandardPriceEnvTestKey,
		stripeSubscriptionPortalEnvTestKey,
	} {
		original, wasSet := os.LookupEnv(key)
		require.NoError(t, os.Unsetenv(key))
		t.Cleanup(func() {
			if wasSet {
				_ = os.Setenv(key, original)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}

type stripeLifecycleSQLRecorder struct {
	mu      sync.Mutex
	queries []string
}

func (r *stripeLifecycleSQLRecorder) LogMode(logger.LogLevel) logger.Interface { return r }
func (r *stripeLifecycleSQLRecorder) Info(context.Context, string, ...interface{}) {
}
func (r *stripeLifecycleSQLRecorder) Warn(context.Context, string, ...interface{}) {
}
func (r *stripeLifecycleSQLRecorder) Error(context.Context, string, ...interface{}) {
}
func (r *stripeLifecycleSQLRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	r.mu.Lock()
	r.queries = append(r.queries, strings.ToLower(sql))
	r.mu.Unlock()
}

func (r *stripeLifecycleSQLRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.queries...)
}

func requireStripeLifecycleLockOrder(t *testing.T, queries []string) {
	t.Helper()
	firstIndex := func(table string) int {
		for index, query := range queries {
			if strings.Contains(query, table) {
				return index
			}
		}
		return -1
	}
	subscriptionIndex := firstIndex("stripe_subscriptions")
	reservationIndex := firstIndex("stripe_subscription_reservations")
	planIndex := firstIndex("subscription_plans")
	require.NotEqual(t, -1, subscriptionIndex, "Stripe subscription row must be checked first")
	require.NotEqual(t, -1, reservationIndex, "reservation row must be checked second")
	require.NotEqual(t, -1, planIndex, "plan row must be checked third")
	assert.Less(t, subscriptionIndex, reservationIndex, "lock order must start with Stripe subscription before reservation")
	assert.Less(t, reservationIndex, planIndex, "lock order must acquire reservation before plan")
}

func setupStripeSubscriptionModelTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	clearStripeSubscriptionEnv(t)
	originalDB := DB
	originalDatabaseType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:stripe-subscription-%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	DB = db
	InvalidateSubscriptionPlanCache(1)
	t.Setenv("GIN_MODE", "debug")
	require.NoError(t, db.AutoMigrate(
		&User{},
		&SubscriptionOrder{},
		&UserSubscription{},
		&SubscriptionPreConsumeRecord{},
		&SubscriptionPlan{},
		&StripeSubscriptionReservation{},
		&StripeSubscriptionFounderClaim{},
		&StripeSubscription{},
		&StripeSubscriptionInvoice{},
		&StripeWebhookEvent{},
	))
	t.Cleanup(func() {
		DB = originalDB
		common.SetMainDatabaseType(originalDatabaseType)
		_ = sqlDB.Close()
	})
	return db
}

func seedStripeSubscriptionPlan(t *testing.T, db *gorm.DB) *SubscriptionPlan {
	t.Helper()
	recurringCode := SandboxStripeSubscriptionPlanCode
	plan := &SubscriptionPlan{
		Id:                          7001,
		Code:                        SandboxStripeSubscriptionPlanCode,
		RecurringCode:               &recurringCode,
		Title:                       "NovaPuraAI DeepSeek V4 Flash Unlimited",
		PriceAmount:                 19.99,
		Currency:                    "CNY",
		DurationUnit:                SubscriptionDurationMonth,
		DurationValue:               1,
		Enabled:                     true,
		StripeSubscriptionEnabled:   true,
		StripeSubscriptionModel:     SandboxStripeSubscriptionModel,
		UpgradeGroup:                SandboxStripeSubscriptionGroup,
		MaxActiveSubscriptions:      20,
		FounderPurchaseLimit:        20,
		MaxActivePerUser:            1,
		FounderStripePriceId:        SandboxStripeSubscriptionFounderPriceID,
		StandardStripePriceId:       SandboxStripeSubscriptionStandardPriceID,
		StripeProductId:             SandboxStripeSubscriptionProductID,
		StripeAccountId:             SandboxStripeSubscriptionAccountID,
		StripePortalConfigurationId: SandboxStripeSubscriptionPortalConfigurationID,
		FounderAmountMinor:          1999,
		StandardAmountMinor:         9999,
		StripeCurrency:              "cny",
		TotalAmount:                 0,
		AllowBalancePay:             common.GetPointer(false),
		AllowWalletOverflow:         common.GetPointer(false),
	}
	require.NoError(t, db.Create(plan).Error)
	return plan
}

func seedStripeSubscriptionUser(t *testing.T, db *gorm.DB, id int) {
	t.Helper()
	require.NoError(t, db.Create(&User{Id: id, Username: fmt.Sprintf("stripe-sub-user-%d", id), AffCode: fmt.Sprintf("stripe-sub-aff-%d", id), Status: common.UserStatusEnabled}).Error)
}

func TestRecurringSubscriptionFundsAllModels(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7101)

	var entitlement *UserSubscription
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		entitlement, err = CreateUserSubscriptionFromPlanAtTx(tx, 7101, plan, "stripe_recurring", common.GetTimestamp())
		return err
	}))
	require.NotNil(t, entitlement)

	target, err := PreConsumeUserSubscription("target-model-request", 7101, "claude-sonnet-4", 0, 100)
	require.NoError(t, err)
	assert.Equal(t, entitlement.Id, target.UserSubscriptionId)

	other, err := PreConsumeUserSubscription("other-model-request", 7101, "gpt-5", 0, 100)
	require.NoError(t, err)
	assert.Equal(t, entitlement.Id, other.UserSubscriptionId)

	var modelRecords int64
	require.NoError(t, db.Model(&SubscriptionPreConsumeRecord{}).
		Where("user_id = ?", 7101).Count(&modelRecords).Error)
	assert.Equal(t, int64(2), modelRecords)
}

func TestEnsureStripeSubscriptionPlanNormalizesLegacyTargetToAllModelScope(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	plan := seedStripeSubscriptionPlan(t, db)
	require.NoError(t, db.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).
		Update("stripe_subscription_model", LegacySandboxStripeSubscriptionModel).Error)

	require.NoError(t, EnsureSandboxStripeSubscriptionPlan())
	require.NoError(t, db.First(plan, plan.Id).Error)
	assert.Empty(t, plan.StripeSubscriptionModel)
	assert.Equal(t, SandboxStripeSubscriptionFounderPriceID, plan.FounderStripePriceId)
	assert.Equal(t, SandboxStripeSubscriptionStandardPriceID, plan.StandardStripePriceId)
	assert.Equal(t, 20, plan.MaxActiveSubscriptions)
}

func TestCheckoutExpiryReleaseCannotReleaseAnActiveReservation(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7102)
	now := common.GetTimestamp()

	reservation, err := ReserveStripeSubscriptionSeat(plan.Id, 7102, "expiry-race", now)
	require.NoError(t, err)
	_, err = ActivateStripeSubscriptionReservation(
		reservation.Id,
		"cs_paid",
		"cus_paid",
		"sub_paid",
		SandboxStripeSubscriptionFounderPriceID,
		now+1,
	)
	require.NoError(t, err)

	require.NoError(t, ReleasePendingStripeSubscriptionReservation(reservation.Id, now+2))
	var stored StripeSubscriptionReservation
	require.NoError(t, db.First(&stored, reservation.Id).Error)
	assert.Equal(t, StripeSubscriptionReservationActive, stored.Status)
}

func TestRecurringCheckoutLifecycleUsesOneDatabaseLockOrder(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7103)
	now := common.GetTimestamp()

	reserveRecorder := &stripeLifecycleSQLRecorder{}
	DB = db.Session(&gorm.Session{Logger: reserveRecorder})
	reservation, err := ReserveStripeSubscriptionSeat(plan.Id, 7103, "lock-order", now)
	require.NoError(t, err)
	requireStripeLifecycleLockOrder(t, reserveRecorder.snapshot())

	bindRecorder := &stripeLifecycleSQLRecorder{}
	DB = db.Session(&gorm.Session{Logger: bindRecorder})
	_, err = BindStripeSubscriptionCheckout(StripeSubscriptionBindingInput{
		ReservationID:        reservation.Id,
		CheckoutSessionID:    "cs_lock_order",
		CustomerID:           "cus_lock_order",
		StripeSubscriptionID: "sub_lock_order",
		StripePriceID:        plan.FounderStripePriceId,
	})
	require.NoError(t, err)
	requireStripeLifecycleLockOrder(t, bindRecorder.snapshot())

	activateRecorder := &stripeLifecycleSQLRecorder{}
	DB = db.Session(&gorm.Session{Logger: activateRecorder})
	_, err = ActivateStripeSubscriptionWithEntitlement(StripeSubscriptionActivationInput{
		ReservationID:        reservation.Id,
		CheckoutSessionID:    "cs_lock_order",
		CustomerID:           "cus_lock_order",
		StripeSubscriptionID: "sub_lock_order",
		StripePriceID:        plan.FounderStripePriceId,
		PeriodStart:          now,
		PeriodEnd:            now + 3600,
	})
	require.NoError(t, err)
	requireStripeLifecycleLockOrder(t, activateRecorder.snapshot())
}

func TestPendingReservationUserSeatIsStructurallyUnique(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7104)
	now := common.GetTimestamp()

	first := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7104,
		ReferenceId: "same-user-slot-first",
		Tier:        StripeSubscriptionTierFounder,
		Status:      StripeSubscriptionReservationPending,
		ExpiresAt:   now + 3600,
	}
	require.NoError(t, db.Create(first).Error)
	second := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7104,
		ReferenceId: "same-user-slot-second",
		Tier:        StripeSubscriptionTierFounder,
		Status:      StripeSubscriptionReservationPending,
		ExpiresAt:   now + 3600,
	}
	err := db.Create(second).Error
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyError(err))
}

func TestZeroExpiryPendingReservationConsumesCapacity(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	require.NoError(t, db.Model(plan).Update("max_active_subscriptions", 1).Error)
	seedStripeSubscriptionUser(t, db, 7105)
	seedStripeSubscriptionUser(t, db, 7106)

	require.NoError(t, db.Create(&StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7105,
		ReferenceId: "zero-expiry-seat",
		Tier:        StripeSubscriptionTierFounder,
		Status:      StripeSubscriptionReservationPending,
		ExpiresAt:   0,
	}).Error)
	_, err := ReserveStripeSubscriptionSeat(plan.Id, 7106, "capacity-after-zero-expiry", common.GetTimestamp())
	require.ErrorIs(t, err, ErrSubscriptionCapacityFull)
}

func TestExpiredPendingUserSeatIsReclaimedOnRetryWithoutWebhook(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7110)
	now := common.GetTimestamp()

	expired := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7110,
		ReferenceId: "expired-before-retry",
		Tier:        StripeSubscriptionTierFounder,
		Status:      StripeSubscriptionReservationPending,
		ExpiresAt:   now - 1,
	}
	require.NoError(t, db.Create(expired).Error)
	require.NotNil(t, expired.ActiveUserId)

	retry, err := ReserveStripeSubscriptionSeat(plan.Id, 7110, "retry-after-expiry", now)
	require.NoError(t, err)
	require.NotNil(t, retry)
	require.NotNil(t, retry.ActiveUserId)
	assert.Equal(t, 7110, *retry.ActiveUserId)

	require.NoError(t, db.First(expired, expired.Id).Error)
	assert.Equal(t, StripeSubscriptionReservationExpired, expired.Status)
	assert.Nil(t, expired.ActiveUserId)
}

func TestExpiredPendingReservationCannotBecomeReconciliation(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7111)
	now := common.GetTimestamp()

	reservation := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7111,
		ReferenceId: "expired-before-reconciliation",
		Tier:        StripeSubscriptionTierFounder,
		Status:      StripeSubscriptionReservationPending,
		ExpiresAt:   now - 1,
	}
	require.NoError(t, db.Create(reservation).Error)

	err := MarkStripeSubscriptionCheckoutReconciliation(
		reservation.Id,
		"cs_expired_reconciliation",
		"https://checkout.stripe.test/expired",
		"late incomplete Checkout response",
		now,
	)
	require.ErrorIs(t, err, ErrStripeSubscriptionReservationExpired)

	require.NoError(t, db.First(reservation, reservation.Id).Error)
	assert.Equal(t, StripeSubscriptionReservationExpired, reservation.Status)
	assert.Nil(t, reservation.ActiveUserId)
}

func TestCheckoutReconciliationBoundsRemoteErrorToDatabaseColumn(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7112)
	now := common.GetTimestamp()
	reservation := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7112,
		ReferenceId: "overlength-checkout-error",
		Tier:        StripeSubscriptionTierFounder,
		Status:      StripeSubscriptionReservationPending,
		ExpiresAt:   now + 1800,
	}
	require.NoError(t, db.Create(reservation).Error)
	longReason := strings.Repeat("结账失败", 100)

	require.NoError(t, MarkStripeSubscriptionCheckoutReconciliation(
		reservation.Id,
		"",
		"",
		longReason,
		now,
	))

	require.NoError(t, db.First(reservation, reservation.Id).Error)
	assert.Equal(t, StripeSubscriptionReservationReconciliation, reservation.Status)
	assert.Equal(t, string([]rune(longReason)[:255]), reservation.RemoteSessionError)
}

func TestStripeSubscriptionCheckoutURLMigrationPreservesLongHostedURL(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7113)
	require.NoError(t, db.AutoMigrate(&legacyStripeSubscriptionReservationCheckoutURL{}))
	legacyColumns, err := db.Migrator().ColumnTypes(&StripeSubscriptionReservation{})
	require.NoError(t, err)
	legacyType := ""
	for _, column := range legacyColumns {
		if column.Name() == "checkout_url" {
			legacyType = strings.ToLower(column.DatabaseTypeName())
			break
		}
	}
	require.Equal(t, "varchar", legacyType)
	longURL := "https://checkout.stripe.com/c/pay/" + strings.Repeat("a", 700)
	reservation := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7113,
		ReferenceId: "long-hosted-checkout-url",
		CheckoutURL: longURL,
		Tier:        StripeSubscriptionTierFounder,
		Status:      StripeSubscriptionReservationPending,
		ExpiresAt:   common.GetTimestamp() + 1800,
	}
	require.NoError(t, db.Create(reservation).Error)
	require.NoError(t, db.AutoMigrate(&StripeSubscriptionReservation{}))

	var migrated StripeSubscriptionReservation
	require.NoError(t, db.First(&migrated, reservation.Id).Error)
	assert.Equal(t, longURL, migrated.CheckoutURL)

	columns, err := db.Migrator().ColumnTypes(&StripeSubscriptionReservation{})
	require.NoError(t, err)

	for _, column := range columns {
		if column.Name() == "checkout_url" {
			assert.Equal(t, "text", strings.ToLower(column.DatabaseTypeName()))
			return
		}
	}
	t.Fatal("checkout_url column not found")
}

func TestReservationActiveUserMigrationRepairsDerivedSeatKeys(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7107)
	seedStripeSubscriptionUser(t, db, 7108)
	now := common.GetTimestamp()

	live := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7107,
		ReferenceId: "migration-live-seat",
		Tier:        StripeSubscriptionTierFounder,
		Status:      StripeSubscriptionReservationPending,
		ExpiresAt:   now + 3600,
	}
	terminal := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7108,
		ReferenceId: "migration-terminal-seat",
		Tier:        StripeSubscriptionTierStandard,
		Status:      StripeSubscriptionReservationReleased,
	}
	require.NoError(t, db.Create(live).Error)
	require.NoError(t, db.Create(terminal).Error)
	require.NoError(t, db.Model(live).UpdateColumn("active_user_id", nil).Error)
	require.NoError(t, db.Model(terminal).UpdateColumn("active_user_id", terminal.UserId).Error)

	require.NoError(t, ensureStripeSubscriptionReservationActiveUsers())
	require.NoError(t, db.First(live, live.Id).Error)
	require.NoError(t, db.First(terminal, terminal.Id).Error)
	require.NotNil(t, live.ActiveUserId)
	assert.Equal(t, live.UserId, *live.ActiveUserId)
	assert.Nil(t, terminal.ActiveUserId)
}

func TestReservationActiveUserMigrationRejectsLegacyDuplicateLiveSeats(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7109)
	now := common.GetTimestamp()
	first := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7109,
		ReferenceId: "legacy-duplicate-first",
		Tier:        StripeSubscriptionTierFounder,
		Status:      StripeSubscriptionReservationReleased,
	}
	second := &StripeSubscriptionReservation{
		PlanId:      plan.Id,
		UserId:      7109,
		ReferenceId: "legacy-duplicate-second",
		Tier:        StripeSubscriptionTierStandard,
		Status:      StripeSubscriptionReservationReleased,
	}
	require.NoError(t, db.Create(first).Error)
	require.NoError(t, db.Create(second).Error)
	require.NoError(t, db.Model(&StripeSubscriptionReservation{}).
		Where("id IN ?", []int64{first.Id, second.Id}).
		Updates(map[string]any{
			"status":         StripeSubscriptionReservationPending,
			"expires_at":     now + 3600,
			"active_user_id": nil,
		}).Error)

	err := ensureStripeSubscriptionReservationActiveUsers()
	require.ErrorIs(t, err, ErrStripeSubscriptionReservation)
}

func TestDefaultSandboxStripeSubscriptionConfigIsDisabledAndUsesVerifiedTestObjects(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	config := DefaultSandboxStripeSubscriptionConfig()

	assert.False(t, config.Enabled)
	assert.Equal(t, SandboxStripeSubscriptionEnvironment, config.Environment)
	assert.Equal(t, SandboxStripeSubscriptionAccountID, config.AccountID)
	assert.Equal(t, SandboxStripeSubscriptionProductID, config.ProductID)
	assert.Equal(t, SandboxStripeSubscriptionProductName, config.ProductName)
	assert.Equal(t, SandboxStripeSubscriptionFounderPriceID, config.FounderPriceID)
	assert.Equal(t, SandboxStripeSubscriptionStandardPriceID, config.StandardPriceID)
	assert.Equal(t, SandboxStripeSubscriptionFounderLookupKey, config.FounderLookupKey)
	assert.Equal(t, SandboxStripeSubscriptionStandardLookupKey, config.StandardLookupKey)
	assert.Equal(t, SandboxStripeSubscriptionPortalConfigurationID, config.PortalConfigurationID)
	assert.Equal(t, SandboxStripeSubscriptionBillingInterval, config.BillingInterval)
	assert.Equal(t, int64(1999), config.FounderAmountMinor)
	assert.Equal(t, int64(9999), config.StandardAmountMinor)
}

func TestDefaultProductionStripeSubscriptionConfigUsesVerifiedLiveObjects(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	config := DefaultProductionStripeSubscriptionConfig()

	assert.False(t, config.Enabled)
	assert.Equal(t, ProductionStripeSubscriptionEnvironment, config.Environment)
	assert.Equal(t, ProductionStripeSubscriptionAccountID, config.AccountID)
	assert.Equal(t, ProductionStripeSubscriptionProductID, config.ProductID)
	assert.Equal(t, ProductionStripeSubscriptionFounderPriceID, config.FounderPriceID)
	assert.Equal(t, ProductionStripeSubscriptionStandardPriceID, config.StandardPriceID)
	assert.Equal(t, ProductionStripeSubscriptionPortalConfigurationID, config.PortalConfigurationID)
	assert.Equal(t, SandboxStripeSubscriptionFounderLookupKey, config.FounderLookupKey)
	assert.Equal(t, SandboxStripeSubscriptionStandardLookupKey, config.StandardLookupKey)
	assert.Equal(t, int64(1999), config.FounderAmountMinor)
	assert.Equal(t, int64(9999), config.StandardAmountMinor)
}

func TestStripeSubscriptionEnvironmentForRuntimeSelectsByGinMode(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	assert.Equal(t, SandboxStripeSubscriptionEnvironment, StripeSubscriptionEnvironmentForRuntime())

	t.Setenv("GIN_MODE", "release")
	assert.Equal(t, ProductionStripeSubscriptionEnvironment, StripeSubscriptionEnvironmentForRuntime())
}

func TestStripeSubscriptionConfigDefaultsClosedAndUsesSelectedFixedCatalog(t *testing.T) {
	clearStripeSubscriptionEnv(t)

	testConfig, err := StripeSubscriptionConfigForEnvironment(SandboxStripeSubscriptionEnvironment)
	require.NoError(t, err)
	assert.False(t, testConfig.Enabled)
	assert.Equal(t, SandboxStripeSubscriptionAccountID, testConfig.AccountID)
	assert.Equal(t, SandboxStripeSubscriptionProductID, testConfig.ProductID)
	assert.Equal(t, SandboxStripeSubscriptionFounderPriceID, testConfig.FounderPriceID)
	assert.Equal(t, SandboxStripeSubscriptionStandardPriceID, testConfig.StandardPriceID)
	assert.Equal(t, SandboxStripeSubscriptionPortalConfigurationID, testConfig.PortalConfigurationID)

	productionConfig, err := StripeSubscriptionConfigForEnvironment(ProductionStripeSubscriptionEnvironment)
	require.NoError(t, err)
	assert.False(t, productionConfig.Enabled)
	assert.Equal(t, ProductionStripeSubscriptionAccountID, productionConfig.AccountID)
	assert.Equal(t, ProductionStripeSubscriptionProductID, productionConfig.ProductID)
	assert.Equal(t, ProductionStripeSubscriptionFounderPriceID, productionConfig.FounderPriceID)
	assert.Equal(t, ProductionStripeSubscriptionStandardPriceID, productionConfig.StandardPriceID)
	assert.Equal(t, ProductionStripeSubscriptionPortalConfigurationID, productionConfig.PortalConfigurationID)
}

func TestStripeSubscriptionConfigReadsExplicitEnableAndMatchingCatalogOverrides(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	t.Setenv(stripeSubscriptionAccountEnvTestKey, SandboxStripeSubscriptionAccountID)
	t.Setenv(stripeSubscriptionProductEnvTestKey, SandboxStripeSubscriptionProductID)
	t.Setenv(stripeSubscriptionFounderPriceEnvTestKey, SandboxStripeSubscriptionFounderPriceID)
	t.Setenv(stripeSubscriptionStandardPriceEnvTestKey, SandboxStripeSubscriptionStandardPriceID)
	t.Setenv(stripeSubscriptionPortalEnvTestKey, SandboxStripeSubscriptionPortalConfigurationID)

	config, err := StripeSubscriptionConfigForEnvironment(SandboxStripeSubscriptionEnvironment)
	require.NoError(t, err)
	assert.True(t, config.Enabled)
	assert.Equal(t, SandboxStripeSubscriptionAccountID, config.AccountID)
	assert.Equal(t, SandboxStripeSubscriptionProductID, config.ProductID)
	assert.Equal(t, SandboxStripeSubscriptionFounderPriceID, config.FounderPriceID)
	assert.Equal(t, SandboxStripeSubscriptionStandardPriceID, config.StandardPriceID)
	assert.Equal(t, SandboxStripeSubscriptionPortalConfigurationID, config.PortalConfigurationID)
}

func TestStripeSubscriptionConfigGateDefaultsClosedForUnsetFalseInvalidAndBlank(t *testing.T) {
	values := []struct {
		name  string
		value *string
	}{
		{name: "unset"},
		{name: "false", value: common.GetPointer("false")},
		{name: "invalid", value: common.GetPointer("not-a-bool")},
		{name: "blank", value: common.GetPointer("")},
	}
	for _, testCase := range values {
		t.Run(testCase.name, func(t *testing.T) {
			clearStripeSubscriptionEnv(t)
			if testCase.value == nil {
				_ = os.Unsetenv(stripeSubscriptionEnabledEnvTestKey)
			} else {
				t.Setenv(stripeSubscriptionEnabledEnvTestKey, *testCase.value)
			}

			config, err := StripeSubscriptionConfigForEnvironment(SandboxStripeSubscriptionEnvironment)
			require.NoError(t, err)
			assert.False(t, config.Enabled)
		})
	}
}

func TestStripeSubscriptionConfigRejectsAnyCrossEnvironmentCatalogOverride(t *testing.T) {
	overrides := []struct {
		name  string
		key   string
		value string
	}{
		{name: "account", key: stripeSubscriptionAccountEnvTestKey, value: ProductionStripeSubscriptionAccountID},
		{name: "product", key: stripeSubscriptionProductEnvTestKey, value: ProductionStripeSubscriptionProductID},
		{name: "founder price", key: stripeSubscriptionFounderPriceEnvTestKey, value: ProductionStripeSubscriptionFounderPriceID},
		{name: "standard price", key: stripeSubscriptionStandardPriceEnvTestKey, value: ProductionStripeSubscriptionStandardPriceID},
		{name: "portal", key: stripeSubscriptionPortalEnvTestKey, value: ProductionStripeSubscriptionPortalConfigurationID},
	}

	for _, override := range overrides {
		t.Run(override.name, func(t *testing.T) {
			clearStripeSubscriptionEnv(t)
			t.Setenv(override.key, override.value)

			_, err := StripeSubscriptionConfigForEnvironment(SandboxStripeSubscriptionEnvironment)
			require.ErrorIs(t, err, ErrStripeSubscriptionPlanInvalid)
		})
	}
}

func TestStripeSubscriptionConfigRejectsBlankCatalogOverrides(t *testing.T) {
	keys := []string{
		stripeSubscriptionAccountEnvTestKey,
		stripeSubscriptionProductEnvTestKey,
		stripeSubscriptionFounderPriceEnvTestKey,
		stripeSubscriptionStandardPriceEnvTestKey,
		stripeSubscriptionPortalEnvTestKey,
	}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			clearStripeSubscriptionEnv(t)
			t.Setenv(key, "")

			_, err := StripeSubscriptionConfigForEnvironment(SandboxStripeSubscriptionEnvironment)
			require.ErrorIs(t, err, ErrStripeSubscriptionPlanInvalid)
		})
	}
}

func TestStripeSubscriptionConfigRejectsTestCatalogOverrideForProduction(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	t.Setenv(stripeSubscriptionAccountEnvTestKey, SandboxStripeSubscriptionAccountID)

	_, err := StripeSubscriptionConfigForEnvironment(ProductionStripeSubscriptionEnvironment)
	require.ErrorIs(t, err, ErrStripeSubscriptionPlanInvalid)
}

func TestEnsureStripeSubscriptionPlanSynchronizesOnlySelectedPlanEnablement(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	other := &SubscriptionPlan{
		Code:                      "unrelated-plan",
		Title:                     "Unrelated plan",
		PriceAmount:               1,
		Currency:                  "USD",
		DurationUnit:              SubscriptionDurationMonth,
		DurationValue:             1,
		Enabled:                   true,
		StripeSubscriptionEnabled: true,
		AllowBalancePay:           common.GetPointer(true),
		AllowWalletOverflow:       common.GetPointer(true),
	}
	require.NoError(t, db.Create(other).Error)

	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	require.NoError(t, EnsureStripeSubscriptionPlanForEnvironment(SandboxStripeSubscriptionEnvironment))

	var plan SubscriptionPlan
	require.NoError(t, db.Where("code = ?", SandboxStripeSubscriptionPlanCode).First(&plan).Error)
	assert.True(t, plan.Enabled)
	assert.True(t, plan.StripeSubscriptionEnabled)

	var unchanged SubscriptionPlan
	require.NoError(t, db.First(&unchanged, other.Id).Error)
	assert.True(t, unchanged.Enabled)
	assert.True(t, unchanged.StripeSubscriptionEnabled)

	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "false")
	require.NoError(t, EnsureStripeSubscriptionPlanForEnvironment(SandboxStripeSubscriptionEnvironment))
	require.NoError(t, db.First(&plan, plan.Id).Error)
	assert.False(t, plan.Enabled)
	assert.False(t, plan.StripeSubscriptionEnabled)
}

func TestEnabledStripeSubscriptionPlanRequiresTheFixedRuntimeCatalog(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	otherCode := "other-enabled-recurring"
	other := *plan
	other.Id = 0
	other.Code = otherCode
	other.RecurringCode = &otherCode
	require.NoError(t, db.Create(&other).Error)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")

	assert.True(t, HasEnabledStripeSubscriptionPlan())
	require.NoError(t, db.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Updates(map[string]any{
		"enabled":                     false,
		"stripe_subscription_enabled": false,
	}).Error)

	enabled, err := EnabledStripeSubscriptionPlan()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestValidateStripeSubscriptionPlanUsesConfigGateOnlyWhenEnabledIsRequired(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)

	assert.ErrorIs(t, ValidateStripeSubscriptionPlan(plan, DefaultSandboxStripeSubscriptionConfig(), true), ErrStripeSubscriptionDisabled)
	assert.NoError(t, ValidateStripeSubscriptionPlan(plan, DefaultSandboxStripeSubscriptionConfig(), false))

	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	config, err := StripeSubscriptionConfigForEnvironment(SandboxStripeSubscriptionEnvironment)
	require.NoError(t, err)
	assert.NoError(t, ValidateStripeSubscriptionPlan(plan, config, true))
}

func TestEnsureStripeSubscriptionPlanForRuntimeUsesProductionGateAndStartupSeam(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv("GIN_MODE", "release")
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")

	require.NoError(t, ensureStripeSubscriptionPlanForRuntime())
	var plan SubscriptionPlan
	require.NoError(t, db.Where("code = ?", SandboxStripeSubscriptionPlanCode).First(&plan).Error)
	assert.True(t, plan.Enabled)
	assert.True(t, plan.StripeSubscriptionEnabled)
	assert.Equal(t, ProductionStripeSubscriptionAccountID, plan.StripeAccountId)
	assert.Equal(t, ProductionStripeSubscriptionProductID, plan.StripeProductId)

	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "false")
	require.NoError(t, ensureStripeSubscriptionPlanForRuntime())
	require.NoError(t, db.First(&plan, plan.Id).Error)
	assert.False(t, plan.Enabled)
	assert.False(t, plan.StripeSubscriptionEnabled)
}

func TestMigrateDBSynchronizesRuntimeSelectedRecurringPlan(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv("GIN_MODE", "release")
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")

	plan := seedStripeSubscriptionPlan(t, db)
	production := DefaultProductionStripeSubscriptionConfig()
	require.NoError(t, db.Model(plan).Updates(map[string]any{
		"enabled":                        false,
		"stripe_subscription_enabled":    false,
		"founder_stripe_price_id":        production.FounderPriceID,
		"standard_stripe_price_id":       production.StandardPriceID,
		"stripe_product_id":              production.ProductID,
		"stripe_account_id":              production.AccountID,
		"stripe_portal_configuration_id": production.PortalConfigurationID,
	}).Error)

	require.NoError(t, migrateDB())

	var migrated SubscriptionPlan
	require.NoError(t, db.Where("code = ?", SandboxStripeSubscriptionPlanCode).First(&migrated).Error)
	assert.True(t, migrated.Enabled)
	assert.True(t, migrated.StripeSubscriptionEnabled)
	assert.Equal(t, production.FounderPriceID, migrated.FounderStripePriceId)
	assert.Equal(t, production.StandardPriceID, migrated.StandardStripePriceId)
	assert.Equal(t, production.ProductID, migrated.StripeProductId)
	assert.Equal(t, production.AccountID, migrated.StripeAccountId)
	assert.Equal(t, production.PortalConfigurationID, migrated.StripePortalConfigurationId)
}

func TestValidatePurchasableSubscriptionPlanUsesStableCodeAndPaymentSource(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)

	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "false")
	require.ErrorIs(t, ValidatePurchasableSubscriptionPlan(plan, "epay"), ErrStripeSubscriptionDisabled)

	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	require.ErrorIs(t, ValidatePurchasableSubscriptionPlan(plan, "epay"), ErrStripeSubscriptionPlanInvalid)
	require.NoError(t, ValidatePurchasableSubscriptionPlan(plan, StripeRecurringPurchaseSource))

	markerOnly := *plan
	markerOnly.Enabled = false
	markerOnly.StripeSubscriptionEnabled = false
	require.ErrorIs(t, ValidatePurchasableSubscriptionPlan(&markerOnly, StripeRecurringPurchaseSource), ErrStripeSubscriptionDisabled)

	mixed := *plan
	mixed.Enabled = true
	mixed.StripeSubscriptionEnabled = false
	require.ErrorIs(t, ValidatePurchasableSubscriptionPlan(&mixed, StripeRecurringPurchaseSource), ErrStripeSubscriptionDisabled)
	require.ErrorIs(t, ValidatePurchasableSubscriptionPlan(&mixed, "balance"), ErrStripeSubscriptionPlanInvalid)
}

func TestValidatePurchasableSubscriptionPlanPreservesLegacyPlans(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	legacy := &SubscriptionPlan{
		Code:    "legacy-plan",
		Enabled: true,
	}
	require.NoError(t, ValidatePurchasableSubscriptionPlan(legacy, "epay"))
}

func TestBalancePurchaseRejectsRecurringBeforeWalletChargeWhenGateIsFalse(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7201)
	require.NoError(t, db.Model(&User{}).Where("id = ?", 7201).Update("quota", 100000).Error)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "false")

	err := PurchaseSubscriptionWithBalance(7201, plan.Id)
	require.ErrorIs(t, err, ErrStripeSubscriptionDisabled)

	var user User
	require.NoError(t, db.Select("quota").First(&user, 7201).Error)
	assert.Equal(t, 100000, user.Quota)
	var orders int64
	require.NoError(t, db.Model(&SubscriptionOrder{}).Where("user_id = ?", 7201).Count(&orders).Error)
	assert.Zero(t, orders)
}

func TestSubscriptionSettlementRejectsRecurringBeforeEntitlementWhenGateIsFalse(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 7202)
	order := &SubscriptionOrder{
		UserId:          7202,
		PlanId:          plan.Id,
		Money:           plan.PriceAmount,
		TradeNo:         "recurring-settlement-gate",
		PaymentMethod:   PaymentProviderEpay,
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}
	require.NoError(t, db.Create(order).Error)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "false")

	require.ErrorIs(t, CompleteSubscriptionOrder(order.TradeNo, "provider-payload", PaymentProviderEpay, "alipay"), ErrStripeSubscriptionDisabled)

	var stored SubscriptionOrder
	require.NoError(t, db.First(&stored, order.Id).Error)
	assert.Equal(t, common.TopUpStatusPending, stored.Status)
	var entitlements int64
	require.NoError(t, db.Model(&UserSubscription{}).Where("user_id = ?", 7202).Count(&entitlements).Error)
	assert.Zero(t, entitlements)
}

func TestMigrateDBRunsRecurringPreflightBeforeOtherMigrations(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv("GIN_MODE", "debug")
	t.Setenv(stripeSubscriptionAccountEnvTestKey, "")

	require.ErrorIs(t, migrateDB(), ErrStripeSubscriptionPlanInvalid)
	hasChannels := db.Migrator().HasTable(&Channel{})
	assert.False(t, hasChannels)
}

func TestMigrateDBReturnsPriceAmountMigrationErrors(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	originalDatabaseType := common.MainDatabaseType()
	t.Cleanup(func() { common.SetMainDatabaseType(originalDatabaseType) })
	common.SetMainDatabaseType(common.DatabaseTypeMySQL)

	err := migrateDB()
	assert.Error(t, err)
	var fixedCount int64
	require.NoError(t, db.Model(&SubscriptionPlan{}).
		Where("code = ?", SandboxStripeSubscriptionPlanCode).Count(&fixedCount).Error)
	assert.Zero(t, fixedCount)
}

func TestMigrateDBStopsBeforeSelectedPlanSyncWhenSQLitePlanAlterFails(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv("GIN_MODE", "debug")
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	require.NoError(t, db.Migrator().DropTable(&SubscriptionPlan{}))
	require.NoError(t, db.Exec("CREATE TABLE subscription_plans (id integer primary key, title varchar(128) NOT NULL, price_amount decimal(10,6) NOT NULL, currency varchar(8) NOT NULL, recurring_code varchar(64))").Error)
	require.NoError(t, db.Exec("INSERT INTO subscription_plans (id, title, price_amount, currency, recurring_code) VALUES (1, 'duplicate one', 1, 'USD', 'duplicate'), (2, 'duplicate two', 1, 'USD', 'duplicate')").Error)

	err := migrateDB()
	assert.Error(t, err)
	// The failed additive transaction must return before the selected-plan
	// synchronization step can add or update the fixed recurring contract.
	var planRows int64
	require.NoError(t, db.Table("subscription_plans").Count(&planRows).Error)
	assert.Equal(t, int64(2), planRows)
}

func TestEnsureStripeSubscriptionPlanRollsBackCreateWhenPostCreateSyncFails(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	require.NoError(t, db.Exec("CREATE TRIGGER stripe_plan_sync_failure BEFORE UPDATE ON subscription_plans BEGIN SELECT RAISE(ABORT, 'sync failure'); END").Error)

	require.Error(t, EnsureStripeSubscriptionPlanForEnvironment(SandboxStripeSubscriptionEnvironment))
	var count int64
	require.NoError(t, db.Model(&SubscriptionPlan{}).Where("code = ?", SandboxStripeSubscriptionPlanCode).Count(&count).Error)
	assert.Zero(t, count)
}

func TestSubscriptionPlanTxReadBypassesStaleGlobalCache(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	require.NoError(t, EnsureStripeSubscriptionPlanForEnvironment(SandboxStripeSubscriptionEnvironment))
	var plan SubscriptionPlan
	require.NoError(t, db.Where("code = ?", SandboxStripeSubscriptionPlanCode).First(&plan).Error)
	_, err := GetSubscriptionPlanById(plan.Id)
	require.NoError(t, err)
	require.NoError(t, db.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Updates(map[string]any{
		"enabled":                     false,
		"stripe_subscription_enabled": false,
	}).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		loaded, err := getSubscriptionPlanByIdTx(tx, plan.Id)
		if err != nil {
			return err
		}
		if loaded.Enabled || loaded.StripeSubscriptionEnabled {
			return errors.New("transaction read used stale subscription-plan cache")
		}
		return nil
	}))
}

func TestEnsureStripeSubscriptionPlanInvalidatesStalePlanCacheAfterCommit(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	require.NoError(t, EnsureStripeSubscriptionPlanForEnvironment(SandboxStripeSubscriptionEnvironment))
	var plan SubscriptionPlan
	require.NoError(t, db.Where("code = ?", SandboxStripeSubscriptionPlanCode).First(&plan).Error)
	_, err := GetSubscriptionPlanById(plan.Id)
	require.NoError(t, err)

	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "false")
	require.NoError(t, EnsureStripeSubscriptionPlanForEnvironment(SandboxStripeSubscriptionEnvironment))
	refreshed, err := GetSubscriptionPlanById(plan.Id)
	require.NoError(t, err)
	assert.False(t, refreshed.Enabled)
	assert.False(t, refreshed.StripeSubscriptionEnabled)
}

func TestValidateStripeSubscriptionPlanRejectsCrossEnvironmentCatalog(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)

	t.Setenv(stripeSubscriptionEnabledEnvTestKey, "true")
	testConfig, err := StripeSubscriptionConfigForEnvironment(SandboxStripeSubscriptionEnvironment)
	require.NoError(t, err)
	productionConfig, err := StripeSubscriptionConfigForEnvironment(ProductionStripeSubscriptionEnvironment)
	require.NoError(t, err)
	require.NoError(t, ValidateStripeSubscriptionPlan(plan, testConfig, true))
	assert.ErrorIs(t, ValidateStripeSubscriptionPlan(plan, productionConfig, true), ErrStripeSubscriptionPlanInvalid)
}

func TestEnsureProductionStripeSubscriptionPlanIsDisabledAndIdempotent(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	require.NoError(t, EnsureStripeSubscriptionPlanForEnvironment(ProductionStripeSubscriptionEnvironment))
	require.NoError(t, EnsureStripeSubscriptionPlanForEnvironment(ProductionStripeSubscriptionEnvironment))

	var plans []SubscriptionPlan
	require.NoError(t, db.Where("code = ?", SandboxStripeSubscriptionPlanCode).Find(&plans).Error)
	require.Len(t, plans, 1)
	assert.False(t, plans[0].Enabled)
	assert.False(t, plans[0].StripeSubscriptionEnabled)
	assert.Equal(t, ProductionStripeSubscriptionAccountID, plans[0].StripeAccountId)
	assert.Equal(t, ProductionStripeSubscriptionProductID, plans[0].StripeProductId)
	assert.Equal(t, ProductionStripeSubscriptionFounderPriceID, plans[0].FounderStripePriceId)
	assert.Equal(t, ProductionStripeSubscriptionStandardPriceID, plans[0].StandardStripePriceId)
	assert.Equal(t, ProductionStripeSubscriptionPortalConfigurationID, plans[0].StripePortalConfigurationId)
}

func TestEnsureStripeSubscriptionPlanMigratesExistingSandboxPlanToProduction(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	seedStripeSubscriptionPlan(t, db)

	require.NoError(t, EnsureStripeSubscriptionPlanForEnvironment(ProductionStripeSubscriptionEnvironment))

	var plan SubscriptionPlan
	require.NoError(t, db.Where("code = ?", SandboxStripeSubscriptionPlanCode).First(&plan).Error)
	assert.Equal(t, ProductionStripeSubscriptionAccountID, plan.StripeAccountId)
	assert.Equal(t, ProductionStripeSubscriptionProductID, plan.StripeProductId)
	assert.Equal(t, ProductionStripeSubscriptionFounderPriceID, plan.FounderStripePriceId)
	assert.Equal(t, ProductionStripeSubscriptionStandardPriceID, plan.StandardStripePriceId)
	assert.Equal(t, ProductionStripeSubscriptionPortalConfigurationID, plan.StripePortalConfigurationId)
}

func TestEnsureSandboxStripeSubscriptionPlanIsDisabledAndIdempotent(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	require.NoError(t, EnsureSandboxStripeSubscriptionPlan())
	require.NoError(t, EnsureSandboxStripeSubscriptionPlan())

	var plans []SubscriptionPlan
	require.NoError(t, db.Where("code = ?", SandboxStripeSubscriptionPlanCode).Find(&plans).Error)
	require.Len(t, plans, 1)
	assert.False(t, plans[0].Enabled)
	assert.False(t, plans[0].StripeSubscriptionEnabled)
	assert.Equal(t, SandboxStripeSubscriptionModel, plans[0].StripeSubscriptionModel)
	assert.Equal(t, SandboxStripeSubscriptionGroup, plans[0].UpgradeGroup)
	assert.Equal(t, int64(1999), plans[0].FounderAmountMinor)
	assert.Equal(t, int64(9999), plans[0].StandardAmountMinor)
	assert.Zero(t, plans[0].TotalAmount)
	require.NotNil(t, plans[0].AllowWalletOverflow)
	assert.False(t, *plans[0].AllowWalletOverflow)
}

func TestEnsureSandboxStripeSubscriptionPlanRejectsContractConflict(t *testing.T) {
	clearStripeSubscriptionEnv(t)
	db := setupStripeSubscriptionModelTestDB(t)
	plan := &SubscriptionPlan{
		Code:                        SandboxStripeSubscriptionPlanCode,
		Title:                       "conflicting sandbox plan",
		PriceAmount:                 19.99,
		Currency:                    "CNY",
		DurationUnit:                SubscriptionDurationMonth,
		DurationValue:               1,
		FounderStripePriceId:        "price_conflict",
		StandardStripePriceId:       SandboxStripeSubscriptionStandardPriceID,
		FounderAmountMinor:          1999,
		StandardAmountMinor:         9999,
		StripeSubscriptionModel:     SandboxStripeSubscriptionModel,
		StripeCurrency:              SandboxStripeSubscriptionCurrency,
		StripeProductId:             SandboxStripeSubscriptionProductID,
		StripeAccountId:             SandboxStripeSubscriptionAccountID,
		StripePortalConfigurationId: SandboxStripeSubscriptionPortalConfigurationID,
		MaxActiveSubscriptions:      20,
		FounderPurchaseLimit:        20,
		MaxActivePerUser:            1,
		UpgradeGroup:                SandboxStripeSubscriptionGroup,
		AllowWalletOverflow:         common.GetPointer(false),
	}
	require.NoError(t, db.Create(plan).Error)
	assert.ErrorIs(t, EnsureSandboxStripeSubscriptionPlan(), ErrStripeSubscriptionPlanInvalid)
}

func TestEnsureSubscriptionPlanTableSQLiteAddsRecurringColumns(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	require.NoError(t, db.Migrator().DropTable(&SubscriptionPlan{}))
	require.NoError(t, db.Exec("CREATE TABLE subscription_plans (id integer primary key, title varchar(128) NOT NULL, price_amount decimal(10,6) NOT NULL, currency varchar(8) NOT NULL)").Error)

	require.NoError(t, ensureSubscriptionPlanTableSQLite())
	for _, column := range []string{
		"code",
		"recurring_code",
		"stripe_subscription_enabled",
		"stripe_subscription_model",
		"max_active_subscriptions",
		"founder_purchase_limit",
		"max_active_per_user",
		"founder_stripe_price_id",
		"standard_stripe_price_id",
		"founder_amount_minor",
		"standard_amount_minor",
		"stripe_currency",
		"stripe_product_id",
		"stripe_account_id",
		"stripe_portal_configuration_id",
	} {
		assert.True(t, db.Migrator().HasColumn(&SubscriptionPlan{}, column), column)
	}
}

func TestRecurringPlanCodeUniqueIndexPreservesLegacyEmptyCodes(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	for _, title := range []string{"legacy one", "legacy two"} {
		require.NoError(t, db.Create(&SubscriptionPlan{
			Title:         title,
			PriceAmount:   1,
			Currency:      "USD",
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
		}).Error)
	}
	plan := seedStripeSubscriptionPlan(t, db)
	recurringCode := SandboxStripeSubscriptionPlanCode
	duplicate := &SubscriptionPlan{
		Code:          "another-recurring-row",
		RecurringCode: &recurringCode,
		Title:         "duplicate recurring row",
		PriceAmount:   plan.PriceAmount,
		Currency:      plan.Currency,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
	}
	assert.Error(t, db.Create(duplicate).Error)
}

func TestStripeWebhookEventLeaseRecoversAndFinalizes(t *testing.T) {
	setupStripeSubscriptionModelTestDB(t)
	first := &StripeWebhookEvent{EventID: "evt_lease_recovery", EventType: "invoice.paid"}
	claimed, terminal, err := ClaimStripeWebhookEvent(first, 100, time.Minute)
	require.NoError(t, err)
	assert.True(t, claimed)
	assert.False(t, terminal)
	assert.Equal(t, StripeWebhookEventProcessing, first.Status)

	duplicate := &StripeWebhookEvent{EventID: first.EventID, EventType: first.EventType}
	claimed, terminal, err = ClaimStripeWebhookEvent(duplicate, 110, time.Minute)
	require.NoError(t, err)
	assert.False(t, claimed)
	assert.True(t, terminal)

	retry := &StripeWebhookEvent{EventID: first.EventID, EventType: first.EventType}
	claimed, terminal, err = ClaimStripeWebhookEvent(retry, 161, time.Minute)
	require.NoError(t, err)
	assert.True(t, claimed)
	assert.False(t, terminal)
	assert.Equal(t, StripeWebhookEventProcessing, retry.Status)

	require.NoError(t, FinalizeStripeWebhookEvent(first.EventID, StripeWebhookEventProcessed, "", 162))
	final := &StripeWebhookEvent{EventID: first.EventID, EventType: first.EventType}
	claimed, terminal, err = ClaimStripeWebhookEvent(final, 163, time.Minute)
	require.NoError(t, err)
	assert.False(t, claimed)
	assert.True(t, terminal)
	assert.Equal(t, StripeWebhookEventProcessed, final.Status)
}

func TestExpireDueStripeSubscriptionsEndsGraceAndIsIdempotent(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 110)
	now := time.Now().Unix()
	reservation, err := ReserveStripeSubscriptionSeat(plan.Id, 110, "grace-expiry", now)
	require.NoError(t, err)
	require.NoError(t, db.Model(reservation).Updates(map[string]any{
		"status":                 StripeSubscriptionReservationActive,
		"expires_at":             0,
		"stripe_subscription_id": "sub_grace_expiry",
		"stripe_customer_id":     "cus_grace_expiry",
		"stripe_price_id":        plan.FounderStripePriceId,
		"activated_at":           now - 3600,
	}).Error)
	entitlement := &UserSubscription{
		UserId:        110,
		PlanId:        plan.Id,
		StartTime:     now - 3600,
		EndTime:       now + 3600,
		Status:        "active",
		Source:        "stripe_recurring",
		UpgradeGroup:  plan.UpgradeGroup,
		PrevUserGroup: "default",
	}
	require.NoError(t, db.Create(entitlement).Error)
	require.NoError(t, db.Model(&User{}).Where("id = ?", 110).Update("group", plan.UpgradeGroup).Error)
	recurring := &StripeSubscription{
		PlanId:               plan.Id,
		UserId:               110,
		ReservationId:        reservation.Id,
		StripeCustomerId:     "cus_grace_expiry",
		StripeSubscriptionId: "sub_grace_expiry",
		StripePriceId:        plan.FounderStripePriceId,
		UserSubscriptionId:   entitlement.Id,
		Tier:                 StripeSubscriptionTierFounder,
		Status:               StripeSubscriptionStatusPastDue,
		GraceUntil:           now - 1,
	}
	require.NoError(t, db.Create(recurring).Error)

	ended, err := ExpireDueStripeSubscriptions(now, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), ended)
	var refreshed StripeSubscription
	require.NoError(t, db.First(&refreshed, recurring.Id).Error)
	assert.Equal(t, StripeSubscriptionStatusCanceled, refreshed.Status)
	assert.Zero(t, refreshed.GraceUntil)
	var refreshedReservation StripeSubscriptionReservation
	require.NoError(t, db.First(&refreshedReservation, reservation.Id).Error)
	assert.Equal(t, StripeSubscriptionReservationReleased, refreshedReservation.Status)
	var refreshedEntitlement UserSubscription
	require.NoError(t, db.First(&refreshedEntitlement, entitlement.Id).Error)
	assert.Equal(t, "expired", refreshedEntitlement.Status)
	assert.Equal(t, now, refreshedEntitlement.EndTime)
	var user User
	require.NoError(t, db.First(&user, 110).Error)
	assert.Equal(t, "default", user.Group)

	ended, err = ExpireDueStripeSubscriptions(now, 20)
	require.NoError(t, err)
	assert.Zero(t, ended)
}

func TestReserveStripeSubscriptionSeatsCapsAtTwentyDistinctUsers(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// Let the test exercise competing SQLite connections. SQLite omits
	// FOR UPDATE, so the writer serialization/unique constraints must still
	// keep the result bounded without relying on a single pooled connection.
	sqlDB.SetMaxOpenConns(4)
	plan := seedStripeSubscriptionPlan(t, db)
	for id := 1; id <= 21; id++ {
		seedStripeSubscriptionUser(t, db, id)
	}

	results := make(chan error, 21)
	var wg sync.WaitGroup
	for id := 1; id <= 21; id++ {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := ReserveStripeSubscriptionSeat(plan.Id, id, fmt.Sprintf("reservation-%d", id), time.Now().Unix())
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	var successCount int
	var capacityErrors int
	for err := range results {
		if err == nil {
			successCount++
			continue
		}
		if assert.ErrorIs(t, err, ErrSubscriptionCapacityFull) {
			capacityErrors++
		}
	}
	assert.Equal(t, 20, successCount)
	assert.Equal(t, 1, capacityErrors)
	var pendingCount int64
	require.NoError(t, db.Model(&StripeSubscriptionReservation{}).Where("plan_id = ? AND status = ?", plan.Id, StripeSubscriptionReservationPending).Count(&pendingCount).Error)
	assert.Equal(t, int64(20), pendingCount)
}

func TestReserveStripeSubscriptionSeatRejectsDuplicateUser(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 101)

	_, err := ReserveStripeSubscriptionSeat(plan.Id, 101, "duplicate-1", time.Now().Unix())
	require.NoError(t, err)
	_, err = ReserveStripeSubscriptionSeat(plan.Id, 101, "duplicate-2", time.Now().Unix())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubscriptionAlreadyPending)
}

func TestExpireStripeSubscriptionReservationsReleasesPendingHold(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 102)
	now := time.Now().Unix()

	reservation, err := ReserveStripeSubscriptionSeat(plan.Id, 102, "expiry-1", now)
	require.NoError(t, err)
	require.NoError(t, db.Model(reservation).Updates(map[string]any{
		"expires_at": now - 1,
	}).Error)

	expired, err := ExpireStripeSubscriptionReservations(now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), expired)
	var refreshed StripeSubscriptionReservation
	require.NoError(t, db.First(&refreshed, reservation.Id).Error)
	assert.Equal(t, StripeSubscriptionReservationExpired, refreshed.Status)
}

func TestActivateAndReleaseKeepsFounderClaimConsumed(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 103)
	seedStripeSubscriptionUser(t, db, 104)
	now := time.Now().Unix()

	first, err := ReserveStripeSubscriptionSeat(plan.Id, 103, "founder-1", now)
	require.NoError(t, err)
	activated, err := ActivateStripeSubscriptionReservation(first.Id, "cs_founder_1", "cus_1", "sub_1", plan.FounderStripePriceId, now)
	require.NoError(t, err)
	assert.Equal(t, StripeSubscriptionTierFounder, activated.Tier)
	require.NoError(t, ReleaseStripeSubscriptionReservation(first.Id, now+1))

	second, err := ReserveStripeSubscriptionSeat(plan.Id, 103, "standard-1", now+2)
	require.NoError(t, err)
	activated, err = ActivateStripeSubscriptionReservation(second.Id, "cs_standard_1", "cus_1", "sub_2", plan.StandardStripePriceId, now+2)
	require.NoError(t, err)
	assert.Equal(t, StripeSubscriptionTierStandard, activated.Tier)

	var claimCount int64
	require.NoError(t, db.Model(&StripeSubscriptionFounderClaim{}).Where("plan_id = ? AND user_id = ?", plan.Id, 103).Count(&claimCount).Error)
	assert.Equal(t, int64(1), claimCount)

	third, err := ReserveStripeSubscriptionSeat(plan.Id, 104, "founder-2", now+3)
	require.NoError(t, err)
	activated, err = ActivateStripeSubscriptionReservation(third.Id, "cs_founder_2", "cus_2", "sub_3", plan.FounderStripePriceId, now+3)
	require.NoError(t, err)
	assert.Equal(t, StripeSubscriptionTierFounder, activated.Tier)
}

func TestActivateStripeSubscriptionReservationRejectsFounderPriceAfterClaimsSellOut(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 106)
	seedStripeSubscriptionUser(t, db, 107)
	now := time.Now().Unix()

	first, err := ReserveStripeSubscriptionSeat(plan.Id, 106, "sold-out-1", now)
	require.NoError(t, err)
	_, err = ActivateStripeSubscriptionReservation(first.Id, "cs_sold_out_1", "cus_106", "sub_sold_out_1", plan.FounderStripePriceId, now)
	require.NoError(t, err)
	for userID := 200; userID < 219; userID++ {
		require.NoError(t, db.Create(&StripeSubscriptionFounderClaim{
			PlanId: plan.Id,
			UserId: userID,
		}).Error)
	}

	second, err := ReserveStripeSubscriptionSeat(plan.Id, 107, "sold-out-2", now+1)
	require.NoError(t, err)
	_, err = ActivateStripeSubscriptionReservation(second.Id, "cs_sold_out_2", "cus_107", "sub_sold_out_2", plan.FounderStripePriceId, now+1)
	assert.ErrorIs(t, err, ErrStripeSubscriptionFounderSoldOut)
	var reservation StripeSubscriptionReservation
	require.NoError(t, db.First(&reservation, second.Id).Error)
	assert.Equal(t, StripeSubscriptionReservationPending, reservation.Status)
}

func TestActivateStripeSubscriptionReservationRequiresReservedPriceTier(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)
	seedStripeSubscriptionUser(t, db, 108)
	now := time.Now().Unix()

	reservation, err := ReserveStripeSubscriptionSeat(plan.Id, 108, "tier-mismatch-1", now)
	require.NoError(t, err)
	assert.Equal(t, StripeSubscriptionTierFounder, reservation.Tier)
	_, err = ActivateStripeSubscriptionReservation(reservation.Id, "cs_tier_mismatch", "cus_108", "sub_tier_mismatch", plan.StandardStripePriceId, now)
	assert.ErrorIs(t, err, ErrStripeSubscriptionPlanInvalid)
	var refreshed StripeSubscriptionReservation
	require.NoError(t, db.First(&refreshed, reservation.Id).Error)
	assert.Equal(t, StripeSubscriptionReservationPending, refreshed.Status)
}

func TestRecordStripeSubscriptionInvoiceIsIdempotent(t *testing.T) {
	db := setupStripeSubscriptionModelTestDB(t)
	plan := seedStripeSubscriptionPlan(t, db)

	first, err := RecordStripeSubscriptionInvoice(StripeSubscriptionInvoiceInput{
		PlanID:               plan.Id,
		UserID:               105,
		StripeSubscriptionID: "sub_invoice_1",
		StripeInvoiceID:      "in_invoice_1",
		EventID:              "evt_invoice_1",
		PeriodStart:          100,
		PeriodEnd:            200,
		AmountPaidMinor:      1999,
		Currency:             "cny",
	})
	require.NoError(t, err)
	assert.True(t, first)

	second, err := RecordStripeSubscriptionInvoice(StripeSubscriptionInvoiceInput{
		PlanID:               plan.Id,
		UserID:               105,
		StripeSubscriptionID: "sub_invoice_1",
		StripeInvoiceID:      "in_invoice_1",
		EventID:              "evt_invoice_1_replay",
		PeriodStart:          100,
		PeriodEnd:            200,
		AmountPaidMinor:      1999,
		Currency:             "cny",
	})
	require.NoError(t, err)
	assert.False(t, second)
	var count int64
	require.NoError(t, db.Model(&StripeSubscriptionInvoice{}).Where("stripe_subscription_id = ? AND stripe_invoice_id = ?", "sub_invoice_1", "in_invoice_1").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
