package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNextEnabledKeyTreatsLegacyNewlineCredentialsAsPool(t *testing.T) {
	channel := &Channel{
		Key: " key-one\r\n\r\nkey-two \n",
		ChannelInfo: ChannelInfo{
			IsMultiKey:   false,
			MultiKeyMode: constant.MultiKeyModeRandom,
		},
	}

	keys := channel.GetKeys()
	key, index, err := channel.GetNextEnabledKey()

	require.Equal(t, []string{"key-one", "key-two"}, keys)
	require.Nil(t, err)
	assert.Contains(t, keys, key)
	assert.Contains(t, []int{0, 1}, index)
}
