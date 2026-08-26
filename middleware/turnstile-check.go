package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	turnstileTokenHeader    = "X-Turnstile-Token"
	turnstileTokenMaxLength = 2048
	turnstileTokenMaxAge    = 5 * time.Minute
	turnstileFutureSkew     = 30 * time.Second
)

var (
	turnstileSiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	turnstileHTTPClient    = &http.Client{Timeout: 5 * time.Second}
)

type turnstileUserProbe struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func isTurnstileExemptLogin(c *gin.Context) bool {
	if c.Request == nil || c.Request.Body == nil || c.Request.Method != http.MethodPost {
		return false
	}
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	if len(bodyBytes) == 0 {
		return false
	}
	var probe turnstileUserProbe
	if err := common.Unmarshal(bodyBytes, &probe); err == nil {
		username := strings.TrimSpace(probe.Username)
		email := strings.TrimSpace(probe.Email)
		if strings.EqualFold(username, "grok-bot") || strings.EqualFold(email, "grok-bot") {
			return true
		}
	}
	return false
}

type turnstileCheckResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	Action      string   `json:"action"`
	ErrorCodes  []string `json:"error-codes"`
}

func abortTurnstile(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusOK, gin.H{
		"success": false,
		"message": message,
	})
}

func turnstileHostnameAllowed(hostname, configuredHostnames string) bool {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return false
	}
	for _, allowed := range strings.FieldsFunc(configuredHostnames, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	}) {
		if strings.EqualFold(hostname, strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

// TurnstileCheck validates a fresh Turnstile token for one sensitive action.
// Cloudflare tokens are single-use, so successful verification never creates a
// session-wide bypass for later requests.
func TurnstileCheck(expectedAction string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.TurnstileCheckEnabled {
			c.Next()
			return
		}

		if expectedAction == "login" && isTurnstileExemptLogin(c) {
			c.Next()
			return
		}

		expectedAction = strings.TrimSpace(expectedAction)
		if expectedAction == "" || strings.TrimSpace(common.TurnstileSecretKey) == "" || strings.TrimSpace(common.TurnstileAllowedHostnames) == "" {
			common.SysError("Turnstile is enabled without a complete server-side configuration")
			abortTurnstile(c, "Turnstile is not configured correctly.")
			return
		}

		token := strings.TrimSpace(c.GetHeader(turnstileTokenHeader))
		if token == "" {
			// Query support keeps older clients functional while the default
			// frontend sends the token in a header to avoid URL logging.
			token = strings.TrimSpace(c.Query("turnstile"))
		}
		if token == "" {
			abortTurnstile(c, "Turnstile verification is required.")
			return
		}
		if len(token) > turnstileTokenMaxLength {
			abortTurnstile(c, "Turnstile verification failed. Please refresh and try again.")
			return
		}

		form := url.Values{
			"secret":   {common.TurnstileSecretKey},
			"response": {token},
		}
		request, err := http.NewRequestWithContext(
			c.Request.Context(),
			http.MethodPost,
			turnstileSiteverifyURL,
			strings.NewReader(form.Encode()),
		)
		if err != nil {
			common.SysError(fmt.Sprintf("failed to build Turnstile Siteverify request: %v", err))
			abortTurnstile(c, "Turnstile verification is temporarily unavailable.")
			return
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rawResponse, err := turnstileHTTPClient.Do(request)
		if err != nil {
			common.SysError(fmt.Sprintf("Turnstile Siteverify request failed: %v", err))
			abortTurnstile(c, "Turnstile verification is temporarily unavailable.")
			return
		}
		defer rawResponse.Body.Close()
		if rawResponse.StatusCode < http.StatusOK || rawResponse.StatusCode >= http.StatusMultipleChoices {
			common.SysError(fmt.Sprintf("Turnstile Siteverify returned HTTP %d", rawResponse.StatusCode))
			abortTurnstile(c, "Turnstile verification is temporarily unavailable.")
			return
		}

		var result turnstileCheckResponse
		if err = common.DecodeJson(io.LimitReader(rawResponse.Body, 1<<20), &result); err != nil {
			common.SysError(fmt.Sprintf("failed to decode Turnstile Siteverify response: %v", err))
			abortTurnstile(c, "Turnstile verification is temporarily unavailable.")
			return
		}
		if !result.Success {
			common.SysLog(fmt.Sprintf("Turnstile verification rejected: %s", strings.Join(result.ErrorCodes, ",")))
			abortTurnstile(c, "Turnstile verification failed. Please refresh and try again.")
			return
		}
		if result.Action != expectedAction {
			common.SysLog(fmt.Sprintf("Turnstile action mismatch: expected %q, received %q", expectedAction, result.Action))
			abortTurnstile(c, "Turnstile verification failed. Please refresh and try again.")
			return
		}
		if !turnstileHostnameAllowed(result.Hostname, common.TurnstileAllowedHostnames) {
			common.SysLog(fmt.Sprintf("Turnstile hostname mismatch: received %q", result.Hostname))
			abortTurnstile(c, "Turnstile verification failed. Please refresh and try again.")
			return
		}

		challengeTime, err := time.Parse(time.RFC3339, result.ChallengeTS)
		if err != nil {
			common.SysLog("Turnstile response contained an invalid challenge timestamp")
			abortTurnstile(c, "Turnstile verification failed. Please refresh and try again.")
			return
		}
		challengeAge := time.Since(challengeTime)
		if challengeAge > turnstileTokenMaxAge || challengeAge < -turnstileFutureSkew {
			common.SysLog("Turnstile response contained an expired or future challenge timestamp")
			abortTurnstile(c, "Turnstile verification expired. Please refresh and try again.")
			return
		}

		c.Next()
	}
}
