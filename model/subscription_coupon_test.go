package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createCouponForTest inserts an enabled SubscriptionCoupon and returns it.
// Centralizing the insert keeps the lifecycle tests focused on the
// reserve/issue/release/reverse transitions rather than re-building the row
// shape in every case.
func createCouponForTest(t *testing.T, code string, overrides ...func(*SubscriptionCoupon)) *SubscriptionCoupon {
	t.Helper()
	coupon := &SubscriptionCoupon{
		Code:           code,
		Name:           code,
		StripeCouponId: "stripe_" + code,
		PercentOff:     20,
		DurationMonths: 1,
		Enabled:        true,
	}
	for _, fn := range overrides {
		fn(coupon)
	}
	require.NoError(t, DB.Create(coupon).Error)
	return coupon
}

// loadRedemptionByOrder reloads a SubscriptionCouponRedemption by its order_id
// so each lifecycle test can assert on the persisted Status / timestamps
// without re-querying through the function under test.
func loadRedemptionByOrder(t *testing.T, orderId string) SubscriptionCouponRedemption {
	t.Helper()
	var rd SubscriptionCouponRedemption
	require.NoError(t, DB.Where("order_id = ?", orderId).First(&rd).Error)
	return rd
}

// loadCoupon reloads a SubscriptionCoupon by id so tests can assert on
// TimesRedeemed after a reserve/release/reverse operation.
func loadCoupon(t *testing.T, id int64) SubscriptionCoupon {
	t.Helper()
	var c SubscriptionCoupon
	require.NoError(t, DB.First(&c, id).Error)
	return c
}

// ---------------------------------------------------------------------------
// ReserveSubscriptionCouponWithTx
// ---------------------------------------------------------------------------

// TestReserveSubscriptionCouponWithTx_ArgumentValidation guards the entry
// validation: nil tx, zero couponId / userId / planId, and empty orderId are
// rejected before any DB read.
func TestReserveSubscriptionCouponWithTx_ArgumentValidation(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "ARGVAL")

	tests := []struct {
		name     string
		tx       *gorm.DB
		couponId int64
		userId   int
		planId   int
		orderId  string
	}{
		{"nil tx rejected", nil, coupon.Id, 1, 1, "order-1"},
		{"zero couponId rejected", DB, 0, 1, 1, "order-1"},
		{"zero userId rejected", DB, coupon.Id, 0, 1, "order-1"},
		{"zero planId rejected", DB, coupon.Id, 1, 0, "order-1"},
		{"empty orderId rejected", DB, coupon.Id, 1, 1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReserveSubscriptionCouponWithTx(tt.tx, tt.couponId, tt.userId, tt.planId, tt.orderId, 20, 1000, 200, 800, "USD")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid subscription coupon reservation")
		})
	}
}

// TestReserveSubscriptionCouponWithTx_HappyPath verifies the happy path:
// a redemption row is inserted in "reserved" status, TimesRedeemed is bumped,
// and the persisted amounts match the inputs.
func TestReserveSubscriptionCouponWithTx_HappyPath(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "HAPPY")

	var redemption *SubscriptionCouponRedemption
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		rd, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9001, 7001, "order-happy", 20, 1000, 200, 800, "USD")
		redemption = rd
		return err
	}))
	require.NotNil(t, redemption)
	assert.Equal(t, CouponRedemptionStatusReserved, redemption.Status)
	assert.Equal(t, coupon.Id, redemption.CouponId)
	assert.Equal(t, 9001, redemption.UserId)
	assert.Equal(t, 7001, redemption.PlanId)
	assert.Equal(t, 20, redemption.PercentOff)
	assert.Equal(t, int64(1000), redemption.OriginalAmount)
	assert.Equal(t, int64(200), redemption.DiscountAmount)
	assert.Equal(t, int64(800), redemption.FinalAmount)
	assert.Equal(t, "USD", redemption.Currency)

	// TimesRedeemed must be bumped exactly once.
	refreshed := loadCoupon(t, coupon.Id)
	assert.Equal(t, 1, refreshed.TimesRedeemed)
}

