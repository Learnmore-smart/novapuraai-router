import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Check, Crown, Sparkles } from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getPublicPlans,
  getSelfSubscriptionFull,
} from '@/features/subscriptions/api'
import { formatDuration } from '@/features/subscriptions/lib'
import type {
  PlanRecord,
  SubscriptionCheckoutCurrency,
} from '@/features/subscriptions/types'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'
import { useBillingCurrencyStore } from '@/stores/billing-currency-store'

import { PurchaseModal } from './purchase-modal'

function currencySymbol(currency: SubscriptionCheckoutCurrency): string {
  return currency === 'CNY' ? '¥' : '$'
}

function planPrice(
  plan: PlanRecord['plan'],
  currency: SubscriptionCheckoutCurrency
): number {
  const base =
    currency === 'CNY' ? plan.price_amount_cny : plan.price_amount_usd
  return typeof base === 'number' ? base : plan.price_amount
}

interface PlanCardProps {
  record: PlanRecord
  currency: SubscriptionCheckoutCurrency
  isCurrent: boolean
  onSubscribe: (plan: PlanRecord) => void
}

function PlanCard(props: PlanCardProps) {
  const { t } = useTranslation()
  const plan = props.record.plan
  const price = planPrice(plan, props.currency)
  const totalAmount = Number(plan.total_amount || 0)

  const benefits = [
    formatDuration(plan, t),
    totalAmount > 0
      ? `${t('Total Quota')}: ${formatQuota(totalAmount)}`
      : `${t('Total Quota')}: ${t('Unlimited')}`,
  ]

  return (
    <Card data-card-hover='false' className='border-primary/60 shadow-sm'>
      <CardContent className='flex h-full flex-col p-5 sm:p-6'>
        <div className='mb-3 flex items-start justify-between gap-3'>
          <div className='min-w-0'>
            <h3 className='text-lg font-semibold'>{plan.title}</h3>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t(
                'Unlimited access to included models. Models outside the plan continue to use your account balance.'
              )}
            </p>
          </div>
          <span className='bg-primary/10 text-primary inline-flex shrink-0 items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium'>
            <Sparkles className='h-3 w-3' />
            {t('Recommended')}
          </span>
        </div>

        <div className='py-3'>
          <span className='text-primary text-4xl font-bold'>
            {currencySymbol(props.currency)}
            {price.toFixed(2)}
          </span>
          <span className='text-muted-foreground ml-1 text-sm'>
            / {t('mo')}
          </span>
        </div>

        <div className='flex-1 space-y-2 pb-4'>
          {benefits.map((label) => (
            <div
              key={label}
              className='text-muted-foreground flex items-center gap-2 text-sm'
            >
              <Check className='text-primary h-4 w-4 shrink-0' />
              <span>{label}</span>
            </div>
          ))}
        </div>

        {props.isCurrent ? (
          <div className='space-y-2'>
            <span className='bg-primary/10 text-primary inline-flex w-full items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium'>
              <Crown className='h-4 w-4' />
              {t('Current plan')}
            </span>
            <Link
              to='/account/subscription'
              className='border-border text-foreground hover:bg-muted inline-flex w-full items-center justify-center rounded-lg border px-3 py-2 text-sm font-medium transition-colors'
            >
              {t('Manage subscription')}
            </Link>
          </div>
        ) : (
          <Button
            className='w-full'
            onClick={() => props.onSubscribe(props.record)}
          >
            {t('Subscribe now')}
          </Button>
        )}
      </CardContent>
    </Card>
  )
}

