package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"

	stripe "github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/account"
	"github.com/stripe/stripe-go/v85/accountlink"
	"github.com/stripe/stripe-go/v85/balance"
	"github.com/stripe/stripe-go/v85/bankaccount"
	"github.com/stripe/stripe-go/v85/payout"
	"github.com/stripe/stripe-go/v85/transfer"
	"github.com/stripe/stripe-go/v85/transferreversal"
)

// stripeConnectClientImpl 是 StripeConnectClient 的生产实现，包装 stripe-go/v85。
// 密钥来自 setting.StripeConnectSecretKey（env 注入）。
type stripeConnectClientImpl struct{}

// NewStripeConnectClient 返回生产实现。若 Connect 未启用返回 nil。
func NewStripeConnectClient() StripeConnectClient {
	if !setting.StripeConnectEnabled {
		return nil
	}
	stripe.Key = setting.StripeConnectSecretKey
	return &stripeConnectClientImpl{}
}

func (s *stripeConnectClientImpl) CreateExpressAccount(ctx context.Context, p CreateAccountParams) (*AccountResult, error) {
	country := p.Country
	if country == "" {
		country = "US"
	}
	params := &stripe.AccountParams{
		// v85 的 AccountControllerParams 无 Type/Profile/Dashboard/PayoutSchedule 字段；
		// Express 账户类型只能通过顶层 Type 指定（v85 中仍可用，仅被标记 deprecated）。
		Type: stripe.String("express"),
		Controller: &stripe.AccountControllerParams{
			StripeDashboard: &stripe.AccountControllerStripeDashboardParams{
				Type: stripe.String("express"),
			},
			Fees: &stripe.AccountControllerFeesParams{
				Payer: stripe.String("application"),
			},
			Losses: &stripe.AccountControllerLossesParams{
				Payments: stripe.String("application"),
			},
		},
		Capabilities: &stripe.AccountCapabilitiesParams{
			Transfers: &stripe.AccountCapabilitiesTransfersParams{
				Requested: stripe.Bool(true),
			},
		},
		Country: stripe.String(country),
		Email:   stripe.String(p.Email),
		// v85 中 payout schedule 配置位于 Settings.Payouts.Schedule，
		// 不存在 AccountControllerPayoutScheduleParams。
		Settings: &stripe.AccountSettingsParams{
			Payouts: &stripe.AccountSettingsPayoutsParams{
				Schedule: &stripe.AccountSettingsPayoutsScheduleParams{
					Interval: stripe.String("manual"),
				},
			},
		},
		Metadata: map[string]string{
			"user_id": strconv.Itoa(p.UserID),
		},
	}
	params.SetIdempotencyKey(CreateAccountIdempotencyKey(p.UserID))

	acc, err := account.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe account.New: %w", err)
	}
	return accountToResult(acc), nil
}

func (s *stripeConnectClientImpl) CreateAccountLink(ctx context.Context, accountID, returnUrl, refreshUrl string) (*AccountLinkResult, error) {
	params := &stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		Type:       stripe.String("account_onboarding"),
		ReturnURL:  stripe.String(returnUrl),
		RefreshURL: stripe.String(refreshUrl),
	}
	// AccountLink 不复用幂等键（每次需新 URL）
	link, err := accountlink.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe accountlink.New: %w", err)
	}
	return &AccountLinkResult{URL: link.URL}, nil
}

func (s *stripeConnectClientImpl) CreateTransfer(ctx context.Context, p TransferParams, idempotencyKey string) (*TransferResult, error) {
	params := &stripe.TransferParams{
		Amount:      stripe.Int64(p.AmountCents),
		Currency:    stripe.String(currencyOrDefault(p.Currency)),
		Destination: stripe.String(p.Destination),
		Metadata: map[string]string{
			"withdrawal_id": strconv.FormatInt(p.WithdrawalID, 10),
			"user_id":       strconv.Itoa(p.UserID),
		},
	}
	params.SetIdempotencyKey(idempotencyKey)
	tr, err := transfer.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe transfer.New: %w", err)
	}
	return &TransferResult{ID: tr.ID, Amount: tr.Amount}, nil
}

