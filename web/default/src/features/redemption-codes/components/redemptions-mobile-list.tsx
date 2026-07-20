import type { Table as TanstackTable } from '@tanstack/react-table'
import { Database } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { DISABLED_ROW_MOBILE } from '@/components/data-table'
import { MaskedValueDisplay } from '@/components/masked-value-display'
import { StatusBadge } from '@/components/status-badge'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import {
  REDEMPTION_CURRENCY_LABELS,
  REDEMPTION_CURRENCY_SYMBOLS,
  REDEMPTION_STATUS,
  REDEMPTION_STATUSES,
  normalizeRedemptionCurrency,
} from '../constants'
import { isRedemptionExpired } from '../lib'
import type { Redemption } from '../types'
import { DataTableRowActions } from './data-table-row-actions'

const MOBILE_SKELETON_KEYS = [
  'redemption-mobile-skeleton-1',
  'redemption-mobile-skeleton-2',
  'redemption-mobile-skeleton-3',
  'redemption-mobile-skeleton-4',
  'redemption-mobile-skeleton-5',
]

function RedemptionsMobileSkeleton() {
  return (
    <div className='divide-border overflow-hidden rounded-lg border'>
      {MOBILE_SKELETON_KEYS.map((key) => (
        <div
          key={key}
          className='space-y-2 border-b px-3 py-2.5 last:border-b-0'
        >
          <div className='flex items-center justify-between'>
            <Skeleton className='h-4 w-32' />
            <Skeleton className='h-5 w-16 rounded-md' />
          </div>
          <div className='flex items-center justify-between gap-3'>
            <Skeleton className='h-7 w-44' />
            <Skeleton className='h-8 w-16' />
          </div>
          <Skeleton className='h-3 w-28' />
        </div>
      ))}
    </div>
  )
}

interface RedemptionsMobileListProps {
  table: TanstackTable<Redemption>
  isLoading: boolean
}

export function RedemptionsMobileList(props: RedemptionsMobileListProps) {
  const { t } = useTranslation()
  const rows = props.table.getRowModel().rows

  if (props.isLoading) return <RedemptionsMobileSkeleton />

  if (!rows.length) {
    return (
      <div className='rounded-lg border p-8'>
        <Empty className='border-none p-0'>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <Database className='size-6' />
            </EmptyMedia>
            <EmptyTitle>{t('No Redemption Codes Found')}</EmptyTitle>
            <EmptyDescription>
              {t(
                'No redemption codes available. Create your first redemption code to get started.'
              )}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      </div>
    )
  }

  return (
    <div className='divide-border overflow-hidden rounded-lg border'>
      {rows.map((row) => {
        const redemption = row.original
        const expired = isRedemptionExpired(
          redemption.expired_time,
          redemption.status
        )
        const statusConfig = REDEMPTION_STATUSES[redemption.status]
        const maskedKey = `${redemption.key.slice(0, 8)}******${redemption.key.slice(-8)}`

        return (
          <div
            key={row.id}
            className={cn(
              'bg-card space-y-2.5 border-b px-3 py-2.5 last:border-b-0',
              expired || redemption.status !== REDEMPTION_STATUS.ENABLED
                ? DISABLED_ROW_MOBILE
                : undefined
            )}
          >
            <div className='flex items-start justify-between gap-3'>
              <div className='min-w-0'>
                <div className='truncate text-sm font-semibold'>
                  {redemption.name}
                </div>
                <div className='text-muted-foreground text-[11px]'>
                  {t('Redemption Code')}
                </div>
              </div>
              {expired ? (
                <StatusBadge
                  label={t('Expired')}
                  variant='warning'
                  copyable={false}
                />
              ) : (
                statusConfig && (
                  <StatusBadge
                    label={t(statusConfig.labelKey)}
                    variant={statusConfig.variant}
                    copyable={false}
                  />
                )
              )}
            </div>

            <div className='flex min-w-0 items-center justify-between gap-2'>
              <div className='min-w-0 flex-1 [&_button:first-child]:max-w-full [&_button:first-child]:truncate [&_button:first-child]:px-0'>
                <MaskedValueDisplay
                  label={t('Full Code')}
                  fullValue={redemption.key}
                  maskedValue={maskedKey}
                  copyTooltip={t('Copy code')}
                  copyAriaLabel={t('Copy redemption code')}
                />
              </div>
              <DataTableRowActions row={row} />
            </div>

            <div className='flex items-center justify-between gap-2 text-xs'>
              <span className='text-muted-foreground'>{t('Price')}</span>
              <span className='font-medium tabular-nums'>
                {(() => {
                  const currency = normalizeRedemptionCurrency(
                    redemption.currency
                  )
                  const amount =
                    redemption.amount > 0
                      ? redemption.amount
                      : redemption.quota / 500000
                  const symbol = REDEMPTION_CURRENCY_SYMBOLS[currency]
                  const label = REDEMPTION_CURRENCY_LABELS[currency]
                  const formatted =
                    amount % 1 === 0 ? amount.toFixed(0) : amount.toFixed(2)
                  return `${symbol}${formatted} ${label}`
                })()}
              </span>
            </div>
            <div className='flex items-center justify-between gap-2 text-xs'>
              <span className='text-muted-foreground'>{t('Usage')}</span>
              <span className='font-medium tabular-nums'>
                {redemption.redeemed_count || 0} /{' '}
                {redemption.max_redeems && redemption.max_redeems > 0
                  ? redemption.max_redeems
                  : 1}
              </span>
            </div>
          </div>
        )
      })}
    </div>
  )
}
