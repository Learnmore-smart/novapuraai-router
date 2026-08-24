package service

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service/deepseekfairuse"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSubscriptionFirstUsesAllModelRecurringEntitlementAndScopedWalletFallback(t *testing.T) {
	originalDB := model.DB
	originalDatabaseType := common.MainDatabaseType()
	originalRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:billing-model-scope-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = originalDB
		common.SetMainDatabaseType(originalDatabaseType)
		common.RedisEnabled = originalRedisEnabled
		_ = sqlDB.Close()
	})
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.BalanceCreditLot{},
		&model.BalanceLedger{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.SubscriptionPreConsumeRecord{},
	))

	const userID = 9201
	const planID = 9202
	require.NoError(t, db.Create(&model.User{
		Id:       userID,
		Username: "billing-model-scope",
		Quota:    1_000,
		Status:   common.UserStatusEnabled,
	}).Error)
	allowWalletOverflow := false
	require.NoError(t, db.Create(&model.SubscriptionPlan{
		Id:                      planID,
		Title:                   "DeepSeek only",
		DurationUnit:            model.SubscriptionDurationMonth,
		DurationValue:           1,
		StripeSubscriptionModel: deepseekfairuse.DeepSeekV4Flash0731Model,
		AllowWalletOverflow:     &allowWalletOverflow,
	}).Error)
	now := time.Now().Unix()
	require.NoError(t, db.Create(&model.UserSubscription{
		UserId:              userID,
		PlanId:              planID,
		AmountTotal:         0,
		StartTime:           now - 60,
		EndTime:             now + 3600,
		Status:              "active",
		AllowWalletOverflow: false,
	}).Error)

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		RequestId:       "wallet-other-model",
		UserId:          userID,
		OriginModelName: "gpt-5",
		IsPlayground:    true,
		UserSetting: dto.UserSetting{
			BillingPreference: "subscription_first",
		},
	}
	session, apiErr := NewBillingSession(c, info, 10)
	require.Nil(t, apiErr)
	require.NotNil(t, session)
	assert.Equal(t, BillingSourceWallet, session.funding.Source())
	require.NoError(t, session.Settle(10))

	var user model.User
	require.NoError(t, db.Select("quota").First(&user, userID).Error)
	assert.Equal(t, 990, user.Quota)

	const recurringUserID = 9203
	const recurringPlanID = 9204
	recurringCode := "all-model-recurring"
	require.NoError(t, db.Create(&model.User{
		Id:       recurringUserID,
		Username: "billing-all-model",
		AffCode:  "billing-all-model-aff",
		Quota:    1_000,
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.SubscriptionPlan{
		Id:                      recurringPlanID,
		Code:                    recurringCode,
		RecurringCode:           &recurringCode,
		Title:                   "All model recurring",
		DurationUnit:            model.SubscriptionDurationMonth,
		DurationValue:           1,
		StripeSubscriptionModel: deepseekfairuse.DeepSeekV4Flash0731Model,
		AllowWalletOverflow:     &allowWalletOverflow,
	}).Error)
	now = time.Now().Unix()
	require.NoError(t, db.Create(&model.UserSubscription{
		UserId:              recurringUserID,
		PlanId:              recurringPlanID,
		AmountTotal:         0,
		StartTime:           now - 60,
		EndTime:             now + 3600,
		Status:              "active",
		AllowWalletOverflow: false,
	}).Error)

	for _, modelName := range []string{"gpt-5", "claude-sonnet-4"} {
		c, _ = gin.CreateTestContext(httptest.NewRecorder())
		info = &relaycommon.RelayInfo{
			RequestId:       "subscription-" + modelName,
			UserId:          recurringUserID,
			OriginModelName: modelName,
			IsPlayground:    true,
			UserSetting: dto.UserSetting{
				BillingPreference: "subscription_first",
			},
		}
		session, apiErr = NewBillingSession(c, info, 10)
		require.Nil(t, apiErr)
		require.NotNil(t, session)
		assert.Equal(t, BillingSourceSubscription, session.funding.Source())
		require.NoError(t, session.Settle(10))
	}

	require.NoError(t, db.Select("quota").First(&user, recurringUserID).Error)
	assert.Equal(t, 1_000, user.Quota)
}
