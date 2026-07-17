package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptChannelKey(t *testing.T) {
	CryptoSecret = "test-crypto-secret-for-channel-keys"
	plain := "sk-test-provider-key-abcdef"
	enc, err := EncryptChannelKey(plain)
	require.NoError(t, err)
	assert.True(t, IsEncryptedChannelKey(enc))
	assert.NotEqual(t, plain, enc)

	out, err := DecryptChannelKey(enc)
	require.NoError(t, err)
	assert.Equal(t, plain, out)

	// legacy plaintext passes through
	legacy, err := DecryptChannelKey("plain-legacy-key")
	require.NoError(t, err)
	assert.Equal(t, "plain-legacy-key", legacy)

	// encrypt is idempotent for already-encrypted
	enc2, err := EncryptChannelKey(enc)
	require.NoError(t, err)
	assert.Equal(t, enc, enc2)
}
