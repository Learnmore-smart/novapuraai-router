package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPublicModelAbilityTest(t *testing.T, modelNames ...string) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &Model{}, &Vendor{}))
	cleanup := func() {
		DB.Where("model IN ?", modelNames).Delete(&Ability{})
		DB.Where("id IN ?", []int{9301, 9302, 9303}).Delete(&Channel{})
		DB.Unscoped().Where("model_name IN ?", modelNames).Delete(&Model{})
		InvalidatePricingCache()
	}
	cleanup()
	t.Cleanup(cleanup)
}

func TestGetPricingListsEveryModelOnEnabledChannels(t *testing.T) {
	alias := "kimi-k3"
	upstream := "moonshotai/kimi-k3"
	channelOnly := "nvidia/llama-3.1-nemotron-70b-instruct"
	setupPublicModelAbilityTest(t, alias, upstream, channelOnly)

	mapping := `{"kimi-k3":"moonshotai/kimi-k3"}`
	require.NoError(t, DB.Create(&Channel{
		Id:           9301,
		Type:         constant.ChannelTypeOpenAI,
		Key:          "channel-models-key",
		Name:         "nvidia-nim",
		Status:       common.ChannelStatusEnabled,
		Models:       alias + "," + upstream + "," + channelOnly,
		Group:        "default",
		ModelMapping: &mapping,
	}).Error)
	for _, name := range []string{alias, upstream, channelOnly} {
		require.NoError(t, DB.Create(&Ability{
			Group:     "default",
			Model:     name,
			ChannelId: 9301,
			Enabled:   true,
		}).Error)
	}

	InvalidatePricingCache()
	names := make([]string, 0)
	for _, item := range GetPricing() {
		names = append(names, item.ModelName)
	}
	assert.ElementsMatch(t, []string{alias, upstream, channelOnly}, names)
	assert.ElementsMatch(t, []string{alias, upstream, channelOnly}, GetEnabledModels())
	assert.ElementsMatch(t, []string{alias, upstream, channelOnly}, GetGroupEnabledModels("default"))
}

func TestGetPricingOmitsAbilityNotOnChannelModels(t *testing.T) {
	listed := "kimi-k3"
	stale := "adept/fuyu-8b"
	setupPublicModelAbilityTest(t, listed, stale)

	require.NoError(t, DB.Create(&Channel{
		Id:     9302,
		Type:   constant.ChannelTypeOpenAI,
		Key:    "channel-csv-key",
		Name:   "nvidia-nim",
		Status: common.ChannelStatusEnabled,
		Models: listed,
		Group:  "default",
	}).Error)
	for _, name := range []string{listed, stale} {
		require.NoError(t, DB.Create(&Ability{
			Group:     "default",
			Model:     name,
			ChannelId: 9302,
			Enabled:   true,
		}).Error)
	}

	InvalidatePricingCache()
	names := make([]string, 0)
	for _, item := range GetPricing() {
		names = append(names, item.ModelName)
	}
	assert.Equal(t, []string{listed}, names)
	assert.Equal(t, []string{listed}, GetEnabledModels())
	assert.NotContains(t, GetEnabledModels(), stale)
}

func TestGetPricingOmitsDisabledChannelModels(t *testing.T) {
	name := "secret-disabled-model"
	setupPublicModelAbilityTest(t, name)

	require.NoError(t, DB.Create(&Channel{
		Id:     9303,
		Type:   constant.ChannelTypeOpenAI,
		Key:    "disabled-key",
		Name:   "disabled-channel",
		Status: common.ChannelStatusManuallyDisabled,
		Models: name,
		Group:  "default",
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     name,
		ChannelId: 9303,
		Enabled:   true,
	}).Error)

	InvalidatePricingCache()
	for _, item := range GetPricing() {
		assert.NotEqual(t, name, item.ModelName)
	}
	assert.NotContains(t, GetEnabledModels(), name)
}
