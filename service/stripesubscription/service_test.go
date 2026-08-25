package stripesubscription

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v85"
	"gorm.io/gorm"
)

type fakeStripeSubscriptionGateway struct {
	checkoutParams *stripe.CheckoutSessionParams
	portalParams   *stripe.BillingPortalSessionParams
	checkoutResult *stripe.CheckoutSession
	portalResult   *stripe.BillingPortalSession
	checkoutErr    error
	portalErr      error
	checkoutCalls  int
	portalCalls    int
	expireCalls    int
	expiredIDs     []string
}

func clearStripeSubscriptionServiceEnv(t *testing.T) {
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

func (f *fakeStripeSubscriptionGateway) CreateCheckoutSession(_ context.Context, params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	f.checkoutCalls++
	f.checkoutParams = params
	if f.checkoutErr != nil {
		return nil, f.checkoutErr
	}
	return f.checkoutResult, nil
}

func (f *fakeStripeSubscriptionGateway) ExpireCheckoutSession(_ context.Context, sessionID string) error {
	f.expireCalls++
	f.expiredIDs = append(f.expiredIDs, sessionID)
	return nil
}

func (f *fakeStripeSubscriptionGateway) CreatePortalSession(_ context.Context, params *stripe.BillingPortalSessionParams) (*stripe.BillingPortalSession, error) {
	f.portalCalls++
	f.portalParams = params
	if f.portalErr != nil {
		return nil, f.portalErr
	}
	return f.portalResult, nil
}

func setupStripeSubscriptionServiceDB(t *testing.T) (*gorm.DB, *model.SubscriptionPlan, *model.User) {
	t.Helper()
	clearStripeSubscriptionServiceEnv(t)
	originalDB := model.DB
	originalDatabaseType := common.MainDatabaseType()
	originalAPISecret := setting.StripeApiSecret
	originalWebhookSecret := setting.StripeWebhookSecret
	originalAccountID := setting.StripeAccountID
	originalRuntime := setting.StripeRuntimeEnvironment
	originalRequireTest := setting.StripeRequireTestKeys
	dsn := fmt.Sprintf("file:stripe-subscription-service-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	model.DB = db
	setting.StripeApiSecret = "sk_test_subscription_service"
	setting.StripeWebhookSecret = "whsec_subscription_service"
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
	user := &model.User{Username: "stripe-subscription-service-user", Email: "subscriber@example.com", StripeCustomer: "cus_existing"}
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

func TestRecurringRuntimeGateBlocksOfferCheckoutPortalAndWebhook(t *testing.T) {
	_, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{
		checkoutResult: &stripe.CheckoutSession{ID: "cs_gate", URL: "https://checkout.example/gate"},
		portalResult:   &stripe.BillingPortalSession{URL: "https://billing.example/gate"},
	}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	gateValues := []struct {
		name  string
		value *string
	}{
		{name: "unset"},
		{name: "false", value: common.GetPointer("false")},
		{name: "invalid", value: common.GetPointer("not-a-bool")},
		{name: "blank", value: common.GetPointer("")},
	}
	for _, gate := range gateValues {
		t.Run(gate.name, func(t *testing.T) {
			if gate.value == nil {
				_ = os.Unsetenv("STRIPE_SUBSCRIPTION_ENABLED")
			} else {
				t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", *gate.value)
			}

			_, err := GetStripeSubscriptionOffer(plan.Id, user.Id)
			require.ErrorIs(t, err, model.ErrStripeSubscriptionDisabled)
			_, err = CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
			require.ErrorIs(t, err, model.ErrStripeSubscriptionDisabled)
			_, err = CreatePortalSession(context.Background(), PortalInput{UserID: user.Id})
			require.Error(t, err)
			err = HandleRecurringEvent(context.Background(), recurringEvent("evt_gate_"+gate.name, stripe.EventTypeInvoicePaid, map[string]interface{}{
				"metadata": map[string]interface{}{"nova_subscription": "recurring"},
			}))
			// Lifecycle dispatch remains available for already-associated events;
			// this deliberately unassociated event must still fail closed.
			require.ErrorIs(t, err, ErrRecurringPaymentMismatch)
		})
	}
}

func TestPendingCheckoutSettlesAfterNewSaleGateIsDisabled(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{
		checkoutResult: &stripe.CheckoutSession{ID: "cs_disable_race", URL: "https://checkout.example/disable-race"},
	}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "false")
	require.NoError(t, db.Model(&model.SubscriptionPlan{}).Where("id = ?", plan.Id).
		Updates(map[string]any{"enabled": false, "stripe_subscription_enabled": false}).Error)

	event := completedCheckoutEvent(
		"evt_disable_race_paid",
		checkout.ReferenceID,
		"cs_disable_race",
		"sub_disable_race",
		plan.FounderStripePriceId,
		plan.FounderAmountMinor,
		plan.StripeSubscriptionModel,
	)
	event.Data.Object["metadata"].(map[string]interface{})["plan_id"] = strconv.Itoa(plan.Id)

	require.NoError(t, HandleRecurringEvent(context.Background(), event))
	var subscription model.StripeSubscription
	require.NoError(t, db.Where("stripe_subscription_id = ?", "sub_disable_race").First(&subscription).Error)
	assert.Equal(t, model.StripeSubscriptionStatusActive, subscription.Status)
	assert.NotZero(t, subscription.UserSubscriptionId)
}

func TestExistingSubscriptionInvoiceSettlesAfterNewSaleGateIsDisabled(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{
		checkoutResult: &stripe.CheckoutSession{ID: "cs_invoice_disable", URL: "https://checkout.example/invoice-disable"},
	}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	completed := completedCheckoutEvent(
		"evt_invoice_disable_checkout",
		checkout.ReferenceID,
		"cs_invoice_disable",
		"sub_invoice_disable",
		plan.FounderStripePriceId,
		plan.FounderAmountMinor,
		plan.StripeSubscriptionModel,
	)
	completed.Data.Object["metadata"].(map[string]interface{})["plan_id"] = strconv.Itoa(plan.Id)
	require.NoError(t, HandleRecurringEvent(context.Background(), completed))

	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "false")
	require.NoError(t, db.Model(&model.SubscriptionPlan{}).Where("id = ?", plan.Id).
		Updates(map[string]any{"enabled": false, "stripe_subscription_enabled": false}).Error)
	invoice := invoiceEvent("evt_invoice_disable_paid", string(stripe.EventTypeInvoicePaid), "sub_invoice_disable", "in_invoice_disable", plan.FounderStripePriceId, plan.FounderAmountMinor, 100, 200)
	require.NoError(t, HandleRecurringEvent(context.Background(), invoice))

	var stored model.StripeSubscriptionInvoice
	require.NoError(t, db.Where("stripe_invoice_id = ?", "in_invoice_disable").First(&stored).Error)
	assert.Equal(t, "paid", stored.Status)
}

func TestExistingActivePortalRemainsAvailableAfterNewSaleGateIsDisabled(t *testing.T) {
	_, plan, user := setupStripeSubscriptionServiceDB(t)
	require.NoError(t, model.DB.Create(&model.StripeSubscription{
		PlanId:               plan.Id,
		UserId:               user.Id,
		StripeCustomerId:     user.StripeCustomer,
		StripeSubscriptionId: "sub_portal_disable",
		StripePriceId:        plan.FounderStripePriceId,
		Tier:                 model.StripeSubscriptionTierFounder,
		Status:               model.StripeSubscriptionStatusActive,
	}).Error)
	fake := &fakeStripeSubscriptionGateway{portalResult: &stripe.BillingPortalSession{URL: "https://billing.example/disable"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "false")
	require.NoError(t, model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", plan.Id).
		Updates(map[string]any{"enabled": false, "stripe_subscription_enabled": false}).Error)

	result, err := CreatePortalSession(context.Background(), PortalInput{UserID: user.Id})
	require.NoError(t, err)
	assert.Equal(t, "https://billing.example/disable", result.URL)
	assert.Equal(t, 1, fake.portalCalls)
}

func TestDisabledRuntimeRejectsNewCheckoutWithoutCallingStripe(t *testing.T) {
	_, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_should_not_exist", URL: "https://checkout.example/should-not-exist"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "false")

	_, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.ErrorIs(t, err, model.ErrStripeSubscriptionDisabled)
	assert.Zero(t, fake.checkoutCalls)
}

func TestOfferSelectsFixedRuntimeCatalogPlanInsteadOfOtherEnabledRecurringRow(t *testing.T) {
	db, plan, _ := setupStripeSubscriptionServiceDB(t)
	otherCode := "other-recurring-contract"
	other := *plan
	other.Id = 0
	other.Code = otherCode
	other.RecurringCode = &otherCode
	other.SortOrder = -1
	require.NoError(t, db.Create(&other).Error)
	t.Setenv("STRIPE_SUBSCRIPTION_ENABLED", "true")

	offer, err := GetStripeSubscriptionOffer(0)
	require.NoError(t, err)
	assert.Equal(t, plan.Id, offer.PlanID)
	_, err = GetStripeSubscriptionOffer(other.Id)
	require.ErrorIs(t, err, model.ErrStripeSubscriptionPlanInvalid)
}

func TestSummaryKeepsFixedRuntimePlanWhenUserHasUnrelatedRecurringRow(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	otherCode := "other-summary-recurring-contract"
	other := *plan
	other.Id = 0
	other.Code = otherCode
	other.RecurringCode = &otherCode
	require.NoError(t, db.Create(&other).Error)
	require.NoError(t, db.Create(&model.StripeSubscription{
		PlanId:               other.Id,
		UserId:               user.Id,
		StripeSubscriptionId: "sub-unrelated-summary",
		StripePriceId:        other.FounderStripePriceId,
		Status:               model.StripeSubscriptionStatusActive,
	}).Error)

	summary, err := GetStripeSubscriptionSummary(user.Id)
	require.NoError(t, err)
	assert.Equal(t, plan.Id, summary.PlanID)
	assert.Equal(t, plan.Code, summary.PlanCode)
	assert.Equal(t, plan.StripeSubscriptionModel, summary.Model)
	payload, err := common.Marshal(summary)
	require.NoError(t, err)
	var encoded map[string]interface{}
	require.NoError(t, common.Unmarshal(payload, &encoded))
	assert.Equal(t, "all", encoded["model_scope"])
}

func recurringEvent(eventID string, eventType stripe.EventType, object map[string]interface{}) stripe.Event {
	return stripe.Event{
		ID:       eventID,
		Type:     eventType,
		Livemode: false,
		Account:  model.SandboxStripeSubscriptionAccountID,
		Data: &stripe.EventData{
			Object: object,
		},
	}
}

func completedCheckoutEvent(eventID, referenceID, sessionID, subscriptionID, priceID string, amount int64, modelName string) stripe.Event {
	return recurringEvent(eventID, stripe.EventTypeCheckoutSessionCompleted, map[string]interface{}{
		"id":                  sessionID,
		"client_reference_id": referenceID,
		"status":              "complete",
		"payment_status":      "paid",
		"customer":            "cus_existing",
		"subscription":        subscriptionID,
		"amount_total":        amount,
		"currency":            "cny",
		"metadata": map[string]interface{}{
			"plan_id":    "1",
			"model":      modelName,
			"price_id":   priceID,
			"product_id": model.SandboxStripeSubscriptionProductID,
		},
	})
}

func invoiceEvent(eventID, eventType, subscriptionID, invoiceID, priceID string, amount int64, periodStart, periodEnd int64) stripe.Event {
	return recurringEvent(eventID, stripe.EventType(eventType), map[string]interface{}{
		"id":           invoiceID,
		"subscription": subscriptionID,
		"amount_paid":  amount,
		"amount_due":   amount,
		"currency":     "cny",
		"period_start": periodStart,
		"period_end":   periodEnd,
		"lines": map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{
					"price": map[string]interface{}{"id": priceID},
				},
			},
		},
	})
}

