package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBalanceCreditLotTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	originalDB := DB
	originalRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:credit-lot-%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB = originalDB
		common.RedisEnabled = originalRedisEnabled
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(
		&User{},
		&TopUp{},
		&StripeTopupOrder{},
		&BalanceLedger{},
		&BalanceCreditLot{},
		&TopupPromotionCampaign{},
		&TopupPromoTier{},
		&TopupPromoRedemption{},
	))
	DB = db
	common.RedisEnabled = false
	return db
}

func TestCreditStripeTopupOrderCreatesSeparatePaidAndPromotionalLots(t *testing.T) {
	db := setupBalanceCreditLotTestDB(t)
	user := &User{Username: "lot-credit", Quota: 0, PromoQuota: 0}
	require.NoError(t, db.Create(user).Error)
	campaign := &TopupPromotionCampaign{Id: 1, Name: "launch", Enabled: true, DefaultPromoExpiryDays: 30}
	require.NoError(t, db.Create(campaign).Error)
	tier := &TopupPromoTier{CampaignID: 1, Code: "lot-tier", Name: "lot-tier", Currency: "cny", PaymentAmountMinor: 1000, BonusAmountMinor: 2000, TotalCreditAmountMinor: 3000, Enabled: true}
	require.NoError(t, db.Create(tier).Error)
	order := &StripeTopupOrder{
		OrderID:                "lot-order",
		UserId:                 user.Id,
		Status:                 StripeOrderPaid,
		PresentmentCurrency:    "cny",
		PresentmentAmountMinor: 1000,
		PaidCreditAmountMinor:  1000,
		PromoCreditAmountMinor: 2000,
		TotalCreditAmountMinor: 3000,
		FxRateSnapshot:         7.3,
		PaidCreditMicroUSD:     1_000_000,
		PromoCreditMicroUSD:    2_000_000,
		TotalCreditMicroUSD:    3_000_000,
		PaidQuota:              100,
		PromoQuota:             200,
		PromotionTierID:        tier.Id,
		PromoExpiryDays:        30,
		PromotionSnapshotJSON:  `{"applied":true}`,
	}
	require.NoError(t, db.Create(order).Error)
	require.NoError(t, ReserveTopupPromotion(order.OrderID, user.Id, tier.Id, order.PromoCreditMicroUSD))

	already, err := CreditStripeTopupOrder(order.OrderID, "cus_lot", "pi_lot", "cs_lot")
	require.NoError(t, err)
	assert.False(t, already)
	already, err = CreditStripeTopupOrder(order.OrderID, "cus_lot", "pi_lot", "cs_lot")
	require.NoError(t, err)
	assert.True(t, already)

	var lots []BalanceCreditLot
	require.NoError(t, db.Where("order_id = ?", order.OrderID).Order("balance_type asc").Find(&lots).Error)
	require.Len(t, lots, 2)
	assert.Equal(t, BalanceTypePaid, lots[0].BalanceType)
	assert.Equal(t, 100, lots[0].RemainingQuota)
	assert.Zero(t, lots[0].ExpiresAt)
	assert.Equal(t, BalanceTypePromotional, lots[1].BalanceType)
	assert.Equal(t, 200, lots[1].RemainingQuota)
	assert.Greater(t, lots[1].ExpiresAt, common.GetTimestamp()+29*24*60*60)

	var ledger []BalanceLedger
	require.NoError(t, db.Where("order_id = ?", order.OrderID).Order("id asc").Find(&ledger).Error)
	require.Len(t, ledger, 2)
	assert.Equal(t, LedgerTypeTopupPaidCredit, ledger[0].EntryType)
	assert.Equal(t, LedgerTypeTopupPromotionalBonus, ledger[1].EntryType)

	var redemption TopupPromoRedemption
	require.NoError(t, db.Where("order_id = ?", order.OrderID).First(&redemption).Error)
	assert.Equal(t, TopupPromoRedemptionIssued, redemption.Status)
}

