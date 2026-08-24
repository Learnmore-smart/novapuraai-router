package operation_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
)

func TestCheckinRewardDefaultsCoverZeroToFiveCNY(t *testing.T) {
	assert.Equal(t, 0, checkinSetting.MinQuota)
	assert.Equal(t, common.CNYYuanToQuota(5, common.DefaultUSDExchangeRate), checkinSetting.MaxQuota)
}

func TestCheckinRewardRangeCapsPersistedLegacyValuesAtFiveCNY(t *testing.T) {
	original := checkinSetting
	originalExchangeRate := USDExchangeRate
	USDExchangeRate = common.DefaultUSDExchangeRate
	checkinSetting.MinQuota = common.CNYYuanToQuota(5, common.DefaultUSDExchangeRate)
	checkinSetting.MaxQuota = common.CNYYuanToQuota(50, common.DefaultUSDExchangeRate)
	t.Cleanup(func() {
		checkinSetting = original
		USDExchangeRate = originalExchangeRate
	})

	minQuota, maxQuota := GetCheckinQuotaRange()
	assert.Equal(t, 0, minQuota)
	assert.Equal(t, common.CNYYuanToQuota(5, common.DefaultUSDExchangeRate), maxQuota)
}
