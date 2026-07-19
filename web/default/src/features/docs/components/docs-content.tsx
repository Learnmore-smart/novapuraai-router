import { Link } from '@tanstack/react-router'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Markdown } from '@/components/ui/markdown'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import {
  getAdjacentSections,
  getDocSectionMeta,
  type DocSectionId,
} from '../config/nav-tree'
import { extractTocFromMarkdown, type TocItem } from '../lib/toc'

type DocsContentProps = {
  section: DocSectionId
  markdown: string
  loading: boolean
  error: string | null
  usedFallback: boolean
  onTocChange?: (items: TocItem[]) => void
  className?: string
}

export function DocsContent(props: DocsContentProps) {
  const { t } = useTranslation()
  const meta = getDocSectionMeta(props.section)
  const adjacent = getAdjacentSections(props.section)

  const toc = useMemo(
    () => extractTocFromMarkdown(props.markdown),
    [props.markdown]
  )

  useEffect(() => {
    props.onTocChange?.(toc)
    // Intentionally depend on toc only; parent stores TOC for the floating sidebar.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onTocChange is a setState
  }, [toc])

  // Assign stable ids to rendered headings so TOC anchors work.
  useEffect(() => {
    if (props.loading || !props.markdown) return
    const root = document.querySelector('#docs-article')
    if (!root) return
    const headings = root.querySelectorAll('h2, h3')
    headings.forEach((heading, index) => {
      const item = toc[index]
      if (item) {
        heading.id = item.id
      }
    })
  }, [props.loading, props.markdown, toc])

  if (props.loading) {
    return (
      <div className={cn('min-w-0 space-y-4', props.className)}>
        <Skeleton className='h-8 w-2/3' />
        <Skeleton className='h-4 w-full' />
        <Skeleton className='h-4 w-5/6' />
        <Skeleton className='h-4 w-4/5' />
        <Skeleton className='mt-6 h-40 w-full' />
      </div>
    )
  }

  if (props.error || !props.markdown.trim()) {
    return (
      <div className={cn('min-w-0', props.className)}>
        <div className='border-border bg-card rounded-lg border border-dashed p-8 text-center'>
          <h1 className='text-xl font-semibold'>{t(meta.titleKey)}</h1>
          <p className='text-muted-foreground mt-2 text-sm'>
            {t('This documentation page is not available yet.')}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className={cn('min-w-0', props.className)}>
      <header className='mb-8 space-y-2 border-b border-border pb-6'>
        <p className='text-muted-foreground text-xs font-medium tracking-wide uppercase'>
          {t(meta.groupTitleKey)}
        </p>
        <h1 className='text-3xl font-semibold tracking-tight'>
          {t(meta.titleKey)}
        </h1>
        {props.usedFallback ? (
          <p className='text-muted-foreground bg-muted/50 rounded-md px-3 py-2 text-sm'>
            {t(
              'This page is shown in a fallback language because a full translation is still loading or missing.'
            )}
          </p>
        ) : null}
      </header>

      <article
        id='docs-article'
        className='prose-neutral dark:prose-invert max-w-none'
      >
        <Markdown className='prose-neutral dark:prose-invert max-w-none'>
          {props.markdown}
        </Markdown>
      </article>

      <nav
        aria-label={t('Page navigation')}
        className='border-border mt-12 grid gap-3 border-t pt-6 sm:grid-cols-2'
      >
        {adjacent.prev ? (
          <Link
            to='/docs/$section'
            params={{ section: adjacent.prev }}
            className='border-border hover:bg-muted/50 group flex flex-col gap-1 rounded-lg border p-4 transition-colors'
          >
            <span className='text-muted-foreground flex items-center gap-1 text-xs'>
              <ArrowLeft className='size-3.5' />
              {t('Previous')}
            </span>
            <span className='text-sm font-medium group-hover:underline'>
              {t(getDocSectionMeta(adjacent.prev).titleKey)}
            </span>
          </Link>
        ) : (
          <div />
        )}
        {adjacent.next ? (
          <Link
            to='/docs/$section'
            params={{ section: adjacent.next }}
            className='border-border hover:bg-muted/50 group flex flex-col items-end gap-1 rounded-lg border p-4 text-right transition-colors'
          >
            <span className='text-muted-foreground flex items-center gap-1 text-xs'>
              {t('Next')}
              <ArrowRight className='size-3.5' />
            </span>
            <span className='text-sm font-medium group-hover:underline'>
              {t(getDocSectionMeta(adjacent.next).titleKey)}
            </span>
          </Link>
        ) : null}
      </nav>
    </div>
  )
}
