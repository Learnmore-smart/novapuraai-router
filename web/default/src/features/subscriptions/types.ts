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
})

export type UserSubscription = z.infer<typeof userSubscriptionSchema>

export interface UserSubscriptionRecord {
  subscription: UserSubscription
}

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
    reservation_id?: number | string
    reservation_expires_at?: number | string
    plan_id?: number
    plan_code?: string
    model?: string
    tier?: 'founder' | 'standard' | string
    price_id?: string
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
  subscription?: SubscriptionLifecycleSummary
  current_subscription?: SubscriptionLifecycleSummary
}

export interface SubscriptionFairUseLimits {
  peak_concurrency?: number
  concurrent_seconds_budget?: number
  window_minutes?: number
  success_requests_per_window?: number
  total_requests_per_window?: number
  lease_seconds?: number
  renew_seconds?: number
  recovery_seconds?: number
}

export interface SubscriptionLifecycleSummary {
  // Legacy metadata kept for cross-feature dashboard compatibility. The
  // subscription offer itself is not scoped to this value.
  model?: string
  stripe_status?: string
  stripe_customer_id?: string
  stripe_subscription_id?: string
  stripe_price_id?: string
  current_period_start?: number | string
  current_period_end?: number | string
  cancel_at_period_end?: boolean
  grace_period_end?: number | string | null
  price_tier?: 'founder' | 'standard' | string
  status?: string
}

export interface SubscriptionOffer {
  // Normalized public-offer state consumed by the UI.
  plan_id?: number
  active?: boolean
  pending?: boolean
  limit?: number
  remaining?: number
  sold_out?: boolean
  founder_claims_remaining?: number
  current_price_tier?: 'founder' | 'standard'
  current_price_minor?: number
  current_price_display?: string
  future_standard_price_minor?: number
  future_standard_price_display?: string
  currency?: string
  fair_use?: SubscriptionFairUseLimits
  reservation_expires_at?: number | string | null
  subscription?: SubscriptionLifecycleSummary

  // Raw fields returned by the sandbox recurring-offer service. Keeping these
  // optional lets the normalization boundary accept the backend contract
  // without leaking backend-specific names into components.
  // Legacy metadata is retained for consumers outside this feature; the offer
  // card intentionally does not present it as a plan restriction.
  model?: string
  enabled?: boolean
  code?: string
  title?: string
  subtitle?: string
  founder_price_id?: string
  standard_price_id?: string
  founder_amount_minor?: number
  standard_amount_minor?: number
  max_active_seats?: number
  founder_purchase_limit?: number
  active_seats?: number
  pending_seats?: number
  founder_claims_used?: number
}

export interface StripeSubscriptionRecord {
  plan_id?: number
  user_id?: number
  reservation_id?: number
  stripe_customer_id?: string
  stripe_subscription_id?: string
  stripe_checkout_session_id?: string
  stripe_price_id?: string
  user_subscription_id?: number
  tier?: string
  status?: string
  cancel_at_period_end?: boolean
  current_period_start?: number | string
  current_period_end?: number | string
  grace_until?: number | string | null
  ended_at?: number | string
}

export interface StripeSubscriptionReservation {
  plan_id?: number
  user_id?: number
  tier?: string
  status?: string
  expires_at?: number | string | null
}

export interface StripeSubscriptionSummary {
  enabled?: boolean
  plan_id?: number
  plan_code?: string
  currency?: string
  active_seats?: number
  max_seats?: number
  subscription?: StripeSubscriptionRecord | null
  reservation?: StripeSubscriptionReservation | null
}

export interface SubscriptionOfferEnvelope {
  offer?: SubscriptionOffer
}

// ============================================================================
// Dialog Types
// ============================================================================

export type SubscriptionsDialogType =
  | 'create'
  | 'update'
  | 'toggle-status'
  | 'reset-subscriptions'
