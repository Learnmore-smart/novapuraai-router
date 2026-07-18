package emaildelivery

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const brevoIdempotencyWindow = 14 * time.Minute

const (
	DeliveryStatusSending     = "sending"
	DeliveryStatusSent        = "sent"
	DeliveryStatusFailed      = "failed"
	DeliveryStatusRetryQueued = "retry_queued"
)

type SendRequest struct {
	BusinessKey string
	Message     Message
}

type DeliveryResult struct {
	Provider          ProviderName `json:"provider"`
	Status            string       `json:"status"`
	Duplicate         bool         `json:"duplicate"`
	ProviderMessageID string       `json:"provider_message_id,omitempty"`
	FailureReason     string       `json:"failure_reason,omitempty"`
}

type RetryResult struct {
	Processed int `json:"processed"`
	Sent      int `json:"sent"`
	Queued    int `json:"queued"`
	Failed    int `json:"failed"`
}

type HealthReport struct {
	SelectedProvider  ProviderName           `json:"selected_provider"`
	Providers         []ProviderHealth       `json:"providers"`
	SafeRetryCount    int64                  `json:"safe_retry_count"`
	ManualReviewCount int64                  `json:"manual_review_count"`
	LatestDelivery    *LatestDeliverySummary `json:"latest_delivery,omitempty"`
}

