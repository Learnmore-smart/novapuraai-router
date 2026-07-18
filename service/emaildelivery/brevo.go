package emaildelivery

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const brevoAPIBaseURL = "https://api.brevo.com"

type brevoProvider struct {
	apiKey         string
	fromAddress    string
	fromName       string
	replyToAddress string
	baseURL        string
	client         *http.Client
}

func newBrevoProvider(apiKey, fromAddress, fromName, baseURL string, client *http.Client) *brevoProvider {
	return &brevoProvider{
		apiKey:      apiKey,
		fromAddress: fromAddress,
		fromName:    fromName,
		baseURL:     strings.TrimRight(baseURL, "/"),
		client:      client,
	}
}

func (provider *brevoProvider) Name() ProviderName {
	return ProviderBrevo
}

func (provider *brevoProvider) Send(ctx context.Context, message Message, idempotencyKey string) (ProviderResult, *DeliveryError) {
	if !provider.configured() {
		return ProviderResult{}, &DeliveryError{Reason: FailureConfiguration}
	}

	payload := struct {
		Sender struct {
			Name  string `json:"name,omitempty"`
			Email string `json:"email"`
		} `json:"sender"`
		To []struct {
			Email string `json:"email"`
		} `json:"to"`
		ReplyTo *struct {
			Email string `json:"email"`
		} `json:"replyTo,omitempty"`
		Subject     string            `json:"subject"`
		HTMLContent string            `json:"htmlContent,omitempty"`
		TextContent string            `json:"textContent,omitempty"`
		Headers     map[string]string `json:"headers"`
	}{}
	payload.Sender.Name = provider.fromName
	payload.Sender.Email = provider.fromAddress
	payload.To = append(payload.To, struct {
		Email string `json:"email"`
	}{Email: message.To})
	if provider.replyToAddress != "" {
		payload.ReplyTo = &struct {
			Email string `json:"email"`
		}{Email: provider.replyToAddress}
	}
	payload.Subject = message.Subject
	payload.HTMLContent = message.HTMLBody
	payload.TextContent = message.TextBody
	payload.Headers = map[string]string{"idempotencyKey": idempotencyKey}

	data, err := common.Marshal(payload)
	if err != nil {
		return ProviderResult{}, &DeliveryError{Reason: FailureInternal}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.baseURL+"/v3/smtp/email", bytes.NewReader(data))
	if err != nil {
		return ProviderResult{}, &DeliveryError{Reason: FailureInternal}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("api-key", provider.apiKey)

	response, err := provider.client.Do(request)
	if err != nil {
		if isAmbiguousTimeout(ctx, err) {
			return ProviderResult{}, &DeliveryError{Reason: FailureTimeoutAmbiguous, Ambiguous: true}
		}
		return ProviderResult{}, &DeliveryError{Reason: FailureProviderUnavailable}
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ProviderResult{}, &DeliveryError{Reason: brevoFailureReason(response.StatusCode)}
	}

	var result struct {
		MessageID string `json:"messageId"`
	}
	if err := common.DecodeJson(response.Body, &result); err != nil || result.MessageID == "" {
		return ProviderResult{}, &DeliveryError{Reason: FailureProviderUnavailable}
	}
	return ProviderResult{MessageID: result.MessageID}, nil
}

func (provider *brevoProvider) Health(ctx context.Context) ProviderHealth {
	health := ProviderHealth{Provider: ProviderBrevo, Configured: provider.configured()}
	if !health.Configured {
		health.FailureReason = FailureConfiguration
		return health
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.baseURL+"/v3/account", nil)
	if err != nil {
		health.FailureReason = FailureInternal
		return health
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("api-key", provider.apiKey)
	response, err := provider.client.Do(request)
	if err != nil {
		health.FailureReason = FailureProviderUnavailable
		return health
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		health.FailureReason = brevoFailureReason(response.StatusCode)
		return health
	}
	health.Reachable = true
	health.Ready = true
	return health
}

func (provider *brevoProvider) SupportsSafeRetry() bool {
	return true
}

func (provider *brevoProvider) configured() bool {
	return provider.apiKey != "" && provider.fromAddress != "" && provider.client != nil && provider.baseURL != ""
}

func brevoFailureReason(statusCode int) string {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return FailureAuthentication
	case statusCode == http.StatusTooManyRequests:
		return FailureRateLimited
	case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
		return FailureRejected
	default:
		return FailureProviderUnavailable
	}
}

func isAmbiguousTimeout(ctx context.Context, err error) bool {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
