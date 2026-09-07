package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChannelCatalogTest(t *testing.T, modelNames ...string) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Model{}, &Channel{}, &Ability{}, &Vendor{}))
	cleanup := func() {
		DB.Where("model IN ?", modelNames).Delete(&Ability{})
		DB.Where("id IN ?", []int{9201, 9202, 9203}).Delete(&Channel{})
		DB.Unscoped().Where("model_name IN ?", modelNames).Delete(&Model{})
	}
	cleanup()
	t.Cleanup(cleanup)
}

func TestSearchModelsListsOnlyEnabledChannelModels(t *testing.T) {
	hosted := "np-catalog-test-hosted"
	orphan := "np-catalog-test-orphan"
	setupChannelCatalogTest(t, hosted, orphan)

	require.NoError(t, DB.Create(&Model{ModelName: hosted, Status: 1, SyncOfficial: 0}).Error)
	require.NoError(t, DB.Create(&Model{ModelName: orphan, Status: 1, SyncOfficial: 0}).Error)
	require.NoError(t, DB.Create(&Channel{
		Id:     9201,
		Type:   1,
		Key:    "catalog-hosted-key",
		Name:   "catalog-hosted",
		Status: 1,
		Models: hosted,
		Group:  "catalog-test",
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "catalog-test",
		Model:     hosted,
		ChannelId: 9201,
		Enabled:   true,
	}).Error)

	models, total, err := SearchModels("np-catalog-test-", "", "", "", 0, 50)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)

	names := make([]string, 0, len(models))
	for _, item := range models {
		names = append(names, item.ModelName)
	}
	assert.Equal(t, []string{hosted}, names)
}

func TestSyncChannelModelMetadataInsertsMissingEnabledModels(t *testing.T) {
	name := "np-catalog-test-sync"
	setupChannelCatalogTest(t, name)

	require.NoError(t, DB.Create(&Channel{
		Id:     9202,
		Type:   1,
		Key:    "catalog-sync-key",
		Name:   "catalog-sync",
		Status: 1,
		Models: name,
		Group:  "catalog-test",
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "catalog-test",
		Model:     name,
		ChannelId: 9202,
		Enabled:   true,
	}).Error)

	SyncChannelModelMetadata()

	var stored Model
	require.NoError(t, DB.Where("model_name = ?", name).First(&stored).Error)
	assert.Equal(t, 1, stored.Status)
	assert.Equal(t, 0, stored.SyncOfficial)

	listed, total, err := SearchModels(name, "", "", "", 0, 50)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, listed, 1)
	assert.Equal(t, name, listed[0].ModelName)

	SyncChannelModelMetadata()
	var count int64
	require.NoError(t, DB.Model(&Model{}).Where("model_name = ?", name).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestGetVendorModelCountsIgnoresUnhostedMetadata(t *testing.T) {
	hosted := "np-catalog-test-count-hosted"
	orphan := "np-catalog-test-count-orphan"
	setupChannelCatalogTest(t, hosted, orphan)

	require.NoError(t, DB.Create(&Model{ModelName: hosted, VendorID: 9188, Status: 1}).Error)
	require.NoError(t, DB.Create(&Model{ModelName: orphan, VendorID: 9189, Status: 1}).Error)
	require.NoError(t, DB.Create(&Channel{
		Id:     9203,
		Type:   1,
		Key:    "catalog-count-key",
		Name:   "catalog-count",
		Status: 1,
		Models: hosted,
		Group:  "catalog-test",
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "catalog-test",
		Model:     hosted,
		ChannelId: 9203,
		Enabled:   true,
	}).Error)

	counts, err := GetVendorModelCounts()
	require.NoError(t, err)
	assert.EqualValues(t, 1, counts[9188])
	assert.EqualValues(t, 0, counts[9189])
}
