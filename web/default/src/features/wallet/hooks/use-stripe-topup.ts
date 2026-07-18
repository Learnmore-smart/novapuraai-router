import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import type { BillingCurrency } from '@/lib/billing-currency'
import { useBillingCurrencyStore } from '@/stores/billing-currency-store'

import {
  createBillingTopupCheckout,
  getBillingTopupConfig,
  saveBillingCurrency,
} from '../api'

export function useStripeTopup() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const selectedCurrency = useBillingCurrencyStore(
    (state) => state.selectedCurrency
  )
  const setSelectedCurrency = useBillingCurrencyStore(
    (state) => state.setSelectedCurrency
  )
  const [selectedAmountMinor, setSelectedAmountMinor] = useState<number | null>(
    null
  )

  const configQuery = useQuery({
    queryKey: ['billing-topup-config'],
    queryFn: getBillingTopupConfig,
    staleTime: 30_000,
  })
  const config = configQuery.data?.data

  useEffect(() => {
    if (config?.selected_currency) {
      setSelectedCurrency(config.selected_currency)
    }
  }, [config?.selected_currency, setSelectedCurrency])

  const offers = useMemo(() => {
    if (!config || config.selected_currency !== selectedCurrency) return []
    return config.offers
  }, [config, selectedCurrency])

  useEffect(() => {
    const current = offers.find(
      (offer) => offer.payment_amount_minor === selectedAmountMinor
    )
    if (current?.available) return
    const recommended = offers.find(
      (offer) => offer.available && offer.recommended
    )
    const firstAvailable = offers.find((offer) => offer.available)
    setSelectedAmountMinor(
      recommended?.payment_amount_minor ??
        firstAvailable?.payment_amount_minor ??
        null
    )
  }, [offers, selectedAmountMinor])

  const currencyMutation = useMutation({
    mutationFn: saveBillingCurrency,
    onMutate: (currency) => setSelectedCurrency(currency),
    onSuccess: async (response) => {
      if (!response.success) {
        throw new Error(response.message || t('Unable to save currency'))
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['billing-topup-config'] }),
        queryClient.invalidateQueries({ queryKey: ['pricing'] }),
      ])
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Unable to save currency'))
      if (config?.selected_currency) {
        setSelectedCurrency(config.selected_currency)
      }
    },
  })

  const checkoutMutation = useMutation({
    mutationFn: createBillingTopupCheckout,
    onSuccess: (response) => {
      if (!response.success || !response.data?.checkout_url) {
        toast.error(response.message || t('Checkout failed'))
        return
      }
      window.location.assign(response.data.checkout_url)
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Checkout failed'))
    },
  })

  const checkout = (amountMinor = selectedAmountMinor) => {
    if (amountMinor == null || amountMinor <= 0) return
    checkoutMutation.mutate({
      currency: selectedCurrency,
      amount_minor: amountMinor,
    })
  }

  return {
    config,
    offers,
    selectedCurrency,
    selectedAmountMinor,
    setSelectedAmountMinor,
    changeCurrency: (currency: BillingCurrency) =>
      currencyMutation.mutate(currency),
    checkout,
    isLoading: configQuery.isPending,
    isCurrencyChanging: currencyMutation.isPending || configQuery.isFetching,
    isCheckingOut: checkoutMutation.isPending,
  }
}
