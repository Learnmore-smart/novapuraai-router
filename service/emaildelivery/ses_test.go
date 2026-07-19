package emaildelivery

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
)

func TestSESProviderMapsProviderIndependentMessage(t *testing.T) {
	client := &fakeSESClient{
		sendOutput: &sesv2.SendEmailOutput{MessageId: aws.String("ses-message-id")},
	}
	provider := newSESProvider(client, "sender@example.com", true, true, true)
	provider.replyToAddress = "support@example.com"

	result, sendErr := provider.Send(context.Background(), testMessage(), "delivery-key")

	require.Nil(t, sendErr)
	assert.Equal(t, "ses-message-id", result.MessageID)
	require.NotNil(t, client.sendInput)
	assert.Equal(t, "sender@example.com", aws.ToString(client.sendInput.FromEmailAddress))
	assert.Equal(t, []string{"recipient@example.com"}, client.sendInput.Destination.ToAddresses)
	assert.Equal(t, []string{"support@example.com"}, client.sendInput.ReplyToAddresses)
	simple := client.sendInput.Content.Simple
	require.NotNil(t, simple)
	assert.Equal(t, "Subject", aws.ToString(simple.Subject.Data))
	assert.Equal(t, "UTF-8", aws.ToString(simple.Subject.Charset))
	assert.Equal(t, "<p>Body</p>", aws.ToString(simple.Body.Html.Data))
	assert.Equal(t, "Body", aws.ToString(simple.Body.Text.Data))
	require.Len(t, client.sendInput.EmailTags, 1)
	assert.Equal(t, "idempotency_key", aws.ToString(client.sendInput.EmailTags[0].Name))
	assert.Equal(t, "delivery-key", aws.ToString(client.sendInput.EmailTags[0].Value))
}

func TestSESProviderHealthRequiresProductionAccess(t *testing.T) {
	client := &fakeSESClient{
		accountOutput: &sesv2.GetAccountOutput{
			SendingEnabled:          true,
			ProductionAccessEnabled: false,
		},
	}
	provider := newSESProvider(client, "sender@example.com", true, true, true)

	health := provider.Health(context.Background())

	assert.Equal(t, ProviderSES, health.Provider)
	assert.True(t, health.Configured)
	assert.True(t, health.CredentialsConfigured)
	assert.True(t, health.RegionConfigured)
	assert.True(t, health.SenderConfigured)
	assert.True(t, health.Reachable)
	assert.True(t, health.SendingEnabled)
	assert.False(t, health.ProductionAccess)
	assert.True(t, health.SandboxRestricted)
	assert.False(t, health.Ready)
	assert.Equal(t, FailureProductionAccessRequired, health.FailureReason)

	client.accountOutput.ProductionAccessEnabled = true
	health = provider.Health(context.Background())
	assert.True(t, health.Ready)
	assert.False(t, health.SandboxRestricted)
	assert.Empty(t, health.FailureReason)
}

func TestSESProviderTimeoutIsAmbiguousAndNeverAutomaticallyRetried(t *testing.T) {
	client := &fakeSESClient{sendErr: timeoutError{}}
	provider := newSESProvider(client, "sender@example.com", true, true, true)

	_, sendErr := provider.Send(context.Background(), testMessage(), "delivery-key")

	require.NotNil(t, sendErr)
	assert.True(t, sendErr.Ambiguous)
	assert.Equal(t, FailureTimeoutAmbiguous, sendErr.Reason)
	assert.False(t, provider.SupportsSafeRetry())
}

func TestSESProviderReportsMissingConfigurationWithoutCallingAWS(t *testing.T) {
	client := &fakeSESClient{}
	provider := newSESProvider(client, "", false, false, false)

	health := provider.Health(context.Background())

	assert.False(t, health.Configured)
	assert.False(t, health.CredentialsConfigured)
	assert.False(t, health.RegionConfigured)
	assert.False(t, health.SenderConfigured)
	assert.False(t, health.Ready)
	assert.Equal(t, FailureConfiguration, health.FailureReason)
	assert.Zero(t, client.accountCalls)
}

func TestSESProviderReportsPartialConfigurationComponents(t *testing.T) {
	client := &fakeSESClient{}
	// Credentials and region present, sender missing — must not call AWS and
	// must not collapse the status into a single opaque "unconfigured" blob.
	provider := newSESProvider(client, "", true, true, false)

	health := provider.Health(context.Background())

	assert.False(t, health.Configured)
	assert.True(t, health.CredentialsConfigured)
	assert.True(t, health.RegionConfigured)
	assert.False(t, health.SenderConfigured)
	assert.Equal(t, FailureConfiguration, health.FailureReason)
	assert.Zero(t, client.accountCalls)
}

func TestSESProviderHealthJSONIncludesFalseComponentFlags(t *testing.T) {
	// false must serialize as false, not be dropped by omitempty, so the
	// Dashboard can show Yes/No per SES component instead of Unknown.
	client := &fakeSESClient{}
	provider := newSESProvider(client, "", true, false, false)

	health := provider.Health(context.Background())
	payload, err := common.Marshal(health)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, common.Unmarshal(payload, &raw))

	assert.Equal(t, true, raw["credentials_configured"])
	assert.Equal(t, false, raw["region_configured"])
	assert.Equal(t, false, raw["sender_configured"])
	assert.Equal(t, false, raw["production_access"])
	assert.Equal(t, false, raw["sandbox_restricted"])
	assert.Equal(t, false, raw["sending_enabled"])
	assert.Equal(t, false, raw["configured"])
}

func testMessage() Message {
	return Message{
		Type:     MessageTypeVerification,
		To:       "recipient@example.com",
		Subject:  "Subject",
		HTMLBody: "<p>Body</p>",
		TextBody: "Body",
	}
}

type fakeSESClient struct {
	sendInput     *sesv2.SendEmailInput
	sendOutput    *sesv2.SendEmailOutput
	sendErr       error
	accountOutput *sesv2.GetAccountOutput
	accountErr    error
	accountCalls  int
}

func (client *fakeSESClient) SendEmail(_ context.Context, input *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	client.sendInput = input
	return client.sendOutput, client.sendErr
}

func (client *fakeSESClient) GetAccount(_ context.Context, _ *sesv2.GetAccountInput, _ ...func(*sesv2.Options)) (*sesv2.GetAccountOutput, error) {
	client.accountCalls++
	return client.accountOutput, client.accountErr
}

var _ sesAPI = (*fakeSESClient)(nil)
var _ = types.Body{}
