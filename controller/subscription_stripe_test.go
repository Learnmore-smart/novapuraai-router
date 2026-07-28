package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stripe/stripe-go/v85"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupSubscriptionStripeTestDB initialises an in-memory SQLite DB with the
// tables the subscription webhook handler touches. Mirrors the pattern in
// stripetopup/webhook_idempotency_test.go.
func setupSubscriptionStripeTestDB(t *testing.T) {
	t.Helper()
	// Initialize model column-name variables (commonGroupCol etc.). These are
	// set by initCol(), which runs as a side effect of model.InitDB(). Without
	// this, queries like getUserGroupByIdTx produce "SELECT  FROM users..."
	// (empty column list) because commonGroupCol is the zero-value "".
	initModelListColumnNames(t)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:sub-stripe-%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
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
		&model.SubscriptionOrder{},
		&model.UserSubscription{},
		&model.SubscriptionCoupon{},
		&model.SubscriptionCouponRedemption{},
		&model.StripeWebhookEvent{},
		&model.TopUp{},
		&model.Log{},
	))
	model.DB = db
	model.LOG_DB = db
}

// ---- Pure logic tests (no DB) ----

func TestIsSubscriptionStripeEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    stripe.Event
		expected bool
	}{
		{
			name:     "invoice.paid is subscription event",
			event:    stripe.Event{Type: stripe.EventTypeInvoicePaid},
			expected: true,
		},
		{
			name:     "invoice.payment_failed is subscription event",
			event:    stripe.Event{Type: stripe.EventTypeInvoicePaymentFailed},
			expected: true,
		},
		{
			name:     "customer.subscription.updated is subscription event",
			event:    stripe.Event{Type: stripe.EventTypeCustomerSubscriptionUpdated},
			expected: true,
		},
		{
			name:     "customer.subscription.deleted is subscription event",
			event:    stripe.Event{Type: stripe.EventTypeCustomerSubscriptionDeleted},
			expected: true,
		},
		{
			name: "checkout.session.completed with auto_renew metadata is subscription event",
			event: stripe.Event{
				Type: stripe.EventTypeCheckoutSessionCompleted,
				Data: &stripe.EventData{
					Raw: json.RawMessage(`{"metadata":{"novapura_mode":"auto_renew"}}`),
				},
			},
			expected: true,
		},
		{
			name: "checkout.session.completed with prepaid metadata is subscription event",
			event: stripe.Event{
				Type: stripe.EventTypeCheckoutSessionCompleted,
				Data: &stripe.EventData{
					Raw: json.RawMessage(`{"metadata":{"novapura_mode":"prepaid"}}`),
				},
			},
			expected: true,
		},
		{
			name: "checkout.session.completed without novapura metadata is NOT subscription event",
			event: stripe.Event{
				Type: stripe.EventTypeCheckoutSessionCompleted,
				Data: &stripe.EventData{
					Raw: json.RawMessage(`{"metadata":{}}`),
				},
			},
			expected: false,
		},
		{
			name: "checkout.session.expired with auto_renew metadata is subscription event",
			event: stripe.Event{
				Type: stripe.EventTypeCheckoutSessionExpired,
				Data: &stripe.EventData{
					Raw: json.RawMessage(`{"metadata":{"novapura_mode":"auto_renew"}}`),
				},
			},
			expected: true,
		},
		{
			name: "checkout.session.expired with prepaid metadata is subscription event",
			event: stripe.Event{
				Type: stripe.EventTypeCheckoutSessionExpired,
				Data: &stripe.EventData{
					Raw: json.RawMessage(`{"metadata":{"novapura_mode":"prepaid"}}`),
				},
			},
			expected: true,
		},
		{
			name: "checkout.session.expired without novapura metadata is NOT subscription event",
			event: stripe.Event{
				Type: stripe.EventTypeCheckoutSessionExpired,
				Data: &stripe.EventData{
					Raw: json.RawMessage(`{"metadata":{}}`),
				},
			},
			expected: false,
		},
		{
			name:     "charge.refunded is NOT subscription event",
			event:    stripe.Event{Type: stripe.EventTypeChargeRefunded},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Populate the event Data.Object so GetObjectValue works for
			// metadata-based routing tests.
			if tt.event.Data != nil && tt.event.Data.Raw != nil {
				var obj map[string]any
				if err := json.Unmarshal(tt.event.Data.Raw, &obj); err == nil {
					tt.event.Data.Object = obj
				}
			}
			assert.Equal(t, tt.expected, IsSubscriptionStripeEvent(tt.event))
		})
	}
}

