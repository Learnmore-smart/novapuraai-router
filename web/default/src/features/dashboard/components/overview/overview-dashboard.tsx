import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import type { TFunction } from 'i18next'
import { ArrowRight, Gift } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  CardStaggerContainer,
  CardStaggerItem,
} from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import { getApiKeys } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import {
  getSelfSubscriptionFull,
  getSubscriptionOffer,
  getStripeSubscriptionSummary,
} from '@/features/subscriptions/api'
import {
  normalizeStripeSubscriptionSummary,
  normalizeSubscriptionOffer,
} from '@/features/subscriptions/lib/subscription-offer'
import type { SelfSubscriptionData } from '@/features/subscriptions/types'
import { getUserModels } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { computeTimeRange } from '@/lib/time'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import {
  getUserQuotaDates,
  getUserSuccessfulRequestStatus,
} from '../../api'
import {
  useApiInfo,
  useDashboardContentVisibility,
} from '../../hooks/use-status-data'
import { hasConfirmedSuccessfulRequest } from '../../lib/dashboard-status'
import { AnnouncementsPanel } from './announcements-panel'
import {
  ApertureCockpit,
  type ApertureSubscriptionSummary,
} from './aperture-cockpit'
import { ApertureOnboarding } from './aperture-onboarding'
import { ApiInfoPanel } from './api-info-panel'
import { getBalanceBreakdown } from './balance-utils'
import { PerformanceHealthPanel } from './performance-health-panel'
import { SummaryCards } from './summary-cards'
import { UptimePanel } from './uptime-panel'

interface DashboardSubscriptionSnapshot {
  active: boolean
  available?: boolean
  pending?: boolean
  tier?: string
  status?: string
  model?: string
  currentPrice?: string
  currentPeriodEnd?: number | string | null
  gracePeriodEnd?: number | string | null
  cancelAtPeriodEnd?: boolean
  legacyPlanId?: number
  fairUse?: ApertureSubscriptionSummary['fairUse']
}

function getCurrentOrigin(): string {
  if (typeof window === 'undefined') return ''
  return window.location.origin
}

function normalizeEndpoint(sourceUrl?: string): string {
  const fallback = `${getCurrentOrigin()}/v1/chat/completions`
  const trimmed = sourceUrl?.trim()
  if (!trimmed) return fallback

  const withoutTrailingSlash = trimmed.replace(/\/+$/, '')
  if (withoutTrailingSlash.endsWith('/v1/chat/completions')) {
    return withoutTrailingSlash
  }
  if (withoutTrailingSlash.endsWith('/v1')) {
    return `${withoutTrailingSlash}/chat/completions`
  }
  return `${withoutTrailingSlash}/v1/chat/completions`
}

function getPreferredKey(keys: ApiKey[]): ApiKey | null {
  return keys.find((item) => item.status === 1) ?? keys[0] ?? null
}

function getActiveKey(keys: ApiKey[]): ApiKey | null {
  return keys.find((item) => item.status === 1) ?? null
}

function isActiveLegacySubscription(record: {
  subscription?: { status?: string; end_time?: number }
}): boolean {
  const subscription = record.subscription
  if (!subscription || subscription.status !== 'active') return false
  const endTime = Number(subscription.end_time ?? 0)
  return endTime <= 0 || endTime > Date.now() / 1000
}

function getLegacyActiveSubscription(data?: SelfSubscriptionData) {
  return (data?.subscriptions ?? []).find(isActiveLegacySubscription)
    ?.subscription
}

