package stripetopup

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/checkout/session"
	"gorm.io/gorm"
)

// CheckoutResult returned to the client after Session creation.
type CheckoutResult struct {
	OrderID     string       `json:"order_id"`
	CheckoutURL string       `json:"checkout_url"`
	ExpiresAt   int64        `json:"expires_at"`
	Quote       *QuoteResult `json:"quote"`
}

// ValidateStripeSecrets ensures sandbox keys in non-live deployments.
func ValidateStripeSecrets() error {
	secret := strings.TrimSpace(setting.StripeApiSecret)
	publishable := strings.TrimSpace(setting.StripePublishableKey)
	webhookSecret := strings.TrimSpace(setting.StripeWebhookSecret)
	if secret == "" {
		return fmt.Errorf("stripe secret key not configured")
	}
	if setting.StripeRequireTestKeys {
		if strings.HasPrefix(secret, "sk_live") || strings.HasPrefix(secret, "rk_live") {
			return fmt.Errorf("live stripe keys rejected outside production policy")
		}
		if !strings.HasPrefix(secret, "sk_test") && !strings.HasPrefix(secret, "rk_test") {
			return fmt.Errorf("invalid stripe secret key prefix")
		}
		if !strings.HasPrefix(publishable, "pk_test") {
			return fmt.Errorf("stripe test publishable key not configured")
		}
	} else {
		if !strings.HasPrefix(secret, "sk_live") && !strings.HasPrefix(secret, "rk_live") {
			return fmt.Errorf("test stripe secret key rejected in production")
		}
		if !strings.HasPrefix(publishable, "pk_live") {
			return fmt.Errorf("test stripe publishable key rejected in production")
		}
	}
	if !strings.HasPrefix(webhookSecret, "whsec_") {
		return fmt.Errorf("stripe webhook signing secret not configured")
	}
	if strings.TrimSpace(setting.StripeTopupProductID) == "" {
		return fmt.Errorf("STRIPE_TOPUP_PRODUCT_ID not configured")
	}
	if strings.TrimSpace(setting.StripeAccountID) == "" {
		return fmt.Errorf("stripe account ID not configured")
	}
	return nil
}

