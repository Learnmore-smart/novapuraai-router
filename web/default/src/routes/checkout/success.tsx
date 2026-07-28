import { createFileRoute } from '@tanstack/react-router'

import { CheckoutSuccess } from '@/features/subscriptions/components/checkout-success'

export const Route = createFileRoute('/checkout/success')({
  component: CheckoutSuccess,
})
