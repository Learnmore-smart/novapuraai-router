package billingexpr

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeTieredQuotaAppliesFrozenGlobalDiscount(t *testing.T) {
	expr := `tier("base", p * 2)`
	snapshot := &BillingSnapshot{
		ExprString:     expr,
		ExprHash:       ExprHashString(expr),
		GroupRatio:     1,
		QuotaPerUnit:   common.QuotaPerUnit,
		GlobalDiscount: 0.5,
		ExprVersion:    1,
	}

	result, err := ComputeTieredQuota(snapshot, TokenParams{P: 1000})
	require.NoError(t, err)
	assert.Equal(t, 500.0, result.ActualQuotaBeforeGroup)
	assert.Equal(t, 500, result.ActualQuotaAfterGroup)
}
