package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type turnstileTestResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts,omitempty"`
	Hostname    string   `json:"hostname,omitempty"`
	Action      string   `json:"action,omitempty"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
}

func configureTurnstileTest(t *testing.T, response turnstileTestResponse) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	originalEnabled := common.TurnstileCheckEnabled
	originalSecret := common.TurnstileSecretKey
	originalHostnames := common.TurnstileAllowedHostnames
	originalURL := turnstileSiteverifyURL
	t.Cleanup(func() {
		common.TurnstileCheckEnabled = originalEnabled
		common.TurnstileSecretKey = originalSecret
		common.TurnstileAllowedHostnames = originalHostnames
		turnstileSiteverifyURL = originalURL
	})

	common.TurnstileCheckEnabled = true
	common.TurnstileSecretKey = "test-secret"
	common.TurnstileAllowedHostnames = "novapuraai.com, localhost"

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "test-secret", r.Form.Get("secret"))
		assert.NotEmpty(t, r.Form.Get("response"))
		assert.NotEmpty(t, r.Form.Get("remoteip"))
		payload, err := common.Marshal(response)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(payload)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	turnstileSiteverifyURL = server.URL

	return server, &calls
}

func runTurnstileRequest(action, token string, nextCalled *bool) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TurnstileCheck(action))
	router.POST("/guarded", func(c *gin.Context) {
		*nextCalled = true
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/guarded", nil)
	if token != "" {
		request.Header.Set(turnstileTokenHeader, token)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertTurnstileRejected(t *testing.T, recorder *httptest.ResponseRecorder, nextCalled bool) {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.False(t, nextCalled)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
}

func TestTurnstileCheckAcceptsValidRequestScopedToken(t *testing.T) {
	_, calls := configureTurnstileTest(t, turnstileTestResponse{
		Success:     true,
		ChallengeTS: time.Now().UTC().Format(time.RFC3339Nano),
		Hostname:    "novapuraai.com",
		Action:      "register",
	})
	nextCalled := false
	recorder := runTurnstileRequest("register", "valid-token", &nextCalled)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, nextCalled)
	assert.EqualValues(t, 1, calls.Load())
}

func TestTurnstileCheckRejectsMissingToken(t *testing.T) {
	_, calls := configureTurnstileTest(t, turnstileTestResponse{})
	nextCalled := false
	recorder := runTurnstileRequest("register", "", &nextCalled)

	assertTurnstileRejected(t, recorder, nextCalled)
	assert.Zero(t, calls.Load())
}

func TestTurnstileCheckRejectsFailedVerification(t *testing.T) {
	_, calls := configureTurnstileTest(t, turnstileTestResponse{
		Success:    false,
		ErrorCodes: []string{"invalid-input-response"},
	})
	nextCalled := false
	recorder := runTurnstileRequest("register", "invalid-token", &nextCalled)

	assertTurnstileRejected(t, recorder, nextCalled)
	assert.EqualValues(t, 1, calls.Load())
}

func TestTurnstileCheckRejectsActionMismatch(t *testing.T) {
	_, _ = configureTurnstileTest(t, turnstileTestResponse{
		Success:     true,
		ChallengeTS: time.Now().UTC().Format(time.RFC3339Nano),
		Hostname:    "novapuraai.com",
		Action:      "login",
	})
	nextCalled := false
	recorder := runTurnstileRequest("register", "wrong-action", &nextCalled)

	assertTurnstileRejected(t, recorder, nextCalled)
}

func TestTurnstileCheckRejectsHostnameMismatch(t *testing.T) {
	_, _ = configureTurnstileTest(t, turnstileTestResponse{
		Success:     true,
		ChallengeTS: time.Now().UTC().Format(time.RFC3339Nano),
		Hostname:    "attacker.example",
		Action:      "register",
	})
	nextCalled := false
	recorder := runTurnstileRequest("register", "wrong-host", &nextCalled)

	assertTurnstileRejected(t, recorder, nextCalled)
}

func TestTurnstileCheckRejectsExpiredChallenge(t *testing.T) {
	_, _ = configureTurnstileTest(t, turnstileTestResponse{
		Success:     true,
		ChallengeTS: time.Now().Add(-turnstileTokenMaxAge - time.Second).UTC().Format(time.RFC3339Nano),
		Hostname:    "novapuraai.com",
		Action:      "register",
	})
	nextCalled := false
	recorder := runTurnstileRequest("register", "expired-token", &nextCalled)

	assertTurnstileRejected(t, recorder, nextCalled)
}

func TestTurnstileCheckSupportsQueryTokenForLegacyClients(t *testing.T) {
	_, calls := configureTurnstileTest(t, turnstileTestResponse{
		Success:     true,
		ChallengeTS: time.Now().UTC().Format(time.RFC3339Nano),
		Hostname:    "novapuraai.com",
		Action:      "login",
	})

	gin.SetMode(gin.TestMode)
	nextCalled := false
	router := gin.New()
	router.Use(TurnstileCheck("login"))
	router.POST("/guarded", func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/guarded?turnstile="+url.QueryEscape("legacy-token"), nil)
	router.ServeHTTP(recorder, request)

	assert.True(t, nextCalled)
	assert.EqualValues(t, 1, calls.Load())
}

func TestTurnstileCheckDoesNotCreateSessionBypass(t *testing.T) {
	originalEnabled := common.TurnstileCheckEnabled
	originalSecret := common.TurnstileSecretKey
	originalHostnames := common.TurnstileAllowedHostnames
	originalURL := turnstileSiteverifyURL
	t.Cleanup(func() {
		common.TurnstileCheckEnabled = originalEnabled
		common.TurnstileSecretKey = originalSecret
		common.TurnstileAllowedHostnames = originalHostnames
		turnstileSiteverifyURL = originalURL
	})
	common.TurnstileCheckEnabled = true
	common.TurnstileSecretKey = "test-secret"
	common.TurnstileAllowedHostnames = "novapuraai.com"

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := calls.Add(1)
		response := turnstileTestResponse{
			Success:     call == 1,
			ChallengeTS: time.Now().UTC().Format(time.RFC3339Nano),
			Hostname:    "novapuraai.com",
			Action:      "share_reward",
		}
		if call > 1 {
			response.ErrorCodes = []string{"timeout-or-duplicate"}
		}
		payload, err := common.Marshal(response)
		require.NoError(t, err)
		_, err = w.Write(payload)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	turnstileSiteverifyURL = server.URL

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(TurnstileCheck("share_reward"))
	router.POST("/share", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodPost, "/share", strings.NewReader("{}"))
	firstRequest.Header.Set(turnstileTokenHeader, "single-use-token")
	router.ServeHTTP(first, firstRequest)
	require.Equal(t, http.StatusNoContent, first.Code)

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/share", strings.NewReader("{}"))
	secondRequest.Header.Set(turnstileTokenHeader, "single-use-token")
	for _, cookieValue := range first.Result().Cookies() {
		secondRequest.AddCookie(cookieValue)
	}
	router.ServeHTTP(second, secondRequest)

	assertTurnstileRejected(t, second, false)
	assert.EqualValues(t, 2, calls.Load())
}
