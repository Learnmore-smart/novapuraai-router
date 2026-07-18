package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
)

func TestUpdateEmailProviderPersistsValidatedProvider(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Option{}))
	require.NoError(t, DB.Where("key = ?", "EmailProvider").Delete(&Option{}).Error)

	common.OptionMapRWMutex.Lock()
	previousOptionMap := common.OptionMap
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previousValue, hadPreviousValue := common.OptionMap["EmailProvider"]
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		_ = DB.Where("key = ?", "EmailProvider").Delete(&Option{}).Error
		common.OptionMapRWMutex.Lock()
		if previousOptionMap == nil {
			common.OptionMap = nil
		} else if hadPreviousValue {
			common.OptionMap["EmailProvider"] = previousValue
		} else {
			delete(common.OptionMap, "EmailProvider")
		}
		common.OptionMapRWMutex.Unlock()
	})

	require.NoError(t, UpdateEmailProvider(EmailProviderSES))
	var stored Option
	require.NoError(t, DB.Where("key = ?", "EmailProvider").First(&stored).Error)
	assert.Equal(t, EmailProviderSES, stored.Value)

	common.OptionMapRWMutex.RLock()
	assert.Equal(t, EmailProviderSES, common.OptionMap["EmailProvider"])
	common.OptionMapRWMutex.RUnlock()

	require.Error(t, UpdateEmailProvider("smtp"))
}
