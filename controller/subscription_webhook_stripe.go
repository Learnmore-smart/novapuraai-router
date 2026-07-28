package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/stripe/stripe-go/v85"
	"gorm.io/gorm"
)

// ErrSubscriptionWebhookIgnored marks events that are not for our subscriptions
// (e.g. a subscription created outside NovaPura). The caller treats this as a
// successful no-op so Stripe does not retry.
var ErrSubscriptionWebhookIgnored = errors.New("subscription webhook event not applicable")

// IsSubscriptionStripeEvent reports whether a Stripe event should be routed to
// the NovaPura v2 subscription webhook handler instead of the topup handler.
//
// The routing rules are:
//   - invoice.paid, invoice.payment_failed, customer.subscription.updated,
//     customer.subscription.deleted are always subscription events (topup
//     never produces these).
//   - checkout.session.completed and checkout.session.expired are subscription
//     events only when the session carries NovaPura v2 metadata
//     (novapura_mode = auto_renew|prepaid). Legacy subscription orders (sub_ref_
//     prefix without novapura_mode) fall through to the existing topup/legacy
//     handler.
func IsSubscriptionStripeEvent(event stripe.Event) bool {
	switch event.Type {
	case stripe.EventTypeInvoicePaid,
		stripe.EventTypeInvoicePaymentFailed,
		stripe.EventTypeCustomerSubscriptionUpdated,
		stripe.EventTypeCustomerSubscriptionDeleted:
		return true
	case stripe.EventTypeCheckoutSessionCompleted,
		stripe.EventTypeCheckoutSessionExpired:
		// stripe's GetObjectValue panics when an intermediate path key is absent
		// (e.g. a checkout session with no metadata block, such as a legacy
		// topup session). Recover and treat such events as non-subscription so
		// they fall through to the topup handler instead of crashing the
		// webhook endpoint.
		mode := subscriptionEventMode(event)
		return mode == subscriptionCheckoutModeAutoRenew || mode == subscriptionCheckoutModePrepaid
	}
	return false
}

// subscriptionEventMode extracts the novapura_mode metadata value from a
// checkout.session.* event. Returns "" when the metadata block is absent or
// the value is missing. The recover guards against stripe's GetObjectValue
// panicking on intermediate-path-absent payloads.
func subscriptionEventMode(event stripe.Event) string {
	var mode string
	func() {
		defer func() { _ = recover() }()
		mode = event.GetObjectValue("metadata", "novapura_mode")
	}()
	return mode
}

// ProcessSubscriptionStripeEvent handles a signature-verified Stripe event for
// the NovaPura v2 subscription flow. It performs its own idempotency tracking
// via the shared stripe_webhook_events table (same as the topup handler) and
// dispatches to subscription-specific sub-handlers.
//
// On processing failure (except ErrSubscriptionWebhookIgnored) the webhook
// event claim is released so Stripe retries the event.
func ProcessSubscriptionStripeEvent(ctx context.Context, event stripe.Event) error {
	// Mode / account guards (mirror stripetopup.ProcessVerifiedEvent).
	if setting.StripeRequireTestKeys && event.Livemode {
		return fmt.Errorf("reject livemode event in sandbox policy")
	}
	if setting.StripeAccountID != "" && event.Account != "" && event.Account != setting.StripeAccountID {
		return fmt.Errorf("unexpected stripe account %s", event.Account)
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
		logger.LogInfo(ctx, fmt.Sprintf("subscription stripe webhook duplicate ignored event_id=%s type=%s", event.ID, event.Type))
		return nil
	}

	var processErr error
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		processErr = handleSubscriptionCheckoutCompleted(ctx, event)
	case stripe.EventTypeCheckoutSessionExpired:
		processErr = handleSubscriptionCheckoutExpired(ctx, event)
	case stripe.EventTypeInvoicePaid:
		processErr = handleSubscriptionInvoicePaid(ctx, event)
	case stripe.EventTypeInvoicePaymentFailed:
		processErr = handleSubscriptionInvoicePaymentFailed(ctx, event)
	case stripe.EventTypeCustomerSubscriptionUpdated:
		processErr = handleSubscriptionUpdated(ctx, event)
	case stripe.EventTypeCustomerSubscriptionDeleted:
		processErr = handleSubscriptionDeleted(ctx, event)
	default:
		logger.LogInfo(ctx, fmt.Sprintf("subscription stripe webhook ignored type=%s id=%s", event.Type, event.ID))
	}

	if processErr == nil || errors.Is(processErr, ErrSubscriptionWebhookIgnored) {
		return nil
	}
	// Release the webhook event claim so Stripe can retry (mirrors ProcessVerifiedEvent).
	if deleteErr := model.DB.Where("event_id = ?", event.ID).Delete(&model.StripeWebhookEvent{}).Error; deleteErr != nil {
		return fmt.Errorf("%w; release webhook event claim: %v", processErr, deleteErr)
	}
	return processErr
}