// TestReserveSubscriptionCouponWithTx_Idempotent verifies that re-reserving
// the same orderId returns the existing redemption without creating a second
// row or re-bumping TimesRedeemed. This is the contract checkout retries rely on.
func TestReserveSubscriptionCouponWithTx_Idempotent(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "IDEMP")

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9002, 7002, "order-idemp", 20, 1000, 200, 800, "USD")
		return err
	}))

	var second *SubscriptionCouponRedemption
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		rd, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9002, 7002, "order-idemp", 20, 1000, 200, 800, "USD")
		second = rd
		return err
	}))
	require.NotNil(t, second)
	assert.Equal(t, "order-idemp", second.OrderId)

	// Only one redemption row, TimesRedeemed still 1.
	var count int64
	require.NoError(t, DB.Model(&SubscriptionCouponRedemption{}).Where("order_id = ?", "order-idemp").Count(&count).Error)
	assert.EqualValues(t, 1, count)
	assert.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed)
}

// TestReserveSubscriptionCouponWithTx_RejectsDuplicateOrderWithDifferentTerms
// guards against a stale order id being reused with different coupon/user
// terms: the second reservation must fail loudly rather than silently
// overwriting the first redemption's coupon_id / user_id.
func TestReserveSubscriptionCouponWithTx_RejectsDuplicateOrderWithDifferentTerms(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "DUP1")
	coupon2 := createCouponForTest(t, "DUP2")

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9003, 7003, "order-dup", 20, 1000, 200, 800, "USD")
		return err
	}))

	err := DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon2.Id, 9003, 7003, "order-dup", 20, 1000, 200, 800, "USD")
		return err
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already reserved with different terms")
}

// TestReserveSubscriptionCouponWithTx_RevalidatesEligibility verifies that
// the in-transaction re-validation catches state changes that happened
// between the read-only ValidateSubscriptionCoupon call and the reservation.
// This is the defense against the TOCTOU race where a coupon is disabled or
// its usage cap is reached by a concurrent request.
func TestReserveSubscriptionCouponWithTx_RevalidatesEligibility(t *testing.T) {
	truncateTables(t)

	tests := []struct {
		name      string
		setup     func(*SubscriptionCoupon)
		wantErrIs error
	}{
		{
			name:      "disabled coupon rejected",
			setup:     func(c *SubscriptionCoupon) { c.Enabled = false },
			wantErrIs: ErrSubscriptionCouponDisabled,
		},
		{
			name:      "expired coupon rejected",
			setup:     func(c *SubscriptionCoupon) { c.EndAt = GetDBTimestamp() - 86400 },
			wantErrIs: ErrSubscriptionCouponExpired,
		},
		{
			name:      "usage cap reached rejected",
			setup:     func(c *SubscriptionCoupon) { c.MaxRedemptions = 1; c.TimesRedeemed = 1 },
			wantErrIs: ErrSubscriptionCouponUsageCapReached,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncateTables(t)
			coupon := createCouponForTest(t, "REVAL_"+tt.name, tt.setup)

			err := DB.Transaction(func(tx *gorm.DB) error {
				_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9100, 7100, "order-reval-"+tt.name, 20, 1000, 200, 800, "USD")
				return err
			})
			require.ErrorIs(t, err, tt.wantErrIs)

			// No redemption row must be created on rejection.
			var count int64
			require.NoError(t, DB.Model(&SubscriptionCouponRedemption{}).Where("coupon_id = ?", coupon.Id).Count(&count).Error)
			assert.EqualValues(t, 0, count)
		})
	}
}

// TestReserveSubscriptionCouponWithTx_PerUserLimitReachedInsideTx verifies
// that the per-user limit is enforced against active redemptions inside the
// transaction. A reserved redemption for the same user blocks a second
// reservation even when TimesRedeemed is below the global cap.
func TestReserveSubscriptionCouponWithTx_PerUserLimitReachedInsideTx(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "PERUSER2", func(c *SubscriptionCoupon) {
		c.PerUserLimit = 1
	})

	// First reservation succeeds.
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9200, 7200, "order-peruser2-1", 20, 1000, 200, 800, "USD")
		return err
	}))

	// Second reservation by the same user must fail with per-user limit.
	err := DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9200, 7200, "order-peruser2-2", 20, 1000, 200, 800, "USD")
		return err
	})
	require.ErrorIs(t, err, ErrSubscriptionCouponPerUserLimitReached)

	// TimesRedeemed stays at 1 (the failed second reservation did not bump it).
	assert.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed)
}

// ---------------------------------------------------------------------------
// IssueSubscriptionCouponWithTx
// ---------------------------------------------------------------------------

