package service

import (
	"context"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fairUseBillingFunding struct {
	settleDeltas []int
	refunded     chan struct{}
}

func (f *fairUseBillingFunding) Source() string { return BillingSourceWallet }

func (f *fairUseBillingFunding) PreConsume(int) error { return nil }

func (f *fairUseBillingFunding) Settle(delta int) error {
	f.settleDeltas = append(f.settleDeltas, delta)
	return nil
}

func (f *fairUseBillingFunding) Refund() error {
	f.refunded <- struct{}{}
	return nil
}

func newFairUseBillingSession(info *relaycommon.RelayInfo, funding *fairUseBillingFunding) *BillingSession {
	return &BillingSession{
		relayInfo:        info,
		funding:          funding,
		preConsumedQuota: 10,
		tokenConsumed:    10,
	}
}

func TestFairUseUsesBillingSessionSettlementAndRefundLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settleContext, _ := gin.CreateTestContext(nil)
	settleInfo := &relaycommon.RelayInfo{
		IsPlayground: true,
		UserId:       901,
		UserQuota:    1_000_000_000,
	}
	settleFunding := &fairUseBillingFunding{}
	settleSession := newFairUseBillingSession(settleInfo, settleFunding)
	settleInfo.Billing = settleSession

	require.NoError(t, SettleBilling(settleContext, settleInfo, 12))
	assert.Equal(t, []int{2}, settleFunding.settleDeltas)
	assert.False(t, settleSession.NeedsRefund())
	assert.Equal(t, int(10), settleSession.GetPreConsumedQuota())

	// A deferred completion path can revisit settlement safely, but it must not
	// charge the same delta twice.
	require.NoError(t, SettleBilling(settleContext, settleInfo, 12))
	assert.Equal(t, []int{2}, settleFunding.settleDeltas)

	refundContext, _ := gin.CreateTestContext(nil)
	refundInfo := &relaycommon.RelayInfo{IsPlayground: true, UserId: 902}
	refundFunding := &fairUseBillingFunding{refunded: make(chan struct{}, 1)}
	refundSession := newFairUseBillingSession(refundInfo, refundFunding)
	refundInfo.Billing = refundSession

	refundSession.Refund(refundContext)
	assert.False(t, refundSession.NeedsRefund(), "refund must become terminal before the async funding call")
	refundWaitContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	select {
	case <-refundFunding.refunded:
	case <-refundWaitContext.Done():
		t.Fatal("billing funding refund was not dispatched")
	}
	refundSession.Refund(refundContext)
	select {
	case <-refundFunding.refunded:
		assert.Fail(t, "duplicate refund was dispatched")
	default:
	}
}
