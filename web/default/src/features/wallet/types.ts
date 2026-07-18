import type { BillingCurrency } from '@/lib/billing-currency'

// ============================================================================
// Wallet Type Definitions
// ============================================================================

/**
 * Generic API response
 */
export interface ApiResponse<T = unknown> {
  success?: boolean
  message?: string
  data?: T
}

/**
 * Standard API response types
 */
export type TopupInfoResponse = ApiResponse<TopupInfo>
export type RedemptionResponse = ApiResponse<number>
export type AmountResponse = ApiResponse<string>
export type PaymentResponse = ApiResponse<Record<string, unknown>> & {
  url?: string
}
export type StripePaymentResponse = ApiResponse<{ pay_link: string }>
export type AffiliateCodeResponse = ApiResponse<string>
export type AffiliateTransferResponse = ApiResponse
export type CreemPaymentResponse = ApiResponse<{ checkout_url: string }>
export type WaffoPaymentResponse = ApiResponse<
  { payment_url?: string } | string
>
export type WaffoPancakePaymentResponse = ApiResponse<
  | {
      checkout_url?: string
      session_id?: string
      expires_at?: number | string
      order_id?: string
      // Self-service session token + expiry — surfaced by the backend so
      // future flows (refund / cancel from new-api's own UI) can use them
      // without re-issuing checkout. Not consumed by the current handler.
      token?: string
      token_expires_at?: number | string
    }
  | string
>

/**
 * Creem product configuration
 */
export interface CreemProduct {
  /** Product display name */
  name: string
  /** Creem product ID */
  productId: string
  /** Product price */
  price: number
  /** Quota amount to credit */
  quota: number
  /** Currency (USD or EUR) */
  currency: 'USD' | 'EUR'
}

/**
 * Creem payment request
 */
export interface CreemPaymentRequest {
  /** Creem product ID */
  product_id: string
  /** Payment method identifier */
  payment_method: 'creem'
}

/**
 * Payment method configuration
 */
export interface PaymentMethod {
  /** Display name of payment method */
  name: string
  /** Payment method type identifier */
  type: string
  /** Legacy optional color for UI display */
  color?: string
  /** Minimum topup amount for this payment method */
  min_topup?: number
  /** Optional react-icons component name or safe icon URL */
  icon?: string
}

/**
 * Waffo payment method configuration
 */
export interface WaffoPayMethod {
  /** Display name of payment method */
  name: string
  /** Optional icon path */
  icon?: string
  /** Waffo pay method type */
  payMethodType?: string
  /** Waffo pay method name */
  payMethodName?: string
}

/**
 * Topup configuration information
 */
export interface TopupInfo {
  /** Whether online topup is enabled */
  enable_online_topup: boolean
  /** Whether Stripe topup is enabled */
  enable_stripe_topup: boolean
  /** Available payment methods */
  pay_methods: PaymentMethod[]
  /** Minimum topup amount for online topup */
  min_topup: number
  /** Minimum topup amount for Stripe */
  stripe_min_topup: number
  /** Preset amount options */
  amount_options: number[]
  /** Discount rates by amount */
  discount: Record<number, number>
  /** Optional topup link for purchasing codes */
  topup_link?: string
  /** Whether Creem topup is enabled */
  enable_creem_topup?: boolean
  /** Available Creem products */
  creem_products?: CreemProduct[]
  /** Whether Waffo topup is enabled */
  enable_waffo_topup?: boolean
  /** Available Waffo payment methods */
  waffo_pay_methods?: WaffoPayMethod[]
  /** Minimum topup amount for Waffo */
  waffo_min_topup?: number
  /** Whether Waffo Pancake topup is enabled */
  enable_waffo_pancake_topup?: boolean
  /** Minimum topup amount for Waffo Pancake */
  waffo_pancake_min_topup?: number
  /** Whether redemption code usage is enabled */
  enable_redemption?: boolean
}

/**
 * Preset amount option with optional discount
 */
export interface PresetAmount {
  /** Preset amount value */
  value: number
  /** Optional discount rate (0-1) */
  discount?: number
}

/**
 * Redemption code request
 */
export interface RedemptionRequest {
  /** Redemption code key */
  key: string
}

/**
 * Payment request parameters
 */
export interface PaymentRequest {
  /** Topup amount */
  amount: number
  /** Payment method identifier */
  payment_method: string
}

/**
 * Waffo payment request parameters
 */
