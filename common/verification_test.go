package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreateVerificationCodeReusesUnexpiredBusinessEvent(t *testing.T) {
	key := "idempotent@example.com"
	purpose := "verification-test-purpose"
	DeleteKey(key, purpose)
	t.Cleanup(func() { DeleteKey(key, purpose) })

	first, created := GetOrCreateVerificationCodeWithKey(key, purpose, 6)
	second, createdAgain := GetOrCreateVerificationCodeWithKey(key, purpose, 6)

	require.Len(t, first, 6)
	assert.True(t, created)
	assert.False(t, createdAgain)
	assert.Equal(t, first, second)
	assert.True(t, VerifyCodeWithKey(key, first, purpose))

	replacement, replacementCreated := GetOrCreateVerificationCodeWithKey(key, purpose, 6)
	assert.True(t, replacementCreated)
	assert.NotEqual(t, first, replacement)
}
