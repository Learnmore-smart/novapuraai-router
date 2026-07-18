import assert from 'node:assert/strict'
import test from 'node:test'

import { getBalanceBreakdown } from './balance-utils.ts'

test('uses explicit backend cash and promotional balances', () => {
  assert.deepEqual(getBalanceBreakdown(120, 80, 40), {
    total: 120,
    cash: 80,
    promo: 40,
  })
})

test('derives cash from total minus promo for older user payloads', () => {
  assert.deepEqual(getBalanceBreakdown(120, undefined, 35), {
    total: 120,
    cash: 85,
    promo: 35,
  })
})

test('never exposes negative or non-finite balances', () => {
  assert.deepEqual(getBalanceBreakdown(-10, Number.NaN, 25), {
    total: 0,
    cash: 0,
    promo: 25,
  })
})
