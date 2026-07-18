import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type { BillingCurrency } from '@/lib/billing-currency'

interface BillingCurrencyState {
  selectedCurrency: BillingCurrency
  setSelectedCurrency: (currency: BillingCurrency) => void
}

export const useBillingCurrencyStore = create<BillingCurrencyState>()(
  persist(
    (set) => ({
      selectedCurrency: 'cny',
      setSelectedCurrency: (selectedCurrency) => set({ selectedCurrency }),
    }),
    {
      name: 'novapura-billing-currency',
    }
  )
)
