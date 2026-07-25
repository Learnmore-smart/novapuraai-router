export type SystemOption = {
  key: string
  value: string
}

export type SystemOptionKey = string

export type SystemOptionsResponse = {
  success: boolean
  message: string
  data: SystemOption[]
}

export type UpdateOptionRequest = {
  key: string
  value: string | boolean | number
}

export type UpdateOptionResponse = {
  success: boolean
  message: string
}

export type TransactionalEmailProvider = 'brevo' | 'ses'

export type EmailProviderHealth = {
  provider: TransactionalEmailProvider
  configured: boolean
  reachable: boolean
  ready: boolean
  credentials_configured?: boolean
  region_configured?: boolean
  sender_configured?: boolean
  sending_enabled?: boolean
  production_access?: boolean
  sandbox_restricted?: boolean
  failure_reason?: string
}

export type TransactionalEmailHealth = {
  selected_provider: TransactionalEmailProvider
  providers: EmailProviderHealth[]
  safe_retry_count: number
  manual_review_count: number
  latest_delivery?: {
    provider: TransactionalEmailProvider
    message_type: 'verification' | 'password_reset' | 'receipt' | 'notification'
    recipient_hash: string
    provider_message_id?: string
    status: 'sending' | 'sent' | 'failed' | 'retry_queued'
    failure_reason?: string
    created_at: string
  }
}

export type TransactionalEmailHealthResponse = {
  success: boolean
  message: string
  data: TransactionalEmailHealth
}

export type SwitchTransactionalEmailProviderResponse = {
  success: boolean
  message: string
  data?: { provider: TransactionalEmailProvider }
}

export type RetryTransactionalEmailResponse = {
  success: boolean
  message: string
  data: {
    processed: number
    sent: number
    queued: number
    failed: number
  }
}

export type SendTransactionalEmailTestResponse = {
  success: boolean
  message: string
  data?: {
    provider: TransactionalEmailProvider
    status: 'sent' | 'failed' | 'retry_queued'
  }
}

export type SESCredentialSource = 'none' | 'environment' | 'database'

export type SESCredentialStatus = {
  configured: boolean
  source: SESCredentialSource
  has_session_token: boolean
  region?: string
  from_address?: string
}

export type SESCredentialUpdateRequest = {
  access_key_id?: string
  secret_access_key?: string
  session_token?: string
  clear_session_token?: boolean
  region?: string
  from_address?: string
}

export type SESCredentialStatusResponse = {
  success: boolean
  message: string
  data: SESCredentialStatus
}

export type StripeCredentialEnvironment = 'test' | 'production'

export type StripeCredentialStatusResponse = {
  success: boolean
  message: string
  data: {
    secret_configured: boolean
    publishable_configured: boolean
    webhook_configured: boolean
  }
}

export type StripeCredentialUpdateRequest = {
  secret_key?: string
  publishable_key?: string
  webhook_secret?: string
}

export type SystemTaskStatus = 'pending' | 'running' | 'succeeded' | 'failed'

export type SystemTask<
  TPayload = Record<string, unknown>,
  TState = Record<string, unknown>,
  TResult = Record<string, unknown>,
> = {
  id: number
  task_id: string
  type: string
  status: SystemTaskStatus
  active_key?: string
  payload?: TPayload
  state?: TState
  result?: TResult
  error?: string
  locked_by?: string
  locked_until?: number
  created_at: number
  updated_at: number
}

export type LogCleanupTaskPayload = {
  target_timestamp: number
  batch_size: number
}

export type LogCleanupTaskState = {
  total: number
  processed: number
  progress: number
  remaining: number
}

export type LogCleanupTaskResult = {
  deleted_count: number
}

export type LogCleanupTask = SystemTask<
  LogCleanupTaskPayload,
  LogCleanupTaskState,
  LogCleanupTaskResult
>

export type SystemTaskResponse<TTask = SystemTask | null> = {
  success: boolean
  message: string
  data?: TTask
}

export type SystemTaskListResponse = {
  success: boolean
  message: string
  data?: SystemTask[]
}

