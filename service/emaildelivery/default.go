package emaildelivery

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/bytedance/gopkg/util/gopool"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	providerOptionKey      = "EmailProvider"
	emailProviderEnv       = "EMAIL_PROVIDER"
	safeRetryPollInterval  = 30 * time.Second
	defaultProviderTimeout = 12 * time.Second
)

var (
	defaultServiceOnce sync.Once
	defaultService     *Service
	retryWorkerOnce    sync.Once
)

func SelectedProvider() ProviderName {
	common.OptionMapRWMutex.RLock()
	persisted := common.OptionMap[providerOptionKey]
	common.OptionMapRWMutex.RUnlock()
	if strings.TrimSpace(persisted) != "" {
		return normalizeProvider(persisted)
	}
	return normalizeProvider(os.Getenv(emailProviderEnv))
}

func SendTransactionalEmail(ctx context.Context, request SendRequest) (DeliveryResult, error) {
	return DefaultService().Send(ctx, request)
}

func GetHealth(ctx context.Context) (HealthReport, error) {
	return DefaultService().Health(ctx)
}

func RetrySafeDeliveries(ctx context.Context, limit int) (RetryResult, error) {
	return DefaultService().RetrySafeQueue(ctx, limit)
}

func DefaultService() *Service {
	defaultServiceOnce.Do(func() {
		client := &http.Client{Timeout: defaultProviderTimeout}
		fromAddress, fromName := defaultEmailSender()

		brevo := newBrevoProvider(
			strings.TrimSpace(os.Getenv("BREVO_API_KEY")),
			fromAddress,
			fromName,
			brevoAPIBaseURL,
			client,
		)
		brevo.replyToAddress = defaultEmailReplyTo()

		ses, err := buildDefaultSESProvider(client)
		if err != nil {
			common.SysError("transactional SES credential loading failed")
			ses = newSESProvider(nil, "", false)
		}

		defaultService = NewService(map[ProviderName]Provider{
			ProviderBrevo: brevo,
			ProviderSES:   ses,
		}, SelectedProvider, time.Now)
	})
	return defaultService
}

func defaultEmailSender() (string, string) {
	fromAddress := strings.TrimSpace(os.Getenv("EMAIL_FROM_ADDRESS"))
	if fromAddress == "" {
		fromAddress = strings.TrimSpace(common.SMTPFrom)
	}
	fromName := strings.TrimSpace(os.Getenv("EMAIL_FROM_NAME"))
	if fromAddress != "" {
		parsed, err := mail.ParseAddress(fromAddress)
		if err != nil || !strings.Contains(parsed.Address, "@") {
			fromAddress = ""
		} else {
			fromAddress = strings.TrimSpace(parsed.Address)
			if fromName == "" {
				fromName = strings.TrimSpace(parsed.Name)
			}
		}
	}
	if fromName == "" {
		fromName = common.SystemName
	}
	return fromAddress, fromName
}

func defaultEmailReplyTo() string {
	return strings.TrimSpace(os.Getenv("EMAIL_REPLY_TO"))
}

func buildDefaultSESProvider(client *http.Client) (*sesProvider, error) {
	resolution, err := resolveSESCredentials(os.Getenv, model.LoadSESCredentials)
	if err != nil {
		return nil, err
	}
	fromAddress, fromName := defaultEmailSender()
	sesFromAddress := fromAddress
	if fromName != "" && fromAddress != "" {
		sesFromAddress = (&mail.Address{Name: fromName, Address: fromAddress}).String()
	}
	region := strings.TrimSpace(os.Getenv("AWS_SES_REGION"))
	sesClient := sesv2.NewFromConfig(aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(
			resolution.Credentials.AccessKeyID,
			resolution.Credentials.SecretAccessKey,
			resolution.Credentials.SessionToken,
		),
		HTTPClient:       client,
		RetryMaxAttempts: 1,
	})
	provider := newSESProvider(
		sesClient,
		sesFromAddress,
		resolution.Status.Configured && region != "" && fromAddress != "",
	)
	provider.replyToAddress = defaultEmailReplyTo()
	return provider, nil
}

func reloadDefaultSESProvider() error {
	provider, err := buildDefaultSESProvider(&http.Client{Timeout: defaultProviderTimeout})
	if err != nil {
		return err
	}
	DefaultService().replaceProvider(ProviderSES, provider)
	return nil
}

func StartSafeRetryWorker() {
	retryWorkerOnce.Do(func() {
		gopool.Go(func() {
			runSafeRetries := func() {
				ctx, cancel := context.WithTimeout(context.Background(), defaultProviderTimeout)
				defer cancel()
				result, err := RetrySafeDeliveries(ctx, 50)
				if err != nil {
					common.SysError("email safe retry worker failed: " + err.Error())
					return
				}
				if result.Processed > 0 {
					common.SysLog(fmt.Sprintf("email safe retry worker processed=%d sent=%d queued=%d failed=%d", result.Processed, result.Sent, result.Queued, result.Failed))
				}
			}

			runSafeRetries()
			ticker := time.NewTicker(safeRetryPollInterval)
			defer ticker.Stop()
			for range ticker.C {
				runSafeRetries()
			}
		})
	})
}

func normalizeProvider(value string) ProviderName {
	if strings.EqualFold(strings.TrimSpace(value), string(ProviderSES)) {
		return ProviderSES
	}
	return ProviderBrevo
}
