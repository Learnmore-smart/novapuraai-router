package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/subscription"
	"gorm.io/gorm"
)

const (
	subscriptionSyncTickInterval = 5 * time.Minute
	subscriptionSyncBatchSize    = 50
	// subscriptionSyncStaleSeconds skips subscriptions whose UpdatedAt falls
	// within this window — a webhook already handled them recently.
	subscriptionSyncStaleSeconds int64 = 5 * 60
)

var (
	subscriptionSyncOnce    sync.Once
	subscriptionSyncRunning atomic.Bool
)

// StartSubscriptionSyncTask periodically reconciles subscription status from
// Stripe for subscriptions that may have changed state without a webhook
// reaching us (defensive sync). Runs on the master node only.
func StartSubscriptionSyncTask() {
	subscriptionSyncOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("subscription sync task started: tick=%s", subscriptionSyncTickInterval))
			runSubscriptionSyncOnce()
			ticker := time.NewTicker(subscriptionSyncTickInterval)
			defer ticker.Stop()
			for range ticker.C {
				runSubscriptionSyncOnce()
			}
		})
	})
}

func runSubscriptionSyncOnce() {
	if !subscriptionSyncRunning.CompareAndSwap(false, true) {
		return
	}
	defer subscriptionSyncRunning.Store(false)

	ctx := context.Background()

	// Sync only auto-renew subscriptions (prepaid don't have a Stripe
	// Subscription to sync). Skip recently-updated rows that webhooks already
	// handled.
	staleCutoff := common.GetTimestamp() - subscriptionSyncStaleSeconds
	var subs []model.UserSubscription
	if err := model.DB.Where("stripe_subscription_id <> '' AND status IN ? AND updated_at < ?",
		[]string{model.SubscriptionStatusActive, model.SubscriptionStatusCanceling, model.SubscriptionStatusPastDue},
		staleCutoff).
		Order("updated_at asc, id asc").
		Limit(subscriptionSyncBatchSize).
		Find(&subs).Error; err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("subscription sync task query failed: %v", err))
		return
	}
	if len(subs) == 0 {
		return
	}

	// Skip the Stripe API entirely when not configured (avoids auth errors).
	if !strings.HasPrefix(setting.StripeApiSecret, "sk_") && !strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		return
	}

	synced := 0
	for i := range subs {
		if syncSingleSubscriptionFromStripe(ctx, &subs[i]) {
			synced++
		}
	}
	if synced > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("subscription sync task reconciled %d/%d subscriptions", synced, len(subs)))
	}
}

// syncSingleSubscriptionFromStripe retrieves the subscription from Stripe and
// reconciles the local status/EndTime. Returns true when a reconciliation
// action was taken. Errors are logged and skipped so one subscription's
// failure never crashes the batch.
func syncSingleSubscriptionFromStripe(ctx context.Context, sub *model.UserSubscription) bool {
	stripe.Key = setting.StripeApiSecret
	params := &stripe.SubscriptionParams{}
	params.AddExpand("latest_invoice")
	stripeSub, err := subscription.Get(sub.StripeSubscriptionId, params)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("subscription sync stripe get failed sub_id=%d stripe_sub=%s err=%v", sub.Id, sub.StripeSubscriptionId, err))
		return false
	}

	stripeStatus := string(stripeSub.Status)
	cancelAtPeriodEnd := stripeSub.CancelAtPeriodEnd
	periodEnd := int64(0)
	if stripeSub.LatestInvoice != nil {
		periodEnd = stripeSub.LatestInvoice.PeriodEnd
	}

	acted := false
	switch {
	case stripeStatus == "canceled" || stripeStatus == "unpaid":
		if sub.Status != model.SubscriptionStatusCanceled {
			// CancelUserSubscriptionFromStripe handles status + group downgrade
			// + coupon reversal, and is idempotent — the same path the
			// customer.subscription.deleted webhook uses. This is equivalent to
			// TransitionSubscriptionStatus("canceled") + group downgrade, plus
			// coupon reversal which the sync should also trigger.
			if err := model.CancelUserSubscriptionFromStripe(sub.StripeSubscriptionId); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("subscription sync cancel failed sub_id=%d stripe_sub=%s err=%v", sub.Id, sub.StripeSubscriptionId, err))
			} else {
				acted = true
			}
		}
	case stripeStatus == "past_due":
		if sub.Status != model.SubscriptionStatusPastDue {
			if err := applySubscriptionStatusTransition(sub.Id, model.SubscriptionStatusPastDue); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("subscription sync past_due failed sub_id=%d err=%v", sub.Id, err))
			} else {
				acted = true
			}
		}
	case stripeStatus == "active" && cancelAtPeriodEnd:
		if sub.Status != model.SubscriptionStatusCanceling {
			if err := applySubscriptionStatusTransition(sub.Id, model.SubscriptionStatusCanceling); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("subscription sync canceling failed sub_id=%d err=%v", sub.Id, err))
			} else {
				acted = true
			}
		}
	case stripeStatus == "active" && !cancelAtPeriodEnd:
		if sub.Status != model.SubscriptionStatusActive {
			if err := applySubscriptionStatusTransition(sub.Id, model.SubscriptionStatusActive); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("subscription sync active failed sub_id=%d err=%v", sub.Id, err))
			} else {
				acted = true
			}
		}
	}

	// Sync EndTime to Stripe's current period end when the local value lags.
	if periodEnd > 0 && sub.EndTime < periodEnd {
		if err := syncSubscriptionEndTime(sub.Id, periodEnd); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("subscription sync end_time failed sub_id=%d err=%v", sub.Id, err))
		} else {
			acted = true
		}
	}
	return acted
}

func applySubscriptionStatusTransition(subId int, newStatus string) error {
	var userId int
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.TransitionSubscriptionStatus(tx, subId, newStatus); err != nil {
			return err
		}
		// Capture userId inside the tx so we can invalidate the coverage cache
		// after commit. TransitionSubscriptionStatus already loaded the row
		// with lockForUpdate, but does not surface the userId; re-read it.
		var s model.UserSubscription
		if qErr := tx.Where("id = ?", subId).First(&s).Error; qErr != nil {
			return qErr
		}
		userId = s.UserId
		return nil
	})
	if err != nil {
		return err
	}
	if userId > 0 {
		model.InvalidateUserSubscriptionCoverageCache(userId)
	}
	return nil
}

func syncSubscriptionEndTime(subId int, periodEnd int64) error {
	var userId int
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var sub model.UserSubscription
		if err := model.LockForUpdate(tx).Where("id = ?", subId).First(&sub).Error; err != nil {
			return err
		}
		userId = sub.UserId
		if sub.EndTime >= periodEnd {
			return nil
		}
		now := common.GetTimestamp()
		return tx.Model(&sub).Updates(map[string]interface{}{
			"end_time":   periodEnd,
			"updated_at": now,
		}).Error
	})
	if err != nil {
		return err
	}
	if userId > 0 {
		model.InvalidateUserSubscriptionCoverageCache(userId)
	}
	return nil
}