// handleSubscriptionCheckoutCompleted completes a NovaPura v2 subscription
// order after the Stripe Checkout Session is paid. It handles both auto-renew
// (subscription mode) and prepaid (payment mode) checkouts.
func handleSubscriptionCheckoutCompleted(ctx context.Context, event stripe.Event) error {
	orderID := event.GetObjectValue("client_reference_id")
	if orderID == "" {
		orderID = event.GetObjectValue("metadata", "novapura_order_id")
	}
	if orderID == "" {
		logger.LogWarn(ctx, "subscription checkout.completed missing order id")
		return ErrSubscriptionWebhookIgnored
	}

	mode := event.GetObjectValue("metadata", "novapura_mode")
	if mode != subscriptionCheckoutModeAutoRenew && mode != subscriptionCheckoutModePrepaid {
		// Not a NovaPura v2 event; let the topup/legacy handler process it.
		return ErrSubscriptionWebhookIgnored
	}

	// Best-effort: record the order id on the webhook event row for traceability.
	_ = model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", event.ID).Update("order_id", orderID)

	status := event.GetObjectValue("status")
	paymentStatus := event.GetObjectValue("payment_status")
	if status != "" && status != "complete" {
		logger.LogWarn(ctx, fmt.Sprintf("subscription checkout not complete order=%s status=%s", orderID, status))
		return nil
	}
	if paymentStatus != "" && paymentStatus != "paid" && paymentStatus != "no_payment_required" {
		logger.LogInfo(ctx, fmt.Sprintf("subscription checkout waiting payment order=%s payment_status=%s", orderID, paymentStatus))
		return nil
	}

	stripeSubscriptionId := event.GetObjectValue("subscription")
	stripeCustomerId := event.GetObjectValue("customer")

	// For prepaid mode (payment mode), there is no Stripe subscription; use
	// the payment_intent as the linkage key instead.
	if mode == subscriptionCheckoutModePrepaid && stripeSubscriptionId == "" {
		stripeSubscriptionId = event.GetObjectValue("payment_intent")
	}

	// Persist the Stripe Customer ID on the user so future checkouts and the
	// Customer Portal can reuse it.
	if stripeCustomerId != "" {
		userIdStr := event.GetObjectValue("metadata", "novapura_user_id")
		if userId, parseErr := strconv.Atoi(userIdStr); parseErr == nil && userId > 0 {
			_ = model.UpdateUserStripeCustomer(userId, stripeCustomerId)
		}
	}

	providerPayload := buildSubscriptionProviderPayload(event)
	// Prepaid mode: complete via the stacking-aware path. If the user already
	// has a prepaid_active subscription for the same plan, this extends its
	// EndTime instead of creating a new UserSubscription row. Auto-renew keeps
	// the original CompleteSubscriptionOrderV2 path (no stacking — auto-renew
	// duplicates are blocked at checkout time).
	if mode == subscriptionCheckoutModePrepaid {
		// The prepaid months are stored on the order; load it to pass the
		// value into CompletePrepaidSubscriptionOrderOrExtend. The function
		// re-loads the order inside its own transaction (with FOR UPDATE), so
		// this read is best-effort and only used to extract PrepaidMonths.
		existingOrder := model.GetSubscriptionOrderByTradeNo(orderID)
		if existingOrder == nil {
			logger.LogWarn(ctx, fmt.Sprintf("subscription checkout completed but order not found order=%s", orderID))
			return ErrSubscriptionWebhookIgnored
		}
		if err := model.CompletePrepaidSubscriptionOrderOrExtend(orderID, providerPayload, existingOrder.PrepaidMonths); err != nil {
			if errors.Is(err, model.ErrSubscriptionOrderNotFound) {
				logger.LogWarn(ctx, fmt.Sprintf("subscription checkout completed but order not found order=%s", orderID))
				return ErrSubscriptionWebhookIgnored
			}
			logger.LogError(ctx, fmt.Sprintf("subscription checkout complete failed order=%s err=%q", orderID, err.Error()))
			return err
		}
		logger.LogInfo(ctx, fmt.Sprintf("subscription checkout complete success order=%s mode=%s sub=%s", orderID, mode, stripeSubscriptionId))
		return nil
	}
	if err := model.CompleteSubscriptionOrderV2(orderID, providerPayload, stripeSubscriptionId, stripeCustomerId, mode); err != nil {
		if errors.Is(err, model.ErrSubscriptionOrderNotFound) {
			logger.LogWarn(ctx, fmt.Sprintf("subscription checkout completed but order not found order=%s", orderID))
			return ErrSubscriptionWebhookIgnored
		}
		logger.LogError(ctx, fmt.Sprintf("subscription checkout complete failed order=%s err=%q", orderID, err.Error()))
		return err
	}
	logger.LogInfo(ctx, fmt.Sprintf("subscription checkout complete success order=%s mode=%s sub=%s", orderID, mode, stripeSubscriptionId))
	return nil
}

