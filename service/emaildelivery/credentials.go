package emaildelivery

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

var (
	ErrIncompleteSESEnvironmentCredentials = errors.New("incomplete environment-managed SES credentials")
)

type SESCredentialSource string

const (
	SESCredentialSourceNone        SESCredentialSource = "none"
	SESCredentialSourceEnvironment SESCredentialSource = "environment"
	SESCredentialSourceDatabase    SESCredentialSource = "database"
)

type SESCredentialStatus struct {
	Configured      bool                `json:"configured"`
	Source          SESCredentialSource `json:"source"`
	HasSessionToken bool                `json:"has_session_token"`
	// Non-secret SES settings shown in the Dashboard form.
	Region      string `json:"region"`
	FromAddress string `json:"from_address"`
}

type SESCredentialUpdate struct {
	AccessKeyID       string  `json:"access_key_id"`
	SecretAccessKey   string  `json:"secret_access_key"`
	SessionToken      string  `json:"session_token"`
	ClearSessionToken bool    `json:"clear_session_token"`
	Region            *string `json:"region"`
	FromAddress       *string `json:"from_address"`
}

type sesCredentialResolution struct {
	Credentials model.SESCredentials
	Status      SESCredentialStatus
}

func GetSESCredentialStatus(context.Context) (SESCredentialStatus, error) {
	resolution, err := resolveSESCredentials(os.Getenv, model.LoadSESCredentials)
	if err != nil {
		return SESCredentialStatus{}, err
	}
	return attachSESSettings(resolution.Status), nil
}

func SaveSESCredentials(_ context.Context, update SESCredentialUpdate) (SESCredentialStatus, error) {
	hasCredentialChange := update.AccessKeyID != "" ||
		update.SecretAccessKey != "" ||
		update.SessionToken != "" ||
		update.ClearSessionToken
	if hasCredentialChange {
		_, err := model.SaveSESCredentials(model.SESCredentialUpdate{
			AccessKeyID:       update.AccessKeyID,
			SecretAccessKey:   update.SecretAccessKey,
			SessionToken:      update.SessionToken,
			ClearSessionToken: update.ClearSessionToken,
		})
		if err != nil {
			return SESCredentialStatus{}, err
		}
	}
	if update.Region != nil {
		if err := model.UpdateOption("AWS_SES_REGION", strings.TrimSpace(*update.Region)); err != nil {
			return SESCredentialStatus{}, err
		}
	}
	if update.FromAddress != nil {
		fromAddress := strings.TrimSpace(*update.FromAddress)
		if fromAddress != "" {
			parsed, err := mail.ParseAddress(fromAddress)
			if err != nil || parsed.Address == "" || !strings.Contains(parsed.Address, "@") {
				return SESCredentialStatus{}, fmt.Errorf("invalid SES from address")
			}
		}
		if err := model.UpdateOption("SMTPFrom", fromAddress); err != nil {
			return SESCredentialStatus{}, err
		}
	}
	if err := reloadDefaultSESProvider(); err != nil {
		return SESCredentialStatus{}, err
	}
	return GetSESCredentialStatus(context.Background())
}

func DeleteSESCredentials(context.Context) (SESCredentialStatus, error) {
	if err := model.DeleteSESCredentials(); err != nil {
		return SESCredentialStatus{}, err
	}
	if err := reloadDefaultSESProvider(); err != nil {
		return SESCredentialStatus{}, err
	}
	return GetSESCredentialStatus(context.Background())
}

func attachSESSettings(status SESCredentialStatus) SESCredentialStatus {
	status.Region = defaultSESRegion()
	status.FromAddress = defaultEmailFromRaw()
	return status
}

// ReloadDefaultSESProvider rebuilds the in-process SES provider after non-secret
// settings such as region or sender change through the options API.
func ReloadDefaultSESProvider() error {
	return reloadDefaultSESProvider()
}

func resolveSESCredentials(
	getenv func(string) string,
	loadDatabase func() (model.SESCredentials, bool, error),
) (sesCredentialResolution, error) {
	environmentCredentials := model.SESCredentials{
		AccessKeyID:     strings.TrimSpace(getenv("AWS_SES_ACCESS_KEY_ID")),
		SecretAccessKey: getenv("AWS_SES_SECRET_ACCESS_KEY"),
		SessionToken:    getenv("AWS_SES_SESSION_TOKEN"),
	}
	environmentHasAny := environmentCredentials.AccessKeyID != "" ||
		environmentCredentials.SecretAccessKey != "" ||
		environmentCredentials.SessionToken != ""
	if environmentHasAny {
		if environmentCredentials.AccessKeyID == "" || environmentCredentials.SecretAccessKey == "" {
			return sesCredentialResolution{}, ErrIncompleteSESEnvironmentCredentials
		}
		return sesCredentialResolution{
			Credentials: environmentCredentials,
			Status: SESCredentialStatus{
				Configured:      true,
				Source:          SESCredentialSourceEnvironment,
				HasSessionToken: environmentCredentials.SessionToken != "",
			},
		}, nil
	}

	databaseCredentials, found, err := loadDatabase()
	if err != nil {
		return sesCredentialResolution{}, err
	}
	if !found {
		return sesCredentialResolution{Status: SESCredentialStatus{Source: SESCredentialSourceNone}}, nil
	}
	if databaseCredentials.AccessKeyID == "" || databaseCredentials.SecretAccessKey == "" {
		return sesCredentialResolution{}, fmt.Errorf("incomplete database-managed SES credentials")
	}
	return sesCredentialResolution{
		Credentials: databaseCredentials,
		Status: SESCredentialStatus{
			Configured:      true,
			Source:          SESCredentialSourceDatabase,
			HasSessionToken: databaseCredentials.SessionToken != "",
		},
	}, nil
}
