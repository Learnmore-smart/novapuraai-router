package common

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
)

var (
	Port         = flag.Int("port", 3000, "the listening port")
	PrintVersion = flag.Bool("version", false, "print version and exit")
	PrintHelp    = flag.Bool("help", false, "print help and exit")
	LogDir       = flag.String("log-dir", "./logs", "specify the log directory")
)

func printHelp() {
	fmt.Println("NewAPI(Based OneAPI) " + Version + " - The next-generation LLM gateway and AI asset management system supports multiple languages.")
	fmt.Println("Original Project: OneAPI by JustSong - https://github.com/songquanpeng/one-api")
	fmt.Println("Maintainer: QuantumNous - https://github.com/QuantumNous/new-api")
	fmt.Println("Usage: newapi [--port <port>] [--log-dir <log directory>] [--version] [--help]")
}

func InitEnv() {
	flag.Parse()
	resolvedPort, err := resolveListeningPort(os.Getenv("PORT"), *Port, commandLineFlagProvided("port"))
	if err != nil {
		log.Fatal(err)
	}
	*Port = resolvedPort

	envVersion := os.Getenv("VERSION")
	if envVersion != "" {
		Version = envVersion
	}

	if *PrintVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	if *PrintHelp {
		printHelp()
		os.Exit(0)
	}

	if os.Getenv("SESSION_SECRET") != "" {
		ss := os.Getenv("SESSION_SECRET")
		if ss == "random_string" {
			log.Println("WARNING: SESSION_SECRET is set to the default value 'random_string', please change it to a random string.")
			log.Println("警告：SESSION_SECRET被设置为默认值'random_string'，请修改为随机字符串。")
			log.Fatal("Please set SESSION_SECRET to a random string.")
		} else {
			SessionSecret = ss
		}
	}
	if os.Getenv("CRYPTO_SECRET") != "" {
		CryptoSecret = os.Getenv("CRYPTO_SECRET")
	} else {
		CryptoSecret = SessionSecret
	}
	if err := InitSessionCookieSettings(); err != nil {
		log.Fatal(err)
	}
	if os.Getenv("SQLITE_PATH") != "" {
		SQLitePath = os.Getenv("SQLITE_PATH")
	}
	// Ensure parent directory exists for file-based SQLite paths.
	// Windows often reports SQLITE_CANTOPEN (14) as "out of memory" when the folder is missing.
	ensureSQLiteParentDir(SQLitePath)
	if *LogDir != "" {
		*LogDir, err = filepath.Abs(*LogDir)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := os.Stat(*LogDir); os.IsNotExist(err) {
			err = os.Mkdir(*LogDir, 0777)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// Initialize variables from constants.go that were using environment variables
	DebugEnabled = os.Getenv("DEBUG") == "true"
	MemoryCacheEnabled = os.Getenv("MEMORY_CACHE_ENABLED") == "true"
	IsMasterNode = os.Getenv("NODE_TYPE") != "slave"
	initNodeNameIdentity()
	TLSInsecureSkipVerify = GetEnvOrDefaultBool("TLS_INSECURE_SKIP_VERIFY", false)
	if TLSInsecureSkipVerify {
		if tr, ok := http.DefaultTransport.(*http.Transport); ok && tr != nil {
			if tr.TLSClientConfig != nil {
				tr.TLSClientConfig.InsecureSkipVerify = true
			} else {
				tr.TLSClientConfig = InsecureTLSConfig
			}
		}
	}
	initSMTPEnv()
	initTurnstileEnv()
	initGitHubOAuthEnv()

	// Parse requestInterval and set RequestInterval
	requestInterval, _ = strconv.Atoi(os.Getenv("POLLING_INTERVAL"))
	RequestInterval = time.Duration(requestInterval) * time.Second

	// Initialize variables with GetEnvOrDefault
	SyncFrequency = GetEnvOrDefault("SYNC_FREQUENCY", 60)
	BatchUpdateInterval = GetEnvOrDefault("BATCH_UPDATE_INTERVAL", 5)
	// Default ~14 minutes: long enough for heavy chat/SSE, under typical reverse-proxy caps.
	// Set RELAY_TIMEOUT=0 for unlimited (not recommended in production).
	RelayTimeout = GetEnvOrDefault("RELAY_TIMEOUT", 840)
	RelayIdleConnTimeout = GetEnvOrDefault("RELAY_IDLE_CONN_TIMEOUT", 90)
	RelayMaxIdleConns = GetEnvOrDefault("RELAY_MAX_IDLE_CONNS", 500)
	RelayMaxIdleConnsPerHost = GetEnvOrDefault("RELAY_MAX_IDLE_CONNS_PER_HOST", 100)

	// Initialize string variables with GetEnvOrDefaultString
	GeminiSafetySetting = GetEnvOrDefaultString("GEMINI_SAFETY_SETTING", "BLOCK_NONE")
	CohereSafetySetting = GetEnvOrDefaultString("COHERE_SAFETY_SETTING", "NONE")

	// Initialize rate limit variables
	GlobalApiRateLimitEnable = GetEnvOrDefaultBool("GLOBAL_API_RATE_LIMIT_ENABLE", true)
	GlobalApiRateLimitNum = GetEnvOrDefault("GLOBAL_API_RATE_LIMIT", 360)
	GlobalApiRateLimitDuration = int64(GetEnvOrDefault("GLOBAL_API_RATE_LIMIT_DURATION", 180))

	GlobalWebRateLimitEnable = GetEnvOrDefaultBool("GLOBAL_WEB_RATE_LIMIT_ENABLE", true)
	GlobalWebRateLimitNum = GetEnvOrDefault("GLOBAL_WEB_RATE_LIMIT", 120)
	GlobalWebRateLimitDuration = int64(GetEnvOrDefault("GLOBAL_WEB_RATE_LIMIT_DURATION", 180))

	CriticalRateLimitEnable = GetEnvOrDefaultBool("CRITICAL_RATE_LIMIT_ENABLE", true)
	CriticalRateLimitNum = GetEnvOrDefault("CRITICAL_RATE_LIMIT", 20)
	CriticalRateLimitDuration = int64(GetEnvOrDefault("CRITICAL_RATE_LIMIT_DURATION", 20*60))

	SearchRateLimitEnable = GetEnvOrDefaultBool("SEARCH_RATE_LIMIT_ENABLE", true)
	SearchRateLimitNum = GetEnvOrDefault("SEARCH_RATE_LIMIT", 10)
	SearchRateLimitDuration = int64(GetEnvOrDefault("SEARCH_RATE_LIMIT_DURATION", 60))
	initConstantEnv()
}

func commandLineFlagProvided(name string) bool {
	provided := false
	flag.Visit(func(current *flag.Flag) {
		if current.Name == name {
			provided = true
		}
	})
	return provided
}

func resolveListeningPort(environmentValue string, flagValue int, flagExplicit bool) (int, error) {
	port := flagValue
	if !flagExplicit && strings.TrimSpace(environmentValue) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(environmentValue))
		if err != nil {
			return 0, fmt.Errorf("invalid PORT value: %w", err)
		}
		port = parsed
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("listening port must be between 1 and 65535")
	}
	return port, nil
}

