package deepseekfairuse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsEligibleRequiresDedicatedGroupAndNonEmptyOriginalModel(t *testing.T) {
	tests := []struct {
		name  string
		input EligibilityInput
		want  bool
	}{
		{
			name: "dedicated unlimited entitlement",
			input: EligibilityInput{
				DedicatedEntitlement: true,
				Group:                "deepseek-v4-flash-unlimited",
				DedicatedGroup:       "deepseek-v4-flash-unlimited",
				OriginalModelName:    DeepSeekV4Flash0731Model,
			},
			want: true,
		},
		{
			name: "pay as you go group",
			input: EligibilityInput{
				DedicatedEntitlement: true,
				Group:                "default",
				DedicatedGroup:       "deepseek-v4-flash-unlimited",
				OriginalModelName:    DeepSeekV4Flash0731Model,
			},
			want: false,
		},
		{
			name: "unlimited flag without entitlement",
			input: EligibilityInput{
				DedicatedEntitlement: false,
				Group:                "deepseek-v4-flash-unlimited",
				DedicatedGroup:       "deepseek-v4-flash-unlimited",
				OriginalModelName:    DeepSeekV4Flash0731Model,
			},
			want: false,
		},
		{
			name: "different platform model",
			input: EligibilityInput{
				DedicatedEntitlement: true,
				Group:                "deepseek-v4-flash-unlimited",
				DedicatedGroup:       "deepseek-v4-flash-unlimited",
				OriginalModelName:    "gpt-5",
			},
			want: true,
		},
		{
			name: "missing model",
			input: EligibilityInput{
				DedicatedEntitlement: true,
				Group:                "deepseek-v4-flash-unlimited",
				DedicatedGroup:       "deepseek-v4-flash-unlimited",
				OriginalModelName:    " ",
			},
			want: false,
		},
		{
			name: "ordinary paid token",
			input: EligibilityInput{
				Group:             "deepseek-v4-flash-unlimited",
				DedicatedGroup:    "deepseek-v4-flash-unlimited",
				OriginalModelName: DeepSeekV4Flash0731Model,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsEligible(tt.input))
		})
	}
}

func TestIdentifiersAreDomainSeparatedAndDoNotExposeSecrets(t *testing.T) {
	input := IdentityInput{
		UserID:    42,
		TokenID:   9001,
		ClientIP:  "203.0.113.7",
		Country:   "ca",
		UserAgent: "curl/8.7.1",
	}
	ids := BuildIdentifiers([]byte("fair-use-test-secret"), input)
	require.NotEmpty(t, ids.AccountHMAC)
	require.NotEmpty(t, ids.RiskHMAC)
	assert.NotEqual(t, ids.AccountHMAC, ids.RiskHMAC)
	assert.NotEqual(t, ids.RiskIPHMAC, ids.RiskCountryHMAC)
	assert.NotEqual(t, ids.RiskIPHMAC, ids.RiskUserAgentHMAC)
	assert.NotEqual(t, ids.RiskCountryHMAC, ids.RiskUserAgentHMAC)
	for _, secret := range []string{input.ClientIP, "sk-secret-token", "Bearer sk-secret-token", "42", "9001"} {
		assert.NotContains(t, ids.AccountHMAC, secret)
		assert.NotContains(t, ids.RiskHMAC, secret)
	}

	otherToken := input
	otherToken.TokenID++
	assert.Equal(t, ids.AccountHMAC, BuildIdentifiers([]byte("fair-use-test-secret"), otherToken).AccountHMAC)
	assert.NotEqual(t, ids.RiskHMAC, BuildIdentifiers([]byte("fair-use-test-secret"), otherToken).RiskHMAC)
	assert.False(t, strings.Contains(ids.RiskHMAC, strings.ToLower(input.Country)))
}
