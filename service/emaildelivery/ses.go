package emaildelivery

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/smithy-go"
)

type sesAPI interface {
	SendEmail(ctx context.Context, input *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
	GetAccount(ctx context.Context, input *sesv2.GetAccountInput, optFns ...func(*sesv2.Options)) (*sesv2.GetAccountOutput, error)
}

type sesProvider struct {
	client         sesAPI
	fromAddress    string
	replyToAddress string
	configured     bool
}

func newSESProvider(client sesAPI, fromAddress string, configured bool) *sesProvider {
	return &sesProvider{client: client, fromAddress: fromAddress, configured: configured}
}

func (provider *sesProvider) Name() ProviderName {
	return ProviderSES
}

func (provider *sesProvider) Send(ctx context.Context, message Message, idempotencyKey string) (ProviderResult, *DeliveryError) {
	if !provider.configured || provider.client == nil || provider.fromAddress == "" {
		return ProviderResult{}, &DeliveryError{Reason: FailureConfiguration}
	}

	charset := aws.String("UTF-8")
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(provider.fromAddress),
		Destination:      &types.Destination{ToAddresses: []string{message.To}},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(message.Subject), Charset: charset},
			Body: &types.Body{
				Html: &types.Content{Data: aws.String(message.HTMLBody), Charset: charset},
				Text: &types.Content{Data: aws.String(message.TextBody), Charset: charset},
			},
		}},
		EmailTags: []types.MessageTag{{
			Name:  aws.String("idempotency_key"),
			Value: aws.String(idempotencyKey),
		}},
	}
	if provider.replyToAddress != "" {
		input.ReplyToAddresses = []string{provider.replyToAddress}
	}

	output, err := provider.client.SendEmail(ctx, input, func(options *sesv2.Options) {
		options.RetryMaxAttempts = 1
	})
	if err != nil {
		if isAmbiguousTimeout(ctx, err) {
			return ProviderResult{}, &DeliveryError{Reason: FailureTimeoutAmbiguous, Ambiguous: true}
		}
		return ProviderResult{}, &DeliveryError{Reason: sesFailureReason(err)}
	}
	if output == nil || aws.ToString(output.MessageId) == "" {
		return ProviderResult{}, &DeliveryError{Reason: FailureProviderUnavailable}
	}
	return ProviderResult{MessageID: aws.ToString(output.MessageId)}, nil
}

func (provider *sesProvider) Health(ctx context.Context) ProviderHealth {
	health := ProviderHealth{Provider: ProviderSES, Configured: provider.configured && provider.client != nil && provider.fromAddress != ""}
	if !health.Configured {
		health.FailureReason = FailureConfiguration
		return health
	}

	output, err := provider.client.GetAccount(ctx, &sesv2.GetAccountInput{}, func(options *sesv2.Options) {
		options.RetryMaxAttempts = 1
	})
	if err != nil {
		health.FailureReason = sesFailureReason(err)
		return health
	}
	if output == nil {
		health.FailureReason = FailureProviderUnavailable
		return health
	}
	health.Reachable = true
	health.SendingEnabled = output.SendingEnabled
	health.ProductionAccess = output.ProductionAccessEnabled
	switch {
	case !health.SendingEnabled:
		health.FailureReason = FailureProviderUnavailable
	case !health.ProductionAccess:
		health.FailureReason = FailureProductionAccessRequired
	default:
		health.Ready = true
	}
	return health
}

func (provider *sesProvider) SupportsSafeRetry() bool {
	return false
}

func sesFailureReason(err error) string {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return FailureProviderUnavailable
	}
	switch apiErr.ErrorCode() {
	case "AccessDeniedException", "UnauthorizedException", "InvalidSignatureException", "UnrecognizedClientException":
		return FailureAuthentication
	case "TooManyRequestsException", "LimitExceededException", "ThrottlingException":
		return FailureRateLimited
	case "BadRequestException", "MessageRejected", "MailFromDomainNotVerifiedException", "NotFoundException":
		return FailureRejected
	default:
		return FailureProviderUnavailable
	}
}
