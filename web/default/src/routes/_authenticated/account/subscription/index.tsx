import { createFileRoute } from '@tanstack/react-router'

import { AccountSubscription } from '@/features/subscriptions/components/account-subscription'

export const Route = createFileRoute('/_authenticated/account/subscription/')({
  component: AccountSubscription,
})