// TestIssueSubscriptionCouponWithTx_HappyAndIdempotent verifies that issuing a
// reserved redemption transitions it to "issued" and sets IssuedAt, and that
// a second call is a no-op (status stays issued, no error).
func TestIssueSubscriptionCouponWithTx_HappyAndIdempotent(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "ISSUE")
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9300, 7300, "order-issue", 20, 1000, 200, 800, "USD")
		return err
	}))

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return IssueSubscriptionCouponWithTx(tx, "order-issue")
	}))
	rd := loadRedemptionByOrder(t, "order-issue")
	assert.Equal(t, CouponRedemptionStatusIssued, rd.Status)
	assert.Greater(t, rd.IssuedAt, int64(0))

	// Idempotent: re-issuing is a no-op.
	preIssuedAt := rd.IssuedAt
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return IssueSubscriptionCouponWithTx(tx, "order-issue")
	}))
	rd = loadRedemptionByOrder(t, "order-issue")
	assert.Equal(t, CouponRedemptionStatusIssued, rd.Status)
	assert.Equal(t, preIssuedAt, rd.IssuedAt, "re-issue must not overwrite IssuedAt")
}

// TestIssueSubscriptionCouponWithTx_MissingRedemptionIsNoOp verifies that
// issuing an order id with no redemption row returns nil (the checkout
// completed handler calls this unconditionally, so a no-coupon order must not
// error).
func TestIssueSubscriptionCouponWithTx_MissingRedemptionIsNoOp(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return IssueSubscriptionCouponWithTx(tx, "order-without-coupon")
	}))
}

// TestIssueSubscriptionCouponWithTx_RejectsNonReserved verifies that only a
// reserved redemption can be issued. A released redemption (e.g. checkout
// expired before payment) must not be re-issued by a late checkout completion.
func TestIssueSubscriptionCouponWithTx_RejectsNonReserved(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "ISSUE_REJECT")
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9301, 7301, "order-issue-reject", 20, 1000, 200, 800, "USD")
		return err
	}))
	// Move it to released first.
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseSubscriptionCouponWithTx(tx, "order-issue-reject")
	}))

	err := DB.Transaction(func(tx *gorm.DB) error {
		return IssueSubscriptionCouponWithTx(tx, "order-issue-reject")
	})
	require.ErrorIs(t, err, ErrSubscriptionCouponRedemptionNotReserved)
}

// TestIssueSubscriptionCouponWithTx_ArgumentValidation guards the entry
// validation.
func TestIssueSubscriptionCouponWithTx_ArgumentValidation(t *testing.T) {
	truncateTables(t)
	require.EqualError(t, IssueSubscriptionCouponWithTx(nil, "order-x"), "tx is nil")
	require.EqualError(t, IssueSubscriptionCouponWithTx(DB, ""), "orderId is empty")
}

// ---------------------------------------------------------------------------
// ReleaseSubscriptionCouponWithTx
// ---------------------------------------------------------------------------

// TestReleaseSubscriptionCouponWithTx_HappyAndIdempotent verifies that
// releasing a reserved redemption transitions it to "released", sets
// ReleasedAt, and decrements TimesRedeemed (the reservation never issued, so
// it must not count as a redemption). A second release is a no-op.
func TestReleaseSubscriptionCouponWithTx_HappyAndIdempotent(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "REL")
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9400, 7400, "order-rel", 20, 1000, 200, 800, "USD")
		return err
	}))
	require.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseSubscriptionCouponWithTx(tx, "order-rel")
	}))
	rd := loadRedemptionByOrder(t, "order-rel")
	assert.Equal(t, CouponRedemptionStatusReleased, rd.Status)
	assert.Greater(t, rd.ReleasedAt, int64(0))
	assert.Equal(t, 0, loadCoupon(t, coupon.Id).TimesRedeemed, "release must decrement TimesRedeemed")

	// Idempotent: re-releasing is a no-op, TimesRedeemed stays at 0.
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseSubscriptionCouponWithTx(tx, "order-rel")
	}))
	assert.Equal(t, 0, loadCoupon(t, coupon.Id).TimesRedeemed)
}

// TestReleaseSubscriptionCouponWithTx_MissingRedemptionIsNoOp verifies that
// releasing an order id with no redemption row is a no-op (the checkout
// expired handler calls this unconditionally).
func TestReleaseSubscriptionCouponWithTx_MissingRedemptionIsNoOp(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseSubscriptionCouponWithTx(tx, "order-without-coupon")
	}))
}