export type SiteSettings = {
  Notice: string
  SystemName: string
  Logo: string
  Footer: string
  About: string
  HomePageContent: string
  ServerAddress: string
  'legal.user_agreement': string
  'legal.privacy_policy': string
  HeaderNavModules: string
  SidebarModulesAdmin: string
}

export type AuthSettings = {
  PasswordLoginEnabled: boolean
  PasswordRegisterEnabled: boolean
  EmailVerificationEnabled: boolean
  RegisterEnabled: boolean
  EmailDomainRestrictionEnabled: boolean
  EmailAliasRestrictionEnabled: boolean
  EmailDomainWhitelist: string
  ServerAddress: string
  GitHubOAuthEnabled: boolean
  GitHubClientId: string
  GitHubClientSecretConfigured: boolean
  'discord.enabled': boolean
  'discord.client_id': string
  'discord.client_secret': string
  'oidc.enabled': boolean
  'oidc.client_id': string
  'oidc.client_secret': string
  'oidc.well_known': string
  'oidc.authorization_endpoint': string
  'oidc.token_endpoint': string
  'oidc.user_info_endpoint': string
  TelegramOAuthEnabled: boolean
  TelegramBotToken: string
  TelegramBotName: string
  LinuxDOOAuthEnabled: boolean
  LinuxDOClientId: string
  LinuxDOClientSecret: string
  LinuxDOMinimumTrustLevel: string
  WeChatAuthEnabled: boolean
  WeChatServerAddress: string
  WeChatServerToken: string
  WeChatAccountQRCodeImageURL: string
  TurnstileCheckEnabled: boolean
  TurnstileSiteKey: string
  TurnstileSecretKeyConfigured: boolean
  TurnstileAllowedHostnames: string
  'passkey.enabled': boolean
  'passkey.rp_display_name': string
  'passkey.rp_id': string
  'passkey.origins': string
  'passkey.allow_insecure_origin': boolean
  'passkey.user_verification': 'required' | 'preferred' | 'discouraged'
  'passkey.attachment_preference': '' | 'platform' | 'cross-platform'
}

export type ContentSettings = {
  'console_setting.api_info': string
  'console_setting.announcements': string
  'console_setting.faq': string
  'console_setting.uptime_kuma_groups': string
  'console_setting.api_info_enabled': boolean
  'console_setting.announcements_enabled': boolean
  'console_setting.faq_enabled': boolean
  'console_setting.uptime_kuma_enabled': boolean
  DataExportEnabled: boolean
  DataExportDefaultTime: string
  DataExportInterval: number
  Chats: string
  DrawingEnabled: boolean
  MjNotifyEnabled: boolean
  MjAccountFilterEnabled: boolean
  MjForwardUrlEnabled: boolean
  MjModeClearEnabled: boolean
  MjActionCheckSuccessEnabled: boolean
}

export type ModelSettings = {
  'global.pass_through_request_enabled': boolean
  'global.thinking_model_blacklist': string
  'global.chat_completions_to_responses_policy': string
  'general_setting.ping_interval_enabled': boolean
  'general_setting.ping_interval_seconds': number
  'gemini.safety_settings': string
  'gemini.version_settings': string
  'gemini.supported_imagine_models': string
  'gemini.thinking_adapter_enabled': boolean
  'gemini.thinking_adapter_budget_tokens_percentage': number
  'gemini.function_call_thought_signature_enabled': boolean
  'gemini.remove_function_response_id_enabled': boolean
  'claude.model_headers_settings': string
  'claude.default_max_tokens': string
  'claude.thinking_adapter_enabled': boolean
  'claude.thinking_adapter_budget_tokens_percentage': number
  'grok.violation_deduction_enabled': boolean
  'grok.violation_deduction_amount': number
  ModelPrice: string
  ModelRatio: string
  CacheRatio: string
  CreateCacheRatio: string
  CompletionRatio: string
  ImageRatio: string
  AudioRatio: string
  AudioCompletionRatio: string
  ModelDiscount: string
  ExposeRatioEnabled: boolean
  'billing_setting.billing_mode': string
  'billing_setting.billing_expr': string
  'tool_price_setting.prices': string
  TopupGroupRatio: string
  GroupRatio: string
  UserUsableGroups: string
  GroupGroupRatio: string
  AutoGroups: string
  DefaultUseAutoGroup: boolean
  'group_ratio_setting.group_special_usable_group': string
  RetryTimes: number
  ChannelDisableThreshold: string
  AutomaticDisableChannelEnabled: boolean
  AutomaticEnableChannelEnabled: boolean
  AutomaticDisableKeywords: string
  AutomaticDisableStatusCodes: string
  AutomaticRetryStatusCodes: string
  'monitor_setting.auto_test_channel_enabled': boolean
  'monitor_setting.auto_test_channel_minutes': number
  'monitor_setting.channel_test_mode': 'scheduled_all' | 'passive_recovery'
  'channel_affinity_setting.enabled': boolean
  'channel_affinity_setting.switch_on_success': boolean
  'channel_affinity_setting.keep_on_channel_disabled': boolean
  'channel_affinity_setting.max_entries': number
  'channel_affinity_setting.default_ttl_seconds': number
  'channel_affinity_setting.rules': string
  'model_deployment.ionet.api_key': string
  'model_deployment.ionet.enabled': boolean
}