func (s *stripeConnectClientImpl) ReverseTransfer(ctx context.Context, transferID string, amountCents int64, idempotencyKey string) (*ReversalResult, error) {
	// v85 中 transfer 包未导出 NewReversal；reversal 接口位于独立的 transferreversal 包，
	// 调用 New(params) 时通过 params.ID 指定被反转的 transfer（拼入 URL 路径）。
	params := &stripe.TransferReversalParams{
		ID:     stripe.String(transferID),
		Amount: stripe.Int64(amountCents),
	}
	params.SetIdempotencyKey(idempotencyKey)
	rev, err := transferreversal.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe transferreversal.New: %w", err)
	}
	return &ReversalResult{ID: rev.ID, AmountReversed: rev.Amount}, nil
}

func (s *stripeConnectClientImpl) CreatePayout(ctx context.Context, p PayoutParams, idempotencyKey, stripeAccountID string) (*PayoutResult, error) {
	params := &stripe.PayoutParams{
		Amount:   stripe.Int64(p.AmountCents),
		Currency: stripe.String(currencyOrDefault(p.Currency)),
		Method:   stripe.String("standard"),
		Metadata: map[string]string{
			"withdrawal_id": strconv.FormatInt(p.WithdrawalID, 10),
		},
	}
	params.SetIdempotencyKey(idempotencyKey)
	params.SetStripeAccount(stripeAccountID)
	po, err := payout.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe payout.New: %w", err)
	}
	return &PayoutResult{ID: po.ID, Status: string(po.Status)}, nil
}

func (s *stripeConnectClientImpl) GetBalanceAvailableUSD(ctx context.Context, stripeAccountID string) (int64, error) {
	params := &stripe.BalanceParams{}
	params.SetStripeAccount(stripeAccountID)
	bal, err := balance.Get(params)
	if err != nil {
		return 0, fmt.Errorf("stripe balance.Get: %w", err)
	}
	for _, b := range bal.Available {
		if string(b.Currency) == "usd" {
			// v85 BalanceAmount 字段名为 Amount（非 Value）。
			return b.Amount, nil
		}
	}
	return 0, nil
}

func (s *stripeConnectClientImpl) RetrieveAccount(ctx context.Context, stripeAccountID string) (*AccountResult, error) {
	acc, err := account.GetByID(stripeAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe account.GetByID: %w", err)
	}
	return accountToResult(acc), nil
}

func (s *stripeConnectClientImpl) ListExternalAccounts(ctx context.Context, stripeAccountID string) ([]ExternalAccount, error) {
	// v85 bankaccount.List 通过 params.Account 拼出 /v1/accounts/{acct}/external_accounts；
	// 若只设置 SetStripeAccount 头而不给 Account，会报 "exactly one of Account or Customer need to be set"。
	// 列举 connected account 的外部账号是平台侧操作（用平台密钥请求），故设置 Account 而非 Stripe-Account 头。
	params := &stripe.BankAccountListParams{
		Account: stripe.String(stripeAccountID),
	}
	params.Limit = stripe.Int64(10)
	iter := bankaccount.List(params)
	var out []ExternalAccount
	for iter.Next() {
		ba := iter.BankAccount()
		out = append(out, ExternalAccount{
			ID:       ba.ID,
			IsUsable: ba.DefaultForCurrency && string(ba.Status) == "new",
		})
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("stripe bankaccount.List: %w", err)
	}
	return out, nil
}

func accountToResult(acc *stripe.Account) *AccountResult {
	r := &AccountResult{
		StripeAccountID:  acc.ID,
		Email:            acc.Email,
		Country:          acc.Country,
		PayoutsEnabled:   acc.PayoutsEnabled,
		DetailsSubmitted: acc.DetailsSubmitted,
	}
	// v85 的 AccountController 响应结构无 PayoutSchedule 字段，
	// payout schedule 仅能从 Settings.Payouts.Schedule 读取。
	if acc.Settings != nil && acc.Settings.Payouts != nil && acc.Settings.Payouts.Schedule != nil {
		r.PayoutScheduleInterval = string(acc.Settings.Payouts.Schedule.Interval)
	}
	if acc.Requirements != nil {
		r.CurrentlyDueJSON = mustJSON(acc.Requirements.CurrentlyDue)
		r.EventuallyDueJSON = mustJSON(acc.Requirements.EventuallyDue)
	}
	return r
}

func currencyOrDefault(c string) string {
	if c == "" {
		return "usd"
	}
	return c
}

func mustJSON(v any) string {
	if v == nil {
		return "[]"
	}
	b, err := common.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}
