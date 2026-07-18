import { createFileRoute, redirect } from '@tanstack/react-router'
import z from 'zod'

import { Rankings } from '@/features/rankings'
import { getFreshModuleAccess } from '@/lib/nav-modules'
import { useAuthStore } from '@/stores/auth-store'

const rankingsSearchSchema = z.object({
  period: z
    .enum(['today', 'week', 'month', 'year'])
    .optional()
    .catch(undefined),
})

export const Route = createFileRoute('/rankings/')({
  validateSearch: rankingsSearchSchema,
  beforeLoad: async ({ location }) => {
    const access = await getFreshModuleAccess('rankings')
    if (!access.enabled) {
      throw redirect({ to: '/' })
    }
    if (access.requireAuth) {
      const { auth } = useAuthStore.getState()
      if (!auth.user) {
        throw redirect({
          to: '/sign-in',
          search: { redirect: location.href },
        })
      }
    }
  },
  component: Rankings,
})
