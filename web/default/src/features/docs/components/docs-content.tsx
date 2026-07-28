import { Link } from '@tanstack/react-router'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import { useMemo, type RefObject } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Markdown } from '@/components/ui/markdown'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import { findDocItem, findDocNeighbor } from '../config/nav-tree'
import { useDocContent } from '../hooks/use-doc-content'

interface DocsContentProps {
  sectionId: string
  contentRef: RefObject<HTMLDivElement | null>
  className?: string
}

export function DocsContent(props: DocsContentProps) {
  const { t } = useTranslation()
  const item = findDocItem(props.sectionId)
  const { content, loading, notFound } = useDocContent(props.sectionId)

  const neighbor = useMemo(
    () => findDocNeighbor(props.sectionId),
    [props.sectionId]
  )

  const title = item ? t(item.titleKey) : props.sectionId

  if (loading) {
    return (
      <div className={cn('space-y-4', props.className)}>
        <Skeleton className='h-9 w-2/3' />
        <Skeleton className='h-4 w-full' />
        <Skeleton className='h-4 w-5/6' />
        <Skeleton className='h-32 w-full' />
        <Skeleton className='h-4 w-full' />
        <Skeleton className='h-4 w-4/6' />
      </div>
    )
  }

  if (notFound || !item) {
    return (
      <div className={cn('py-16 text-center', props.className)}>
        <h1 className='text-2xl font-semibold'>{t('Doc not found')}</h1>
        <p className='text-muted-foreground mt-2 text-sm'>
          {t(
            'The documentation section you requested does not exist or has not been translated yet.'
          )}
        </p>
        <Button className='mt-6' render={<Link to='/docs' />}>
          {t('Back to Quickstart')}
        </Button>
      </div>
    )
  }

  return (
    <article className={cn('min-w-0', props.className)}>
      <header className='border-border border-b pb-4'>
        <p className='editorial-kicker'>{t('Documentation')}</p>
        <h1 className='mt-1 text-3xl font-semibold tracking-tight sm:text-4xl'>
          {title}
        </h1>
      </header>

      <div ref={props.contentRef} className='np-docs-content py-6'>
        <Markdown>{content}</Markdown>
      </div>

      <nav
        className='border-border mt-8 flex items-center justify-between gap-3 border-t pt-6'
        aria-label={t('Doc pagination')}
      >
        {neighbor.prev ? (
          <Button
            variant='outline'
            render={<Link to={neighbor.prev.href} />}
            className='max-w-[48%] justify-start text-left'
          >
            <ArrowLeft data-icon='inline-start' />
            <span className='flex min-w-0 flex-col'>
              <span className='text-muted-foreground text-[0.7rem] font-medium tracking-wider uppercase'>
                {t('Previous')}
              </span>
              <span className='truncate text-sm font-medium'>
                {t(neighbor.prev.titleKey)}
              </span>
            </span>
          </Button>
        ) : (
          <span />
        )}
        {neighbor.next ? (
          <Button
            variant='outline'
            render={<Link to={neighbor.next.href} />}
            className='max-w-[48%] justify-end text-right'
          >
            <span className='flex min-w-0 flex-col items-end'>
              <span className='text-muted-foreground text-[0.7rem] font-medium tracking-wider uppercase'>
                {t('Next')}
              </span>
              <span className='truncate text-sm font-medium'>
                {t(neighbor.next.titleKey)}
              </span>
            </span>
            <ArrowRight data-icon='inline-end' />
          </Button>
        ) : (
          <span />
        )}
      </nav>
    </article>
  )
}
