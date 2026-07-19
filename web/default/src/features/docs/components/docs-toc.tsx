import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import type { TocItem } from '../lib/toc'

export type { TocItem }

type DocsTocProps = {
  items: TocItem[]
  className?: string
}

export function DocsToc(props: DocsTocProps) {
  const { t } = useTranslation()
  const [activeId, setActiveId] = useState<string>('')

  const headingIds = useMemo(
    () => props.items.map((item) => item.id),
    [props.items]
  )

  useEffect(() => {
    if (headingIds.length === 0) return

    const elements = headingIds
      .map((id) => document.querySelector<HTMLElement>(`#${CSS.escape(id)}`))
      .filter((el): el is HTMLElement => Boolean(el))

    if (elements.length === 0) return

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort(
            (a, b) =>
              (a.target as HTMLElement).offsetTop -
              (b.target as HTMLElement).offsetTop
          )
        if (visible[0]?.target?.id) {
          setActiveId(visible[0].target.id)
        }
      },
      {
        rootMargin: '-20% 0px -70% 0px',
        threshold: [0, 1],
      }
    )

    for (const el of elements) observer.observe(el)
    return () => observer.disconnect()
  }, [headingIds])

  if (props.items.length === 0) return null

  return (
    <nav
      aria-label={t('On this page')}
      className={cn(
        'sticky top-[calc(var(--app-header-height)+1.5rem)] hidden max-h-[calc(100vh-var(--app-header-height)-3rem)] overflow-y-auto xl:block',
        props.className
      )}
    >
      <p className='text-muted-foreground mb-3 text-xs font-semibold tracking-wide uppercase'>
        {t('On this page')}
      </p>
      <ul className='border-border space-y-1.5 border-l pl-3'>
        {props.items.map((item) => (
          <li key={item.id}>
            <a
              href={`#${item.id}`}
              className={cn(
                'block text-sm transition-colors',
                item.level === 3 && 'pl-3 text-[13px]',
                activeId === item.id
                  ? 'text-foreground font-medium'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {item.text}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  )
}
