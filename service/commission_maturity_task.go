package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const commissionMaturityTickInterval = 10 * time.Minute

var (
	commissionMaturityOnce    sync.Once
	commissionMaturityRunning atomic.Bool
)

// StartCommissionMaturityTask releases frozen commissions whose
// CommissionFreezeDays hold has elapsed, moving them from PendingCommissionCents
// to CommissionBalanceCents (withdrawable). Runs on the master node only.
func StartCommissionMaturityTask() {
	commissionMaturityOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("commission maturity task started: tick=%s", commissionMaturityTickInterval))
			ticker := time.NewTicker(commissionMaturityTickInterval)
			defer ticker.Stop()

			runCommissionMaturityOnce()
			for range ticker.C {
				runCommissionMaturityOnce()
			}
		})
	})
}

func runCommissionMaturityOnce() {
	if !commissionMaturityRunning.CompareAndSwap(false, true) {
		return
	}
	defer commissionMaturityRunning.Store(false)

	ctx := context.Background()
	released, err := model.ReleaseMaturedCommissions()
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("commission maturity task failed: released=%d err=%v", released, err))
		return
	}
	if released > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("commission maturity task released %d commissions", released))
	}
}