export type BillingSettings = {
  QuotaForNewUser: number
  PreConsumedQuota: number
  QuotaForInviter: number
  QuotaForInvitee: number
  TopUpLink: string
  'general_setting.docs_link': string
  'quota_setting.enable_free_model_pre_consume': boolean
  QuotaPerUnit: number
  USDExchangeRate: number
  'general_setting.quota_display_type': string
  'general_setting.custom_currency_symbol': string
  'general_setting.custom_currency_exchange_rate': number
  'general_setting.cad_exchange_rate': number
  DisplayInCurrencyEnabled: boolean
  DisplayTokenStatEnabled: boolean
  ModelPrice: string
  ModelRatio: string
  CacheRatio: string
  CreateCacheRatio: string
  CompletionRatio: string
  ImageRatio: string
  AudioRatio: string
  AudioCompletionRatio: string
  ModelDiscount: string
  ExposeRatioEnabled: boolean
  'billing_setting.billing_mode': string
  'billing_setting.billing_expr': string
  'tool_price_setting.prices': string
  TopupGroupRatio: string
  GroupRatio: string
  UserUsableGroups: string
  GroupGroupRatio: string
  AutoGroups: string
  DefaultUseAutoGroup: boolean
  'group_ratio_setting.group_special_usable_group': string
  PayAddress: string
  EpayId: string
  EpayKey: string
  Price: number
  MinTopUp: number
  CustomCallbackAddress: string
  PayMethods: string
  'payment_setting.amount_options': string
  'payment_setting.amount_discount': string
  StripePriceId: string
  StripeTopupProductID: string
  StripeTestTopupProductID: string
  StripeProdTopupProductID: string
  StripeTestAccountID: string
  StripeProdAccountID: string
  StripeRuntimeEnvironment: 'test' | 'production' | 'disabled'
  StripeTestSecretConfigured: boolean
  StripeTestPublishableConfigured: boolean
  StripeTestWebhookConfigured: boolean
  StripeProdSecretConfigured: boolean
  StripeProdPublishableConfigured: boolean
  StripeProdWebhookConfigured: boolean
  StripeTopupEnabled: boolean
  StripeUnitPrice: number
  StripeMinTopUp: number
  StripePromotionCodesEnabled: boolean
  CreemApiKey: string
  CreemWebhookSecret: string
  CreemTestMode: boolean
  CreemProducts: string
  WaffoEnabled: boolean
  WaffoApiKey: string
  WaffoPrivateKey: string
  WaffoPublicCert: string
  WaffoSandboxPublicCert: string
  WaffoSandboxApiKey: string
  WaffoSandboxPrivateKey: string
  WaffoSandbox: boolean
  WaffoMerchantId: string
  WaffoCurrency: string
  WaffoUnitPrice: number
  WaffoMinTopUp: number
  WaffoNotifyUrl: string
  WaffoReturnUrl: string
  WaffoPayMethods: string
  WaffoPancakeMerchantID: string
  WaffoPancakePrivateKey: string
  WaffoPancakeReturnURL: string
  // Bound by the operator through the catalog flow in the admin Pancake
  // section (saved via /api/option/waffo-pancake/save).
  WaffoPancakeStoreID: string
  WaffoPancakeProductID: string
  'checkin_setting.enabled': boolean
  'checkin_setting.min_quota': number
  'checkin_setting.max_quota': number
}

