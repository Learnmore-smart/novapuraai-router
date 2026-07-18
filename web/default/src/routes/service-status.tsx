import { createFileRoute } from '@tanstack/react-router'

import { ServiceStatusPage } from '@/features/legal'

export const Route = createFileRoute('/service-status')({
  component: ServiceStatusPage,
})