// TestReleaseSubscriptionCouponWithTx_RejectsIssued verifies that an issued
// redemption cannot be released (it must be reversed). This guards the
// lifecycle invariant: issued -> released is not a legal transition.
func TestReleaseSubscriptionCouponWithTx_RejectsIssued(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "REL_ISSUED")
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9401, 7401, "order-rel-issued", 20, 1000, 200, 800, "USD")
		return err
	}))
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return IssueSubscriptionCouponWithTx(tx, "order-rel-issued")
	}))

	err := DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseSubscriptionCouponWithTx(tx, "order-rel-issued")
	})
	require.ErrorIs(t, err, ErrSubscriptionCouponRedemptionNotReserved)
	// TimesRedeemed must remain 1 (issue does not decrement; release rejected).
	assert.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed)
}

// TestReleaseSubscriptionCouponWithTx_TimesRedeemedNeverNegative guards the
// billing-safety invariant that TimesRedeemed never goes negative. We force a
// negative starting state and confirm ReleaseSubscriptionCouponWithTx clamps
// to zero rather than underflowing.
func TestReleaseSubscriptionCouponWithTx_TimesRedeemedNeverNegative(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "REL_CLAMP", func(c *SubscriptionCoupon) {
		c.TimesRedeemed = 0
	})
	// Insert a reserved redemption directly so TimesRedeemed is 0 at release
	// time (simulating a coupon row whose TimesRedeemed was already 0 due to a
	// prior concurrent release that raced the reservation).
	now := GetDBTimestamp()
	require.NoError(t, DB.Create(&SubscriptionCouponRedemption{
		OrderId: "order-clamp", CouponId: coupon.Id, UserId: 9402, PlanId: 7402,
		Status: CouponRedemptionStatusReserved, PercentOff: 20,
		OriginalAmount: 1000, DiscountAmount: 200, FinalAmount: 800,
		Currency: "USD", CreatedAt: now, UpdatedAt: now,
	}).Error)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseSubscriptionCouponWithTx(tx, "order-clamp")
	}))
	assert.Equal(t, 0, loadCoupon(t, coupon.Id).TimesRedeemed, "TimesRedeemed must clamp at 0, not go negative")
}

// ---------------------------------------------------------------------------
// ReverseSubscriptionCouponWithTx
// ---------------------------------------------------------------------------

// TestReverseSubscriptionCouponWithTx_HappyAndIdempotent verifies that
// reversing an issued redemption transitions it to "reversed", sets
// ReversedAt, and does NOT decrement TimesRedeemed (the redemption did
// happen, it is just being reversed for accounting). A second reverse is a
// no-op.
func TestReverseSubscriptionCouponWithTx_HappyAndIdempotent(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "REV")
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9500, 7500, "order-rev", 20, 1000, 200, 800, "USD")
		return err
	}))
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return IssueSubscriptionCouponWithTx(tx, "order-rev")
	}))
	require.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReverseSubscriptionCouponWithTx(tx, "order-rev", "refund issued")
	}))
	rd := loadRedemptionByOrder(t, "order-rev")
	assert.Equal(t, CouponRedemptionStatusReversed, rd.Status)
	assert.Greater(t, rd.ReversedAt, int64(0))
	// TimesRedeemed must NOT be decremented on reverse.
	assert.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed)

	// Idempotent: re-reversing is a no-op.
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReverseSubscriptionCouponWithTx(tx, "order-rev", "duplicate")
	}))
	rd = loadRedemptionByOrder(t, "order-rev")
	assert.Equal(t, CouponRedemptionStatusReversed, rd.Status)
}

// TestReverseSubscriptionCouponWithTx_MissingRedemptionIsNoOp verifies that
// reversing an order id with no redemption row is a no-op.
func TestReverseSubscriptionCouponWithTx_MissingRedemptionIsNoOp(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReverseSubscriptionCouponWithTx(tx, "order-without-coupon", "refund")
	}))
}

