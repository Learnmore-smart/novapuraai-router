package operation_setting

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

// CheckinSetting 签到功能配置
type CheckinSetting struct {
	Enabled  bool `json:"enabled"`   // 是否启用签到功能
	MinQuota int  `json:"min_quota"` // 签到最小额度奖励
	MaxQuota int  `json:"max_quota"` // 签到最大额度奖励
}

// 默认配置
var checkinSetting = CheckinSetting{
	Enabled:  false,                                                   // 默认关闭
	MinQuota: 0,                                                       // 默认最小奖励 0 CNY
	MaxQuota: common.CNYYuanToQuota(5, common.DefaultUSDExchangeRate), // 默认最大奖励 5 CNY
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("checkin_setting", &checkinSetting)
}

// GetCheckinSetting 获取签到配置
func GetCheckinSetting() *CheckinSetting {
	effective := checkinSetting
	effective.MinQuota, effective.MaxQuota = GetCheckinQuotaRange()
	return &effective
}

// IsCheckinEnabled 是否启用签到功能
func IsCheckinEnabled() bool {
	return checkinSetting.Enabled
}

// GetCheckinQuotaRange 获取签到额度范围
func GetCheckinQuotaRange() (min, max int) {
	max = checkinSetting.MaxQuota
	maxAllowed := common.CNYYuanToQuota(5, USDExchangeRate)
	if max < 0 {
		max = 0
	} else if max > maxAllowed {
		max = maxAllowed
	}
	return 0, max
}
