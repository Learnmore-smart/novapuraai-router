package deepseekfairuse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountKeysSupplyEveryLuaKeyInOneHashSlot(t *testing.T) {
	keys := accountKeys("account-hmac")
	require.Len(t, keys, 10)

	for _, key := range keys {
		assert.Contains(t, key, "{account-hmac}")
	}
	assert.Equal(t, "deepseek:fup:v1:{account-hmac}:penalty:events", keys[9])
	assert.Equal(t, 1, strings.Count(keys[9], "{account-hmac}"))
	assert.Contains(t, FairUseScript, "local events_key = KEYS[10]")
}
