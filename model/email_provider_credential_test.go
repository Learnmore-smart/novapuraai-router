package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEmailProviderCredentialTest(t *testing.T) {
	t.Helper()
	originalSecret := common.CryptoSecret
	common.CryptoSecret = "email-provider-credential-test-secret"
	require.NoError(t, DB.AutoMigrate(&EmailProviderCredential{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&EmailProviderCredential{}).Error)
	t.Cleanup(func() {
		common.CryptoSecret = originalSecret
		_ = DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&EmailProviderCredential{}).Error
	})
}

func TestEmailProviderCredentialEncryptsAndSupportsWriteOnlyUpdates(t *testing.T) {
	setupEmailProviderCredentialTest(t)

	status, err := SaveSESCredentials(SESCredentialUpdate{
		AccessKeyID:     "AKIAEXAMPLE123456789",
		SecretAccessKey: "initial-secret-access-key",
		SessionToken:    "initial-session-token",
	})
	require.NoError(t, err)
	assert.True(t, status.Configured)
	assert.True(t, status.HasSessionToken)

	var stored EmailProviderCredential
	require.NoError(t, DB.Where("provider = ?", EmailProviderSES).First(&stored).Error)
	assert.NotEqual(t, "AKIAEXAMPLE123456789", stored.AccessKeyIdCiphertext)
	assert.NotEqual(t, "initial-secret-access-key", stored.SecretAccessKeyCiphertext)
	assert.NotEqual(t, "initial-session-token", stored.SessionTokenCiphertext)
	assert.NotContains(t, stored.AccessKeyIdCiphertext, "AKIAEXAMPLE")

	status, err = SaveSESCredentials(SESCredentialUpdate{SecretAccessKey: "replacement-secret-access-key"})
	require.NoError(t, err)
	assert.True(t, status.Configured)
	assert.True(t, status.HasSessionToken)

	credentials, found, err := LoadSESCredentials()
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "AKIAEXAMPLE123456789", credentials.AccessKeyID)
	assert.Equal(t, "replacement-secret-access-key", credentials.SecretAccessKey)
	assert.Equal(t, "initial-session-token", credentials.SessionToken)

	status, err = SaveSESCredentials(SESCredentialUpdate{ClearSessionToken: true})
	require.NoError(t, err)
	assert.True(t, status.Configured)
	assert.False(t, status.HasSessionToken)

	credentials, found, err = LoadSESCredentials()
	require.NoError(t, err)
	require.True(t, found)
	assert.Empty(t, credentials.SessionToken)

	require.NoError(t, DeleteSESCredentials())
	_, found, err = LoadSESCredentials()
	require.NoError(t, err)
	assert.False(t, found)
}

func TestEmailProviderCredentialRejectsIncompleteInitialCredentials(t *testing.T) {
	setupEmailProviderCredentialTest(t)

	_, err := SaveSESCredentials(SESCredentialUpdate{AccessKeyID: "AKIAEXAMPLE123456789"})
	require.Error(t, err)

	var count int64
	require.NoError(t, DB.Model(&EmailProviderCredential{}).Count(&count).Error)
	assert.Zero(t, count)
}
