package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateChannelAppendsPastedKeysToOriginalSingleKey(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	original := &model.Channel{
		Type:   constant.ChannelTypeOpenAI,
		Key:    "key-one",
		Name:   "credential-pool",
		Models: "gpt-test",
		Group:  "default",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, db.Create(original).Error)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/channel/", map[string]any{
		"id":       original.Id,
		"type":     original.Type,
		"key":      " key-two\r\nkey-three\nkey-two ",
		"key_mode": "append",
		"name":     original.Name,
		"models":   original.Models,
		"group":    original.Group,
	}, 1)
	ctx.Set("role", common.RoleRootUser)

	UpdateChannel(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	updated, err := model.GetChannelById(original.Id, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"key-one", "key-two", "key-three"}, updated.GetKeys())
	assert.True(t, updated.ChannelInfo.IsMultiKey)
	assert.Equal(t, 3, updated.ChannelInfo.MultiKeySize)
	assert.Equal(t, constant.MultiKeyModeRandom, updated.ChannelInfo.MultiKeyMode)
}
