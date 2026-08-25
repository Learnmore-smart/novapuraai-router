package stripesubscription

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/deepseekfairuse"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stripe/stripe-go/v85"
	portalsession "github.com/stripe/stripe-go/v85/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v85/checkout/session"
	"gorm.io/gorm"
)

var (
	ErrRecurringPaymentMismatch = errors.New("recurring payment mismatch")
	ErrRecurringEventNotHandled = errors.New("recurring event not handled")
	ErrRecurringPaymentPending  = errors.New("recurring payment awaiting local binding")
)

type verifiedWebhookContextKey struct{}

// WithVerifiedWebhookContext is set only by the HTTP controller after Stripe's
// signature has been checked. Direct-account events may omit Event.Account in
// Stripe's SDK representation, so that omission is accepted only with this
// marker plus the verified active runtime profile.
func WithVerifiedWebhookContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, verifiedWebhookContextKey{}, true)
}

func verifiedWebhookContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	verified, _ := ctx.Value(verifiedWebhookContextKey{}).(bool)
	return verified
}

type StripeSubscriptionGateway interface {
	CreateCheckoutSession(context.Context, *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error)
	ExpireCheckoutSession(context.Context, string) error
	CreatePortalSession(context.Context, *stripe.BillingPortalSessionParams) (*stripe.BillingPortalSession, error)
}

type stripeGateway struct{}

func (stripeGateway) CreateCheckoutSession(_ context.Context, params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	stripe.Key = setting.StripeApiSecret
	return checkoutsession.New(params)
}

func (stripeGateway) ExpireCheckoutSession(_ context.Context, sessionID string) error {
	stripe.Key = setting.StripeApiSecret
	_, err := checkoutsession.Expire(strings.TrimSpace(sessionID), &stripe.CheckoutSessionExpireParams{})
	return err
}

func (stripeGateway) CreatePortalSession(_ context.Context, params *stripe.BillingPortalSessionParams) (*stripe.BillingPortalSession, error) {
	stripe.Key = setting.StripeApiSecret
	return portalsession.New(params)
}

var gatewayState = struct {
	sync.RWMutex
	gateway StripeSubscriptionGateway
}{gateway: stripeGateway{}}

// SetGatewayForTest replaces the Stripe boundary and returns a restore
// function. Production code always uses the real SDK gateway; tests never
// need network access.
func SetGatewayForTest(gateway StripeSubscriptionGateway) func() {
	gatewayState.Lock()
	previous := gatewayState.gateway
	if gateway == nil {
		gateway = stripeGateway{}
	}
	gatewayState.gateway = gateway
	gatewayState.Unlock()
	return func() {
		gatewayState.Lock()
		gatewayState.gateway = previous
		gatewayState.Unlock()
	}
}

func currentGateway() StripeSubscriptionGateway {
	gatewayState.RLock()
	defer gatewayState.RUnlock()
	return gatewayState.gateway
}

type CheckoutInput struct {
	UserID     int
	PlanID     int
	Email      string
	CustomerID string
	SuccessURL string
	CancelURL  string
}

type CheckoutResult struct {
	PayLink              string `json:"pay_link"`
	ReferenceID          string `json:"reference_id"`
	ReservationID        int64  `json:"reservation_id"`
	ReservationExpiresAt int64  `json:"reservation_expires_at"`
	PlanID               int    `json:"plan_id"`
	PlanCode             string `json:"plan_code"`
	Model                string `json:"model"`
	TargetModel          string `json:"target_model,omitempty"`
	ModelScope           string `json:"model_scope"`
	Tier                 string `json:"tier"`
	CurrentPriceTier     string `json:"current_price_tier"`
	PriceID              string `json:"price_id"`
}

type PortalInput struct {
	UserID    int
	ReturnURL string
}

type PortalResult struct {
	URL string `json:"url"`
}

// NoResaleCopyIdentifier is a stable copy key for the public subscription
// terms. Clients should use it to select translated copy, never to grant or
// infer access. The actual resale check remains server-side in the relay.
const NoResaleCopyIdentifier = "deepseek-v4-flash-0731.no-resale.v1"

// FairUseLimits is the server-authored policy published with the recurring
// offer. Keep these values sourced from the limiter package so the marketing
// contract cannot drift from enforcement.
type FairUseLimits struct {
	PeakConcurrency           int    `json:"peak_concurrency"`
	RollingWindowSeconds      int64  `json:"rolling_window_seconds"`
	ConcurrentSecondsBudget   int64  `json:"concurrent_seconds_budget"`
	SuccessfulRequests        int    `json:"successful_requests"`
	AdmittedRequests          int    `json:"admitted_requests"`
	HeartbeatIntervalSeconds  int64  `json:"heartbeat_interval_seconds"`
	StaleLeaseRecoverySeconds int64  `json:"stale_lease_recovery_seconds"`
	HeartbeatRequired         bool   `json:"heartbeat_required"`
	StaleLeaseRecoveryEnabled bool   `json:"stale_lease_recovery_enabled"`
	NoResaleCopyID            string `json:"no_resale_copy_id"`
	NoResaleCopyIdentifier    string `json:"no_resale_copy_identifier"`

	// Deprecated aliases are retained for the current dashboard contract while
	// the canonical fields above are rolled out.
	WindowMinutes            int64 `json:"window_minutes"`
	SuccessRequestsPerWindow int   `json:"success_requests_per_window"`
	TotalRequestsPerWindow   int   `json:"total_requests_per_window"`
	LeaseSeconds             int64 `json:"lease_seconds"`
	RenewSeconds             int64 `json:"renew_seconds"`
	RecoverySeconds          int64 `json:"recovery_seconds"`
}

func sandboxFairUseLimits() FairUseLimits {
	windowSeconds := int64(deepseekfairuse.WindowDuration / time.Second)
	heartbeatSeconds := int64(deepseekfairuse.HeartbeatInterval / time.Second)
	staleSeconds := int64(deepseekfairuse.StaleLeaseRecovery / time.Second)
	return FairUseLimits{
		PeakConcurrency:           deepseekfairuse.PeakConcurrency,
		RollingWindowSeconds:      windowSeconds,
		ConcurrentSecondsBudget:   deepseekfairuse.ConcurrentSecondsBudget,
		SuccessfulRequests:        deepseekfairuse.SuccessRequestLimit,
		AdmittedRequests:          deepseekfairuse.AdmittedRequestLimit,
		HeartbeatIntervalSeconds:  heartbeatSeconds,
		StaleLeaseRecoverySeconds: staleSeconds,
		HeartbeatRequired:         true,
		StaleLeaseRecoveryEnabled: true,
		NoResaleCopyID:            NoResaleCopyIdentifier,
		NoResaleCopyIdentifier:    NoResaleCopyIdentifier,
		WindowMinutes:             int64(deepseekfairuse.WindowDuration / time.Minute),
		SuccessRequestsPerWindow:  deepseekfairuse.SuccessRequestLimit,
		TotalRequestsPerWindow:    deepseekfairuse.AdmittedRequestLimit,
		LeaseSeconds:              staleSeconds,
		RenewSeconds:              heartbeatSeconds,
		RecoverySeconds:           staleSeconds,
	}
}

type SubscriptionOffer struct {
	Enabled                  bool          `json:"enabled"`
	Active                   bool          `json:"active"`
	Pending                  bool          `json:"pending"`
	Limit                    int           `json:"limit"`
	Remaining                int64         `json:"remaining"`
	SoldOut                  bool          `json:"sold_out"`
	PlanID                   int           `json:"plan_id"`
	Code                     string        `json:"code"`
	Title                    string        `json:"title"`
	Subtitle                 string        `json:"subtitle"`
	Model                    string        `json:"model"`
	TargetModel              string        `json:"target_model,omitempty"`
	ModelScope               string        `json:"model_scope"`
	Currency                 string        `json:"currency"`
	CurrentPriceTier         string        `json:"current_price_tier"`
	CurrentPriceMinor        int64         `json:"current_price_minor"`
	FutureStandardPriceMinor int64         `json:"future_standard_price_minor"`
	FounderPriceID           string        `json:"founder_price_id"`
	StandardPriceID          string        `json:"standard_price_id"`
	FounderAmountMinor       int64         `json:"founder_amount_minor"`
	StandardAmountMinor      int64         `json:"standard_amount_minor"`
	MaxActiveSeats           int           `json:"max_active_seats"`
	FounderPurchaseLimit     int           `json:"founder_purchase_limit"`
	ActiveSeats              int64         `json:"active_seats"`
	PendingSeats             int64         `json:"pending_seats"`
	FounderClaimsUsed        int64         `json:"founder_claims_used"`
	FounderClaimsRemaining   int64         `json:"founder_claims_remaining"`
	FairUse                  FairUseLimits `json:"fair_use"`
	UserStateKnown           bool          `json:"user_state_known"`
	CheckoutAllowed          bool          `json:"checkout_allowed"`
	AlreadyActive            bool          `json:"already_active"`
	AlreadyPending           bool          `json:"already_pending"`
	PendingReservationID     int64         `json:"pending_reservation_id,omitempty"`
	ReservationExpiresAt     int64         `json:"reservation_expires_at,omitempty"`
}