func TestDecreaseUserQuotaConsumesPromotionalLotsByExpiryBeforePaid(t *testing.T) {
	db := setupBalanceCreditLotTestDB(t)
	user := &User{Username: "lot-spend", Quota: 300, PromoQuota: 200}
	require.NoError(t, db.Create(user).Error)
	now := time.Now().Unix()
	soon := &BalanceCreditLot{UserId: user.Id, OrderID: "promo-soon", BalanceType: BalanceTypePromotional, OriginalQuota: 100, RemainingQuota: 100, ExpiresAt: now + 60, Status: CreditLotActive, CreatedAt: now}
	later := &BalanceCreditLot{UserId: user.Id, OrderID: "promo-later", BalanceType: BalanceTypePromotional, OriginalQuota: 100, RemainingQuota: 100, ExpiresAt: now + 3600, Status: CreditLotActive, CreatedAt: now}
	paid := &BalanceCreditLot{UserId: user.Id, OrderID: "paid", BalanceType: BalanceTypePaid, OriginalQuota: 100, RemainingQuota: 100, Status: CreditLotActive, CreatedAt: now}
	require.NoError(t, db.Create(soon).Error)
	require.NoError(t, db.Create(later).Error)
	require.NoError(t, db.Create(paid).Error)

	first, err := DecreaseUserQuotaWithSplit(user.Id, 150, true)
	require.NoError(t, err)
	assert.Equal(t, QuotaWalletSplit{Promo: 150, Cash: 0, Allocations: first.Allocations}, first)
	require.NoError(t, db.First(soon, soon.Id).Error)
	require.NoError(t, db.First(later, later.Id).Error)
	require.NoError(t, db.First(paid, paid.Id).Error)
	assert.Zero(t, soon.RemainingQuota)
	assert.Equal(t, 50, later.RemainingQuota)
	assert.Equal(t, 100, paid.RemainingQuota)

	second, err := DecreaseUserQuotaWithSplit(user.Id, 100, true)
	require.NoError(t, err)
	assert.Equal(t, 50, second.Promo)
	assert.Equal(t, 50, second.Cash)
	require.NoError(t, db.First(later, later.Id).Error)
	require.NoError(t, db.First(paid, paid.Id).Error)
	assert.Zero(t, later.RemainingQuota)
	assert.Equal(t, 50, paid.RemainingQuota)
}

func TestRestoreUserQuotaSplitRestoresOriginalLot(t *testing.T) {
	db := setupBalanceCreditLotTestDB(t)
	user := &User{Username: "lot-restore", Quota: 100, PromoQuota: 100}
	require.NoError(t, db.Create(user).Error)
	lot := &BalanceCreditLot{UserId: user.Id, OrderID: "restore-promo", BalanceType: BalanceTypePromotional, OriginalQuota: 100, RemainingQuota: 100, ExpiresAt: time.Now().Add(time.Hour).Unix(), Status: CreditLotActive, CreatedAt: common.GetTimestamp()}
	require.NoError(t, db.Create(lot).Error)

	split, err := DecreaseUserQuotaWithSplit(user.Id, 60, true)
	require.NoError(t, err)
	require.NoError(t, RestoreUserQuotaSplit(user.Id, split))

	require.NoError(t, db.First(lot, lot.Id).Error)
	assert.Equal(t, 100, lot.RemainingQuota)
	var refreshed User
	require.NoError(t, db.First(&refreshed, user.Id).Error)
	assert.Equal(t, 100, refreshed.Quota)
	assert.Equal(t, 100, refreshed.PromoQuota)
}

func TestExpireUserPromotionLotsDoesNotExpirePaidBalance(t *testing.T) {
	db := setupBalanceCreditLotTestDB(t)
	user := &User{Username: "lot-expire", Quota: 150, PromoQuota: 100}
	require.NoError(t, db.Create(user).Error)
	now := common.GetTimestamp()
	promo := &BalanceCreditLot{UserId: user.Id, OrderID: "expired-promo", BalanceType: BalanceTypePromotional, OriginalQuota: 100, RemainingQuota: 100, ExpiresAt: now - 1, Status: CreditLotActive, CreatedAt: now - 100}
	paid := &BalanceCreditLot{UserId: user.Id, OrderID: "old-paid", BalanceType: BalanceTypePaid, OriginalQuota: 50, RemainingQuota: 50, ExpiresAt: 0, Status: CreditLotActive, CreatedAt: now - 100}
	require.NoError(t, db.Create(promo).Error)
	require.NoError(t, db.Create(paid).Error)

	expired, err := ExpireUserPromotionLots(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, expired)
	require.NoError(t, db.First(promo, promo.Id).Error)
	require.NoError(t, db.First(paid, paid.Id).Error)
	assert.Equal(t, CreditLotExpired, promo.Status)
	assert.Zero(t, promo.RemainingQuota)
	assert.Equal(t, CreditLotActive, paid.Status)
	assert.Equal(t, 50, paid.RemainingQuota)

	var refreshed User
	require.NoError(t, db.First(&refreshed, user.Id).Error)
	assert.Equal(t, 50, refreshed.Quota)
	assert.Zero(t, refreshed.PromoQuota)
	var expirationEntries int64
	require.NoError(t, db.Model(&BalanceLedger{}).Where("entry_type = ?", LedgerTypeExpiration).Count(&expirationEntries).Error)
	assert.Equal(t, int64(1), expirationEntries)
}
