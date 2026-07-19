package emaildelivery

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/QuantumNous/new-api/common"
)

func TestSelectedProviderUsesEnvironmentUntilAdminOverrideExists(t *testing.T) {
	t.Setenv("EMAIL_PROVIDER", "ses")
	common.OptionMapRWMutex.Lock()
	previous := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previous
		common.OptionMapRWMutex.Unlock()
	})

	assert.Equal(t, ProviderSES, SelectedProvider())

	common.OptionMapRWMutex.Lock()
	common.OptionMap["EmailProvider"] = "brevo"
	common.OptionMapRWMutex.Unlock()
	assert.Equal(t, ProviderBrevo, SelectedProvider())
}

func TestNormalizeProviderNeverPromotesInvalidValueToSES(t *testing.T) {
	assert.Equal(t, ProviderBrevo, normalizeProvider(""))
	assert.Equal(t, ProviderBrevo, normalizeProvider("unknown"))
	assert.Equal(t, ProviderBrevo, normalizeProvider(" BREVO "))
	assert.Equal(t, ProviderSES, normalizeProvider(" SES "))
}

func TestDefaultEmailSenderParsesDisplayAddress(t *testing.T) {
	previousSMTPFrom := common.SMTPFrom
	previousSystemName := common.SystemName
	common.SMTPFrom = "NovaPuraAI <noreply@novapuraai.com>"
	common.SystemName = "Fallback Name"
	t.Setenv("EMAIL_FROM_ADDRESS", "")
	t.Setenv("EMAIL_FROM_NAME", "")
	t.Cleanup(func() {
		common.SMTPFrom = previousSMTPFrom
		common.SystemName = previousSystemName
	})

	address, name := defaultEmailSender()

	assert.Equal(t, "noreply@novapuraai.com", address)
	assert.Equal(t, "NovaPuraAI", name)
}

func TestDefaultEmailSenderRejectsMalformedDisplayAddress(t *testing.T) {
	previousSMTPFrom := common.SMTPFrom
	common.SMTPFrom = "NovaPuraAI <not-an-email>"
	t.Setenv("EMAIL_FROM_ADDRESS", "")
	t.Cleanup(func() { common.SMTPFrom = previousSMTPFrom })

	address, _ := defaultEmailSender()

	assert.Empty(t, address)
}

func TestDefaultSESRegionPrefersEnvironmentOverOption(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	previous := common.OptionMap
	common.OptionMap = map[string]string{"AWS_SES_REGION": "ap-southeast-1"}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previous
		common.OptionMapRWMutex.Unlock()
	})

	t.Setenv("AWS_SES_REGION", "us-east-2")
	assert.Equal(t, "us-east-2", defaultSESRegion())

	t.Setenv("AWS_SES_REGION", "")
	assert.Equal(t, "ap-southeast-1", defaultSESRegion())
}

func TestDefaultEmailFromRawDoesNotRequireLegacySMTPHost(t *testing.T) {
	previousSMTPFrom := common.SMTPFrom
	previousSMTPServer := common.SMTPServer
	common.SMTPFrom = "noreply@novapuraai.com"
	common.SMTPServer = ""
	t.Setenv("EMAIL_FROM_ADDRESS", "")
	t.Cleanup(func() {
		common.SMTPFrom = previousSMTPFrom
		common.SMTPServer = previousSMTPServer
	})

	assert.Equal(t, "noreply@novapuraai.com", defaultEmailFromRaw())
}