type SubscriptionSummary struct {
	Enabled                  bool                                 `json:"enabled"`
	PlanID                   int                                  `json:"plan_id"`
	PlanCode                 string                               `json:"plan_code"`
	Model                    string                               `json:"model"`
	TargetModel              string                               `json:"target_model,omitempty"`
	ModelScope               string                               `json:"model_scope"`
	Currency                 string                               `json:"currency"`
	ActiveSeats              int64                                `json:"active_seats"`
	MaxSeats                 int                                  `json:"max_seats"`
	StripeStatus             string                               `json:"stripe_status"`
	StripePriceID            string                               `json:"stripe_price_id"`
	CurrentPeriodStart       int64                                `json:"current_period_start"`
	CurrentPeriodEnd         int64                                `json:"current_period_end"`
	CancelAtPeriodEnd        bool                                 `json:"cancel_at_period_end"`
	GracePeriodEnd           int64                                `json:"grace_period_end"`
	PriceTier                string                               `json:"price_tier"`
	CurrentPriceTier         string                               `json:"current_price_tier"`
	CurrentPriceMinor        int64                                `json:"current_price_minor"`
	FutureStandardPriceMinor int64                                `json:"future_standard_price_minor"`
	PendingSeats             int64                                `json:"pending_seats"`
	Remaining                int64                                `json:"remaining"`
	SoldOut                  bool                                 `json:"sold_out"`
	UserStateKnown           bool                                 `json:"user_state_known"`
	CheckoutAllowed          bool                                 `json:"checkout_allowed"`
	AlreadyActive            bool                                 `json:"already_active"`
	AlreadyPending           bool                                 `json:"already_pending"`
	PendingReservationID     int64                                `json:"pending_reservation_id,omitempty"`
	ReservationExpiresAt     int64                                `json:"reservation_expires_at,omitempty"`
	FairUse                  FairUseLimits                        `json:"fair_use"`
	Subscription             *model.StripeSubscription            `json:"subscription"`
	Reservation              *model.StripeSubscriptionReservation `json:"reservation"`
	Entitlement              *model.UserSubscription              `json:"entitlement"`
}

func validateStripeRuntimeWithGate(requireWebhook bool, requireEnabled bool) error {
	config, err := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRecurringPaymentMismatch, err)
	}
	if requireEnabled && !config.Enabled {
		return model.ErrStripeSubscriptionDisabled
	}
	if strings.TrimSpace(setting.StripeAccountID) != config.AccountID {
		return fmt.Errorf("%w: Stripe account mismatch", ErrRecurringPaymentMismatch)
	}
	secret := strings.TrimSpace(setting.StripeApiSecret)
	switch config.Environment {
	case model.SandboxStripeSubscriptionEnvironment:
		if !setting.StripeRequireTestKeys || (!strings.HasPrefix(secret, "sk_test") && !strings.HasPrefix(secret, "rk_test")) {
			return fmt.Errorf("%w: test Stripe credentials required", ErrRecurringPaymentMismatch)
		}
	case model.ProductionStripeSubscriptionEnvironment:
		if setting.StripeRequireTestKeys || (!strings.HasPrefix(secret, "sk_live") && !strings.HasPrefix(secret, "rk_live")) {
			return fmt.Errorf("%w: live Stripe credentials required", ErrRecurringPaymentMismatch)
		}
	}
	if requireWebhook && !strings.HasPrefix(strings.TrimSpace(setting.StripeWebhookSecret), "whsec_") {
		return fmt.Errorf("%w: webhook secret missing", ErrRecurringPaymentMismatch)
	}
	return nil
}

func validateStripeRuntime(requireWebhook bool) error {
	return validateStripeRuntimeWithGate(requireWebhook, true)
}

func validateStripeRuntimeForLifecycle(requireWebhook bool) error {
	return validateStripeRuntimeWithGate(requireWebhook, false)
}

func validateRecurringPlan(plan *model.SubscriptionPlan) error {
	config, err := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
	if err != nil {
		return err
	}
	return model.ValidateStripeSubscriptionPlan(plan, config, true)
}