// handleSubscriptionInvoicePaid handles recurring invoice payments for
// auto-renew subscriptions. The first invoice is paid at checkout time (and
// the checkout.session.completed handler already created the subscription);
// subsequent invoices represent renewals and extend the subscription's EndTime.
//
// Affiliate commission is settled on every renewal invoice (using the Stripe
// invoice ID as the commission topUpId, so each renewal generates a separate
// commission row). The first invoice's commission was already settled by
// CompleteSubscriptionOrderV2 using the order TradeNo, so we skip it here
// (billing_reason == "subscription_create"). Commission settlement failure
// does NOT block the renewal — log and continue.
func handleSubscriptionInvoicePaid(ctx context.Context, event stripe.Event) error {
	stripeSubId := event.GetObjectValue("subscription")
	if stripeSubId == "" {
		// Invoice without a subscription link — not ours.
		return ErrSubscriptionWebhookIgnored
	}

	// Best-effort: record the subscription id on the webhook event row.
	_ = model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", event.ID).Update("order_id", stripeSubId)

	// Determine the billing period end. Stripe invoices carry
	// lines.data[0].period.end or subscription_details.metadata. We use
	// period_end as the canonical source (it is the timestamp by which the
	// customer must pay, which coincides with the subscription period end for
	// renewals).
	periodEndStr := event.GetObjectValue("period_end")
	periodEnd, parseErr := strconv.ParseInt(periodEndStr, 10, 64)
	if parseErr != nil || periodEnd <= 0 {
		logger.LogWarn(ctx, fmt.Sprintf("subscription invoice.paid missing period_end sub=%s value=%q", stripeSubId, periodEndStr))
		return nil
	}

	// Check if a subscription exists for this Stripe ID. If not, it may be a
	// first invoice that arrived before checkout.session.completed (race); the
	// checkout handler will create the subscription with a placeholder EndTime,
	// and this event will be a no-op due to the idempotency guard.
	sub, err := model.FindUserSubscriptionByStripeId(stripeSubId)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription invoice.paid lookup failed sub=%s err=%q", stripeSubId, err.Error()))
		return err
	}
	if sub == nil {
		// Not our subscription — ignore (could be a subscription created
		// outside NovaPura, or the checkout handler hasn't run yet).
		logger.LogInfo(ctx, fmt.Sprintf("subscription invoice.paid no local subscription sub=%s", stripeSubId))
		return ErrSubscriptionWebhookIgnored
	}

	if err := model.RenewUserSubscriptionFromStripe(stripeSubId, periodEnd); err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription invoice.paid renew failed sub=%s err=%q", stripeSubId, err.Error()))
		return err
	}
	logger.LogInfo(ctx, fmt.Sprintf("subscription invoice.paid renew success sub=%s period_end=%d", stripeSubId, periodEnd))

	// Settle affiliate commission on renewal. The first invoice
	// (billing_reason == "subscription_create") was already commissioned by
	// CompleteSubscriptionOrderV2 using the order TradeNo as topUpId, so only
	// settle for actual renewals. Using the Stripe invoice ID as topUpId
	// guarantees each renewal gets its own commission row (the unique index
	// is on (topup_id, inviter_id)).
	billingReason := event.GetObjectValue("billing_reason")
	invoiceID := event.GetObjectValue("id")
	if billingReason != "subscription_create" && invoiceID != "" {
		settleSubscriptionRenewalCommission(ctx, event, sub.UserId, invoiceID)
	}
	return nil
}

