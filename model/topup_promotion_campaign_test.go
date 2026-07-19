package model

import (
	"fmt"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTopupPromotionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	originalDB := DB
	dsn := fmt.Sprintf("file:topup-promotion-%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB = originalDB
		require.NoError(t, sqlDB.Close())
	})
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&TopupPromotionCampaign{},
		&TopupPromoTier{},
		&TopupPromoRedemption{},
	))
	DB = db
	return db
}

func TestSeedLaunchTopupPromotionCreatesSharedRepeatableBands(t *testing.T) {
	db := setupTopupPromotionTestDB(t)
	require.NoError(t, SeedLaunchTopupPromotion(db))
	require.NoError(t, SeedLaunchTopupPromotion(db))

	campaign, err := GetTopupPromotionCampaign()
	require.NoError(t, err)
	assert.True(t, campaign.Enabled)
	assert.Zero(t, campaign.PerUserLimit)
	assert.Zero(t, campaign.GlobalBudgetMicroUSD)
	assert.Zero(t, campaign.DefaultPromoExpiryDays)

	var tiers []TopupPromoTier
	require.NoError(t, db.Where("campaign_id = ? AND currency = ? AND payment_amount_minor = ?", campaign.Id, "*", 0).Order("sort_order asc").Find(&tiers).Error)
	require.Len(t, tiers, 6)

	want := [][3]int{
		{10, 19, 20000},
		{20, 49, 30000},
		{50, 99, 40000},
		{100, 199, 50000},
		{200, 499, 60000},
		{500, 0, 70000},
	}
	for i, tier := range tiers {
		assert.True(t, tier.Enabled)
		assert.Zero(t, tier.PerUserLimit)
		assert.Equal(t, want[i][0], tier.MinPresentmentMajor)
		assert.Equal(t, want[i][1], tier.MaxPresentmentMajor)
		assert.Equal(t, want[i][2], tier.PercentBonusBps)
	}
}

func TestReserveTopupPromotionTreatsZeroAsUnlimitedAndPositiveAsMaximum(t *testing.T) {
	db := setupTopupPromotionTestDB(t)
	campaign := &TopupPromotionCampaign{Id: 1, Name: "test", Enabled: true, DefaultPromoExpiryDays: 30}
	require.NoError(t, db.Create(campaign).Error)
	tier := &TopupPromoTier{CampaignID: campaign.Id, Code: "repeat", Name: "repeat", Currency: "cny", PaymentAmountMinor: 1000, BonusAmountMinor: 2000, TotalCreditAmountMinor: 3000, Enabled: true}
	require.NoError(t, db.Create(tier).Error)

	for i := 0; i < 3; i++ {
		require.NoError(t, ReserveTopupPromotion(fmt.Sprintf("unlimited-%d", i), 7, tier.Id, 2_000_000))
	}

	campaign.PerUserLimit = 5
	require.NoError(t, db.Save(campaign).Error)
	require.NoError(t, ReserveTopupPromotion("limited-4", 7, tier.Id, 2_000_000))
	require.NoError(t, ReserveTopupPromotion("limited-5", 7, tier.Id, 2_000_000))
	require.ErrorIs(t, ReserveTopupPromotion("limited-6", 7, tier.Id, 2_000_000), ErrTopupPromotionUserLimit)
}

func TestReserveTopupPromotionBudgetReleaseAndIssueAreIdempotent(t *testing.T) {
	db := setupTopupPromotionTestDB(t)
	campaign := &TopupPromotionCampaign{Id: 1, Name: "budget", Enabled: true, GlobalBudgetMicroUSD: 10_000_000, DefaultPromoExpiryDays: 30}
	require.NoError(t, db.Create(campaign).Error)
	tier := &TopupPromoTier{CampaignID: campaign.Id, Code: "budget-tier", Name: "budget-tier", Currency: "cny", PaymentAmountMinor: 1000, BonusAmountMinor: 2000, TotalCreditAmountMinor: 3000, Enabled: true}
	require.NoError(t, db.Create(tier).Error)

	require.NoError(t, ReserveTopupPromotion("order-a", 1, tier.Id, 6_000_000))
	require.ErrorIs(t, ReserveTopupPromotion("order-b", 2, tier.Id, 6_000_000), ErrTopupPromotionBudgetReached)
	require.NoError(t, ReleaseTopupPromotion("order-a"))
	require.NoError(t, ReleaseTopupPromotion("order-a"))
	require.NoError(t, ReserveTopupPromotion("order-b", 2, tier.Id, 6_000_000))
	require.NoError(t, IssueTopupPromotion("order-b"))
	require.NoError(t, IssueTopupPromotion("order-b"))

	require.NoError(t, db.First(campaign, campaign.Id).Error)
	assert.Zero(t, campaign.ReservedPromoMicroUSD)
	assert.Equal(t, int64(6_000_000), campaign.IssuedPromoMicroUSD)
}

func TestConcurrentTopupPromotionReservationsCannotExceedBudget(t *testing.T) {
	db := setupTopupPromotionTestDB(t)
	campaign := &TopupPromotionCampaign{Id: 1, Name: "concurrent", Enabled: true, GlobalBudgetMicroUSD: 6_000_000, DefaultPromoExpiryDays: 30}
	require.NoError(t, db.Create(campaign).Error)
	tier := &TopupPromoTier{CampaignID: campaign.Id, Code: "concurrent-tier", Name: "concurrent-tier", Currency: "cny", PaymentAmountMinor: 1000, BonusAmountMinor: 2000, TotalCreditAmountMinor: 3000, Enabled: true}
	require.NoError(t, db.Create(tier).Error)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs <- ReserveTopupPromotion(fmt.Sprintf("concurrent-%d", i), i+1, tier.Id, 6_000_000)
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	budgetFailures := 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case assert.ErrorIs(t, err, ErrTopupPromotionBudgetReached):
			budgetFailures++
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, budgetFailures)
}
