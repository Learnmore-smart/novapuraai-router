package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
)

func TestUpdateOptionMapKeepsTurnstileSecretWriteOnly(t *testing.T) {
	originalSecretKey := common.TurnstileSecretKey
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.TurnstileSecretKey = originalSecretKey
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	assert.NoError(t, updateOptionMap("TurnstileSecretKey", "dashboard-secret"))
	assert.Equal(t, "dashboard-secret", common.TurnstileSecretKey)

	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	assert.Empty(t, common.OptionMap["TurnstileSecretKey"])
	assert.Equal(t, "true", common.OptionMap["TurnstileSecretKeyConfigured"])
}

func TestUpdateOptionMapRefreshesGitHubSecretStatus(t *testing.T) {
	originalSecret := common.GitHubClientSecret
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.GitHubClientSecret = originalSecret
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	assert.NoError(t, updateOptionMap("GitHubClientSecret", "dashboard-secret"))
	assert.Equal(t, "dashboard-secret", common.GitHubClientSecret)

	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	assert.Empty(t, common.OptionMap["GitHubClientSecret"])
	assert.Equal(t, "true", common.OptionMap["GitHubClientSecretConfigured"])
}
