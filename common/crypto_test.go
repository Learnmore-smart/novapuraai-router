package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptSensitiveStringRoundTrip(t *testing.T) {
	originalSecret := CryptoSecret
	CryptoSecret = "email-test-secret"
	t.Cleanup(func() {
		CryptoSecret = originalSecret
	})

	encrypted, err := EncryptSensitiveString("recipient and rendered message")
	require.NoError(t, err)
	assert.NotContains(t, encrypted, "recipient")

	plain, err := DecryptSensitiveString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "recipient and rendered message", plain)
}

func TestDecryptSensitiveStringRejectsChannelCiphertext(t *testing.T) {
	originalSecret := CryptoSecret
	CryptoSecret = "email-test-secret"
	t.Cleanup(func() {
		CryptoSecret = originalSecret
	})

	encrypted, err := EncryptChannelKey("provider-secret")
	require.NoError(t, err)

	_, err = DecryptSensitiveString(encrypted)
	require.Error(t, err)
}
