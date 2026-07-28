import { useMemo } from 'react'

import { useStatus } from '@/hooks/use-status'
import { parseHeaderNavModulesFromStatus } from '@/lib/nav-modules'
import { useAuthStore } from '@/stores/auth-store'

export type TopNavLink = {
  /**
   * i18n key (English source string). Consumers must translate via `t(title)`.
   * Storing the raw key instead of a pre-translated string avoids double
   * translation and keeps the link stable across language switches.
   */
  title: string
  href: string
  disabled?: boolean
  requiresAuth?: boolean
  external?: boolean
}

/**
 * Public top navigation from HeaderNavModules (/api/status).
 *
 * Auth policy:
 * - Logged-out users never see Dashboard/Console in the nav.
 * - Logged-in users may see Console when enabled (CTA in header shows Dashboard).
 * - Docs points to the in-app documentation at /docs when the module is enabled.
 * - About remains retired from product navigation.
 */
export function useTopNavLinks(): TopNavLink[] {
  const { status } = useStatus()
  const { auth } = useAuthStore()

  const modules = useMemo(() => {
    return parseHeaderNavModulesFromStatus(
      status as Record<string, unknown> | null
    )
  }, [status])

  const isAuthed = !!auth?.user
  const links: TopNavLink[] = []

  if (modules?.home !== false) {
    links.push({ title: 'Home', href: '/' })
  }

  // Console only when signed in — prevents dual Login + Dashboard for guests
  if (modules?.console !== false && isAuthed) {
    links.push({ title: 'Console', href: '/dashboard' })
  }

  const pricing = modules?.pricing
  if (pricing && typeof pricing === 'object' && pricing.enabled) {
    const requiresAuth = pricing.requireAuth && !isAuthed
    links.push({ title: 'Model Square', href: '/pricing', requiresAuth })
  }

  const rankings = modules?.rankings
  if (rankings && typeof rankings === 'object' && rankings.enabled) {
    const requiresAuth = rankings.requireAuth && !isAuthed
    links.push({ title: 'Rankings', href: '/rankings', requiresAuth })
  }

  // NovaPura subscription plans — public pricing page. Auth is enforced
  // inside the page (the PurchaseModal flow requires a session), but the
  // link itself is shown to everyone so logged-out users can browse plans.
  links.push({ title: 'Plans', href: '/plans' })

  if (modules?.docs !== false) {
    links.push({
      title: 'Docs',
      href: '/docs',
    })
  }

  return links
}
