import { Activity, BarChart3, Gift, WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { formatQuota } from '@/lib/format'

import type { UserWalletData } from '../types'

interface WalletStatsCardProps {
  user: UserWalletData | null
  loading?: boolean
}

export function WalletStatsCard(props: WalletStatsCardProps) {
  const { t } = useTranslation()
  if (props.loading) {
    return (
      <div className='grid grid-cols-2 divide-x divide-y rounded-lg border sm:grid-cols-4 sm:divide-y-0'>
        {['total', 'cash', 'promo', 'usage'].map((key) => (
          <div key={key} className='min-w-0 px-2.5 py-2.5 sm:px-5 sm:py-4'>
            <Skeleton className='h-3.5 w-full' />
            <Skeleton className='mt-2 h-6 w-full sm:h-7' />
            <Skeleton className='mt-1.5 hidden h-3.5 w-24 md:block' />
          </div>
        ))}
      </div>
    )
  }

  const total = props.user?.quota ?? 0
  const promo = props.user?.promo_quota ?? 0
  const cash =
    props.user?.cash_quota ?? Math.max(0, total - promo)

  const stats: {
    label: string
    value: string
    description: string
    icon: typeof WalletCards
    tone: IconBadgeTone
  }[] = [
    {
      label: t('API Balance'),
      value: formatQuota(total),
      description: t(
        'Spendable for API calls. Cash + gift (gift is used first).'
      ),
      icon: WalletCards,
      tone: 'success',
    },
    {
      label: t('Cash'),
      value: formatQuota(cash),
      description: t('From top-ups (refundable per policy)'),
      icon: WalletCards,
      tone: 'info',
    },
    {
      label: t('Gift / Promo'),
      value: formatQuota(promo),
      description: t('Register, invite, share, and top-up bonuses'),
      icon: Gift,
      tone: 'chart-4',
    },
    {
      label: t('Total Usage'),
      value: formatQuota(props.user?.used_quota ?? 0),
      description: t('Requests: {{count}}', {
        count: props.user?.request_count ?? 0,
      }),
      icon: BarChart3,
      tone: 'warning',
    },
  ]

  return (
    <div className='grid grid-cols-2 divide-x divide-y rounded-lg border sm:grid-cols-4 sm:divide-y-0'>
      {stats.map((item) => (
        <div key={item.label} className='min-w-0 px-2.5 py-2.5 sm:px-5 sm:py-4'>
          <div className='flex items-center gap-1.5 sm:gap-2.5'>
            <IconBadge tone={item.tone} size='stat'>
              <item.icon />
            </IconBadge>
            <div className='text-muted-foreground truncate text-[11px] font-medium tracking-wider uppercase sm:text-xs'>
              {item.label}
            </div>
          </div>

          <div className='text-foreground mt-1.5 font-mono text-sm font-bold tracking-tight break-all tabular-nums sm:mt-2.5 sm:text-2xl'>
            {item.value}
          </div>
          <div className='text-muted-foreground/60 mt-1 hidden text-xs md:block'>
            {item.description}
          </div>
        </div>
      ))}
      {/* Keep request count accessible without fifth column on small screens */}
      <div className='text-muted-foreground col-span-2 flex items-center gap-2 border-t px-2.5 py-2 text-xs sm:col-span-4 sm:px-5'>
        <Activity className='size-3.5 shrink-0' />
        {t('API Requests')}:{' '}
        <span className='text-foreground font-mono font-medium'>
          {(props.user?.request_count ?? 0).toLocaleString()}
        </span>
      </div>
    </div>
  )
}
