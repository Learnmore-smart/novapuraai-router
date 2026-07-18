import { api } from '@/lib/api'
import type { BillingCurrency } from '@/lib/billing-currency'

export interface AdminCurrencyDefinition {
  enabled: boolean
  fx_presentment_per_usd: number
}

export interface AdminBillingCurrencyConfig {
  default_currency: BillingCurrency
  auto_update_fx: boolean
  fx_source?: string
  fx_updated_at?: number
  reference_fx_presentment_per_usd?: Record<BillingCurrency, number>
  currencies: Record<BillingCurrency, AdminCurrencyDefinition>
}

export interface AdminTopupCampaign {
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
  created_at: number
  updated_at: number
}

export interface AdminTopupTier {
  id: number
  campaign_id: number
  code: string
  name: string
  currency: BillingCurrency
  payment_amount_minor: number
  bonus_amount_minor: number
  total_credit_amount_minor: number
  recommended: boolean
  sort_order: number
  promo_expiry_days: number
  per_user_limit: number
  enabled: boolean
  start_at: number
  end_at: number
  created_at: number
  updated_at: number
}

export interface AdminTopupPreviewOffer {
  tier_id: number
  name: string
  currency: BillingCurrency
  payment_display: string
  bonus_display: string
  total_display: string
  available: boolean
  unavailable_reason?: string
  recommended: boolean
}

export interface AdminTopupPreview {
  currency: BillingCurrency
  campaign_active: boolean
  repeatable: boolean
  offers: AdminTopupPreviewOffer[]
}

interface AdminResponse<T> {
  success: boolean
  message?: string
  data: T
}

function requireData<T>(response: AdminResponse<T>): T {
  if (!response.success) {
    throw new Error(response.message || 'Request failed')
  }
  return response.data
}

export async function getAdminBillingCurrencies() {
  const res = await api.get<AdminResponse<AdminBillingCurrencyConfig>>(
    '/api/billing/admin/currencies'
  )
  return requireData(res.data)
}

export async function updateAdminBillingCurrencies(
  config: AdminBillingCurrencyConfig
) {
  const res = await api.put<AdminResponse<AdminBillingCurrencyConfig>>(
    '/api/billing/admin/currencies',
    config
  )
  return requireData(res.data)
}

export async function getAdminTopupCampaign() {
  const res = await api.get<AdminResponse<AdminTopupCampaign>>(
    '/api/billing/admin/top-up/campaign'
  )
  return requireData(res.data)
}

export async function updateAdminTopupCampaign(campaign: AdminTopupCampaign) {
  const res = await api.put<AdminResponse<AdminTopupCampaign>>(
    '/api/billing/admin/top-up/campaign',
    campaign
  )
  return requireData(res.data)
}

export async function getAdminTopupTiers() {
  const res = await api.get<AdminResponse<AdminTopupTier[]>>(
    '/api/billing/admin/top-up/promo-tiers'
  )
  return requireData(res.data)
}

export async function updateAdminTopupTier(tier: AdminTopupTier) {
  const res = await api.put<AdminResponse<AdminTopupTier>>(
    '/api/billing/admin/top-up/promo-tiers',
    tier
  )
  return requireData(res.data)
}

export async function getAdminTopupPreview(currency: BillingCurrency) {
  const res = await api.get<AdminResponse<AdminTopupPreview>>(
    '/api/billing/admin/top-up/preview',
    { params: { currency } }
  )
  return requireData(res.data)
}
