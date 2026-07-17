package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCNYYuanToQuotaPositive(t *testing.T) {
	// 7.3 CNY = 1 USD → 200 CNY ≈ 200/7.3 USD * QuotaPerUnit
	q := CNYYuanToQuota(200, 7.3)
	require.Greater(t, q, 0)
	// round-trip roughly
	yuan := QuotaToCNYYuan(q, 7.3)
	assert.InDelta(t, 200, yuan, 1.0)
}

func TestCNYYuanToQuotaZero(t *testing.T) {
	assert.Equal(t, 0, CNYYuanToQuota(0, 7.3))
	assert.Equal(t, 0, CNYYuanToQuota(-1, 7.3))
}

func TestCNYYuanToQuotaStrict(t *testing.T) {
	_, err := CNYYuanToQuotaStrict(-1, 7.3)
	require.Error(t, err)
	q, err := CNYYuanToQuotaStrict(50, 7.3)
	require.NoError(t, err)
	assert.Greater(t, q, 0)
}
