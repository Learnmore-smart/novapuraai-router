import { api } from '@/lib/api'

import type {
  ApiResponse,
  PlanRecord,
  PlanPayload,
  UserSubscriptionRecord,
  CreateUserSubscriptionRequest,
  ResetUserSubscriptionsRequest,
  ResetPlanSubscriptionsRequest,
  SubscriptionResetResult,
  SubscriptionPayResponse,
  SubscriptionPayRequest,
  SelfSubscriptionData,
  SubscriptionCheckoutRequest,
  SubscriptionCheckoutData,
  CouponValidationResponse,
  SubscriptionPortalData,
} from './types'

// ============================================================================
// Admin Plan Management
// ============================================================================

export async function getAdminPlans(): Promise<ApiResponse<PlanRecord[]>> {
  const res = await api.get('/api/subscription/admin/plans')
  return res.data
}

export async function createPlan(
  data: PlanPayload
): Promise<ApiResponse<PlanRecord>> {
  const res = await api.post('/api/subscription/admin/plans', data)
  return res.data
}

export async function updatePlan(
  id: number,
  data: PlanPayload
): Promise<ApiResponse<PlanRecord>> {
  const res = await api.put(`/api/subscription/admin/plans/${id}`, data)
  return res.data
}

export async function patchPlanStatus(
  id: number,
  enabled: boolean
): Promise<ApiResponse> {
  const res = await api.patch(`/api/subscription/admin/plans/${id}`, {
    enabled,
  })
  return res.data
}

// ============================================================================
// Admin User Subscription Management
// ============================================================================

export async function getUserSubscriptions(
  userId: number
): Promise<ApiResponse<UserSubscriptionRecord[]>> {
  const res = await api.get(
    `/api/subscription/admin/users/${userId}/subscriptions`
  )
  return res.data
}

export async function createUserSubscription(
  userId: number,
  data: CreateUserSubscriptionRequest
): Promise<ApiResponse<{ message?: string }>> {
  const res = await api.post(
    `/api/subscription/admin/users/${userId}/subscriptions`,
    data
  )
  return res.data
}

export async function invalidateUserSubscription(
  subId: number
): Promise<ApiResponse<{ message?: string }>> {
  const res = await api.post(
    `/api/subscription/admin/user_subscriptions/${subId}/invalidate`
  )
  return res.data
}

export async function deleteUserSubscription(
  subId: number
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/subscription/admin/user_subscriptions/${subId}`
  )
  return res.data
}

export async function resetUserSubscriptionsByPlan(
  userId: number,
  data: ResetUserSubscriptionsRequest
): Promise<ApiResponse<SubscriptionResetResult>> {
  const res = await api.post(
    `/api/subscription/admin/users/${userId}/subscriptions/reset`,
    data
  )
  return res.data
}

export async function resetPlanSubscriptions(
  planId: number,
  data: ResetPlanSubscriptionsRequest
): Promise<ApiResponse<SubscriptionResetResult>> {
  const res = await api.post(
    `/api/subscription/admin/plans/${planId}/subscriptions/reset`,
    data
  )
  return res.data
}

// ============================================================================
// User-facing Subscription Payment
// ============================================================================

export async function paySubscriptionStripe(
  data: SubscriptionPayRequest
): Promise<SubscriptionPayResponse> {
  const res = await api.post('/api/subscription/stripe/pay', data)
  return res.data
}

export async function paySubscriptionCreem(
  data: SubscriptionPayRequest
): Promise<SubscriptionPayResponse> {
  const res = await api.post('/api/subscription/creem/pay', data)
  return res.data
}

export async function paySubscriptionWaffoPancake(
  data: SubscriptionPayRequest
): Promise<SubscriptionPayResponse> {
  const res = await api.post('/api/subscription/waffo-pancake/pay', data)
  return res.data
}

export async function paySubscriptionBalance(
  data: SubscriptionPayRequest
): Promise<SubscriptionPayResponse> {
  const res = await api.post('/api/subscription/balance/pay', data)
  return res.data
}

// Mints a Pancake OnetimeProduct (see controller for the OnetimeProduct vs
// SubscriptionProduct rationale) using persisted creds + StoreID.
export async function createWaffoPancakeSubscriptionProduct(data: {
  name: string
  amount: string
}): Promise<
  ApiResponse<{ product_id: string; product_name: string; store_id: string }>
> {
  const res = await api.post(
    '/api/option/waffo-pancake/subscription-product',
    data
  )
  return res.data
}

// Returns the OnetimeProducts in the saved Pancake store; empty when the
// gateway isn't fully configured.
export async function listWaffoPancakeSubscriptionProductOptions(): Promise<
  ApiResponse<{
    store_id: string
    products: { id: string; name: string; status: string }[]
  }>
> {
  const res = await api.get(
    '/api/option/waffo-pancake/subscription-product-options'
  )
  return res.data
}

export async function paySubscriptionEpay(
  data: SubscriptionPayRequest & { payment_method: string }
): Promise<SubscriptionPayResponse & { url?: string }> {
  const res = await api.post('/api/subscription/epay/pay', data)
  return {
    ...res.data,
    url: res.data.url || (res as unknown as { url?: string }).url,
  }
}

// ============================================================================
// User Self Subscriptions
// ============================================================================

export async function getSelfSubscriptions(): Promise<
  ApiResponse<UserSubscriptionRecord[]>
> {
  const res = await api.get('/api/subscription/self')
  return res.data
}

export async function getSelfSubscriptionFull(): Promise<
  ApiResponse<SelfSubscriptionData>
> {
  const res = await api.get('/api/subscription/self')
  return res.data
}

export async function getPublicPlans(): Promise<ApiResponse<PlanRecord[]>> {
  const res = await api.get('/api/subscription/plans')
  return res.data
}

export async function updateBillingPreference(
  preference: string
): Promise<ApiResponse<{ billing_preference?: string }>> {
  const res = await api.put('/api/subscription/self/preference', {
    billing_preference: preference,
  })
  return res.data
}

export async function getGroups(): Promise<ApiResponse<string[]>> {
  const res = await api.get('/api/group')
  return res.data
}

// ============================================================================
// NovaPura v2 Stripe Checkout, Coupon Validation & Customer Portal
// ============================================================================

// V2 checkout — supports auto_renew + prepaid, coupon, currency selection.
// The frontend never sends a payment amount; the backend selects the Stripe
// Price based on plan_id + currency + mode.
export async function createSubscriptionCheckout(
  data: SubscriptionCheckoutRequest
): Promise<ApiResponse<SubscriptionCheckoutData>> {
  const res = await api.post('/api/subscription/stripe/checkout', data)
  return res.data
}

// Validate a coupon code (read-only) — returns the price breakdown the
// frontend uses to render the coupon confirmation modal.
export async function validateSubscriptionCoupon(data: {
  code: string
  plan_id: number
  mode: 'auto_renew' | 'prepaid'
  currency: 'CNY' | 'USD'
  prepaid_months?: number
}): Promise<ApiResponse<CouponValidationResponse>> {
  const res = await api.post('/api/subscription/coupons/validate', data)
  return res.data
}

// Create a Stripe Customer Portal session for subscription management
// (update payment method, cancel, view invoices).
export async function createSubscriptionPortal(): Promise<
  ApiResponse<SubscriptionPortalData>
> {
  const res = await api.post('/api/subscription/stripe/portal', {})
  return res.data
}
