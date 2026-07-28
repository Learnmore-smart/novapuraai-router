import { ExternalLink, Gift, Loader2 } from 'lucide-react'
import { type FormEvent, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StatusBadge, type StatusBadgeProps } from '@/components/status-badge'
import { Turnstile } from '@/components/turnstile'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { useTurnstile } from '@/features/auth/hooks/use-turnstile'
import { formatQuota } from '@/lib/format'

import { getMyShareSubmissions, isApiSuccess, submitShareReward } from '../api'
import type { ShareSubmission, ShareSubmissionStatus } from '../types'

const statusVariants: Record<
  ShareSubmissionStatus,
  StatusBadgeProps['variant']
> = {
  pending: 'warning',
  approved: 'success',
  rejected: 'danger',
}

const statusLabels: Record<ShareSubmissionStatus, string> = {
  pending: 'Pending review',
  approved: 'Approved',
  rejected: 'Rejected',
}

export function ShareRewardCard() {
  const { t } = useTranslation()
  const [submissions, setSubmissions] = useState<ShareSubmission[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [url, setUrl] = useState('')
  const [platform, setPlatform] = useState('')
  const [note, setNote] = useState('')
  const {
    isTurnstileEnabled,
    turnstileSiteKey,
    turnstileToken,
    turnstileResetKey,
    setTurnstileToken,
    resetTurnstile,
    validateTurnstile,
  } = useTurnstile()

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const response = await getMyShareSubmissions()
        if (!cancelled && isApiSuccess(response)) {
          setSubmissions(response.data ?? [])
        }
      } catch {
        if (!cancelled) {
          toast.error(t('Failed to load share reward status'))
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [t])

  const latest = submissions[0]
  const canSubmit = !latest || latest.status === 'rejected'

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!url.trim() || !platform.trim() || !validateTurnstile()) return

    setSubmitting(true)
    try {
      const response = await submitShareReward(
        { url: url.trim(), platform: platform.trim(), note: note.trim() },
        turnstileToken
      )
      const submission = response.data
      if (!isApiSuccess(response) || !submission) {
        toast.error(response.message || t('Failed to submit share reward'))
        return
      }
      setSubmissions((current) => [submission, ...current])
      setUrl('')
      setPlatform('')
      setNote('')
      toast.success(t('Share submitted for review'))
    } catch {
      toast.error(t('Failed to submit share reward'))
    } finally {
      setSubmitting(false)
      if (isTurnstileEnabled) resetTurnstile()
    }
  }

  if (loading) {
    return (
      <Card data-card-hover='false' className='bg-muted/20 py-0'>
        <CardContent className='space-y-3 p-3 sm:p-4'>
          <Skeleton className='h-5 w-40' />
          <Skeleton className='h-10 w-full' />
          <Skeleton className='h-20 w-full' />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card data-card-hover='false' className='bg-muted/20 py-0'>
      <CardContent className='space-y-4 p-3 sm:p-4'>
        <div className='flex items-start gap-2.5'>
          <IconBadge tone='chart-4'>
            <Gift />
          </IconBadge>
          <div className='min-w-0 flex-1'>
            <div className='flex flex-wrap items-center gap-2'>
              <h3 className='text-sm font-semibold'>{t('Share Reward')}</h3>
              {latest ? (
                <StatusBadge
                  label={t(statusLabels[latest.status])}
                  variant={statusVariants[latest.status]}
                  size='sm'
                  copyable={false}
                />
              ) : null}
            </div>
            <p className='text-muted-foreground mt-0.5 text-xs'>
              {t(
                'Submit a public post for review. Approved submissions receive promotional quota once.'
              )}
            </p>
          </div>
        </div>

        {latest ? (
          <div className='bg-background/70 space-y-2 rounded-lg border p-3 text-xs'>
            <div className='flex min-w-0 items-center justify-between gap-3'>
              <a
                href={latest.url}
                target='_blank'
                rel='noopener noreferrer'
                className='text-primary flex min-w-0 items-center gap-1 font-medium hover:underline'
              >
                <span className='truncate'>{latest.url}</span>
                <ExternalLink className='size-3 shrink-0' />
              </a>
              <span className='shrink-0 font-medium tabular-nums'>
                {formatQuota(latest.amount)}
              </span>
            </div>
            <div className='text-muted-foreground flex flex-wrap gap-x-3 gap-y-1'>
              <span>
                {t('Platform')}: {latest.platform || t('Not provided')}
              </span>
              <span>
                {t('Submitted')}:{' '}
                {new Date(latest.created_at * 1000).toLocaleString()}
              </span>
            </div>
            {latest.review_reason ? (
              <p className='text-muted-foreground break-words'>
                {t('Review note')}: {latest.review_reason}
              </p>
            ) : null}
          </div>
        ) : null}

        {canSubmit ? (
          <form className='grid gap-3 lg:grid-cols-2' onSubmit={handleSubmit}>
            <div className='space-y-1.5 lg:col-span-2'>
              <Label htmlFor='share-reward-url'>{t('Public post URL')}</Label>
              <Input
                id='share-reward-url'
                type='url'
                value={url}
                onChange={(event) => setUrl(event.target.value)}
                placeholder='https://'
                maxLength={1024}
                disabled={submitting}
                required
              />
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='share-reward-platform'>{t('Platform')}</Label>
              <Input
                id='share-reward-platform'
                value={platform}
                onChange={(event) => setPlatform(event.target.value)}
                placeholder={t('Example: X, Reddit, WeChat')}
                maxLength={64}
                disabled={submitting}
                required
              />
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='share-reward-note'>{t('Note')}</Label>
              <Textarea
                id='share-reward-note'
                value={note}
                onChange={(event) => setNote(event.target.value)}
                placeholder={t('Optional context for the reviewer')}
                maxLength={512}
                disabled={submitting}
                className='min-h-9'
              />
            </div>
            {isTurnstileEnabled ? (
              <Turnstile
                siteKey={turnstileSiteKey}
                action='share_reward'
                onVerify={setTurnstileToken}
                onExpire={resetTurnstile}
                resetKey={turnstileResetKey}
                className='lg:col-span-2'
              />
            ) : null}
            <div className='flex justify-end lg:col-span-2'>
              <Button type='submit' size='sm' disabled={submitting}>
                {submitting ? (
                  <Loader2 className='size-4 animate-spin' />
                ) : null}
                {submitting ? t('Submitting...') : t('Submit for review')}
              </Button>
            </div>
          </form>
        ) : null}
      </CardContent>
    </Card>
  )
}
