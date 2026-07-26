package model

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

// testHelpersUserSeq generates unique suffixes for test users created by
// CreateTestUserWithBalance, keeping username/aff_code/email unique across
// tests so the unique indexes are never violated.
var testHelpersUserSeq atomic.Int64

// CreateTestUserWithBalance inserts an enabled, commission-approved user with
// CommissionBalanceCents = balanceCents and returns its id. Intended for
// model-package tests that exercise withdrawal / commission flows.
//
// Exported so that external test files (package model_test) compiled into this
// package's test binary can reuse it. Note: symbols declared in _test.go files
// are only linked into this package's own test binary, so service-package tests
// must duplicate the helper rather than call model.CreateTestUserWithBalance
// directly.
func CreateTestUserWithBalance(t testing.TB, balanceCents int64) int {
	t.Helper()
	seq := testHelpersUserSeq.Add(1)
	user := &User{
		Username:               fmt.Sprintf("th_%d", seq),
		Password:               "test-password",
		Email:                  fmt.Sprintf("th_%d@example.com", seq),
		AffCode:                fmt.Sprintf("th_aff_%d", seq),
		Role:                   common.RoleCommonUser,
		Status:                 common.UserStatusEnabled,
		CommissionApproved:     true,
		CommissionBalanceCents: balanceCents,
	}
	require.NoError(t, DB.Create(user).Error)
	return user.Id
}