func firstNonEmptyEnv(names ...string) string {
	for _, name := range names {
		value := os.Getenv(name)
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func initSMTPEnv() {
	if value := firstNonEmptyEnv("SMTP_SERVER", "SMTPServer"); value != "" {
		SMTPServer = strings.TrimSpace(value)
	}
	if value := firstNonEmptyEnv("SMTP_PORT", "SMTPPort"); value != "" {
		port, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || port < 1 || port > 65535 {
			log.Printf("warning: invalid SMTP port configuration; keeping port %d", SMTPPort)
		} else {
			SMTPPort = port
		}
	}
	if value := firstNonEmptyEnv("SMTP_ACCOUNT", "SMTPAccount"); value != "" {
		SMTPAccount = strings.TrimSpace(value)
	}
	if value := firstNonEmptyEnv("SMTP_TOKEN", "SMTPToken"); value != "" {
		SMTPToken = value
	}
	if value := firstNonEmptyEnv("SMTP_FROM", "SMTPFrom"); value != "" {
		SMTPFrom = strings.TrimSpace(value)
	}

	SMTPSSLEnabled = GetEnvOrDefaultBool("SMTP_SSL_ENABLED", GetEnvOrDefaultBool("SMTPSSLEnabled", SMTPSSLEnabled))
	SMTPStartTLSEnabled = GetEnvOrDefaultBool(
		"SMTP_STARTTLS_ENABLED",
		GetEnvOrDefaultBool("SMTP_STARTTLS_ENABLE", GetEnvOrDefaultBool("SMTPStartTLSEnabled", SMTPStartTLSEnabled)),
	)
	SMTPInsecureSkipVerify = GetEnvOrDefaultBool(
		"SMTP_INSECURE_SKIP_VERIFY",
		GetEnvOrDefaultBool("SMTP_TLS_INSECURE_SKIP_VERIFY", GetEnvOrDefaultBool("SMTPInsecureSkipVerify", SMTPInsecureSkipVerify)),
	)
	SMTPForceAuthLogin = GetEnvOrDefaultBool("SMTP_FORCE_AUTH_LOGIN", GetEnvOrDefaultBool("SMTPForceAuthLogin", SMTPForceAuthLogin))
}

func initTurnstileEnv() {
	TurnstileCheckEnabled = GetEnvOrDefaultBool("TURNSTILE_CHECK_ENABLED", TurnstileCheckEnabled)
	if value := firstNonEmptyEnv("TURNSTILE_SITE_KEY"); value != "" {
		TurnstileSiteKey = strings.TrimSpace(value)
	}
	if value := firstNonEmptyEnv("TURNSTILE_SECRET_KEY"); value != "" {
		TurnstileSecretKey = value
	}
	if value := firstNonEmptyEnv("TURNSTILE_ALLOWED_HOSTNAMES"); value != "" {
		TurnstileAllowedHostnames = strings.TrimSpace(value)
	}
}

func initGitHubOAuthEnv() {
	GitHubOAuthEnabled = GetEnvOrDefaultBool("GITHUB_OAUTH_ENABLED", GitHubOAuthEnabled)
	if value := firstNonEmptyEnv("GITHUB_OAUTH_CLIENT_ID", "GITHUB_CLIENT_ID"); value != "" {
		GitHubClientId = strings.TrimSpace(value)
	}
	if value := firstNonEmptyEnv("GITHUB_OAUTH_CLIENT_SECRET", "GITHUB_CLIENT_SECRET"); value != "" {
		GitHubClientSecret = value
	}
}

// EnvManagedSecretOptionKeys are option keys that must never be accepted from
// the admin API or trusted from the options table. Process environment (e.g.
// Cloud Run + Secret Manager) takes precedence whenever it is configured.
//
// SMTPToken and GitHubClientSecret are intentionally NOT in this set: they are
// env-preferred but DB-writable. When the corresponding environment variable is
// present it still wins (see ApplyEnvManagedSecrets), otherwise the value
// persisted through the admin settings UI is used. They are still scrubbed from
// OptionMap so the settings API never echoes them back.
var EnvManagedSecretOptionKeys = map[string]struct{}{
	"StripeApiSecret":           {},
	"StripeWebhookSecret":       {},
	"AWS_SES_ACCESS_KEY_ID":     {},
	"AWS_SES_SECRET_ACCESS_KEY": {},
	"AWS_SES_SESSION_TOKEN":     {},
}

func IsEnvManagedSecretOptionKey(key string) bool {
	_, ok := EnvManagedSecretOptionKeys[key]
	return ok
}

// ApplyEnvManagedSecrets re-applies environment-backed secrets after option
// map / database load so persisted rows cannot clobber Secret Manager values.
// Also refreshes non-secret configured-status flags used by the admin UI.
func ApplyEnvManagedSecrets() {
	// Strictly managed secrets: database values are never trusted for these keys.
	if value := firstNonEmptyEnv("GITHUB_OAUTH_CLIENT_SECRET", "GITHUB_CLIENT_SECRET"); value != "" {
		GitHubClientSecret = value
	} else if _, loaded := os.LookupEnv("GITHUB_OAUTH_CLIENT_SECRET"); loaded {
		GitHubClientSecret = ""
	} else if _, loaded := os.LookupEnv("GITHUB_CLIENT_SECRET"); loaded {
		GitHubClientSecret = ""
	}

	// Turnstile is env-preferred and DB-writable. A Secret Manager value wins
	// when present, while a Dashboard-saved value remains usable otherwise.
	if value := firstNonEmptyEnv("TURNSTILE_SECRET_KEY"); value != "" {
		TurnstileSecretKey = value
	}

	// SMTPToken is env-preferred and DB-writable. A Secret Manager value wins
	// when present, while a Dashboard-saved value remains usable otherwise.
	if value := firstNonEmptyEnv("SMTP_TOKEN", "SMTPToken"); value != "" {
		SMTPToken = value
	}

	// Public fields: when present in env, win over DB so Cloud Run can fully
	// configure login/bot-protection/email without admin API writes.
	if value := firstNonEmptyEnv("GITHUB_OAUTH_CLIENT_ID", "GITHUB_CLIENT_ID"); value != "" {
		GitHubClientId = strings.TrimSpace(value)
	}
	if _, loaded := os.LookupEnv("GITHUB_OAUTH_ENABLED"); loaded {
		GitHubOAuthEnabled = GetEnvOrDefaultBool("GITHUB_OAUTH_ENABLED", GitHubOAuthEnabled)
	}
	if value := firstNonEmptyEnv("TURNSTILE_SITE_KEY"); value != "" {
		TurnstileSiteKey = strings.TrimSpace(value)
	}
	if value := firstNonEmptyEnv("TURNSTILE_ALLOWED_HOSTNAMES"); value != "" {
		TurnstileAllowedHostnames = strings.TrimSpace(value)
	}
	if _, loaded := os.LookupEnv("TURNSTILE_CHECK_ENABLED"); loaded {
		TurnstileCheckEnabled = GetEnvOrDefaultBool("TURNSTILE_CHECK_ENABLED", TurnstileCheckEnabled)
	}
	if value := firstNonEmptyEnv("SMTP_SERVER", "SMTPServer"); value != "" {
		SMTPServer = strings.TrimSpace(value)
	}
	if value := firstNonEmptyEnv("SMTP_PORT", "SMTPPort"); value != "" {
		if port, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && port >= 1 && port <= 65535 {
			SMTPPort = port
		}
	}
	if value := firstNonEmptyEnv("SMTP_ACCOUNT", "SMTPAccount"); value != "" {
		SMTPAccount = strings.TrimSpace(value)
	}
	if value := firstNonEmptyEnv("SMTP_FROM", "SMTPFrom"); value != "" {
		SMTPFrom = strings.TrimSpace(value)
	}
	if _, loaded := os.LookupEnv("SMTP_STARTTLS_ENABLED"); loaded || envKeyPresent("SMTP_STARTTLS_ENABLE") || envKeyPresent("SMTPStartTLSEnabled") {
		SMTPStartTLSEnabled = GetEnvOrDefaultBool(
			"SMTP_STARTTLS_ENABLED",
			GetEnvOrDefaultBool("SMTP_STARTTLS_ENABLE", GetEnvOrDefaultBool("SMTPStartTLSEnabled", SMTPStartTLSEnabled)),
		)
	}
	if _, loaded := os.LookupEnv("SMTP_SSL_ENABLED"); loaded || envKeyPresent("SMTPSSLEnabled") {
		SMTPSSLEnabled = GetEnvOrDefaultBool("SMTP_SSL_ENABLED", GetEnvOrDefaultBool("SMTPSSLEnabled", SMTPSSLEnabled))
	}

	OptionMapRWMutex.Lock()
	defer OptionMapRWMutex.Unlock()
	// Never expose secret values through OptionMap / settings APIs.
	OptionMap["GitHubClientSecret"] = ""
	OptionMap["TurnstileSecretKey"] = ""
	OptionMap["SMTPToken"] = ""
	OptionMap["GitHubClientId"] = GitHubClientId
	OptionMap["GitHubOAuthEnabled"] = strconv.FormatBool(GitHubOAuthEnabled)
	OptionMap["TurnstileSiteKey"] = TurnstileSiteKey
	OptionMap["TurnstileAllowedHostnames"] = TurnstileAllowedHostnames
	OptionMap["TurnstileCheckEnabled"] = strconv.FormatBool(TurnstileCheckEnabled)
	OptionMap["SMTPServer"] = SMTPServer
	OptionMap["SMTPPort"] = strconv.Itoa(SMTPPort)
	OptionMap["SMTPAccount"] = SMTPAccount
	OptionMap["SMTPFrom"] = SMTPFrom
	OptionMap["SMTPSSLEnabled"] = strconv.FormatBool(SMTPSSLEnabled)
	OptionMap["SMTPStartTLSEnabled"] = strconv.FormatBool(SMTPStartTLSEnabled)
	OptionMap["GitHubClientSecretConfigured"] = strconv.FormatBool(strings.TrimSpace(GitHubClientSecret) != "")
	OptionMap["TurnstileSecretKeyConfigured"] = strconv.FormatBool(strings.TrimSpace(TurnstileSecretKey) != "")
	OptionMap["SMTPTokenConfigured"] = strconv.FormatBool(SMTPToken != "")
}

func envKeyPresent(name string) bool {
	_, loaded := os.LookupEnv(name)
	return loaded
}

// GitHubOAuthReady reports whether GitHub login can actually complete.
func GitHubOAuthReady() bool {
	return strings.TrimSpace(GitHubClientId) != "" && strings.TrimSpace(GitHubClientSecret) != ""
}

// GitHubOAuthActive is the fail-closed public status flag.
func GitHubOAuthActive() bool {
	return GitHubOAuthEnabled && GitHubOAuthReady()
}

// ensureSQLiteParentDir creates the directory for a file-based SQLite DSN.
// Paths like "./data/new-api.db" fail with SQLITE_CANTOPEN (14) if "data/" is missing;
// on Windows that error is often surface as "out of memory".
func ensureSQLiteParentDir(sqlitePath string) {
	if sqlitePath == "" {
		return
	}
	// Skip pure memory DSNs
	lower := strings.ToLower(sqlitePath)
	if strings.Contains(lower, "mode=memory") || strings.HasPrefix(lower, ":memory:") {
		return
	}
	// Strip query string (?_busy_timeout=...)
	pathOnly := sqlitePath
	if i := strings.Index(pathOnly, "?"); i >= 0 {
		pathOnly = pathOnly[:i]
	}
	// file: URI form
	if strings.HasPrefix(pathOnly, "file:") {
		pathOnly = strings.TrimPrefix(pathOnly, "file:")
	}
	dir := filepath.Dir(pathOnly)
	if dir == "" || dir == "." {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("warning: could not create SQLite directory %q: %v", dir, err)
	}
}

func initConstantEnv() {
	constant.StreamingTimeout = GetEnvOrDefault("STREAMING_TIMEOUT", 300)
	constant.DifyDebug = GetEnvOrDefaultBool("DIFY_DEBUG", true)
	constant.MaxFileDownloadMB = GetEnvOrDefault("MAX_FILE_DOWNLOAD_MB", 64)
	constant.StreamScannerMaxBufferMB = GetEnvOrDefault("STREAM_SCANNER_MAX_BUFFER_MB", 128)
	// MaxRequestBodyMB 请求体最大大小（解压后），用于防止超大请求/zip bomb导致内存暴涨
	constant.MaxRequestBodyMB = GetEnvOrDefault("MAX_REQUEST_BODY_MB", 128)
	constant.AnonymousRequestBodyLimitKB = GetEnvOrDefault("ANONYMOUS_REQUEST_BODY_LIMIT_KB", 512)
	// ForceStreamOption 覆盖请求参数，强制返回usage信息
	constant.ForceStreamOption = GetEnvOrDefaultBool("FORCE_STREAM_OPTION", true)
	constant.CountToken = GetEnvOrDefaultBool("CountToken", true)
	constant.GetMediaToken = GetEnvOrDefaultBool("GET_MEDIA_TOKEN", true)
	constant.GetMediaTokenNotStream = GetEnvOrDefaultBool("GET_MEDIA_TOKEN_NOT_STREAM", false)
	constant.UpdateTask = GetEnvOrDefaultBool("UPDATE_TASK", true)
	constant.AzureDefaultAPIVersion = GetEnvOrDefaultString("AZURE_DEFAULT_API_VERSION", "2025-04-01-preview")
	constant.NotifyLimitCount = GetEnvOrDefault("NOTIFY_LIMIT_COUNT", 2)
	constant.NotificationLimitDurationMinute = GetEnvOrDefault("NOTIFICATION_LIMIT_DURATION_MINUTE", 10)
	// GenerateDefaultToken 是否生成初始令牌，默认关闭。
	constant.GenerateDefaultToken = GetEnvOrDefaultBool("GENERATE_DEFAULT_TOKEN", false)
	// 是否启用错误日志
	constant.ErrorLogEnabled = GetEnvOrDefaultBool("ERROR_LOG_ENABLED", false)
	// 任务轮询时查询的最大数量
	constant.TaskQueryLimit = GetEnvOrDefault("TASK_QUERY_LIMIT", 1000)
	// 异步任务超时时间（分钟），超过此时间未完成的任务将被标记为失败并退款。0 表示禁用。
	constant.TaskTimeoutMinutes = GetEnvOrDefault("TASK_TIMEOUT_MINUTES", 1440)

	soraPatchStr := GetEnvOrDefaultString("TASK_PRICE_PATCH", "")
	if soraPatchStr != "" {
		var taskPricePatches []string
		soraPatches := strings.Split(soraPatchStr, ",")
		for _, patch := range soraPatches {
			trimmedPatch := strings.TrimSpace(patch)
			if trimmedPatch != "" {
				taskPricePatches = append(taskPricePatches, trimmedPatch)
			}
		}
		constant.TaskPricePatches = taskPricePatches
	}

	// Initialize trusted redirect domains for URL validation
	trustedDomainsStr := GetEnvOrDefaultString("TRUSTED_REDIRECT_DOMAINS", "")
	var trustedDomains []string
	domains := strings.Split(trustedDomainsStr, ",")
	for _, domain := range domains {
		trimmedDomain := strings.TrimSpace(domain)
		if trimmedDomain != "" {
			// Normalize domain to lowercase
			trustedDomains = append(trustedDomains, strings.ToLower(trimmedDomain))
		}
	}
	constant.TrustedRedirectDomains = trustedDomains
}
