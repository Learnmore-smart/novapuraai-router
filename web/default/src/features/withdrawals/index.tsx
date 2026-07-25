import {
  Check,
  ChevronLeft,
  ChevronRight,
  Loader2,
  RefreshCw,
  X,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { cn } from '@/lib/utils'

import { getWithdrawalQueue, processWithdrawal } from './api'
import { WITHDRAWAL_PAGE_SIZE } from './constants'
import type { ProcessWithdrawalPayload, WithdrawalRequest } from './types'

// ============================================================================
// Helpers
// ============================================================================

/** Convert USD cents (int64) to a display string via the user's display currency. */
function formatCents(cents: number): string {
  return formatCurrencyFromUSD(cents / 100)
}

function formatTime(unixSeconds: number): string {
  if (!unixSeconds) return '-'
  return new Date(unixSeconds * 1000).toLocaleString()
}

// ============================================================================
// Status Filter
// ============================================================================

type StatusFilter = '' | 'pending' | 'paid' | 'rejected'

const STATUS_FILTER_OPTIONS: { value: StatusFilter; labelKey: string }[] = [
  { value: '', labelKey: 'All' },
  { value: 'pending', labelKey: 'Pending' },
  { value: 'paid', labelKey: 'Paid' },
  { value: 'rejected', labelKey: 'Rejected' },
]

// ============================================================================
// Process Withdrawal Dialog
// ============================================================================

interface ProcessDialogState {
  withdrawal: WithdrawalRequest
  action: 'paid' | 'rejected'
}

interface ProcessWithdrawalDialogProps {
  state: ProcessDialogState | null
  onOpenChange: (state: ProcessDialogState | null) => void
  onSuccess: () => void
}

function ProcessWithdrawalDialog({
  state,
  onOpenChange,
  onSuccess,
}: ProcessWithdrawalDialogProps) {
  const { t } = useTranslation()
  const [payoutChannel, setPayoutChannel] = useState('manual')
  const [payoutTxId, setPayoutTxId] = useState('')
  const [adminRemark, setAdminRemark] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (state) {
      setPayoutChannel('manual')
      setPayoutTxId('')
      setAdminRemark('')
    }
  }, [state])

  if (!state) return null

  const isPaid = state.action === 'paid'

  const handleSubmit = async () => {
    if (!state) return
    setSubmitting(true)
    try {
      const payload: ProcessWithdrawalPayload = {
        action: state.action,
      }
      if (isPaid) {
        payload.payout_channel = payoutChannel || 'manual'
        payload.payout_tx_id = payoutTxId
      }
      payload.admin_remark = adminRemark

      const result = await processWithdrawal(state.withdrawal.id, payload)
      if (!result.success) {
        toast.error(result.message || t('Failed to process withdrawal'))
        return
      }
      toast.success(
        isPaid ? t('Withdrawal marked as paid') : t('Withdrawal rejected')
      )
      onOpenChange(null)
      onSuccess()
    } catch {
      toast.error(t('Failed to process withdrawal'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={state !== null}
      onOpenChange={(open) => !open && onOpenChange(null)}
      title={
        isPaid
          ? t('Mark Withdrawal as Paid')
          : t('Reject Withdrawal')
      }
      description={
        isPaid
          ? t(
              'Confirm that the manual payout has been completed. The funds were already debited from the user balance at request time.'
            )
          : t(
              'Rejecting this request refunds the full amount back to the user withdrawable balance.'
            )
      }
      contentClassName='sm:max-w-lg'
      bodyClassName='space-y-4'
      footer={
        <div className='flex justify-end gap-2'>
          <Button
            variant='outline'
            onClick={() => onOpenChange(null)}
            disabled={submitting}
          >
            {t('Cancel')}
          </Button>
          <Button
            variant={isPaid ? 'default' : 'destructive'}
            onClick={() => void handleSubmit()}
            disabled={submitting}
          >
            {submitting ? (
              <Loader2 className='size-4 animate-spin' />
            ) : isPaid ? (
              <Check className='size-4' />
            ) : (
              <X className='size-4' />
            )}
            {isPaid ? t('Confirm Paid') : t('Reject')}
          </Button>
        </div>
      }
    >
      <div className='space-y-1'>
        <div className='flex items-center justify-between text-sm'>
          <span className='text-muted-foreground'>{t('Request ID')}</span>
          <span className='font-medium'>#{state.withdrawal.id}</span>
        </div>
        <div className='flex items-center justify-between text-sm'>
          <span className='text-muted-foreground'>{t('User ID')}</span>
          <span className='font-medium'>{state.withdrawal.user_id}</span>
        </div>
        <div className='flex items-center justify-between text-sm'>
          <span className='text-muted-foreground'>{t('Amount')}</span>
          <span className='font-semibold tabular-nums'>
            {formatCents(state.withdrawal.amount_cents)}
          </span>
        </div>
      </div>

      {isPaid && (
        <>
          <div className='space-y-1.5'>
            <Label htmlFor='payout-channel'>
              {t('Payout Channel')}
            </Label>
            <Input
              id='payout-channel'
              value={payoutChannel}
              onChange={(e) => setPayoutChannel(e.target.value)}
              placeholder='manual'
              disabled={submitting}
            />
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='payout-tx-id'>
              {t('Payout Transaction ID')}
            </Label>
            <Input
              id='payout-tx-id'
              value={payoutTxId}
              onChange={(e) => setPayoutTxId(e.target.value)}
              placeholder={t('External transaction reference (optional)')}
              disabled={submitting}
            />
          </div>
        </>
      )}

      <div className='space-y-1.5'>
        <Label htmlFor='admin-remark'>
          {isPaid ? t('Admin Remark') : t('Rejection Reason')}
        </Label>
        <Textarea
          id='admin-remark'
          value={adminRemark}
          onChange={(e) => setAdminRemark(e.target.value)}
          placeholder={t('Record notes for this decision')}
          rows={3}
          maxLength={512}
          disabled={submitting}
        />
      </div>
    </Dialog>
  )
}

// ============================================================================
// Withdrawal Card
// ============================================================================

interface WithdrawalCardProps {
  withdrawal: WithdrawalRequest
  onProcess: (withdrawal: WithdrawalRequest, action: 'paid' | 'rejected') => void
  processing: boolean
}

function WithdrawalCard({
  withdrawal,
  onProcess,
  processing,
}: WithdrawalCardProps) {
  const { t } = useTranslation()
  const isPending = withdrawal.status === 'pending'
  const statusConfig = {
    pending: { label: t('Pending'), variant: 'warning' as const },
    paid: { label: t('Paid'), variant: 'success' as const },
    rejected: { label: t('Rejected'), variant: 'danger' as const },
  }
  const status = statusConfig[withdrawal.status]

  return (
    <article className='bg-background space-y-3 rounded-lg border p-3 sm:p-4'>
      <div className='flex flex-wrap items-start justify-between gap-2'>
        <div className='min-w-0 space-y-1'>
          <div className='flex flex-wrap items-center gap-2'>
            <StatusBadge
              label={`#${withdrawal.id}`}
              variant='neutral'
              size='sm'
              copyable={false}
            />
            <span className='text-sm font-semibold'>
              {t('User ID')}: {withdrawal.user_id}
            </span>
            <StatusBadge
              label={status.label}
              variant={status.variant}
              size='sm'
              copyable={false}
            />
          </div>
        </div>
        <div className='text-right'>
          <div className='text-sm font-semibold tabular-nums'>
            {formatCents(withdrawal.amount_cents)}
          </div>
          <div className='text-muted-foreground text-xs'>
            {formatTime(withdrawal.requested_at)}
          </div>
        </div>
      </div>

      {withdrawal.status !== 'pending' && (
        <div className='space-y-1 text-xs'>
          {withdrawal.payout_channel && (
            <div className='flex gap-2'>
              <span className='text-muted-foreground'>
                {t('Payout Channel')}:
              </span>
              <span>{withdrawal.payout_channel}</span>
            </div>
          )}
          {withdrawal.payout_tx_id && (
            <div className='flex gap-2'>
              <span className='text-muted-foreground'>
                {t('Payout Transaction ID')}:
              </span>
              <span className='break-all'>{withdrawal.payout_tx_id}</span>
            </div>
          )}
          {withdrawal.admin_remark && (
            <div className='flex gap-2'>
              <span className='text-muted-foreground'>
                {t('Admin Remark')}:
              </span>
              <span className='break-words'>{withdrawal.admin_remark}</span>
            </div>
          )}
          <div className='flex gap-2'>
            <span className='text-muted-foreground'>
              {t('Reviewed At')}:
            </span>
            <span>{formatTime(withdrawal.reviewed_at)}</span>
          </div>
        </div>
      )}

      {isPending && (
        <div className='flex flex-wrap justify-end gap-2'>
          <Button
            type='button'
            variant='destructive'
            size='sm'
            onClick={() => onProcess(withdrawal, 'rejected')}
            disabled={processing}
          >
            <X className='size-4' />
            {t('Reject')}
          </Button>
          <Button
            type='button'
            size='sm'
            onClick={() => onProcess(withdrawal, 'paid')}
            disabled={processing}
          >
            <Check className='size-4' />
            {t('Mark Paid')}
          </Button>
        </div>
      )}
    </article>
  )
}

// ============================================================================
// Main Page
// ============================================================================

export function Withdrawals() {
  const { t } = useTranslation()
  const [items, setItems] = useState<WithdrawalRequest[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('')
  const [loading, setLoading] = useState(false)
  const [processState, setProcessState] = useState<ProcessDialogState | null>(
    null
  )

  const loadQueue = useCallback(async () => {
    setLoading(true)
    try {
      const response = await getWithdrawalQueue(
        page,
        WITHDRAWAL_PAGE_SIZE,
        statusFilter
      )
      if (!response.success || !response.data) {
        toast.error(response.message || t('Failed to load withdrawals'))
        return
      }
      setItems(response.data.items ?? [])
      setTotal(response.data.total ?? 0)
    } catch {
      toast.error(t('Failed to load withdrawals'))
    } finally {
      setLoading(false)
    }
  }, [page, statusFilter, t])

  useEffect(() => {
    void loadQueue()
  }, [loadQueue])

  const totalPages = useMemo(
    () => Math.max(1, Math.ceil(total / WITHDRAWAL_PAGE_SIZE)),
    [total]
  )

  const handleProcess = (
    withdrawal: WithdrawalRequest,
    action: 'paid' | 'rejected'
  ) => {
    setProcessState({ withdrawal, action })
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Withdrawal Review')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => void loadQueue()}
          disabled={loading}
        >
          <RefreshCw className={loading ? 'size-4 animate-spin' : 'size-4'} />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          {/* Status filter tabs */}
          <div className='flex flex-wrap items-center gap-2'>
            {STATUS_FILTER_OPTIONS.map((option) => (
              <Button
                key={option.value}
                type='button'
                variant={statusFilter === option.value ? 'default' : 'outline'}
                size='sm'
                onClick={() => {
                  setStatusFilter(option.value)
                  setPage(1)
                }}
              >
                {t(option.labelKey)}
              </Button>
            ))}
            <span className='text-muted-foreground ml-auto text-xs'>
              {t('{{count}} requests', { count: total })}
            </span>
          </div>

          {/* Loading skeleton */}
          {loading && (
            <div className='space-y-3'>
              {[
                'withdrawal-skeleton-1',
                'withdrawal-skeleton-2',
                'withdrawal-skeleton-3',
              ].map((skeletonId) => (
                <div
                  key={skeletonId}
                  className='space-y-3 rounded-lg border p-4'
                >
                  <Skeleton className='h-5 w-2/3' />
                  <Skeleton className='h-9 w-48' />
                </div>
              ))}
            </div>
          )}

          {/* Empty state */}
          {!loading && items.length === 0 && (
            <div
              className={cn(
                'text-muted-foreground flex min-h-44 items-center justify-center rounded-lg border border-dashed text-sm'
              )}
            >
              {t('No withdrawal requests found')}
            </div>
          )}

          {/* Withdrawal cards */}
          {!loading && items.length > 0 && (
            <div className='space-y-3'>
              {items.map((withdrawal) => (
                <WithdrawalCard
                  key={withdrawal.id}
                  withdrawal={withdrawal}
                  onProcess={handleProcess}
                  processing={processState !== null}
                />
              ))}
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className='flex items-center justify-between border-t pt-3'>
              <span className='text-muted-foreground text-xs'>
                {t('Page {{page}} of {{total}}', {
                  page,
                  total: totalPages,
                })}
              </span>
              <div className='flex gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  size='icon-sm'
                  onClick={() => setPage((c) => Math.max(1, c - 1))}
                  disabled={page <= 1 || loading}
                  aria-label={t('Previous page')}
                >
                  <ChevronLeft className='size-4' />
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  size='icon-sm'
                  onClick={() =>
                    setPage((c) => Math.min(totalPages, c + 1))
                  }
                  disabled={page >= totalPages || loading}
                  aria-label={t('Next page')}
                >
                  <ChevronRight className='size-4' />
                </Button>
              </div>
            </div>
          )}
        </div>
      </SectionPageLayout.Content>

      <ProcessWithdrawalDialog
        state={processState}
        onOpenChange={setProcessState}
        onSuccess={() => void loadQueue()}
      />
    </SectionPageLayout>
  )
}