export interface WaffoPaymentRequest {
  /** Topup amount */
  amount: number
  /** Optional server-side Waffo payment method index */
  pay_method_index?: number
}

/**
 * Waffo Pancake payment request parameters
 */
export interface WaffoPancakePaymentRequest {
  /** Topup amount */
  amount: number
}

/**
 * Amount calculation request
 */
export interface AmountRequest {
  /** Topup amount to calculate */
  amount: number
}

/**
 * Affiliate quota transfer request
 */
export interface AffiliateTransferRequest {
  /** Quota amount to transfer */
  quota: number
}

/**
 * User wallet data
 */
export interface UserWalletData {
  /** User ID */
  id: number
  /** Username */
  username: string
  /** Current total spendable quota (cash + promo) */
  quota: number
  /** Gift / campaign balance (deducted first) */
  promo_quota?: number
  /** Cash top-up balance (derived: quota - promo_quota) */
  cash_quota?: number
  /** Total used quota */
  used_quota: number
  /** Total request count */
  request_count: number
  /** Affiliate quota (pending rewards) */
  aff_quota: number
  /** Total affiliate quota earned (historical) */
  aff_history_quota: number
  /** Number of successful affiliate invites */
  aff_count: number
  /** User group */
  group: string
}

/**
 * Topup record status
 */
export type TopupStatus = 'success' | 'pending' | 'expired'

/**
 * Topup billing record
 */
export interface TopupRecord {
  /** Record ID */
  id: number
  /** User ID */
  user_id: number
  /** Topup amount (quota) */
  amount: number
  /** Payment amount (actual money paid) */
  money: number
  /** Trade/order number */
  trade_no: string
  /** Payment method type */
  payment_method: string
  /** Creation timestamp */
  create_time: number
  /** Completion timestamp */
  complete_time?: number
  /** Payment status */
  status: TopupStatus
}

/**
 * Billing history response
 */
export interface BillingHistoryResponse {
  items: TopupRecord[]
  total: number
}

/**
 * Complete order request (admin only)
 */
export interface CompleteOrderRequest {
  trade_no: string
  reason: string
}

export type ShareSubmissionStatus = 'pending' | 'approved' | 'rejected'

export interface ShareSubmission {
  id: number
  user_id: number
  url: string
  platform: string
  note: string
  status: ShareSubmissionStatus
  reviewer_id: number
  review_reason: string
  amount: number
  created_at: number
  reviewed_at: number
}

export interface ShareSubmissionRequest {
  url: string
  platform: string
  note: string
}

export interface BillingTopupOffer {
  tier_id: number
  code: string
  name: string
  currency: BillingCurrency
  payment_amount_minor: number
  bonus_amount_minor: number
  total_credit_amount_minor: number
  payment_display: string
  bonus_display: string
  total_display: string
  available: boolean
  unavailable_reason?:
    | 'currency_unavailable'
    | 'tier_unavailable'
    | 'campaign_unavailable'
    | 'limit_reached'
    | 'budget_reached'
  recommended: boolean
  promo_expiry_days: number
  start_at: number
  end_at: number
}

export interface BillingTopupCampaign {
  id: number
  name: string
  enabled: boolean
  start_at: number
  end_at: number
  global_budget_micro_usd: number
  reserved_promo_micro_usd: number
  issued_promo_micro_usd: number
  per_user_limit: number
  default_promo_expiry_days: number
}

export interface BillingTopupConfig {
  selected_currency: BillingCurrency
  default_currency: BillingCurrency
  payment_methods_note: string
  sandbox: boolean
  campaign?: BillingTopupCampaign
  campaign_active: boolean
  repeatable: boolean
  offers: BillingTopupOffer[]
  api_balance: {
    total_quota: number
    promo_quota: number
    cash_quota: number
    currency: BillingCurrency
    total_amount_minor: number
    promo_amount_minor: number
    cash_amount_minor: number
    total_display: string
    promo_display: string
  }
  config: {
    enabled: boolean
    currencies: BillingCurrency[]
    default_currency: BillingCurrency
    min_max_major: Record<BillingCurrency, [number, number]>
  }
}

export interface BillingTopupCheckoutResult {
  order_id: string
  checkout_url: string
}

export interface BillingTopupQuote {
  tier_id: number
  currency: BillingCurrency
  amount_minor: number
  paid_credit_amount_minor: number
  promo_credit_amount_minor: number
  total_credit_amount_minor: number
  payment_display: string
  bonus_display: string
  total_display: string
}
