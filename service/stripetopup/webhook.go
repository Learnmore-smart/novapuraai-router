package stripetopup

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/emaildelivery"
	"github.com/QuantumNous/new-api/setting"

	"github.com/stripe/stripe-go/v85"
)

// ErrWebhookPaymentMismatch marks permanent payment evidence failures that
// require operator review rather than Stripe delivery retries.
var ErrWebhookPaymentMismatch = errors.New("stripe webhook payment evidence mismatch")

// ErrWebhookNotRetryable marks permanent admission failures. Stripe retries
// 4xx/5xx for days and then disables the endpoint, so the HTTP layer must
// acknowledge these after signature verification.
var ErrWebhookNotRetryable = errors.New("stripe webhook not retryable")

// ProcessVerifiedEvent handles a signature-verified Stripe event (idempotent).
func ProcessVerifiedEvent(ctx context.Context, event stripe.Event) error {
	// Mode / account guards
	if setting.StripeRequireTestKeys && event.Livemode {
		return fmt.Errorf("%w: reject livemode event in sandbox policy", ErrWebhookNotRetryable)
	}
	if setting.StripeAccountID != "" && event.Account != "" && event.Account != setting.StripeAccountID {
		return fmt.Errorf("%w: unexpected stripe account %s", ErrWebhookNotRetryable, event.Account)
	}

	now := common.GetTimestamp()
	inserted, err := model.TryInsertStripeWebhookEvent(&model.StripeWebhookEvent{
		EventID:   event.ID,
		EventType: string(event.Type),
		Livemode:  event.Livemode,
		AccountID: event.Account,
		CreatedAt: now,
	})
	if err != nil {
		return err
	}
	if !inserted {
		logger.LogInfo(ctx, fmt.Sprintf("stripe webhook duplicate ignored event_id=%s type=%s", event.ID, event.Type))
		return nil
	}

	var processErr error
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		processErr = handleCheckoutPaid(ctx, event, false)
	case stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded:
		processErr = handleCheckoutPaid(ctx, event, true)
	case stripe.EventTypeCheckoutSessionAsyncPaymentFailed:
		processErr = handleCheckoutFailed(ctx, event, "async_payment_failed")
	case stripe.EventTypeCheckoutSessionExpired:
		processErr = handleCheckoutExpired(ctx, event)
	case stripe.EventTypePaymentIntentPaymentFailed:
		processErr = handlePaymentIntentFailed(ctx, event)
	case stripe.EventTypeChargeRefunded:
		processErr = handleChargeRefunded(ctx, event)
	default:
		// charge.dispute.created etc. — mark manual review if we can resolve order
		if strings.Contains(string(event.Type), "dispute") {
			processErr = handleDispute(ctx, event)
			break
		}
		logger.LogInfo(ctx, fmt.Sprintf("stripe webhook ignored type=%s id=%s", event.Type, event.ID))
	}
	if processErr == nil || errors.Is(processErr, ErrWebhookPaymentMismatch) || errors.Is(processErr, ErrWebhookNotRetryable) {
		return processErr
	}
	if deleteErr := model.DB.Where("event_id = ?", event.ID).Delete(&model.StripeWebhookEvent{}).Error; deleteErr != nil {
		return fmt.Errorf("%w; release webhook event claim: %v", processErr, deleteErr)
	}
	return processErr
}