func TestCreateCheckoutUsesHostedSubscriptionAndReservesOneFounderSeat(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_subscription_1", URL: "https://checkout.example/subscription"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	result, err := CreateCheckout(context.Background(), CheckoutInput{
		UserID:     user.Id,
		PlanID:     plan.Id,
		Email:      user.Email,
		CustomerID: user.StripeCustomer,
		SuccessURL: "https://novapura.example/console/subscription?status=success",
		CancelURL:  "https://novapura.example/console/subscription?status=cancel",
	})
	require.NoError(t, err)
	require.NotNil(t, fake.checkoutParams)
	require.NotNil(t, fake.checkoutParams.Mode)
	assert.Equal(t, string(stripe.CheckoutSessionModeSubscription), *fake.checkoutParams.Mode)
	require.Len(t, fake.checkoutParams.LineItems, 1)
	assert.Equal(t, int64(1), *fake.checkoutParams.LineItems[0].Quantity)
	assert.Equal(t, model.SandboxStripeSubscriptionFounderPriceID, *fake.checkoutParams.LineItems[0].Price)
	assert.Equal(t, model.SandboxStripeSubscriptionFounderLookupKey, fake.checkoutParams.Metadata["price_lookup_key"])
	assert.Nil(t, fake.checkoutParams.PaymentMethodTypes)
	require.NotNil(t, fake.checkoutParams.AutomaticTax)
	require.NotNil(t, fake.checkoutParams.AutomaticTax.Enabled)
	assert.False(t, *fake.checkoutParams.AutomaticTax.Enabled)
	require.NotNil(t, fake.checkoutParams.IntegrationIdentifier)
	integrationID := *fake.checkoutParams.IntegrationIdentifier
	require.GreaterOrEqual(t, len(integrationID), 8)
	assert.True(t, isASCIIAlpha(integrationID[len(integrationID)-8:]))
	require.NotNil(t, fake.checkoutParams.ExpiresAt)
	assert.InDelta(t, time.Now().Add(30*time.Minute).Unix(), *fake.checkoutParams.ExpiresAt, 90)
	assert.Equal(t, "https://checkout.example/subscription", result.PayLink)
	assert.Equal(t, model.StripeSubscriptionTierFounder, result.Tier)
	assert.Equal(t, model.StripeSubscriptionTierFounder, result.CurrentPriceTier)
	assert.Equal(t, model.SandboxStripeSubscriptionFounderPriceID, result.PriceID)
	assert.Equal(t, plan.Code, result.PlanCode)
	assert.Empty(t, result.Model)
	assert.Equal(t, "all", fake.checkoutParams.Metadata["model_scope"])
	assert.Empty(t, fake.checkoutParams.Metadata["model"])
	assert.Greater(t, result.ReservationExpiresAt, time.Now().Unix())
	assert.Equal(t, 1, fake.checkoutCalls)

	offer, err := GetStripeSubscriptionOffer(plan.Id)
	require.NoError(t, err)
	assert.True(t, offer.Active)
	assert.False(t, offer.Pending)
	assert.Equal(t, 20, offer.Limit)
	assert.Equal(t, int64(19), offer.Remaining)
	assert.False(t, offer.SoldOut)
	assert.Equal(t, model.StripeSubscriptionTierFounder, offer.CurrentPriceTier)
	assert.Equal(t, int64(1999), offer.CurrentPriceMinor)
	assert.Equal(t, int64(9999), offer.FutureStandardPriceMinor)

	var reservation model.StripeSubscriptionReservation
	require.NoError(t, db.Where("checkout_session_id = ?", "cs_subscription_1").First(&reservation).Error)
	assert.Equal(t, model.StripeSubscriptionReservationPending, reservation.Status)
	assert.Equal(t, "cs_subscription_1", reservation.CheckoutSessionId)
	assert.Equal(t, result.ReservationID, reservation.Id)
}

