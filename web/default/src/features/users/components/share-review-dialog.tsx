/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import {
  Check,
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  Loader2,
  RefreshCw,
  X,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { formatQuota } from '@/lib/format'

import { getShareReviewQueue, reviewShareSubmission } from '../api'
import type { ShareReviewSubmission } from '../types'

const PAGE_SIZE = 20

interface ShareReviewDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ShareReviewDialog({
  open,
  onOpenChange,
}: ShareReviewDialogProps) {
  const { t } = useTranslation()
  const [items, setItems] = useState<ShareReviewSubmission[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [reviewing, setReviewing] = useState<string | null>(null)
  const [reasons, setReasons] = useState<Record<number, string>>({})

  const loadQueue = useCallback(async () => {
    setLoading(true)
    try {
      const response = await getShareReviewQueue(page, PAGE_SIZE)
      if (!response.success || !response.data) {
        toast.error(response.message || t('Failed to load share review queue'))
        return
      }
      setItems(response.data.items ?? [])
      setTotal(response.data.total ?? 0)
    } catch {
      toast.error(t('Failed to load share review queue'))
    } finally {
      setLoading(false)
    }
  }, [page, t])

  useEffect(() => {
    if (open) void loadQueue()
  }, [loadQueue, open])

  const totalPages = useMemo(
    () => Math.max(1, Math.ceil(total / PAGE_SIZE)),
    [total]
  )

  const handleReview = async (
    submission: ShareReviewSubmission,
    approve: boolean
  ) => {
    const reason = (reasons[submission.id] ?? '').trim()
    if (!reason) {
      toast.info(t('Enter a review reason before continuing'))
      return
    }

    const actionKey = `${submission.id}:${approve ? 'approve' : 'reject'}`
    setReviewing(actionKey)
    try {
      const response = await reviewShareSubmission(
        submission.id,
        approve,
        reason
      )
      if (!response.success) {
        toast.error(response.message || t('Failed to review submission'))
        return
      }
      toast.success(
        approve ? t('Share reward approved') : t('Share reward rejected')
      )
      setReasons((current) => {
        const next = { ...current }
        delete next[submission.id]
        return next
      })
      if (items.length === 1 && page > 1) {
        setPage((current) => current - 1)
      } else {
        await loadQueue()
      }
    } catch {
      toast.error(t('Failed to review submission'))
    } finally {
      setReviewing(null)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Share Review Queue')}
      description={t(
        'Review public share evidence. Each user can receive this promotional reward once.'
      )}
      contentClassName='sm:max-w-4xl'
      bodyClassName='space-y-3'
    >
      <div className='flex items-center justify-between gap-3'>
        <p className='text-muted-foreground text-xs'>
          {t('{{count}} pending submissions', { count: total })}
        </p>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => void loadQueue()}
          disabled={loading || reviewing !== null}
        >
          <RefreshCw className={loading ? 'size-4 animate-spin' : 'size-4'} />
          {t('Refresh')}
        </Button>
      </div>

      {loading && (
        <div className='space-y-3'>
          {['review-skeleton-1', 'review-skeleton-2', 'review-skeleton-3'].map(
            (skeletonId) => (
              <div
                key={skeletonId}
                className='space-y-3 rounded-lg border p-4'
              >
                <Skeleton className='h-5 w-2/3' />
                <Skeleton className='h-16 w-full' />
                <Skeleton className='h-9 w-48' />
              </div>
            )
          )}
        </div>
      )}

      {!loading && items.length === 0 && (
        <div className='text-muted-foreground flex min-h-44 items-center justify-center rounded-lg border border-dashed text-sm'>
          {t('No pending share submissions')}
        </div>
      )}

      {!loading && items.length > 0 && (
        <div className='space-y-3'>
          {items.map((submission) => {
            const approveKey = `${submission.id}:approve`
            const rejectKey = `${submission.id}:reject`
            const busy = reviewing?.startsWith(`${submission.id}:`) ?? false
            return (
              <article
                key={submission.id}
                className='bg-background space-y-3 rounded-lg border p-3 sm:p-4'
              >
                <div className='flex flex-wrap items-start justify-between gap-2'>
                  <div className='min-w-0 space-y-1'>
                    <div className='flex flex-wrap items-center gap-2'>
                      <StatusBadge
                        label={`#${submission.id}`}
                        variant='neutral'
                        size='sm'
                        copyable={false}
                      />
                      <span className='text-sm font-semibold'>
                        {t('User ID')}: {submission.user_id}
                      </span>
                      <span className='text-muted-foreground text-xs'>
                        {submission.platform || t('Platform not provided')}
                      </span>
                    </div>
                    <a
                      href={submission.url}
                      target='_blank'
                      rel='noopener noreferrer'
                      className='text-primary flex min-w-0 items-center gap-1 text-sm hover:underline'
                    >
                      <span className='break-all'>{submission.url}</span>
                      <ExternalLink className='size-3 shrink-0' />
                    </a>
                  </div>
                  <div className='text-right'>
                    <div className='text-sm font-semibold tabular-nums'>
                      {formatQuota(submission.amount)}
                    </div>
                    <div className='text-muted-foreground text-xs'>
                      {new Date(submission.created_at * 1000).toLocaleString()}
                    </div>
                  </div>
                </div>

                {submission.note ? (
                  <p className='bg-muted/40 break-words rounded-md p-2 text-xs'>
                    {submission.note}
                  </p>
                ) : null}

                <div className='space-y-1.5'>
                  <Label htmlFor={`share-review-reason-${submission.id}`}>
                    {t('Review reason')}
                  </Label>
                  <Textarea
                    id={`share-review-reason-${submission.id}`}
                    value={reasons[submission.id] ?? ''}
                    onChange={(event) =>
                      setReasons((current) => ({
                        ...current,
                        [submission.id]: event.target.value,
                      }))
                    }
                    placeholder={t('Record what was verified or why it failed')}
                    maxLength={512}
                    disabled={busy}
                    className='min-h-16'
                  />
                </div>

                <div className='flex flex-wrap justify-end gap-2'>
                  <Button
                    type='button'
                    variant='destructive'
                    size='sm'
                    onClick={() => void handleReview(submission, false)}
                    disabled={busy}
                  >
                    {reviewing === rejectKey ? (
                      <Loader2 className='size-4 animate-spin' />
                    ) : (
                      <X className='size-4' />
                    )}
                    {t('Reject')}
                  </Button>
                  <Button
                    type='button'
                    size='sm'
                    onClick={() => void handleReview(submission, true)}
                    disabled={busy}
                  >
                    {reviewing === approveKey ? (
                      <Loader2 className='size-4 animate-spin' />
                    ) : (
                      <Check className='size-4' />
                    )}
                    {t('Approve')}
                  </Button>
                </div>
              </article>
            )
          })}
        </div>
      )}

      {totalPages > 1 ? (
        <div className='flex items-center justify-between border-t pt-3'>
          <span className='text-muted-foreground text-xs'>
            {t('Page {{page}} of {{total}}', { page, total: totalPages })}
          </span>
          <div className='flex gap-2'>
            <Button
              type='button'
              variant='outline'
              size='icon-sm'
              onClick={() => setPage((current) => Math.max(1, current - 1))}
              disabled={page <= 1 || loading || reviewing !== null}
              aria-label={t('Previous page')}
            >
              <ChevronLeft className='size-4' />
            </Button>
            <Button
              type='button'
              variant='outline'
              size='icon-sm'
              onClick={() =>
                setPage((current) => Math.min(totalPages, current + 1))
              }
              disabled={page >= totalPages || loading || reviewing !== null}
              aria-label={t('Next page')}
            >
              <ChevronRight className='size-4' />
            </Button>
          </div>
        </div>
      ) : null}
    </Dialog>
  )
}
