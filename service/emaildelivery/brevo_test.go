package emaildelivery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
)

func TestBrevoProviderSendUsesProviderIndependentMessageAndIdempotencyKey(t *testing.T) {
	var request struct {
		Sender struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"sender"`
		To []struct {
			Email string `json:"email"`
		} `json:"to"`
		ReplyTo struct {
			Email string `json:"email"`
		} `json:"replyTo"`
		Subject     string            `json:"subject"`
		HTMLContent string            `json:"htmlContent"`
		TextContent string            `json:"textContent"`
		Headers     map[string]string `json:"headers"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/smtp/email", r.URL.Path)
		assert.Equal(t, "secret-api-key", r.Header.Get("api-key"))
		require.NoError(t, common.DecodeJson(r.Body, &request))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messageId":"brevo-message-id"}`))
	}))
	t.Cleanup(server.Close)

	provider := newBrevoProvider("secret-api-key", "sender@example.com", "Example", server.URL, server.Client())
	provider.replyToAddress = "support@example.com"
	result, sendErr := provider.Send(context.Background(), testMessage(), "delivery-key")

	require.Nil(t, sendErr)
	assert.Equal(t, "brevo-message-id", result.MessageID)
	assert.Equal(t, "Example", request.Sender.Name)
	assert.Equal(t, "sender@example.com", request.Sender.Email)
	require.Len(t, request.To, 1)
	assert.Equal(t, "recipient@example.com", request.To[0].Email)
	assert.Equal(t, "support@example.com", request.ReplyTo.Email)
	assert.Equal(t, "Subject", request.Subject)
	assert.Equal(t, "<p>Body</p>", request.HTMLContent)
	assert.Equal(t, "Body", request.TextContent)
	assert.Equal(t, "delivery-key", request.Headers["idempotencyKey"])
}

func TestBrevoProviderHealthRequiresConfigurationAndReachability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/account", r.URL.Path)
		assert.Equal(t, "secret-api-key", r.Header.Get("api-key"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	provider := newBrevoProvider("secret-api-key", "sender@example.com", "Example", server.URL, server.Client())
	health := provider.Health(context.Background())

	assert.Equal(t, ProviderBrevo, health.Provider)
	assert.True(t, health.Configured)
	assert.True(t, health.Reachable)
	assert.True(t, health.Ready)
	assert.Empty(t, health.FailureReason)
}

func TestBrevoProviderTreatsTimeoutAsAmbiguousWithoutLeakingDetails(t *testing.T) {
	provider := newBrevoProvider(
		"secret-api-key",
		"sender@example.com",
		"Example",
		"https://api.example.invalid",
		&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, timeoutError{}
		})},
	)

	_, sendErr := provider.Send(context.Background(), testMessage(), "delivery-key")

	require.NotNil(t, sendErr)
	assert.True(t, sendErr.Ambiguous)
	assert.Equal(t, FailureTimeoutAmbiguous, sendErr.Reason)
	assert.NotContains(t, sendErr.Error(), "secret-api-key")
	assert.True(t, provider.SupportsSafeRetry())
}

func TestBrevoProviderReturnsBoundedAuthenticationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("upstream details must not escape"))
	}))
	t.Cleanup(server.Close)

	provider := newBrevoProvider("secret-api-key", "sender@example.com", "Example", server.URL, server.Client())
	_, sendErr := provider.Send(context.Background(), testMessage(), "delivery-key")

	require.NotNil(t, sendErr)
	assert.False(t, sendErr.Ambiguous)
	assert.Equal(t, FailureAuthentication, sendErr.Reason)
	assert.NotContains(t, sendErr.Error(), "upstream details")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "request timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
func (timeoutError) Unwrap() error   { return errors.New("private upstream details") }