func TestCreateCheckoutUsesProductionCatalogOnlyWithLiveRuntime(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	config := model.DefaultProductionStripeSubscriptionConfig()
	t.Setenv("GIN_MODE", "release")
	require.NoError(t, db.Model(plan).Updates(map[string]any{
		"founder_stripe_price_id":        config.FounderPriceID,
		"standard_stripe_price_id":       config.StandardPriceID,
		"stripe_product_id":              config.ProductID,
		"stripe_account_id":              config.AccountID,
		"stripe_portal_configuration_id": config.PortalConfigurationID,
	}).Error)
	require.NoError(t, db.First(plan, plan.Id).Error)
	setting.StripeApiSecret = "sk_live_subscription_service"
	setting.StripeAccountID = config.AccountID
	setting.StripeRuntimeEnvironment = setting.StripeRuntimeProduction
	setting.StripeRequireTestKeys = false

	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_live_subscription_1", URL: "https://checkout.example/live-subscription"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	result, err := CreateCheckout(context.Background(), CheckoutInput{
		UserID: user.Id,
		PlanID: plan.Id,
		Email:  user.Email,
	})
	require.NoError(t, err)
	require.NotNil(t, fake.checkoutParams)
	require.Len(t, fake.checkoutParams.LineItems, 1)
	assert.Equal(t, config.FounderPriceID, *fake.checkoutParams.LineItems[0].Price)
	assert.Equal(t, setting.StripeRuntimeProduction, fake.checkoutParams.Metadata["stripe_environment"])
	assert.Equal(t, config.AccountID, fake.checkoutParams.Metadata["stripe_account_id"])
	assert.Equal(t, config.ProductID, fake.checkoutParams.Metadata["product_id"])
	assert.Equal(t, config.FounderPriceID, fake.checkoutParams.Metadata["price_id"])
	assert.Equal(t, config.FounderPriceID, result.PriceID)

	completed := completedCheckoutEvent(
		"evt_live_checkout_completed",
		result.ReferenceID,
		"cs_live_subscription_1",
		"sub_live_subscription_1",
		config.FounderPriceID,
		config.FounderAmountMinor,
		config.Model,
	)
	completed.Livemode = true
	completed.Account = config.AccountID
	metadata := completed.Data.Object["metadata"].(map[string]interface{})
	metadata["plan_id"] = strconv.Itoa(plan.Id)
	metadata["product_id"] = config.ProductID
	metadata["stripe_environment"] = config.Environment
	metadata["stripe_account_id"] = config.AccountID
	require.NoError(t, HandleRecurringEvent(context.Background(), completed))

	var subscription model.StripeSubscription
	require.NoError(t, db.Where("stripe_subscription_id = ?", "sub_live_subscription_1").First(&subscription).Error)
	assert.Equal(t, model.StripeSubscriptionStatusActive, subscription.Status)
}

