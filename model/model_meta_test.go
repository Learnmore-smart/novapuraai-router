package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCanonicalMetadataTest(t *testing.T, modelNames ...string) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Model{}, &Channel{}, &Ability{}))
	cleanup := func() {
		DB.Where("model IN ?", modelNames).Delete(&Ability{})
		DB.Where("id IN ?", []int{9101, 9102, 9103, 9104}).Delete(&Channel{})
		DB.Unscoped().Where("model_name IN ?", modelNames).Delete(&Model{})
	}
	cleanup()
	t.Cleanup(cleanup)
}

func TestRepairCanonicalDeepSeekModelMetadataRepairsModelAndAbility(t *testing.T) {
	canonical := common.CanonicalDeepSeekV4Flash0731
	malformed := canonical + `"`
	setupCanonicalMetadataTest(t, canonical, malformed)

	metadata := &Model{ModelName: malformed, Description: "preserve me", Status: 1, SyncOfficial: 1}
	require.NoError(t, DB.Create(metadata).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "canonicalization-test",
		Model:     malformed,
		ChannelId: 9101,
		Enabled:   true,
	}).Error)

	require.NoError(t, RepairCanonicalDeepSeekModelMetadata())

	var repaired Model
	require.NoError(t, DB.Where("model_name = ?", canonical).First(&repaired).Error)
	assert.Equal(t, metadata.Id, repaired.Id)
	assert.Equal(t, "preserve me", repaired.Description)

	var repairedAbility Ability
	require.NoError(t, DB.Where("model = ? AND channel_id = ?", canonical, 9101).First(&repairedAbility).Error)
	assert.Equal(t, canonical, repairedAbility.Model)

	var malformedCount int64
	require.NoError(t, DB.Model(&Model{}).Where("model_name = ?", malformed).Count(&malformedCount).Error)
	assert.Zero(t, malformedCount)

	require.NoError(t, RepairCanonicalDeepSeekModelMetadata())
	var canonicalCount int64
	require.NoError(t, DB.Model(&Model{}).Where("model_name = ?", canonical).Count(&canonicalCount).Error)
	assert.EqualValues(t, 1, canonicalCount)
}

func TestRepairCanonicalDeepSeekModelMetadataFailsClosedOnConflict(t *testing.T) {
	canonical := common.CanonicalDeepSeekV4Flash0731
	malformed := canonical + `"`
	setupCanonicalMetadataTest(t, canonical, malformed)

	require.NoError(t, DB.Create(&Model{ModelName: canonical, Description: "canonical", Status: 1, SyncOfficial: 1}).Error)
	require.NoError(t, DB.Create(&Model{ModelName: malformed, Description: "malformed", Status: 1, SyncOfficial: 1}).Error)

	err := RepairCanonicalDeepSeekModelMetadata()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting")

	var canonicalModel, malformedModel Model
	require.NoError(t, DB.Where("model_name = ?", canonical).First(&canonicalModel).Error)
	require.NoError(t, DB.Where("model_name = ?", malformed).First(&malformedModel).Error)
	assert.Equal(t, "canonical", canonicalModel.Description)
	assert.Equal(t, "malformed", malformedModel.Description)
}

func TestModelMetadataWritesRejectTrailingQuoteNames(t *testing.T) {
	canonical := common.CanonicalDeepSeekV4Flash0731
	malformed := canonical + `"`
	setupCanonicalMetadataTest(t, canonical, malformed)

	assert.Error(t, (&Model{ModelName: malformed}).Insert())

	valid := &Model{ModelName: canonical, Status: 1, SyncOfficial: 1}
	require.NoError(t, valid.Insert())
	valid.ModelName = malformed
	assert.Error(t, valid.Update())

	var stored Model
	require.NoError(t, DB.Where("id = ?", valid.Id).First(&stored).Error)
	assert.Equal(t, canonical, stored.ModelName)
}

