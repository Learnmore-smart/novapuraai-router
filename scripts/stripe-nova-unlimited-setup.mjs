#!/usr/bin/env node
// stripe-nova-unlimited-setup.mjs
//
// One-time Stripe setup for the NovaPura subscription feature.
// Creates:
//   1. A Stripe Product (e.g. "NovaPura Unlimited")
//   2. Two recurring Prices on that product — one in USD, one in CNY — for
//      auto-renew mode (Stripe Checkout subscription mode).
//   3. (Optional) A Stripe Coupon + Promotion Code for subscription discounts.
//
// After the script runs, paste the printed IDs into the NovaPura admin UI
// (Subscription Plans → edit plan) or the DB:
//   - stripe_product_id
//   - stripe_price_id_usd
//   - stripe_price_id_cny
//   - (optional) stripe_coupon_id on the SubscriptionCoupon record
//
// Usage:
//   STRIPE_API_KEY=sk_test_... node scripts/stripe-nova-unlimited-setup.mjs \
//     --product "NovaPura Unlimited" \
//     --usd-amount 19.99 \
//     --cny-amount 149.00 \
//     --interval month \
//     [--coupon-percent 20] \
//     [--coupon-duration 3]
//
// All amounts are in major units (dollars / yuan). The script converts to
// minor units (cents / fen) for the Stripe API.

import process from 'node:process'

function parseArgs(argv) {
  const args = {}
  for (let i = 2; i < argv.length; i++) {
    const arg = argv[i]
    if (arg.startsWith('--')) {
      const key = arg.slice(2)
      const value = argv[i + 1] && !argv[i + 1].startsWith('--') ? argv[++i] : 'true'
      args[key] = value
    }
  }
  return args
}

const args = parseArgs(process.argv)

const apiKey = process.env.STRIPE_API_KEY
if (!apiKey) {
  console.error('Error: STRIPE_API_KEY environment variable is required.')
  process.exit(1)
}

const productName = args.product || 'NovaPura Unlimited'
const usdAmount = parseFloat(args['usd-amount'] || '19.99')
const cnyAmount = parseFloat(args['cny-amount'] || '149.00')
const interval = args.interval || 'month'
const intervalCount = parseInt(args['interval-count'] || '1', 10)
const couponPercent = args['coupon-percent'] ? parseInt(args['coupon-percent'], 10) : null
const couponDuration = parseInt(args['coupon-duration'] || '1', 10)

if (!['day', 'week', 'month', 'year'].includes(interval)) {
  console.error(`Error: --interval must be one of day, week, month, year (got "${interval}")`)
  process.exit(1)
}

async function stripeFetch(path, body) {
  const res = await fetch(`https://api.stripe.com/v1${path}`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    body,
  })
  const data = await res.json()
  if (!res.ok) {
    console.error(`Stripe API error (${res.status}) on ${path}:`)
    console.error(JSON.stringify(data, null, 2))
    process.exit(1)
  }
  return data
}

function toMinorUnits(amount) {
  // Stripe uses minor units (cents / fen). Round to avoid floating-point drift.
  return Math.round(amount * 100)
}

async function main() {
  console.log(`Creating Stripe Product: "${productName}"...`)
  const product = await stripeFetch('/products', new URLSearchParams({
    name: productName,
    description: 'NovaPura unlimited subscription plan (auto-renew)',
  }))
  console.log(`  ✓ Product ID: ${product.id}`)

  console.log(`Creating USD Price ($${usdAmount} / ${interval})...`)
  const usdPrice = await stripeFetch('/prices', new URLSearchParams({
    product: product.id,
    currency: 'usd',
    unit_amount: String(toMinorUnits(usdAmount)),
    'recurring[interval]': interval,
    'recurring[interval_count]': String(intervalCount),
  }))
  console.log(`  ✓ USD Price ID: ${usdPrice.id}`)

  console.log(`Creating CNY Price (¥${cnyAmount} / ${interval})...`)
  const cnyPrice = await stripeFetch('/prices', new URLSearchParams({
    product: product.id,
    currency: 'cny',
    unit_amount: String(toMinorUnits(cnyAmount)),
    'recurring[interval]': interval,
    'recurring[interval_count]': String(intervalCount),
  }))
  console.log(`  ✓ CNY Price ID: ${cnyPrice.id}`)

  let coupon = null
  if (couponPercent && couponPercent > 0 && couponPercent < 100) {
    console.log(`Creating Coupon (${couponPercent}% off, ${couponDuration} months)...`)
    coupon = await stripeFetch('/coupons', new URLSearchParams({
      percent_off: String(couponPercent),
      duration: 'repeating',
      duration_in_months: String(couponDuration),
    }))
    console.log(`  ✓ Coupon ID: ${coupon.id}`)

    const promoCode = await stripeFetch('/promotion_codes', new URLSearchParams({
      coupon: coupon.id,
      active: 'true',
    }))
    console.log(`  ✓ Promotion Code: ${promoCode.code} (ID: ${promoCode.id})`)
  }

  console.log('\n--- NovaPura Admin UI values ---')
  console.log(`  stripe_product_id:      ${product.id}`)
  console.log(`  stripe_price_id_usd:    ${usdPrice.id}`)
  console.log(`  stripe_price_id_cny:    ${cnyPrice.id}`)
  if (coupon) {
    console.log(`  stripe_coupon_id:       ${coupon.id}`)
  }
  console.log('\nNext steps:')
  console.log('  1. In the NovaPura admin UI, create or edit a Subscription Plan and')
  console.log('     paste the IDs above into the corresponding fields.')
  console.log('  2. Set price_amount_usd, price_amount_cny, renewal_price_usd,')
  console.log('     renewal_price_cny to match the Stripe Price amounts.')
  console.log('  3. Ensure the Stripe webhook endpoint is registered at:')
  console.log('     https://your-domain/api/stripe/webhook')
  console.log('     Subscribe to these events:')
  console.log('       - checkout.session.completed')
  console.log('       - invoice.paid')
  console.log('       - invoice.payment_failed')
  console.log('       - customer.subscription.updated')
  console.log('       - customer.subscription.deleted')
}

main().catch((err) => {
  console.error('Fatal error:', err)
  process.exit(1)
})
