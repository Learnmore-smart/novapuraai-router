/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
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
 * - Docs and About are removed from product navigation.
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

  return links
}
