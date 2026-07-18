package emaildelivery

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSESCredentialsUsesExactlyOneCompleteSource(t *testing.T) {
	databaseCredentials := model.SESCredentials{
		AccessKeyID:     "database-access-key",
		SecretAccessKey: "database-secret-key",
		SessionToken:    "database-token",
	}

	t.Run("complete environment wins", func(t *testing.T) {
		loadedDatabase := false
		resolution, err := resolveSESCredentials(
			func(key string) string {
				return map[string]string{
					"AWS_SES_ACCESS_KEY_ID":     "environment-access-key",
					"AWS_SES_SECRET_ACCESS_KEY": "environment-secret-key",
					"AWS_SES_SESSION_TOKEN":     "environment-token",
				}[key]
			},
			func() (model.SESCredentials, bool, error) {
				loadedDatabase = true
				return databaseCredentials, true, nil
			},
		)
		require.NoError(t, err)
		assert.False(t, loadedDatabase)
		assert.Equal(t, SESCredentialSourceEnvironment, resolution.Status.Source)
		assert.Equal(t, "environment-access-key", resolution.Credentials.AccessKeyID)
		assert.True(t, resolution.Status.Configured)
		assert.True(t, resolution.Status.HasSessionToken)
	})

	t.Run("database is used only when environment is empty", func(t *testing.T) {
		resolution, err := resolveSESCredentials(
			func(string) string { return "" },
			func() (model.SESCredentials, bool, error) { return databaseCredentials, true, nil },
		)
		require.NoError(t, err)
		assert.Equal(t, SESCredentialSourceDatabase, resolution.Status.Source)
		assert.Equal(t, databaseCredentials, resolution.Credentials)
		assert.True(t, resolution.Status.Configured)
	})

	t.Run("partial environment fails closed instead of mixing sources", func(t *testing.T) {
		loadedDatabase := false
		_, err := resolveSESCredentials(
			func(key string) string {
				if key == "AWS_SES_ACCESS_KEY_ID" {
					return "environment-access-key"
				}
				return ""
			},
			func() (model.SESCredentials, bool, error) {
				loadedDatabase = true
				return databaseCredentials, true, nil
			},
		)
		require.Error(t, err)
		assert.False(t, loadedDatabase)
	})
}

func TestSaveSESCredentialsStoresDashboardFallbackBehindEnvironmentOverride(t *testing.T) {
	setupEmailServiceTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.EmailProviderCredential{}))
	t.Setenv("AWS_SES_ACCESS_KEY_ID", "environment-access-key")
	t.Setenv("AWS_SES_SECRET_ACCESS_KEY", "environment-secret-key")
	t.Setenv("AWS_SES_SESSION_TOKEN", "environment-token")

	status, err := SaveSESCredentials(context.Background(), SESCredentialUpdate{
		AccessKeyID:     "dashboard-access-key",
		SecretAccessKey: "dashboard-secret-key",
	})
	require.NoError(t, err)
	assert.Equal(t, SESCredentialSourceEnvironment, status.Source)

	credentials, found, err := model.LoadSESCredentials()
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "dashboard-access-key", credentials.AccessKeyID)
	assert.Equal(t, "dashboard-secret-key", credentials.SecretAccessKey)
}

func TestServiceProviderReplacementIsVisibleToHealth(t *testing.T) {
	now := setupEmailServiceTest(t)
	oldSES := &fakeProvider{name: ProviderSES, health: ProviderHealth{Provider: ProviderSES, Configured: false}}
	service := NewService(map[ProviderName]Provider{
		ProviderBrevo: &fakeProvider{name: ProviderBrevo},
		ProviderSES:   oldSES,
	}, func() ProviderName { return ProviderSES }, func() time.Time { return now })

	service.replaceProvider(ProviderSES, &fakeProvider{name: ProviderSES, health: ProviderHealth{
		Provider:         ProviderSES,
		Configured:       true,
		Reachable:        true,
		Ready:            true,
		SendingEnabled:   true,
		ProductionAccess: true,
	}})

	report, err := service.Health(context.Background())
	require.NoError(t, err)
	sesHealth := report.Providers[1]
	assert.True(t, sesHealth.Ready)
	assert.True(t, sesHealth.ProductionAccess)
}
