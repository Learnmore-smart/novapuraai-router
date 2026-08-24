package service

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service/deepseekfairuse"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendDeepSeekFairUseAdminInfoIsNestedAndDerived(t *testing.T) {
	rawToken := "sk-fup-log-raw-token"
	rawAuthorization := "Bearer " + rawToken
	rawIP := "198.51.100.22"
	rawUserAgent := "Mozilla/5.0"
	identifiers := deepseekfairuse.BuildIdentifiers([]byte("log-audit-secret"), deepseekfairuse.IdentityInput{
		UserID:    21,
		TokenID:   22,
		ClientIP:  rawIP,
		Country:   "CA",
		UserAgent: rawUserAgent,
	})
	info := &relaycommon.RelayInfo{}
	info.DeepSeekFairUse = &relaycommon.DeepSeekFairUseAudit{
		AccountHMAC:          identifiers.AccountHMAC,
		RiskHMAC:             identifiers.RiskHMAC,
		RiskIPHMAC:           identifiers.RiskIPHMAC,
		RiskCountryHMAC:      identifiers.RiskCountryHMAC,
		RiskUserAgentHMAC:    identifiers.RiskUserAgentHMAC,
		PeakLimit:            deepseekfairuse.PeakConcurrency,
		Active:               3,
		ConcurrentSeconds:    40,
		Admitted:             4,
		Successful:           2,
		EffectiveConcurrency: 1,
		Degraded:             true,
		ExhaustionEvents:     3,
		RiskMarked:           true,
	}

	adminInfo := map[string]interface{}{}
	AppendDeepSeekFairUseAdminInfo(info, adminInfo)
	payload, ok := adminInfo["deepseek_fair_use"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, identifiers.AccountHMAC, payload["account_hmac"])
	assert.Equal(t, identifiers.RiskHMAC, payload["risk_hmac"])
	assert.Equal(t, identifiers.RiskIPHMAC, payload["risk_ip_hmac"])
	assert.Equal(t, identifiers.RiskCountryHMAC, payload["risk_country_hmac"])
	assert.Equal(t, identifiers.RiskUserAgentHMAC, payload["risk_user_agent_hmac"])
	assert.Equal(t, 1, payload["effective_concurrency"])
	assert.Equal(t, true, payload["risk_marked"])
	assert.NotContains(t, payload, "client_ip")
	assert.NotContains(t, payload, "authorization")
	assert.NotContains(t, payload, "token_key")
	serializedPayload := fmt.Sprint(payload)
	assert.NotContains(t, serializedPayload, rawToken)
	assert.NotContains(t, serializedPayload, rawAuthorization)
	assert.NotContains(t, serializedPayload, rawIP)
	assert.NotContains(t, serializedPayload, rawUserAgent)
	assert.NotContains(t, identifiers.AccountHMAC, rawIP)
}

func TestGenerateTextOtherInfoIncludesDeepSeekFairUseAdminInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		StartTime:         time.Unix(100, 0),
		FirstResponseTime: time.Unix(101, 0),
		ChannelMeta:       &relaycommon.ChannelMeta{},
		DeepSeekFairUse: &relaycommon.DeepSeekFairUseAudit{
			AccountHMAC: "account-hmac",
			RiskHMAC:    "risk-hmac",
		},
	}

	other := GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 1, 0, 1)
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	fairUseInfo, ok := adminInfo["deepseek_fair_use"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "account-hmac", fairUseInfo["account_hmac"])
	assert.Equal(t, "risk-hmac", fairUseInfo["risk_hmac"])
}
