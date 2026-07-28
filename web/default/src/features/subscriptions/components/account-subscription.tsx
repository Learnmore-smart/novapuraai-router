import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Crown } from 'lucide-react'
import { useEffect, useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import dayjs from '@/lib/dayjs'

import { getPublicPlans, getSelfSubscriptionFull } from '../api'
import type {
  PlanRecord,
  SubscriptionCheckoutCurrency,
  SubscriptionSelfDto,
  UserSubscriptionRecord,
} from '../types'
import { PortalButton } from './portal-button'
import { SubscriptionStatusBadge } from './subscription-status-badge'

function currencySymbol(currency: string): string {
  return currency.toUpperCase() === 'CNY' ? '¥' : '$'
}

function formatDate(ts: number | undefined): string {
  if (!ts) return '-'
  return dayjs(ts * 1000).format('YYYY-MM-DD')
}

interface DetailRowProps {
  label: string
  children: ReactNode
}

function DetailRow(props: DetailRowProps) {
  return (
    <div className='flex items-start justify-between gap-3 py-1.5'>
      <span className='text-muted-foreground text-sm'>{props.label}</span>
      <span className='text-right text-sm font-medium'>{props.children}</span>
    </div>
  )
}

interface ActiveSubscriptionCardProps {
  subscription: SubscriptionSelfDto
  plan: PlanRecord['plan'] | undefined
}

function ActiveSubscriptionCard(props: ActiveSubscriptionCardProps) {
  const { t } = useTranslation()
  const sub = props.subscription
  const plan = props.plan
  const currency = (sub.currency || 'USD') as SubscriptionCheckoutCurrency
  const status = sub.status
  const isPrepaid = status === 'prepaid_active'
  // Renewal vs expiry label: prepaid / terminal statuses show "Expires".
  const showExpiryLabel = isPrepaid || status === 'canceled' || status === 'expired'
  const dateValue = sub.next_renewal_date ?? sub.end_time

  let price = 0
  if (plan) {
    const base =
      currency === 'CNY' ? plan.price_amount_cny : plan.price_amount_usd
    price = typeof base === 'number' ? base : plan.price_amount
  }

  return (
    <Card className='border-primary/60'>
      <CardContent className='space-y-4 p-5 sm:p-6'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div className='min-w-0'>
            <div className='flex items-center gap-2'>
              <Crown className='text-primary h-5 w-5 shrink-0' />
              <h2 className='truncate text-lg font-semibold'>
                {plan?.title ?? t('Plan')}
              </h2>
            </div>
            {plan?.subtitle && (
              <p className='text-muted-foreground mt-1 text-sm'>
                {plan.subtitle}
              </p>
            )}
          </div>
          <SubscriptionStatusBadge
            status={status}
            displayStatus={sub.display_status}
          />
        </div>

        <div className='divide-y divide-border'>
          {price > 0 && (
            <DetailRow label={t('Price')}>
              {currencySymbol(currency)}
              {price.toFixed(2)}
              <span className='text-muted-foreground'>
                {t('/month')}
              </span>
            </DetailRow>
          )}
          <DetailRow label={t('Start date')}>
            {formatDate(sub.start_time)}
          </DetailRow>
          <DetailRow label={showExpiryLabel ? t('Expires') : t('Next renewal')}>
            {formatDate(dateValue)}
          </DetailRow>
          {sub.coupon_id != null && (
            <DetailRow label={t('Current coupon')}>{`#${sub.coupon_id}`}</DetailRow>
          )}
        </div>

        {sub.cancel_at_period_end && dateValue && (
          <Alert>
            <AlertDescription className='text-xs'>
              {t('Your subscription will not renew. Active until {{date}}.', {
                date: formatDate(dateValue),
              })}
            </AlertDescription>
          </Alert>
        )}

        <div className='flex flex-wrap gap-2'>
          {/* Stripe-managed subscriptions: portal handles all actions. */}
          {sub.is_auto_renew && (
            <PortalButton label={t('Manage subscription')} />
          )}
          <PortalButton label={t('View invoices')} variant='outline' />
          {status === 'canceling' && (
            <PortalButton label={t('Reactivate')} variant='outline' />
          )}
          {(status === 'past_due' || status === 'payment_failed') && (
            <PortalButton
              label={t('Update payment method')}
              variant='outline'
            />
          )}
        </div>
      </CardContent>
    </Card>
  )
}

interface HistoryListProps {
  subscriptions: UserSubscriptionRecord[]
  planTitleById: Map<number, string>
}

function HistoryList(props: HistoryListProps) {
  const { t } = useTranslation()
  if (props.subscriptions.length === 0) return null

  return (
    <div className='space-y-2'>
      <h3 className='text-sm font-medium'>{t('Subscription history')}</h3>
      <div className='divide-y divide-border overflow-hidden rounded-lg border'>
        {props.subscriptions.map((record) => {
          const sub = record.subscription
          const title = props.planTitleById.get(sub.plan_id) ?? `#${sub.plan_id}`
          return (
            <div
              key={sub.id}
              className='flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm'
            >
              <span className='min-w-0 truncate font-medium'>{title}</span>
              <div className='flex shrink-0 items-center gap-3'>
                <span className='text-muted-foreground'>
                  {formatDate(sub.start_time)} – {formatDate(sub.end_time)}
                </span>
                <SubscriptionStatusBadge status={sub.status} />
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export function AccountSubscription() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const selfQuery = useQuery({
    queryKey: ['subscription-self'],
    queryFn: getSelfSubscriptionFull,
  })
  const plansQuery = useQuery({
    queryKey: ['subscription-plans-public'],
    queryFn: getPublicPlans,
  })

  // When redirected from checkout with ?refresh=1, force a fresh fetch so the
  // newly-activated subscription appears without waiting for staleTime.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('refresh') === '1') {
      void queryClient.invalidateQueries({ queryKey: ['subscription-self'] })
      // Clean the param so a manual refresh doesn't re-trigger.
      window.history.replaceState({}, '', window.location.pathname)
    }
  }, [queryClient])

  const plansData = plansQuery.data?.data
  const planById = useMemo(() => {
    const map = new Map<number, PlanRecord['plan']>()
    for (const record of plansData ?? []) {
      map.set(record.plan.id, record.plan)
    }
    return map
  }, [plansData])

  const planTitleById = useMemo(() => {
    const map = new Map<number, string>()
    for (const record of plansData ?? []) {
      map.set(record.plan.id, record.plan.title)
    }
    return map
  }, [plansData])

  const data = selfQuery.data?.data
  const current = data?.current_subscription ?? null
  const isLoading = selfQuery.isPending

  let content: ReactNode
  if (isLoading) {
    content = (
      <div className='mx-auto max-w-2xl space-y-3'>
        <Skeleton className='h-64 w-full' />
      </div>
    )
  } else if (!current) {
    content = (
      <Card className='mx-auto max-w-2xl'>
        <CardContent className='flex flex-col items-center gap-4 p-8 text-center'>
          <Crown className='text-muted-foreground h-8 w-8' />
          <p className='text-muted-foreground'>
            {t("You don't have an active subscription.")}
          </p>
          <Button render={<Link to='/plans' />}>{t('Browse Plans')}</Button>
        </CardContent>
      </Card>
    )
  } else {
    const plan = planById.get(current.plan_id)
    // History excludes the current active sub to avoid duplication.
    const history = (data?.all_subscriptions ?? []).filter(
      (record) => record.subscription.id !== current.id
    )
    content = (
      <div className='mx-auto max-w-2xl space-y-4'>
        <ActiveSubscriptionCard subscription={current} plan={plan} />
        <HistoryList subscriptions={history} planTitleById={planTitleById} />
      </div>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('My Subscription')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>{content}</SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
