import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  formatOfferReservationExpiry,
  canStartSubscriptionCheckout,
  getOfferAvailabilityLabel,
  getSubscriptionCheckoutErrorKey,
  getOfferTierLabel,
  getSubscriptionPriceCards,
  hasSubscriptionCheckoutConflict,
  isSubscriptionLifecycleActive,
  normalizeSubscriptionOffer,
  normalizeStripeSubscriptionSummary,
} from './subscription-offer'

describe('subscription offer display helpers', () => {
  test('keeps founder and standard wording separate from discount math', () => {
    assert.equal(getOfferTierLabel('founder'), 'Founder price')
    assert.equal(getOfferTierLabel('standard'), 'Future standard seat')
    assert.equal(getOfferTierLabel(undefined), 'Current price')
  })

  test('projects separate Founder 20 and standard price cards from server values', () => {
    const cards = getSubscriptionPriceCards({
      active: true,
      current_price_tier: 'founder',
      current_price_minor: 2_500,
      current_price_display: '$25.00',
      future_standard_price_minor: 10_000,
      future_standard_price_display: '$100.00',
      currency: 'USD',
      limit: 20,
      remaining: 7,
    })

    assert.deepEqual(cards, [
      {
        tier: 'founder',
        titleKey: 'Founder 20',
        priceMinor: 2_500,
        priceDisplay: '$25.00',
        limit: 20,
        remaining: 7,
        available: true,
      },
      {
        tier: 'standard',
        titleKey: 'Standard price',
        priceMinor: 10_000,
        priceDisplay: '$100.00',
        available: false,
      },
    ])
  })

  test('keeps the server-provided Founder price visible after the limited tier sells out', () => {
    const cards = getSubscriptionPriceCards({
      active: true,
      sold_out: false,
      current_price_tier: 'standard',
      current_price_minor: 9_999,
      founder_amount_minor: 1_999,
      standard_amount_minor: 9_999,
      currency: 'CNY',
      limit: 20,
      remaining: 0,
    })

    assert.equal(cards[0].priceMinor, 1_999)
    assert.equal(cards[0].available, false)
    assert.equal(cards[1].priceMinor, 9_999)
    assert.equal(cards[1].available, true)
  })

  test('uses server availability state without deriving remaining seats', () => {
    assert.equal(getOfferAvailabilityLabel({ sold_out: true }), 'Sold out')
    assert.equal(
      getOfferAvailabilityLabel({ pending: true }),
      'Reservation pending'
    )
    assert.equal(getOfferAvailabilityLabel({ active: true }), 'Available')
    assert.equal(getOfferAvailabilityLabel({}), 'Unavailable')
  })

  test('normalizes a model-agnostic direct or wrapped public offer response', () => {
    const offer = {
      active: true,
      remaining: 7,
    }
    assert.deepEqual(normalizeSubscriptionOffer(offer), offer)
    assert.deepEqual(normalizeSubscriptionOffer({ offer }), offer)
    assert.equal(normalizeSubscriptionOffer({ offer: undefined }), null)
    assert.equal(normalizeSubscriptionOffer(undefined), null)
  })

  test('normalizes subscription lifecycle without a model-specific field', () => {
    const lifecycle = normalizeStripeSubscriptionSummary({
      subscription: {
        status: 'active',
        tier: 'founder',
      },
    })

    assert.equal(lifecycle?.status, 'active')
    assert.equal(lifecycle?.price_tier, 'founder')
    assert.equal('model' in (lifecycle ?? {}), false)
  })

  test('treats malformed offer envelopes as unavailable', () => {
    assert.equal(
      normalizeSubscriptionOffer('not-an-offer' as never),
      null
    )
    assert.equal(normalizeSubscriptionOffer([] as never), null)
  })

  test('does not derive public offer state from raw sandbox fields', () => {
    const raw = {
      enabled: true,
      plan_id: 42,
      currency: 'CNY',
      founder_amount_minor: 1999,
      standard_amount_minor: 9999,
      max_active_seats: 20,
      active_seats: 3,
      pending_seats: 2,
      founder_claims_remaining: 17,
    }
    const offer = normalizeSubscriptionOffer(raw)

    assert.equal(offer?.active, undefined)
    assert.equal(offer?.remaining, undefined)
    assert.equal(offer?.sold_out, undefined)
    assert.equal(offer?.current_price_tier, undefined)
    assert.equal(offer?.current_price_minor, undefined)
    assert.deepEqual(offer, raw)
  })

  test('formats a reservation expiry without inventing a value', () => {
    assert.equal(formatOfferReservationExpiry(undefined), null)
    assert.match(
      formatOfferReservationExpiry('2026-08-22T12:00:00.000Z') ?? '',
      /2026/
    )
  })

  test('maps server checkout conflicts to clear localized copy keys', () => {
    assert.equal(
      getSubscriptionCheckoutErrorKey({
        response: {
          status: 409,
          data: { code: 'subscription_capacity_full' },
        },
      }),
      'Founder capacity is full'
    )
    assert.equal(
      getSubscriptionCheckoutErrorKey({
        response: {
          status: 409,
          data: { code: 'subscription_already_active' },
        },
      }),
      'You already have an active subscription'
    )
    assert.equal(
      getSubscriptionCheckoutErrorKey({
        response: {
          status: 409,
          data: { code: 'subscription_already_pending' },
        },
      }),
      'You already have a pending reservation'
    )
    assert.equal(
      getSubscriptionCheckoutErrorKey({
        response: { status: 409, data: { code: 'unknown_conflict' } },
      }),
      'Subscription checkout is unavailable'
    )
    assert.equal(
      getSubscriptionCheckoutErrorKey({
        response: { status: 500, data: { code: 'subscription_capacity_full' } },
      }),
      null
    )
  })

  test('honors the server-provided grace-period boundary', () => {
    const now = Date.parse('2026-08-22T12:00:00.000Z')
    assert.equal(isSubscriptionLifecycleActive({ status: 'active' }, now), true)
    assert.equal(
      isSubscriptionLifecycleActive(
        { status: 'canceled', grace_period_end: '2026-08-22T12:01:00.000Z' },
        now
      ),
      true
    )
    assert.equal(
      isSubscriptionLifecycleActive(
        { status: 'canceled', grace_period_end: '2026-08-22T11:59:00.000Z' },
        now
      ),
      false
    )
    assert.equal(
      isSubscriptionLifecycleActive({ status: 'canceled' }, now),
      false
    )
  })

  test('requires explicit server availability before starting checkout', () => {
    assert.equal(
      canStartSubscriptionCheckout({
        plan_id: 1,
        active: true,
        pending: false,
        sold_out: false,
      }),
      true
    )
    assert.equal(
      canStartSubscriptionCheckout({
        plan_id: 1,
        pending: false,
        sold_out: false,
      }),
      false
    )
    assert.equal(
      canStartSubscriptionCheckout({
        plan_id: 1,
        active: true,
        pending: true,
        sold_out: false,
      }),
      false
    )
    assert.equal(
      canStartSubscriptionCheckout(
        {
          plan_id: 1,
          active: true,
          pending: false,
          sold_out: false,
        },
        { status: 'active' }
      ),
      false
    )
  })

  test('treats fetched active, grace, and pending states as checkout conflicts', () => {
    const now = Date.parse('2026-08-22T12:00:00.000Z')
    assert.equal(
      hasSubscriptionCheckoutConflict({ status: 'active' }, undefined, now),
      true
    )
    assert.equal(
      hasSubscriptionCheckoutConflict(
        { status: 'canceled', grace_period_end: '2026-08-22T12:01:00.000Z' },
        undefined,
        now
      ),
      true
    )
    assert.equal(
      hasSubscriptionCheckoutConflict(
        undefined,
        { status: 'pending', expires_at: '2026-08-22T12:30:00.000Z' },
        now
      ),
      true
    )
    assert.equal(
      hasSubscriptionCheckoutConflict(
        { status: 'canceled' },
        { status: 'expired', expires_at: '2026-08-22T11:00:00.000Z' },
        now
      ),
      false
    )
  })
})
