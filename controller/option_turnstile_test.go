package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTurnstileConfigurationReady(t *testing.T) {
	originalSiteKey := common.TurnstileSiteKey
	originalSecretKey := common.TurnstileSecretKey
	originalHostnames := common.TurnstileAllowedHostnames
	t.Cleanup(func() {
		common.TurnstileSiteKey = originalSiteKey
		common.TurnstileSecretKey = originalSecretKey
		common.TurnstileAllowedHostnames = originalHostnames
	})

	tests := []struct {
		name      string
		siteKey   string
		secretKey string
		hostnames string
		ready     bool
	}{
		{name: "complete", siteKey: "site", secretKey: "secret", hostnames: "novapuraai.com", ready: true},
		{name: "missing site key", secretKey: "secret", hostnames: "novapuraai.com"},
		{name: "missing secret", siteKey: "site", hostnames: "novapuraai.com"},
		{name: "missing hostname", siteKey: "site", secretKey: "secret"},
		{name: "whitespace is missing", siteKey: " ", secretKey: "secret", hostnames: "novapuraai.com"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			common.TurnstileSiteKey = test.siteKey
			common.TurnstileSecretKey = test.secretKey
			common.TurnstileAllowedHostnames = test.hostnames
			assert.Equal(t, test.ready, turnstileConfigurationReady())
		})
	}
}

func TestGetOptionsReturnsTurnstileSiteKeyButNotSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	common.OptionMapRWMutex.Lock()
	optionMapWasNil := common.OptionMap == nil
	if optionMapWasNil {
		common.OptionMap = make(map[string]string)
	}
	originalSiteKey, siteKeyExists := common.OptionMap["TurnstileSiteKey"]
	originalSecretKey, secretKeyExists := common.OptionMap["TurnstileSecretKey"]
	common.OptionMap["TurnstileSiteKey"] = "public-site-key"
	common.OptionMap["TurnstileSecretKey"] = "must-not-be-returned"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if optionMapWasNil {
			common.OptionMap = nil
			return
		}
		if siteKeyExists {
			common.OptionMap["TurnstileSiteKey"] = originalSiteKey
		} else {
			delete(common.OptionMap, "TurnstileSiteKey")
		}
		if secretKeyExists {
			common.OptionMap["TurnstileSecretKey"] = originalSecretKey
		} else {
			delete(common.OptionMap, "TurnstileSecretKey")
		}
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	GetOptions(ctx)

	require.Equal(t, 200, recorder.Code)
	var response struct {
		Success bool            `json:"success"`
		Data    []*model.Option `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	returned := make(map[string]string, len(response.Data))
	for _, option := range response.Data {
		returned[option.Key] = option.Value
	}
	assert.Equal(t, "public-site-key", returned["TurnstileSiteKey"])
	_, secretReturned := returned["TurnstileSecretKey"]
	assert.False(t, secretReturned)
}
