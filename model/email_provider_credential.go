package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type EmailProviderCredential struct {
	Id                        int       `json:"-" gorm:"primaryKey"`
	Provider                  string    `json:"-" gorm:"type:varchar(16);not null;uniqueIndex"`
	AccessKeyIdCiphertext     string    `json:"-" gorm:"type:text;not null"`
	SecretAccessKeyCiphertext string    `json:"-" gorm:"type:text;not null"`
	SessionTokenCiphertext    string    `json:"-" gorm:"type:text"`
	CreatedAt                 time.Time `json:"-"`
	UpdatedAt                 time.Time `json:"-"`
}

func (EmailProviderCredential) TableName() string {
	return "email_provider_credentials"
}

type SESCredentialUpdate struct {
	AccessKeyID       string
	SecretAccessKey   string
	SessionToken      string
	ClearSessionToken bool
}

type SESCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type EmailProviderCredentialStatus struct {
	Configured      bool
	HasSessionToken bool
}

func SaveSESCredentials(update SESCredentialUpdate) (EmailProviderCredentialStatus, error) {
	if update.SessionToken != "" && update.ClearSessionToken {
		return EmailProviderCredentialStatus{}, fmt.Errorf("session token replacement and clearing are mutually exclusive")
	}

	status := EmailProviderCredentialStatus{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var credential EmailProviderCredential
		err := lockForUpdate(tx).
			Where("provider = ?", EmailProviderSES).
			First(&credential).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if strings.TrimSpace(update.AccessKeyID) != "" {
			credential.AccessKeyIdCiphertext, err = common.EncryptSensitiveString(strings.TrimSpace(update.AccessKeyID))
			if err != nil {
				return err
			}
		}
		if update.SecretAccessKey != "" {
			credential.SecretAccessKeyCiphertext, err = common.EncryptSensitiveString(update.SecretAccessKey)
			if err != nil {
				return err
			}
		}
		if update.SessionToken != "" {
			credential.SessionTokenCiphertext, err = common.EncryptSensitiveString(update.SessionToken)
			if err != nil {
				return err
			}
		} else if update.ClearSessionToken {
			credential.SessionTokenCiphertext = ""
		}

		if credential.AccessKeyIdCiphertext == "" || credential.SecretAccessKeyCiphertext == "" {
			return fmt.Errorf("complete SES credentials are required")
		}
		credential.Provider = EmailProviderSES
		if credential.Id == 0 {
			if err := tx.Create(&credential).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&credential).Error; err != nil {
			return err
		}

		status.Configured = true
		status.HasSessionToken = credential.SessionTokenCiphertext != ""
		return nil
	})
	return status, err
}

func LoadSESCredentials() (SESCredentials, bool, error) {
	var credential EmailProviderCredential
	err := DB.Where("provider = ?", EmailProviderSES).First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return SESCredentials{}, false, nil
	}
	if err != nil {
		return SESCredentials{}, false, err
	}

	accessKeyID, err := common.DecryptSensitiveString(credential.AccessKeyIdCiphertext)
	if err != nil {
		return SESCredentials{}, false, err
	}
	secretAccessKey, err := common.DecryptSensitiveString(credential.SecretAccessKeyCiphertext)
	if err != nil {
		return SESCredentials{}, false, err
	}
	credentials := SESCredentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}
	if credential.SessionTokenCiphertext != "" {
		credentials.SessionToken, err = common.DecryptSensitiveString(credential.SessionTokenCiphertext)
		if err != nil {
			return SESCredentials{}, false, err
		}
	}
	return credentials, true, nil
}

func GetSESCredentialStatus() (EmailProviderCredentialStatus, error) {
	var credential EmailProviderCredential
	err := DB.Select("access_key_id_ciphertext", "secret_access_key_ciphertext", "session_token_ciphertext").
		Where("provider = ?", EmailProviderSES).
		First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return EmailProviderCredentialStatus{}, nil
	}
	if err != nil {
		return EmailProviderCredentialStatus{}, err
	}
	return EmailProviderCredentialStatus{
		Configured:      credential.AccessKeyIdCiphertext != "" && credential.SecretAccessKeyCiphertext != "",
		HasSessionToken: credential.SessionTokenCiphertext != "",
	}, nil
}

func DeleteSESCredentials() error {
	return DB.Where("provider = ?", EmailProviderSES).Delete(&EmailProviderCredential{}).Error
}
