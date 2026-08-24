import type {
  SubscriptionLifecycleSummary,
  SubscriptionOffer,
  SubscriptionOfferEnvelope,
  StripeSubscriptionReservation,
  StripeSubscriptionSummary,
} from '../types'

export interface SubscriptionPriceCard {
  tier: 'founder' | 'standard'
  titleKey: 'Founder 20' | 'Standard price'
  priceMinor?: number
  priceDisplay?: string
  limit?: number
  remaining?: number
  available: boolean
}

export function getSubscriptionPriceCards(
  offer: SubscriptionOffer
): SubscriptionPriceCard[] {
  const founderCard: SubscriptionPriceCard = {
    tier: 'founder',
    titleKey: 'Founder 20',
    priceMinor:
      offer.current_price_tier === 'founder'
        ? offer.current_price_minor
        : offer.founder_amount_minor,
    priceDisplay:
      offer.current_price_tier === 'founder'
        ? offer.current_price_display
        : undefined,
    limit: offer.limit,
    remaining: offer.remaining,
    available:
      offer.active === true &&
      offer.current_price_tier === 'founder' &&
      !offer.sold_out,
  }
  const standardIsCurrent = offer.current_price_tier === 'standard'
  const standardCard: SubscriptionPriceCard = {
    tier: 'standard',
    titleKey: 'Standard price',
    priceMinor: standardIsCurrent
      ? offer.current_price_minor
      : (offer.future_standard_price_minor ?? offer.standard_amount_minor),
    priceDisplay: standardIsCurrent
      ? offer.current_price_display
      : offer.future_standard_price_display,
    available: offer.active === true && standardIsCurrent && !offer.sold_out,
  }

  return [founderCard, standardCard]
}

export function getOfferTierLabel(
  tier: SubscriptionOffer['current_price_tier']
): string {
  if (tier === 'founder') return 'Founder price'
  if (tier === 'standard') return 'Future standard seat'
  return 'Current price'
}

export function getOfferAvailabilityLabel(offer: {
  active?: boolean
  pending?: boolean
  sold_out?: boolean
}): string {
  if (offer.sold_out) return 'Sold out'
  if (offer.pending) return 'Reservation pending'
  if (offer.active) return 'Available'
  return 'Unavailable'
}

export function getSubscriptionCheckoutErrorKey(error: unknown): string | null {
  if (!error || typeof error !== 'object') return null

  const response = (error as { response?: unknown }).response
  if (!response || typeof response !== 'object') return null

  const status = (response as { status?: unknown }).status
  if (Number(status) !== 409) return null

  const data = (response as { data?: unknown }).data
  const code =
    data && typeof data === 'object' && 'code' in data
      ? (data as { code?: unknown }).code
      : undefined

  switch (code) {
    case 'subscription_capacity_full':
    case 'subscription_founder_sold_out':
      return 'Founder capacity is full'
    case 'subscription_already_active':
      return 'You already have an active subscription'
    case 'subscription_already_pending':
      return 'You already have a pending reservation'
    default:
      return 'Subscription checkout is unavailable'
  }
}

export function isSubscriptionLifecycleActive(
  lifecycle: SubscriptionLifecycleSummary | null | undefined,
  now = Date.now()
): boolean {
  if (!lifecycle) return false

  const status = (
    lifecycle.stripe_status ||
    lifecycle.status ||
    ''
  ).toLowerCase()
  if (!['canceled', 'cancelled', 'expired'].includes(status)) return true

  const gracePeriodEnd = lifecycle.grace_period_end
  if (gracePeriodEnd === null || gracePeriodEnd === undefined) return false
  if (typeof gracePeriodEnd === 'string' && gracePeriodEnd.trim() === '') {
    return false
  }

  let gracePeriodTimestamp: number
  if (typeof gracePeriodEnd === 'number') {
    gracePeriodTimestamp =
      gracePeriodEnd < 10_000_000_000 ? gracePeriodEnd * 1000 : gracePeriodEnd
  } else {
    gracePeriodTimestamp = Date.parse(gracePeriodEnd)
  }

  return Number.isFinite(gracePeriodTimestamp) && gracePeriodTimestamp > now
}

export function canStartSubscriptionCheckout(
  offer: SubscriptionOffer,
  fallbackLifecycle?: SubscriptionLifecycleSummary
): boolean {
  const lifecycle = offer.subscription || fallbackLifecycle
  return (
    offer.active === true &&
    Boolean(offer.plan_id) &&
    !offer.pending &&
    !offer.sold_out &&
    !isSubscriptionLifecycleActive(lifecycle)
  )
}

export function hasSubscriptionCheckoutConflict(
  lifecycle: SubscriptionLifecycleSummary | null | undefined,
  reservation: StripeSubscriptionReservation | null | undefined,
  now = Date.now()
): boolean {
  if (isSubscriptionLifecycleActive(lifecycle, now)) return true
  if (!reservation) return false

  const status = reservation.status?.trim().toLowerCase()
  if (
    !status ||
    ['expired', 'released', 'canceled', 'cancelled'].includes(status)
  ) {
    return false
  }

  // A non-terminal reservation is authoritative even if its expiry is
  // missing or malformed. The server owns reservation expiry and will return
  // a terminal state when it is released.
  return true
}

export function normalizeSubscriptionOffer(
  value: SubscriptionOffer | SubscriptionOfferEnvelope | null | undefined
): SubscriptionOffer | null {
  if (
    !value ||
    typeof value !== 'object' ||
    Array.isArray(value)
  ) {
    return null
  }

  const candidate = value as SubscriptionOffer | SubscriptionOfferEnvelope
  const offer = 'offer' in candidate ? candidate.offer : candidate
  if (
    !offer ||
    typeof offer !== 'object' ||
    Array.isArray(offer)
  ) {
    return null
  }

  // This boundary only unwraps the envelope. Availability, capacity, price
  // tier, and discount state are server-owned fields and must never be
  // reconstructed from sandbox implementation fields.
  return offer as SubscriptionOffer
}

export function normalizeStripeSubscriptionSummary(
  summary: StripeSubscriptionSummary | null | undefined
): SubscriptionLifecycleSummary | null {
  const subscription = summary?.subscription
  if (!subscription) return null

  return {
    stripe_status: subscription.status,
    stripe_customer_id: subscription.stripe_customer_id,
    stripe_subscription_id: subscription.stripe_subscription_id,
    stripe_price_id: subscription.stripe_price_id,
    current_period_start: subscription.current_period_start,
    current_period_end: subscription.current_period_end,
    cancel_at_period_end: subscription.cancel_at_period_end,
    grace_period_end: subscription.grace_until,
    price_tier: subscription.tier,
    status: subscription.status,
  }
}

export function formatOfferReservationExpiry(
  value: number | string | null | undefined
): string | null {
  if (value === null || value === undefined || value === '') return null
  const timestamp =
    typeof value === 'number'
      ? new Date(value < 10_000_000_000 ? value * 1000 : value)
      : new Date(value)
  if (Number.isNaN(timestamp.getTime())) return null
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(timestamp)
}
