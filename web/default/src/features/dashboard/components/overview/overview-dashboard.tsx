import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  Activity,
  ArrowRight,
  BookOpen,
  Check,
  ChevronDown,
  ChevronUp,
  Circle,
  Copy,
  CreditCard,
  FileText,
  KeyRound,
  ListChecks,
  RadioTower,
  ShieldCheck,
  TerminalSquare,
  Timer,
  type LucideIcon,
} from 'lucide-react'
import { motion, useReducedMotion } from 'motion/react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  CardStaggerContainer,
  CardStaggerItem,
} from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { fetchTokenKey, getApiKeys } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { getUserModels } from '@/lib/api'
import { MOTION_TRANSITION } from '@/lib/motion'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import {
  useApiInfo,
  useDashboardContentVisibility,
} from '../../hooks/use-status-data'
import { AnnouncementsPanel } from './announcements-panel'
import { ApiInfoPanel } from './api-info-panel'
import { FAQPanel } from './faq-panel'
import { PerformanceHealthPanel } from './performance-health-panel'
import { SummaryCards } from './summary-cards'
import { UptimePanel } from './uptime-panel'

const SETUP_GUIDE_VISIBILITY_STORAGE_KEY =
  'dashboard_overview_setup_guide_expanded'

type DashboardActionPath =
  | '/keys'
  | '/wallet'
  | '/playground'
  | '/channels'
  | '/usage-logs'
  | '/pricing'

interface StartStep {
  title: string
  description: string
  to: DashboardActionPath
  icon: LucideIcon
  completed: boolean
}

interface QuickAction {
  title: string
  description: string
  to: DashboardActionPath
  icon: LucideIcon
  adminOnly?: boolean
}

interface RequestExample {
  endpoint: string
  model: string
  keyName: string
  keyId?: number
  displayKey: string
  ready: boolean
}

interface HeroSignal {
  label: string
  value: string
  icon: LucideIcon
  tone: IconBadgeTone
}

function getSavedSetupGuideExpanded(): boolean | null {
  if (typeof window === 'undefined') return null
  const saved = window.localStorage.getItem(SETUP_GUIDE_VISIBILITY_STORAGE_KEY)
  if (saved === 'expanded') return true
  if (saved === 'collapsed') return false
  return null
}

