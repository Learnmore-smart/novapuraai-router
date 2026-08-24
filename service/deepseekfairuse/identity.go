package deepseekfairuse

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	DeepSeekV4Flash0731Model      = "deepseek-v4-flash-0731"
	DeepSeekV4FlashDedicatedGroup = "deepseek-v4-flash-unlimited"
)

// EligibilityInput is account-scoped. API-key quota and token-group overrides
// must not bypass the dedicated subscription's fair-use policy.
type EligibilityInput struct {
	DedicatedEntitlement bool
	Group                string
	DedicatedGroup       string
	OriginalModelName    string
}

func IsEligible(input EligibilityInput) bool {
	return input.DedicatedEntitlement &&
		strings.TrimSpace(input.Group) != "" &&
		input.Group == input.DedicatedGroup &&
		strings.TrimSpace(input.OriginalModelName) != ""
}

// IdentityInput is request-local input. Raw values are never returned by
// BuildIdentifiers and must not be logged by callers.
type IdentityInput struct {
	UserID    int
	TokenID   int
	ClientIP  string
	Country   string
	UserAgent string
}

type Identifiers struct {
	AccountHMAC       string
	RiskHMAC          string
	RiskIPHMAC        string
	RiskCountryHMAC   string
	RiskUserAgentHMAC string
}

type RiskSignals struct {
	IPHMAC        string
	CountryHMAC   string
	UserAgentHMAC string
}

func (ids Identifiers) RiskSignals() RiskSignals {
	return RiskSignals{
		IPHMAC:        ids.RiskIPHMAC,
		CountryHMAC:   ids.RiskCountryHMAC,
		UserAgentHMAC: ids.RiskUserAgentHMAC,
	}
}

// BuildIdentifiers uses separate domains for account, risk, and each risk
// dimension. User aggregation stays stable across token rotation while the
// abuse signals remain privacy preserving.
func BuildIdentifiers(secret []byte, input IdentityInput) Identifiers {
	accountPayload := fmt.Sprintf("deepseek-fair-use:account:v1:%d", input.UserID)
	accountHMAC := common.GenerateHMACWithKey(secret, accountPayload)

	ip := normalizeClientIP(input.ClientIP)
	country := normalizeCountry(input.Country)
	uaFamily := normalizeUserAgentFamily(input.UserAgent)
	riskIPHMAC := common.GenerateHMACWithKey(secret, "deepseek-fair-use:risk-ip:v1:"+ip)
	riskCountryHMAC := common.GenerateHMACWithKey(secret, "deepseek-fair-use:risk-country:v1:"+country)
	riskUserAgentHMAC := common.GenerateHMACWithKey(secret, "deepseek-fair-use:risk-ua:v1:"+uaFamily)
	riskPayload := "deepseek-fair-use:risk:v1:" + ip + "\x00" + country + "\x00" + uaFamily + "\x00" + strconv.Itoa(input.TokenID)

	return Identifiers{
		AccountHMAC:       accountHMAC,
		RiskHMAC:          common.GenerateHMACWithKey(secret, riskPayload),
		RiskIPHMAC:        riskIPHMAC,
		RiskCountryHMAC:   riskCountryHMAC,
		RiskUserAgentHMAC: riskUserAgentHMAC,
	}
}

func normalizeClientIP(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeCountry(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeUserAgentFamily(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	for _, separator := range []string{"/", " ", "(", ";"} {
		if index := strings.Index(value, separator); index >= 0 {
			value = value[:index]
		}
	}
	if value == "" {
		return "unknown"
	}
	return value
}
