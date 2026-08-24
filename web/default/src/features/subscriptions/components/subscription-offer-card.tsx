import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  ExternalLink,
  LockKeyhole,
  Users,
} from 'lucide-react'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/stores/auth-store'

import {
  createCustomerPortalSession,
  getSubscriptionOffer,
  getStripeSubscriptionSummary,
  paySubscriptionStripe,
} from '../api'
import {
  canStartSubscriptionCheckout,
  formatOfferReservationExpiry,
  getOfferAvailabilityLabel,
  getSubscriptionCheckoutErrorKey,
  getSubscriptionPriceCards,
  hasSubscriptionCheckoutConflict,
  isSubscriptionLifecycleActive,
  normalizeStripeSubscriptionSummary,
  normalizeSubscriptionOffer,
} from '../lib/subscription-offer'
import type { SubscriptionLifecycleSummary, SubscriptionOffer } from '../types'

interface SubscriptionOfferCardProps {
  onPurchaseSuccess?: () => void | Promise<void>
  onAvailabilityChange?: (available: boolean) => void
  subscription?: SubscriptionLifecycleSummary
}

function formatPrice(
  offer: SubscriptionOffer,
  display?: string,
  minor?: number
) {
  if (display) return display
  if (minor === undefined || !offer.currency) return null
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: offer.currency,
    }).format(minor / 100)
  } catch {
    return null
  }
}

