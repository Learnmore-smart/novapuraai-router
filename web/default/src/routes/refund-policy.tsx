import { createFileRoute } from '@tanstack/react-router'

import { RefundPolicy } from '@/features/legal'

export const Route = createFileRoute('/refund-policy')({
  component: RefundPolicy,
})
