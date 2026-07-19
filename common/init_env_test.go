package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveListeningPort(t *testing.T) {
	tests := []struct {
		name         string
		environment  string
		flagValue    int
		flagExplicit bool
		expected     int
		wantError    bool
	}{
		{name: "default flag value", flagValue: 3000, expected: 3000},
		{name: "cloud run environment", environment: "8080", flagValue: 3000, expected: 8080},
		{name: "explicit flag wins", environment: "8080", flagValue: 9000, flagExplicit: true, expected: 9000},
		{name: "invalid text", environment: "eight-thousand", flagValue: 3000, wantError: true},
		{name: "zero rejected", environment: "0", flagValue: 3000, wantError: true},
		{name: "too large rejected", environment: "65536", flagValue: 3000, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			port, err := resolveListeningPort(test.environment, test.flagValue, test.flagExplicit)
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, port)
		})
	}
}

func TestInitSMTPEnvLoadsCanonicalValuesAndAliases(t *testing.T) {
	originalServer := SMTPServer
	originalPort := SMTPPort
	originalAccount := SMTPAccount
	originalToken := SMTPToken
	originalFrom := SMTPFrom
	originalSSL := SMTPSSLEnabled
	originalStartTLS := SMTPStartTLSEnabled
	originalSkipVerify := SMTPInsecureSkipVerify
	originalForceLogin := SMTPForceAuthLogin
	t.Cleanup(func() {
		SMTPServer = originalServer
		SMTPPort = originalPort
		SMTPAccount = originalAccount
		SMTPToken = originalToken
		SMTPFrom = originalFrom
		SMTPSSLEnabled = originalSSL
		SMTPStartTLSEnabled = originalStartTLS
		SMTPInsecureSkipVerify = originalSkipVerify
		SMTPForceAuthLogin = originalForceLogin
	})

	t.Setenv("SMTP_SERVER", "email-smtp.ca-central-1.amazonaws.com")
	t.Setenv("SMTPServer", "legacy.example.com")
	t.Setenv("SMTP_PORT", "2587")
	t.Setenv("SMTP_ACCOUNT", "smtp-user")
	t.Setenv("SMTP_TOKEN", "smtp-password")
	t.Setenv("SMTP_FROM", "no-reply@novapuraai.com")
	t.Setenv("SMTP_SSL_ENABLED", "false")
	t.Setenv("SMTP_STARTTLS_ENABLED", "true")
	t.Setenv("SMTP_INSECURE_SKIP_VERIFY", "false")
	t.Setenv("SMTP_FORCE_AUTH_LOGIN", "true")

	initSMTPEnv()

	assert.Equal(t, "email-smtp.ca-central-1.amazonaws.com", SMTPServer)
	assert.Equal(t, 2587, SMTPPort)
	assert.Equal(t, "smtp-user", SMTPAccount)
	assert.Equal(t, "smtp-password", SMTPToken)
	assert.Equal(t, "no-reply@novapuraai.com", SMTPFrom)
	assert.False(t, SMTPSSLEnabled)
	assert.True(t, SMTPStartTLSEnabled)
	assert.False(t, SMTPInsecureSkipVerify)
	assert.True(t, SMTPForceAuthLogin)
}

func TestInitSMTPEnvKeepsSafePortForInvalidValue(t *testing.T) {
	originalPort := SMTPPort
	t.Cleanup(func() { SMTPPort = originalPort })
	SMTPPort = 587
	t.Setenv("SMTP_PORT", "70000")
	t.Setenv("SMTPPort", "")

	initSMTPEnv()

	assert.Equal(t, 587, SMTPPort)
}

func TestInitTurnstileEnv(t *testing.T) {
	originalEnabled := TurnstileCheckEnabled
	originalSiteKey := TurnstileSiteKey
	originalSecretKey := TurnstileSecretKey
	originalHostnames := TurnstileAllowedHostnames
	t.Cleanup(func() {
		TurnstileCheckEnabled = originalEnabled
		TurnstileSiteKey = originalSiteKey
		TurnstileSecretKey = originalSecretKey
		TurnstileAllowedHostnames = originalHostnames
	})

	t.Setenv("TURNSTILE_CHECK_ENABLED", "true")
	t.Setenv("TURNSTILE_SITE_KEY", "site-key")
	t.Setenv("TURNSTILE_SECRET_KEY", "secret-key")
	t.Setenv("TURNSTILE_ALLOWED_HOSTNAMES", "novapuraai.com,localhost")

	initTurnstileEnv()

	assert.True(t, TurnstileCheckEnabled)
	assert.Equal(t, "site-key", TurnstileSiteKey)
	assert.Equal(t, "secret-key", TurnstileSecretKey)
	assert.Equal(t, "novapuraai.com,localhost", TurnstileAllowedHostnames)
}

