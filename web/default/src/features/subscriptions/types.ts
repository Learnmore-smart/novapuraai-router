import { z } from 'zod'

// ============================================================================
// Subscription Plan Schema & Types
// ============================================================================

export const subscriptionPlanSchema = z.object({
  id: z.number(),
  title: z.string(),
  subtitle: z.string().optional(),
  price_amount: z.number(),
  currency: z.string().default('USD'),
  duration_unit: z.enum(['year', 'month', 'day', 'hour', 'custom']),
  duration_value: z.number(),
  custom_seconds: z.number().optional(),
  quota_reset_period: z.enum(['never', 'daily', 'weekly', 'monthly', 'custom']),
  quota_reset_custom_seconds: z.number().optional(),
  enabled: z.boolean(),
  sort_order: z.number(),
  allow_balance_pay: z.boolean().optional().default(true),
  allow_wallet_overflow: z.boolean().optional().default(true),
  max_purchase_per_user: z.number(),
  total_amount: z.number(),
  upgrade_group: z.string().optional(),
  downgrade_group: z.string().optional(),
  stripe_price_id: z.string().optional(),
  creem_product_id: z.string().optional(),
  waffo_pancake_product_id: z.string().optional(),
  // NovaPura v2 dual-currency pricing + billing options (optional for
  // back-compat with plans created before these fields existed).
  price_amount_cny: z.number().optional(),
  price_amount_usd: z.number().optional(),
  auto_renew: z.boolean().optional(),
  prepaid_months: z.string().optional(),
  renewal_price_cny: z.number().optional(),
  renewal_price_usd: z.number().optional(),
})

export type SubscriptionPlan = z.infer<typeof subscriptionPlanSchema>

export interface PlanRecord {
  plan: SubscriptionPlan
}

// ============================================================================
// User Subscription Schema & Types
// ============================================================================

export const userSubscriptionSchema = z.object({
  id: z.number(),
  user_id: z.number(),
  plan_id: z.number(),
  status: z.string(),
  source: z.string().optional(),
  start_time: z.number(),
  end_time: z.number(),
  amount_total: z.number(),
  amount_used: z.number(),
  next_reset_time: z.number().optional(),
  // NovaPura lifecycle / Stripe linkage fields (optional for back-compat with
  // subscriptions created before these columns existed).
  stripe_subscription_id: z.string().optional(),
  stripe_customer_id: z.string().optional(),
  billing_cycle_anchor: z.number().optional(),
  cancel_at_period_end: z.boolean().optional(),
  coupon_id: z.number().nullable().optional(),
  coupon_redemption_id: z.number().nullable().optional(),
})

export type UserSubscription = z.infer<typeof userSubscriptionSchema>

export interface UserSubscriptionRecord {
  subscription: UserSubscription
}

// NovaPura Phase 3: the /api/subscription/self endpoint returns a display DTO
// for the user's most-recent active subscription. It embeds UserSubscription
// and adds display-only fields resolved from the plan / status.
export const subscriptionSelfDtoSchema = userSubscriptionSchema.extend({
  // Mirror fields also present on UserSubscription; redeclared here so the
  // DTO type is self-documenting and survives even if the base schema drops them.
  stripe_subscription_id: z.string().optional(),
  stripe_customer_id: z.string().optional(),
  cancel_at_period_end: z.boolean().optional(),
  coupon_id: z.number().nullable().optional(),
  coupon_redemption_id: z.number().nullable().optional(),
  billing_cycle_anchor: z.number().optional(),
  // Display-only fields resolved by the backend from the plan / status.
  currency: z.string().optional(),
  next_renewal_date: z.number().optional(),
  is_auto_renew: z.boolean().optional(),
  display_status: z.string().optional(),
})

export type SubscriptionSelfDto = z.infer<typeof subscriptionSelfDtoSchema>

// ============================================================================
// API Request/Response Types
// ============================================================================

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface PlanPayload {
  plan: Partial<SubscriptionPlan>
}

export interface SubscriptionPayRequest {
  plan_id: number
  payment_method?: string
}

export interface SubscriptionPayResponse {
  success: boolean
  message?: string
  data?: {
    // Stripe-style hosted checkout link.
    pay_link?: string
    // Waffo Pancake / Creem hosted checkout URL.
    checkout_url?: string
    // Pancake-only: order metadata + self-service buyer session token,
    // surfaced for future flows (refund / cancel from new-api's own UI).
    session_id?: string
    expires_at?: number | string
    order_id?: string
    token?: string
    token_expires_at?: number | string
  }
  url?: string
}

export interface CreateUserSubscriptionRequest {
  plan_id: number
}

export interface ResetUserSubscriptionsRequest {
  plan_id: number
  advance_reset_time: boolean
}

export interface ResetPlanSubscriptionsRequest {
  advance_reset_time: boolean
}

export interface SubscriptionResetResult {
  plan_id: number
  matched_count: number
  reset_count: number
  user_count: number
  advance_reset_time: boolean
}

// ============================================================================
// Self Subscription Data (user-facing)
// ============================================================================

export interface SelfSubscriptionData {
  billing_preference: string
  subscriptions: UserSubscriptionRecord[]
  all_subscriptions: UserSubscriptionRecord[]
  // NovaPura Phase 3: display DTO for the most-recent active subscription
  // (null when the user has no active subscription).
  current_subscription?: SubscriptionSelfDto | null
  // NovaPura resolved default currency (CNY/USD).
  default_currency?: string
}

// ============================================================================
// Dialog Types
// ============================================================================

export type SubscriptionsDialogType =
  | 'create'
  | 'update'
  | 'toggle-status'
  | 'reset-subscriptions'

// ============================================================================
// NovaPura v2 Checkout, Coupon Validation & Customer Portal
// ============================================================================

export type SubscriptionCheckoutMode = 'auto_renew' | 'prepaid'
export type SubscriptionCheckoutCurrency = 'CNY' | 'USD'

export const subscriptionCheckoutRequestSchema = z.object({
  plan_id: z.number(),
  mode: z.enum(['auto_renew', 'prepaid']),
  currency: z.enum(['CNY', 'USD']),
  prepaid_months: z.number().optional(),
  coupon_code: z.string().optional(),
  success_url: z.string().optional(),
  cancel_url: z.string().optional(),
})

export type SubscriptionCheckoutRequest = z.infer<
  typeof subscriptionCheckoutRequestSchema
>

export interface SubscriptionCheckoutData {
  checkout_url: string
  order_id: string
}

export const couponValidationResponseSchema = z.object({
  valid: z.boolean(),
  reason: z.string(),
  coupon_name: z.string(),
  percent_off: z.number(),
  duration_months: z.number(),
  original_price: z.number(),
  discount_amount: z.number(),
  final_amount: z.number(),
  next_renewal_price: z.number(),
  currency: z.string(),
})

export type CouponValidationResponse = z.infer<
  typeof couponValidationResponseSchema
>

export interface SubscriptionPortalData {
  url: string
}
