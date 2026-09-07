package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

// GetMissingModels returns model names that are referenced in the system
func GetMissingModels() ([]string, error) {
	// 1. 获取所有已启用模型（去重）
	models := GetEnabledModels()
	if len(models) == 0 {
		return []string{}, nil
	}

	// 2. 查询已有的元数据模型名
	var existing []string
	if err := DB.Model(&Model{}).Where("model_name IN ?", models).Pluck("model_name", &existing).Error; err != nil {
		return nil, err
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		existingSet[e] = struct{}{}
	}

	// 3. 收集缺失模型
	var missing []string
	for _, name := range models {
		if _, ok := existingSet[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing, nil
}

// SyncChannelModelMetadata inserts models-table rows for enabled channel
// abilities that do not already have metadata. It does not import adaptor
// ModelList or official upstream catalogs.
func SyncChannelModelMetadata() {
	if DB == nil {
		return
	}
	missing, err := GetMissingModels()
	if err != nil {
		common.SysError("SyncChannelModelMetadata: " + err.Error())
		return
	}
	if len(missing) == 0 {
		return
	}

	var vendors []Vendor
	if err := DB.Find(&vendors).Error; err != nil {
		common.SysError("SyncChannelModelMetadata load vendors: " + err.Error())
		return
	}
	vendorMap := make(map[int]*Vendor, len(vendors))
	for i := range vendors {
		vendorMap[vendors[i].Id] = &vendors[i]
	}

	created := 0
	for _, name := range missing {
		item := &Model{
			ModelName:    name,
			Status:       1,
			SyncOfficial: 0,
			NameRule:     NameRuleExact,
		}
		if vendorName, ok := matchDefaultVendor(name); ok {
			item.VendorID = getOrCreateVendor(vendorName, vendorMap)
		}
		if err := item.Insert(); err != nil {
			common.SysError(fmt.Sprintf("SyncChannelModelMetadata insert %q: %v", name, err))
			continue
		}
		created++
	}
	if created > 0 {
		common.SysLog(fmt.Sprintf("SyncChannelModelMetadata: created %d model metadata row(s) from enabled channels", created))
	}
}