export type OperationsSettings = {
  DefaultCollapseSidebar: boolean
  DemoSiteEnabled: boolean
  SelfUseModeEnabled: boolean
  QuotaRemindThreshold: string
  SMTPServer: string
  SMTPPort: string
  SMTPAccount: string
  SMTPFrom: string
  SMTPTokenConfigured: boolean
  SMTPSSLEnabled: boolean
  SMTPStartTLSEnabled: boolean
  SMTPInsecureSkipVerify: boolean
  SMTPForceAuthLogin: boolean
  WorkerUrl: string
  WorkerValidKey: string
  WorkerAllowHttpImageRequestEnabled: boolean
  LogConsumeEnabled: boolean
  AffCommissionRate: number
  MinWithdrawalCents: number
  CommissionFreezeDays: number
  'performance_setting.disk_cache_enabled': boolean
  'performance_setting.disk_cache_threshold_mb': number
  'performance_setting.disk_cache_max_size_mb': number
  'performance_setting.disk_cache_path': string
  'performance_setting.monitor_enabled': boolean
  'performance_setting.monitor_cpu_threshold': number
  'performance_setting.monitor_memory_threshold': number
  'performance_setting.monitor_disk_threshold': number
  'perf_metrics_setting.enabled': boolean
  'perf_metrics_setting.flush_interval': number
  'perf_metrics_setting.bucket_time': 'hour' | 'minute' | '5min'
  'perf_metrics_setting.retention_days': number
}

export type SecuritySettings = {
  ModelRequestRateLimitEnabled: boolean
  ModelRequestRateLimitCount: number
  ModelRequestRateLimitSuccessCount: number
  ModelRequestRateLimitDurationMinutes: number
  ModelRequestRateLimitGroup: string
  CheckSensitiveEnabled: boolean
  CheckSensitiveOnPromptEnabled: boolean
  SensitiveWords: string
  'fetch_setting.enable_ssrf_protection': boolean
  'fetch_setting.allow_private_ip': boolean
  'fetch_setting.domain_filter_mode': boolean
  'fetch_setting.ip_filter_mode': boolean
  'fetch_setting.domain_list': string[]
  'fetch_setting.ip_list': string[]
  'fetch_setting.allowed_ports': number[]
  'fetch_setting.apply_ip_filter_for_domain': boolean
  'token_setting.max_user_tokens': number
}

export type UpstreamChannel = {
  id: number
  name: string
  base_url: string
  status: number
  type?: number
}

export type RatioType =
  | 'model_ratio'
  | 'completion_ratio'
  | 'cache_ratio'
  | 'create_cache_ratio'
  | 'image_ratio'
  | 'audio_ratio'
  | 'audio_completion_ratio'
  | 'model_price'
  | 'billing_mode'
  | 'billing_expr'

export type RatioDifference = {
  current: number | string | null
  upstreams: Record<string, number | string | 'same'>
  confidence: Record<string, boolean>
}

export type DifferencesMap = Record<
  string,
  Partial<Record<RatioType, RatioDifference>>
>

export type UpstreamChannelsResponse = {
  success: boolean
  message: string
  data: UpstreamChannel[]
}

export type UpstreamConfig = {
  id: number
  name: string
  base_url: string
  endpoint: string
}

export type FetchUpstreamRatiosRequest = {
  upstreams: UpstreamConfig[]
  timeout: number
}

export type TestResult = {
  name: string
  status: 'success' | 'error'
  error?: string
}

export type UpstreamRatiosResponse = {
  success: boolean
  message: string
  data: {
    differences: DifferencesMap
    test_results: TestResult[]
  }
}
