package emaildelivery

import "context"

type ProviderName string

const (
	ProviderBrevo ProviderName = "brevo"
	ProviderSES   ProviderName = "ses"
)

type MessageType string

const (
	MessageTypeVerification  MessageType = "verification"
	MessageTypePasswordReset MessageType = "password_reset"
	MessageTypeReceipt       MessageType = "receipt"
	MessageTypeNotification  MessageType = "notification"
)

const (
	FailureConfiguration            = "configuration"
	FailureAuthentication           = "authentication"
	FailureRejected                 = "rejected"
	FailureRateLimited              = "rate_limited"
	FailureProviderUnavailable      = "provider_unavailable"
	FailureTimeoutAmbiguous         = "timeout_ambiguous"
	FailureProductionAccessRequired = "production_access_required"
	FailureInternal                 = "internal"
)

type Message struct {
	Type     MessageType `json:"type"`
	To       string      `json:"to"`
	Subject  string      `json:"subject"`
	HTMLBody string      `json:"html_body"`
	TextBody string      `json:"text_body"`
}

type ProviderResult struct {
	MessageID string
}

type ProviderHealth struct {
	Provider         ProviderName `json:"provider"`
	Configured       bool         `json:"configured"`
	Reachable        bool         `json:"reachable"`
	Ready            bool         `json:"ready"`
	SendingEnabled   bool         `json:"sending_enabled,omitempty"`
	ProductionAccess bool         `json:"production_access,omitempty"`
	FailureReason    string       `json:"failure_reason,omitempty"`
}

type DeliveryError struct {
	Reason    string
	Ambiguous bool
}

func (err *DeliveryError) Error() string {
	return err.Reason
}

type Provider interface {
	Name() ProviderName
	Send(ctx context.Context, message Message, idempotencyKey string) (ProviderResult, *DeliveryError)
	Health(ctx context.Context) ProviderHealth
	SupportsSafeRetry() bool
}