func TestCreateCheckoutFailureReleasesReservationSeat(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutErr: errors.New("stripe unavailable")}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	_, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.Error(t, err)
	var pendingCount int64
	require.NoError(t, db.Model(&model.StripeSubscriptionReservation{}).
		Where("plan_id = ? AND user_id = ? AND status = ?", plan.Id, user.Id, model.StripeSubscriptionReservationPending).
		Count(&pendingCount).Error)
	assert.Zero(t, pendingCount)
	var reconciliationCount int64
	require.NoError(t, db.Model(&model.StripeSubscriptionReservation{}).
		Where("plan_id = ? AND user_id = ? AND status = ?", plan.Id, user.Id, model.StripeSubscriptionReservationReconciliation).
		Count(&reconciliationCount).Error)
	assert.Zero(t, reconciliationCount)
	var released model.StripeSubscriptionReservation
	require.NoError(t, db.Where("plan_id = ? AND user_id = ? AND status = ?", plan.Id, user.Id, model.StripeSubscriptionReservationReleased).First(&released).Error)
	assert.Nil(t, released.ActiveUserId)
}

func TestCreateCheckoutRetryReusesPersistedSessionAndIdempotencyKey(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_retry", URL: "https://checkout.example/retry"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	first, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	require.NotNil(t, fake.checkoutParams)
	require.NotNil(t, fake.checkoutParams.IdempotencyKey)
	key := *fake.checkoutParams.IdempotencyKey
	assert.NotEmpty(t, key)
	var reservation model.StripeSubscriptionReservation
	require.NoError(t, db.First(&reservation, first.ReservationID).Error)
	assert.Equal(t, key, reservation.IdempotencyKey)

	// Simulate an older local row that lost the key during a partial migration;
	// retry must repair it before returning the already-created hosted session.
	require.NoError(t, db.Model(&reservation).Update("idempotency_key", "").Error)
	second, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	assert.Equal(t, first.PayLink, second.PayLink)
	assert.Equal(t, first.ReservationID, second.ReservationID)
	assert.Equal(t, 1, fake.checkoutCalls)
	require.NoError(t, db.First(&reservation, first.ReservationID).Error)
	assert.Equal(t, key, reservation.IdempotencyKey)
}

