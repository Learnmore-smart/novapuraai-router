package dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
)

func TestNewNotifyAssignsPrivateStableEventID(t *testing.T) {
	first := NewNotify(NotifyTypeChannelUpdate, "Title", "Content", nil)
	second := NewNotify(NotifyTypeChannelUpdate, "Title", "Content", nil)

	assert.NotEmpty(t, first.EventID)
	assert.NotEqual(t, first.EventID, second.EventID)

	data, err := common.Marshal(first)
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(data), first.EventID))
}