// TestReverseSubscriptionCouponWithTx_RejectsNonIssued verifies that only an
// issued redemption can be reversed. A reserved redemption (still pending
// payment) must not be reversed — it must be released instead.
func TestReverseSubscriptionCouponWithTx_RejectsNonIssued(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "REV_RESERVED")
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9501, 7501, "order-rev-reserved", 20, 1000, 200, 800, "USD")
		return err
	}))

	err := DB.Transaction(func(tx *gorm.DB) error {
		return ReverseSubscriptionCouponWithTx(tx, "order-rev-reserved", "refund")
	})
	require.ErrorIs(t, err, ErrSubscriptionCouponRedemptionNotIssued)
}

// ---------------------------------------------------------------------------
// ReleaseSubscriptionCouponRedemption (thin wrapper)
// ---------------------------------------------------------------------------

// TestReleaseSubscriptionCouponRedemption_WrapperOpensOwnTransaction
// verifies the public wrapper opens its own transaction and behaves the same
// as the WithTx variant. This is the path the checkout-failure handler uses.
func TestReleaseSubscriptionCouponRedemption_WrapperOpensOwnTransaction(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "WRAP")
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9600, 7600, "order-wrap", 20, 1000, 200, 800, "USD")
		return err
	}))
	require.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed)

	require.NoError(t, ReleaseSubscriptionCouponRedemption("order-wrap"))
	assert.Equal(t, CouponRedemptionStatusReleased, loadRedemptionByOrder(t, "order-wrap").Status)
	assert.Equal(t, 0, loadCoupon(t, coupon.Id).TimesRedeemed)
}

// TestReleaseSubscriptionCouponRedemption_EmptyOrderIdRejected guards the
// wrapper's entry validation.
func TestReleaseSubscriptionCouponRedemption_EmptyOrderIdRejected(t *testing.T) {
	truncateTables(t)
	require.EqualError(t, ReleaseSubscriptionCouponRedemption(""), "orderId is empty")
}

// ---------------------------------------------------------------------------
// Full lifecycle: reserve -> issue -> reverse (and reserve -> release)
// ---------------------------------------------------------------------------

// TestSubscriptionCouponLifecycle_FullReserveIssueReverse covers the
// successful-payment path end-to-end: reserve at checkout, issue on payment,
// reverse on refund. TimesRedeemed goes 0 -> 1 (reserve) -> 1 (issue, no-op)
// -> 1 (reverse, no decrement). This is the contract the webhook handlers
// (checkout.completed, customer.subscription.deleted) rely on.
func TestSubscriptionCouponLifecycle_FullReserveIssueReverse(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "LIFE")

	// reserve
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9700, 7700, "order-life", 20, 1000, 200, 800, "USD")
		return err
	}))
	assert.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed)
	assert.Equal(t, CouponRedemptionStatusReserved, loadRedemptionByOrder(t, "order-life").Status)

	// issue
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return IssueSubscriptionCouponWithTx(tx, "order-life")
	}))
	assert.Equal(t, CouponRedemptionStatusIssued, loadRedemptionByOrder(t, "order-life").Status)
	assert.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed, "issue must not bump TimesRedeemed")

	// reverse
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReverseSubscriptionCouponWithTx(tx, "order-life", "refund")
	}))
	assert.Equal(t, CouponRedemptionStatusReversed, loadRedemptionByOrder(t, "order-life").Status)
	assert.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed, "reverse must not decrement TimesRedeemed")
}

// TestSubscriptionCouponLifecycle_FullReserveRelease covers the
// abandoned-checkout path: reserve at checkout, release on checkout expire.
// TimesRedeemed goes 0 -> 1 (reserve) -> 0 (release decrements). This is the
// contract the checkout.session.expired webhook handler relies on.
func TestSubscriptionCouponLifecycle_FullReserveRelease(t *testing.T) {
	truncateTables(t)
	coupon := createCouponForTest(t, "LIFE_REL")

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := ReserveSubscriptionCouponWithTx(tx, coupon.Id, 9701, 7701, "order-life-rel", 20, 1000, 200, 800, "USD")
		return err
	}))
	assert.Equal(t, 1, loadCoupon(t, coupon.Id).TimesRedeemed)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return ReleaseSubscriptionCouponWithTx(tx, "order-life-rel")
	}))
	assert.Equal(t, CouponRedemptionStatusReleased, loadRedemptionByOrder(t, "order-life-rel").Status)
	assert.Equal(t, 0, loadCoupon(t, coupon.Id).TimesRedeemed, "release must decrement TimesRedeemed back to 0")
}
