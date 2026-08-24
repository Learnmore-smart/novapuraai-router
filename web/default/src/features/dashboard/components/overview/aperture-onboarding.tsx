import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  Check,
  Circle,
  KeyRound,
  UserRound,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  CardStaggerContainer,
  CardStaggerItem,
} from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import type { ApiKey } from '@/features/keys/types'
import type { AuthUser } from '@/stores/auth-store'

import { ApertureRequestCard } from './aperture-request-card'

type OnboardingPath = '/profile' | '/keys' | '/playground'

interface OnboardingStep {
  title: string
  description: string
  detail: string
  actionLabel: string
  to: OnboardingPath
  icon: LucideIcon
  complete: boolean
}

export interface ApertureOnboardingProps {
  user: AuthUser | null
  primaryKey: ApiKey | null
  activeKey: ApiKey | null
  model: string
  modelAvailable: boolean
  endpoint: string
}

function formatMaskedKey(key?: string): string {
  if (!key) return 'sk-...'
  const normalized = key.startsWith('sk-') ? key : `sk-${key}`
  if (normalized.includes('...')) return normalized
  if (normalized.length <= 14) return `${normalized.slice(0, 5)}...`
  return `${normalized.slice(0, 7)}...${normalized.slice(-4)}`
}

function OnboardingStepItem(props: { step: OnboardingStep; index: number }) {
  const Icon = props.step.icon
  const StatusIcon = props.step.complete ? Check : Circle

  return (
    <li className='relative flex gap-3 pb-4 last:pb-0'>
      {props.index < 2 && (
        <span
          className='bg-border absolute top-9 bottom-0 left-4 w-px'
          aria-hidden='true'
        />
      )}
      <span
        className={
          props.step.complete
            ? 'border-success/30 bg-success/10 relative z-10 flex size-8 shrink-0 items-center justify-center rounded-md border'
            : 'bg-background relative z-10 flex size-8 shrink-0 items-center justify-center rounded-md border'
        }
      >
        <StatusIcon
          className={props.step.complete ? 'text-success size-4' : 'size-4'}
          aria-hidden='true'
        />
      </span>

      <div className='bg-card border-border min-w-0 flex-1 rounded-md border p-3'>
        <div className='flex flex-wrap items-start justify-between gap-2'>
          <div className='flex min-w-0 items-start gap-2.5'>
            <IconBadge
              tone={props.step.complete ? 'success' : 'neutral'}
              size='xs'
            >
              <Icon />
            </IconBadge>
            <div className='min-w-0'>
              <div className='flex items-center gap-2'>
                <span className='text-muted-foreground font-mono text-[11px] tabular-nums'>
                  0{props.index + 1}
                </span>
                <h3 className='truncate text-sm font-semibold'>
                  {props.step.title}
                </h3>
              </div>
              <p className='text-muted-foreground mt-1 text-xs leading-relaxed'>
                {props.step.description}
              </p>
            </div>
          </div>
          <Button
            variant='ghost'
            size='sm'
            className='h-7 shrink-0 px-2 text-xs'
            render={<Link to={props.step.to} />}
          >
            {props.step.actionLabel}
            <ArrowRight data-icon='inline-end' className='size-3.5' />
          </Button>
        </div>
        <p className='text-muted-foreground/80 mt-2 border-t pt-2 font-mono text-[11px] leading-relaxed'>
          {props.step.detail}
        </p>
      </div>
    </li>
  )
}

export function ApertureOnboarding(props: ApertureOnboardingProps) {
  const { t } = useTranslation()
  const maskedKey = formatMaskedKey(props.primaryKey?.key)
  const accountDetail =
    props.user?.email || props.user?.username || t('Account ready')
  let keyDetail = t('No API key yet')
  if (props.activeKey) {
    keyDetail = `${props.activeKey.name} · ${formatMaskedKey(props.activeKey.key)}`
  } else if (props.primaryKey) {
    keyDetail = t('A key exists but needs enabling')
  }
  const modelDetail = props.modelAvailable
    ? t('Ready for {{model}}', { model: props.model })
    : t('Choose an available model in Playground')
  const steps: OnboardingStep[] = [
    {
      title: t('Account'),
      description: t('Your N Aperture workspace is ready.'),
      detail: accountDetail,
      actionLabel: t('Review account'),
      to: '/profile',
      icon: UserRound,
      complete: Boolean(props.user),
    },
    {
      title: t('API Key'),
      description: t('Use a key to authenticate routed requests.'),
      detail: keyDetail,
      actionLabel: props.activeKey ? t('Manage keys') : t('Create API Key'),
      to: '/keys',
      icon: KeyRound,
      complete: Boolean(props.activeKey),
    },
    {
      title: t('First request'),
      description: t(
        'Confirm one successful response to unlock the live cockpit.'
      ),
      detail: modelDetail,
      actionLabel: t('Open Playground'),
      to: '/playground',
      icon: ArrowRight,
      complete: false,
    },
  ]

  return (
    <CardStaggerContainer>
      <CardStaggerItem>
        <section
          aria-labelledby='aperture-onboarding-title'
          className='border-border bg-card overflow-hidden rounded-lg border'
        >
          <div className='grid lg:grid-cols-[minmax(0,0.82fr)_minmax(0,1.18fr)]'>
            <div className='border-border flex flex-col gap-5 border-b p-4 sm:p-6 lg:border-r lg:border-b-0'>
              <div>
                <p className='editorial-kicker'>
                  {t('N Aperture')} / {t('First route')}
                </p>
                <h2
                  id='aperture-onboarding-title'
                  className='mt-2 max-w-lg text-2xl font-semibold tracking-[-0.035em] sm:text-3xl'
                >
                  {t('Bring your first route online.')}
                </h2>
                <p className='text-muted-foreground mt-3 max-w-xl text-sm leading-7'>
                  {t(
                    'A clear path from account to API key to one confirmed request. Your live cockpit opens after performance metrics record a successful response.'
                  )}
                </p>
              </div>

              <div className='border-border mt-auto border-t pt-4'>
                <div className='text-muted-foreground flex items-center gap-2 text-xs'>
                  <span
                    className='bg-primary size-1.5 rounded-full'
                    aria-hidden='true'
                  />
                  {t('Setup is based on observed routing, not account totals.')}
                </div>
              </div>
            </div>

            <div className='bg-muted/20 flex flex-col gap-4 p-4 sm:p-6'>
              <ol aria-label={t('N Aperture setup steps')}>
                {steps.map((step, index) => (
                  <OnboardingStepItem
                    key={step.title}
                    step={step}
                    index={index}
                  />
                ))}
              </ol>
              <ApertureRequestCard
                endpoint={props.endpoint}
                model={props.model}
                keyId={props.activeKey?.id}
                keyName={props.activeKey?.name}
                maskedKey={maskedKey}
                title={t('Send your first request')}
              />
            </div>
          </div>
        </section>
      </CardStaggerItem>
    </CardStaggerContainer>
  )
}