type LatestDeliverySummary struct {
	Provider          ProviderName `json:"provider"`
	MessageType       MessageType  `json:"message_type"`
	RecipientHash     string       `json:"recipient_hash"`
	ProviderMessageID string       `json:"provider_message_id,omitempty"`
	Status            string       `json:"status"`
	FailureReason     string       `json:"failure_reason,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
}

type Service struct {
	providersMu      sync.RWMutex
	providers        map[ProviderName]Provider
	selectedProvider func() ProviderName
	now              func() time.Time
}

func NewService(providers map[ProviderName]Provider, selectedProvider func() ProviderName, now func() time.Time) *Service {
	providerCopy := make(map[ProviderName]Provider, len(providers))
	for name, provider := range providers {
		providerCopy[name] = provider
	}
	return &Service{providers: providerCopy, selectedProvider: selectedProvider, now: now}
}

func (service *Service) Send(ctx context.Context, request SendRequest) (DeliveryResult, error) {
	if err := validateSendRequest(request); err != nil {
		return DeliveryResult{Status: DeliveryStatusFailed, FailureReason: FailureRejected}, err
	}

	providerName := service.selectedProvider()
	provider, ok := service.provider(providerName)
	recipientHash := common.GenerateHMAC("email-recipient:v1:" + strings.ToLower(strings.TrimSpace(request.Message.To)))
	payload, err := service.encryptMessage(request.Message)
	if err != nil {
		service.logDelivery(string(providerName), string(request.Message.Type), recipientHash, "", DeliveryStatusFailed, FailureInternal)
		return DeliveryResult{Provider: providerName, Status: DeliveryStatusFailed, FailureReason: FailureInternal}, fmt.Errorf("transactional email preparation failed: %s", FailureInternal)
	}

	idempotencyKey := service.idempotencyKey(request.BusinessKey, request.Message.Type)
	delivery, created, err := model.ReserveEmailDelivery(&model.EmailDelivery{
		IdempotencyKey:   idempotencyKey,
		Provider:         string(providerName),
		MessageType:      string(request.Message.Type),
		RecipientHash:    recipientHash,
		Status:           model.EmailDeliveryStatusSending,
		EncryptedPayload: payload,
		AttemptCount:     1,
	})
	if err != nil {
		service.logDelivery(string(providerName), string(request.Message.Type), recipientHash, "", DeliveryStatusFailed, FailureInternal)
		return DeliveryResult{Provider: providerName, Status: DeliveryStatusFailed, FailureReason: FailureInternal}, fmt.Errorf("transactional email reservation failed: %s", FailureInternal)
	}

	if !created {
		if delivery.Status != model.EmailDeliveryStatusFailed {
			result := deliveryResultFromModel(delivery, true)
			service.logDelivery(
				delivery.Provider,
				delivery.MessageType,
				delivery.RecipientHash,
				delivery.ProviderMessageId,
				"duplicate_"+delivery.Status,
				delivery.FailureReason,
			)
			return result, nil
		}
		claimed, err := model.ClaimFailedEmailDelivery(delivery.Id, string(providerName), recipientHash, payload)
		if err != nil {
			return deliveryResultFromModel(delivery, true), fmt.Errorf("transactional email reservation failed: %s", FailureInternal)
		}
		if !claimed {
			current, getErr := model.GetEmailDeliveryByKey(idempotencyKey)
			if getErr != nil {
				return deliveryResultFromModel(delivery, true), fmt.Errorf("transactional email reservation failed: %s", FailureInternal)
			}
			return deliveryResultFromModel(current, true), nil
		}
		delivery.Provider = string(providerName)
		delivery.RecipientHash = recipientHash
		delivery.Status = model.EmailDeliveryStatusSending
		delivery.FailureReason = ""
		delivery.EncryptedPayload = payload
		delivery.AttemptCount++
	}

	if !ok || provider == nil {
		return service.finishFailure(delivery, request.Message.Type, recipientHash, &DeliveryError{Reason: FailureConfiguration})
	}
	return service.sendReserved(ctx, delivery, request.Message, provider, recipientHash)
}

func (service *Service) RetrySafeQueue(ctx context.Context, limit int) (RetryResult, error) {
	now := service.now().UTC()
	deliveries, err := model.ListSafeRetryEmailDeliveries(now, limit)
	if err != nil {
		return RetryResult{}, err
	}

	result := RetryResult{}
	for index := range deliveries {
		delivery := &deliveries[index]
		claimed, err := model.ClaimEmailDeliveryForSafeRetry(delivery.Id, now)
		if err != nil {
			return result, err
		}
		if !claimed {
			continue
		}
		result.Processed++

		message, err := service.decryptMessage(delivery.EncryptedPayload)
		if err != nil {
			if markErr := model.MarkEmailDeliveryFailed(delivery.Id, FailureInternal); markErr != nil {
				return result, markErr
			}
			result.Failed++
			service.logDelivery(delivery.Provider, delivery.MessageType, delivery.RecipientHash, "", DeliveryStatusFailed, FailureInternal)
			continue
		}
		provider, _ := service.provider(ProviderName(delivery.Provider))
		if provider == nil || !provider.SupportsSafeRetry() {
			if markErr := model.MarkEmailDeliveryFailed(delivery.Id, FailureConfiguration); markErr != nil {
				return result, markErr
			}
			result.Failed++
			service.logDelivery(delivery.Provider, delivery.MessageType, delivery.RecipientHash, "", DeliveryStatusFailed, FailureConfiguration)
			continue
		}

		providerResult, sendErr := provider.Send(ctx, message, delivery.IdempotencyKey)
		if sendErr == nil {
			if err := model.MarkEmailDeliverySent(delivery.Id, providerResult.MessageID, now); err != nil {
				service.logDelivery(delivery.Provider, delivery.MessageType, delivery.RecipientHash, providerResult.MessageID, DeliveryStatusFailed, FailureInternal)
				return result, err
			}
			result.Sent++
			service.logDelivery(delivery.Provider, delivery.MessageType, delivery.RecipientHash, providerResult.MessageID, DeliveryStatusSent, "")
			continue
		}

		failureStatus := DeliveryStatusFailed
		if sendErr.Ambiguous && delivery.RetryDeadline != nil && !now.After(*delivery.RetryDeadline) {
			if err := model.MarkEmailDeliveryRetryQueued(delivery.Id, sendErr.Reason, delivery.RetryDeadline); err != nil {
				service.logDelivery(delivery.Provider, delivery.MessageType, delivery.RecipientHash, "", DeliveryStatusFailed, sendErr.Reason)
				return result, err
			}
			failureStatus = DeliveryStatusRetryQueued
			result.Queued++
		} else if err := model.MarkEmailDeliveryFailed(delivery.Id, sendErr.Reason); err != nil {
			service.logDelivery(delivery.Provider, delivery.MessageType, delivery.RecipientHash, "", DeliveryStatusFailed, sendErr.Reason)
			return result, err
		} else {
			result.Failed++
		}
		service.logDelivery(delivery.Provider, delivery.MessageType, delivery.RecipientHash, "", failureStatus, sendErr.Reason)
	}
	return result, nil
}

func (service *Service) Health(ctx context.Context) (HealthReport, error) {
	report := HealthReport{
		SelectedProvider: service.selectedProvider(),
		Providers:        make([]ProviderHealth, 2),
	}
	names := []ProviderName{ProviderBrevo, ProviderSES}
	type healthResult struct {
		index  int
		health ProviderHealth
	}
	results := make(chan healthResult, len(names))
	for index, name := range names {
		provider, _ := service.provider(name)
		go func(index int, name ProviderName, provider Provider) {
			if provider == nil {
				results <- healthResult{index: index, health: ProviderHealth{
					Provider:      name,
					FailureReason: FailureConfiguration,
				}}
				return
			}
			results <- healthResult{index: index, health: provider.Health(ctx)}
		}(index, name, provider)
	}
	for range names {
		providerResult := <-results
		report.Providers[providerResult.index] = providerResult.health
	}

	summary, err := model.GetEmailDeliverySummary(service.now().UTC())
	if err != nil {
		return HealthReport{}, err
	}
	report.SafeRetryCount = summary.SafeRetryCount
	report.ManualReviewCount = summary.ManualReviewCount
	if summary.Latest != nil {
		report.LatestDelivery = &LatestDeliverySummary{
			Provider:          ProviderName(summary.Latest.Provider),
			MessageType:       MessageType(summary.Latest.MessageType),
			RecipientHash:     summary.Latest.RecipientHash,
			ProviderMessageID: summary.Latest.ProviderMessageId,
			Status:            summary.Latest.Status,
			FailureReason:     summary.Latest.FailureReason,
			CreatedAt:         summary.Latest.CreatedAt,
		}
	}
	return report, nil
}

func (service *Service) sendReserved(ctx context.Context, delivery *model.EmailDelivery, message Message, provider Provider, recipientHash string) (DeliveryResult, error) {
	result, sendErr := provider.Send(ctx, message, delivery.IdempotencyKey)
	if sendErr != nil {
		return service.finishFailure(delivery, message.Type, recipientHash, sendErr)
	}

	now := service.now().UTC()
	if err := model.MarkEmailDeliverySent(delivery.Id, result.MessageID, now); err != nil {
		service.logDelivery(string(provider.Name()), string(message.Type), recipientHash, result.MessageID, DeliveryStatusFailed, FailureInternal)
		return DeliveryResult{Provider: provider.Name(), Status: DeliveryStatusFailed, FailureReason: FailureInternal}, err
	}
	service.logDelivery(string(provider.Name()), string(message.Type), recipientHash, result.MessageID, DeliveryStatusSent, "")
	return DeliveryResult{
		Provider:          provider.Name(),
		Status:            DeliveryStatusSent,
		ProviderMessageID: result.MessageID,
	}, nil
}

func (service *Service) finishFailure(delivery *model.EmailDelivery, messageType MessageType, recipientHash string, sendErr *DeliveryError) (DeliveryResult, error) {
	status := DeliveryStatusFailed
	if sendErr.Ambiguous {
		status = DeliveryStatusRetryQueued
		var retryDeadline *time.Time
		provider, _ := service.provider(ProviderName(delivery.Provider))
		if provider != nil && provider.SupportsSafeRetry() {
			deadline := service.now().UTC().Add(brevoIdempotencyWindow)
			retryDeadline = &deadline
		}
		if err := model.MarkEmailDeliveryRetryQueued(delivery.Id, sendErr.Reason, retryDeadline); err != nil {
			service.logDelivery(delivery.Provider, string(messageType), recipientHash, "", DeliveryStatusFailed, sendErr.Reason)
			return DeliveryResult{Provider: ProviderName(delivery.Provider), Status: DeliveryStatusFailed, FailureReason: FailureInternal}, err
		}
	} else if err := model.MarkEmailDeliveryFailed(delivery.Id, sendErr.Reason); err != nil {
		service.logDelivery(delivery.Provider, string(messageType), recipientHash, "", DeliveryStatusFailed, sendErr.Reason)
		return DeliveryResult{Provider: ProviderName(delivery.Provider), Status: DeliveryStatusFailed, FailureReason: FailureInternal}, err
	}

	service.logDelivery(delivery.Provider, string(messageType), recipientHash, "", status, sendErr.Reason)
	return DeliveryResult{
		Provider:      ProviderName(delivery.Provider),
		Status:        status,
		FailureReason: sendErr.Reason,
	}, fmt.Errorf("transactional email delivery failed: %s", sendErr.Reason)
}

func (service *Service) provider(name ProviderName) (Provider, bool) {
	service.providersMu.RLock()
	defer service.providersMu.RUnlock()
	provider, ok := service.providers[name]
	return provider, ok
}

func (service *Service) replaceProvider(name ProviderName, provider Provider) {
	service.providersMu.Lock()
	defer service.providersMu.Unlock()
	service.providers[name] = provider
}

func (service *Service) encryptMessage(message Message) (string, error) {
	data, err := common.Marshal(message)
	if err != nil {
		return "", err
	}
	return common.EncryptSensitiveString(string(data))
}

func (service *Service) decryptMessage(payload string) (Message, error) {
	plain, err := common.DecryptSensitiveString(payload)
	if err != nil {
		return Message{}, err
	}
	var message Message
	if err := common.UnmarshalJsonStr(plain, &message); err != nil {
		return Message{}, err
	}
	return message, nil
}

func (service *Service) idempotencyKey(businessKey string, messageType MessageType) string {
	return common.GenerateHMAC("email-delivery:v1:" + string(messageType) + ":" + strings.TrimSpace(businessKey))
}

func (service *Service) logDelivery(provider, messageType, recipientHash, providerMessageID, deliveryResult, failureReason string) {
	entry := fmt.Sprintf(
		"email_delivery provider=%s message_type=%s recipient_hash=%s provider_message_id=%s delivery_result=%s failure_reason=%s",
		provider,
		messageType,
		recipientHash,
		providerMessageID,
		deliveryResult,
		failureReason,
	)
	if failureReason == "" {
		common.SysLog(entry)
	} else {
		common.SysError(entry)
	}
}

func deliveryResultFromModel(delivery *model.EmailDelivery, duplicate bool) DeliveryResult {
	return DeliveryResult{
		Provider:          ProviderName(delivery.Provider),
		Status:            delivery.Status,
		Duplicate:         duplicate,
		ProviderMessageID: delivery.ProviderMessageId,
		FailureReason:     delivery.FailureReason,
	}
}

func validateSendRequest(request SendRequest) error {
	if strings.TrimSpace(request.BusinessKey) == "" || strings.TrimSpace(request.Message.To) == "" || strings.TrimSpace(request.Message.Subject) == "" {
		return fmt.Errorf("invalid transactional email request")
	}
	switch request.Message.Type {
	case MessageTypeVerification, MessageTypePasswordReset, MessageTypeReceipt, MessageTypeNotification:
		return nil
	default:
		return fmt.Errorf("invalid transactional email message type")
	}
}
