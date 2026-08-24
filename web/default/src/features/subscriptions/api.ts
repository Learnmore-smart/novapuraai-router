import { api, type ApiRequestConfig } from '@/lib/api'

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
  SubscriptionOffer,
  SubscriptionOfferEnvelope,
  StripeSubscriptionSummary,
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

type SubscriptionPayOptions = Pick<
  ApiRequestConfig,
  'skipBusinessError' | 'skipErrorHandler'
>

export async function paySubscriptionStripe(
  data: SubscriptionPayRequest,
  options?: SubscriptionPayOptions
): Promise<SubscriptionPayResponse> {
  const res = await api.post('/api/subscription/stripe/pay', data, options)
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

export async function getSubscriptionOffer(): Promise<
  ApiResponse<SubscriptionOffer | SubscriptionOfferEnvelope>
> {
  const res = await api.get('/api/subscription/stripe/offer', {
    skipErrorHandler: true,
    skipBusinessError: true,
  })
  return res.data
}

export async function getStripeSubscriptionSummary(): Promise<
  ApiResponse<StripeSubscriptionSummary>
> {
  const res = await api.get('/api/subscription/stripe/summary', {
    skipErrorHandler: true,
    skipBusinessError: true,
  })
  return res.data
}

export async function createCustomerPortalSession(): Promise<
  ApiResponse<{ url: string }>
> {
  const res = await api.post('/api/subscription/stripe/portal', undefined, {
    skipErrorHandler: true,
    skipBusinessError: true,
  })
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
