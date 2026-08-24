import { Link } from '@tanstack/react-router'
import {
  Activity,
  ArrowRight,
  CircleAlert,
  Clock3,
  CreditCard,
  ExternalLink,
  Gauge,
  History,
  KeyRound,
  RadioTower,
  ShieldCheck,
  Sparkles,
  WalletCards,
} from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  CardStaggerContainer,
  CardStaggerItem,
} from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import type { QuotaDataItem } from '@/features/dashboard/types'
import type { ApiKey } from '@/features/keys/types'
import { formatNumber, formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { AuthUser } from '@/stores/auth-store'

import { formatLifecycleDate } from './aperture-cockpit-helpers'
import { ApertureRequestCard } from './aperture-request-card'
import type { BalanceBreakdown } from './balance-utils'

export interface ApertureSubscriptionSummary {
  active: boolean
  available?: boolean
  pending?: boolean
  planLabel?: string
  status?: string
  model?: string
  currentPrice?: string
  currentPeriodEnd?: number | string | null
  gracePeriodEnd?: number | string | null
  cancelAtPeriodEnd?: boolean
  fairUse?: {
    successRequestsPerWindow?: number
    totalRequestsPerWindow?: number
    windowMinutes?: number
  }
}

export interface ApertureCockpitProps {
  user: AuthUser
  primaryKey: ApiKey | null
  activeKey: ApiKey | null
  model: string
  modelAvailable: boolean
  endpoint: string
  balance: BalanceBreakdown
  subscription: ApertureSubscriptionSummary | null
  recentRequests: QuotaDataItem[]
  recentRequestsLoading: boolean
  successRate: number | null
  isAdmin: boolean
}

type RoutingState = {
  label: string
  detail: string
  tone: IconBadgeTone
  icon: typeof RadioTower
}

function formatMaskedKey(key?: string): string {
  if (!key) return 'sk-...'
  const normalized = key.startsWith('sk-') ? key : `sk-${key}`
  if (normalized.includes('...')) return normalized
  if (normalized.length <= 14) return `${normalized.slice(0, 5)}...`
  return `${normalized.slice(0, 7)}...${normalized.slice(-4)}`
}

function formatRequestDate(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '—'
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(value < 10_000_000_000 ? value * 1000 : value)
}

function getRoutingState(
  props: ApertureCockpitProps,
  t: (key: string) => string
): RoutingState {
  if (!props.activeKey) {
    return {
      label: t('Needs API key'),
      detail: t('Create or enable a primary key to keep routing live.'),
      tone: 'warning',
      icon: KeyRound,
    }
  }
  if (!props.modelAvailable) {
    return {
      label: t('No model available'),
      detail: t('Choose an available model before sending more traffic.'),
      tone: 'warning',
      icon: RadioTower,
    }
  }
  if (props.balance.total <= 0 && !props.subscription?.active) {
    return {
      label: t('Balance depleted'),
      detail: t('Add balance or activate a subscription to keep routing live.'),
      tone: 'warning',
      icon: WalletCards,
    }
  }
  if (
    props.successRate !== null &&
    Number.isFinite(props.successRate) &&
    props.successRate < 70
  ) {
    return {
      label: t('Needs attention'),
      detail: t('Recent performance metrics show a lower success rate.'),
      tone: 'warning',
      icon: CircleAlert,
    }
  }
  return {
    label: t('Live routing'),
    detail: t(
      'The gateway has confirmed successful traffic and is ready for the next request.'
    ),
    tone: 'success',
    icon: RadioTower,
  }
}

function BalancePanel(props: { balance: BalanceBreakdown }) {
  const { t } = useTranslation()

  return (
    <section className='bg-muted/20 border-border rounded-md border p-4'>
      <div className='flex items-center justify-between gap-3'>
        <div className='flex items-center gap-2'>
          <IconBadge tone='primary' size='sm'>
            <WalletCards />
          </IconBadge>
          <h3 className='text-sm font-semibold'>{t('Account balance')}</h3>
        </div>
        <Button
          variant='ghost'
          size='sm'
          className='h-7 px-2 text-xs'
          render={<Link to='/wallet' />}
        >
          {t('Open wallet')}
          <ArrowRight data-icon='inline-end' className='size-3.5' />
        </Button>
      </div>
      <div className='mt-4 font-mono text-2xl font-semibold tracking-tight tabular-nums'>
        {formatQuota(props.balance.total)}
      </div>
      <div className='text-muted-foreground mt-1 text-xs'>
        {t('Available balance')}
      </div>
      <dl className='mt-4 grid grid-cols-2 gap-2'>
        <div className='bg-card border-border rounded-md border px-2.5 py-2'>
          <dt className='text-muted-foreground text-[11px]'>
            {t('Cash balance')}
          </dt>
          <dd className='mt-1 font-mono text-xs font-semibold tabular-nums'>
            {formatQuota(props.balance.cash)}
          </dd>
        </div>
        <div className='bg-card border-border rounded-md border px-2.5 py-2'>
          <dt className='text-muted-foreground text-[11px]'>
            {t('Promotional balance')}
          </dt>
          <dd className='text-primary mt-1 font-mono text-xs font-semibold tabular-nums'>
            {formatQuota(props.balance.promo)}
          </dd>
        </div>
      </dl>
    </section>
  )
}

function SubscriptionPanel(props: {
  subscription: ApertureSubscriptionSummary | null
}) {
  const { t } = useTranslation()
  const subscription = props.subscription
  const periodEnd = formatLifecycleDate(subscription?.currentPeriodEnd)
  const gracePeriodEnd = formatLifecycleDate(subscription?.gracePeriodEnd)
  let subscriptionLabel = t('No active subscription')
  if (subscription?.active) {
    subscriptionLabel = subscription.planLabel || t('Active subscription')
  } else if (subscription?.pending) {
    subscriptionLabel = t('Reservation pending')
  } else if (subscription?.available) {
    subscriptionLabel = t('Subscription available')
  }

  return (
    <section className='bg-muted/20 border-border rounded-md border p-4'>
      <div className='flex items-center justify-between gap-3'>
        <div className='flex items-center gap-2'>
          <IconBadge tone='warning' size='sm'>
            <Sparkles />
          </IconBadge>
          <h3 className='text-sm font-semibold'>{t('Subscription')}</h3>
        </div>
        <Button
          variant='ghost'
          size='sm'
          className='h-7 px-2 text-xs'
          render={<Link to='/wallet' />}
        >
          {subscription?.active
            ? t('Manage subscription')
            : t('Explore subscriptions')}
          <ExternalLink data-icon='inline-end' className='size-3.5' />
        </Button>
      </div>

      <div className='mt-4 flex items-center gap-2'>
        <span
          className={cn(
            'size-2 rounded-full',
            subscription?.active ? 'bg-success' : 'bg-muted-foreground/50'
          )}
          aria-hidden='true'
        />
        <span className='text-sm font-semibold'>{subscriptionLabel}</span>
      </div>
      <div className='text-muted-foreground mt-1 text-xs'>
        {subscription?.model ||
          t('Use wallet balance for pay-as-you-go routing.')}
      </div>

      {(subscription?.status ||
        periodEnd ||
        gracePeriodEnd ||
        subscription?.currentPrice) && (
        <dl className='border-border mt-4 space-y-2 border-t pt-3 text-xs'>
          {subscription?.status && (
            <div className='flex items-center justify-between gap-3'>
              <dt className='text-muted-foreground'>{t('Status')}</dt>
              <dd className='font-mono'>{subscription.status}</dd>
            </div>
          )}
          {subscription?.currentPrice && (
            <div className='flex items-center justify-between gap-3'>
              <dt className='text-muted-foreground'>{t('Current price')}</dt>
              <dd className='font-mono'>{subscription.currentPrice}</dd>
            </div>
          )}
          {periodEnd && (
            <div className='flex items-center justify-between gap-3'>
              <dt className='text-muted-foreground'>
                {t('Current period ends')}
              </dt>
              <dd>{periodEnd}</dd>
            </div>
          )}
          {subscription?.cancelAtPeriodEnd && (
            <div className='text-warning'>{t('Cancels at period end')}</div>
          )}
          {gracePeriodEnd && (
            <div className='text-warning'>
              {t('Payment grace period ends {{date}}', {
                date: gracePeriodEnd,
              })}
            </div>
          )}
        </dl>
      )}

      {subscription?.fairUse && (
        <div className='text-muted-foreground mt-4 flex items-start gap-2 border-t pt-3 text-xs leading-relaxed'>
          <Gauge
            className='text-primary mt-0.5 size-3.5 shrink-0'
            aria-hidden='true'
          />
          <span>
            {t(
              'Fair use: {{requests}} successful / {{total}} total requests per window.',
              {
                requests: subscription.fairUse.successRequestsPerWindow ?? '—',
                total: subscription.fairUse.totalRequestsPerWindow ?? '—',
              }
            )}
          </span>
        </div>
      )}
    </section>
  )
}

function RecentRequestsPanel(props: {
  requests: QuotaDataItem[]
  loading: boolean
}) {
  const { t } = useTranslation()
  const requests = useMemo(
    () =>
      [...props.requests]
        .filter((item) => Number(item.count) > 0 || Boolean(item.model_name))
        .sort((a, b) => Number(b.created_at) - Number(a.created_at))
        .slice(0, 5),
    [props.requests]
  )

  return (
    <section className='border-border bg-card overflow-hidden rounded-lg border'>
      <div className='border-border flex items-center justify-between gap-3 border-b p-4 sm:p-5'>
        <div className='flex items-center gap-2'>
          <IconBadge tone='info' size='sm'>
            <History />
          </IconBadge>
          <div>
            <h3 className='text-sm font-semibold'>{t('Recent requests')}</h3>
            <p className='text-muted-foreground mt-0.5 text-xs'>
              {t('Last 24 hours')}
            </p>
          </div>
        </div>
        <Button
          variant='ghost'
          size='sm'
          className='h-7 px-2 text-xs'
          render={<Link to='/usage-logs' />}
        >
          {t('View logs')}
          <ArrowRight data-icon='inline-end' className='size-3.5' />
        </Button>
      </div>

      <div className='divide-border/60 divide-y'>
        {props.loading && (
          <div className='text-muted-foreground flex items-center gap-2 p-4 text-xs'>
            <Clock3 className='size-3.5' aria-hidden='true' />
            {t('Loading')}
          </div>
        )}
        {!props.loading && requests.length === 0 && (
          <div className='text-muted-foreground p-4 text-xs leading-relaxed'>
            {t('No recent requests yet')}
          </div>
        )}
        {!props.loading &&
          requests.map((request) => (
            <div
              key={
                request.id ??
                `${request.created_at}-${request.model_name || 'request'}-${request.count || 0}-${request.quota || 0}`
              }
              className='flex items-center justify-between gap-3 px-4 py-3 sm:px-5'
            >
              <div className='flex min-w-0 items-center gap-2.5'>
                <span className='bg-success/10 text-success flex size-7 shrink-0 items-center justify-center rounded-md'>
                  <Activity className='size-3.5' aria-hidden='true' />
                </span>
                <div className='min-w-0'>
                  <p className='truncate font-mono text-xs font-semibold'>
                    {request.model_name || t('Unknown model')}
                  </p>
                  <p className='text-muted-foreground mt-0.5 flex items-center gap-1 text-[11px]'>
                    <span>{formatRequestDate(Number(request.created_at))}</span>
                    <span aria-hidden='true'>·</span>
                    <span>
                      {formatNumber(Number(request.count) || 0)} {t('requests')}
                    </span>
                  </p>
                </div>
              </div>
              <span className='text-muted-foreground shrink-0 font-mono text-[11px] tabular-nums'>
                {formatQuota(Number(request.quota) || 0)}
              </span>
            </div>
          ))}
      </div>
    </section>
  )
}

function QuickActions(props: { isAdmin: boolean }) {
  const { t } = useTranslation()
  const actions = [
    { label: t('Open Playground'), to: '/playground', icon: RadioTower },
    { label: t('Manage keys'), to: '/keys', icon: KeyRound },
    { label: t('Open wallet'), to: '/wallet', icon: CreditCard },
    { label: t('View logs'), to: '/usage-logs', icon: History },
    ...(props.isAdmin
      ? [{ label: t('Channels'), to: '/channels', icon: ShieldCheck }]
      : []),
  ] as const

  return (
    <nav
      aria-label={t('Quick actions')}
      className='border-border border-t pt-4'
    >
      <div className='mb-2 flex items-center gap-2'>
        <IconBadge tone='neutral' size='xs'>
          <Sparkles />
        </IconBadge>
        <span className='text-muted-foreground text-xs font-semibold'>
          {t('Quick actions')}
        </span>
      </div>
      <div className='flex flex-wrap gap-2'>
        {actions.map((action) => {
          const Icon = action.icon
          return (
            <Button
              key={action.label}
              variant='outline'
              size='sm'
              className='h-8 gap-1.5 px-2.5'
              render={<Link to={action.to} />}
            >
              <Icon data-icon='inline-start' />
              {action.label}
            </Button>
          )
        })}
      </div>
    </nav>
  )
}

export function ApertureCockpit(props: ApertureCockpitProps) {
  const { t } = useTranslation()
  const routingState = getRoutingState(props, t)
  const RoutingIcon = routingState.icon
  const maskedKey = formatMaskedKey(props.primaryKey?.key)
  const accountLabel = props.user.display_name || props.user.username
  const successRateLabel =
    props.successRate !== null && Number.isFinite(props.successRate)
      ? `${props.successRate.toFixed(2)}%`
      : '—'

  return (
    <CardStaggerContainer>
      <CardStaggerItem>
        <section
          aria-labelledby='aperture-cockpit-title'
          className='border-border bg-card overflow-hidden rounded-lg border'
        >
          <div className='border-border border-b p-4 sm:p-6'>
            <div className='flex flex-wrap items-start justify-between gap-4'>
              <div className='flex min-w-0 items-start gap-3'>
                <IconBadge tone='success' size='lg'>
                  <RadioTower />
                </IconBadge>
                <div className='min-w-0'>
                  <p className='editorial-kicker'>
                    {t('N Aperture')} / {t('Live cockpit')}
                  </p>
                  <h2
                    id='aperture-cockpit-title'
                    className='mt-2 text-2xl font-semibold tracking-[-0.035em] sm:text-3xl'
                  >
                    {t('Your gateway is in motion.')}
                  </h2>
                  <p className='text-muted-foreground mt-2 max-w-2xl text-sm leading-7'>
                    {t(
                      'A live view of the account, routes, balance, and request path you use most.'
                    )}
                  </p>
                </div>
              </div>

              <div className='border-border bg-muted/20 min-w-56 rounded-md border px-3 py-2.5'>
                <div className='text-muted-foreground flex items-center gap-2 text-[11px] font-semibold tracking-[0.12em] uppercase'>
                  <span
                    className={cn(
                      'size-1.5 rounded-full',
                      routingState.tone === 'success'
                        ? 'bg-success'
                        : 'bg-warning'
                    )}
                    aria-hidden='true'
                  />
                  {t('Routing status')}
                </div>
                <div className='mt-1.5 flex items-center gap-2'>
                  <RoutingIcon
                    className={cn(
                      'size-4',
                      routingState.tone === 'success'
                        ? 'text-success'
                        : 'text-warning'
                    )}
                    aria-hidden='true'
                  />
                  <span className='text-sm font-semibold'>
                    {routingState.label}
                  </span>
                </div>
                <p className='text-muted-foreground mt-1 text-xs leading-relaxed'>
                  {routingState.detail}
                </p>
              </div>
            </div>

            <div className='mt-5 grid gap-2 sm:grid-cols-3'>
              <div className='bg-muted/20 border-border flex items-center gap-2 rounded-md border px-3 py-2.5'>
                <IconBadge tone='info' size='xs'>
                  <ShieldCheck />
                </IconBadge>
                <div className='min-w-0'>
                  <div className='text-muted-foreground text-[11px]'>
                    {t('Account')}
                  </div>
                  <div className='truncate text-xs font-semibold'>
                    {accountLabel}
                  </div>
                </div>
              </div>
              <div className='bg-muted/20 border-border flex items-center gap-2 rounded-md border px-3 py-2.5'>
                <IconBadge tone='success' size='xs'>
                  <Activity />
                </IconBadge>
                <div className='min-w-0'>
                  <div className='text-muted-foreground text-[11px]'>
                    {t('Observed success')}
                  </div>
                  <div className='truncate text-xs font-semibold tabular-nums'>
                    {successRateLabel}
                  </div>
                </div>
              </div>
              <div className='bg-muted/20 border-border flex items-center gap-2 rounded-md border px-3 py-2.5'>
                <IconBadge tone='chart-4' size='xs'>
                  <Gauge />
                </IconBadge>
                <div className='min-w-0'>
                  <div className='text-muted-foreground text-[11px]'>
                    {t('Primary model')}
                  </div>
                  <div className='truncate font-mono text-xs font-semibold'>
                    {props.model}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className='grid gap-4 p-4 sm:p-6 xl:grid-cols-2'>
            <BalancePanel balance={props.balance} />
            <SubscriptionPanel subscription={props.subscription} />
          </div>

          <div className='grid gap-4 border-t p-4 sm:p-6 lg:grid-cols-[minmax(0,1.05fr)_minmax(18rem,0.95fr)]'>
            <ApertureRequestCard
              endpoint={props.endpoint}
              model={props.model}
              keyId={props.activeKey?.id}
              keyName={props.primaryKey?.name || t('Primary API key')}
              maskedKey={maskedKey}
              title={t('Primary API key')}
            />
            <RecentRequestsPanel
              requests={props.recentRequests}
              loading={props.recentRequestsLoading}
            />
          </div>

          <div className='px-4 pb-4 sm:px-6 sm:pb-6'>
            <QuickActions isAdmin={props.isAdmin} />
          </div>
        </section>
      </CardStaggerItem>
    </CardStaggerContainer>
  )
}
