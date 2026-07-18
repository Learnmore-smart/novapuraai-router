package stripetopup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v85"
	checkoutsession "github.com/stripe/stripe-go/v85/checkout/session"
	"github.com/stripe/stripe-go/v85/webhook"
	"gorm.io/gorm"
)

type sandboxE2ESession struct {
	Label              string   `json:"label"`
	Currency           string   `json:"currency"`
	AmountMinor        int64    `json:"amount_minor"`
	OrderID            string   `json:"order_id"`
	SessionID          string   `json:"session_id"`
	CheckoutURL        string   `json:"checkout_url"`
	PaymentStatus      string   `json:"payment_status,omitempty"`
	PaymentMethodTypes []string `json:"payment_method_types,omitempty"`
	ObservedMethods    []string `json:"observed_payment_methods,omitempty"`
	BrowserResult      string   `json:"browser_result,omitempty"`
}

type sandboxE2EArtifact struct {
	DatabasePath          string              `json:"database_path"`
	UserID                int                 `json:"user_id"`
	BalanceBefore         int                 `json:"balance_before"`
	PromotionalBefore     int                 `json:"promotional_before"`
	BalanceAfter          int                 `json:"balance_after,omitempty"`
	PromotionalAfter      int                 `json:"promotional_after,omitempty"`
	SignatureVerified     bool                `json:"signature_verified,omitempty"`
	DuplicateReplayStable bool                `json:"duplicate_replay_stable,omitempty"`
	Sessions              []sandboxE2ESession `json:"sessions"`
}

func sandboxE2EPaths(t *testing.T) (string, string) {
	t.Helper()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	temporaryDirectory := filepath.Join(repositoryRoot, ".tmp")
	require.NoError(t, os.MkdirAll(temporaryDirectory, 0o755))
	return filepath.Join(temporaryDirectory, "stripe-sandbox-e2e.db"), filepath.Join(temporaryDirectory, "stripe-sandbox-e2e.json")
}

func openSandboxE2EDatabase(t *testing.T, databasePath string, migrate bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	require.NoError(t, err)
	if migrate {
		require.NoError(t, db.AutoMigrate(
			&model.User{},
			&model.StripeTopupOrder{},
			&model.BalanceLedger{},
			&model.BalanceCreditLot{},
			&model.StripeWebhookEvent{},
			&model.TopupPromoTier{},
			&model.TopupPromotionCampaign{},
			&model.TopupPromoRedemption{},
			&model.TopUp{},
			&model.Log{},
		))
	}
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	return db
}

func loadSandboxStripeEnvironment(t *testing.T) {
	t.Helper()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	require.NoError(t, godotenv.Overload(filepath.Join(repositoryRoot, ".env")))
	t.Setenv("GIN_MODE", "debug")
	setting.InitStripeEnv()
	setting.StripeTopupEnabled = true
}

func TestStripeSandboxCreateCheckoutSessions(t *testing.T) {
	if os.Getenv("STRIPE_SANDBOX_E2E_CREATE") != "1" {
		t.Skip("set STRIPE_SANDBOX_E2E_CREATE=1 to create real sandbox Checkout Sessions")
	}
	databasePath, artifactPath := sandboxE2EPaths(t)
	require.NoError(t, os.RemoveAll(databasePath))
	require.NoError(t, os.RemoveAll(artifactPath))
	db := openSandboxE2EDatabase(t, databasePath, true)
	loadSandboxStripeEnvironment(t)
	require.NoError(t, model.SeedLaunchTopupPromotion(db))

	user := &model.User{Username: "stripe-sandbox-e2e", Password: "not-used", Quota: 0, PromoQuota: 0}
	require.NoError(t, db.Create(user).Error)
	testCases := []struct {
		label       string
		currency    string
		amountMinor int64
	}{
		{label: "cad_0_50", currency: "cad", amountMinor: 50},
		{label: "usd_0_50", currency: "usd", amountMinor: 50},
		{label: "cny_5_00", currency: "cny", amountMinor: 500},
		{label: "cny_10_00_promo", currency: "cny", amountMinor: 1000},
	}
	artifact := sandboxE2EArtifact{DatabasePath: databasePath, UserID: user.Id}
	for _, testCase := range testCases {
		result, err := CreateCheckout(user, QuoteRequest{Currency: testCase.currency, AmountMinor: testCase.amountMinor}, "https://example.com/stripe-success", "https://example.com/stripe-cancel")
		require.NoError(t, err)
		order, err := model.GetStripeTopupOrderByOrderID(result.OrderID)
		require.NoError(t, err)
		artifact.Sessions = append(artifact.Sessions, sandboxE2ESession{
			Label:       testCase.label,
			Currency:    testCase.currency,
			AmountMinor: testCase.amountMinor,
			OrderID:     result.OrderID,
			SessionID:   order.StripeCheckoutSessionID,
			CheckoutURL: result.CheckoutURL,
		})
	}
	artifact.BalanceBefore = user.Quota
	artifact.PromotionalBefore = user.PromoQuota
	payload, err := common.Marshal(artifact)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(artifactPath, payload, 0o600))
	t.Logf("sandbox Checkout artifact: %s", artifactPath)
	for _, session := range artifact.Sessions {
		t.Logf("%s session=%s url=%s", session.Label, session.SessionID, session.CheckoutURL)
	}
}

