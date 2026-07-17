package channel

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessHeaderOverride_ChannelTestSkipsPassthroughRules(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Empty(t, headers)
}

func TestProcessHeaderOverride_ChannelTestSkipsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	_, ok := headers["x-upstream-trace"]
	require.False(t, ok)
}

func TestProcessHeaderOverride_NonTestKeepsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-upstream-trace"])
}

func TestProcessHeaderOverride_RuntimeOverrideIsFinalHeaderMap(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		IsChannelTest:             false,
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]any{
			"x-static":  "runtime-value",
			"x-runtime": "runtime-only",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
				"X-Legacy": "legacy-only",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "runtime-value", headers["x-static"])
	require.Equal(t, "runtime-only", headers["x-runtime"])
	_, exists := headers["x-legacy"]
	require.False(t, exists)
}

func TestProcessHeaderOverride_PassthroughSkipsAcceptEncoding(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")
	ctx.Request.Header.Set("Accept-Encoding", "gzip")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-trace-id"])

	_, hasAcceptEncoding := headers["accept-encoding"]
	require.False(t, hasAcceptEncoding)
}

func TestProcessHeaderOverride_PassHeadersTemplateSetsRuntimeHeaders(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Request.Header.Set("Originator", "Codex CLI")
	ctx.Request.Header.Set("Session_id", "sess-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		RequestHeaders: map[string]string{
			"Originator": "Codex CLI",
			"Session_id": "sess-123",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ParamOverride: map[string]any{
				"operations": []any{
					map[string]any{
						"mode":  "pass_headers",
						"value": []any{"Originator", "Session_id", "X-Codex-Beta-Features"},
					},
				},
			},
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
			},
		},
	}

	_, err := relaycommon.ApplyParamOverrideWithRelayInfo([]byte(`{"model":"gpt-4.1"}`), info)
	require.NoError(t, err)
	require.True(t, info.UseRuntimeHeadersOverride)
	require.Equal(t, "Codex CLI", info.RuntimeHeadersOverride["originator"])
	require.Equal(t, "sess-123", info.RuntimeHeadersOverride["session_id"])
	_, exists := info.RuntimeHeadersOverride["x-codex-beta-features"]
	require.False(t, exists)
	require.Equal(t, "legacy-value", info.RuntimeHeadersOverride["x-static"])

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "Codex CLI", headers["originator"])
	require.Equal(t, "sess-123", headers["session_id"])
	_, exists = headers["x-codex-beta-features"]
	require.False(t, exists)

	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	applyHeaderOverrideToRequest(upstreamReq, headers)
	require.Equal(t, "Codex CLI", upstreamReq.Header.Get("Originator"))
	require.Equal(t, "sess-123", upstreamReq.Header.Get("Session_id"))
	require.Empty(t, upstreamReq.Header.Get("X-Codex-Beta-Features"))
}

func TestMapUpstreamRequestError_ClientCanceled(t *testing.T) {
	t.Parallel()

	apiErr := mapUpstreamRequestError(context.Canceled)
	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodeClientDisconnected, apiErr.GetErrorCode())
	assert.Equal(t, 499, apiErr.StatusCode)
	assert.True(t, types.IsSkipRetryError(apiErr))
	assert.True(t, errors.Is(apiErr, context.Canceled))
}

func TestMapUpstreamRequestError_DeadlineExceeded(t *testing.T) {
	t.Parallel()

	apiErr := mapUpstreamRequestError(context.DeadlineExceeded)
	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodeUpstreamTimeout, apiErr.GetErrorCode())
	assert.Equal(t, http.StatusGatewayTimeout, apiErr.StatusCode)
	assert.True(t, types.IsSkipRetryError(apiErr))
}

func TestMapUpstreamRequestError_ClientTimeoutMessage(t *testing.T) {
	t.Parallel()

	apiErr := mapUpstreamRequestError(errors.New(`net/http: request canceled (Client.Timeout exceeded while awaiting headers)`))
	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodeUpstreamTimeout, apiErr.GetErrorCode())
	assert.True(t, types.IsSkipRetryError(apiErr))
}

func TestMapUpstreamRequestError_OtherError(t *testing.T) {
	t.Parallel()
	assert.Nil(t, mapUpstreamRequestError(errors.New("connection refused")))
}

func TestWrapDoRequestError_PreservesNewAPIError(t *testing.T) {
	t.Parallel()

	orig := types.NewErrorWithStatusCode(context.Canceled, types.ErrorCodeClientDisconnected, 499, types.ErrOptionWithSkipRetry())
	wrapped := wrapDoRequestError(orig)
	var apiErr *types.NewAPIError
	require.True(t, errors.As(wrapped, &apiErr))
	assert.Equal(t, types.ErrorCodeClientDisconnected, apiErr.GetErrorCode())
	assert.True(t, types.IsSkipRetryError(apiErr))
}

func TestNewUpstreamRequest_UsesClientContext(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()
	ctx.Request = httptest.NewRequestWithContext(reqCtx, http.MethodPost, "/v1/chat/completions", nil)

	upstream, err := newUpstreamRequest(ctx, http.MethodPost, "http://example.invalid/v1/chat/completions", nil)
	require.NoError(t, err)
	assert.ErrorIs(t, upstream.Context().Err(), context.Canceled)
}

func TestDoRequest_CancelsUpstreamOnClientDisconnect(t *testing.T) {
	// Integration-style: slow upstream blocked until client context cancels.
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			return
		case <-release:
			_, _ = io.WriteString(w, `{"ok":true}`)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })

	// Ensure shared client has no total timeout interfering with this cancel test.
	prevClient := service.GetHttpClient()
	service.InitHttpClient()
	t.Cleanup(func() {
		// re-init with process defaults after test
		_ = prevClient
		service.InitHttpClient()
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	reqCtx, cancel := context.WithCancel(context.Background())
	c.Request = httptest.NewRequestWithContext(reqCtx, http.MethodPost, "/v1/chat/completions", nil)

	upstreamReq, err := newUpstreamRequest(c, http.MethodPost, server.URL, nil)
	require.NoError(t, err)

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}

	errCh := make(chan error, 1)
	go func() {
		resp, doErr := doRequest(c, upstreamReq, info)
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		errCh <- doErr
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream did not receive request")
	}

	cancel()

	select {
	case doErr := <-errCh:
		require.Error(t, doErr)
		var apiErr *types.NewAPIError
		require.True(t, errors.As(doErr, &apiErr), "got %T %v", doErr, doErr)
		assert.Equal(t, types.ErrorCodeClientDisconnected, apiErr.GetErrorCode())
		assert.True(t, types.IsSkipRetryError(apiErr))
	case <-time.After(5 * time.Second):
		t.Fatal("doRequest did not return after client cancel")
	}
}
