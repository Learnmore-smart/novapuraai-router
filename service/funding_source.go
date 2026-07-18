package service

import (
	"time"

	"github.com/QuantumNous/new-api/model"
)

// ---------------------------------------------------------------------------
// FundingSource — 资金来源接口（钱包 or 订阅）
// ---------------------------------------------------------------------------

// FundingSource 抽象了预扣费的资金来源。
type FundingSource interface {
	// Source 返回资金来源标识："wallet" 或 "subscription"
	Source() string
	// PreConsume 从该资金来源预扣 amount 额度
	PreConsume(amount int) error
	// Settle 根据差额调整资金来源（正数补扣，负数退还）
	Settle(delta int) error
	// Refund 退还所有预扣费
	Refund() error
}

// ---------------------------------------------------------------------------
// WalletFunding — 钱包资金来源实现（promo 先于 cash）
// ---------------------------------------------------------------------------

type WalletFunding struct {
	userId   int
	consumed int // 实际预扣的用户额度
	// split tracks promo vs cash for accurate refund (MVP dual balance).
	split model.QuotaWalletSplit
}

func (w *WalletFunding) Source() string { return BillingSourceWallet }

func (w *WalletFunding) PreConsume(amount int) error {
	if amount <= 0 {
		return nil
	}
	split, err := model.DecreaseUserQuotaWithSplit(w.userId, amount, true)
	if err != nil {
		return err
	}
	w.consumed = amount
	w.split = split
	return nil
}

func (w *WalletFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		// Additional deduct also promo-first; accumulate split for refund path.
		split, err := model.DecreaseUserQuotaWithSplit(w.userId, delta, true)
		if err != nil {
			return err
		}
		w.consumed += delta
		w.split.Promo += split.Promo
		w.split.Cash += split.Cash
		w.split.Allocations = append(w.split.Allocations, split.Allocations...)
		return nil
	}
	// Refund delta (negative): restore cash first then promo proportionally to last split is hard;
	// restore as cash for under-settle simplicity when refunding settle surplus.
	refund := -delta
	if refund > w.consumed {
		refund = w.consumed
	}
	refundSplit, remainingSplit := takeWalletRefund(w.split, refund)
	if err := model.RestoreUserQuotaSplit(w.userId, refundSplit); err != nil {
		return err
	}
	w.consumed -= refund
	w.split = remainingSplit
	return nil
}

func takeWalletRefund(split model.QuotaWalletSplit, amount int) (model.QuotaWalletSplit, model.QuotaWalletSplit) {
	refund := model.QuotaWalletSplit{}
	if amount <= 0 {
		return refund, split
	}
	split.Allocations = append([]model.BalanceLotAllocation(nil), split.Allocations...)
	remaining := amount
	for _, balanceType := range []string{model.BalanceTypePaid, model.BalanceTypePromotional} {
		for i := len(split.Allocations) - 1; i >= 0 && remaining > 0; i-- {
			allocation := &split.Allocations[i]
			if allocation.BalanceType != balanceType || allocation.Amount <= 0 {
				continue
			}
			take := allocation.Amount
			if take > remaining {
				take = remaining
			}
			refundedAllocation := *allocation
			refundedAllocation.Amount = take
			refund.Allocations = append(refund.Allocations, refundedAllocation)
			allocation.Amount -= take
			remaining -= take
			if balanceType == model.BalanceTypePaid {
				refund.Cash += take
				split.Cash -= take
			} else {
				refund.Promo += take
				split.Promo -= take
			}
		}
	}
	compacted := split.Allocations[:0]
	for _, allocation := range split.Allocations {
		if allocation.Amount > 0 {
			compacted = append(compacted, allocation)
		}
	}
	split.Allocations = compacted
	return refund, split
}

func (w *WalletFunding) Refund() error {
	if w.consumed <= 0 {
		return nil
	}
	// Restore exact promo/cash split — not plain IncreaseUserQuota (would all become cash).
	err := model.RestoreUserQuotaSplit(w.userId, w.split)
	if err != nil {
		return err
	}
	w.consumed = 0
	w.split = model.QuotaWalletSplit{}
	return nil
}

// ---------------------------------------------------------------------------
// SubscriptionFunding — 订阅资金来源实现
// ---------------------------------------------------------------------------

type SubscriptionFunding struct {
	requestId      string
	userId         int
	modelName      string
	amount         int64 // 预扣的订阅额度（subConsume）
	subscriptionId int
	preConsumed    int64
	// 以下字段在 PreConsume 成功后填充，供 RelayInfo 同步使用
	AmountTotal     int64
	AmountUsedAfter int64
	PlanId          int
	PlanTitle       string
}

func (s *SubscriptionFunding) Source() string { return BillingSourceSubscription }

func (s *SubscriptionFunding) PreConsume(_ int) error {
	// amount 参数被忽略，使用内部 s.amount（已在构造时根据 preConsumedQuota 计算）
	res, err := model.PreConsumeUserSubscription(s.requestId, s.userId, s.modelName, 0, s.amount)
	if err != nil {
		return err
	}
	s.subscriptionId = res.UserSubscriptionId
	s.preConsumed = res.PreConsumed
	s.AmountTotal = res.AmountTotal
	s.AmountUsedAfter = res.AmountUsedAfter
	// 获取订阅计划信息
	if planInfo, err := model.GetSubscriptionPlanInfoByUserSubscriptionId(res.UserSubscriptionId); err == nil && planInfo != nil {
		s.PlanId = planInfo.PlanId
		s.PlanTitle = planInfo.PlanTitle
	}
	return nil
}

func (s *SubscriptionFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	return model.PostConsumeUserSubscriptionDelta(s.subscriptionId, int64(delta))
}

func (s *SubscriptionFunding) Refund() error {
	if s.preConsumed <= 0 {
		return nil
	}
	return refundWithRetry(func() error {
		return model.RefundSubscriptionPreConsume(s.requestId)
	})
}

// refundWithRetry 尝试多次执行退款操作以提高成功率，只能用于基于事务的退款函数！！！！！！
func refundWithRetry(fn func() error) error {
	var lastErr error
	for i := 0; i < 3; i++ {
		if err := fn(); err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * 50 * time.Millisecond)
			continue
		}
		return nil
	}
	return lastErr
}