func loadRecurringPlan(planID int) (*model.SubscriptionPlan, error) {
	plan, err := model.GetStripeSubscriptionPlan(planID)
	if err != nil {
		return nil, err
	}
	if err := validateRecurringPlan(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func loadRecurringLifecyclePlan(planID int) (*model.SubscriptionPlan, error) {
	plan, err := model.GetStripeSubscriptionPlan(planID)
	if err != nil {
		return nil, err
	}
	config, err := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
	if err != nil {
		return nil, err
	}
	if err := model.ValidateStripeSubscriptionPlan(plan, config, false); err != nil {
		return nil, err
	}
	return plan, nil
}

func enabledRecurringPlan() (*model.SubscriptionPlan, error) {
	if model.DB == nil {
		return nil, gorm.ErrRecordNotFound
	}
	config, err := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
	if err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, model.ErrStripeSubscriptionDisabled
	}
	return model.GetFixedStripeSubscriptionPlan(config, true)
}

func randomASCIIAlpha(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if length <= 0 {
		return "", errors.New("invalid random length")
	}
	result := make([]byte, length)
	buffer := make([]byte, length)
	limit := 256 - (256 % len(alphabet))
	filled := 0
	for filled < length {
		if _, err := rand.Read(buffer); err != nil {
			return "", err
		}
		for _, value := range buffer {
			if int(value) >= limit {
				continue
			}
			result[filled] = alphabet[int(value)%len(alphabet)]
			filled++
			if filled == length {
				break
			}
		}
	}
	return string(result), nil
}

func defaultSubscriptionURL(value string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	base := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	if base == "" {
		base = "https://novapura.example"
	}
	return base + "/console/subscription"
}

type recurringCheckoutState struct {
	AlreadyActive  bool
	AlreadyPending bool
	Active         *model.StripeSubscriptionReservation
	Pending        *model.StripeSubscriptionReservation
}

// recurringCheckoutStateForUser is deliberately computed from the primary
// database. It is advisory for rendering, while ReserveStripeSubscriptionSeat
// remains the authoritative lock-protected decision at Checkout time.
func recurringCheckoutStateForUser(userID int, now int64) (*recurringCheckoutState, error) {
	state := &recurringCheckoutState{}
	if userID <= 0 {
		return state, nil
	}
	var reservations []model.StripeSubscriptionReservation
	if err := model.DB.Where("user_id = ? AND status IN ?", userID, []string{
		model.StripeSubscriptionReservationPending,
		model.StripeSubscriptionReservationActive,
		model.StripeSubscriptionReservationReconciliation,
	}).Order("id DESC").Find(&reservations).Error; err != nil {
		return nil, err
	}
	for index := range reservations {
		reservation := &reservations[index]
		switch reservation.Status {
		case model.StripeSubscriptionReservationActive:
			if state.Active == nil {
				state.Active = reservation
			}
			state.AlreadyActive = true
		case model.StripeSubscriptionReservationPending:
			if reservation.ExpiresAt > 0 && reservation.ExpiresAt <= now {
				continue
			}
			if state.Pending == nil {
				state.Pending = reservation
			}
			state.AlreadyPending = true
		case model.StripeSubscriptionReservationReconciliation:
			if state.Pending == nil {
				state.Pending = reservation
			}
			state.AlreadyPending = true
		}
	}

	var subscriptions []model.StripeSubscription
	if err := model.DB.Where("user_id = ?", userID).Order("id DESC").Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	for index := range subscriptions {
		subscription := &subscriptions[index]
		switch strings.ToLower(strings.TrimSpace(subscription.Status)) {
		case model.StripeSubscriptionStatusIncomplete:
			state.AlreadyPending = true
		case model.StripeSubscriptionStatusActive,
			model.StripeSubscriptionStatusPastDue,
			model.StripeSubscriptionStatusUnpaid,
			"trialing":
			state.AlreadyActive = true
		case model.StripeSubscriptionStatusCanceled:
			if subscription.EndedAt == 0 && subscription.GraceUntil > now {
				state.AlreadyActive = true
			}
		case "":
			// An incomplete local row is not a reason to block a user.
		default:
			// Fail closed for an unknown non-terminal Stripe state. This keeps
			// the UI from inviting a duplicate while the webhook state catches up.
			state.AlreadyActive = true
		}
	}
	return state, nil
}

func recurringSeatCounts(planID int, now int64) (int64, int64, error) {
	var activeSeats, pendingSeats int64
	if err := model.DB.Model(&model.StripeSubscriptionReservation{}).
		Where("plan_id = ? AND status = ?", planID, model.StripeSubscriptionReservationActive).
		Count(&activeSeats).Error; err != nil {
		return 0, 0, err
	}
	if err := model.DB.Model(&model.StripeSubscriptionReservation{}).
		Where("plan_id = ? AND ((status = ? AND (expires_at = 0 OR expires_at > ?)) OR status = ?)", planID, model.StripeSubscriptionReservationPending, now, model.StripeSubscriptionReservationReconciliation).
		Count(&pendingSeats).Error; err != nil {
		return 0, 0, err
	}
	return activeSeats, pendingSeats, nil
}

func priceForReservation(plan *model.SubscriptionPlan, reservation *model.StripeSubscriptionReservation) (string, int64, error) {
	if reservation == nil {
		return "", 0, model.ErrStripeSubscriptionReservation
	}
	if reservation.Tier == model.StripeSubscriptionTierFounder {
		return plan.FounderStripePriceId, plan.FounderAmountMinor, nil
	}
	if reservation.Tier == model.StripeSubscriptionTierStandard {
		return plan.StandardStripePriceId, plan.StandardAmountMinor, nil
	}
	return "", 0, model.ErrStripeSubscriptionPlanInvalid
}

func CreateCheckout(ctx context.Context, input CheckoutInput) (*CheckoutResult, error) {
	if err := validateStripeRuntime(true); err != nil {
		return nil, err
	}
	plan, err := loadRecurringPlan(input.PlanID)
	if err != nil {
		return nil, err
	}
	user, err := model.GetUserById(input.UserID, false)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	now := common.GetTimestamp()
	state, err := recurringCheckoutStateForUser(user.Id, now)
	if err != nil {
		return nil, err
	}
	if state.Active != nil || (state.AlreadyActive && state.Pending == nil) {
		return nil, model.ErrSubscriptionAlreadyActive
	}
	var reservation *model.StripeSubscriptionReservation
	if state.Pending != nil {
		if state.Pending.Status == model.StripeSubscriptionReservationPending && strings.TrimSpace(state.Pending.CheckoutSessionId) == "" {
			return nil, model.ErrSubscriptionAlreadyPending
		}
		reservation = state.Pending
	} else {
		referenceSuffix, err := randomASCIIAlpha(20)
		if err != nil {
			return nil, err
		}
		reservation, err = model.ReserveStripeSubscriptionSeat(plan.Id, user.Id, "sub_ref_"+referenceSuffix, now)
		if err != nil {
			return nil, err
		}
	}
	priceID, amountMinor, err := priceForReservation(plan, reservation)
	if err != nil {
		return nil, err
	}
	priceLookupKey, lookupKeyOK := recurringPriceLookupKey(plan, priceID)
	if !lookupKeyOK {
		return nil, fmt.Errorf("%w: recurring price lookup mapping missing", ErrRecurringPaymentMismatch)
	}
	targetModel := model.SubscriptionPlanTargetModel(plan)
	modelScope := model.SubscriptionPlanModelScope(plan)
	idempotencyKey, err := model.EnsureStripeSubscriptionReservationIdempotencyKey(reservation.Id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(reservation.CheckoutSessionId) != "" && strings.TrimSpace(reservation.CheckoutURL) != "" {
		return &CheckoutResult{
			PayLink:              reservation.CheckoutURL,
			ReferenceID:          reservation.ReferenceId,
			ReservationID:        reservation.Id,
			ReservationExpiresAt: reservation.ExpiresAt,
			PlanID:               plan.Id,
			PlanCode:             plan.Code,
			Model:                targetModel,
			TargetModel:          targetModel,
			ModelScope:           modelScope,
			Tier:                 reservation.Tier,
			CurrentPriceTier:     reservation.Tier,
			PriceID:              priceID,
		}, nil
	}
	integrationSuffix, err := randomASCIIAlpha(8)
	if err != nil {
		return nil, err
	}
	metadata := map[string]string{
		"nova_subscription":        "recurring",
		"stripe_environment":       setting.StripeRuntimeEnvironment,
		"stripe_account_id":        plan.StripeAccountId,
		"local_plan_id":            strconv.Itoa(plan.Id),
		"local_plan_code":          plan.Code,
		"plan_id":                  strconv.Itoa(plan.Id),
		"model_scope":              modelScope,
		"product_id":               plan.StripeProductId,
		"price_id":                 priceID,
		"price_lookup_key":         priceLookupKey,
		"tier":                     reservation.Tier,
		"amount_minor":             strconv.FormatInt(amountMinor, 10),
		"currency":                 strings.ToLower(strings.TrimSpace(plan.StripeCurrency)),
		"reservation_id":           strconv.FormatInt(reservation.Id, 10),
		"reservation_reference_id": reservation.ReferenceId,
	}
	if targetModel != "" {
		metadata["model"] = targetModel
		metadata["target_model"] = targetModel
	}
	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(reservation.ReferenceId),
		SuccessURL:        stripe.String(defaultSubscriptionURL(input.SuccessURL)),
		CancelURL:         stripe.String(defaultSubscriptionURL(input.CancelURL)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Price:    stripe.String(priceID),
			Quantity: stripe.Int64(1),
		}},
		Mode:                  stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		ExpiresAt:             stripe.Int64(now + int64(model.StripeSubscriptionReservationTTL/time.Second)),
		AutomaticTax:          &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(false)},
		IntegrationIdentifier: stripe.String(integrationSuffix),
		Metadata:              metadata,
		SubscriptionData:      &stripe.CheckoutSessionSubscriptionDataParams{Metadata: metadata},
	}
	params.SetIdempotencyKey(idempotencyKey)
	customerID := strings.TrimSpace(user.StripeCustomer)
	if customerID != "" {
		params.Customer = stripe.String(customerID)
	} else {
		email := strings.TrimSpace(user.Email)
		if email == "" {
			email = strings.TrimSpace(input.Email)
		}
		if email != "" {
			params.CustomerEmail = stripe.String(email)
		}
	}

	checkout, err := currentGateway().CreateCheckoutSession(ctx, params)
	if err != nil || checkout == nil || strings.TrimSpace(checkout.ID) == "" || strings.TrimSpace(checkout.URL) == "" {
		if checkout != nil && strings.TrimSpace(checkout.ID) != "" {
			_ = currentGateway().ExpireCheckoutSession(ctx, checkout.ID)
		}
		_ = model.ReleasePendingStripeSubscriptionReservation(reservation.Id, common.GetTimestamp())
		if err != nil {
			return nil, err
		}
		return nil, errors.New("Stripe returned an incomplete Checkout session")
	}
	if reservation.CheckoutSessionId != "" && reservation.CheckoutSessionId != checkout.ID {
		_ = currentGateway().ExpireCheckoutSession(ctx, checkout.ID)
		_ = model.ReleasePendingStripeSubscriptionReservation(reservation.Id, common.GetTimestamp())
		return nil, fmt.Errorf("%w: Checkout idempotency session mismatch", ErrRecurringPaymentMismatch)
	}
	if err := model.SetStripeSubscriptionCheckoutSessionDetails(reservation.Id, checkout.ID, checkout.URL, customerID); err != nil {
		_ = currentGateway().ExpireCheckoutSession(ctx, checkout.ID)
		_ = model.ReleasePendingStripeSubscriptionReservation(reservation.Id, common.GetTimestamp())
		return nil, err
	}
	return &CheckoutResult{
		PayLink:              checkout.URL,
		ReferenceID:          reservation.ReferenceId,
		ReservationID:        reservation.Id,
		ReservationExpiresAt: reservation.ExpiresAt,
		PlanID:               plan.Id,
		PlanCode:             plan.Code,
		Model:                targetModel,
		TargetModel:          targetModel,
		ModelScope:           modelScope,
		Tier:                 reservation.Tier,
		CurrentPriceTier:     reservation.Tier,
		PriceID:              priceID,
	}, nil
}

func asRecurringMismatch(err error) error {
	if err == nil || errors.Is(err, ErrRecurringPaymentMismatch) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, model.ErrStripeSubscriptionPlanInvalid) ||
		errors.Is(err, model.ErrStripeSubscriptionDisabled) ||
		errors.Is(err, model.ErrStripeSubscriptionReservation) ||
		errors.Is(err, model.ErrStripeSubscriptionReservationExpired) ||
		errors.Is(err, model.ErrStripeSubscriptionFounderClaimUsed) ||
		errors.Is(err, model.ErrStripeSubscriptionFounderSoldOut) ||
		errors.Is(err, model.ErrSubscriptionAlreadyActive) ||
		errors.Is(err, model.ErrStripeSubscriptionEnded) {
		return fmt.Errorf("%w: %v", ErrRecurringPaymentMismatch, err)
	}
	return err
}

