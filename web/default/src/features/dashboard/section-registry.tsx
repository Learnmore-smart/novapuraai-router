import type { TFunction } from 'i18next'

import { createSectionRegistry } from '@/features/system-settings/utils/section-registry'

/**
 * Dashboard page section definitions
 */
const DASHBOARD_SECTIONS = [
  {
    id: 'overview',
    titleKey: 'Overview',
    build: () => null,
  },
  {
    id: 'models',
    titleKey: 'Model Call Analytics',
    build: () => null,
  },
  {
    id: 'flow',
    titleKey: 'Flow',
    build: () => null,
  },
  {
    id: 'users',
    titleKey: 'User Analytics',
    adminOnly: true,
    build: () => null,
  },
] as const

export type DashboardSectionId = (typeof DASHBOARD_SECTIONS)[number]['id']

const ADMIN_ONLY_SECTIONS = new Set<string>(['users'])

const dashboardRegistry = createSectionRegistry<
  DashboardSectionId,
  Record<string, never>,
  []
>({
  sections: DASHBOARD_SECTIONS,
  defaultSection: 'overview',
  basePath: '/dashboard',
  urlStyle: 'path',
})

export const DASHBOARD_SECTION_IDS = dashboardRegistry.sectionIds
export const DASHBOARD_DEFAULT_SECTION = dashboardRegistry.defaultSection

export function getDashboardSectionNavItems(
  t: TFunction,
  options?: { isAdmin?: boolean }
) {
  const all = dashboardRegistry.getSectionNavItems(t)
  if (options?.isAdmin) return all
  return all.filter(
    (_, idx) => !ADMIN_ONLY_SECTIONS.has(DASHBOARD_SECTIONS[idx].id)
  )
}