func TestInitGitHubOAuthEnv(t *testing.T) {
	originalEnabled := GitHubOAuthEnabled
	originalClientId := GitHubClientId
	originalClientSecret := GitHubClientSecret
	t.Cleanup(func() {
		GitHubOAuthEnabled = originalEnabled
		GitHubClientId = originalClientId
		GitHubClientSecret = originalClientSecret
	})

	t.Setenv("GITHUB_OAUTH_ENABLED", "true")
	t.Setenv("GITHUB_OAUTH_CLIENT_ID", "gh-client-id")
	t.Setenv("GITHUB_OAUTH_CLIENT_SECRET", "gh-client-secret")

	initGitHubOAuthEnv()

	assert.True(t, GitHubOAuthEnabled)
	assert.Equal(t, "gh-client-id", GitHubClientId)
	assert.Equal(t, "gh-client-secret", GitHubClientSecret)
	assert.True(t, GitHubOAuthReady())
	assert.True(t, GitHubOAuthActive())
}

func TestApplyEnvManagedSecretsOverridesInMemoryAndClearsOptionMap(t *testing.T) {
	originalClientId := GitHubClientId
	originalClientSecret := GitHubClientSecret
	originalTurnstileSecret := TurnstileSecretKey
	originalSMTPToken := SMTPToken
	t.Cleanup(func() {
		GitHubClientId = originalClientId
		GitHubClientSecret = originalClientSecret
		TurnstileSecretKey = originalTurnstileSecret
		SMTPToken = originalSMTPToken
		OptionMapRWMutex.Lock()
		delete(OptionMap, "GitHubClientSecret")
		delete(OptionMap, "TurnstileSecretKey")
		delete(OptionMap, "SMTPToken")
		delete(OptionMap, "GitHubClientSecretConfigured")
		delete(OptionMap, "TurnstileSecretKeyConfigured")
		delete(OptionMap, "SMTPTokenConfigured")
		OptionMapRWMutex.Unlock()
	})

	GitHubClientSecret = "db-should-not-win"
	TurnstileSecretKey = "db-turnstile"
	SMTPToken = "db-smtp"
	OptionMapRWMutex.Lock()
	if OptionMap == nil {
		OptionMap = make(map[string]string)
	}
	OptionMap["GitHubClientSecret"] = "leaked"
	OptionMap["TurnstileSecretKey"] = "leaked"
	OptionMap["SMTPToken"] = "leaked"
	OptionMapRWMutex.Unlock()

	t.Setenv("GITHUB_OAUTH_CLIENT_ID", "env-client-id")
	t.Setenv("GITHUB_OAUTH_CLIENT_SECRET", "env-secret")
	t.Setenv("TURNSTILE_SECRET_KEY", "env-turnstile")
	t.Setenv("SMTP_TOKEN", "env-smtp")

	ApplyEnvManagedSecrets()

	assert.Equal(t, "env-client-id", GitHubClientId)
	assert.Equal(t, "env-secret", GitHubClientSecret)
	assert.Equal(t, "env-turnstile", TurnstileSecretKey)
	assert.Equal(t, "env-smtp", SMTPToken)

	OptionMapRWMutex.RLock()
	defer OptionMapRWMutex.RUnlock()
	assert.Equal(t, "", OptionMap["GitHubClientSecret"])
	assert.Equal(t, "", OptionMap["TurnstileSecretKey"])
	assert.Equal(t, "", OptionMap["SMTPToken"])
	assert.Equal(t, "true", OptionMap["GitHubClientSecretConfigured"])
	assert.Equal(t, "true", OptionMap["TurnstileSecretKeyConfigured"])
	assert.Equal(t, "true", OptionMap["SMTPTokenConfigured"])
}

