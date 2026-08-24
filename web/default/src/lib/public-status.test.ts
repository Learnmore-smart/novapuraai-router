import assert from 'node:assert/strict'
import test from 'node:test'

import { getRegisterPromo } from './public-status.ts'

test('accepts only the enabled canonical registration promo fields', () => {
  assert.deepEqual(
    getRegisterPromo({
      register_promo_enabled: true,
      register_promo_amount: 10,
      register_promo_currency: 'CNY',
      register_promo_cny_yuan: 50,
    }),
    { amount: 10, currency: 'CNY' }
  )
})

test('does not expose a disabled or incomplete registration promo', () => {
  assert.equal(
    getRegisterPromo({
      register_promo_enabled: false,
      register_promo_amount: 10,
      register_promo_currency: 'CNY',
    }),
    null
  )
  assert.equal(
    getRegisterPromo({
      register_promo_enabled: true,
      register_promo_amount: 10,
    }),
    null
  )
})
