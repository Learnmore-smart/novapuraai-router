import { HandCoins, History, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { CommissionSummary } from '@/features/withdrawals/types'
import { formatCurrencyFromUSD } from '@/lib/currency'

interface CommissionCardProps {
  summary: CommissionSummary | null
  loading: boolean
  approved: boolean
  withdrawing: boolean
  onWithdraw: () => void
  onShowHistory: () => void
  hasHistory: boolean
}

function StatBlock({
  label,
  value,
  hint,
}: {
  label: string
  value: string
  hint?: string
}) {
  return (
    <div className='min-w-0'>
      <div className='text-muted-foreground truncate text-[10px] font-medium tracking-wider uppercase'>
        {label}
      </div>
      <div className='mt-0.5 truncate text-sm font-semibold tabular-nums'>
        {value}
      </div>
      {hint && (
        <div className='text-muted-foreground mt-0.5 truncate text-[10px]'>
          {hint}
        </div>
      )}
    </div>
  )
}

export function CommissionCard({
  summary,
  loading,
  approved,
  withdrawing,
  onWithdraw,
  onShowHistory,
  hasHistory,
}: CommissionCardProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <Card data-card-hover='false' className='bg-muted/20 py-0'>
        <CardContent className='grid gap-4 p-3 sm:p-4 lg:grid-cols-[minmax(220px,1fr)_minmax(220px,0.72fr)_minmax(320px,1.15fr)] lg:items-center'>
          <div>
            <Skeleton className='h-5 w-32' />
            <Skeleton className='mt-2 h-4 w-48' />
          </div>
          <Skeleton className='h-14 rounded-lg' />
          <Skeleton className='h-10 rounded-lg' />
        </CardContent>
      </Card>
    )
  }

  if (!approved) {
    // Non-approved users don't see the cash commission card at all — they
    // continue on the legacy ¥100 API quota invite reward path.
    return null
  }

  const pending = summary?.pending_cents ?? 0
  const balance = summary?.balance_cents ?? 0
  const total = summary?.total_cents ?? 0
  const withdrawn = summary?.withdrawn_cents ?? 0
  const freezeDays = summary?.freeze_days ?? 0

  const freezeHint =
    freezeDays > 0
      ? t('Frozen {{days}}d', { days: freezeDays })
      : t('Available immediately')

  return (
    <Card data-card-hover='false' className='bg-muted/20 py-0'>
      <CardContent className='grid gap-3 p-3 sm:gap-4 sm:p-4 lg:grid-cols-[minmax(200px,1fr)_minmax(220px,0.9fr)_minmax(280px,1fr)] lg:items-center'>
        <div className='flex min-w-0 items-center gap-2.5'>
          <IconBadge tone='chart-4'>
            <HandCoins />
          </IconBadge>
          <div className='min-w-0'>
            <h3 className='truncate text-sm font-semibold'>
              {t('Cash Commission')}
            </h3>
            <TooltipProvider delay={0}>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <p className='text-muted-foreground line-clamp-1 cursor-help text-xs' />
                  }
                >
                  {t(
                    'Earn 25% cash commission on invitee payments. This is withdrawable cash, not API usage quota.'
                  )}
                </TooltipTrigger>
                <TooltipContent className='max-w-sm text-xs leading-relaxed'>
                  {t(
                    'Earn 25% cash commission on invitee payments. This is withdrawable cash, not API usage quota.'
                  )}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        </div>

        <div className='grid grid-cols-2 gap-2 sm:grid-cols-4 lg:gap-3'>
          <StatBlock
            label={t('Frozen')}
            value={formatCurrencyFromUSD(pending / 100)}
            hint={freezeHint}
          />
          <StatBlock
            label={t('Withdrawable')}
            value={formatCurrencyFromUSD(balance / 100)}
          />
          <StatBlock
            label={t('Lifetime Earned')}
            value={formatCurrencyFromUSD(total / 100)}
          />
          <StatBlock
            label={t('Withdrawn')}
            value={formatCurrencyFromUSD(withdrawn / 100)}
          />
        </div>

        <div className='flex flex-wrap items-center justify-end gap-2'>
          {hasHistory && (
            <Button
              onClick={onShowHistory}
              variant='outline'
              size='sm'
              className='h-9 shrink-0 px-3'
              disabled={withdrawing}
            >
              <History className='size-4' />
              {t('History')}
            </Button>
          )}
          <Button
            onClick={onWithdraw}
            size='sm'
            className='h-9 shrink-0 px-3'
            disabled={withdrawing || balance <= 0}
          >
            {withdrawing ? (
              <Loader2 className='size-4 animate-spin' />
            ) : (
              <HandCoins className='size-4' />
            )}
            {t('Withdraw')}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
