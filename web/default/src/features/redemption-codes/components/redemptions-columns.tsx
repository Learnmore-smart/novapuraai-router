import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { MaskedValueDisplay } from '@/components/masked-value-display'
import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatTimestampToDate } from '@/lib/format'

import {
  REDEMPTION_CURRENCY_LABELS,
  REDEMPTION_CURRENCY_SYMBOLS,
  REDEMPTION_FILTER_EXPIRED,
  REDEMPTION_STATUSES,
  normalizeRedemptionCurrency,
} from '../constants'
import { isRedemptionExpired, isTimestampExpired } from '../lib'
import { type Redemption } from '../types'
import { DataTableRowActions } from './data-table-row-actions'

// Formats the stored (currency, amount) into a single string like "¥10 CNY".
// Legacy rows without currency/amount fall back to a USD-derived price.
function formatRedemptionPrice(redemption: Redemption): string {
  const currency = normalizeRedemptionCurrency(redemption.currency)
  const amount =
    redemption.amount > 0
      ? redemption.amount
      : redemption.quota / 500000 // legacy: derive USD price from quota
  const symbol = REDEMPTION_CURRENCY_SYMBOLS[currency]
  const label = REDEMPTION_CURRENCY_LABELS[currency]
  const formattedAmount =
    amount % 1 === 0 ? amount.toFixed(0) : amount.toFixed(2)
  return `${symbol}${formattedAmount} ${label}`
}

export function useRedemptionsColumns(): ColumnDef<Redemption>[] {
  const { t } = useTranslation()
  return [
    {
      id: 'select',
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          indeterminate={table.getIsSomePageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label={t('Select all')}
          className='translate-y-[2px]'
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label={t('Select row')}
          className='translate-y-[2px]'
        />
      ),
      enableSorting: false,
      enableHiding: false,
      size: 40,
    },
    {
      accessorKey: 'id',
      header: t('ID'),
      meta: { mobileHidden: true },
      cell: ({ row }) => {
        return (
          <TableId value={row.getValue('id') as number} className='w-[60px]' />
        )
      },
      size: 80,
    },
    {
      accessorKey: 'name',
      header: t('Name'),
      meta: { mobileTitle: true },
      cell: ({ row }) => (
        <span className='font-medium'>{row.getValue('name')}</span>
      ),
      size: 180,
    },
    {
      accessorKey: 'status',
      header: t('Status'),
      meta: { mobileBadge: true },
      cell: ({ row }) => {
        const redemption = row.original
        const statusValue = row.getValue('status') as number

        // Check if expired
        if (isRedemptionExpired(redemption.expired_time, statusValue)) {
          return (
            <StatusBadge
              label={t('Expired')}
              variant='warning'
              copyable={false}
              className='-ml-1.5'
            />
          )
        }

        const statusConfig = REDEMPTION_STATUSES[statusValue]

        if (!statusConfig) {
          return null
        }

        return (
          <StatusBadge
            label={t(statusConfig.labelKey)}
            variant={statusConfig.variant}
            copyable={false}
            className='-ml-1.5'
          />
        )
      },
      filterFn: (row, id, value) => {
        const redemption = row.original
        const statusValue = row.getValue(id) as number

        // Check if expired status is being filtered
        if (value.includes(REDEMPTION_FILTER_EXPIRED)) {
          if (isRedemptionExpired(redemption.expired_time, statusValue)) {
            return true
          }
        }

        // Check regular status
        return value.includes(String(statusValue))
      },
      size: 120,
    },
    {
      id: 'code',
      accessorKey: 'key',
      header: t('Code'),
      cell: function CodeCell({ row }) {
        const redemption = row.original
        const key = redemption.key
        const maskedKey = `${key.slice(0, 8)}${'*'.repeat(16)}${key.slice(-8)}`

        return (
          <MaskedValueDisplay
            label={t('Full Code')}
            fullValue={key}
            maskedValue={maskedKey}
            copyTooltip={t('Copy code')}
            copyAriaLabel={t('Copy redemption code')}
          />
        )
      },
      enableSorting: false,
      size: 320,
    },
    {
      id: 'price',
      header: t('Price'),
      cell: ({ row }) => {
        const redemption = row.original
        return (
          <StatusBadge
            label={formatRedemptionPrice(redemption)}
            variant='neutral'
            copyable={false}
            className='-ml-1.5'
          />
        )
      },
      size: 120,
    },
    {
      id: 'usage',
      header: t('Usage'),
      meta: { mobileHidden: true },
      cell: ({ row }) => {
        const redemption = row.original
        const redeemed = redemption.redeemed_count || 0
        const max =
          redemption.max_redeems && redemption.max_redeems > 0
            ? redemption.max_redeems
            : 1
        return (
          <StatusBadge
            label={`${redeemed} / ${max}`}
            variant={redeemed >= max ? 'neutral' : 'success'}
            copyable={false}
            className='-ml-1.5'
          />
        )
      },
      size: 100,
    },
    {
      accessorKey: 'created_time',
      header: t('Created'),
      meta: { mobileHidden: true },
      cell: ({ row }) => {
        return (
          <div className='min-w-[160px] font-mono text-sm'>
            {formatTimestampToDate(row.getValue('created_time'))}
          </div>
        )
      },
      size: 180,
    },
    {
      accessorKey: 'expired_time',
      header: t('Expires'),
      meta: { mobileHidden: true },
      cell: ({ row }) => {
        const expiredTime = row.getValue('expired_time') as number
        if (expiredTime === 0) {
          return (
            <StatusBadge
              label={t('Never')}
              variant='neutral'
              copyable={false}
              className='-ml-1.5'
            />
          )
        }
        const isExpired = isTimestampExpired(expiredTime)
        return (
          <div
            className={`min-w-[160px] font-mono text-sm ${isExpired ? 'text-destructive' : ''}`}
          >
            {formatTimestampToDate(expiredTime)}
          </div>
        )
      },
      size: 180,
    },
    {
      accessorKey: 'used_user_id',
      header: t('Redeemed By'),
      meta: { mobileHidden: true },
      cell: ({ row }) => {
        const userId = row.getValue('used_user_id') as number
        const redemption = row.original

        if (userId === 0) {
          return <span className='text-muted-foreground text-sm'>-</span>
        }

        return (
          <Tooltip>
            <TooltipTrigger
              render={
                <StatusBadge
                  label={t('User {{id}}', { id: userId })}
                  variant='neutral'
                  copyable={false}
                  className='cursor-help'
                />
              }
            ></TooltipTrigger>
            <TooltipContent>
              <div className='space-y-1 text-xs'>
                <div>
                  {t('User ID:')} {userId}
                </div>
                {redemption.redeemed_time > 0 && (
                  <div>
                    {t('Redeemed:')}{' '}
                    {formatTimestampToDate(redemption.redeemed_time)}
                  </div>
                )}
              </div>
            </TooltipContent>
          </Tooltip>
        )
      },
      size: 140,
    },
    {
      id: 'actions',
      header: () => t('Actions'),
      cell: ({ row }) => <DataTableRowActions row={row} />,
      meta: { pinned: 'right' as const },
    },
  ]
}