func TestOfferPublishesAuthoritativeFairUseContract(t *testing.T) {
	_, plan, _ := setupStripeSubscriptionServiceDB(t)

	offer, err := GetStripeSubscriptionOffer(plan.Id)
	require.NoError(t, err)
	assert.False(t, offer.UserStateKnown)
	assert.True(t, offer.CheckoutAllowed)
	assert.Empty(t, offer.Model)
	assert.Equal(t, 10, offer.FairUse.PeakConcurrency)
	assert.Equal(t, int64(600), offer.FairUse.RollingWindowSeconds)
	assert.Equal(t, int64(1800), offer.FairUse.ConcurrentSecondsBudget)
	assert.Equal(t, 600, offer.FairUse.SuccessfulRequests)
	assert.Equal(t, 750, offer.FairUse.AdmittedRequests)
	assert.Equal(t, int64(15), offer.FairUse.HeartbeatIntervalSeconds)
	assert.Equal(t, int64(45), offer.FairUse.StaleLeaseRecoverySeconds)
	assert.True(t, offer.FairUse.HeartbeatRequired)
	assert.True(t, offer.FairUse.StaleLeaseRecoveryEnabled)
	assert.Equal(t, NoResaleCopyIdentifier, offer.FairUse.NoResaleCopyID)
	assert.Equal(t, NoResaleCopyIdentifier, offer.FairUse.NoResaleCopyIdentifier)

	payload, err := common.Marshal(offer)
	require.NoError(t, err)
	var encoded map[string]interface{}
	require.NoError(t, common.Unmarshal(payload, &encoded))
	fairUse, ok := encoded["fair_use"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(10), fairUse["peak_concurrency"])
	assert.Equal(t, float64(600), fairUse["rolling_window_seconds"])
	assert.Equal(t, float64(1800), fairUse["concurrent_seconds_budget"])
	assert.Equal(t, float64(600), fairUse["successful_requests"])
	assert.Equal(t, float64(750), fairUse["admitted_requests"])
	assert.Equal(t, float64(15), fairUse["heartbeat_interval_seconds"])
	assert.Equal(t, float64(45), fairUse["stale_lease_recovery_seconds"])
	assert.Equal(t, NoResaleCopyIdentifier, fairUse["no_resale_copy_id"])
	assert.Equal(t, "all", encoded["model_scope"])
}

func TestFounderClaimedUserSeesAuthoritativeStandardOffer(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	require.NoError(t, db.Create(&model.StripeSubscriptionFounderClaim{
		PlanId: plan.Id,
		UserId: user.Id,
	}).Error)

	offer, err := GetStripeSubscriptionOffer(plan.Id, user.Id)
	require.NoError(t, err)
	assert.Equal(t, model.StripeSubscriptionTierStandard, offer.CurrentPriceTier)
	assert.Equal(t, int64(9999), offer.CurrentPriceMinor)
	assert.Equal(t, int64(19), offer.FounderClaimsRemaining)
}

func TestOfferAndSummaryExposeUserDuplicateCheckoutState(t *testing.T) {
	_, plan, user := setupStripeSubscriptionServiceDB(t)
	reservation, err := model.ReserveStripeSubscriptionSeat(plan.Id, user.Id, "pending-duplicate-state", common.GetTimestamp())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = model.ReleaseStripeSubscriptionReservation(reservation.Id, common.GetTimestamp())
	})

	offer, err := GetStripeSubscriptionOffer(plan.Id, user.Id)
	require.NoError(t, err)
	assert.True(t, offer.UserStateKnown)
	assert.True(t, offer.AlreadyPending)
	assert.False(t, offer.AlreadyActive)
	assert.False(t, offer.CheckoutAllowed)
	assert.Equal(t, reservation.Id, offer.PendingReservationID)
	assert.Equal(t, reservation.ExpiresAt, offer.ReservationExpiresAt)

	summary, err := GetStripeSubscriptionSummary(user.Id)
	require.NoError(t, err)
	assert.True(t, summary.AlreadyPending)
	assert.False(t, summary.AlreadyActive)
	assert.False(t, summary.CheckoutAllowed)
	assert.Equal(t, reservation.Id, summary.PendingReservationID)
	assert.Equal(t, reservation.ExpiresAt, summary.ReservationExpiresAt)
	assert.Equal(t, int64(600), summary.FairUse.RollingWindowSeconds)
}

