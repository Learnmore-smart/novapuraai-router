/*
Copyright (C) 2023-2026 QuantumNous / NovaPuraAI fork tests

Failure-policy tests for the 号池: call shipped ShouldDisableChannel and
status-range helpers with representative upstream status codes.
*/
package service

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

// designAlignedDisableRanges mirrors NovaPura 号池 design: 401/403 → disable.
var designAlignedDisableRanges = []operation_setting.StatusCodeRange{
	{Start: 401, End: 403},
}

// designAlignedRetryRanges: 429 rate-limit + 5xx (except always-skip 504/524) → retry other channel.
var designAlignedRetryRanges = []operation_setting.StatusCodeRange{
	{Start: 429, End: 429},
	{Start: 500, End: 599},
}

func withAutoDisable(t *testing.T, enabled bool) {
	t.Helper()
	prev := common.AutomaticDisableChannelEnabled
	common.AutomaticDisableChannelEnabled = enabled
	t.Cleanup(func() { common.AutomaticDisableChannelEnabled = prev })
}

func withDisableRanges(t *testing.T, ranges []operation_setting.StatusCodeRange) {
	t.Helper()
	prev := operation_setting.AutomaticDisableStatusCodeRanges
	operation_setting.AutomaticDisableStatusCodeRanges = ranges
	t.Cleanup(func() { operation_setting.AutomaticDisableStatusCodeRanges = prev })
}

func withRetryRanges(t *testing.T, ranges []operation_setting.StatusCodeRange) {
	t.Helper()
	prev := operation_setting.AutomaticRetryStatusCodeRanges
	operation_setting.AutomaticRetryStatusCodeRanges = ranges
	t.Cleanup(func() { operation_setting.AutomaticRetryStatusCodeRanges = prev })
}

func statusErr(code int) *types.NewAPIError {
	return types.NewErrorWithStatusCode(
		errors.New("upstream status"),
		types.ErrorCodeBadResponseStatusCode,
		code,
	)
}

func TestShouldDisableChannel_401And403Disable(t *testing.T) {
	withAutoDisable(t, true)
	withDisableRanges(t, designAlignedDisableRanges)

	require.True(t, ShouldDisableChannel(statusErr(401)), "401 must disable channel")
	require.True(t, ShouldDisableChannel(statusErr(403)), "403 must disable channel")
}

func TestShouldDisableChannel_429And5xxDoNotDisableByStatus(t *testing.T) {
	withAutoDisable(t, true)
	withDisableRanges(t, designAlignedDisableRanges)

	// Rate limit and server errors switch channels (retry), not ban permanently via status alone.
	require.False(t, ShouldDisableChannel(statusErr(429)), "429 should not auto-disable via status ranges")
	require.False(t, ShouldDisableChannel(statusErr(500)), "500 should not auto-disable via status ranges")
	require.False(t, ShouldDisableChannel(statusErr(502)), "502 should not auto-disable via status ranges")
	require.False(t, ShouldDisableChannel(statusErr(200)), "success must not disable")
	require.False(t, ShouldDisableChannel(statusErr(404)), "404 not in disable ranges")
}

func TestShouldDisableChannel_FeatureFlagOff(t *testing.T) {
	withAutoDisable(t, false)
	withDisableRanges(t, designAlignedDisableRanges)

	require.False(t, ShouldDisableChannel(statusErr(401)))
	require.False(t, ShouldDisableChannel(statusErr(403)))
}

func TestShouldDisableChannel_NilError(t *testing.T) {
	withAutoDisable(t, true)
	require.False(t, ShouldDisableChannel(nil))
}

func TestShouldRetryByStatusCode_DesignAligned(t *testing.T) {
	withRetryRanges(t, designAlignedRetryRanges)

	// Design: 429 and 5xx → try another channel
	require.True(t, operation_setting.ShouldRetryByStatusCode(429))
	require.True(t, operation_setting.ShouldRetryByStatusCode(500))
	require.True(t, operation_setting.ShouldRetryByStatusCode(502))
	require.True(t, operation_setting.ShouldRetryByStatusCode(503))

	// Always-skip codes never retry even if inside a broad 5xx range
	require.False(t, operation_setting.ShouldRetryByStatusCode(504))
	require.False(t, operation_setting.ShouldRetryByStatusCode(524))

	// Auth failures are disable-class, not retry-class under design ranges
	require.False(t, operation_setting.ShouldRetryByStatusCode(401))
	require.False(t, operation_setting.ShouldRetryByStatusCode(403))
	require.False(t, operation_setting.ShouldRetryByStatusCode(200))
	require.False(t, operation_setting.ShouldRetryByStatusCode(400))
}

func TestShouldDisableByStatusCode_DesignAligned(t *testing.T) {
	withDisableRanges(t, designAlignedDisableRanges)

	require.True(t, operation_setting.ShouldDisableByStatusCode(401))
	require.True(t, operation_setting.ShouldDisableByStatusCode(402))
	require.True(t, operation_setting.ShouldDisableByStatusCode(403))
	require.False(t, operation_setting.ShouldDisableByStatusCode(429))
	require.False(t, operation_setting.ShouldDisableByStatusCode(500))
}