// CreateCheckout builds order + Stripe Checkout Session (mode=payment, dynamic price_data).
func CreateCheckout(user *model.User, req QuoteRequest, successURL, cancelURL string) (*CheckoutResult, error) {
	if user == nil || user.Id <= 0 {
		return nil, fmt.Errorf("unauthorized")
	}
	if !setting.StripeTopupEnabled {
		return nil, fmt.Errorf("stripe top-up is disabled")
	}
	if err := ValidateStripeSecrets(); err != nil {
		return nil, err
	}

	quote, err := BuildQuote(user.Id, req)
	if err != nil {
		return nil, err
	}

	orderID := "np_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	idem := fmt.Sprintf("topup_%d_%s_%d_%d", user.Id, quote.Currency, quote.AmountMinor, time.Now().UnixNano())

	order, err := createReservedOrder(user, quote, orderID, idem)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	stripe.Key = setting.StripeApiSecret

	base := strings.TrimRight(system_setting.ServerAddress, "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	if successURL == "" {
		successURL = fmt.Sprintf("%s/wallet/topup/success?order_id=%s", base, orderID)
	} else if !strings.Contains(successURL, "order_id") {
		sep := "?"
		if strings.Contains(successURL, "?") {
			sep = "&"
		}
		successURL = successURL + sep + "order_id=" + orderID
	}
	if cancelURL == "" {
		cancelURL = base + "/wallet/topup?canceled=1"
	}

	environment := "sandbox"
	if strings.HasPrefix(setting.StripeApiSecret, "sk_live") || strings.HasPrefix(setting.StripeApiSecret, "rk_live") {
		environment = "live"
	}

	params := &stripe.CheckoutSessionParams{
		Mode:                  stripe.String(string(stripe.CheckoutSessionModePayment)),
		ClientReferenceID:     stripe.String(orderID),
		SuccessURL:            stripe.String(successURL),
		CancelURL:             stripe.String(cancelURL),
		Locale:                stripe.String("auto"),
		IntegrationIdentifier: stripe.String("novapura_router_qjmvkzpt"),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(quote.Currency),
					Product:    stripe.String(setting.StripeTopupProductID),
					UnitAmount: stripe.Int64(quote.AmountMinor),
				},
			},
		},
		Metadata: map[string]string{
			"novapura_user_id":        fmt.Sprintf("%d", user.Id),
			"novapura_order_id":       orderID,
			"environment":             environment,
			"selected_currency":       quote.Currency,
			"canonical_credit_amount": fmt.Sprintf("%d", quote.PaidCreditMicroUSD),
			"promo_credit_amount":     fmt.Sprintf("%d", quote.PromoCreditMicroUSD),
			"promotion_snapshot_id":   fmt.Sprintf("%d", quote.PromotionTierID),
		},
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"novapura_order_id": orderID,
				"novapura_user_id":  fmt.Sprintf("%d", user.Id),
			},
		},
	}
	params.SetIdempotencyKey(idem)

	if user.StripeCustomer != "" {
		params.Customer = stripe.String(user.StripeCustomer)
	} else {
		params.CustomerCreation = stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways))
		if user.Email != "" {
			params.CustomerEmail = stripe.String(user.Email)
		}
	}

	// Dynamic payment methods (card, wallets, Alipay/WeChat when enabled in Dashboard).
	// Do not hard-code payment_method_types.

	sess, err := session.New(params)
	if err != nil {
		_ = model.MarkStripeTopupOrderStatus(orderID, model.StripeOrderPending, model.StripeOrderFailed, err.Error())
		return nil, fmt.Errorf("stripe checkout: %w", err)
	}

	order.Status = model.StripeOrderCheckoutCreated
	order.StripeCheckoutSessionID = sess.ID
	if sess.Customer != nil {
		order.StripeCustomerID = sess.Customer.ID
	}
	if sess.PaymentIntent != nil {
		order.StripePaymentIntentID = sess.PaymentIntent.ID
	}
	if sess.ExpiresAt > 0 {
		order.CheckoutExpiresAt = sess.ExpiresAt
	}
	if err := model.DB.Save(order).Error; err != nil {
		return nil, fmt.Errorf("update order session: %w", err)
	}

	return &CheckoutResult{
		OrderID:     orderID,
		CheckoutURL: sess.URL,
		ExpiresAt:   order.CheckoutExpiresAt,
		Quote:       quote,
	}, nil
}

func createReservedOrder(user *model.User, quote *QuoteResult, orderID, idempotencyKey string) (*model.StripeTopupOrder, error) {
	if user == nil || user.Id <= 0 || quote == nil || orderID == "" || idempotencyKey == "" {
		return nil, fmt.Errorf("invalid reserved order")
	}
	order := &model.StripeTopupOrder{
		OrderID:                orderID,
		UserId:                 user.Id,
		StripeCustomerID:       user.StripeCustomer,
		Status:                 model.StripeOrderPending,
		PresentmentCurrency:    quote.Currency,
		PresentmentAmountMinor: quote.AmountMinor,
		PaidCreditAmountMinor:  quote.PaidCreditAmountMinor,
		PromoCreditAmountMinor: quote.PromoCreditAmountMinor,
		TotalCreditAmountMinor: quote.TotalCreditAmountMinor,
		FxRateSnapshot:         quote.FxRateSnapshot,
		PaidCreditMicroUSD:     quote.PaidCreditMicroUSD,
		PromoCreditMicroUSD:    quote.PromoCreditMicroUSD,
		TotalCreditMicroUSD:    quote.TotalCreditMicroUSD,
		PaidQuota:              quote.PaidQuota,
		PromoQuota:             quote.PromoQuota,
		PromotionSnapshotJSON:  quote.PromotionSnapshotJSON,
		PromotionTierID:        quote.PromotionTierID,
		PromoExpiryDays:        quote.PromoExpiryDays,
		IdempotencyKey:         idempotencyKey,
		CreatedAt:              common.GetTimestamp(),
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if order.PromoCreditMicroUSD <= 0 {
			return nil
		}
		return model.ReserveTopupPromotionWithTx(tx, order.OrderID, order.UserId, order.PromotionTierID, order.PromoCreditMicroUSD)
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}
