package model

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// StripeConnectAccount 1:1 关联用户与 Stripe Connect Express 账户。
// 独立于 WithdrawalRequest：开户与打款解耦。
type StripeConnectAccount struct {
	ID                     int64  `json:"id" gorm:"primaryKey"`
	UserId                 int    `json:"user_id" gorm:"column:user_id;uniqueIndex"`
	StripeAccountId        string `json:"stripe_account_id" gorm:"column:stripe_account_id;type:varchar(64);uniqueIndex"`
	Email                  string `json:"email" gorm:"column:email;type:varchar(255)"`
	Country                string `json:"country" gorm:"column:country;type:varchar(8)"`
	PayoutsEnabled         bool   `json:"payouts_enabled" gorm:"column:payouts_enabled"`
	DetailsSubmitted       bool   `json:"details_submitted" gorm:"column:details_submitted"`
	PayoutScheduleInterval string `json:"payout_schedule_interval" gorm:"column:payout_schedule_interval;type:varchar(16)"`
	CurrentlyDueJSON       string `json:"currently_due" gorm:"column:currently_due;type:text"`
	EventuallyDueJSON      string `json:"eventually_due" gorm:"column:eventually_due;type:text"`
	OnboardingState        string `json:"onboarding_state" gorm:"column:onboarding_state;type:varchar(32);index"`
	CreatedAt              int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt              int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (StripeConnectAccount) TableName() string { return "stripe_connect_accounts" }

const (
	ConnectOnboardingCreated    = "created"
	ConnectOnboardingOnboarding = "onboarding"
	ConnectOnboardingEnabled    = "enabled"
	ConnectOnboardingRestricted = "restricted"
	ConnectOnboardingRejected   = "rejected"
)

// GetOrCreateStripeConnectAccount 返回用户的连接账户记录；若无则用给定 stripeAccountId 建一条。
func GetOrCreateStripeConnectAccount(userId int, stripeAccountId string) (*StripeConnectAccount, error) {
	if userId <= 0 || stripeAccountId == "" {
		return nil, errors.New("invalid user id or stripe account id")
	}
	var acc StripeConnectAccount
	err := DB.Where("user_id = ?", userId).First(&acc).Error
	if err == nil {
		return &acc, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	acc = StripeConnectAccount{
		UserId:                 userId,
		StripeAccountId:        stripeAccountId,
		OnboardingState:        ConnectOnboardingCreated,
		PayoutScheduleInterval: "manual",
	}
	if e := DB.Create(&acc).Error; e != nil {
		// 并发下可能已被创建；重读
		var acc2 StripeConnectAccount
		if e2 := DB.Where("user_id = ?", userId).First(&acc2).Error; e2 == nil {
			return &acc2, nil
		}
		return nil, e
	}
	return &acc, nil
}

// GetStripeConnectAccount 返回用户的连接账户记录；不存在返回 gorm.ErrRecordNotFound。
func GetStripeConnectAccount(userId int) (*StripeConnectAccount, error) {
	var acc StripeConnectAccount
	if err := DB.Where("user_id = ?", userId).First(&acc).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

// CreateStripeConnectAccountRecord 用给定 stripeAccountId 建一条记录（并发安全：已存在则返回旧记录）。
func CreateStripeConnectAccountRecord(userId int, stripeAccountId string) (*StripeConnectAccount, error) {
	acc := StripeConnectAccount{
		UserId:                 userId,
		StripeAccountId:        stripeAccountId,
		OnboardingState:        ConnectOnboardingCreated,
		PayoutScheduleInterval: "manual",
	}
	if e := DB.Create(&acc).Error; e != nil {
		var existing StripeConnectAccount
		if e2 := DB.Where("user_id = ?", userId).First(&existing).Error; e2 == nil {
			return &existing, nil
		}
		return nil, e
	}
	return &acc, nil
}

// UpdateStripeConnectAccountFromStripe 用 account.updated 的快照更新本地记录。
// payoutsEnabled/detailsSubmitted/interval/currentlyDue 均来自 Stripe。
func UpdateStripeConnectAccountFromStripe(userId int, stripeAccountId string, email string, country string,
	payoutsEnabled bool, detailsSubmitted bool, interval string, currentlyDueJSON string, eventuallyDueJSON string) error {
	state := ConnectOnboardingOnboarding
	if payoutsEnabled && detailsSubmitted {
		state = ConnectOnboardingEnabled
	} else if currentlyDueJSON != "" && currentlyDueJSON != "[]" {
		state = ConnectOnboardingRestricted
	}
	updates := map[string]any{
		"stripe_account_id":        stripeAccountId,
		"email":                    email,
		"country":                  country,
		"payouts_enabled":          payoutsEnabled,
		"details_submitted":        detailsSubmitted,
		"payout_schedule_interval": interval,
		"currently_due":            currentlyDueJSON,
		"eventually_due":           eventuallyDueJSON,
		"onboarding_state":         state,
	}
	if interval != "" && interval != "manual" {
		common.SysError("stripe connect account payout_schedule not manual: user=" + strconv.Itoa(userId) + " interval=" + interval)
	}
	return DB.Model(&StripeConnectAccount{}).Where("user_id = ?", userId).Updates(updates).Error
}

func GetStripeConnectAccountByStripeId(stripeAccountId string) (*StripeConnectAccount, error) {
	var acc StripeConnectAccount
	if err := DB.Where("stripe_account_id = ?", stripeAccountId).First(&acc).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}
