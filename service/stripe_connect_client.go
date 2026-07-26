package service

import (
	"context"
	"fmt"
)

// StripeConnectClient 抽象所有 Stripe Connect 所需的网络调用。
// 生产实现在 stripe_connect_client_impl.go（包 stripe-go/v85）；
// 测试用 mock（stripe_connect_client_mock.go）。model/controller 不直接 import stripe-go。
type StripeConnectClient interface {
	CreateExpressAccount(ctx context.Context, params CreateAccountParams) (*AccountResult, error)
	CreateAccountLink(ctx context.Context, accountID string, returnUrl string, refreshUrl string) (*AccountLinkResult, error)
	CreateTransfer(ctx context.Context, params TransferParams, idempotencyKey string) (*TransferResult, error)
	ReverseTransfer(ctx context.Context, transferID string, amountCents int64, idempotencyKey string) (*ReversalResult, error)
	CreatePayout(ctx context.Context, params PayoutParams, idempotencyKey string, stripeAccountID string) (*PayoutResult, error)
	GetBalanceAvailableUSD(ctx context.Context, stripeAccountID string) (int64, error)
	RetrieveAccount(ctx context.Context, stripeAccountID string) (*AccountResult, error)
	ListExternalAccounts(ctx context.Context, stripeAccountID string) ([]ExternalAccount, error)
}

type CreateAccountParams struct {
	UserID  int
	Email   string
	Country string // 默认 US
}

type AccountResult struct {
	StripeAccountID        string
	Email                  string
	Country                string
	PayoutsEnabled         bool
	DetailsSubmitted       bool
	PayoutScheduleInterval string
	CurrentlyDueJSON       string
	EventuallyDueJSON      string
}

type AccountLinkResult struct {
	URL string
}

type TransferParams struct {
	AmountCents  int64
	Currency     string // "usd"
	Destination  string // acct_...
	WithdrawalID int64
	UserID       int
}

type TransferResult struct {
	ID     string // tr_...
	Amount int64
}

type ReversalResult struct {
	ID             string // trr_...
	AmountReversed int64
}

type PayoutParams struct {
	AmountCents  int64
	Currency     string // "usd"
	WithdrawalID int64
}

type PayoutResult struct {
	ID     string // po_...
	Status string // pending | in_transit | paid | failed | canceled
}

type ExternalAccount struct {
	ID       string
	IsUsable bool // default_for_currency && status=="new" or usable
}

// --- Stripe idempotency keys（spec §六）---

func TransferIdempotencyKey(withdrawalID int64) string {
	return fmt.Sprintf("withdrawal:%d:transfer", withdrawalID)
}

func PayoutIdempotencyKey(withdrawalID int64, attempt int) string {
	return fmt.Sprintf("withdrawal:%d:payout:%d", withdrawalID, attempt)
}

func ReversalIdempotencyKey(withdrawalID int64) string {
	return fmt.Sprintf("withdrawal:%d:reversal", withdrawalID)
}

func CreateAccountIdempotencyKey(userID int) string {
	return fmt.Sprintf("connect_account:%d:create", userID)
}

// AccountLink 每次需新 URL，故带 nonce；nonce 由调用方生成。
func AccountLinkIdempotencyKey(userID int, nonce string) string {
	return fmt.Sprintf("connect_account:%d:link:%s", userID, nonce)
}