func TestStripeSandboxSettleCompletedSessions(t *testing.T) {
	if os.Getenv("STRIPE_SANDBOX_E2E_SETTLE") != "1" {
		t.Skip("set STRIPE_SANDBOX_E2E_SETTLE=1 after completing the sandbox Checkout Sessions")
	}
	databasePath, artifactPath := sandboxE2EPaths(t)
	db := openSandboxE2EDatabase(t, databasePath, false)
	loadSandboxStripeEnvironment(t)
	payload, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	artifact := sandboxE2EArtifact{}
	require.NoError(t, common.Unmarshal(payload, &artifact))

	stripe.Key = setting.StripeApiSecret
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var promoEvent stripe.Event
	for index := range artifact.Sessions {
		sessionRecord := &artifact.Sessions[index]
		session, err := checkoutsession.Get(sessionRecord.SessionID, nil)
		require.NoError(t, err)
		require.Equal(t, stripe.CheckoutSessionPaymentStatusPaid, session.PaymentStatus)
		sessionRecord.PaymentStatus = string(session.PaymentStatus)
		sessionRecord.PaymentMethodTypes = nil
		for _, paymentMethodType := range session.PaymentMethodTypes {
			sessionRecord.PaymentMethodTypes = append(sessionRecord.PaymentMethodTypes, string(paymentMethodType))
		}
		if len(sessionRecord.ObservedMethods) == 0 {
			sessionRecord.ObservedMethods = append([]string{}, sessionRecord.PaymentMethodTypes...)
		}
		if sessionRecord.BrowserResult == "" {
			sessionRecord.BrowserResult = "paid_redirect_observed"
		}

		sessionPayload, err := common.Marshal(session)
		require.NoError(t, err)
		eventPayload, err := common.Marshal(map[string]any{
			"id":          fmt.Sprintf("evt_codex_%d_%s", index, session.ID),
			"object":      "event",
			"account":     setting.StripeAccountID,
			"api_version": stripe.APIVersion,
			"created":     time.Now().Unix(),
			"livemode":    false,
			"type":        string(stripe.EventTypeCheckoutSessionCompleted),
			"data":        map[string]json.RawMessage{"object": sessionPayload},
		})
		require.NoError(t, err)
		signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{Payload: eventPayload, Secret: setting.StripeWebhookSecret})
		event, err := webhook.ConstructEventWithOptions(signed.Payload, signed.Header, setting.StripeWebhookSecret, webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
		require.NoError(t, err)
		artifact.SignatureVerified = true
		require.NoError(t, ProcessVerifiedEvent(ctx, event))
		if sessionRecord.Label == "cny_10_00_promo" {
			promoEvent = event
		}
	}
	require.NotEmpty(t, promoEvent.ID)
	var beforeReplay model.User
	require.NoError(t, db.First(&beforeReplay, artifact.UserID).Error)
	require.NoError(t, ProcessVerifiedEvent(ctx, promoEvent))
	var afterReplay model.User
	require.NoError(t, db.First(&afterReplay, artifact.UserID).Error)
	artifact.DuplicateReplayStable = assert.Equal(t, beforeReplay.Quota, afterReplay.Quota) && assert.Equal(t, beforeReplay.PromoQuota, afterReplay.PromoQuota)
	artifact.BalanceAfter = afterReplay.Quota
	artifact.PromotionalAfter = afterReplay.PromoQuota

	updatedPayload, err := common.Marshal(artifact)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(artifactPath, updatedPayload, 0o600))
	t.Logf("balance before=%d promo_before=%d after=%d promo_after=%d duplicate_stable=%v", artifact.BalanceBefore, artifact.PromotionalBefore, artifact.BalanceAfter, artifact.PromotionalAfter, artifact.DuplicateReplayStable)
}
