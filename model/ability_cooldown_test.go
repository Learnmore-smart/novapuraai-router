package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterAbilitiesSkipsLiveCooldownWithoutRequestPath(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Channel{}))
	baseID := int(time.Now().UnixNano()%1_000_000) + 1_000_000
	channels := []*Channel{
		{
			Id: baseID, Name: fmt.Sprintf("cooling-%d", baseID),
			Status: common.ChannelStatusEnabled, CooldownUntil: time.Now().Unix() + 300,
		},
		{
			Id: baseID + 1, Name: fmt.Sprintf("healthy-%d", baseID),
			Status: common.ChannelStatusEnabled,
		},
	}
	for _, channel := range channels {
		require.NoError(t, DB.Create(channel).Error)
	}
	t.Cleanup(func() {
		_ = DB.Delete(&Channel{}, []int{baseID, baseID + 1}).Error
	})

	filtered := filterAbilitiesByRequestPathAndModel([]Ability{
		{ChannelId: baseID},
		{ChannelId: baseID + 1},
	}, "", "model")

	require.Len(t, filtered, 1)
	assert.Equal(t, baseID+1, filtered[0].ChannelId)
}
