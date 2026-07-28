import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

interface TocItem {
  id: string
  text: string
  level: 2 | 3
}

interface DocsTocProps {
  containerRef: React.RefObject<HTMLDivElement | null>
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replaceAll(/[^\p{L}\p{N}\s-]/gu, '')
    .replaceAll(/\s+/g, '-')
}

function extractTocFromContainer(container: HTMLElement | null): TocItem[] {
  if (!container) return []
  const headings = container.querySelectorAll<HTMLElement>('h2, h3')
  const items: TocItem[] = []
  headings.forEach((heading) => {
    const text = heading.textContent?.trim() ?? ''
    if (!text) return
    let id = heading.id
    if (!id) {
      id = slugify(text)
      heading.id = id
    }
    items.push({
      id,
      text,
      level: heading.tagName === 'H2' ? 2 : 3,
    })
  })
  return items
}

export function DocsToc(props: DocsTocProps) {
  const { t } = useTranslation()
  const [toc, setToc] = useState<TocItem[]>([])
  const [activeId, setActiveId] = useState<string>('')

  // Re-extract TOC when content changes (after markdown renders).
  useEffect(() => {
    const container = props.containerRef.current
    if (!container) return

    const updateToc = () => setToc(extractTocFromContainer(container))

    updateToc()

    const observer = new MutationObserver(() => updateToc())
    observer.observe(container, { childList: true, subtree: true })
    return () => observer.disconnect()
  }, [props.containerRef])

  // Highlight active heading using IntersectionObserver.
  useEffect(() => {
    if (toc.length === 0) return
    const container = props.containerRef.current
    if (!container) return

    const headingEls = toc
      .map((item) =>
        container.querySelector<HTMLElement>(`#${CSS.escape(item.id)}`)
      )
      .filter((el): el is HTMLElement => el !== null)

    if (headingEls.length === 0) return

    const visibleSet = new Set<string>()
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          const id = entry.target.id
          if (entry.isIntersecting) visibleSet.add(id)
          else visibleSet.delete(id)
        })
        if (visibleSet.size > 0) {
          // Pick the topmost visible heading.
          const topMost = headingEls.find((el) => visibleSet.has(el.id))
          if (topMost) setActiveId(topMost.id)
        }
      },
      {
        rootMargin: '-80px 0px -70% 0px',
        threshold: 0,
      }
    )

    headingEls.forEach((el) => observer.observe(el))
    return () => observer.disconnect()
  }, [toc, props.containerRef])

  const handleJump = (id: string) => (event: React.MouseEvent) => {
    event.preventDefault()
    const container = props.containerRef.current
    const target = container?.querySelector<HTMLElement>(`#${CSS.escape(id)}`)
    if (target) {
      target.scrollIntoView({ behavior: 'smooth', block: 'start' })
      setActiveId(id)
    }
  }

  const hasToc = toc.length > 0
  const groups = useMemo(() => {
    // Group h3 under the preceding h2.
    const result: { h2: TocItem; h3s: TocItem[] }[] = []
    let current: { h2: TocItem; h3s: TocItem[] } | null = null
    toc.forEach((item) => {
      if (item.level === 2) {
        current = { h2: item, h3s: [] }
        result.push(current)
      } else if (current) {
        current.h3s.push(item)
      }
    })
    return result
  }, [toc])

  if (!hasToc) return null

  return (
    <aside className='hidden xl:block' aria-label={t('On this page')}>
      <div className='sticky top-[calc(var(--app-header-height)+2rem)]'>
        <p className='text-muted-foreground mb-2 text-[0.7rem] font-semibold tracking-wider uppercase'>
          {t('On this page')}
        </p>
        <ul className='border-border space-y-1 border-l'>
          {groups.map((group) => (
            <li key={group.h2.id}>
              <a
                href={`#${group.h2.id}`}
                onClick={handleJump(group.h2.id)}
                className={cn(
                  'hover:text-foreground -ml-px block border-l border-transparent px-3 py-1 text-sm transition-colors',
                  activeId === group.h2.id
                    ? 'text-foreground border-primary font-medium'
                    : 'text-muted-foreground'
                )}
              >
                {group.h2.text}
              </a>
              {group.h3s.length > 0 && (
                <ul className='space-y-0.5'>
                  {group.h3s.map((h3) => (
                    <li key={h3.id}>
                      <a
                        href={`#${h3.id}`}
                        onClick={handleJump(h3.id)}
                        className={cn(
                          'hover:text-foreground -ml-px block border-l border-transparent px-3 py-0.5 text-[0.8125rem] transition-colors',
                          activeId === h3.id
                            ? 'text-foreground border-primary font-medium'
                            : 'text-muted-foreground'
                        )}
                      >
                        {h3.text}
                      </a>
                    </li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ul>
      </div>
    </aside>
  )
}
