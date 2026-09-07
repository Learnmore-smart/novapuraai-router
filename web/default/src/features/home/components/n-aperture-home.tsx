import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowUpRight, Check, KeyRound, WalletCards } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import { getSubscriptionOffer } from '@/features/subscriptions/api'
import {
  formatOfferReservationExpiry,
  getOfferAvailabilityLabel,
  getOfferTierLabel,
  normalizeSubscriptionOffer,
} from '@/features/subscriptions/lib/subscription-offer'
import type { SubscriptionOffer } from '@/features/subscriptions/types'
import { useStatus } from '@/hooks/use-status'
import { formatPublicCurrency, getRegisterPromo } from '@/lib/public-status'
import { useAuthStore } from '@/stores/auth-store'

import { getHomeRouteModelNames } from '../lib/home-content'
import { NAperture } from './n-aperture'
import { FAQ } from './sections/faq'
import { ProviderStrip } from './sections/provider-strip'

interface NApertureHomeProps {
  isAuthenticated: boolean
}

const EVIDENCE = [
  {
    icon: KeyRound,
    title: 'OpenAI-compatible',
    description:
      'Keep the client, bearer token, and chat-completions route you already use.',
  },
  {
    icon: Check,
    title: 'Request-level ledger',
    description: 'See what routed, what completed, and what each request cost.',
  },
  {
    icon: WalletCards,
    title: 'Prepaid + subscriptions',
    description:
      'Choose pay-as-you-go capacity or a server-defined fair-use offer.',
  },
] as const

function formatOfferPrice(offer: SubscriptionOffer): string | null {
  const display = offer.current_price_display?.trim()
  if (display) return display
  if (
    typeof offer.current_price_minor !== 'number' ||
    !Number.isFinite(offer.current_price_minor) ||
    !offer.currency
  ) {
    return null
  }

  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: offer.currency,
      maximumFractionDigits: 2,
    }).format(offer.current_price_minor / 100)
  } catch {
    return null
  }
}

function formatOfferFuturePrice(offer: SubscriptionOffer): string | null {
  const display = offer.future_standard_price_display?.trim()
  if (display) return display
  if (
    typeof offer.future_standard_price_minor !== 'number' ||
    !Number.isFinite(offer.future_standard_price_minor) ||
    !offer.currency
  ) {
    return null
  }

  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: offer.currency,
      maximumFractionDigits: 2,
    }).format(offer.future_standard_price_minor / 100)
  } catch {
    return null
  }
}

function OfferStrip(props: {
  isAuthenticated: boolean
  offer: SubscriptionOffer | null
  offerLoading: boolean
  registerPromo: ReturnType<typeof getRegisterPromo>
}) {
  const { t } = useTranslation()
  const offer = props.offer
  const availabilityKey = props.offerLoading
    ? 'Loading'
    : getOfferAvailabilityLabel(offer ?? {})
  const availability = t(availabilityKey)
  const offerTitle = offer?.limit
    ? t('Founding {{limit}}', { limit: offer.limit })
    : t('Founding access')
  const currentPrice = offer ? formatOfferPrice(offer) : null
  const futurePrice = offer ? formatOfferFuturePrice(offer) : null
  const reservationExpiry = offer
    ? formatOfferReservationExpiry(offer.reservation_expires_at)
    : null

  return (
    <section
      className='np-aperture-offer np-aperture-container'
      aria-labelledby='np-aperture-offer-title'
      data-offer-state={availabilityKey.toLowerCase().replaceAll(' ', '-')}
    >
      <div className='np-aperture-offer-main'>
        <div>
          <p className='np-aperture-overline'>{offerTitle}</p>
          <h2 id='np-aperture-offer-title'>
            {t('Unlimited access, with a fair-use boundary.')}
          </h2>
          <p className='np-aperture-copy'>
            {t(
              'Unlimited tokens for the target model, subject to peak concurrency, fair use, and a strict no-resale policy.'
            )}
          </p>
        </div>

        <div className='np-aperture-offer-meta'>
          <div className='np-aperture-offer-status'>
            <span className='np-aperture-ink-dot' aria-hidden='true' />
            <span>{availability}</span>
          </div>
          {offer && (
            <p className='np-aperture-meta-note'>
              {t(getOfferTierLabel(offer.current_price_tier))}
            </p>
          )}
          <dl className='np-aperture-offer-stats'>
            <div>
              <dt>{t('Seats remaining')}</dt>
              <dd>
                {offer?.remaining ?? '—'} / {offer?.limit ?? '—'}
              </dd>
            </div>
            <div>
              <dt>{t('Current price')}</dt>
              <dd>{currentPrice || '—'}</dd>
            </div>
            {futurePrice && (
              <div>
                <dt>{t('Future standard seat')}</dt>
                <dd>{futurePrice}</dd>
              </div>
            )}
          </dl>
          {reservationExpiry && (
            <p className='np-aperture-meta-note'>
              {t('Reservation expires {{date}}', { date: reservationExpiry })}
            </p>
          )}
          <Link className='np-aperture-text-link' to='/pricing'>
            {t('See availability')}
            <ArrowUpRight aria-hidden='true' />
          </Link>
        </div>
      </div>

      {!props.isAuthenticated && props.registerPromo && (
        <p className='np-aperture-offer-promo'>
          {t('New accounts receive {{amount}} in API credits.', {
            amount:
              formatPublicCurrency(
                props.registerPromo.amount,
                props.registerPromo.currency
              ) || '—',
          })}
        </p>
      )}
    </section>
  )
}

