import { createFileRoute } from '@tanstack/react-router'

import { CheckoutCancel } from '@/features/subscriptions/components/checkout-cancel'

export const Route = createFileRoute('/checkout/cancel')({
  component: CheckoutCancel,
})