func TestValidateCheckoutRequest(t *testing.T) {
	enabledPlan := &model.SubscriptionPlan{
		Enabled:          true,
		StripePriceIdCNY: "price_cny_xxx",
		StripePriceIdUSD: "price_usd_xxx",
		StripeProductId:  "prod_xxx",
		PrepaidMonths:    "1,3,6,12",
	}

	tests := []struct {
		name    string
		req     *SubscriptionStripeCheckoutRequest
		plan    *model.SubscriptionPlan
		wantErr string
	}{
		{
			name: "auto_renew USD valid",
			req:  &SubscriptionStripeCheckoutRequest{Mode: subscriptionCheckoutModeAutoRenew, Currency: "USD"},
			plan: enabledPlan,
		},
		{
			name: "auto_renew CNY valid",
			req:  &SubscriptionStripeCheckoutRequest{Mode: subscriptionCheckoutModeAutoRenew, Currency: "CNY"},
			plan: enabledPlan,
		},
		{
			name: "prepaid valid with 6 months",
			req:  &SubscriptionStripeCheckoutRequest{Mode: subscriptionCheckoutModePrepaid, Currency: "USD", PrepaidMonths: 6},
			plan: enabledPlan,
		},
		{
			name: "disabled plan rejected",
			req:  &SubscriptionStripeCheckoutRequest{Mode: subscriptionCheckoutModeAutoRenew, Currency: "USD"},
			plan: &model.SubscriptionPlan{Enabled: false, StripePriceIdUSD: "price_x"},
			wantErr: "plan is not enabled",
		},
		{
			name: "invalid mode rejected",
			req:  &SubscriptionStripeCheckoutRequest{Mode: "invalid", Currency: "USD"},
			plan: enabledPlan,
			wantErr: "invalid mode",
		},
		{
			name: "invalid currency rejected",
			req:  &SubscriptionStripeCheckoutRequest{Mode: subscriptionCheckoutModeAutoRenew, Currency: "EUR"},
			plan: enabledPlan,
			wantErr: "invalid currency",
		},
		{
			name: "auto_renew without Stripe price rejected",
			req:  &SubscriptionStripeCheckoutRequest{Mode: subscriptionCheckoutModeAutoRenew, Currency: "USD"},
			plan: &model.SubscriptionPlan{Enabled: true, StripePriceIdCNY: "price_cny"},
			wantErr: "does not have a Stripe USD price",
		},
		{
			name: "prepaid without product id rejected",
			req:  &SubscriptionStripeCheckoutRequest{Mode: subscriptionCheckoutModePrepaid, Currency: "USD", PrepaidMonths: 3},
			plan: &model.SubscriptionPlan{Enabled: true, StripePriceIdUSD: "price_usd"},
			wantErr: "does not have a Stripe product id",
		},
		{
			name: "prepaid with 0 months rejected",
			req:  &SubscriptionStripeCheckoutRequest{Mode: subscriptionCheckoutModePrepaid, Currency: "USD", PrepaidMonths: 0},
			plan: enabledPlan,
			wantErr: "prepaid_months must be > 0",
		},
		{
			name: "prepaid with disallowed months rejected",
			req:  &SubscriptionStripeCheckoutRequest{Mode: subscriptionCheckoutModePrepaid, Currency: "USD", PrepaidMonths: 5},
			plan: enabledPlan,
			wantErr: "is not in plan's allowed list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCheckoutRequest(tt.req, tt.plan)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsPrepaidMonthsAllowed(t *testing.T) {
	assert.True(t, isPrepaidMonthsAllowed("1,3,6,12", 1))
	assert.True(t, isPrepaidMonthsAllowed("1,3,6,12", 12))
	assert.True(t, isPrepaidMonthsAllowed(" 1 , 3 , 6 ", 3))
	assert.False(t, isPrepaidMonthsAllowed("1,3,6,12", 5))
	assert.False(t, isPrepaidMonthsAllowed("", 1))
	assert.False(t, isPrepaidMonthsAllowed("abc", 1))
}

func TestComputePrepaidAmountMinor(t *testing.T) {
	plan := &model.SubscriptionPlan{
		PriceAmountUSD: 19.99,
		PriceAmountCNY: 149.00,
	}

	// No coupon: original == final, discount == 0
	original, discount, final := computePrepaidAmountMinor(plan, "USD", 3, 0)
	assert.Equal(t, int64(1999*3), original)
	assert.Equal(t, int64(0), discount)
	assert.Equal(t, original, final)

	// 20% off coupon on 6 months USD. computePrepaidAmountMinor uses
	// common.QuotaFromDecimal which rounds half-away-from-zero, so
	// round(11994 * 20 / 100) = round(2398.8) = 2399 (not truncated 2398).
	original, discount, final = computePrepaidAmountMinor(plan, "USD", 6, 20)
	assert.Equal(t, int64(1999*6), original)
	assert.Equal(t, int64(2399), discount)
	assert.Equal(t, original-discount, final)
	assert.True(t, final > 0)

	// CNY with 50% off, 12 months
	original, discount, final = computePrepaidAmountMinor(plan, "CNY", 12, 50)
	assert.Equal(t, int64(14900*12), original)
	assert.Equal(t, int64(14900*12*50/100), discount)
	assert.Equal(t, original-discount, final)

	// 100% off → final is 0 (but the validator rejects percentOff >= 100 at the
	// coupon level; the function itself clamps to 0 safely)
	original, discount, final = computePrepaidAmountMinor(plan, "USD", 1, 100)
	assert.Equal(t, int64(1999), original)
	assert.Equal(t, int64(0), discount, "percentOff >= 100 is treated as no discount")
	assert.Equal(t, original, final)
}

// ---- Webhook handler tests (with DB) ----

// checkoutCompletedV2Event builds a checkout.session.completed event with
// NovaPura v2 metadata, mirroring what the v2 checkout handler creates.
func checkoutCompletedV2Event(eventID, orderID, mode string) stripe.Event {
	payload := map[string]any{
		"id":                  "cs_" + eventID,
		"object":              "checkout.session",
		"client_reference_id": orderID,
		"status":              "complete",
		"payment_status":      "paid",
		"currency":            "usd",
		"customer":            "cus_" + eventID,
		"metadata": map[string]any{
			"novapura_order_id": orderID,
			"novapura_mode":     mode,
			"novapura_user_id":  "1",
		},
	}
	// Auto-renew checkouts create a Stripe Subscription; prepaid (payment mode)
	// checkouts create a PaymentIntent instead. The webhook handler uses the
	// subscription id for auto_renew and falls back to payment_intent for prepaid.
	if mode == subscriptionCheckoutModeAutoRenew {
		payload["subscription"] = "sub_" + eventID
	} else {
		payload["payment_intent"] = "pi_" + eventID
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		panic(err)
	}
	// Unmarshal into map[string]any so nested maps (metadata) have the dynamic
	// type stripe's getValue expects (map[string]interface{}). Setting Object to
	// the typed payload directly works only when every nested map is also
	// map[string]any; unmarshaling guarantees that, mirroring stripe's own
	// EventData.UnmarshalJSON.
	var obj map[string]any
	if err := common.Unmarshal(raw, &obj); err != nil {
		panic(err)
	}
	return stripe.Event{
		ID:       eventID,
		Livemode: false,
		Type:     stripe.EventTypeCheckoutSessionCompleted,
		Data: &stripe.EventData{
			Raw:    json.RawMessage(raw),
			Object: obj,
		},
	}
}

func invoicePaidEvent(eventID, stripeSubId string, periodEnd int64) stripe.Event {
	payload := map[string]any{
		"id":           "in_" + eventID,
		"object":       "invoice",
		"subscription": stripeSubId,
		"period_end":   periodEnd,
		"status":       "paid",
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return stripe.Event{
		ID:       eventID,
		Livemode: false,
		Type:     stripe.EventTypeInvoicePaid,
		Data: &stripe.EventData{
			Raw:    json.RawMessage(raw),
			Object: payload,
		},
	}
}

func subscriptionDeletedEvent(eventID, stripeSubId string) stripe.Event {
	payload := map[string]any{
		"id":     stripeSubId,
		"object": "subscription",
		"status": "canceled",
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return stripe.Event{
		ID:       eventID,
		Livemode: false,
		Type:     stripe.EventTypeCustomerSubscriptionDeleted,
		Data: &stripe.EventData{
			Raw:    json.RawMessage(raw),
			Object: payload,
		},
	}
}

func subscriptionUpdatedEvent(eventID, stripeSubId, stripeStatus string, cancelAtPeriodEnd bool, periodEnd int64) stripe.Event {
	payload := map[string]any{
		"id":                   stripeSubId,
		"object":               "subscription",
		"status":               stripeStatus,
		"cancel_at_period_end": cancelAtPeriodEnd,
		"current_period_end":   periodEnd,
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return stripe.Event{
		ID:       eventID,
		Livemode: false,
		Type:     stripe.EventTypeCustomerSubscriptionUpdated,
		Data: &stripe.EventData{
			Raw:    json.RawMessage(raw),
			Object: payload,
		},
	}
}

func invoicePaymentFailedEvent(eventID, stripeSubId string) stripe.Event {
	payload := map[string]any{
		"id":           "in_" + eventID,
		"object":       "invoice",
		"subscription": stripeSubId,
		"status":       "open",
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return stripe.Event{
		ID:       eventID,
		Livemode: false,
		Type:     stripe.EventTypeInvoicePaymentFailed,
		Data: &stripe.EventData{
			Raw:    json.RawMessage(raw),
			Object: payload,
		},
	}
}

func TestProcessSubscriptionStripeEvent_CheckoutCompletedAutoRenew(t *testing.T) {
	setupSubscriptionStripeTestDB(t)
	originalRequireTest := setting.StripeRequireTestKeys
	originalAccountID := setting.StripeAccountID
	t.Cleanup(func() {
		setting.StripeRequireTestKeys = originalRequireTest
		setting.StripeAccountID = originalAccountID
	})
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	user := &model.User{Username: "sub-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{
		Title:        "NovaPura Pro",
		Enabled:      true,
		DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   1_000_000,
		PriceAmountUSD: 19.99,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	order := &model.SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "sub_ref_test_1",
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}
	require.NoError(t, model.DB.Create(order).Error)

	event := checkoutCompletedV2Event("evt_sub_1", "sub_ref_test_1", subscriptionCheckoutModeAutoRenew)
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))

	// Verify the order was completed.
	var refreshedOrder model.SubscriptionOrder
	require.NoError(t, model.DB.Where("trade_no = ?", "sub_ref_test_1").First(&refreshedOrder).Error)
	assert.Equal(t, common.TopUpStatusSuccess, refreshedOrder.Status)
	assert.Equal(t, "sub_evt_sub_1", refreshedOrder.StripeSubscriptionId)

	// Verify a UserSubscription was created.
	var sub model.UserSubscription
	require.NoError(t, model.DB.Where("user_id = ?", user.Id).First(&sub).Error)
	assert.Equal(t, model.SubscriptionStatusActive, sub.Status)
	assert.Equal(t, "sub_evt_sub_1", sub.StripeSubscriptionId)
	assert.Equal(t, "cus_evt_sub_1", sub.StripeCustomerId)

	// Verify the user's StripeCustomer was persisted.
	var refreshedUser model.User
	require.NoError(t, model.DB.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, "cus_evt_sub_1", refreshedUser.StripeCustomer)

	// Idempotency: replaying the same event must not create a second subscription.
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))
	var count int64
	require.NoError(t, model.DB.Model(&model.UserSubscription{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestProcessSubscriptionStripeEvent_CheckoutCompletedPrepaid(t *testing.T) {
	setupSubscriptionStripeTestDB(t)
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	user := &model.User{Username: "prepaid-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{
		Title:         "NovaPura Prepaid",
		Enabled:       true,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   1_000_000,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	order := &model.SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "sub_ref_prepaid_1",
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
		PrepaidMonths:   3,
	}
	require.NoError(t, model.DB.Create(order).Error)

	event := checkoutCompletedV2Event("evt_prepaid_1", "sub_ref_prepaid_1", subscriptionCheckoutModePrepaid)
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))

	var sub model.UserSubscription
	require.NoError(t, model.DB.Where("user_id = ?", user.Id).First(&sub).Error)
	assert.Equal(t, model.SubscriptionStatusPrepaidActive, sub.Status)
	// Prepaid 3 months × 1 month duration = ~3 months end time
	assert.True(t, sub.EndTime > sub.StartTime, "prepaid end time must be in the future")

	// For prepaid mode, the linkage key is the payment_intent (no Stripe subscription).
	assert.Equal(t, "pi_evt_prepaid_1", sub.StripeSubscriptionId)
}

func TestProcessSubscriptionStripeEvent_InvoicePaidRenewsSubscription(t *testing.T) {
	setupSubscriptionStripeTestDB(t)
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	user := &model.User{Username: "renew-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{
		Title:           "NovaPura Renew",
		Enabled:         true,
		DurationUnit:    model.SubscriptionDurationMonth,
		DurationValue:   1,
		TotalAmount:     1_000_000,
		QuotaResetPeriod: model.SubscriptionResetMonthly,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	// Create a subscription with a placeholder EndTime (as the checkout handler would).
	oldEndTime := common.GetTimestamp() + 86400
	sub := &model.UserSubscription{
		UserId:               user.Id,
		PlanId:               plan.Id,
		Status:               model.SubscriptionStatusActive,
		StartTime:            common.GetTimestamp() - 86400,
		EndTime:              oldEndTime,
		AmountUsed:           500_000,
		StripeSubscriptionId: "sub_renew_1",
	}
	require.NoError(t, model.DB.Create(sub).Error)

	newPeriodEnd := common.GetTimestamp() + 2592000 // ~30 days from now
	event := invoicePaidEvent("evt_invoice_1", "sub_renew_1", newPeriodEnd)
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))

	var refreshed model.UserSubscription
	require.NoError(t, model.DB.First(&refreshed, sub.Id).Error)
	assert.Equal(t, newPeriodEnd, refreshed.EndTime)
	assert.Equal(t, int64(0), refreshed.AmountUsed, "renewal must reset AmountUsed")
	assert.Equal(t, model.SubscriptionStatusActive, refreshed.Status)

	// Idempotency: replaying the same event must not change EndTime again.
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))
	require.NoError(t, model.DB.First(&refreshed, sub.Id).Error)
	assert.Equal(t, newPeriodEnd, refreshed.EndTime)
}

func TestProcessSubscriptionStripeEvent_InvoicePaymentFailedMarksPastDue(t *testing.T) {
	setupSubscriptionStripeTestDB(t)
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	user := &model.User{Username: "pastdue-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{
		Title:         "NovaPura PastDue",
		Enabled:       true,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	sub := &model.UserSubscription{
		UserId:               user.Id,
		PlanId:               plan.Id,
		Status:               model.SubscriptionStatusActive,
		StartTime:            common.GetTimestamp() - 86400,
		EndTime:              common.GetTimestamp() + 2592000,
		StripeSubscriptionId: "sub_pastdue_1",
	}
	require.NoError(t, model.DB.Create(sub).Error)

	event := invoicePaymentFailedEvent("evt_pfail_1", "sub_pastdue_1")
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))

	var refreshed model.UserSubscription
	require.NoError(t, model.DB.First(&refreshed, sub.Id).Error)
	assert.Equal(t, model.SubscriptionStatusPastDue, refreshed.Status)
}