func handleCheckoutPaid(ctx context.Context, event stripe.Event, async bool) error {
	orderID := novapuraOrderID(event)
	if orderID == "" {
		logger.LogInfo(ctx, fmt.Sprintf("stripe webhook ignored type=%s id=%s reason=missing_order_id", event.Type, event.ID))
		return nil
	}

	// Update webhook event row with order id (best effort)
	_ = model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", event.ID).Update("order_id", orderID)

	status := eventObjectString(event, "status")
	paymentStatus := eventObjectString(event, "payment_status")
	if !async && status != "" && status != "complete" {
		logger.LogWarn(ctx, fmt.Sprintf("checkout not complete order=%s status=%s", orderID, status))
		return nil
	}
	if paymentStatus != "" && paymentStatus != "paid" && paymentStatus != "no_payment_required" {
		logger.LogInfo(ctx, fmt.Sprintf("checkout waiting payment order=%s payment_status=%s", orderID, paymentStatus))
		return nil
	}

	order, err := model.GetStripeTopupOrderByOrderID(orderID)
	if err != nil {
		// Fall through: legacy trade_no path not handled here
		return fmt.Errorf("order not found: %s", orderID)
	}

	// Verify amount + currency against Stripe
	amountTotalStr := eventObjectString(event, "amount_total")
	currency := strings.ToLower(eventObjectString(event, "currency"))
	amountTotal, parseErr := strconv.ParseInt(amountTotalStr, 10, 64)
	if parseErr != nil || amountTotal <= 0 {
		_ = model.MarkStripeTopupOrderStatus(orderID, "*", model.StripeOrderManualReview, "missing or invalid amount")
		return fmt.Errorf("%w: invalid amount order=%s value=%q", ErrWebhookPaymentMismatch, orderID, amountTotalStr)
	}
	if currency == "" {
		_ = model.MarkStripeTopupOrderStatus(orderID, "*", model.StripeOrderManualReview, "missing currency")
		return fmt.Errorf("%w: missing currency order=%s", ErrWebhookPaymentMismatch, orderID)
	}
	if currency != strings.ToLower(order.PresentmentCurrency) {
		_ = model.MarkStripeTopupOrderStatus(orderID, "*", model.StripeOrderManualReview, "currency mismatch")
		return fmt.Errorf("%w: currency mismatch order=%s got=%s want=%s", ErrWebhookPaymentMismatch, orderID, currency, order.PresentmentCurrency)
	}
	if amountTotal != order.PresentmentAmountMinor {
		_ = model.MarkStripeTopupOrderStatus(orderID, "*", model.StripeOrderManualReview, "amount mismatch")
		return fmt.Errorf("%w: amount mismatch order=%s got=%d want=%d", ErrWebhookPaymentMismatch, orderID, amountTotal, order.PresentmentAmountMinor)
	}
	sessionID := eventObjectString(event, "id")
	if sessionID == "" || (order.StripeCheckoutSessionID != "" && order.StripeCheckoutSessionID != sessionID) {
		_ = model.MarkStripeTopupOrderStatus(orderID, "*", model.StripeOrderManualReview, "checkout session mismatch")
		return fmt.Errorf("%w: checkout session mismatch order=%s", ErrWebhookPaymentMismatch, orderID)
	}

	customerID := eventObjectID(event, "customer")
	pi := eventObjectID(event, "payment_intent")

	// Mark paid then credit (credit is idempotent to credited)
	paidAt := common.GetTimestamp()
	_ = model.DB.Model(&model.StripeTopupOrder{}).Where("order_id = ? AND status IN ?", orderID,
		[]string{model.StripeOrderPending, model.StripeOrderCheckoutCreated}).
		Updates(map[string]interface{}{
			"status":                     model.StripeOrderPaid,
			"stripe_customer_id":         customerID,
			"stripe_checkout_session_id": sessionID,
			"stripe_payment_intent_id":   pi,
			"paid_at":                    paidAt,
		})

	already, err := model.CreditStripeTopupOrder(orderID, customerID, pi, sessionID)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("credit failed order=%s err=%q", orderID, err.Error()))
		return err
	}
	if already {
		logger.LogInfo(ctx, fmt.Sprintf("credit already applied order=%s", orderID))
	} else {
		logger.LogInfo(ctx, fmt.Sprintf("credit success order=%s async=%v", orderID, async))
	}

	recipient, emailErr := model.GetUserEmail(order.UserId)
	if emailErr != nil || strings.TrimSpace(recipient) == "" {
		logger.LogWarn(ctx, fmt.Sprintf("receipt email skipped order=%s reason=recipient_unavailable", orderID))
		return nil
	}
	receipt, buildErr := emaildelivery.BuildReceiptMessage(emaildelivery.ReceiptTemplateData{
		SystemName: common.SystemName,
		Recipient:  recipient,
		ReceiptID:  orderID,
		Amount:     fmt.Sprintf("%.2f %s", float64(order.PresentmentAmountMinor)/100, strings.ToUpper(order.PresentmentCurrency)),
		PaidAt:     time.Unix(paidAt, 0).UTC().Format(time.RFC3339),
	})
	if buildErr != nil {
		logger.LogWarn(ctx, fmt.Sprintf("receipt email failed order=%s reason=template", orderID))
		return nil
	}
	if _, emailErr = emaildelivery.SendTransactionalEmail(ctx, emaildelivery.SendRequest{
		BusinessKey: "stripe-receipt:" + orderID,
		Message:     receipt,
	}); emailErr != nil {
		logger.LogWarn(ctx, fmt.Sprintf("receipt email failed order=%s reason=%s", orderID, emailErr.Error()))
	}
	return nil
}

