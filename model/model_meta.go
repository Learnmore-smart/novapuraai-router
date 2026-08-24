package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	NameRuleExact = iota
	NameRulePrefix
	NameRuleContains
	NameRuleSuffix
)

type BoundChannel struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

type Model struct {
	Id           int            `json:"id"`
	ModelName    string         `json:"model_name" gorm:"size:128;not null;uniqueIndex:uk_model_name_delete_at,priority:1"`
	Description  string         `json:"description,omitempty" gorm:"type:text"`
	Icon         string         `json:"icon,omitempty" gorm:"type:varchar(128)"`
	Tags         string         `json:"tags,omitempty" gorm:"type:varchar(255)"`
	VendorID     int            `json:"vendor_id,omitempty" gorm:"index"`
	Endpoints    string         `json:"endpoints,omitempty" gorm:"type:text"`
	Status       int            `json:"status" gorm:"default:1"`
	SyncOfficial int            `json:"sync_official" gorm:"default:1"`
	CreatedTime  int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime  int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index;uniqueIndex:uk_model_name_delete_at,priority:2"`

	BoundChannels []BoundChannel `json:"bound_channels,omitempty" gorm:"-"`
	EnableGroups  []string       `json:"enable_groups,omitempty" gorm:"-"`
	QuotaTypes    []int          `json:"quota_types,omitempty" gorm:"-"`
	NameRule      int            `json:"name_rule" gorm:"default:0"`

	MatchedModels []string `json:"matched_models,omitempty" gorm:"-"`
	MatchedCount  int      `json:"matched_count,omitempty" gorm:"-"`
}

// RepairCanonicalDeepSeekModelMetadata repairs the one known malformed
// DeepSeek identity in model metadata and channel capabilities. The repair is
// deliberately exact and transactional: if any storage area already contains
// both spellings, no row is changed.
func RepairCanonicalDeepSeekModelMetadata() error {
	return DB.Transaction(repairCanonicalDeepSeekModelMetadata)
}