func TestRecurringLifecycleIsIdempotentAndKeepsOneEntitlement(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_lifecycle", URL: "https://checkout.example/lifecycle"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	completed := completedCheckoutEvent("evt_checkout_lifecycle", checkout.ReferenceID, "cs_lifecycle", "sub_lifecycle", plan.FounderStripePriceId, plan.FounderAmountMinor, plan.StripeSubscriptionModel)
	completed.Data.Object["metadata"].(map[string]interface{})["plan_id"] = fmt.Sprintf("%d", plan.Id)
	require.NoError(t, HandleRecurringEvent(context.Background(), completed))
	require.NoError(t, HandleRecurringEvent(context.Background(), completed))
	var userRecord model.User
	require.NoError(t, db.First(&userRecord, user.Id).Error)
	assert.Equal(t, model.SandboxStripeSubscriptionGroup, userRecord.Group)

	var recurring model.StripeSubscription
	require.NoError(t, db.Where("stripe_subscription_id = ?", "sub_lifecycle").First(&recurring).Error)
	assert.Equal(t, model.StripeSubscriptionTierFounder, recurring.Tier)
	summary, err := GetStripeSubscriptionSummary(user.Id)
	require.NoError(t, err)
	assert.Equal(t, model.StripeSubscriptionStatusActive, summary.StripeStatus)
	assert.Equal(t, plan.FounderStripePriceId, summary.StripePriceID)
	assert.Equal(t, model.StripeSubscriptionTierFounder, summary.CurrentPriceTier)
	assert.Equal(t, int64(1999), summary.CurrentPriceMinor)
	assert.True(t, summary.AlreadyActive)
	assert.False(t, summary.AlreadyPending)
	assert.False(t, summary.CheckoutAllowed)
	var entitlementCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Count(&entitlementCount).Error)
	assert.Equal(t, int64(1), entitlementCount)

	periodStart := time.Now().Unix()
	periodEnd := periodStart + 60
	paid := invoiceEvent("evt_invoice_lifecycle", "invoice.paid", "sub_lifecycle", "in_lifecycle", plan.FounderStripePriceId, plan.FounderAmountMinor, periodStart, periodEnd)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", user.Id).Update("group", "manually-changed").Error)
	require.NoError(t, HandleRecurringEvent(context.Background(), paid))
	require.NoError(t, HandleRecurringEvent(context.Background(), paid))
	paidReplay := invoiceEvent("evt_invoice_lifecycle_replay", "invoice.paid", "sub_lifecycle", "in_lifecycle", plan.FounderStripePriceId, plan.FounderAmountMinor, periodStart, periodEnd)
	require.NoError(t, HandleRecurringEvent(context.Background(), paidReplay))

	var invoiceCount int64
	require.NoError(t, db.Model(&model.StripeSubscriptionInvoice{}).Where("stripe_invoice_id = ?", "in_lifecycle").Count(&invoiceCount).Error)
	assert.Equal(t, int64(1), invoiceCount)
	require.NoError(t, db.First(&recurring, recurring.Id).Error)
	assert.Equal(t, periodEnd, recurring.CurrentPeriodEnd)
	var entitlement model.UserSubscription
	require.NoError(t, db.First(&entitlement, recurring.UserSubscriptionId).Error)
	assert.Equal(t, periodEnd, entitlement.EndTime)

	deleted := recurringEvent("evt_subscription_deleted", stripe.EventTypeCustomerSubscriptionDeleted, map[string]interface{}{
		"id":                 "sub_lifecycle",
		"customer":           "cus_existing",
		"status":             "canceled",
		"current_period_end": periodEnd,
	})
	require.NoError(t, HandleRecurringEvent(context.Background(), deleted))
	require.NoError(t, db.First(&recurring, recurring.Id).Error)
	assert.Equal(t, model.StripeSubscriptionStatusCanceled, recurring.Status)
	require.NoError(t, db.First(&entitlement, entitlement.Id).Error)
	assert.Equal(t, "expired", entitlement.Status)
	require.NoError(t, db.First(&userRecord, user.Id).Error)
	assert.Equal(t, "default", userRecord.Group)
	var reservation model.StripeSubscriptionReservation
	require.NoError(t, db.First(&reservation, checkout.ReservationID).Error)
	assert.Equal(t, model.StripeSubscriptionReservationReleased, reservation.Status)
	var claimCount int64
	require.NoError(t, db.Model(&model.StripeSubscriptionFounderClaim{}).Where("plan_id = ? AND user_id = ?", plan.Id, user.Id).Count(&claimCount).Error)
	assert.Equal(t, int64(1), claimCount)
}

func TestUnpaidCheckoutBindsBeforeInvoicePaidActivates(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_unpaid", URL: "https://checkout.example/unpaid"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	completed := completedCheckoutEvent("evt_checkout_unpaid", checkout.ReferenceID, "cs_unpaid", "sub_unpaid", plan.FounderStripePriceId, plan.FounderAmountMinor, plan.StripeSubscriptionModel)
	completed.Data.Object["payment_status"] = "unpaid"
	completed.Data.Object["metadata"].(map[string]interface{})["plan_id"] = fmt.Sprintf("%d", plan.Id)
	require.NoError(t, HandleRecurringEvent(context.Background(), completed))

	var reservation model.StripeSubscriptionReservation
	require.NoError(t, db.First(&reservation, checkout.ReservationID).Error)
	assert.Equal(t, model.StripeSubscriptionReservationPending, reservation.Status)
	var bound model.StripeSubscription
	require.NoError(t, db.Where("stripe_subscription_id = ?", "sub_unpaid").First(&bound).Error)
	assert.Equal(t, model.StripeSubscriptionStatusIncomplete, bound.Status)
	assert.Zero(t, bound.UserSubscriptionId)

	paid := invoiceEvent("evt_invoice_unpaid_recovered", "invoice.paid", "sub_unpaid", "in_unpaid", plan.FounderStripePriceId, plan.FounderAmountMinor, time.Now().Unix(), time.Now().Add(time.Hour).Unix())
	require.NoError(t, HandleRecurringEvent(context.Background(), paid))
	require.NoError(t, db.First(&bound, bound.Id).Error)
	assert.Equal(t, model.StripeSubscriptionStatusActive, bound.Status)
	assert.NotZero(t, bound.UserSubscriptionId)
	require.NoError(t, db.First(&reservation, reservation.Id).Error)
	assert.Equal(t, model.StripeSubscriptionReservationActive, reservation.Status)
	var entitlementCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Count(&entitlementCount).Error)
	assert.Equal(t, int64(1), entitlementCount)
}