func CreatePortalSession(ctx context.Context, input PortalInput) (*PortalResult, error) {
	if err := validateStripeRuntimeForLifecycle(false); err != nil {
		return nil, err
	}
	user, err := model.GetUserById(input.UserID, false)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	subscription, err := model.GetLatestStripeSubscriptionForUser(user.Id)
	if err != nil {
		return nil, err
	}
	if subscription.Status != model.StripeSubscriptionStatusActive && subscription.Status != model.StripeSubscriptionStatusPastDue {
		return nil, fmt.Errorf("%w: portal unavailable for subscription status", model.ErrStripeSubscriptionReservation)
	}
	plan, err := loadRecurringLifecyclePlan(subscription.PlanId)
	if err != nil {
		return nil, err
	}
	customerID := strings.TrimSpace(subscription.StripeCustomerId)
	if customerID == "" {
		return nil, fmt.Errorf("%w: Stripe customer missing", ErrRecurringPaymentMismatch)
	}
	params := &stripe.BillingPortalSessionParams{
		Customer:      stripe.String(customerID),
		Configuration: stripe.String(plan.StripePortalConfigurationId),
		ReturnURL:     stripe.String(defaultSubscriptionURL(input.ReturnURL)),
	}
	portal, err := currentGateway().CreatePortalSession(ctx, params)
	if err != nil {
		return nil, err
	}
	if portal == nil || strings.TrimSpace(portal.URL) == "" {
		return nil, errors.New("Stripe returned an incomplete portal session")
	}
	return &PortalResult{URL: portal.URL}, nil
}

func GetStripeSubscriptionOffer(planID int, userIDs ...int) (*SubscriptionOffer, error) {
	fairUse := sandboxFairUseLimits()
	config, err := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
	if err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, model.ErrStripeSubscriptionDisabled
	}
	userID := 0
	if len(userIDs) > 0 {
		userID = userIDs[0]
	}
	plan, err := enabledRecurringPlan()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &SubscriptionOffer{Enabled: false, FairUse: fairUse}, nil
		}
		return nil, err
	}
	if planID > 0 && planID != plan.Id {
		return nil, fmt.Errorf("%w: requested recurring plan is not the fixed runtime contract", model.ErrStripeSubscriptionPlanInvalid)
	}
	if err := validateStripeRuntime(false); err != nil {
		return nil, err
	}
	if err := validateRecurringPlan(plan); err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	activeSeats, pendingSeats, err := recurringSeatCounts(plan.Id, now)
	if err != nil {
		return nil, err
	}
	var pendingFounderSeats, claims int64
	if err := model.DB.Model(&model.StripeSubscriptionReservation{}).
		Where("plan_id = ? AND tier = ? AND ((status = ? AND (expires_at = 0 OR expires_at > ?)) OR status = ?)", plan.Id, model.StripeSubscriptionTierFounder, model.StripeSubscriptionReservationPending, now, model.StripeSubscriptionReservationReconciliation).
		Count(&pendingFounderSeats).Error; err != nil {
		return nil, err
	}
	if err := model.DB.Model(&model.StripeSubscriptionFounderClaim{}).Where("plan_id = ?", plan.Id).Count(&claims).Error; err != nil {
		return nil, err
	}
	remaining := int64(plan.FounderPurchaseLimit) - claims - pendingFounderSeats
	if remaining < 0 {
		remaining = 0
	}
	seatLimit := int64(plan.MaxActiveSubscriptions)
	seatsRemaining := seatLimit - activeSeats - pendingSeats
	if seatsRemaining < 0 {
		seatsRemaining = 0
	}
	state, err := recurringCheckoutStateForUser(userID, now)
	if err != nil {
		return nil, err
	}
	founderClaimedByUser := false
	if userID > 0 {
		founderClaimedByUser, err = model.HasStripeSubscriptionFounderClaim(plan.Id, userID)
		if err != nil {
			return nil, err
		}
	}
	checkoutAllowed := seatsRemaining > 0 && !state.AlreadyActive && !state.AlreadyPending
	currentPriceTier := model.StripeSubscriptionTierStandard
	currentPriceMinor := plan.StandardAmountMinor
	if remaining > 0 && !founderClaimedByUser {
		currentPriceTier = model.StripeSubscriptionTierFounder
		currentPriceMinor = plan.FounderAmountMinor
	}
	offer := &SubscriptionOffer{
		Enabled:                  true,
		Active:                   true,
		Pending:                  pendingSeats > 0 && seatsRemaining == 0,
		Limit:                    plan.MaxActiveSubscriptions,
		Remaining:                seatsRemaining,
		SoldOut:                  seatsRemaining == 0,
		PlanID:                   plan.Id,
		Code:                     plan.Code,
		Title:                    plan.Title,
		Subtitle:                 plan.Subtitle,
		Model:                    model.SubscriptionPlanTargetModel(plan),
		TargetModel:              model.SubscriptionPlanTargetModel(plan),
		ModelScope:               model.SubscriptionPlanModelScope(plan),
		Currency:                 strings.ToUpper(plan.StripeCurrency),
		CurrentPriceTier:         currentPriceTier,
		CurrentPriceMinor:        currentPriceMinor,
		FutureStandardPriceMinor: plan.StandardAmountMinor,
		FounderPriceID:           plan.FounderStripePriceId,
		StandardPriceID:          plan.StandardStripePriceId,
		FounderAmountMinor:       plan.FounderAmountMinor,
		StandardAmountMinor:      plan.StandardAmountMinor,
		MaxActiveSeats:           plan.MaxActiveSubscriptions,
		FounderPurchaseLimit:     plan.FounderPurchaseLimit,
		ActiveSeats:              activeSeats,
		PendingSeats:             pendingSeats,
		FounderClaimsUsed:        claims,
		FounderClaimsRemaining:   remaining,
		FairUse:                  fairUse,
		UserStateKnown:           userID > 0,
		CheckoutAllowed:          checkoutAllowed,
		AlreadyActive:            state.AlreadyActive,
		AlreadyPending:           state.AlreadyPending,
	}
	if state.Pending != nil {
		offer.PendingReservationID = state.Pending.Id
		offer.ReservationExpiresAt = state.Pending.ExpiresAt
	}
	return offer, nil
}

