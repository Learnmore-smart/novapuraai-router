import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/console/log')({
  beforeLoad: () => {
    throw redirect({ to: '/usage-logs' })
  },
})
