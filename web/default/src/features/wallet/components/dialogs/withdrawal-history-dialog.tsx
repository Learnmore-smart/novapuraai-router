import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { StatusBadge } from '@/components/status-badge'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { WITHDRAWAL_STATUSES } from '@/features/withdrawals/constants'
import type { WithdrawalRequest } from '@/features/withdrawals/types'

interface WithdrawalHistoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  items: WithdrawalRequest[]
  loading: boolean
}

function formatTime(unixSeconds: number): string {
  if (!unixSeconds) return '-'
  return new Date(unixSeconds * 1000).toLocaleString()
}

function formatCents(cents: number): string {
  return formatCurrencyFromUSD(cents / 100)
}

export function WithdrawalHistoryDialog({
  open,
  onOpenChange,
  items,
  loading,
}: WithdrawalHistoryDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Withdrawal History')}
      description={t('Your cash commission withdrawal requests')}
      contentClassName='sm:max-w-2xl'
      contentHeight='auto'
      bodyClassName='space-y-3'
      footer={
        <Button
          variant='outline'
          onClick={() => onOpenChange(false)}
        >
          {t('Close')}
        </Button>
      }
    >
      {loading && (
        <div className='space-y-3'>
          {['h1', 'h2', 'h3'].map((id) => (
            <Skeleton key={id} className='h-16 w-full rounded-lg' />
          ))}
        </div>
      )}

      {!loading && items.length === 0 && (
        <div className='text-muted-foreground flex min-h-32 items-center justify-center rounded-lg border border-dashed text-sm'>
          {t('No withdrawal requests yet')}
        </div>
      )}

      {!loading && items.length > 0 && (
        <div className='space-y-3'>
          {items.map((w) => {
            const status = WITHDRAWAL_STATUSES[w.status]
            return (
              <article
                key={w.id}
                className='bg-background space-y-2 rounded-lg border p-3'
              >
                <div className='flex flex-wrap items-center justify-between gap-2'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <span className='text-sm font-semibold tabular-nums'>
                      {formatCents(w.amount_cents)}
                    </span>
                    {status && (
                      <StatusBadge
                        label={t(status.labelKey)}
                        variant={status.variant}
                        size='sm'
                        copyable={false}
                      />
                    )}
                  </div>
                  <span className='text-muted-foreground text-xs'>
                    {formatTime(w.requested_at)}
                  </span>
                </div>

                {w.status !== 'pending' && (
                  <div className='space-y-1 text-xs'>
                    {w.payout_channel && (
                      <div className='flex gap-2'>
                        <span className='text-muted-foreground'>
                          {t('Payout Channel')}:
                        </span>
                        <span>{w.payout_channel}</span>
                      </div>
                    )}
                    {w.payout_tx_id && (
                      <div className='flex gap-2'>
                        <span className='text-muted-foreground'>
                          {t('Payout Transaction ID')}:
                        </span>
                        <span className='break-all'>{w.payout_tx_id}</span>
                      </div>
                    )}
                    {w.admin_remark && (
                      <div className='flex gap-2'>
                        <span className='text-muted-foreground'>
                          {t('Admin Remark')}:
                        </span>
                        <span className='break-words'>{w.admin_remark}</span>
                      </div>
                    )}
                  </div>
                )}
              </article>
            )
          })}
        </div>
      )}

      {loading && (
        <div className='text-muted-foreground flex items-center justify-center gap-2 text-xs'>
          <Loader2 className='size-3 animate-spin' />
          {t('Loading...')}
        </div>
      )}
    </Dialog>
  )
}