func handleCheckoutFailed(ctx context.Context, event stripe.Event, reason string) error {
	orderID := novapuraOrderID(event)
	if orderID == "" {
		return nil
	}
	return model.MarkStripeTopupOrderStatus(orderID, "*", model.StripeOrderFailed, reason)
}

func handleCheckoutExpired(ctx context.Context, event stripe.Event) error {
	orderID := eventObjectString(event, "client_reference_id")
	if orderID == "" {
		return nil
	}
	return model.MarkStripeTopupOrderStatus(orderID, "*", model.StripeOrderExpired, "checkout_expired")
}

func handlePaymentIntentFailed(ctx context.Context, event stripe.Event) error {
	orderID := eventObjectString(event, "metadata", "novapura_order_id")
	if orderID == "" {
		return nil
	}
	return model.MarkStripeTopupOrderStatus(orderID, "*", model.StripeOrderFailed, "payment_intent_failed")
}

func handleChargeRefunded(ctx context.Context, event stripe.Event) error {
	pi := eventObjectID(event, "payment_intent")
	orderID := eventObjectString(event, "metadata", "novapura_order_id")
	if orderID == "" && pi != "" {
		var o model.StripeTopupOrder
		if err := model.DB.Where("stripe_payment_intent_id = ?", pi).First(&o).Error; err == nil {
			orderID = o.OrderID
		}
	}
	if orderID == "" {
		logger.LogWarn(ctx, "refund event without order id")
		return nil
	}
	return ReconcileRefund(ctx, orderID)
}

func handleDispute(ctx context.Context, event stripe.Event) error {
	orderID := eventObjectString(event, "metadata", "novapura_order_id")
	if orderID == "" {
		return nil
	}
	return model.MarkStripeTopupOrderStatus(orderID, "*", model.StripeOrderManualReview, "dispute")
}

func novapuraOrderID(event stripe.Event) string {
	if orderID := eventObjectString(event, "client_reference_id"); orderID != "" {
		return orderID
	}
	return eventObjectString(event, "metadata", "novapura_order_id")
}

func eventObjectID(event stripe.Event, keys ...string) string {
	switch typed := eventObjectValue(event, keys...).(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]interface{}:
		if id, ok := typed["id"].(string); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func eventObjectString(event stripe.Event, keys ...string) string {
	value := eventObjectValue(event, keys...)
	if value == nil {
		return ""
	}
	if stringValue, ok := value.(string); ok {
		return strings.TrimSpace(stringValue)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func eventObjectValue(event stripe.Event, keys ...string) any {
	if event.Data == nil || event.Data.Object == nil || len(keys) == 0 {
		return nil
	}
	current := any(event.Data.Object)
	for _, key := range keys {
		typed, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		next, exists := typed[key]
		if !exists {
			return nil
		}
		current = next
	}
	return current
}