func repairCanonicalDeepSeekModelMetadata(tx *gorm.DB) error {
	const malformedName = common.CanonicalDeepSeekV4Flash0731 + `"`
	canonicalName := common.CanonicalDeepSeekV4Flash0731

	type channelRepair struct {
		id      int
		updates map[string]any
	}
	channelRepairs := make([]channelRepair, 0)
	if tx.Migrator().HasTable(&Channel{}) {
		var channels []Channel
		likeMalformed := "%" + malformedName + "%"
		if err := lockForUpdate(tx).
			Where("models LIKE ? OR test_model = ? OR model_mapping LIKE ? OR settings LIKE ?", likeMalformed, malformedName, likeMalformed, likeMalformed).
			Find(&channels).Error; err != nil {
			return err
		}
		channelRepairs = make([]channelRepair, 0, len(channels))
		for _, channel := range channels {
			updates := make(map[string]any)
			repairedModels, changed, err := repairCanonicalDeepSeekChannelModels(channel.Models, canonicalName, malformedName)
			if err != nil {
				return fmt.Errorf("channel %d: %w", channel.Id, err)
			}
			if changed {
				updates["models"] = repairedModels
			}
			if channel.TestModel != nil && strings.TrimSpace(*channel.TestModel) == malformedName {
				updates["test_model"] = canonicalName
			}
			if channel.ModelMapping != nil && strings.Contains(*channel.ModelMapping, malformedName) {
				repairedMapping, mappingChanged, err := repairCanonicalDeepSeekModelMapping(*channel.ModelMapping, canonicalName, malformedName)
				if err != nil {
					return fmt.Errorf("channel %d model mapping: %w", channel.Id, err)
				}
				if mappingChanged {
					updates["model_mapping"] = repairedMapping
				}
			}
			if strings.Contains(channel.OtherSettings, malformedName) {
				repairedSettings, settingsChanged, err := repairCanonicalDeepSeekChannelSettings(channel.OtherSettings, canonicalName, malformedName)
				if err != nil {
					return fmt.Errorf("channel %d settings: %w", channel.Id, err)
				}
				if settingsChanged {
					updates["settings"] = repairedSettings
				}
			}
			if len(updates) > 0 {
				channelRepairs = append(channelRepairs, channelRepair{id: channel.Id, updates: updates})
			}
		}
	}

	var models []Model
	if tx.Migrator().HasTable(&Model{}) {
		if err := lockForUpdate(tx).Where("model_name IN ?", []string{canonicalName, malformedName}).Find(&models).Error; err != nil {
			return err
		}
	}
	canonicalModels := 0
	malformedModels := make([]Model, 0, 1)
	for _, item := range models {
		if item.ModelName == canonicalName {
			canonicalModels++
		} else if item.ModelName == malformedName {
			malformedModels = append(malformedModels, item)
		}
	}
	if canonicalModels > 0 && len(malformedModels) > 0 {
		return fmt.Errorf("conflicting model metadata identities for %q and %q", malformedName, canonicalName)
	}
	if len(malformedModels) > 1 {
		return fmt.Errorf("conflicting duplicate model metadata identities for %q", malformedName)
	}

	var abilities []Ability
	if tx.Migrator().HasTable(&Ability{}) {
		if err := lockForUpdate(tx).Where("model IN ?", []string{canonicalName, malformedName}).Find(&abilities).Error; err != nil {
			return err
		}
	}
	canonicalAbilities := 0
	malformedAbilities := 0
	for _, item := range abilities {
		if item.Model == canonicalName {
			canonicalAbilities++
		} else if item.Model == malformedName {
			malformedAbilities++
		}
	}
	if canonicalAbilities > 0 && malformedAbilities > 0 {
		return fmt.Errorf("conflicting channel capability identities for %q and %q", malformedName, canonicalName)
	}

	if len(malformedModels) == 1 {
		if err := tx.Model(&Model{}).
			Where("id = ?", malformedModels[0].Id).
			Update("model_name", canonicalName).Error; err != nil {
			return err
		}
	}
	if malformedAbilities > 0 {
		if err := tx.Model(&Ability{}).
			Where("model = ?", malformedName).
			Update("model", canonicalName).Error; err != nil {
			return err
		}
	}
	for _, repair := range channelRepairs {
		if err := tx.Model(&Channel{}).
			Where("id = ?", repair.id).
			Updates(repair.updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func repairCanonicalDeepSeekChannelModels(models, canonicalName, malformedName string) (string, bool, error) {
	parts := strings.Split(models, ",")
	hasCanonical := false
	hasMalformed := false
	for _, part := range parts {
		switch strings.TrimSpace(part) {
		case canonicalName:
			hasCanonical = true
		case malformedName:
			hasMalformed = true
		}
	}
	if hasCanonical && hasMalformed {
		return models, false, fmt.Errorf("conflicting channel model identities for %q and %q", malformedName, canonicalName)
	}
	if !hasMalformed {
		return models, false, nil
	}
	for index, part := range parts {
		if strings.TrimSpace(part) == malformedName {
			parts[index] = canonicalName
		}
	}
	return strings.Join(parts, ","), true, nil
}

func repairCanonicalDeepSeekModelMapping(mapping, canonicalName, malformedName string) (string, bool, error) {
	values := make(map[string]string)
	if err := common.Unmarshal([]byte(mapping), &values); err != nil {
		return mapping, false, err
	}
	changed := false
	if malformedTarget, malformedFound := values[malformedName]; malformedFound {
		if canonicalTarget, canonicalFound := values[canonicalName]; canonicalFound && canonicalTarget != malformedTarget {
			return mapping, false, fmt.Errorf("conflicting mapping values for %q and %q", malformedName, canonicalName)
		}
		if _, canonicalFound := values[canonicalName]; !canonicalFound {
			values[canonicalName] = malformedTarget
		}
		delete(values, malformedName)
		changed = true
	}
	for source, target := range values {
		if strings.TrimSpace(target) == malformedName {
			values[source] = canonicalName
			changed = true
		}
	}
	if !changed {
		return mapping, false, nil
	}
	encoded, err := common.Marshal(values)
	if err != nil {
		return mapping, false, err
	}
	return string(encoded), true, nil
}

func repairCanonicalDeepSeekChannelSettings(settings, canonicalName, malformedName string) (string, bool, error) {
	if strings.TrimSpace(settings) == "" {
		return settings, false, nil
	}
	var values map[string]json.RawMessage
	if err := common.Unmarshal([]byte(settings), &values); err != nil {
		return settings, false, err
	}
	changed := false
	for _, key := range []string{
		"upstream_model_update_last_detected_models",
		"upstream_model_update_last_removed_models",
		"upstream_model_update_ignored_models",
	} {
		raw, ok := values[key]
		if !ok {
			continue
		}
		var models []string
		if err := common.Unmarshal(raw, &models); err != nil {
			return settings, false, err
		}
		repaired, listChanged, err := repairCanonicalDeepSeekModelList(models, canonicalName, malformedName)
		if err != nil {
			return settings, false, err
		}
		if listChanged {
			encoded, err := common.Marshal(repaired)
			if err != nil {
				return settings, false, err
			}
			values[key] = encoded
			changed = true
		}
	}
	if raw, ok := values["advanced_custom"]; ok {
		var advanced map[string]json.RawMessage
		if err := common.Unmarshal(raw, &advanced); err != nil {
			return settings, false, err
		}
		if routesRaw, ok := advanced["advanced_routes"]; ok {
			var routes []map[string]json.RawMessage
			if err := common.Unmarshal(routesRaw, &routes); err != nil {
				return settings, false, err
			}
			routesChanged := false
			for _, route := range routes {
				modelsRaw, ok := route["models"]
				if !ok {
					continue
				}
				var models []string
				if err := common.Unmarshal(modelsRaw, &models); err != nil {
					return settings, false, err
				}
				repaired, listChanged, err := repairCanonicalDeepSeekModelList(models, canonicalName, malformedName)
				if err != nil {
					return settings, false, err
				}
				if listChanged {
					encoded, err := common.Marshal(repaired)
					if err != nil {
						return settings, false, err
					}
					route["models"] = encoded
					routesChanged = true
				}
			}
			if routesChanged {
				advancedRoutes, err := common.Marshal(routes)
				if err != nil {
					return settings, false, err
				}
				advanced["advanced_routes"] = advancedRoutes
				advancedJSON, err := common.Marshal(advanced)
				if err != nil {
					return settings, false, err
				}
				values["advanced_custom"] = advancedJSON
				changed = true
			}
		}
	}
	if !changed {
		return settings, false, nil
	}
	encoded, err := common.Marshal(values)
	if err != nil {
		return settings, false, err
	}
	return string(encoded), true, nil
}

func repairCanonicalDeepSeekModelList(models []string, canonicalName, malformedName string) ([]string, bool, error) {
	hasCanonical := false
	hasMalformed := false
	for _, modelName := range models {
		switch strings.TrimSpace(modelName) {
		case canonicalName:
			hasCanonical = true
		case malformedName:
			hasMalformed = true
		}
	}
	if hasCanonical && hasMalformed {
		return models, false, fmt.Errorf("conflicting model identities for %q and %q", malformedName, canonicalName)
	}
	if !hasMalformed {
		return models, false, nil
	}
	result := append([]string(nil), models...)
	for index, modelName := range result {
		if strings.TrimSpace(modelName) == malformedName {
			result[index] = canonicalName
		}
	}
	return result, true, nil
}

func validateModelMetadataName(name string) error {
	if common.IsInvalidModelName(name) {
		return fmt.Errorf("invalid model name %q: trailing quote is not allowed", name)
	}
	return nil
}

func (mi *Model) Insert() error {
	if err := validateModelMetadataName(mi.ModelName); err != nil {
		return err
	}
	now := common.GetTimestamp()
	mi.CreatedTime = now
	mi.UpdatedTime = now

	// 保存原始值（因为 Create 后可能被 GORM 的 default 标签覆盖为 1）
	originalStatus := mi.Status
	originalSyncOfficial := mi.SyncOfficial

	// 先创建记录（GORM 会对零值字段应用默认值）
	if err := DB.Create(mi).Error; err != nil {
		return err
	}

	// 使用保存的原始值进行更新，确保零值能正确保存
	return DB.Model(&Model{}).Where("id = ?", mi.Id).Updates(map[string]interface{}{
		"status":        originalStatus,
		"sync_official": originalSyncOfficial,
	}).Error
}

func IsModelNameDuplicated(id int, name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	var cnt int64
	err := DB.Model(&Model{}).Where("model_name = ? AND id <> ?", name, id).Count(&cnt).Error
	return cnt > 0, err
}

func (mi *Model) Update() error {
	if err := validateModelMetadataName(mi.ModelName); err != nil {
		return err
	}
	mi.UpdatedTime = common.GetTimestamp()
	// 使用 Select 强制更新所有字段，包括零值
	return DB.Model(&Model{}).Where("id = ?", mi.Id).
		Select("model_name", "description", "icon", "tags", "vendor_id", "endpoints", "status", "sync_official", "name_rule", "updated_time").
		Updates(mi).Error
}

func (mi *Model) Delete() error {
	return DB.Delete(mi).Error
}

func GetVendorModelCounts() (map[int64]int64, error) {
	var stats []struct {
		VendorID int64
		Count    int64
	}
	if err := DB.Model(&Model{}).
		Select("vendor_id as vendor_id, count(*) as count").
		Group("vendor_id").
		Scan(&stats).Error; err != nil {
		return nil, err
	}
	m := make(map[int64]int64, len(stats))
	for _, s := range stats {
		m[s.VendorID] = s.Count
	}
	return m, nil
}

func GetAllModels(offset int, limit int) ([]*Model, error) {
	models, _, err := SearchModels("", "", "", "", offset, limit)
	return models, err
}

func GetBoundChannelsByModelsMap(modelNames []string) (map[string][]BoundChannel, error) {
	result := make(map[string][]BoundChannel)
	if len(modelNames) == 0 {
		return result, nil
	}
	type row struct {
		Model string
		Name  string
		Type  int
	}
	var rows []row
	err := DB.Table("channels").
		Select("abilities.model as model, channels.name as name, channels.type as type").
		Joins("JOIN abilities ON abilities.channel_id = channels.id").
		Where("abilities.model IN ? AND abilities.enabled = ?", modelNames, true).
		Distinct().
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		result[r.Model] = append(result[r.Model], BoundChannel{Name: r.Name, Type: r.Type})
	}
	return result, nil
}

func normalizeLookupValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func GetPreferredModelOwnerChannelTypes(modelNames []string, groups []string) (map[string]int, error) {
	result := make(map[string]int)
	modelNames = normalizeLookupValues(modelNames)
	if len(modelNames) == 0 {
		return result, nil
	}

	type row struct {
		Model       string
		ChannelType int
	}
	var rows []row

	query := DB.Table("abilities").
		Select("abilities.model as model, channels.type as channel_type").
		Joins("JOIN channels ON abilities.channel_id = channels.id").
		Where("abilities.model IN ? AND abilities.enabled = ? AND channels.status = ?", modelNames, true, common.ChannelStatusEnabled).
		Order("COALESCE(abilities.priority, 0) DESC").
		Order("abilities.weight DESC").
		Order("abilities.channel_id ASC")

	groups = normalizeLookupValues(groups)
	if len(groups) > 0 {
		query = query.Where("abilities."+commonGroupCol+" IN ?", groups)
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, r := range rows {
		if _, ok := result[r.Model]; ok {
			continue
		}
		result[r.Model] = r.ChannelType
	}
	return result, nil
}

func SearchModels(keyword string, vendor string, status string, syncOfficial string, offset int, limit int) ([]*Model, int64, error) {
	var models []*Model
	db := DB.Model(&Model{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("model_name LIKE ? OR description LIKE ? OR tags LIKE ?", like, like, like)
	}
	if vendor != "" {
		if vid, err := strconv.Atoi(vendor); err == nil {
			db = db.Where("models.vendor_id = ?", vid)
		} else {
			db = db.Joins("JOIN vendors ON vendors.id = models.vendor_id").Where("vendors.name LIKE ?", "%"+vendor+"%")
		}
	}
	if statusValue, ok := parseModelStatusFilter(status); ok {
		db = db.Where("models.status = ?", statusValue)
	}
	if syncValue, ok := parseModelSyncFilter(syncOfficial); ok {
		db = db.Where("models.sync_official = ?", syncValue)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("models.id DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	return models, total, nil
}

// parseModelStatusFilter maps UI/API status values to the models.status column.
// Returns ok=false when no status filter should be applied.
func parseModelStatusFilter(status string) (value int, ok bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "all":
		return 0, false
	case "enabled", "1":
		return 1, true
	case "disabled", "0":
		return 0, true
	default:
		n, err := strconv.Atoi(status)
		if err != nil {
			return 0, false
		}
		return n, true
	}
}

// parseModelSyncFilter maps UI/API sync values to the models.sync_official column.
// Returns ok=false when no sync filter should be applied.
func parseModelSyncFilter(syncOfficial string) (value int, ok bool) {
	switch strings.ToLower(strings.TrimSpace(syncOfficial)) {
	case "", "all":
		return 0, false
	case "yes", "1":
		return 1, true
	case "no", "0":
		return 0, true
	default:
		n, err := strconv.Atoi(syncOfficial)
		if err != nil {
			return 0, false
		}
		return n, true
	}
}
