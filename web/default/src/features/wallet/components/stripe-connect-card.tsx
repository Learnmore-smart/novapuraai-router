import {
  AlertTriangle,
  CreditCard,
  ExternalLink,
  Loader2,
  RefreshCw,
  Check,
  X,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'

import type {
  StripeConnectAccount,
  StripeConnectOnboardingState,
  StripeConnectStatus,
} from '../hooks/use-stripe-connect'

interface StripeConnectCardProps {
  status: StripeConnectStatus | null
  loading: boolean
  starting: boolean
  onStart: () => Promise<void>
  onRefresh?: () => void
}

// currently_due is a JSON-encoded string array from the backend.
function parseCurrentlyDue(raw: string | undefined): string[] {
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return parsed.filter((item): item is string => typeof item === 'string')
    }
  } catch {
    /* malformed payload — render nothing */
  }
  return []
}

export function StripeConnectCard({
  status,
  loading,
  starting,
  onStart,
  onRefresh,
}: StripeConnectCardProps) {
  const { t } = useTranslation()
  const [refreshing, setRefreshing] = useState(false)

  const handleRefresh = async () => {
    if (!onRefresh) return
    setRefreshing(true)
    try {
      await onRefresh()
    } finally {
      setRefreshing(false)
    }
  }

  const account = status?.started ? status.account : undefined
  const state = account?.onboarding_state

  const dueItems = useMemo(
    () => parseCurrentlyDue(account?.currently_due),
    [account?.currently_due]
  )

  if (loading) {
    return (
      <Card data-card-hover='false' className='bg-muted/20 py-0'>
        <CardContent className='flex items-center gap-3 p-3 sm:p-4'>
          <Skeleton className='size-8 rounded-lg' />
          <div className='min-w-0 flex-1'>
            <Skeleton className='h-4 w-40' />
            <Skeleton className='mt-2 h-3 w-56' />
          </div>
          <Skeleton className='h-9 w-36 rounded-lg' />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card data-card-hover='false' className='bg-muted/20 py-0'>
      <CardContent className='flex flex-col gap-3 p-3 sm:flex-row sm:items-center sm:gap-4 sm:p-4'>
        <div className='flex min-w-0 flex-1 items-center gap-2.5'>
          <StripeStateIcon state={state} />
          <div className='min-w-0 space-y-1'>
            <div className='flex flex-wrap items-center gap-2'>
              <h3 className='text-sm font-semibold'>{t('Stripe Connect')}</h3>
              <StateBadge state={state} />
            </div>
            <StripeStateDescription state={state} dueItems={dueItems} />
          </div>
        </div>

        <div className='flex shrink-0 items-center gap-2'>
          {onRefresh && state && state !== 'enabled' && (
            <Button
              onClick={() => void handleRefresh()}
              variant='outline'
              size='sm'
              className='h-9 px-3'
              disabled={refreshing || starting}
            >
              <RefreshCw
                className={refreshing ? 'size-4 animate-spin' : 'size-4'}
              />
              {t('Refresh')}
            </Button>
          )}
          <StripeActionButton
            state={state}
            starting={starting}
            onStart={onStart}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function StripeStateIcon({
  state,
}: {
  state: StripeConnectOnboardingState | undefined
}) {
  switch (state) {
    case 'enabled':
      return (
        <IconBadge tone='success' size='md'>
          <Check />
        </IconBadge>
      )
    case 'restricted':
      return (
        <IconBadge tone='warning' size='md'>
          <AlertTriangle />
        </IconBadge>
      )
    case 'rejected':
      return (
        <IconBadge tone='destructive' size='md'>
          <X />
        </IconBadge>
      )
    default:
      return (
        <IconBadge tone='info' size='md'>
          <CreditCard />
        </IconBadge>
      )
  }
}

function StateBadge({
  state,
}: {
  state: StripeConnectOnboardingState | undefined
}) {
  const { t } = useTranslation()
  if (state === 'enabled') {
    return (
      <Badge variant='secondary' className='bg-success/15 text-success'>
        {t('Stripe account connected')}
      </Badge>
    )
  }
  if (state === 'rejected') {
    return <Badge variant='destructive'>{t('Rejected')}</Badge>
  }
  if (state === 'restricted') {
    return <Badge variant='outline'>{t('Action required')}</Badge>
  }
  if (state === 'created' || state === 'onboarding') {
    return <Badge variant='secondary'>{t('Onboarding in progress')}</Badge>
  }
  // Not started
  return <Badge variant='outline'>{t('Not connected')}</Badge>
}

function StripeStateDescription({
  state,
  dueItems,
}: {
  state: StripeConnectOnboardingState | undefined
  dueItems: string[]
}) {
  const { t } = useTranslation()
  switch (state) {
    case 'enabled':
      return (
        <p className='text-muted-foreground text-xs leading-relaxed'>
          {t('Your Stripe account is connected and ready to receive payouts.')}
        </p>
      )
    case 'rejected':
      return (
        <p className='text-destructive text-xs leading-relaxed'>
          {t('Stripe account rejected — please contact support')}
        </p>
      )
    case 'restricted':
      return (
        <div className='space-y-1'>
          <p className='text-warning text-xs leading-relaxed'>
            {t('Action required — please complete Stripe onboarding')}
          </p>
          {dueItems.length > 0 && (
            <ul className='text-muted-foreground list-disc pl-4 text-xs'>
              {dueItems.slice(0, 5).map((item) => (
                <li key={item} className='break-all'>
                  {item}
                </li>
              ))}
            </ul>
          )}
        </div>
      )
    case 'created':
    case 'onboarding':
      return (
        <p className='text-muted-foreground text-xs leading-relaxed'>
          {t('Onboarding in progress')} —{' '}
          {t('Resume to finish linking your Stripe account.')}
        </p>
      )
    default:
      return (
        <p className='text-muted-foreground text-xs leading-relaxed'>
          {t(
            'Connect a Stripe account to enable automatic payouts for cash commission withdrawals.'
          )}
        </p>
      )
  }
}

function StripeActionButton({
  state,
  starting,
  onStart,
}: {
  state: StripeConnectOnboardingState | undefined
  starting: boolean
  onStart: () => Promise<void>
}) {
  const { t } = useTranslation()

  // enabled / rejected: no primary action button (badge + description suffice).
  if (state === 'enabled' || state === 'rejected') {
    return null
  }

  const isResume = state === 'created' || state === 'onboarding'
  const label = isResume ? t('Resume onboarding') : t('Connect Stripe Account')

  return (
    <Button
      onClick={() => void onStart()}
      size='sm'
      className='h-9 shrink-0 px-3'
      disabled={starting}
    >
      {starting ? (
        <Loader2 className='size-4 animate-spin' />
      ) : (
        <ExternalLink className='size-4' />
      )}
      {label}
    </Button>
  )
}

// Re-export the account type for callers that need it (e.g. WithdrawalDialog).
export type { StripeConnectAccount }
