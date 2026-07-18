package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestStripeCredentialProfilesAreEncryptedAndWriteOnly(t *testing.T) {
	originalDB := DB
	originalCryptoSecret := common.CryptoSecret
	t.Cleanup(func() {
		DB = originalDB
		common.CryptoSecret = originalCryptoSecret
	})

	var err error
	DB, err = gorm.Open(sqlite.Open("file:stripe-credentials?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, DB.AutoMigrate(&StripeCredential{}))
	common.CryptoSecret = "stripe-credential-test-secret"

	status, err := SaveStripeCredentials(StripeEnvironmentTest, StripeCredentialUpdate{
		SecretKey:      "sk_test_secret",
		PublishableKey: "pk_test_public",
		WebhookSecret:  "whsec_test_secret",
	})
	require.NoError(t, err)
	assert.True(t, status.SecretConfigured)
	assert.True(t, status.PublishableConfigured)
	assert.True(t, status.WebhookConfigured)

	var stored StripeCredential
	require.NoError(t, DB.Where("environment = ?", StripeEnvironmentTest).First(&stored).Error)
	assert.NotContains(t, stored.SecretKeyCiphertext, "sk_test_secret")
	assert.NotContains(t, stored.PublishableKeyCiphertext, "pk_test_public")
	assert.NotContains(t, stored.WebhookSecretCiphertext, "whsec_test_secret")

	credentials, found, err := LoadStripeCredentials(StripeEnvironmentTest)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "sk_test_secret", credentials.SecretKey)
	assert.Equal(t, "pk_test_public", credentials.PublishableKey)
	assert.Equal(t, "whsec_test_secret", credentials.WebhookSecret)

	_, err = SaveStripeCredentials(StripeEnvironmentTest, StripeCredentialUpdate{
		WebhookSecret: "whsec_rotated",
	})
	require.NoError(t, err)
	credentials, found, err = LoadStripeCredentials(StripeEnvironmentTest)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "sk_test_secret", credentials.SecretKey)
	assert.Equal(t, "pk_test_public", credentials.PublishableKey)
	assert.Equal(t, "whsec_rotated", credentials.WebhookSecret)
}

func TestStripeCredentialsRequireACompleteProfileAndCanBeDeleted(t *testing.T) {
	originalDB := DB
	originalCryptoSecret := common.CryptoSecret
	t.Cleanup(func() {
		DB = originalDB
		common.CryptoSecret = originalCryptoSecret
	})

	var err error
	DB, err = gorm.Open(sqlite.Open("file:stripe-credentials-delete?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, DB.AutoMigrate(&StripeCredential{}))
	common.CryptoSecret = "stripe-credential-test-secret"

	_, err = SaveStripeCredentials(StripeEnvironmentProduction, StripeCredentialUpdate{
		SecretKey: "sk_live_incomplete",
	})
	require.ErrorContains(t, err, "complete Stripe credentials are required")

	_, err = SaveStripeCredentials(StripeEnvironmentProduction, StripeCredentialUpdate{
		SecretKey:      "sk_live_secret",
		PublishableKey: "pk_live_public",
		WebhookSecret:  "whsec_live_secret",
	})
	require.NoError(t, err)
	require.NoError(t, DeleteStripeCredentials(StripeEnvironmentProduction))

	_, found, err := LoadStripeCredentials(StripeEnvironmentProduction)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestOptionSyncReloadsDatabaseStripeCredentials(t *testing.T) {
	originalDB := DB
	originalCryptoSecret := common.CryptoSecret
	t.Cleanup(func() {
		DB = originalDB
		common.CryptoSecret = originalCryptoSecret
		setting.ClearStripeCredentialProfile(setting.StripeRuntimeTest)
	})

	var err error
	DB, err = gorm.Open(sqlite.Open("file:stripe-credentials-option-sync?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, DB.AutoMigrate(&Option{}, &StripeCredential{}))
	common.CryptoSecret = "stripe-credential-test-secret"
	t.Setenv("GIN_MODE", "debug")
	setting.InitStripeEnv()
	_, err = SaveStripeCredentials(StripeEnvironmentTest, StripeCredentialUpdate{
		SecretKey:      "rk_test_synced",
		PublishableKey: "pk_test_synced",
		WebhookSecret:  "whsec_synced",
	})
	require.NoError(t, err)

	loadOptionsFromDatabase()

	assert.Equal(t, "rk_test_synced", setting.StripeApiSecret)
	assert.Equal(t, "pk_test_synced", setting.StripePublishableKey)
	assert.Equal(t, "whsec_synced", setting.StripeWebhookSecret)
}
