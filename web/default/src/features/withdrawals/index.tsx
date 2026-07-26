import {
  Check,
  ChevronLeft,
  ChevronRight,
  Loader2,
  RefreshCw,
  RotateCcw,
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
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatCurrencyFromUSD } from '@/lib/currency'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import { getWithdrawalQueue, processWithdrawal, reverseWithdrawal } from './api'
import {
  PAYOUT_CHANNEL_MANUAL,
  PAYOUT_CHANNEL_STRIPE_CONNECT,
  WITHDRAWAL_AUTO_REFRESH_MS,
  WITHDRAWAL_PAGE_SIZE,
  WITHDRAWAL_STATUSES,
  hasNonTerminalItems,
} from './constants'
import type {
  ProcessWithdrawalPayload,
  ReverseWithdrawalPayload,
  WithdrawalRequest,
  WithdrawalStatus,
} from './types'

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

function formatRelativeTime(unixSeconds: number): string {
  if (!unixSeconds) return '-'
  return dayjs(unixSeconds * 1000).fromNow()
}

/** Truncate free-text error to a preview length; full text is shown in the tooltip. */
function truncate(text: string, max = 60): string {
  if (text.length <= max) return text
  return `${text.slice(0, max - 1)}…`
}

// ============================================================================
// Status Filter
// ============================================================================

type StatusFilter = '' | WithdrawalStatus

const STATUS_FILTER_OPTIONS: { value: StatusFilter; labelKey: string }[] = [
  { value: '', labelKey: 'All' },
  { value: 'pending', labelKey: 'Pending' },
  { value: 'transfer_creating', labelKey: 'Transfer Creating' },
  { value: 'awaiting_funds', labelKey: 'Awaiting Funds' },
  { value: 'payout_creating', labelKey: 'Payout Creating' },
  { value: 'processing', labelKey: 'Processing' },
  { value: 'action_required', labelKey: 'Action Required' },
  { value: 'failed', labelKey: 'Failed' },
  { value: 'paid', labelKey: 'Paid' },
  { value: 'rejected', labelKey: 'Rejected' },
]

