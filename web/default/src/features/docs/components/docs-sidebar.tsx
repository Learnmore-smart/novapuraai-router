import { Link, useRouterState } from '@tanstack/react-router'
import { ChevronRight } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'

import { DOC_NAV_GROUPS } from '../config/nav-tree'

interface DocsSidebarProps {
  onNavigate?: () => void
  className?: string
}

export function DocsSidebar(props: DocsSidebarProps) {
  const { t } = useTranslation()
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(
    new Set()
  )

  const activeItemId = useMemo(() => {
    const segments = pathname.split('/').filter(Boolean)
    // /docs/<section>
    return segments.length >= 2 ? segments[1] : ''
  }, [pathname])

  const toggleGroup = (groupId: string) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(groupId)) next.delete(groupId)
      else next.add(groupId)
      return next
    })
  }

  return (
    <nav
      aria-label={t('Documentation navigation')}
      className={cn('flex h-full flex-col', props.className)}
    >
      <ScrollArea className='np-docs-sidebar-scroll h-[calc(100svh-7rem)]'>
        <div className='space-y-5 pr-2'>
          {DOC_NAV_GROUPS.map((group) => {
            const collapsed = collapsedGroups.has(group.id)
            const hasActive = group.items.some(
              (item) => item.id === activeItemId
            )
            return (
              <div key={group.id}>
                <button
                  type='button'
                  onClick={() => toggleGroup(group.id)}
                  className={cn(
                    'text-muted-foreground hover:text-foreground mb-1.5 flex w-full items-center gap-1.5 px-2 text-left text-[0.7rem] font-semibold tracking-wider uppercase transition-colors',
                    hasActive && 'text-foreground'
                  )}
                  aria-expanded={!collapsed}
                >
                  <ChevronRight
                    className={cn(
                      'size-3 transition-transform',
                      !collapsed && 'rotate-90'
                    )}
                    aria-hidden='true'
                  />
                  {t(group.titleKey)}
                </button>
                {!collapsed && (
                  <ul className='space-y-0.5'>
                    {group.items.map((item) => {
                      const Icon = item.icon
                      const isActive = item.id === activeItemId
                      return (
                        <li key={item.id}>
                          <Link
                            to={item.href}
                            onClick={props.onNavigate}
                            className={cn(
                              'flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors',
                              isActive
                                ? 'bg-muted text-foreground font-medium'
                                : 'text-muted-foreground hover:text-foreground hover:bg-muted/60'
                            )}
                          >
                            {Icon ? (
                              <Icon
                                className={cn(
                                  'size-3.5 shrink-0',
                                  isActive && 'text-primary'
                                )}
                                aria-hidden='true'
                              />
                            ) : null}
                            <span className='truncate'>{t(item.titleKey)}</span>
                          </Link>
                        </li>
                      )
                    })}
                  </ul>
                )}
              </div>
            )
          })}
        </div>
      </ScrollArea>
    </nav>
  )
}