func TestProcessSubscriptionStripeEvent_SubscriptionDeletedCancels(t *testing.T) {
	setupSubscriptionStripeTestDB(t)
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	user := &model.User{Username: "cancel-user", Password: "password123", Group: "premium"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{
		Title:         "NovaPura Cancel",
		Enabled:       true,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		UpgradeGroup:  "premium",
	}
	require.NoError(t, model.DB.Create(plan).Error)

	sub := &model.UserSubscription{
		UserId:               user.Id,
		PlanId:               plan.Id,
		Status:               model.SubscriptionStatusActive,
		StartTime:            common.GetTimestamp() - 86400,
		EndTime:              common.GetTimestamp() + 2592000,
		StripeSubscriptionId: "sub_cancel_1",
		UpgradeGroup:         "premium",
		PrevUserGroup:        "default",
	}
	require.NoError(t, model.DB.Create(sub).Error)

	event := subscriptionDeletedEvent("evt_del_1", "sub_cancel_1")
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))

	var refreshed model.UserSubscription
	require.NoError(t, model.DB.First(&refreshed, sub.Id).Error)
	assert.Equal(t, model.SubscriptionStatusCanceled, refreshed.Status)
	assert.True(t, refreshed.EndTime <= common.GetTimestamp()+5, "canceled subscription end_time must be ~now")

	// Idempotency: replaying deletion must not error.
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))
}