async function getDashboardSubscriptionSnapshot(): Promise<DashboardSubscriptionSnapshot | null> {
  const [offerResult, selfResult, stripeSummaryResult] =
    await Promise.allSettled([
      getSubscriptionOffer(),
      getSelfSubscriptionFull(),
      getStripeSubscriptionSummary(),
    ])
  const offerResponse =
    offerResult.status === 'fulfilled' && offerResult.value.success
      ? normalizeSubscriptionOffer(offerResult.value.data)
      : null
  const selfData =
    selfResult.status === 'fulfilled' && selfResult.value.success
      ? selfResult.value.data
      : undefined
  const stripeLifecycle =
    stripeSummaryResult.status === 'fulfilled' &&
    stripeSummaryResult.value.success
      ? normalizeStripeSubscriptionSummary(stripeSummaryResult.value.data)
      : null
  const legacyActive = getLegacyActiveSubscription(selfData)
  const lifecycle =
    offerResponse?.subscription ??
    stripeLifecycle ??
    selfData?.subscription ??
    selfData?.current_subscription
  const lifecycleStatus =
    lifecycle?.stripe_status?.trim() || lifecycle?.status?.trim() || undefined
  const normalizedStatus = lifecycleStatus?.toLowerCase()
  const lifecycleActive = Boolean(
    lifecycle &&
    normalizedStatus &&
    ['active', 'trialing', 'past_due', 'grace_period'].includes(
      normalizedStatus
    )
  )

  if (!offerResponse && !selfData && !stripeLifecycle) return null

  return {
    active: lifecycleActive || Boolean(legacyActive),
    available: Boolean(offerResponse?.active || offerResponse?.pending),
    pending: Boolean(offerResponse?.pending),
    tier: lifecycle?.price_tier,
    status: lifecycleStatus || legacyActive?.status,
    model: lifecycle?.model || offerResponse?.model,
    currentPrice: offerResponse?.current_price_display,
    currentPeriodEnd: lifecycle?.current_period_end || legacyActive?.end_time,
    gracePeriodEnd: lifecycle?.grace_period_end,
    cancelAtPeriodEnd: lifecycle?.cancel_at_period_end,
    legacyPlanId: legacyActive?.plan_id,
    fairUse: offerResponse?.fair_use
      ? {
          successRequestsPerWindow:
            offerResponse.fair_use.success_requests_per_window,
          totalRequestsPerWindow:
            offerResponse.fair_use.total_requests_per_window,
          windowMinutes: offerResponse.fair_use.window_minutes,
        }
      : undefined,
  }
}

function getSubscriptionPlanLabel(
  snapshot: DashboardSubscriptionSnapshot,
  t: TFunction
): string | undefined {
  if (snapshot.tier === 'founder') return t('Founder access')
  if (snapshot.tier === 'standard') return t('Standard access')
  if (snapshot.legacyPlanId) {
    return t('Subscription #{{id}}', { id: snapshot.legacyPlanId })
  }
  return undefined
}

function getRecentRequestRange(): {
  start_timestamp: number
  end_timestamp: number
} {
  return computeTimeRange(1)
}