func TestInvoicePaidFirstReconcilesFromSubscriptionMetadata(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_invoice_first", URL: "https://checkout.example/invoice-first"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	paid := invoiceEvent("evt_invoice_first", "invoice.paid", "sub_invoice_first", "in_invoice_first", plan.FounderStripePriceId, plan.FounderAmountMinor, time.Now().Unix(), time.Now().Add(time.Hour).Unix())
	paid.Data.Object["metadata"] = map[string]interface{}{
		"nova_subscription":        "recurring",
		"stripe_environment":       model.SandboxStripeSubscriptionEnvironment,
		"stripe_account_id":        model.SandboxStripeSubscriptionAccountID,
		"plan_id":                  fmt.Sprintf("%d", plan.Id),
		"model":                    plan.StripeSubscriptionModel,
		"product_id":               plan.StripeProductId,
		"price_id":                 plan.FounderStripePriceId,
		"reservation_id":           fmt.Sprintf("%d", checkout.ReservationID),
		"reservation_reference_id": checkout.ReferenceID,
	}
	require.NoError(t, HandleRecurringEvent(context.Background(), paid))
	var subscription model.StripeSubscription
	require.NoError(t, db.Where("stripe_subscription_id = ?", "sub_invoice_first").First(&subscription).Error)
	assert.Equal(t, model.StripeSubscriptionStatusActive, subscription.Status)
	assert.NotZero(t, subscription.UserSubscriptionId)
	var reservation model.StripeSubscriptionReservation
	require.NoError(t, db.First(&reservation, checkout.ReservationID).Error)
	assert.Equal(t, model.StripeSubscriptionReservationActive, reservation.Status)
}

func TestInvoiceFirstInvalidMetadataDoesNotBindSubscription(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_invoice_invalid_metadata", URL: "https://checkout.example/invoice-invalid-metadata"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	paid := invoiceEvent("evt_invoice_invalid_metadata", "invoice.paid", "sub_invoice_invalid_metadata", "in_invoice_invalid_metadata", plan.FounderStripePriceId, plan.FounderAmountMinor, time.Now().Unix(), time.Now().Add(time.Hour).Unix())
	paid.Data.Object["metadata"] = map[string]interface{}{
		"stripe_environment":       model.SandboxStripeSubscriptionEnvironment,
		"stripe_account_id":        model.SandboxStripeSubscriptionAccountID,
		"plan_id":                  strconv.Itoa(plan.Id),
		"model":                    plan.StripeSubscriptionModel,
		"product_id":               "prod_wrong_for_invoice",
		"price_id":                 plan.FounderStripePriceId,
		"reservation_id":           strconv.FormatInt(checkout.ReservationID, 10),
		"reservation_reference_id": checkout.ReferenceID,
	}

	err = HandleRecurringEvent(context.Background(), paid)
	require.ErrorIs(t, err, ErrRecurringPaymentMismatch)
	var subscriptionCount int64
	require.NoError(t, db.Model(&model.StripeSubscription{}).Count(&subscriptionCount).Error)
	assert.Zero(t, subscriptionCount)
}

func TestRecurringCheckoutRejectsWrongPriceLookupMetadata(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_lookup_mismatch", URL: "https://checkout.example/lookup-mismatch"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	event := completedCheckoutEvent("evt_lookup_mismatch", checkout.ReferenceID, "cs_lookup_mismatch", "sub_lookup_mismatch", plan.FounderStripePriceId, plan.FounderAmountMinor, plan.StripeSubscriptionModel)
	event.Data.Object["metadata"].(map[string]interface{})["plan_id"] = fmt.Sprintf("%d", plan.Id)
	event.Data.Object["metadata"].(map[string]interface{})["price_lookup_key"] = "novapura_deepseek_v4_flash_standard_cny_monthly_v1"

	err = HandleRecurringEvent(context.Background(), event)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecurringPaymentMismatch)
	var entitlementCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Count(&entitlementCount).Error)
	assert.Zero(t, entitlementCount)
}

func TestRecurringMismatchNeverGrantsEntitlement(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_mismatch", URL: "https://checkout.example/mismatch"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)
	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	event := completedCheckoutEvent("evt_checkout_mismatch", checkout.ReferenceID, "cs_mismatch", "sub_mismatch", plan.FounderStripePriceId, 1, plan.StripeSubscriptionModel)
	event.Data.Object["metadata"].(map[string]interface{})["plan_id"] = fmt.Sprintf("%d", plan.Id)
	err = HandleRecurringEvent(context.Background(), event)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecurringPaymentMismatch)
	var recurringCount int64
	require.NoError(t, db.Model(&model.StripeSubscription{}).Count(&recurringCount).Error)
	assert.Zero(t, recurringCount)
	var entitlementCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Count(&entitlementCount).Error)
	assert.Zero(t, entitlementCount)
}