func TestRepairCanonicalDeepSeekModelMetadataRepairsChannelModelsBeforeAbilityRegeneration(t *testing.T) {
	canonical := common.CanonicalDeepSeekV4Flash0731
	malformed := canonical + `"`
	setupCanonicalMetadataTest(t, canonical, malformed)

	channel := &Channel{
		Id:     9102,
		Key:    "channel-key",
		Name:   "channel-model-repair",
		Models: "other-model," + malformed,
		Group:  "default",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     malformed,
		ChannelId: channel.Id,
		Enabled:   true,
	}).Error)

	require.NoError(t, RepairCanonicalDeepSeekModelMetadata())

	var repaired Channel
	require.NoError(t, DB.First(&repaired, "id = ?", channel.Id).Error)
	assert.Equal(t, "other-model,"+canonical, repaired.Models)

	var repairedAbility Ability
	require.NoError(t, DB.Where("channel_id = ? AND model = ?", channel.Id, canonical).First(&repairedAbility).Error)

	malformedCount := int64(0)
	require.NoError(t, DB.Model(&Ability{}).Where("channel_id = ? AND model = ?", channel.Id, malformed).Count(&malformedCount).Error)
	assert.Zero(t, malformedCount)

	// A later ability rebuild must read the repaired persisted source and must
	// not recreate the malformed capability.
	require.NoError(t, repaired.UpdateAbilities(nil))
	require.NoError(t, DB.Model(&Ability{}).Where("channel_id = ? AND model = ?", channel.Id, malformed).Count(&malformedCount).Error)
	assert.Zero(t, malformedCount)
}

func TestRepairCanonicalDeepSeekModelMetadataFailsClosedOnChannelModelConflict(t *testing.T) {
	canonical := common.CanonicalDeepSeekV4Flash0731
	malformed := canonical + `"`
	setupCanonicalMetadataTest(t, canonical, malformed)

	channel := &Channel{
		Id:     9103,
		Key:    "channel-key",
		Name:   "channel-model-conflict",
		Models: canonical + "," + malformed,
		Group:  "default",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, DB.Create(channel).Error)

	err := RepairCanonicalDeepSeekModelMetadata()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting")

	var unchanged Channel
	require.NoError(t, DB.First(&unchanged, "id = ?", channel.Id).Error)
	assert.Equal(t, canonical+","+malformed, unchanged.Models)
}

func TestRepairCanonicalDeepSeekModelMetadataRepairsChannelIdentityFields(t *testing.T) {
	canonical := common.CanonicalDeepSeekV4Flash0731
	malformed := canonical + `"`
	setupCanonicalMetadataTest(t, canonical, malformed)
	mappingBytes, err := common.Marshal(map[string]string{malformed: malformed})
	require.NoError(t, err)
	mapping := string(mappingBytes)
	testModel := malformed
	settingsBytes, err := common.Marshal(map[string]any{
		"upstream_model_update_last_detected_models": []string{malformed},
		"advanced_custom": map[string]any{
			"advanced_routes": []map[string]any{{
				"models":        []string{malformed},
				"upstream_path": "/v1/responses",
			}},
		},
		"preserved": map[string]any{"value": true},
	})
	require.NoError(t, err)
	settings := string(settingsBytes)
	channel := &Channel{
		Id:            9104,
		Key:           "channel-key",
		Name:          "channel-identity-repair",
		Models:        "other-model",
		Group:         "default",
		Status:        common.ChannelStatusEnabled,
		TestModel:     &testModel,
		ModelMapping:  &mapping,
		OtherSettings: settings,
	}
	require.NoError(t, DB.Create(channel).Error)

	require.NoError(t, RepairCanonicalDeepSeekModelMetadata())

	var repaired Channel
	require.NoError(t, DB.First(&repaired, "id = ?", channel.Id).Error)
	require.NotNil(t, repaired.TestModel)
	assert.Equal(t, canonical, *repaired.TestModel)
	require.NotNil(t, repaired.ModelMapping)
	assert.NotContains(t, *repaired.ModelMapping, malformed)
	assert.Contains(t, *repaired.ModelMapping, canonical)
	assert.NotContains(t, repaired.OtherSettings, malformed)
	assert.Contains(t, repaired.OtherSettings, `"preserved"`)
}
