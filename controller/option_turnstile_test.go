package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
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