// settleSubscriptionRenewalCommission converts the invoice's amount_paid to
// USD cents and credits the inviter's commission in a dedicated transaction.
// Idempotent: SettleRechargeCommission's unique-index check guarantees a
// renewal invoice never pays commission twice even if Stripe retries the
// event. Failure is logged but not propagated — a commission glitch must not
// block the renewal.
func settleSubscriptionRenewalCommission(ctx context.Context, event stripe.Event, userId int, invoiceID string) {
	amountPaidStr := event.GetObjectValue("amount_paid")
	amountPaid, parseErr := strconv.ParseInt(amountPaidStr, 10, 64)
	if parseErr != nil || amountPaid <= 0 {
		logger.LogInfo(ctx, fmt.Sprintf("subscription invoice.paid commission skipped: missing amount_paid invoice=%s", invoiceID))
		return
	}
	currency := strings.ToUpper(event.GetObjectValue("currency"))
	paidCentsUSD := model.ConvertAmountToUSDCents(float64(amountPaid)/100.0, currency)
	if paidCentsUSD <= 0 {
		logger.LogInfo(ctx, fmt.Sprintf("subscription invoice.paid commission skipped: unconvertible amount invoice=%s currency=%s amount=%d", invoiceID, currency, amountPaid))
		return
	}

	settlement, settleErr := settleSubscriptionCommissionInTx(userId, invoiceID, paidCentsUSD)
	if settleErr != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription invoice.paid commission settle failed invoice=%s err=%q", invoiceID, settleErr.Error()))
		return
	}
	if settlement != nil {
		logger.LogInfo(ctx, fmt.Sprintf("subscription invoice.paid commission settled invoice=%s inviter=%d cents=%d",
			invoiceID, settlement.InviterId, settlement.CommissionCents))
	}
}

// settleSubscriptionCommissionInTx wraps SettleRechargeCommission in a
// dedicated transaction. Used by the renewal path (the purchase path settles
// inside CompleteSubscriptionOrderV2's own transaction).
func settleSubscriptionCommissionInTx(userId int, topUpId string, paidCentsUSD int64) (*model.CommissionSettlement, error) {
	var settlement *model.CommissionSettlement
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		s, err := model.SettleRechargeCommission(tx, userId, topUpId, paidCentsUSD, "USD")
		if err != nil {
			return err
		}
		settlement = s
		return nil
	})
	if err != nil {
		return nil, err
	}
	return settlement, nil
}

// handleSubscriptionInvoicePaymentFailed marks a subscription as past_due when
// a renewal invoice payment fails. Stripe will retry the charge; if all retries
// fail, customer.subscription.deleted is eventually fired.
func handleSubscriptionInvoicePaymentFailed(ctx context.Context, event stripe.Event) error {
	stripeSubId := event.GetObjectValue("subscription")
	if stripeSubId == "" {
		return ErrSubscriptionWebhookIgnored
	}

	_ = model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", event.ID).Update("order_id", stripeSubId)

	sub, err := model.FindUserSubscriptionByStripeId(stripeSubId)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription invoice.payment_failed lookup failed sub=%s err=%q", stripeSubId, err.Error()))
		return err
	}
	if sub == nil {
		return ErrSubscriptionWebhookIgnored
	}

	if err := model.MarkUserSubscriptionPastDue(stripeSubId); err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription invoice.payment_failed mark past_due failed sub=%s err=%q", stripeSubId, err.Error()))
		return err
	}
	logger.LogWarn(ctx, fmt.Sprintf("subscription invoice.payment_failed marked past_due sub=%s", stripeSubId))
	return nil
}

