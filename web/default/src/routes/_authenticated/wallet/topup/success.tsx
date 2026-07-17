import { createFileRoute } from '@tanstack/react-router'

import { StripeTopupSuccess } from '@/features/wallet/components/stripe-topup-success'

export const Route = createFileRoute('/_authenticated/wallet/topup/success')({
  component: StripeTopupSuccess,
  validateSearch: (search: Record<string, unknown>) => ({
    order_id: typeof search.order_id === 'string' ? search.order_id : '',
  }),
})