export function OverviewDashboard() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const { items: apiInfoItems } = useApiInfo()
  const {
    apiInfo: showApiInfoPanel,
    announcements: showAnnouncementsPanel,
    uptimeKuma: showUptimePanel,
  } = useDashboardContentVisibility()
  const isAdmin = Boolean(user?.role && user.role >= ROLE.ADMIN)

  const apiKeysQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'api-keys'],
    queryFn: async () => {
      const result = await getApiKeys({ p: 1, size: 10 })
      return result.success ? (result.data?.items ?? []) : []
    },
    staleTime: 60 * 1000,
    retry: false,
  })

  const modelsQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'user-models'],
    queryFn: async () => {
      const result = await getUserModels()
      return result.success ? (result.data ?? []) : []
    },
    staleTime: 5 * 60 * 1000,
    retry: false,
  })

  const successfulRequestQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'successful-request', user?.id],
    queryFn: async () => {
      const result = await getUserSuccessfulRequestStatus()
      return result.success && result.data?.has_successful_request === true
    },
    enabled: Boolean(user?.id),
    staleTime: 30 * 1000,
    retry: false,
  })

  const recentRequestRange = useMemo(getRecentRequestRange, [])
  const recentRequestsQuery = useQuery({
    queryKey: [
      'dashboard',
      'overview',
      'recent-requests',
      recentRequestRange.start_timestamp,
      recentRequestRange.end_timestamp,
    ],
    queryFn: async () => {
      const result = await getUserQuotaDates({
        ...recentRequestRange,
        default_time: 'hour',
      })
      return result.success ? (result.data ?? []) : []
    },
    staleTime: 60 * 1000,
    retry: false,
  })

  const subscriptionQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'subscription', user?.id],
    queryFn: getDashboardSubscriptionSnapshot,
    enabled: Boolean(user?.id),
    staleTime: 60 * 1000,
    retry: false,
  })

  const keys = apiKeysQuery.data ?? []
  const primaryKey = getPreferredKey(keys)
  const activeKey = getActiveKey(keys)
  const availableModels = modelsQuery.data ?? []
  const model =
    availableModels[0] || 'gpt-4o-mini'
  const modelAvailable = availableModels.length > 0
  const confirmedSuccessfulRequest =
    Boolean(user) && hasConfirmedSuccessfulRequest(successfulRequestQuery.data)
  const balance = getBalanceBreakdown(
    user?.quota,
    user?.cash_quota,
    user?.promo_quota
  )
  const endpoint = normalizeEndpoint(apiInfoItems[0]?.url)
  const subscription = useMemo<ApertureSubscriptionSummary | null>(() => {
    const snapshot = subscriptionQuery.data
    if (!snapshot) return null
    return {
      ...snapshot,
      planLabel: getSubscriptionPlanLabel(snapshot, t),
    }
  }, [subscriptionQuery.data, t])
  const displayName = user?.display_name || user?.username || t('Developer')
  const showLeftContentPanels =
    isAdmin || showApiInfoPanel || showAnnouncementsPanel
  const showContentPanels = showLeftContentPanels || showUptimePanel
  const onboarding = (
    <ApertureOnboarding
      user={user}
      primaryKey={primaryKey}
      activeKey={activeKey}
      model={model}
      modelAvailable={modelAvailable}
      endpoint={endpoint}
    />
  )
  const liveCockpit = user ? (
    <ApertureCockpit
      user={user}
      primaryKey={primaryKey}
      activeKey={activeKey}
      model={model}
      modelAvailable={modelAvailable}
      endpoint={endpoint}
      balance={balance}
      subscription={subscription}
      recentRequests={recentRequestsQuery.data ?? []}
      recentRequestsLoading={recentRequestsQuery.isLoading}
      successRate={null}
      isAdmin={isAdmin}
    />
  ) : null
  const apertureExperience = confirmedSuccessfulRequest
    ? liveCockpit
    : onboarding

  return (
    <div className='flex flex-col gap-5'>
      <div className='flex flex-wrap items-end justify-between gap-4 px-1'>
        <div>
          <p className='editorial-kicker'>
            {t('N Aperture')} / {t('Gateway workspace')}
          </p>
          <h1 className='mt-2 text-2xl font-semibold tracking-[-0.03em] sm:text-3xl'>
            {t('Good to see you, {{name}}', { name: displayName })}
          </h1>
          <p className='text-muted-foreground mt-1.5 text-sm'>
            {confirmedSuccessfulRequest
              ? t(
                  'Live routing, account health, and your next useful action in one place.'
                )
              : t('Set up one route, then keep the live signal close.')}
          </p>
        </div>
        <span className='bg-card border-border inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-xs font-semibold'>
          <span
            className={cn(
              'size-1.5 rounded-full',
              confirmedSuccessfulRequest ? 'bg-success' : 'bg-primary'
            )}
            aria-hidden='true'
          />
          {confirmedSuccessfulRequest ? t('Live cockpit') : t('First route')}
        </span>
      </div>

      {apertureExperience}

      <CardStaggerContainer>
        <CardStaggerItem className='border-border bg-card overflow-hidden rounded-lg border'>
          <div className='flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between sm:p-5'>
            <div className='flex min-w-0 items-start gap-3'>
              <span className='bg-primary/10 text-primary flex size-10 shrink-0 items-center justify-center rounded-lg'>
                <Gift className='size-5' aria-hidden='true' />
              </span>
              <div className='min-w-0'>
                <h2 className='text-sm font-semibold sm:text-base'>
                  {t('Refer friends. Earn API credits.')}
                </h2>
                <p className='text-muted-foreground mt-1 max-w-2xl text-xs leading-relaxed sm:text-sm'>
                  {t(
                    'Invite a friend with your personal link. Once they verify their email, create a token, and make their first billable request, you both receive ¥50 in API credits.'
                  )}
                </p>
              </div>
            </div>
            <Button
              variant='outline'
              className='shrink-0'
              render={<Link to='/wallet' />}
            >
              {t('Open referral program')}
              <ArrowRight data-icon='inline-end' />
            </Button>
          </div>
        </CardStaggerItem>
      </CardStaggerContainer>

      <SummaryCards />

      {showContentPanels && (
        <CardStaggerContainer
          className={cn(
            'grid grid-cols-1 gap-4',
            showLeftContentPanels &&
              showUptimePanel &&
              'xl:grid-cols-[minmax(0,1fr)_22rem]'
          )}
        >
          {showLeftContentPanels && (
            <div
              className={cn(
                'grid min-w-0 grid-cols-1 gap-4',
                (showApiInfoPanel || showAnnouncementsPanel) && 'lg:grid-cols-2'
              )}
            >
              {isAdmin && (
                <CardStaggerItem className='lg:col-span-2'>
                  <PerformanceHealthPanel />
                </CardStaggerItem>
              )}
              {showApiInfoPanel && (
                <CardStaggerItem>
                  <ApiInfoPanel />
                </CardStaggerItem>
              )}
              {showAnnouncementsPanel && (
                <CardStaggerItem>
                  <AnnouncementsPanel />
                </CardStaggerItem>
              )}
            </div>
          )}
          {showUptimePanel && (
            <CardStaggerItem>
              <UptimePanel />
            </CardStaggerItem>
          )}
        </CardStaggerContainer>
      )}
    </div>
  )
}