// handleSubscriptionUpdated syncs the local subscription status with the Stripe
// subscription status. This handles cancel_at_period_end transitions (user
// cancels via Stripe Customer Portal) and status changes (active -> past_due).
func handleSubscriptionUpdated(ctx context.Context, event stripe.Event) error {
	stripeSubId := event.GetObjectValue("id")
	if stripeSubId == "" {
		return ErrSubscriptionWebhookIgnored
	}

	_ = model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", event.ID).Update("order_id", stripeSubId)

	sub, err := model.FindUserSubscriptionByStripeId(stripeSubId)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription updated lookup failed sub=%s err=%q", stripeSubId, err.Error()))
		return err
	}
	if sub == nil {
		return ErrSubscriptionWebhookIgnored
	}

	stripeStatus := event.GetObjectValue("status")
	cancelAtPeriodEndStr := event.GetObjectValue("cancel_at_period_end")
	cancelAtPeriodEnd := cancelAtPeriodEndStr == "true"
	periodEndStr := event.GetObjectValue("current_period_end")
	periodEnd, _ := strconv.ParseInt(periodEndStr, 10, 64)

	if err := model.SetUserSubscriptionStatusFromStripe(stripeSubId, stripeStatus, cancelAtPeriodEnd, periodEnd); err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription updated status sync failed sub=%s err=%q", stripeSubId, err.Error()))
		return err
	}
	logger.LogInfo(ctx, fmt.Sprintf("subscription updated status sync sub=%s status=%s cancel_at_period_end=%v", stripeSubId, stripeStatus, cancelAtPeriodEnd))
	return nil
}

// handleSubscriptionDeleted marks a subscription as canceled after the Stripe
// subscription is deleted (either by the user via the Customer Portal, by
// Stripe after all payment retries fail, or by an admin). It downgrades the
// user group and reverses any issued coupon redemptions.
//
// Commission reversal is NOT triggered here: a deletion is not a refund. A
// subscription can be canceled at period end (no refund) or deleted
// immediately (prorated refund). Per requirement 13, affiliate commission is
// only reversed on actual refunds, which arrive as `charge.refunded` events
// (handled by handleSubscriptionChargeRefunded via the topup webhook path).
func handleSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	stripeSubId := event.GetObjectValue("id")
	if stripeSubId == "" {
		return ErrSubscriptionWebhookIgnored
	}

	_ = model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", event.ID).Update("order_id", stripeSubId)

	sub, err := model.FindUserSubscriptionByStripeId(stripeSubId)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription deleted lookup failed sub=%s err=%q", stripeSubId, err.Error()))
		return err
	}
	if sub == nil {
		return ErrSubscriptionWebhookIgnored
	}

	if err := model.CancelUserSubscriptionFromStripe(stripeSubId); err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription deleted cancel failed sub=%s err=%q", stripeSubId, err.Error()))
		return err
	}
	logger.LogInfo(ctx, fmt.Sprintf("subscription deleted canceled sub=%s", stripeSubId))
	return nil
}

// handleSubscriptionCheckoutExpired handles checkout.session.expired events for
// NovaPura v2 subscription orders. It marks the order expired and releases any
// reserved coupon redemption so the user's per-user coupon quota is not burned
// by an abandoned checkout. Idempotent: an order already in a terminal status
// (or with no coupon reservation) is a no-op.
func handleSubscriptionCheckoutExpired(ctx context.Context, event stripe.Event) error {
	orderID := event.GetObjectValue("client_reference_id")
	if orderID == "" {
		orderID = event.GetObjectValue("metadata", "novapura_order_id")
	}
	if orderID == "" {
		logger.LogWarn(ctx, "subscription checkout.expired missing order id")
		return ErrSubscriptionWebhookIgnored
	}
	mode := subscriptionEventMode(event)
	if mode != subscriptionCheckoutModeAutoRenew && mode != subscriptionCheckoutModePrepaid {
		// Not a NovaPura v2 event; let the topup/legacy handler process it.
		return ErrSubscriptionWebhookIgnored
	}

	// Best-effort: record the order id on the webhook event row for traceability.
	_ = model.DB.Model(&model.StripeWebhookEvent{}).Where("event_id = ?", event.ID).Update("order_id", orderID)

	// Expire the order and release the coupon in a single transaction so a
	// crash between the two never leaves a reserved coupon stranded. Both
	// MarkSubscriptionOrderStatus and ReleaseSubscriptionCouponWithTx are
	// idempotent, so a Stripe retry re-runs this safely.
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		// Mark the order expired. MarkSubscriptionOrderStatus refuses to
		// downgrade a paid order, so a retry after checkout.session.completed
		// is a safe no-op.
		if markErr := model.MarkSubscriptionOrderStatus(orderID, "*", common.TopUpStatusExpired, "checkout session expired"); markErr != nil &&
			!errors.Is(markErr, model.ErrSubscriptionOrderNotFound) &&
			!errors.Is(markErr, model.ErrSubscriptionOrderStatusInvalid) {
			return markErr
		}
		return model.ReleaseSubscriptionCouponWithTx(tx, orderID)
	})
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription checkout expired release failed order=%s err=%q", orderID, err.Error()))
		return err
	}
	logger.LogInfo(ctx, fmt.Sprintf("subscription checkout expired order=%s coupon released", orderID))
	return nil
}