function saveSetupGuideExpanded(expanded: boolean): void {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(
    SETUP_GUIDE_VISIBILITY_STORAGE_KEY,
    expanded ? 'expanded' : 'collapsed'
  )
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

function formatDisplayKey(key?: string): string {
  if (!key) return 'sk-...'
  if (key.length <= 14) return key
  return `${key.slice(0, 7)}...${key.slice(-4)}`
}

function buildCurlCommand(args: {
  endpoint: string
  apiKey: string
  model: string
}): string {
  return [
    `curl ${args.endpoint} \\`,
    '  -H "Content-Type: application/json" \\',
    `  -H "Authorization: Bearer ${args.apiKey}" \\`,
    `  -d '{"model":"${args.model}","messages":[{"role":"user","content":"Say hello in one sentence."}]}'`,
  ].join('\n')
}

function SetupGuideBackdrop(_props: { compact?: boolean }) {
  // Flat editorial surface — no gradient mesh or glass.
  return null
}

function StartStepItem(props: {
  step: StartStep
  index: number
  isLast: boolean
}) {
  const Icon = props.step.icon
  const StatusIcon = props.step.completed ? Check : Circle

  return (
    <li className='relative flex gap-3 pb-2.5 last:pb-0'>
      {!props.isLast && (
        <span
          className='bg-border absolute top-9 bottom-0 left-4 w-px'
          aria-hidden='true'
        />
      )}
      <span
        className={cn(
          'bg-background relative z-10 flex size-8 shrink-0 items-center justify-center rounded-md border',
          props.step.completed && 'border-success/30 bg-success/10'
        )}
      >
        <StatusIcon
          className={props.step.completed ? 'text-success size-4' : 'size-4'}
          aria-hidden='true'
        />
      </span>

      <Link
        to={props.step.to}
        className='bg-card hover:bg-muted/60 focus-visible:ring-ring flex min-w-0 flex-1 items-center justify-between gap-3 rounded-md border border-border px-3 py-2.5 text-left transition-colors outline-none focus-visible:ring-2'
      >
        <span className='flex min-w-0 items-start gap-2.5'>
          <span className='bg-muted mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md border border-border'>
            <Icon className='size-3.5' aria-hidden='true' />
          </span>
          <span className='flex min-w-0 flex-col gap-0.5'>
            <span className='flex items-center gap-2 text-sm font-medium'>
              <span className='text-muted-foreground font-mono text-xs tabular-nums'>
                {props.index + 1}.
              </span>
              <span className='truncate'>{props.step.title}</span>
            </span>
            <span className='text-muted-foreground line-clamp-1 text-xs'>
              {props.step.description}
            </span>
          </span>
        </span>
        <ArrowRight
          className='text-muted-foreground size-4 shrink-0'
          aria-hidden='true'
        />
      </Link>
    </li>
  )
}

function RequestPreview(props: {
  example: RequestExample
  signals: HeroSignal[]
}) {
  const { t } = useTranslation()
  const shouldReduceMotion = useReducedMotion()
  const [isCopying, setIsCopying] = useState(false)
  const { copyToClipboard } = useCopyToClipboard({ notify: false })
  const previewCurl = buildCurlCommand({
    endpoint: props.example.endpoint,
    apiKey: props.example.displayKey,
    model: props.example.model,
  })
  const previewLines = previewCurl.split('\n')
  const handleCopyRequest = async () => {
    if (!props.example.keyId || isCopying) return

    setIsCopying(true)
    try {
      const result = await fetchTokenKey(props.example.keyId)
      const key = result.success && result.data?.key ? result.data.key : ''
      if (!key) {
        toast.error(result.message || t('Failed to copy to clipboard'))
        return
      }

      const realCurl = buildCurlCommand({
        endpoint: props.example.endpoint,
        apiKey: `sk-${key}`,
        model: props.example.model,
      })
      const copied = await copyToClipboard(realCurl)
      if (copied) {
        toast.success(t('Copied to clipboard'))
      } else {
        toast.error(t('Failed to copy to clipboard'))
      }
    } finally {
      setIsCopying(false)
    }
  }

  return (
    <motion.div
      initial={shouldReduceMotion ? false : { opacity: 0 }}
      animate={shouldReduceMotion ? undefined : { opacity: 1 }}
      transition={MOTION_TRANSITION.slow}
      className='relative overflow-hidden rounded-lg border border-border bg-card p-3'
    >
      <div className='flex items-center justify-between gap-3 border-b border-border pb-3'>
        <div className='flex min-w-0 items-center gap-2'>
          <IconBadge tone='info'>
            <TerminalSquare />
          </IconBadge>
          <div className='min-w-0'>
            <div className='truncate text-sm font-medium'>
              {t('First API request')}
            </div>
            <div className='text-muted-foreground truncate text-xs'>
              {props.example.ready
                ? props.example.keyName
                : t('Create an API key to unlock the real request')}
            </div>
          </div>
        </div>
        {props.example.ready ? (
          <Button
            variant='outline'
            size='sm'
            className='h-7 gap-1.5 px-2 text-xs'
            disabled={isCopying}
            onClick={handleCopyRequest}
            aria-label={t('Copy ready-to-run curl')}
          >
            <Copy data-icon='inline-start' />
            {isCopying ? t('Loading') : t('Copy')}
          </Button>
        ) : (
          <Button size='sm' variant='outline' render={<Link to='/keys' />}>
            {t('Create API Key')}
          </Button>
        )}
      </div>

      <div className='bg-muted my-3 rounded-md border border-border p-3 font-mono text-xs'>
        <div className='mb-2 flex items-center gap-1.5'>
          <span className='bg-destructive size-2 rounded-full' />
          <span className='bg-warning size-2 rounded-full' />
          <span className='bg-success size-2 rounded-full' />
        </div>
        <div className='flex flex-col gap-1 overflow-hidden'>
          {previewLines.map((line) => (
            <code
              key={line}
              className='text-muted-foreground truncate'
              title={line}
            >
              {line}
            </code>
          ))}
        </div>
      </div>

      <div className='grid gap-2'>
        {props.signals.map((signal) => {
          const Icon = signal.icon

          return (
            <div
              key={signal.label}
              className='bg-muted/50 flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2'
            >
              <span className='flex min-w-0 items-center gap-2'>
                <IconBadge tone={signal.tone} size='xs'>
                  <Icon />
                </IconBadge>
                <span className='truncate text-xs font-medium'>
                  {signal.label}
                </span>
              </span>
              <span className='text-muted-foreground shrink-0 text-xs'>
                {signal.value}
              </span>
            </div>
          )
        })}
      </div>
    </motion.div>
  )
}

function QuickActionItem(props: { action: QuickAction }) {
  const Icon = props.action.icon

  return (
    <Button
      variant='outline'
      className='bg-card hover:bg-muted/60 h-auto justify-start rounded-md border border-border px-3 py-3 text-left'
      render={<Link to={props.action.to} />}
    >
      <span className='bg-muted flex size-9 shrink-0 items-center justify-center rounded-md border border-border'>
        <Icon className='size-4' aria-hidden='true' />
      </span>
      <span className='flex min-w-0 flex-1 flex-col gap-0.5'>
        <span className='truncate text-sm font-medium'>
          {props.action.title}
        </span>
        <span className='text-muted-foreground line-clamp-2 text-xs leading-relaxed'>
          {props.action.description}
        </span>
      </span>
    </Button>
  )
}

function CompactQuickAction(props: { action: QuickAction }) {
  const Icon = props.action.icon

  return (
    <Button
      variant='outline'
      size='sm'
      className='bg-background/70 h-8 min-w-24 gap-1.5 px-2.5'
      render={<Link to={props.action.to} />}
    >
      <Icon data-icon='inline-start' />
      <span>{props.action.title}</span>
    </Button>
  )
}

export function OverviewDashboard() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const { items: apiInfoItems } = useApiInfo()
  const {
    apiInfo: showApiInfoPanel,
    announcements: showAnnouncementsPanel,
    faq: showFAQPanel,
    uptimeKuma: showUptimePanel,
  } = useDashboardContentVisibility()
  const [manualSetupGuideExpanded, setManualSetupGuideExpanded] = useState<
    boolean | null
  >(() => getSavedSetupGuideExpanded())

  const requestCount = Number(user?.request_count ?? 0)
  const remainQuota = Number(user?.quota ?? 0)
  const usedQuota = Number(user?.used_quota ?? 0)
  const isAdmin = Boolean(user?.role && user.role >= ROLE.ADMIN)
  const displayName = user?.display_name || user?.username || t('Developer')

  const apiKeysQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'api-keys'],
    queryFn: async () => {
      const result = await getApiKeys({ p: 1, size: 10 })
      return result.success ? (result.data?.items ?? []) : []
    },
    staleTime: 60 * 1000,
  })

  const modelsQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'user-models'],
    queryFn: async () => {
      const result = await getUserModels()
      return result.success ? (result.data ?? []) : []
    },
    staleTime: 5 * 60 * 1000,
  })

  const preferredKey = useMemo(
    () => getPreferredKey(apiKeysQuery.data ?? []),
    [apiKeysQuery.data]
  )

  const startSteps = useMemo<StartStep[]>(
    () => [
      {
        title: t('Create API Key'),
        description: t('Create a key for your app or service'),
        to: '/keys',
        icon: KeyRound,
        completed: Boolean(preferredKey),
      },
      {
        title: t('Add credits'),
        description: t('Keep enough balance before production traffic'),
        to: '/wallet',
        icon: CreditCard,
        completed: remainQuota > 0 || usedQuota > 0,
      },
      {
        title: t('Send a request'),
        description: t('Verify routing with Playground or your client'),
        to: '/playground',
        icon: TerminalSquare,
        completed: requestCount > 0,
      },
    ],
    [preferredKey, remainQuota, requestCount, t, usedQuota]
  )

  const quickActions = useMemo<QuickAction[]>(
    () => [
      {
        title: t('API Keys'),
        description: t('Create a key for your app or service'),
        to: '/keys',
        icon: KeyRound,
      },
      {
        title: t('Channels'),
        description: t('Configure upstream providers and routing.'),
        to: '/channels',
        icon: RadioTower,
        adminOnly: true,
      },
      {
        title: t('Usage Logs'),
        description: t('Inspect requests, errors, and billing details'),
        to: '/usage-logs',
        icon: FileText,
      },
      {
        title: t('Pricing'),
        description: t('Review model rates before scaling traffic'),
        to: '/pricing',
        icon: BookOpen,
      },
    ],
    [t]
  )

  const visibleQuickActions = useMemo(
    () => quickActions.filter((action) => !action.adminOnly || isAdmin),
    [isAdmin, quickActions]
  )

  const heroSignals = useMemo<HeroSignal[]>(
    () => [
      {
        label: t('Route active'),
        value: apiInfoItems.length > 0 ? t('Online') : t('Current domain'),
        icon: RadioTower,
        tone: 'info',
      },
      {
        label: t('Auth configured'),
        value: preferredKey ? t('Secured') : t('Needs API key'),
        icon: ShieldCheck,
        tone: 'success',
      },
      {
        label: t('Model selected'),
        value: modelsQuery.data?.[0] ?? t('Loading'),
        icon: Timer,
        tone: 'chart-4',
      },
    ],
    [apiInfoItems.length, modelsQuery.data, preferredKey, t]
  )

  const requestExample = useMemo<RequestExample>(() => {
    const endpoint = normalizeEndpoint(apiInfoItems[0]?.url)
    const model = modelsQuery.data?.[0] ?? 'gpt-4o-mini'
    const keyName = preferredKey?.name ?? t('No API key yet')
    const ready = Boolean(preferredKey?.id && model)

    return {
      endpoint,
      model,
      keyName,
      keyId: preferredKey?.id,
      displayKey: preferredKey
        ? formatDisplayKey(`sk-${preferredKey.key}`)
        : 'sk-...',
      ready,
    }
  }, [apiInfoItems, modelsQuery.data, preferredKey, t])

  const completedStepCount = startSteps.filter((step) => step.completed).length
  const setupComplete = completedStepCount === startSteps.length
  const hasActiveKey = preferredKey?.status === 1
  const hasAvailableModel = (modelsQuery.data?.length ?? 0) > 0
  const routeReady = hasActiveKey && remainQuota > 0 && hasAvailableModel
  let routeReadinessLabel = t('Ready to route')
  if (!hasActiveKey) {
    routeReadinessLabel = t('Needs API key')
  } else if (remainQuota <= 0) {
    routeReadinessLabel = t('Balance depleted')
  } else if (!hasAvailableModel) {
    routeReadinessLabel = t('Setup in progress')
  }
  const setupStatusReady = apiKeysQuery.isFetched && Boolean(user)
  const setupGuideExpanded =
    manualSetupGuideExpanded ?? (setupStatusReady && !setupComplete)
  const showLeftContentPanels =
    isAdmin || showApiInfoPanel || showAnnouncementsPanel || showFAQPanel
  const showContentPanels = showLeftContentPanels || showUptimePanel

  const handleSetupGuideToggle = () => {
    const nextExpanded = !setupGuideExpanded
    setManualSetupGuideExpanded(nextExpanded)
    saveSetupGuideExpanded(nextExpanded)
  }

  return (
    <div className='flex flex-col gap-5'>
      <div className='flex flex-wrap items-end justify-between gap-4 px-1'>
        <div>
          <p className='editorial-kicker'>{t('Gateway workspace')}</p>
          <h2 className='mt-2 text-2xl font-semibold tracking-[-0.03em] sm:text-3xl'>
            {t('Good to see you, {{name}}', { name: displayName })}
          </h2>
          <p className='text-muted-foreground mt-1.5 text-sm'>
            {t('Balance, readiness, and the next useful action in one place.')}
          </p>
        </div>
        <span className='bg-card border-border inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-xs font-semibold'>
          <Activity
            className={cn(
              'size-3.5',
              routeReady ? 'text-success' : 'text-primary'
            )}
            aria-hidden='true'
          />
          {routeReadinessLabel}
        </span>
      </div>

      {setupGuideExpanded ? (
        <CardStaggerContainer className='grid items-stretch gap-4 xl:grid-cols-[minmax(0,1fr)_23rem]'>
          <CardStaggerItem className='h-full overflow-hidden rounded-lg border border-border bg-card'>
            <div className='relative h-full overflow-hidden p-4 sm:p-6'>
              <SetupGuideBackdrop />
              <div className='relative grid gap-5 lg:grid-cols-[minmax(0,1fr)_21rem]'>
                <div className='flex min-w-0 flex-col gap-5'>
                  <div className='flex flex-wrap items-start justify-between gap-3'>
                    <div className='flex max-w-2xl flex-col gap-1'>
                      <div className='editorial-kicker flex items-center gap-2'>
                        <ListChecks className='size-3.5' aria-hidden='true' />
                        {t('Get started')}
                      </div>
                      <h3 className='text-xl font-semibold tracking-tight sm:text-2xl'>
                        {t('Prepare your first routed request')}
                      </h3>
                      <p className='text-muted-foreground max-w-xl text-sm leading-relaxed'>
                        {t(
                          'Create a key, confirm your balance, and send a request with the endpoint already prepared for you.'
                        )}
                      </p>
                    </div>
                    <div className='flex flex-wrap items-center gap-2'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={handleSetupGuideToggle}
                      >
                        <ChevronUp data-icon='inline-start' />
                        {t('Hide setup guide')}
                      </Button>
                      <Button size='sm' render={<Link to='/keys' />}>
                        <KeyRound data-icon='inline-start' />
                        {t('Create API Key')}
                      </Button>
                    </div>
                  </div>

                  <ol className='bg-muted/40 rounded-md border border-border p-2'>
                    {startSteps.map((step, index) => (
                      <StartStepItem
                        key={step.title}
                        step={step}
                        index={index}
                        isLast={index === startSteps.length - 1}
                      />
                    ))}
                  </ol>
                </div>

                <RequestPreview
                  example={requestExample}
                  signals={heroSignals}
                />
              </div>
            </div>
          </CardStaggerItem>

          <CardStaggerItem className='h-full rounded-lg border border-border bg-card p-4 sm:p-5'>
            <div className='flex h-full flex-col gap-4'>
              <div className='flex flex-col gap-1'>
                <div className='editorial-kicker'>{t('Recommended actions')}</div>
                <h3 className='text-lg font-semibold tracking-tight'>
                  {t('Keep the platform ready')}
                </h3>
              </div>
              <div className='grid gap-2'>
                {visibleQuickActions.map((action) => (
                  <QuickActionItem key={action.title} action={action} />
                ))}
              </div>
            </div>
          </CardStaggerItem>
        </CardStaggerContainer>
      ) : (
        <CardStaggerContainer>
          <CardStaggerItem className='overflow-hidden rounded-lg border border-border bg-card'>
            <div className='relative overflow-hidden px-4 py-3 sm:px-5'>
              <SetupGuideBackdrop compact />
              <div className='relative flex flex-wrap items-center justify-between gap-3'>
                <div className='flex min-w-0 items-center gap-3'>
                  <span className='bg-muted flex size-9 shrink-0 items-center justify-center rounded-md border border-border'>
                    <Check className='text-success size-4' aria-hidden='true' />
                  </span>
                  <div className='min-w-0'>
                    <div className='flex items-center gap-2'>
                      <h3 className='truncate text-sm font-semibold'>
                        {setupComplete
                          ? t('Setup guide complete')
                          : t('Setup guide')}
                      </h3>
                      <span className='text-muted-foreground bg-background/60 rounded-md border px-2 py-0.5 text-xs'>
                        {t('Setup progress: {{completed}}/{{total}}', {
                          completed: completedStepCount,
                          total: startSteps.length,
                        })}
                      </span>
                    </div>
                    <p className='text-muted-foreground line-clamp-1 text-xs'>
                      {setupComplete
                        ? t(
                            'Your setup guide is collapsed so usage stays in focus.'
                          )
                        : t('Setup guide is collapsed. Expand it anytime.')}
                    </p>
                  </div>
                </div>

                <div className='flex flex-wrap items-center gap-2'>
                  {visibleQuickActions.map((action) => (
                    <CompactQuickAction key={action.title} action={action} />
                  ))}
                  <Button
                    variant='outline'
                    size='sm'
                    className='bg-background/70 h-8 min-w-28'
                    onClick={handleSetupGuideToggle}
                  >
                    <ChevronDown data-icon='inline-start' />
                    {t('Show setup guide')}
                  </Button>
                </div>
              </div>
            </div>
          </CardStaggerItem>
        </CardStaggerContainer>
      )}

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
                (showApiInfoPanel || showAnnouncementsPanel || showFAQPanel) &&
                  'lg:grid-cols-2'
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
              {showFAQPanel && (
                <CardStaggerItem>
                  <FAQPanel />
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