func TestApplyEnvManagedSecretsKeepsGitHubSecretFromDBWhenEnvAbsent(t *testing.T) {
	originalClientSecret := GitHubClientSecret
	t.Cleanup(func() {
		GitHubClientSecret = originalClientSecret
		OptionMapRWMutex.Lock()
		delete(OptionMap, "GitHubClientSecret")
		delete(OptionMap, "GitHubClientSecretConfigured")
		OptionMapRWMutex.Unlock()
	})

	// Simulate a value hydrated from the options table / admin write with no
	// GITHUB_OAUTH_CLIENT_SECRET in the environment.
	GitHubClientSecret = "db-configured-secret"
	OptionMapRWMutex.Lock()
	if OptionMap == nil {
		OptionMap = make(map[string]string)
	}
	OptionMap["GitHubClientSecret"] = "db-configured-secret"
	OptionMapRWMutex.Unlock()

	ApplyEnvManagedSecrets()

	// DB value is preserved (env-preferred, DB-fallback)...
	assert.Equal(t, "db-configured-secret", GitHubClientSecret)

	OptionMapRWMutex.RLock()
	defer OptionMapRWMutex.RUnlock()
	// ...but never echoed back through the settings API.
	assert.Equal(t, "", OptionMap["GitHubClientSecret"])
	assert.Equal(t, "true", OptionMap["GitHubClientSecretConfigured"])
}

func TestApplyEnvManagedSecretsKeepsTurnstileSecretWhenEnvironmentIsBlank(t *testing.T) {
	originalSecret := TurnstileSecretKey
	OptionMapRWMutex.Lock()
	originalOptionMap := OptionMap
	OptionMap = map[string]string{}
	OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		TurnstileSecretKey = originalSecret
		OptionMapRWMutex.Lock()
		OptionMap = originalOptionMap
		OptionMapRWMutex.Unlock()
	})

	TurnstileSecretKey = "dashboard-secret"
	t.Setenv("TURNSTILE_SECRET_KEY", "")

	ApplyEnvManagedSecrets()

	assert.Equal(t, "dashboard-secret", TurnstileSecretKey)
}

func TestApplyEnvManagedSecretsKeepsSMTPTokenFromDBWhenEnvironmentIsBlank(t *testing.T) {
	originalToken := SMTPToken
	OptionMapRWMutex.Lock()
	originalOptionMap := OptionMap
	OptionMap = map[string]string{}
	OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		SMTPToken = originalToken
		OptionMapRWMutex.Lock()
		OptionMap = originalOptionMap
		OptionMapRWMutex.Unlock()
	})

	// Simulate a value hydrated from the options table / admin write with no
	// SMTP_TOKEN in the environment.
	SMTPToken = "dashboard-smtp-token"
	t.Setenv("SMTP_TOKEN", "")

	ApplyEnvManagedSecrets()

	// DB value is preserved (env-preferred, DB-fallback)...
	assert.Equal(t, "dashboard-smtp-token", SMTPToken)

	OptionMapRWMutex.RLock()
	defer OptionMapRWMutex.RUnlock()
	// ...but never echoed back through the settings API.
	assert.Equal(t, "", OptionMap["SMTPToken"])
	assert.Equal(t, "true", OptionMap["SMTPTokenConfigured"])
}

func TestGitHubOAuthFailClosedWithoutSecret(t *testing.T) {
	originalEnabled := GitHubOAuthEnabled
	originalClientId := GitHubClientId
	originalClientSecret := GitHubClientSecret
	t.Cleanup(func() {
		GitHubOAuthEnabled = originalEnabled
		GitHubClientId = originalClientId
		GitHubClientSecret = originalClientSecret
	})

	GitHubOAuthEnabled = true
	GitHubClientId = "client"
	GitHubClientSecret = ""
	assert.False(t, GitHubOAuthReady())
	assert.False(t, GitHubOAuthActive())

	GitHubClientSecret = "secret"
	assert.True(t, GitHubOAuthActive())
}

func TestIsEnvManagedSecretOptionKey(t *testing.T) {
	assert.False(t, IsEnvManagedSecretOptionKey("TurnstileSecretKey"))
	// SMTPToken is env-preferred but DB-writable, so it is not treated as a
	// strictly env-managed key that the admin API must reject.
	assert.False(t, IsEnvManagedSecretOptionKey("SMTPToken"))
	assert.True(t, IsEnvManagedSecretOptionKey("BREVO_API_KEY"))
	assert.True(t, IsEnvManagedSecretOptionKey("AWS_SES_ACCESS_KEY_ID"))
	assert.True(t, IsEnvManagedSecretOptionKey("AWS_SES_SECRET_ACCESS_KEY"))
	assert.True(t, IsEnvManagedSecretOptionKey("AWS_SES_SESSION_TOKEN"))
	assert.False(t, IsEnvManagedSecretOptionKey("GitHubClientId"))
	// GitHubClientSecret is env-preferred but DB-writable, so it is not treated
	// as a strictly env-managed key that the admin API must reject.
	assert.False(t, IsEnvManagedSecretOptionKey("GitHubClientSecret"))
}