export function NApertureHome(props: NApertureHomeProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const userId = useAuthStore((state) => state.auth.user?.id)
  const { models, isLoading: modelsLoading } = usePricingData()
  const offerQuery = useQuery({
    queryKey: ['subscription', 'offer', userId],
    queryFn: async () => {
      const response = await getSubscriptionOffer()
      if (!response.success) return null
      return normalizeSubscriptionOffer(response.data)
    },
    staleTime: 30_000,
  })
  const modelNames = useMemo(() => getHomeRouteModelNames(models), [models])
  const registerPromo = getRegisterPromo(status)

  return (
    <main className='np-aperture-page'>
      <section className='np-aperture-hero has-banner'>
        <div className='np-aperture-container np-aperture-hero-grid'>
          <div className='np-aperture-hero-copy'>
            <p className='np-aperture-overline'>
              {t('ONE ENDPOINT · LIVE ROUTING')}
            </p>
            <h1>{t('The cleanest path to every model.')}</h1>
            <p className='np-aperture-copy np-aperture-hero-description'>
              {t(
                'Connect once. NovaPura routes every request through the best available model path — and shows you exactly what happened.'
              )}
            </p>
            <div className='np-aperture-actions'>
              {props.isAuthenticated ? (
                <Link className='np-aperture-button' to='/dashboard'>
                  {t('Open dashboard')}
                  <ArrowUpRight aria-hidden='true' />
                </Link>
              ) : (
                <Link className='np-aperture-button' to='/sign-up'>
                  {t('Create your first key')}
                  <ArrowUpRight aria-hidden='true' />
                </Link>
              )}
              <Link className='np-aperture-text-link' to='/pricing'>
                {t('View model catalogue')}
                <ArrowUpRight aria-hidden='true' />
              </Link>
            </div>
          </div>

          <NAperture
            accessibleLabel={t('Live model routes')}
            isLoading={modelsLoading}
            modelNames={modelNames}
            routeLabel={t('ready')}
          />
        </div>
      </section>

      <ProviderStrip />

      <OfferStrip
        isAuthenticated={props.isAuthenticated}
        offer={offerQuery.data ?? null}
        offerLoading={offerQuery.isLoading}
        registerPromo={registerPromo}
      />

      <section
        className='np-aperture-container np-aperture-evidence'
        aria-labelledby='np-aperture-evidence-title'
      >
        <div className='np-aperture-section-heading'>
          <p className='np-aperture-overline'>{t('What stays visible')}</p>
          <h2 id='np-aperture-evidence-title'>
            {t('Infrastructure you can actually inspect.')}
          </h2>
        </div>
        <div className='np-aperture-evidence-grid'>
          {EVIDENCE.map((item) => {
            const Icon = item.icon
            return (
              <article key={item.title} className='np-aperture-evidence-item'>
                <Icon aria-hidden='true' />
                <h3>{t(item.title)}</h3>
                <p>{t(item.description)}</p>
              </article>
            )
          })}
        </div>
      </section>

      <FAQ />

      <section
        className='np-aperture-container np-aperture-final-cta'
        aria-labelledby='np-aperture-final-cta-title'
      >
        <div>
          <p className='np-aperture-overline'>
            {t('Start with one clean signal')}
          </p>
          <h2 id='np-aperture-final-cta-title'>
            {t('Route the next request with less to think about.')}
          </h2>
          <p className='np-aperture-copy'>
            {t('One endpoint, visible usage, predictable access.')}
          </p>
        </div>
        {props.isAuthenticated ? (
          <Link className='np-aperture-button' to='/dashboard'>
            {t('Go to dashboard')}
            <ArrowUpRight aria-hidden='true' />
          </Link>
        ) : (
          <Link className='np-aperture-button' to='/sign-up'>
            {t('Start building')}
            <ArrowUpRight aria-hidden='true' />
          </Link>
        )}
      </section>
    </main>
  )
}