func GetStripeSubscriptionSummary(userID int) (*SubscriptionSummary, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	if model.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	summary := &SubscriptionSummary{
		FairUse:        sandboxFairUseLimits(),
		UserStateKnown: true,
	}
	var currentPlan *model.SubscriptionPlan
	var runtimeConfig model.StripeSubscriptionConfig
	var fixedPlan *model.SubscriptionPlan
	var configErr error
	var planErr error
	runtimeConfig, configErr = model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
	if configErr == nil {
		fixedPlan, planErr = model.GetFixedStripeSubscriptionPlan(runtimeConfig, true)
		if errors.Is(planErr, model.ErrStripeSubscriptionDisabled) {
			// Summary/readiness still describes the user's associated fixed plan
			// after new sales are disabled, but never promotes it to an offer.
			fixedPlan, planErr = model.GetFixedStripeSubscriptionPlan(runtimeConfig, false)
		}
	}
	if configErr == nil && planErr == nil && fixedPlan != nil {
		currentPlan = fixedPlan
		summary.PlanID = fixedPlan.Id
		summary.PlanCode = fixedPlan.Code
		summary.Model = model.SubscriptionPlanTargetModel(fixedPlan)
		summary.TargetModel = model.SubscriptionPlanTargetModel(fixedPlan)
		summary.ModelScope = model.SubscriptionPlanModelScope(fixedPlan)
		summary.Currency = strings.ToUpper(fixedPlan.StripeCurrency)
		summary.MaxSeats = fixedPlan.MaxActiveSubscriptions
		summary.FutureStandardPriceMinor = fixedPlan.StandardAmountMinor
		summary.Enabled = runtimeConfig.Enabled && fixedPlan.Enabled && fixedPlan.StripeSubscriptionEnabled &&
			validateStripeRuntime(false) == nil && model.ValidateStripeSubscriptionPlan(fixedPlan, runtimeConfig, true) == nil
	}
	subscription, err := model.GetLatestStripeSubscriptionForUser(userID)
	if err == nil {
		summary.Subscription = subscription
		summary.StripeStatus = subscription.Status
		summary.StripePriceID = subscription.StripePriceId
		summary.CurrentPeriodStart = subscription.CurrentPeriodStart
		summary.CurrentPeriodEnd = subscription.CurrentPeriodEnd
		summary.CancelAtPeriodEnd = subscription.CancelAtPeriodEnd
		summary.GracePeriodEnd = subscription.GraceUntil
		summary.PriceTier = subscription.Tier
		summary.CurrentPriceTier = subscription.Tier
		if subscription.UserSubscriptionId > 0 {
			var entitlement model.UserSubscription
			if err := model.DB.First(&entitlement, subscription.UserSubscriptionId).Error; err == nil {
				summary.Entitlement = &entitlement
			}
		}
		if fixedPlan != nil && subscription.PlanId == fixedPlan.Id {
			currentPlan = fixedPlan
			if currentAmount, amountOK := recurringPriceAmount(fixedPlan, subscription.StripePriceId); amountOK {
				summary.CurrentPriceMinor = currentAmount
			}
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := common.GetTimestamp()
	state, err := recurringCheckoutStateForUser(userID, now)
	if err != nil {
		return nil, err
	}
	summary.AlreadyActive = state.AlreadyActive
	summary.AlreadyPending = state.AlreadyPending
	if state.Pending != nil {
		summary.Reservation = state.Pending
		summary.PendingReservationID = state.Pending.Id
		summary.ReservationExpiresAt = state.Pending.ExpiresAt
	} else if state.Active != nil {
		summary.Reservation = state.Active
	}
	if currentPlan != nil && summary.Enabled {
		activeSeats, pendingSeats, err := recurringSeatCounts(currentPlan.Id, now)
		if err != nil {
			return nil, err
		}
		summary.ActiveSeats = activeSeats
		summary.PendingSeats = pendingSeats
		summary.Remaining = int64(currentPlan.MaxActiveSubscriptions) - activeSeats - pendingSeats
		if summary.Remaining < 0 {
			summary.Remaining = 0
		}
		summary.SoldOut = summary.Remaining == 0
		summary.CheckoutAllowed = summary.Remaining > 0 && !state.AlreadyActive && !state.AlreadyPending
	}
	return summary, nil
}

func IsWebhookEnabled() bool {
	enabled, err := IsWebhookEnabledWithError()
	return err == nil && enabled
}

// IsWebhookEnabledWithError lets the shared webhook controller distinguish a
// deliberately disabled recurring feature from a database outage that must
// remain retryable.
func IsWebhookEnabledWithError() (bool, error) {
	if err := validateStripeRuntimeForLifecycle(true); err != nil {
		return false, nil
	}
	return model.HasStripeSubscriptionLifecyclePlan()
}

func IsRecurringEvent(event stripe.Event) bool {
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted, stripe.EventTypeCheckoutSessionExpired:
		return recurringObjectMarker(&event)
	case stripe.EventTypeInvoicePaid, stripe.EventTypeInvoicePaymentFailed,
		stripe.EventTypeCustomerSubscriptionUpdated, stripe.EventTypeCustomerSubscriptionDeleted:
		if recurringObjectMarker(&event) {
			return true
		}
		subscriptionID := objectString(event.Data, "subscription")
		if subscriptionID == "" {
			subscriptionID = objectString(event.Data, "id")
		}
		// Invoice/subscription event types are recurring by Stripe contract;
		// do not let a transient local lookup failure fall through to the
		// one-time top-up processor and get acknowledged as a no-op.
		return subscriptionID != "" || hasLocalRecurringSubscription(subscriptionID)
	default:
		return false
	}
}

func recurringObjectMarker(event *stripe.Event) bool {
	if event == nil || event.Data == nil {
		return false
	}
	if strings.EqualFold(recurringMetadataString(*event, "nova_subscription"), "recurring") ||
		strings.EqualFold(recurringMetadataString(*event, "subscription_type"), "recurring") {
		return true
	}
	metadata := recurringMetadataString(*event, "plan_id")
	return metadata != "" && (recurringMetadataString(*event, "model") != "" || recurringMetadataString(*event, "product_id") != "")
}

func hasLocalRecurringSubscription(subscriptionID string) bool {
	if strings.TrimSpace(subscriptionID) == "" || model.DB == nil {
		return false
	}
	_, err := model.GetStripeSubscriptionByStripeID(subscriptionID)
	return err == nil
}

func recurringWebhookRecord(event stripe.Event) *model.StripeWebhookEvent {
	return &model.StripeWebhookEvent{
		EventID:   event.ID,
		EventType: string(event.Type),
		Livemode:  event.Livemode,
		AccountID: event.Account,
		OrderID:   objectString(event.Data, "id"),
		CreatedAt: common.GetTimestamp(),
	}
}

func markRecurringEventManualReview(event stripe.Event, reason error) error {
	if reason == nil || strings.TrimSpace(event.ID) == "" {
		return reason
	}
	record := recurringWebhookRecord(event)
	claimed, terminal, err := model.ClaimStripeWebhookEvent(record, common.GetTimestamp(), 5*time.Minute)
	if err != nil {
		return err
	}
	if !claimed || terminal {
		// The event evidence is already terminal or another worker owns its
		// lease. The current delivery is still a permanent account mismatch;
		// never acknowledge it as a successful recurring event.
		return reason
	}
	if err := model.FinalizeStripeWebhookEvent(event.ID, model.StripeWebhookEventManualReview, reason.Error(), common.GetTimestamp()); err != nil {
		return err
	}
	return reason
}

func HandleRecurringEvent(ctx context.Context, event stripe.Event) error {
	if err := validateStripeRuntimeForLifecycle(true); err != nil {
		return err
	}
	if !IsRecurringEvent(event) {
		return ErrRecurringEventNotHandled
	}
	if event.ID == "" || (event.Account == "" && !verifiedWebhookContext(ctx)) {
		return fmt.Errorf("%w: event environment mismatch", ErrRecurringPaymentMismatch)
	}
	config, configErr := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
	if configErr != nil {
		return fmt.Errorf("%w: %v", ErrRecurringPaymentMismatch, configErr)
	}
	expectedLivemode := config.Environment == model.ProductionStripeSubscriptionEnvironment
	if event.Livemode != expectedLivemode || (event.Account != "" && event.Account != config.AccountID) {
		return markRecurringEventManualReview(event, fmt.Errorf("%w: event environment mismatch", ErrRecurringPaymentMismatch))
	}
	webhookRecord := recurringWebhookRecord(event)
	claimed, terminal, err := model.ClaimStripeWebhookEvent(webhookRecord, common.GetTimestamp(), 5*time.Minute)
	if err != nil {
		return err
	}
	if !claimed || terminal {
		if strings.TrimSpace(webhookRecord.Status) == model.StripeWebhookEventManualReview {
			return fmt.Errorf("%w: event is already marked for manual review", ErrRecurringPaymentMismatch)
		}
		if strings.TrimSpace(webhookRecord.Status) == model.StripeWebhookEventProcessing {
			// A second delivery must keep receiving a retryable response while
			// the first worker owns the lease. Returning 2xx here could make
			// Stripe stop retrying an event that has not reached a terminal row.
			return ErrRecurringPaymentPending
		}
		return nil
	}
	if err := handleRecurringEvent(event); err != nil {
		if errors.Is(err, ErrRecurringPaymentMismatch) {
			if finalizeErr := model.FinalizeStripeWebhookEvent(event.ID, model.StripeWebhookEventManualReview, err.Error(), common.GetTimestamp()); finalizeErr != nil {
				return fmt.Errorf("recurring webhook manual-review finalization failed: %w", finalizeErr)
			}
		}
		return err
	}
	if err := model.FinalizeStripeWebhookEvent(event.ID, model.StripeWebhookEventProcessed, "", common.GetTimestamp()); err != nil {
		return err
	}
	return nil
}

func handleRecurringEvent(event stripe.Event) error {
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		return handleCheckoutCompleted(event)
	case stripe.EventTypeCheckoutSessionExpired:
		return handleCheckoutExpired(event)
	case stripe.EventTypeInvoicePaid:
		return handleInvoicePaid(event)
	case stripe.EventTypeInvoicePaymentFailed:
		return handleInvoicePaymentFailed(event)
	case stripe.EventTypeCustomerSubscriptionUpdated:
		return handleSubscriptionUpdated(event)
	case stripe.EventTypeCustomerSubscriptionDeleted:
		return handleSubscriptionDeleted(event)
	default:
		return ErrRecurringEventNotHandled
	}
}