func TestProcessSubscriptionStripeEvent_SubscriptionUpdatedCancelAtPeriodEnd(t *testing.T) {
	setupSubscriptionStripeTestDB(t)
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	user := &model.User{Username: "updating-user", Password: "password123"}
	require.NoError(t, model.DB.Create(user).Error)

	plan := &model.SubscriptionPlan{
		Title:         "NovaPura Update",
		Enabled:       true,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	sub := &model.UserSubscription{
		UserId:               user.Id,
		PlanId:               plan.Id,
		Status:               model.SubscriptionStatusActive,
		StartTime:            common.GetTimestamp() - 86400,
		EndTime:              common.GetTimestamp() + 2592000,
		StripeSubscriptionId: "sub_update_1",
	}
	require.NoError(t, model.DB.Create(sub).Error)

	periodEnd := common.GetTimestamp() + 2592000
	event := subscriptionUpdatedEvent("evt_upd_1", "sub_update_1", "active", true, periodEnd)
	require.NoError(t, ProcessSubscriptionStripeEvent(context.Background(), event))

	var refreshed model.UserSubscription
	require.NoError(t, model.DB.First(&refreshed, sub.Id).Error)
	assert.Equal(t, model.SubscriptionStatusCanceling, refreshed.Status)
	assert.True(t, refreshed.CancelAtPeriodEnd, "cancel_at_period_end must be true")
}

func TestProcessSubscriptionStripeEvent_IgnoresUnknownSubscription(t *testing.T) {
	setupSubscriptionStripeTestDB(t)
	setting.StripeRequireTestKeys = true
	setting.StripeAccountID = ""

	// invoice.paid for a subscription that doesn't exist locally → ignored, no error
	event := invoicePaidEvent("evt_unknown_1", "sub_not_ours", common.GetTimestamp()+2592000)
	err := ProcessSubscriptionStripeEvent(context.Background(), event)
	require.NoError(t, err, "unknown subscription events must be silently ignored")

	// The webhook event claim must still be persisted (we processed it successfully as "ignored").
	var claimed int64
	require.NoError(t, model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", "evt_unknown_1").Count(&claimed).Error)
	assert.EqualValues(t, 1, claimed)
}

func TestProcessSubscriptionStripeEvent_LivemodeRejectedInSandbox(t *testing.T) {
	setupSubscriptionStripeTestDB(t)
	originalRequireTest := setting.StripeRequireTestKeys
	t.Cleanup(func() { setting.StripeRequireTestKeys = originalRequireTest })
	setting.StripeRequireTestKeys = true

	event := stripe.Event{
		ID:       "evt_live_1",
		Livemode: true,
		Type:     stripe.EventTypeInvoicePaid,
	}
	err := ProcessSubscriptionStripeEvent(context.Background(), event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "livemode")

	// The webhook event claim must NOT be persisted (processing failed before insertion
	// because the mode guard runs first).
	var claimed int64
	require.NoError(t, model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", "evt_live_1").Count(&claimed).Error)
	assert.Zero(t, claimed)
}