// ============================================================================
// Process Withdrawal Dialog (Approve / Reject)
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
  const [payoutChannel, setPayoutChannel] = useState(PAYOUT_CHANNEL_MANUAL)
  const [payoutTxId, setPayoutTxId] = useState('')
  const [adminRemark, setAdminRemark] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (state) {
      setPayoutChannel(PAYOUT_CHANNEL_MANUAL)
      setPayoutTxId('')
      setAdminRemark('')
    }
  }, [state])

  if (!state) return null

  const isPaid = state.action === 'paid'
  // stripe_connect only applies to the approve path. Reject always refunds the
  // balance immediately and never touches Stripe.
  const showPayoutChannel = isPaid
  // Resolve the submit button icon outside of JSX to avoid nested ternaries.
  let SubmitIcon: typeof Loader2 = X
  if (submitting) {
    SubmitIcon = Loader2
  } else if (isPaid) {
    SubmitIcon = Check
  }

  const handleSubmit = async () => {
    if (!state) return
    setSubmitting(true)
    try {
      const payload: ProcessWithdrawalPayload = {
        action: state.action,
      }
      if (isPaid) {
        payload.payout_channel = payoutChannel || PAYOUT_CHANNEL_MANUAL
        // payout_tx_id is only meaningful for the manual channel — Stripe
        // Connect creates its own Transfer/Payout IDs server-side.
        if (payoutChannel === PAYOUT_CHANNEL_MANUAL) {
          payload.payout_tx_id = payoutTxId
        }
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
            <SubmitIcon
              className={cn('size-4', submitting && 'animate-spin')}
            />
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

      {showPayoutChannel && (
        <div className='space-y-2'>
          <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
            {t('Payout Channel')}
          </Label>
          <RadioGroup
            value={payoutChannel}
            onValueChange={(value) => setPayoutChannel(value as string)}
            className='grid gap-2 sm:grid-cols-2'
          >
            <label
              htmlFor='payout-manual'
              className={cn(
                'flex cursor-pointer items-center gap-2 rounded-lg border p-2.5 transition-colors hover:bg-muted',
                payoutChannel === PAYOUT_CHANNEL_MANUAL && 'border-primary'
              )}
            >
              <RadioGroupItem
                id='payout-manual'
                value={PAYOUT_CHANNEL_MANUAL}
              />
              <span className='text-sm font-medium'>{t('Manual')}</span>
            </label>
            <label
              htmlFor='payout-stripe'
              className={cn(
                'flex cursor-pointer items-center gap-2 rounded-lg border p-2.5 transition-colors hover:bg-muted',
                payoutChannel === PAYOUT_CHANNEL_STRIPE_CONNECT &&
                  'border-primary'
              )}
            >
              <RadioGroupItem
                id='payout-stripe'
                value={PAYOUT_CHANNEL_STRIPE_CONNECT}
              />
              <span className='text-sm font-medium'>
                {t('Stripe Connect')}
              </span>
            </label>
          </RadioGroup>
          <p className='text-muted-foreground text-xs'>
            {payoutChannel === PAYOUT_CHANNEL_STRIPE_CONNECT
              ? t(
                  'Creates a Stripe Transfer to the user\'s connected account; the user receives the payout via Stripe.'
                )
              : t(
                  'Mark this withdrawal as paid after completing the manual payout outside the system.'
                )}
          </p>
        </div>
      )}

      {isPaid && payoutChannel === PAYOUT_CHANNEL_MANUAL && (
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
// Reverse Transfer Dialog
// ============================================================================

interface ReverseDialogState {
  withdrawal: WithdrawalRequest
}

interface ReverseTransferDialogProps {
  state: ReverseDialogState | null
  onOpenChange: (state: ReverseDialogState | null) => void
  onSuccess: () => void
}

function ReverseTransferDialog({
  state,
  onOpenChange,
  onSuccess,
}: ReverseTransferDialogProps) {
  const { t } = useTranslation()
  const [reason, setReason] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (state) {
      setReason('')
    }
  }, [state])

  if (!state) return null
  const { withdrawal } = state

  const handleSubmit = async () => {
    if (!state) return
    setSubmitting(true)
    try {
      const payload: ReverseWithdrawalPayload = { reason }
      const result = await reverseWithdrawal(withdrawal.id, payload)
      if (!result.success) {
        toast.error(result.message || t('Failed to reverse transfer'))
        return
      }
      toast.success(t('Transfer reversed'))
      onOpenChange(null)
      onSuccess()
    } catch {
      toast.error(t('Failed to reverse transfer'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={state !== null}
      onOpenChange={(open) => !open && onOpenChange(null)}
      title={t('Reverse Transfer')}
      description={t(
        'Are you sure you want to reverse the Stripe Transfer for this withdrawal? The user\'s commission balance will be refunded.'
      )}
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
            variant='destructive'
            onClick={() => void handleSubmit()}
            disabled={submitting}
          >
            {submitting ? (
              <Loader2 className='size-4 animate-spin' />
            ) : (
              <RotateCcw className='size-4' />
            )}
            {t('Confirm Reversal')}
          </Button>
        </div>
      }
    >
      <div className='space-y-1'>
        <div className='flex items-center justify-between text-sm'>
          <span className='text-muted-foreground'>{t('Request ID')}</span>
          <span className='font-medium'>#{withdrawal.id}</span>
        </div>
        <div className='flex items-center justify-between text-sm'>
          <span className='text-muted-foreground'>{t('User ID')}</span>
          <span className='font-medium'>{withdrawal.user_id}</span>
        </div>
        <div className='flex items-center justify-between text-sm'>
          <span className='text-muted-foreground'>{t('Amount')}</span>
          <span className='font-semibold tabular-nums'>
            {formatCents(withdrawal.amount_cents)}
          </span>
        </div>
        {withdrawal.stripe_transfer_id && (
          <div className='flex items-center justify-between text-sm'>
            <span className='text-muted-foreground'>{t('Transfer ID')}</span>
            <span className='break-all font-mono text-xs'>
              {withdrawal.stripe_transfer_id}
            </span>
          </div>
        )}
      </div>

      <div className='space-y-1.5'>
        <Label htmlFor='reverse-reason'>
          {t('Reverse transfer reason')}
        </Label>
        <Textarea
          id='reverse-reason'
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder={t('Enter reason for reversal...')}
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
  onReverse: (withdrawal: WithdrawalRequest) => void
  processing: boolean
  reversing: boolean
}

function WithdrawalCard({
  withdrawal,
  onProcess,
  onReverse,
  processing,
  reversing,
}: WithdrawalCardProps) {
  const { t } = useTranslation()
  const isPending = withdrawal.status === 'pending'
  const isActionRequired = withdrawal.status === 'action_required'
  const statusConfig = WITHDRAWAL_STATUSES[withdrawal.status]

  const isStripeConnect =
    withdrawal.payout_channel === PAYOUT_CHANNEL_STRIPE_CONNECT
  const canReverse =
    isActionRequired && !!withdrawal.stripe_transfer_id && !reversing

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
              label={t(statusConfig.labelKey)}
              variant={statusConfig.variant}
              size='sm'
              copyable={false}
            />
            {isStripeConnect && (
              <StatusBadge
                label={t('Stripe Connect')}
                variant='purple'
                size='sm'
                copyable={false}
              />
            )}
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

      {/* Stripe Connect details */}
      {isStripeConnect && (
        <StripeConnectDetails withdrawal={withdrawal} />
      )}

      {/* Non-pending legacy / common fields */}
      {withdrawal.status !== 'pending' && (
        <div className='space-y-1 text-xs'>
          {!isStripeConnect && withdrawal.payout_channel && (
            <div className='flex gap-2'>
              <span className='text-muted-foreground'>
                {t('Payout Channel')}:
              </span>
              <span>{withdrawal.payout_channel}</span>
            </div>
          )}
          {!isStripeConnect && withdrawal.payout_tx_id && (
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

      {/* Action buttons */}
      {(isPending || canReverse) && (
        <div className='flex flex-wrap justify-end gap-2'>
          {isPending && (
            <>
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
            </>
          )}
          {canReverse && (
            <Button
              type='button'
              variant='destructive'
              size='sm'
              onClick={() => onReverse(withdrawal)}
              disabled={reversing}
            >
              <RotateCcw className='size-4' />
              {t('Reverse Transfer')}
            </Button>
          )}
        </div>
      )}
    </article>
  )
}

// ============================================================================
// Stripe Connect Details (sub-component of WithdrawalCard)
// ============================================================================

function StripeConnectDetails({
  withdrawal,
}: {
  withdrawal: WithdrawalRequest
}) {
  const { t } = useTranslation()

  const reconcileError = withdrawal.last_reconcile_error?.trim() ?? ''
  const showReconcileError = reconcileError.length > 0
  const hasReversal =
    !!withdrawal.stripe_transfer_amount_reversed &&
    withdrawal.stripe_transfer_amount_reversed > 0

  return (
    <div className='bg-muted/30 space-y-2 rounded-md border p-2.5 text-xs'>
      <div className='flex flex-wrap gap-x-4 gap-y-1'>
        {withdrawal.stripe_transfer_id && (
          <div className='flex min-w-0 gap-1.5'>
            <span className='text-muted-foreground shrink-0'>
              {t('Transfer ID')}:
            </span>
            <span className='break-all font-mono'>
              {withdrawal.stripe_transfer_id}
            </span>
          </div>
        )}
        {withdrawal.stripe_transfer_status && (
          <div className='flex gap-1.5'>
            <span className='text-muted-foreground'>
              {t('Transfer Status')}:
            </span>
            <span>{withdrawal.stripe_transfer_status}</span>
          </div>
        )}
        {withdrawal.stripe_payout_id && (
          <div className='flex min-w-0 gap-1.5'>
            <span className='text-muted-foreground shrink-0'>
              {t('Payout ID')}:
            </span>
            <span className='break-all font-mono'>
              {withdrawal.stripe_payout_id}
            </span>
          </div>
        )}
        {withdrawal.stripe_payout_status && (
          <div className='flex gap-1.5'>
            <span className='text-muted-foreground'>
              {t('Payout Status')}:
            </span>
            <span>{withdrawal.stripe_payout_status}</span>
          </div>
        )}
        {typeof withdrawal.stripe_payout_attempt === 'number' &&
          withdrawal.stripe_payout_attempt > 0 && (
            <div className='flex gap-1.5'>
              <span className='text-muted-foreground'>
                {t('Payout Attempt')}:
              </span>
              <span>{withdrawal.stripe_payout_attempt}</span>
            </div>
          )}
      </div>

      <div className='flex flex-wrap gap-x-4 gap-y-1'>
        <div className='flex gap-1.5'>
          <span className='text-muted-foreground'>
            {t('Last Reconcile')}:
          </span>
          <span>{formatRelativeTime(withdrawal.last_reconcile_at ?? 0)}</span>
        </div>
        {showReconcileError && (
          <div className='flex min-w-0 gap-1.5'>
            <span className='text-muted-foreground shrink-0'>
              {t('Reconcile Error')}:
            </span>
            <TooltipProvider delay={0}>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <span className='text-destructive truncate max-w-[280px] cursor-help underline decoration-dotted underline-offset-2' />
                  }
                >
                  {truncate(reconcileError)}
                </TooltipTrigger>
                <TooltipContent>{reconcileError}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        )}
        {hasReversal && (
          <div className='flex gap-1.5'>
            <span className='text-muted-foreground'>
              {t('Amount Reversed')}:
            </span>
            <span className='text-warning tabular-nums'>
              {formatCents(withdrawal.stripe_transfer_amount_reversed ?? 0)}
            </span>
          </div>
        )}
      </div>
    </div>
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
  const [reverseState, setReverseState] = useState<ReverseDialogState | null>(
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

  // Auto-refresh the queue every 30s while there are rows in non-terminal
  // Stripe Connect states (transfer_creating / awaiting_funds /
  // payout_creating / processing) so the admin sees webhook-driven
  // transitions without manual refresh. Stops when the queue has only
  // terminal rows to avoid background polling noise.
  const shouldAutoRefresh = useMemo(() => hasNonTerminalItems(items), [items])
  useEffect(() => {
    if (!shouldAutoRefresh) return
    const timer = window.setInterval(() => {
      void loadQueue()
    }, WITHDRAWAL_AUTO_REFRESH_MS)
    return () => window.clearInterval(timer)
  }, [shouldAutoRefresh, loadQueue])

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

  const handleReverse = (withdrawal: WithdrawalRequest) => {
    setReverseState({ withdrawal })
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
                  onReverse={handleReverse}
                  processing={processState !== null}
                  reversing={reverseState !== null}
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
      <ReverseTransferDialog
        state={reverseState}
        onOpenChange={setReverseState}
        onSuccess={() => void loadQueue()}
      />
    </SectionPageLayout>
  )
}