func TestEmptyDirectAccountRequiresVerifiedWebhookContext(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_empty_account", URL: "https://checkout.example/empty-account"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	event := completedCheckoutEvent("evt_empty_account", checkout.ReferenceID, "cs_empty_account", "sub_empty_account", plan.FounderStripePriceId, plan.FounderAmountMinor, plan.StripeSubscriptionModel)
	event.Data.Object["metadata"].(map[string]interface{})["plan_id"] = fmt.Sprintf("%d", plan.Id)
	event.Account = ""
	err = HandleRecurringEvent(context.Background(), event)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecurringPaymentMismatch)

	wrongAccount := event
	wrongAccount.ID = "evt_wrong_account"
	wrongAccount.Account = "acct_not_novapura"
	err = HandleRecurringEvent(context.Background(), wrongAccount)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecurringPaymentMismatch)
	var manual model.StripeWebhookEvent
	require.NoError(t, db.Where("event_id = ?", wrongAccount.ID).First(&manual).Error)
	assert.Equal(t, model.StripeWebhookEventManualReview, manual.Status)

	// The first rejection happened before claiming the event, so a verified
	// controller context can safely retry the same signed event.
	require.NoError(t, HandleRecurringEvent(WithVerifiedWebhookContext(context.Background()), event))
	var count int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestInvoicePaymentFailureUsesGraceAndPaidRecoveryIsIdempotent(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	fake := &fakeStripeSubscriptionGateway{checkoutResult: &stripe.CheckoutSession{ID: "cs_grace", URL: "https://checkout.example/grace"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	checkout, err := CreateCheckout(context.Background(), CheckoutInput{UserID: user.Id, PlanID: plan.Id, Email: user.Email})
	require.NoError(t, err)
	completed := completedCheckoutEvent("evt_checkout_grace", checkout.ReferenceID, "cs_grace", "sub_grace", plan.FounderStripePriceId, plan.FounderAmountMinor, plan.StripeSubscriptionModel)
	completed.Data.Object["metadata"].(map[string]interface{})["plan_id"] = fmt.Sprintf("%d", plan.Id)
	require.NoError(t, HandleRecurringEvent(context.Background(), completed))

	periodStart := time.Now().Unix()
	periodEnd := periodStart + 60
	var graceEntitlement model.UserSubscription
	var graceSubscription model.StripeSubscription
	require.NoError(t, db.Where("stripe_subscription_id = ?", "sub_grace").First(&graceSubscription).Error)
	require.NoError(t, db.First(&graceEntitlement, graceSubscription.UserSubscriptionId).Error)
	require.NoError(t, db.Model(&graceEntitlement).Update("end_time", periodEnd).Error)
	failure := invoiceEvent("evt_invoice_failed", "invoice.payment_failed", "sub_grace", "in_grace", plan.FounderStripePriceId, plan.FounderAmountMinor, periodStart, periodEnd)
	failure.Data.Object["amount_paid"] = int64(0)
	require.NoError(t, HandleRecurringEvent(context.Background(), failure))
	var recurring model.StripeSubscription
	require.NoError(t, db.Where("stripe_subscription_id = ?", "sub_grace").First(&recurring).Error)
	assert.Equal(t, model.StripeSubscriptionStatusPastDue, recurring.Status)
	assert.InDelta(t, time.Now().Add(model.StripeSubscriptionGracePeriod).Unix(), recurring.GraceUntil, 2)
	graceUntil := recurring.GraceUntil
	require.NoError(t, db.First(&graceEntitlement, recurring.UserSubscriptionId).Error)
	assert.Equal(t, graceUntil, graceEntitlement.EndTime)

	failureReplay := invoiceEvent("evt_invoice_failed_replay", "invoice.payment_failed", "sub_grace", "in_grace", plan.FounderStripePriceId, plan.FounderAmountMinor, periodStart, periodEnd)
	require.NoError(t, HandleRecurringEvent(context.Background(), failureReplay))
	require.NoError(t, db.First(&recurring, recurring.Id).Error)
	assert.Equal(t, graceUntil, recurring.GraceUntil)

	paid := invoiceEvent("evt_invoice_recovered", "invoice.paid", "sub_grace", "in_grace", plan.FounderStripePriceId, plan.FounderAmountMinor, periodStart, periodEnd)
	require.NoError(t, HandleRecurringEvent(context.Background(), paid))
	require.NoError(t, HandleRecurringEvent(context.Background(), paid))
	require.NoError(t, db.First(&recurring, recurring.Id).Error)
	assert.Equal(t, model.StripeSubscriptionStatusActive, recurring.Status)
	assert.Zero(t, recurring.GraceUntil)
	var invoice model.StripeSubscriptionInvoice
	require.NoError(t, db.Where("stripe_invoice_id = ?", "in_grace").First(&invoice).Error)
	assert.Equal(t, "paid", invoice.Status)
}

func TestPortalUsesOwnedCustomerAndFixedSandboxConfiguration(t *testing.T) {
	db, plan, user := setupStripeSubscriptionServiceDB(t)
	require.NoError(t, db.Create(&model.StripeSubscription{
		PlanId:               plan.Id,
		UserId:               user.Id,
		StripeCustomerId:     "cus_owned",
		StripeSubscriptionId: "sub_owned",
		StripePriceId:        plan.FounderStripePriceId,
		Tier:                 model.StripeSubscriptionTierFounder,
		Status:               model.StripeSubscriptionStatusActive,
	}).Error)
	fake := &fakeStripeSubscriptionGateway{portalResult: &stripe.BillingPortalSession{URL: "https://billing.example/portal"}}
	restore := SetGatewayForTest(fake)
	t.Cleanup(restore)

	result, err := CreatePortalSession(context.Background(), PortalInput{UserID: user.Id, ReturnURL: "https://novapura.example/console/subscription"})
	require.NoError(t, err)
	require.NotNil(t, fake.portalParams)
	assert.Equal(t, "cus_owned", *fake.portalParams.Customer)
	assert.Equal(t, model.SandboxStripeSubscriptionPortalConfigurationID, *fake.portalParams.Configuration)
	assert.Equal(t, "https://novapura.example/console/subscription", *fake.portalParams.ReturnURL)
	assert.Equal(t, "https://billing.example/portal", result.URL)

	other := &model.User{Username: "stripe-subscription-other", AffCode: "stripe-subscription-other-aff"}
	require.NoError(t, db.Create(other).Error)
	_, err = CreatePortalSession(context.Background(), PortalInput{UserID: other.Id, ReturnURL: "https://novapura.example/console/subscription"})
	require.Error(t, err)
	assert.Equal(t, 1, fake.portalCalls)
}

func isASCIIAlpha(value string) bool {
	if len(value) != 8 {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}