func handleCheckoutCompleted(event stripe.Event) error {
	if status := objectString(event.Data, "status"); status != "" && status != "complete" {
		return nil
	}
	paymentStatus := objectString(event.Data, "payment_status")
	referenceID := objectString(event.Data, "client_reference_id")
	sessionID := objectString(event.Data, "id")
	stripeSubscriptionID := objectString(event.Data, "subscription")
	customerID := objectString(event.Data, "customer")
	priceID := objectString(event.Data, "metadata", "price_id")
	amount, amountOK := objectInt64(event.Data, "amount_total")
	currency := strings.ToLower(strings.TrimSpace(objectString(event.Data, "currency")))
	if referenceID == "" || sessionID == "" || stripeSubscriptionID == "" || priceID == "" || !amountOK || currency == "" {
		return fmt.Errorf("%w: incomplete Checkout payment", ErrRecurringPaymentMismatch)
	}
	reservation, err := model.GetStripeSubscriptionReservationByReference(referenceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: reservation not found", ErrRecurringPaymentMismatch)
		}
		return err
	}
	plan, err := loadRecurringLifecyclePlan(reservation.PlanId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, model.ErrStripeSubscriptionPlanInvalid) || errors.Is(err, model.ErrStripeSubscriptionDisabled) {
			return fmt.Errorf("%w: %v", ErrRecurringPaymentMismatch, err)
		}
		return err
	}
	if reservation.CheckoutSessionId != "" && reservation.CheckoutSessionId != sessionID {
		return fmt.Errorf("%w: Checkout session mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataPlanID := objectString(event.Data, "metadata", "plan_id"); metadataPlanID == "" || metadataPlanID != strconv.Itoa(plan.Id) {
		return fmt.Errorf("%w: plan metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataEnvironment := strings.ToLower(objectString(event.Data, "metadata", "stripe_environment")); metadataEnvironment != "" && metadataEnvironment != setting.StripeRuntimeEnvironment {
		return fmt.Errorf("%w: environment metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataAccount := objectString(event.Data, "metadata", "stripe_account_id"); metadataAccount != "" && metadataAccount != plan.StripeAccountId {
		return fmt.Errorf("%w: account metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if err := validateRecurringModelMetadata(event, plan); err != nil {
		return err
	}
	if metadataProduct := objectString(event.Data, "metadata", "product_id"); metadataProduct != plan.StripeProductId {
		return fmt.Errorf("%w: product metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataAmount := objectString(event.Data, "metadata", "amount_minor"); metadataAmount != "" && metadataAmount != strconv.FormatInt(amount, 10) {
		return fmt.Errorf("%w: amount metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataCurrency := strings.ToLower(objectString(event.Data, "metadata", "currency")); metadataCurrency != "" && metadataCurrency != currency {
		return fmt.Errorf("%w: currency metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if reservation.StripeCustomerId != "" && reservation.StripeCustomerId != customerID {
		return fmt.Errorf("%w: customer mismatch", ErrRecurringPaymentMismatch)
	}
	expectedPrice, expectedAmount, err := priceForReservation(plan, reservation)
	if err != nil || priceID != expectedPrice || amount != expectedAmount || currency != strings.ToLower(strings.TrimSpace(plan.StripeCurrency)) {
		return fmt.Errorf("%w: amount, currency, or price mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataLookupKey := objectString(event.Data, "metadata", "price_lookup_key"); metadataLookupKey != "" {
		expectedLookupKey, lookupKeyOK := recurringPriceLookupKey(plan, expectedPrice)
		if !lookupKeyOK || metadataLookupKey != expectedLookupKey {
			return fmt.Errorf("%w: price lookup metadata mismatch", ErrRecurringPaymentMismatch)
		}
	}
	if paymentStatus != "" && paymentStatus != "paid" && paymentStatus != "no_payment_required" {
		if _, err := model.BindStripeSubscriptionCheckout(model.StripeSubscriptionBindingInput{
			ReservationID:        reservation.Id,
			CheckoutSessionID:    sessionID,
			CustomerID:           customerID,
			StripeSubscriptionID: stripeSubscriptionID,
			StripePriceID:        priceID,
		}); err != nil {
			return asRecurringMismatch(err)
		}
		return nil
	}
	periodStart, _ := objectInt64(event.Data, "current_period_start")
	periodEnd, _ := objectInt64(event.Data, "current_period_end")
	_, err = model.ActivateStripeSubscriptionWithEntitlement(model.StripeSubscriptionActivationInput{
		ReservationID:        reservation.Id,
		CheckoutSessionID:    sessionID,
		CustomerID:           customerID,
		StripeSubscriptionID: stripeSubscriptionID,
		StripePriceID:        priceID,
		PeriodStart:          periodStart,
		PeriodEnd:            periodEnd,
	})
	if err != nil {
		return asRecurringMismatch(err)
	}
	return err
}

func handleCheckoutExpired(event stripe.Event) error {
	referenceID := objectString(event.Data, "client_reference_id")
	sessionID := objectString(event.Data, "id")
	if referenceID == "" || sessionID == "" {
		return fmt.Errorf("%w: missing checkout reference", ErrRecurringPaymentMismatch)
	}
	reservation, err := model.GetStripeSubscriptionReservationByReference(referenceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: reservation not found", ErrRecurringPaymentMismatch)
		}
		return err
	}
	if reservation.CheckoutSessionId != "" && reservation.CheckoutSessionId != sessionID {
		return fmt.Errorf("%w: Checkout session mismatch", ErrRecurringPaymentMismatch)
	}
	plan, err := model.GetStripeSubscriptionPlan(reservation.PlanId)
	if err != nil {
		return asRecurringMismatch(err)
	}
	config, configErr := model.StripeSubscriptionConfigForEnvironment(setting.StripeRuntimeEnvironment)
	if configErr != nil {
		return asRecurringMismatch(configErr)
	}
	if err := model.ValidateStripeSubscriptionPlan(plan, config, false); err != nil {
		return asRecurringMismatch(err)
	}
	if metadataPlanID := recurringMetadataString(event, "plan_id"); metadataPlanID != "" && metadataPlanID != strconv.Itoa(plan.Id) {
		return fmt.Errorf("%w: plan metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if err := validateRecurringMetadata(event, plan); err != nil {
		return err
	}
	if reservation.Status == model.StripeSubscriptionReservationActive {
		// A late checkout.session.expired event must not release a seat already
		// owned by a paid recurring subscription.
		return nil
	}
	return asRecurringMismatch(model.ReleasePendingStripeSubscriptionReservation(reservation.Id, common.GetTimestamp()))
}

func findRecurringSubscriptionFromEvent(event stripe.Event) (*model.StripeSubscription, *model.SubscriptionPlan, error) {
	stripeSubscriptionID := objectString(event.Data, "subscription")
	if stripeSubscriptionID == "" && (event.Type == stripe.EventTypeCustomerSubscriptionUpdated || event.Type == stripe.EventTypeCustomerSubscriptionDeleted) {
		stripeSubscriptionID = objectString(event.Data, "id")
	}
	if stripeSubscriptionID == "" {
		stripeSubscriptionID = recurringMetadataString(event, "subscription_id")
	}
	if stripeSubscriptionID == "" {
		return nil, nil, fmt.Errorf("%w: recurring subscription id missing", ErrRecurringPaymentMismatch)
	}
	subscription, err := model.GetStripeSubscriptionByStripeID(stripeSubscriptionID)
	var plan *model.SubscriptionPlan
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, err
		}
		reservationID, _ := strconv.ParseInt(recurringMetadataString(event, "reservation_id"), 10, 64)
		var reservation *model.StripeSubscriptionReservation
		if reservationID > 0 {
			var lookup model.StripeSubscriptionReservation
			if lookupErr := model.DB.Where("id = ?", reservationID).First(&lookup).Error; lookupErr == nil {
				reservation = &lookup
			} else if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				return nil, nil, lookupErr
			}
		}
		if reservation == nil {
			if referenceID := recurringMetadataString(event, "reservation_reference_id"); referenceID != "" {
				reservation, err = model.GetStripeSubscriptionReservationByReference(referenceID)
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil, err
				}
			}
		}
		if reservation == nil {
			return nil, nil, ErrRecurringPaymentPending
		}
		if referenceID := recurringMetadataString(event, "reservation_reference_id"); referenceID != "" && referenceID != reservation.ReferenceId {
			return nil, nil, fmt.Errorf("%w: reservation reference metadata mismatch", ErrRecurringPaymentMismatch)
		}
		planID, parseErr := strconv.Atoi(recurringMetadataString(event, "plan_id"))
		if parseErr != nil || planID <= 0 || planID != reservation.PlanId {
			return nil, nil, fmt.Errorf("%w: invoice plan metadata cannot be reconciled", ErrRecurringPaymentMismatch)
		}
		priceID := objectString(event.Data, "lines", "data", "0", "price", "id")
		if priceID == "" {
			priceID = recurringMetadataString(event, "price_id")
		}
		if priceID == "" {
			return nil, nil, ErrRecurringPaymentPending
		}
		plan, err = loadRecurringLifecyclePlan(reservation.PlanId)
		if err != nil {
			return nil, nil, asRecurringMismatch(err)
		}
		if err := validateRecurringLifecycleAnchor(event, plan, priceID); err != nil {
			return nil, nil, err
		}
		bound, bindErr := model.BindStripeSubscriptionCheckout(model.StripeSubscriptionBindingInput{
			ReservationID:        reservation.Id,
			CheckoutSessionID:    reservation.CheckoutSessionId,
			CustomerID:           objectString(event.Data, "customer"),
			StripeSubscriptionID: stripeSubscriptionID,
			StripePriceID:        priceID,
		})
		if bindErr != nil {
			return nil, nil, asRecurringMismatch(bindErr)
		}
		subscription = bound
	}
	if plan == nil {
		plan, err = loadRecurringLifecyclePlan(subscription.PlanId)
		if err != nil {
			return nil, nil, asRecurringMismatch(err)
		}
	}
	if customerID := objectString(event.Data, "customer"); customerID != "" && subscription.StripeCustomerId != "" && customerID != subscription.StripeCustomerId {
		return nil, nil, fmt.Errorf("%w: customer mismatch", ErrRecurringPaymentMismatch)
	}
	return subscription, plan, nil
}

// validateRecurringLifecycleAnchor performs the full event-contract check
// before invoice-first reconciliation writes a local StripeSubscription row.
// Reservation metadata authorizes a possible bind, but it cannot bypass the
// same catalog, price, amount, currency, and product checks used after bind.
func validateRecurringLifecycleAnchor(event stripe.Event, plan *model.SubscriptionPlan, priceID string) error {
	if err := validateRecurringMetadata(event, plan); err != nil {
		return err
	}
	if _, ok := recurringPriceAmount(plan, priceID); !ok {
		return fmt.Errorf("%w: invoice price mismatch", ErrRecurringPaymentMismatch)
	}
	switch event.Type {
	case stripe.EventTypeInvoicePaid, stripe.EventTypeInvoicePaymentFailed:
		_, _, _, validatedPriceID, _, _, err := invoiceDetails(event, plan)
		if err != nil {
			return err
		}
		if validatedPriceID != priceID {
			return fmt.Errorf("%w: invoice price anchor mismatch", ErrRecurringPaymentMismatch)
		}
	}
	return nil
}

func recurringMetadataString(event stripe.Event, key string) string {
	if value := objectString(event.Data, "metadata", key); value != "" {
		return value
	}
	return objectString(event.Data, "subscription_details", "metadata", key)
}

func validateRecurringModelMetadata(event stripe.Event, plan *model.SubscriptionPlan) error {
	expectedTarget := model.SubscriptionPlanTargetModel(plan)
	expectedScope := model.SubscriptionPlanModelScope(plan)
	metadataModel := strings.TrimSpace(recurringMetadataString(event, "model"))
	metadataTarget := strings.TrimSpace(recurringMetadataString(event, "target_model"))
	metadataScope := strings.TrimSpace(recurringMetadataString(event, "model_scope"))

	if metadataScope != "" && !strings.EqualFold(metadataScope, expectedScope) {
		return fmt.Errorf("%w: model scope mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataModel != "" && metadataModel != expectedTarget &&
		!(expectedScope == model.StripeSubscriptionModelScopeAll && metadataModel == model.LegacySandboxStripeSubscriptionModel) {
		return fmt.Errorf("%w: model mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataTarget != "" && metadataTarget != expectedTarget &&
		!(expectedScope == model.StripeSubscriptionModelScopeAll && metadataTarget == model.LegacySandboxStripeSubscriptionModel) {
		return fmt.Errorf("%w: target model mismatch", ErrRecurringPaymentMismatch)
	}
	if expectedTarget != "" && metadataModel == "" && metadataTarget == "" {
		return fmt.Errorf("%w: model metadata missing", ErrRecurringPaymentMismatch)
	}
	return nil
}

func validateRecurringMetadata(event stripe.Event, plan *model.SubscriptionPlan) error {
	if value := strings.ToLower(recurringMetadataString(event, "stripe_environment")); value != "" && value != setting.StripeRuntimeEnvironment {
		return fmt.Errorf("%w: environment mismatch", ErrRecurringPaymentMismatch)
	}
	if value := recurringMetadataString(event, "stripe_account_id"); value != "" && value != plan.StripeAccountId {
		return fmt.Errorf("%w: account mismatch", ErrRecurringPaymentMismatch)
	}
	if err := validateRecurringModelMetadata(event, plan); err != nil {
		return err
	}
	if value := recurringMetadataString(event, "product_id"); value != "" && value != plan.StripeProductId {
		return fmt.Errorf("%w: product mismatch", ErrRecurringPaymentMismatch)
	}
	return nil
}

func validateRecurringStripeObjectFields(event stripe.Event, plan *model.SubscriptionPlan, expectedPriceID string) error {
	if err := validateRecurringMetadata(event, plan); err != nil {
		return err
	}
	priceID := objectString(event.Data, "items", "data", "0", "price", "id")
	if priceID == "" {
		priceID = objectString(event.Data, "metadata", "price_id")
	}
	if metadataPriceID := recurringMetadataString(event, "price_id"); metadataPriceID != "" && expectedPriceID != "" && metadataPriceID != expectedPriceID {
		return fmt.Errorf("%w: subscription price metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataLookupKey := recurringMetadataString(event, "price_lookup_key"); metadataLookupKey != "" {
		expectedLookupKey, lookupKeyOK := recurringPriceLookupKey(plan, expectedPriceID)
		if !lookupKeyOK || metadataLookupKey != expectedLookupKey {
			return fmt.Errorf("%w: subscription price lookup metadata mismatch", ErrRecurringPaymentMismatch)
		}
	}
	if priceID != "" && expectedPriceID != "" && priceID != expectedPriceID {
		return fmt.Errorf("%w: subscription price mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataAmount := recurringMetadataString(event, "amount_minor"); metadataAmount != "" {
		expectedAmount, amountOK := recurringPriceAmount(plan, expectedPriceID)
		parsedAmount, parseErr := strconv.ParseInt(metadataAmount, 10, 64)
		if !amountOK || parseErr != nil || parsedAmount != expectedAmount {
			return fmt.Errorf("%w: subscription amount metadata mismatch", ErrRecurringPaymentMismatch)
		}
	}
	if metadataCurrency := strings.ToLower(recurringMetadataString(event, "currency")); metadataCurrency != "" && metadataCurrency != strings.ToLower(strings.TrimSpace(plan.StripeCurrency)) {
		return fmt.Errorf("%w: subscription currency metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if unitAmount, ok := objectInt64(event.Data, "items", "data", "0", "price", "unit_amount"); ok {
		expectedAmount, amountOK := recurringPriceAmount(plan, expectedPriceID)
		if !amountOK || unitAmount != expectedAmount {
			return fmt.Errorf("%w: subscription amount mismatch", ErrRecurringPaymentMismatch)
		}
	}
	if currency := strings.ToLower(objectString(event.Data, "items", "data", "0", "price", "currency")); currency != "" && currency != strings.ToLower(strings.TrimSpace(plan.StripeCurrency)) {
		return fmt.Errorf("%w: subscription currency mismatch", ErrRecurringPaymentMismatch)
	}
	productID := objectString(event.Data, "items", "data", "0", "price", "product")
	if productID == "" {
		productID = objectString(event.Data, "product")
	}
	if productID != "" && productID != plan.StripeProductId {
		return fmt.Errorf("%w: subscription product mismatch", ErrRecurringPaymentMismatch)
	}
	return nil
}

func invoiceDetails(event stripe.Event, plan *model.SubscriptionPlan) (string, int64, string, string, int64, int64, error) {
	if err := validateRecurringMetadata(event, plan); err != nil {
		return "", 0, "", "", 0, 0, err
	}
	invoiceID := objectString(event.Data, "id")
	priceID := objectString(event.Data, "lines", "data", "0", "price", "id")
	if priceID == "" {
		priceID = recurringMetadataString(event, "price_id")
	} else if metadataPriceID := recurringMetadataString(event, "price_id"); metadataPriceID != "" && metadataPriceID != priceID {
		return "", 0, "", "", 0, 0, fmt.Errorf("%w: invoice price metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataLookupKey := recurringMetadataString(event, "price_lookup_key"); metadataLookupKey != "" {
		expectedLookupKey, lookupKeyOK := recurringPriceLookupKey(plan, priceID)
		if !lookupKeyOK || metadataLookupKey != expectedLookupKey {
			return "", 0, "", "", 0, 0, fmt.Errorf("%w: invoice price lookup metadata mismatch", ErrRecurringPaymentMismatch)
		}
	}
	productID := objectString(event.Data, "lines", "data", "0", "price", "product")
	if productID != "" && productID != plan.StripeProductId {
		return "", 0, "", "", 0, 0, fmt.Errorf("%w: invoice product mismatch", ErrRecurringPaymentMismatch)
	}
	if unitAmount, ok := objectInt64(event.Data, "lines", "data", "0", "price", "unit_amount"); ok {
		expectedAmount, amountOK := recurringPriceAmount(plan, priceID)
		if !amountOK || unitAmount != expectedAmount {
			return "", 0, "", "", 0, 0, fmt.Errorf("%w: invoice price amount mismatch", ErrRecurringPaymentMismatch)
		}
	}
	if priceCurrency := strings.ToLower(objectString(event.Data, "lines", "data", "0", "price", "currency")); priceCurrency != "" && priceCurrency != strings.ToLower(strings.TrimSpace(plan.StripeCurrency)) {
		return "", 0, "", "", 0, 0, fmt.Errorf("%w: invoice price currency mismatch", ErrRecurringPaymentMismatch)
	}
	amount, amountOK := objectInt64(event.Data, "amount_paid")
	// A failed invoice normally reports amount_paid=0 while amount_due still
	// carries the exact recurring charge that must be validated. Do not let
	// that Stripe representation bypass the mismatch check or block grace.
	if event.Type == stripe.EventTypeInvoicePaymentFailed && (!amountOK || amount == 0) {
		amount, amountOK = objectInt64(event.Data, "amount_due")
	} else if !amountOK {
		amount, amountOK = objectInt64(event.Data, "amount_due")
	}
	currency := strings.ToLower(strings.TrimSpace(objectString(event.Data, "currency")))
	periodStart, _ := objectInt64(event.Data, "period_start")
	periodEnd, _ := objectInt64(event.Data, "period_end")
	if invoiceID == "" || priceID == "" || !amountOK || currency == "" {
		return "", 0, "", "", 0, 0, fmt.Errorf("%w: incomplete invoice", ErrRecurringPaymentMismatch)
	}
	if metadataAmount := recurringMetadataString(event, "amount_minor"); metadataAmount != "" && metadataAmount != strconv.FormatInt(amount, 10) {
		return "", 0, "", "", 0, 0, fmt.Errorf("%w: invoice amount metadata mismatch", ErrRecurringPaymentMismatch)
	}
	if metadataCurrency := strings.ToLower(recurringMetadataString(event, "currency")); metadataCurrency != "" && metadataCurrency != currency {
		return "", 0, "", "", 0, 0, fmt.Errorf("%w: invoice currency metadata mismatch", ErrRecurringPaymentMismatch)
	}
	return invoiceID, amount, currency, priceID, periodStart, periodEnd, nil
}

func validateInvoiceAmount(plan *model.SubscriptionPlan, priceID string, amount int64, currency string) error {
	expectedAmount, ok := recurringPriceAmount(plan, priceID)
	if !ok {
		return fmt.Errorf("%w: price mismatch", ErrRecurringPaymentMismatch)
	}
	if amount != expectedAmount || strings.ToLower(strings.TrimSpace(currency)) != strings.ToLower(strings.TrimSpace(plan.StripeCurrency)) {
		return fmt.Errorf("%w: invoice amount or currency mismatch", ErrRecurringPaymentMismatch)
	}
	return nil
}

func recurringPriceAmount(plan *model.SubscriptionPlan, priceID string) (int64, bool) {
	if plan == nil {
		return 0, false
	}
	switch priceID {
	case plan.FounderStripePriceId:
		return plan.FounderAmountMinor, true
	case plan.StandardStripePriceId:
		return plan.StandardAmountMinor, true
	default:
		return 0, false
	}
}

func recurringPriceLookupKey(plan *model.SubscriptionPlan, priceID string) (string, bool) {
	if plan == nil {
		return "", false
	}
	switch priceID {
	case plan.FounderStripePriceId:
		return model.SandboxStripeSubscriptionFounderLookupKey, true
	case plan.StandardStripePriceId:
		return model.SandboxStripeSubscriptionStandardLookupKey, true
	default:
		return "", false
	}
}

func handleInvoicePaid(event stripe.Event) error {
	subscription, plan, err := findRecurringSubscriptionFromEvent(event)
	if err != nil {
		return err
	}
	invoiceID, amount, currency, priceID, periodStart, periodEnd, err := invoiceDetails(event, plan)
	if err != nil {
		return err
	}
	if err := validateInvoiceAmount(plan, priceID, amount, currency); err != nil {
		return err
	}
	if priceID != subscription.StripePriceId {
		return fmt.Errorf("%w: subscription price mismatch", ErrRecurringPaymentMismatch)
	}
	_, err = model.ApplyStripeSubscriptionPaid(model.StripeSubscriptionInvoiceInput{
		PlanID:               subscription.PlanId,
		UserID:               subscription.UserId,
		StripeSubscriptionID: subscription.StripeSubscriptionId,
		StripeInvoiceID:      invoiceID,
		EventID:              event.ID,
		PeriodStart:          periodStart,
		PeriodEnd:            periodEnd,
		AmountPaidMinor:      amount,
		Currency:             currency,
		Status:               "paid",
	})
	if errors.Is(err, model.ErrStripeSubscriptionEnded) {
		return fmt.Errorf("%w: invoice arrived after subscription ended", ErrRecurringPaymentMismatch)
	}
	return asRecurringMismatch(err)
}

func handleInvoicePaymentFailed(event stripe.Event) error {
	subscription, plan, err := findRecurringSubscriptionFromEvent(event)
	if err != nil {
		return err
	}
	invoiceID, amount, currency, priceID, periodStart, periodEnd, err := invoiceDetails(event, plan)
	if err != nil {
		return err
	}
	if err := validateInvoiceAmount(plan, priceID, amount, currency); err != nil {
		return err
	}
	if priceID != subscription.StripePriceId {
		return fmt.Errorf("%w: subscription price mismatch", ErrRecurringPaymentMismatch)
	}
	_, err = model.MarkStripeSubscriptionPaymentFailedWithInvoice(model.StripeSubscriptionInvoiceInput{
		PlanID:               subscription.PlanId,
		UserID:               subscription.UserId,
		StripeSubscriptionID: subscription.StripeSubscriptionId,
		StripeInvoiceID:      invoiceID,
		EventID:              event.ID,
		PeriodStart:          periodStart,
		PeriodEnd:            periodEnd,
		AmountPaidMinor:      amount,
		Currency:             currency,
		Status:               "payment_failed",
	}, "invoice payment failed", common.GetTimestamp())
	if errors.Is(err, model.ErrStripeSubscriptionEnded) {
		return fmt.Errorf("%w: failed invoice arrived after subscription ended", ErrRecurringPaymentMismatch)
	}
	return asRecurringMismatch(err)
}

func handleSubscriptionUpdated(event stripe.Event) error {
	subscription, plan, err := findRecurringSubscriptionFromEvent(event)
	if err != nil {
		return err
	}
	if err := validateRecurringStripeObjectFields(event, plan, subscription.StripePriceId); err != nil {
		return err
	}
	if priceID := recurringMetadataString(event, "price_id"); priceID != "" && priceID != subscription.StripePriceId {
		return fmt.Errorf("%w: subscription price mismatch", ErrRecurringPaymentMismatch)
	}
	status := objectString(event.Data, "status")
	if status == "" {
		return fmt.Errorf("%w: subscription status missing", ErrRecurringPaymentMismatch)
	}
	switch status {
	case model.StripeSubscriptionStatusIncomplete, model.StripeSubscriptionStatusActive,
		model.StripeSubscriptionStatusPastDue, model.StripeSubscriptionStatusCanceled,
		model.StripeSubscriptionStatusUnpaid, "trialing":
	default:
		return fmt.Errorf("%w: unsupported subscription status", ErrRecurringPaymentMismatch)
	}
	cancelAtPeriodEnd, _ := objectBool(event.Data, "cancel_at_period_end")
	periodStart, _ := objectInt64(event.Data, "current_period_start")
	periodEnd, _ := objectInt64(event.Data, "current_period_end")
	err = model.UpdateStripeSubscriptionState(subscription.StripeSubscriptionId, status, cancelAtPeriodEnd, periodStart, periodEnd, common.GetTimestamp())
	if errors.Is(err, model.ErrStripeSubscriptionEnded) {
		return fmt.Errorf("%w: subscription update arrived after subscription ended", ErrRecurringPaymentMismatch)
	}
	return err
}

func handleSubscriptionDeleted(event stripe.Event) error {
	subscription, plan, err := findRecurringSubscriptionFromEvent(event)
	if err != nil {
		return err
	}
	if err := validateRecurringStripeObjectFields(event, plan, subscription.StripePriceId); err != nil {
		return err
	}
	return model.EndStripeSubscription(subscription.StripeSubscriptionId, common.GetTimestamp())
}

func objectString(data *stripe.EventData, keys ...string) string {
	if data == nil || data.Object == nil || len(keys) == 0 {
		return ""
	}
	value, ok := nestedObjectValue(data.Object, keys)
	if !ok || value == nil {
		return ""
	}
	if stringValue, ok := value.(string); ok {
		return strings.TrimSpace(stringValue)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func nestedObjectValue(value any, keys []string) (any, bool) {
	if len(keys) == 0 {
		return value, true
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		next, ok := typed[keys[0]]
		if !ok {
			return nil, false
		}
		return nestedObjectValue(next, keys[1:])
	case map[string]string:
		next, ok := typed[keys[0]]
		if !ok {
			return nil, false
		}
		return nestedObjectValue(next, keys[1:])
	case []interface{}:
		index, err := strconv.Atoi(keys[0])
		if err != nil || index < 0 || index >= len(typed) {
			return nil, false
		}
		return nestedObjectValue(typed[index], keys[1:])
	default:
		return nil, false
	}
}

func objectInt64(data *stripe.EventData, keys ...string) (int64, bool) {
	value := objectString(data, keys...)
	if value == "" {
		return 0, false
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed, true
	}
	parsedFloat, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsedFloat) || math.IsInf(parsedFloat, 0) || parsedFloat >= float64(math.MaxInt64) || parsedFloat < float64(math.MinInt64) || parsedFloat != math.Trunc(parsedFloat) {
		return 0, false
	}
	return int64(parsedFloat), true
}

func objectBool(data *stripe.EventData, keys ...string) (bool, bool) {
	value := strings.ToLower(objectString(data, keys...))
	parsed, err := strconv.ParseBool(value)
	return parsed, err == nil
}
