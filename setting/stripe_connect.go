package setting

import (
	"fmt"
	"os"
	"strings"
)

// Stripe Connect（分佣自动打款）运行时配置。
// 密钥由 Cloud Run 从 Secret Manager 注入为环境变量；Go 代码只读 os.Getenv，
// 不引入 GCSM SDK。模式由 STRIPE_CONNECT_MODE 显式控制（test/live），
// 不像充值凭证那样按 GIN_MODE 静默抹除。

const (
	StripeConnectModeTest = "test"
	StripeConnectModeLive = "live"
)

var (
	StripeConnectMode           = StripeConnectModeTest
	StripeConnectEnabled        = false
	StripeConnectSecretKey      = "" // sk_test_... / sk_live_...
	StripeConnectWebhookSecret  = "" // whsec_...
	StripeConnectReturnURL      = ""
	StripeConnectRefreshURL     = ""
	StripeConnectEnvInitialized = false
)

// InitStripeConnectEnv 从环境变量加载 Connect 配置并做模式感知校验。
// STRIPE_CONNECT_ENABLED=true 但校验失败时 fail-fast（返回 error）。
// 由 main.go 在 DB 初始化后调用。
func InitStripeConnectEnv() error {
	StripeConnectMode = strings.ToLower(strings.TrimSpace(os.Getenv("STRIPE_CONNECT_MODE")))
	if StripeConnectMode == "" {
		StripeConnectMode = StripeConnectModeTest
	}
	if StripeConnectMode != StripeConnectModeTest && StripeConnectMode != StripeConnectModeLive {
		return fmt.Errorf("STRIPE_CONNECT_MODE must be 'test' or 'live', got %q", StripeConnectMode)
	}

	StripeConnectSecretKey = strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	StripeConnectWebhookSecret = strings.TrimSpace(os.Getenv("STRIPE_CONNECT_WEBHOOK_SECRET"))
	StripeConnectReturnURL = strings.TrimSpace(os.Getenv("STRIPE_CONNECT_RETURN_URL"))
	StripeConnectRefreshURL = strings.TrimSpace(os.Getenv("STRIPE_CONNECT_REFRESH_URL"))

	enabledStr := strings.ToLower(strings.TrimSpace(os.Getenv("STRIPE_CONNECT_ENABLED")))
	StripeConnectEnabled = enabledStr == "true" || enabledStr == "1"

	StripeConnectEnvInitialized = true

	if !StripeConnectEnabled {
		return nil // 未启用则不校验密钥
	}

	if StripeConnectMode == StripeConnectModeTest {
		if !strings.HasPrefix(StripeConnectSecretKey, "sk_test_") {
			return fmt.Errorf("STRIPE_CONNECT_MODE=test requires STRIPE_SECRET_KEY prefix 'sk_test_'")
		}
		if strings.HasPrefix(StripeConnectSecretKey, "sk_live_") || strings.HasPrefix(StripeConnectSecretKey, "rk_live_") {
			return fmt.Errorf("STRIPE_CONNECT_MODE=test rejects live secret key")
		}
	} else { // live
		if !strings.HasPrefix(StripeConnectSecretKey, "sk_live_") && !strings.HasPrefix(StripeConnectSecretKey, "rk_live_") {
			return fmt.Errorf("STRIPE_CONNECT_MODE=live requires STRIPE_SECRET_KEY prefix 'sk_live_' or 'rk_live_'")
		}
	}
	if !strings.HasPrefix(StripeConnectWebhookSecret, "whsec_") {
		return fmt.Errorf("STRIPE_CONNECT_WEBHOOK_SECRET must have prefix 'whsec_'")
	}
	if StripeConnectReturnURL == "" || StripeConnectRefreshURL == "" {
		return fmt.Errorf("STRIPE_CONNECT_RETURN_URL and STRIPE_CONNECT_REFRESH_URL must be set when Stripe Connect is enabled")
	}
	return nil
}