// buildSubscriptionProviderPayload serializes the key Stripe event fields into
// a JSON string for persistence on the order's ProviderPayload column. Uses
// common.Marshal per the project JSON convention (no direct encoding/json).
func buildSubscriptionProviderPayload(event stripe.Event) string {
	payload := map[string]any{
		"event_id":   event.ID,
		"event_type": string(event.Type),
		"livemode":   event.Livemode,
		"customer":   event.GetObjectValue("customer"),
	}
	if sub := event.GetObjectValue("subscription"); sub != "" {
		payload["subscription"] = sub
	}
	if pi := event.GetObjectValue("payment_intent"); pi != "" {
		payload["payment_intent"] = pi
	}
	if amt := event.GetObjectValue("amount_total"); amt != "" {
		payload["amount_total"] = amt
	}
	if curr := event.GetObjectValue("currency"); curr != "" {
		payload["currency"] = strings.ToUpper(curr)
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(raw)
}

// handleSubscriptionChargeRefunded reverses affiliate commission when a Stripe
// charge tied to a NovaPura subscription is refunded. It is invoked from
// StripeWebhookV2 AFTER the topup path's own handleChargeRefunded runs (the
// topup path handles non-subscription refunds via stripetopup.ReconcileRefund;
// this handler handles subscription commission clawback). Both can run on the
// same event safely — each only acts if its own commission/order row exists.
//
// Refund → commission mapping:
//  1. If the charge carries an `invoice` field, the refund is for a renewal
//     invoice. The renewal commission's topUpId is the invoice ID, so we
//     revert by `topup_id = invoiceId`.
//  2. Otherwise, fall back to the `novapura_order_id` metadata (set on the
//     checkout session and propagated to the charge by Stripe). The purchase
//     commission's topUpId is the order TradeNo, so we revert by that.
//
// Defensive: if no commission exists for the resolved topUpId, the function
// logs and returns nil (not all charges have commission — e.g. the user had
// no inviter, or the commission was already reverted). Idempotent via
// RevertCommission's already-reverted check.
func handleSubscriptionChargeRefunded(ctx context.Context, event stripe.Event) error {
	invoiceID := event.GetObjectValue("invoice")
	topUpId := invoiceID
	if topUpId == "" {
		// Not an invoice refund: try the order TradeNo via checkout metadata.
		// This covers prepaid (payment-mode) refunds where the charge carries
		// the novapura_order_id metadata from the checkout session.
		topUpId = event.GetObjectValue("metadata", "novapura_order_id")
	}
	if topUpId == "" {
		// No invoice and no order metadata — not a subscription commission
		// refund we can map. The topup path has already handled non-sub
		// refunds; nothing to do.
		return nil
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		revertErr := model.RevertCommission(tx, topUpId, "charge refunded")
		if errors.Is(revertErr, model.ErrCommissionNotFound) {
			// No commission was ever credited for this payment (e.g. the user
			// had no inviter, or it was a non-subscription charge). Not an
			// error — return nil so the transaction commits.
			return nil
		}
		return revertErr
	})
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("subscription charge.refunded commission revert failed topup=%s err=%q", topUpId, err.Error()))
		return nil // do not propagate — the topup path already processed the refund
	}
	logger.LogInfo(ctx, fmt.Sprintf("subscription charge.refunded commission reverted topup=%s", topUpId))
	return nil
}