export function PlansPricing() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const isAuthed = !!auth.user
  const billingCurrency = useBillingCurrencyStore((s) => s.selectedCurrency)

  const [currency, setCurrency] = useState<SubscriptionCheckoutCurrency>(() =>
    isAuthed && billingCurrency === 'cny' ? 'CNY' : 'USD'
  )
  const [selectedPlan, setSelectedPlan] = useState<PlanRecord | null>(null)
  const [purchaseOpen, setPurchaseOpen] = useState(false)

  const plansQuery = useQuery({
    queryKey: ['subscription-plans-public'],
    queryFn: getPublicPlans,
    enabled: isAuthed,
  })
  const selfQuery = useQuery({
    queryKey: ['subscription-self'],
    queryFn: getSelfSubscriptionFull,
    enabled: isAuthed,
  })

  const plans = plansQuery.data?.data ?? []
  const isLoading = isAuthed && plansQuery.isPending

  // currentPlanIds is the set of plan IDs the user currently holds, across
  // every status that still represents an active entitlement: active (auto-
  // renew paid & current), prepaid_active (prepaid months remaining),
  // canceling (auto-renew with cancel_at_period_end, still entitled until
  // period end), and past_due (renewal failed, grace period). Expired /
  // canceled / payment_failed subscriptions do NOT count — they confer no
  // current entitlement.
  //
  // We read from `all_subscriptions` (not `subscriptions`) because the
  // backend's `subscriptions` field is populated by
  // GetAllActiveUserSubscriptions, which only returns status='active'.
  // Without this, a user with a prepaid_active or canceling subscription
  // would not see the "Current plan" badge and could attempt a duplicate
  // purchase (the backend's HasActiveAutoRenewSubscription catches auto-
  // renew duplicates, but the UI should reflect reality first).
  const currentPlanIds = useMemo(() => {
    const ids = new Set<number>()
    const currentStatuses = new Set([
      'active',
      'prepaid_active',
      'canceling',
      'past_due',
    ])
    for (const sub of selfQuery.data?.data?.all_subscriptions ?? []) {
      const planId = sub?.subscription?.plan_id
      const status = sub?.subscription?.status
      if (planId && status && currentStatuses.has(status)) {
        ids.add(planId)
      }
    }
    return ids
  }, [selfQuery.data])

  const handleSubscribe = (plan: PlanRecord) => {
    setSelectedPlan(plan)
    setPurchaseOpen(true)
  }

  let body: ReactNode
  if (isLoading) {
    body = (
      <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
        <Skeleton className='h-80 w-full' />
        <Skeleton className='h-80 w-full' />
      </div>
    )
  } else if (!isAuthed) {
    body = (
      <Card>
        <CardContent className='flex flex-col items-center gap-4 p-8 text-center'>
          <Crown className='text-primary h-8 w-8' />
          <p className='text-muted-foreground'>
            {t('Sign in to view plans and subscribe')}
          </p>
          <div className='flex gap-2'>
            <Button render={<Link to='/sign-in' />}>{t('Sign in')}</Button>
            <Button variant='outline' render={<Link to='/sign-up' />}>
              {t('Sign up')}
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  } else if (plans.length === 0) {
    body = (
      <p className='text-muted-foreground py-10 text-center text-sm'>
        {t('No plans available')}
      </p>
    )
  } else {
    body = (
      <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
        {plans.map((record) => (
          <PlanCard
            key={record.plan.id}
            record={record}
            currency={currency}
            isCurrent={currentPlanIds.has(record.plan.id)}
            onSubscribe={handleSubscribe}
          />
        ))}
      </div>
    )
  }

  return (
    <PublicLayout>
      <div className='mx-auto max-w-5xl space-y-6'>
        <div className='flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between'>
          <div>
            <h1 className='text-2xl font-bold sm:text-3xl'>
              {t('Plans & Pricing')}
            </h1>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t('Subscribe to a plan for model access')}
            </p>
          </div>
          <div className='inline-flex rounded-lg border p-0.5' role='group'>
            {(['CNY', 'USD'] as const).map((c) => (
              <button
                key={c}
                type='button'
                onClick={() => setCurrency(c)}
                className={cn(
                  'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                  currency === c
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                {currencySymbol(c)} {c}
              </button>
            ))}
          </div>
        </div>

        {body}
      </div>

      {selectedPlan && (
        <PurchaseModal
          plan={selectedPlan}
          currency={currency}
          open={purchaseOpen}
          onOpenChange={setPurchaseOpen}
        />
      )}
    </PublicLayout>
  )
}
