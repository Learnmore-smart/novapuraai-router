package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	StripeEnvironmentTest       = "test"
	StripeEnvironmentProduction = "production"
)

type StripeCredential struct {
	Id                       int       `json:"-" gorm:"primaryKey"`
	Environment              string    `json:"-" gorm:"type:varchar(16);not null;uniqueIndex"`
	SecretKeyCiphertext      string    `json:"-" gorm:"type:text;not null"`
	PublishableKeyCiphertext string    `json:"-" gorm:"type:text;not null"`
	WebhookSecretCiphertext  string    `json:"-" gorm:"type:text;not null"`
	CreatedAt                time.Time `json:"-"`
	UpdatedAt                time.Time `json:"-"`
}

func (StripeCredential) TableName() string {
	return "stripe_credentials"
}

type StripeCredentialUpdate struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
}

type StripeCredentials struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
}

type StripeCredentialStatus struct {
	SecretConfigured      bool `json:"secret_configured"`
	PublishableConfigured bool `json:"publishable_configured"`
	WebhookConfigured     bool `json:"webhook_configured"`
}

func validateStripeEnvironment(environment string) error {
	if environment != StripeEnvironmentTest && environment != StripeEnvironmentProduction {
		return fmt.Errorf("unsupported Stripe environment")
	}
	return nil
}

func SaveStripeCredentials(environment string, update StripeCredentialUpdate) (StripeCredentialStatus, error) {
	if err := validateStripeEnvironment(environment); err != nil {
		return StripeCredentialStatus{}, err
	}

	status := StripeCredentialStatus{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var credential StripeCredential
		err := lockForUpdate(tx).Where("environment = ?", environment).First(&credential).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if strings.TrimSpace(update.SecretKey) != "" {
			credential.SecretKeyCiphertext, err = common.EncryptSensitiveString(strings.TrimSpace(update.SecretKey))
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(update.PublishableKey) != "" {
			credential.PublishableKeyCiphertext, err = common.EncryptSensitiveString(strings.TrimSpace(update.PublishableKey))
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(update.WebhookSecret) != "" {
			credential.WebhookSecretCiphertext, err = common.EncryptSensitiveString(strings.TrimSpace(update.WebhookSecret))
			if err != nil {
				return err
			}
		}

		if credential.SecretKeyCiphertext == "" || credential.PublishableKeyCiphertext == "" || credential.WebhookSecretCiphertext == "" {
			return fmt.Errorf("complete Stripe credentials are required")
		}
		credential.Environment = environment
		if credential.Id == 0 {
			if err := tx.Create(&credential).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&credential).Error; err != nil {
			return err
		}
		status = StripeCredentialStatus{
			SecretConfigured:      true,
			PublishableConfigured: true,
			WebhookConfigured:     true,
		}
		return nil
	})
	return status, err
}

func LoadStripeCredentials(environment string) (StripeCredentials, bool, error) {
	if err := validateStripeEnvironment(environment); err != nil {
		return StripeCredentials{}, false, err
	}
	var credential StripeCredential
	err := DB.Where("environment = ?", environment).First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return StripeCredentials{}, false, nil
	}
	if err != nil {
		return StripeCredentials{}, false, err
	}

	secretKey, err := common.DecryptSensitiveString(credential.SecretKeyCiphertext)
	if err != nil {
		return StripeCredentials{}, false, err
	}
	publishableKey, err := common.DecryptSensitiveString(credential.PublishableKeyCiphertext)
	if err != nil {
		return StripeCredentials{}, false, err
	}
	webhookSecret, err := common.DecryptSensitiveString(credential.WebhookSecretCiphertext)
	if err != nil {
		return StripeCredentials{}, false, err
	}
	return StripeCredentials{
		SecretKey:      secretKey,
		PublishableKey: publishableKey,
		WebhookSecret:  webhookSecret,
	}, true, nil
}

func GetStripeCredentialStatus(environment string) (StripeCredentialStatus, error) {
	if err := validateStripeEnvironment(environment); err != nil {
		return StripeCredentialStatus{}, err
	}
	var credential StripeCredential
	err := DB.Select("secret_key_ciphertext", "publishable_key_ciphertext", "webhook_secret_ciphertext").
		Where("environment = ?", environment).First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return StripeCredentialStatus{}, nil
	}
	if err != nil {
		return StripeCredentialStatus{}, err
	}
	return StripeCredentialStatus{
		SecretConfigured:      credential.SecretKeyCiphertext != "",
		PublishableConfigured: credential.PublishableKeyCiphertext != "",
		WebhookConfigured:     credential.WebhookSecretCiphertext != "",
	}, nil
}

func DeleteStripeCredentials(environment string) error {
	if err := validateStripeEnvironment(environment); err != nil {
		return err
	}
	return DB.Where("environment = ?", environment).Delete(&StripeCredential{}).Error
}
