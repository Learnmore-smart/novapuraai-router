package model

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dualTestUserName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()%1_000_000_000)
}

func requireDualBalanceTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&BalanceCreditLot{}, &BalanceLedger{}))
}

// These tests require DB initialized by package tests; skip if unavailable.
func TestDecreaseUserQuotaPromoFirst(t *testing.T) {
	if DB == nil {
		t.Skip("DB not initialized")
	}
	requireDualBalanceTables(t)
	u := User{
		Username:   dualTestUserName("dual_bal"),
		Password:   "hashed",
		Role:       1,
		Status:     1,
		Quota:      1000,
		PromoQuota: 400,
	}
	require.NoError(t, DB.Create(&u).Error)
	t.Cleanup(func() { _ = DB.Unscoped().Delete(&User{}, u.Id) })

	split, err := DecreaseUserQuotaWithSplit(u.Id, 500, true)
	require.NoError(t, err)
	assert.Equal(t, 400, split.Promo)
	assert.Equal(t, 100, split.Cash)

	var got User
	require.NoError(t, DB.First(&got, u.Id).Error)
	assert.Equal(t, 500, got.Quota)
	assert.Equal(t, 0, got.PromoQuota)
}

func TestDecreaseUserQuotaInsufficient(t *testing.T) {
	if DB == nil {
		t.Skip("DB not initialized")
	}
	requireDualBalanceTables(t)
	u := User{
		Username: dualTestUserName("dual_insuf"),
		Password: "hashed",
		Role:     1,
		Status:   1,
		Quota:    50,
	}
	require.NoError(t, DB.Create(&u).Error)
	t.Cleanup(func() { _ = DB.Unscoped().Delete(&User{}, u.Id) })

	_, err := DecreaseUserQuotaWithSplit(u.Id, 100, true)
	require.Error(t, err)
}

func TestDecreaseUserQuotaConcurrentNoNegative(t *testing.T) {
	if DB == nil {
		t.Skip("DB not initialized")
	}
	requireDualBalanceTables(t)
	u := User{
		Username: dualTestUserName("dual_conc"),
		Password: "hashed",
		Role:     1,
		Status:   1,
		Quota:    500, // enough for 5 x 100
	}
	require.NoError(t, DB.Create(&u).Error)
	t.Cleanup(func() { _ = DB.Unscoped().Delete(&User{}, u.Id) })

	var wg sync.WaitGroup
	success := make(chan struct{}, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := DecreaseUserQuotaWithSplit(u.Id, 100, true); err == nil {
				success <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(success)
	n := 0
	for range success {
		n++
	}
	assert.Equal(t, 5, n)

	var got User
	require.NoError(t, DB.First(&got, u.Id).Error)
	assert.GreaterOrEqual(t, got.Quota, 0)
	assert.Equal(t, 0, got.Quota)
}
