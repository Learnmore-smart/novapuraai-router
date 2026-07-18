package emaildelivery

import (
	"context"
	"errors"
	"fmt"
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
}

type SESCredentialUpdate struct {
	AccessKeyID       string `json:"access_key_id"`
	SecretAccessKey   string `json:"secret_access_key"`
	SessionToken      string `json:"session_token"`
	ClearSessionToken bool   `json:"clear_session_token"`
}

type sesCredentialResolution struct {
	Credentials model.SESCredentials
	Status      SESCredentialStatus
}

func GetSESCredentialStatus(context.Context) (SESCredentialStatus, error) {
	resolution, err := resolveSESCredentials(os.Getenv, model.LoadSESCredentials)
	return resolution.Status, err
}

func SaveSESCredentials(_ context.Context, update SESCredentialUpdate) (SESCredentialStatus, error) {
	_, err := model.SaveSESCredentials(model.SESCredentialUpdate{
		AccessKeyID:       update.AccessKeyID,
		SecretAccessKey:   update.SecretAccessKey,
		SessionToken:      update.SessionToken,
		ClearSessionToken: update.ClearSessionToken,
	})
	if err != nil {
		return SESCredentialStatus{}, err
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
