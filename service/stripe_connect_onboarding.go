package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
)

// StartConnectOnboarding returns a Stripe-hosted onboarding URL for the user.
// If the user has no local StripeConnectAccount row, an Express account is
// created via the client and persisted. A fresh AccountLink is always created
// (each link carries a new URL).
func StartConnectOnboarding(ctx context.Context, client StripeConnectClient, userID int, email string) (string, error) {
	if !setting.StripeConnectEnabled {
		return "", errors.New("stripe connect is not enabled")
	}
	if client == nil {
		return "", errors.New("stripe connect client unavailable")
	}

	acc, err := model.GetStripeConnectAccount(userID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		created, cerr := client.CreateExpressAccount(ctx, CreateAccountParams{
			UserID:  userID,
			Email:   email,
			Country: "US",
		})
		if cerr != nil {
			return "", cerr
		}
		acc, err = model.CreateStripeConnectAccountRecord(userID, created.StripeAccountID)
		if err != nil {
			return "", err
		}
	}

	link, err := client.CreateAccountLink(ctx, acc.StripeAccountId, setting.StripeConnectReturnURL, setting.StripeConnectRefreshURL)
	if err != nil {
		return "", err
	}
	return link.URL, nil
}

// GetConnectOnboardingStatus returns the local record plus a fresh snapshot
// from Stripe. If the user has no local record, returns (nil, nil) so the
// caller can surface "not started". If the Stripe retrieve call fails, the
// last known local record is returned (the status endpoint should still show
// the cached state rather than failing).
func GetConnectOnboardingStatus(ctx context.Context, client StripeConnectClient, userID int) (*model.StripeConnectAccount, error) {
	if !setting.StripeConnectEnabled {
		return nil, errors.New("stripe connect is not enabled")
	}
	if client == nil {
		return nil, errors.New("stripe connect client unavailable")
	}

	acc, err := model.GetStripeConnectAccount(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	snapshot, serr := client.RetrieveAccount(ctx, acc.StripeAccountId)
	if serr != nil {
		common.SysError(fmt.Sprintf("stripe connect retrieve account failed: user=%d err=%s", userID, serr))
		return acc, nil
	}
	if snapshot == nil {
		return acc, nil
	}
	if uerr := model.UpdateStripeConnectAccountFromStripe(
		userID,
		snapshot.StripeAccountID,
		snapshot.Email,
		snapshot.Country,
		snapshot.PayoutsEnabled,
		snapshot.DetailsSubmitted,
		snapshot.PayoutScheduleInterval,
		snapshot.CurrentlyDueJSON,
		snapshot.EventuallyDueJSON,
	); uerr != nil {
		common.SysError(fmt.Sprintf("stripe connect update local record failed: user=%d err=%s", userID, uerr))
		return acc, nil
	}
	updated, uerr := model.GetStripeConnectAccount(userID)
	if uerr != nil {
		return acc, nil
	}
	return updated, nil
}
