import { api } from '@/lib/api'
import type { BillingCurrency } from '@/lib/billing-currency'

import type { PricingData } from './types'

// ----------------------------------------------------------------------------
// Pricing APIs
// ----------------------------------------------------------------------------

// Get model pricing data
export async function getPricing(
  currency: BillingCurrency
): Promise<PricingData> {
  const res = await api.get('/api/pricing', { params: { currency } })
  return res.data
}