export function SubscriptionOfferCard(props: SubscriptionOfferCardProps) {
  const { t } = useTranslation()
  const userId = useAuthStore((state) => state.auth.user?.id)
  const queryClient = useQueryClient()
  const offerQuery = useQuery({
    queryKey: ['subscription', 'offer', userId],
    queryFn: async () => {
      const response = await getSubscriptionOffer()
      if (!response.success) return null
      return normalizeSubscriptionOffer(response.data)
    },
    staleTime: 30_000,
    retry: false,
  })
  const summaryQuery = useQuery({
    queryKey: ['subscription', 'stripe-summary', userId],
    queryFn: async () => {
      const response = await getStripeSubscriptionSummary()
      if (!response.success) return null
      return {
        lifecycle: normalizeStripeSubscriptionSummary(response.data),
        reservation: response.data?.reservation ?? null,
      }
    },
    staleTime: 30_000,
    retry: false,
    enabled: Boolean(userId),
  })
  const onAvailabilityChange = props.onAvailabilityChange

  useEffect(() => {
    onAvailabilityChange?.(
      offerQuery.isPending ||
        Boolean(offerQuery.data?.active && !offerQuery.data.sold_out)
    )
  }, [offerQuery.data, offerQuery.isPending, onAvailabilityChange])

  const checkoutMutation = useMutation({
    mutationFn: async (planId: number) =>
      paySubscriptionStripe(
        { plan_id: planId },
        { skipBusinessError: true, skipErrorHandler: true }
      ),
    onSuccess: async (response) => {
      const url =
        response.data?.pay_link || response.data?.checkout_url || response.url
      if (!url) {
        toast.error(t('Checkout link unavailable'))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: ['subscription', 'offer'],
      })
      await props.onPurchaseSuccess?.()
      window.location.assign(url)
    },
    onError: (error) => {
      void queryClient.invalidateQueries({
        queryKey: ['subscription', 'offer'],
      })
      const errorKey = getSubscriptionCheckoutErrorKey(error)
      toast.error(t(errorKey || 'Checkout could not be started'))
    },
  })

  const portalMutation = useMutation({
    mutationFn: createCustomerPortalSession,
    onSuccess: (response) => {
      if (response.success && response.data?.url) {
        window.location.assign(response.data.url)
        return
      }
      toast.error(t('Customer Portal is unavailable'))
    },
    onError: () => toast.error(t('Customer Portal is unavailable')),
  })

  const offer = offerQuery.data
  if (!offer) return null

  const availabilityKey = getOfferAvailabilityLabel(offer)
  const availability = t(availabilityKey)
  const priceCards = getSubscriptionPriceCards(offer)
  const reservationExpiry = formatOfferReservationExpiry(
    offer.reservation_expires_at
  )
  const fetchedLifecycle = summaryQuery.data?.lifecycle
  const fetchedReservation = summaryQuery.data?.reservation
  const lifecycle =
    fetchedLifecycle ||
    offer.subscription ||
    props.subscription ||
    undefined
  const periodStart = formatOfferReservationExpiry(
    lifecycle?.current_period_start
  )
  const periodEnd = formatOfferReservationExpiry(lifecycle?.current_period_end)
  const gracePeriodEnd = formatOfferReservationExpiry(
    lifecycle?.grace_period_end
  )
  const hasCurrentSubscription = isSubscriptionLifecycleActive(lifecycle)
  const hasFetchedCheckoutConflict = hasSubscriptionCheckoutConflict(
    fetchedLifecycle,
    fetchedReservation
  )
  const canCheckout =
    Boolean(userId) &&
    !summaryQuery.isPending &&
    !summaryQuery.isError &&
    summaryQuery.data !== null &&
    !hasFetchedCheckoutConflict &&
    canStartSubscriptionCheckout(offer, lifecycle)
  const hasLifecycle = Boolean(lifecycle)
  let checkoutUnavailableLabel = t('Reservation unavailable')
  if (offer.sold_out) {
    checkoutUnavailableLabel = t('Sold out')
  } else if (offer.pending) {
    checkoutUnavailableLabel = t('Reservation pending')
  } else if (hasCurrentSubscription) {
    checkoutUnavailableLabel = t('Already subscribed')
  }

  let checkoutAction = (
    <Button className='w-full' variant='outline' disabled>
      {checkoutUnavailableLabel}
    </Button>
  )
  if (!userId) {
    checkoutAction = (
      <Button className='w-full' render={<Link to='/sign-up' />}>
        {t('Create account')}
        <ArrowRight data-icon='inline-end' />
      </Button>
    )
  } else if (canCheckout) {
    checkoutAction = (
      <Button
        className='w-full'
        disabled={checkoutMutation.isPending}
        onClick={() => {
          if (offer.plan_id) checkoutMutation.mutate(offer.plan_id)
        }}
      >
        {checkoutMutation.isPending ? t('Loading') : t('Reserve a seat')}
        <ArrowRight data-icon='inline-end' />
      </Button>
    )
  }

  return (
    <section
      aria-labelledby='subscription-offer-title'
      data-offer-state={availabilityKey.toLowerCase().replaceAll(' ', '-')}
      className='border-border bg-background overflow-hidden border-y'
    >
      <div className='grid gap-6 p-4 sm:p-6 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.7fr)] lg:p-8'>
        <div className='min-w-0'>
          <div className='flex flex-wrap items-center gap-x-3 gap-y-2'>
            <span className='text-primary font-mono text-xs font-semibold tracking-[0.16em] uppercase'>
              {t('Founding 20')}
            </span>
            <span className='text-muted-foreground text-xs'>·</span>
            <span className='text-muted-foreground text-xs'>
              {availability}
            </span>
          </div>
          <h2
            id='subscription-offer-title'
            className='mt-3 text-2xl font-semibold tracking-[-0.035em] sm:text-3xl'
          >
            {t('Unlimited access, with a fair-use boundary.')}
          </h2>
          <p className='text-muted-foreground mt-3 max-w-2xl text-sm leading-7'>
            <span className='font-medium'>{t('All Models')}</span>
            <span className='mx-1'>·</span>
            {t('Unlimited access, with a fair-use boundary.')}
          </p>

          <div className='mt-6 grid gap-3 sm:grid-cols-2'>
            <div className='border-border border-t pt-3'>
              <div className='text-muted-foreground flex items-center gap-2 text-xs'>
                <Users className='size-3.5' aria-hidden='true' />
                {t('Seats remaining')}
              </div>
              <div className='mt-1 font-mono text-lg tabular-nums'>
                {offer.remaining ?? '—'} / {offer.limit ?? '—'}
              </div>
            </div>
            <div className='border-border border-t pt-3'>
              <div className='text-muted-foreground flex items-center gap-2 text-xs'>
                <LockKeyhole className='size-3.5' aria-hidden='true' />
                {t('Founder claims remaining')}
              </div>
              <div className='mt-1 font-mono text-lg tabular-nums'>
                {offer.founder_claims_remaining ?? '—'}
              </div>
            </div>
          </div>

          {reservationExpiry && (
            <dl className='mt-6 grid gap-3 text-sm sm:grid-cols-2'>
              <div>
                <dt className='text-muted-foreground'>
                  {t('Reservation expires')}
                </dt>
                <dd className='mt-1'>{reservationExpiry}</dd>
              </div>
            </dl>
          )}
        </div>

        <div className='border-border flex min-w-0 flex-col justify-between border-t pt-5 lg:border-t-0 lg:border-l lg:pt-0 lg:pl-6'>
          <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-1'>
            {priceCards.map((card) => (
              <article
                key={card.tier}
                className={
                  card.tier === 'founder'
                    ? 'border-primary/45 bg-primary/5 rounded-xl border p-4'
                    : 'border-border bg-muted/20 rounded-xl border p-4'
                }
              >
                <div className='flex items-center justify-between gap-3'>
                  <h3 className='text-sm font-semibold'>{t(card.titleKey)}</h3>
                  <span className='text-muted-foreground text-xs'>
                    {card.tier === 'founder'
                      ? t('Limited offer')
                      : t('Future standard seat')}
                  </span>
                </div>
                <div className='mt-3 text-3xl font-semibold tracking-[-0.04em]'>
                  {formatPrice(offer, card.priceDisplay, card.priceMinor) ||
                    t('Price at checkout')}
                </div>
                <p className='text-muted-foreground mt-1 text-xs'>
                  {t('per month')}
                </p>
                {card.tier === 'founder' && (
                  <p className='text-muted-foreground mt-3 text-xs'>
                    {t('{{remaining}} of {{limit}} seats remaining', {
                      remaining: card.remaining ?? '—',
                      limit: card.limit ?? '—',
                    })}
                  </p>
                )}
              </article>
            ))}
          </div>

          <div className='mt-6 space-y-2'>
            {hasLifecycle && (
              <div className='border-border space-y-2 border-t py-3 text-xs'>
                <div className='flex items-center justify-between gap-3'>
                  <span className='text-muted-foreground'>
                    {t('Stripe status')}
                  </span>
                  <span className='font-mono'>
                    {lifecycle?.stripe_status || lifecycle?.status || '—'}
                  </span>
                </div>
                {(periodStart || periodEnd) && (
                  <div className='flex items-center justify-between gap-3'>
                    <span className='text-muted-foreground'>
                      {t('Current period')}
                    </span>
                    <span className='text-right'>
                      {periodStart || '—'} – {periodEnd || '—'}
                    </span>
                  </div>
                )}
                {lifecycle?.cancel_at_period_end && (
                  <div className='text-warning'>
                    {t('Cancels at period end')}
                  </div>
                )}
                {gracePeriodEnd && (
                  <div className='text-warning'>
                    {t('Payment grace period ends {{date}}', {
                      date: gracePeriodEnd,
                    })}
                  </div>
                )}
              </div>
            )}

            {checkoutAction}

            {hasLifecycle && (
              <Button
                variant='ghost'
                className='w-full'
                disabled={portalMutation.isPending}
                onClick={() => portalMutation.mutate()}
              >
                {portalMutation.isPending ? t('Loading') : t('Manage billing')}
                <ExternalLink data-icon='inline-end' />
              </Button>
            )}
          </div>
        </div>
      </div>
    </section>
  )
}
