import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  getGlobalDiscountDraft,
  setGlobalDiscountDraft,
} from './model-global-discount.ts'

describe('global model discount draft', () => {
  test('enables and updates the global rate without changing model rates', () => {
    const enabled = setGlobalDiscountDraft('{"model-a":0.9}', true, 0.8)
    assert.deepEqual(JSON.parse(enabled), { 'model-a': 0.9, '*': 0.8 })
    assert.deepEqual(getGlobalDiscountDraft(enabled), {
      enabled: true,
      rate: 0.8,
    })
  })

  test('disabling removes only the global override', () => {
    const disabled = setGlobalDiscountDraft(
      '{"model-a":0.9,"*":0.8}',
      false,
      0.8
    )
    assert.deepEqual(JSON.parse(disabled), { 'model-a': 0.9 })
    assert.deepEqual(getGlobalDiscountDraft(disabled), {
      enabled: false,
      rate: null,
    })
  })

  test('keeps a rate of one active so it overrides individual discounts', () => {
    assert.deepEqual(getGlobalDiscountDraft('{"model-a":0.8,"*":1}'), {
      enabled: true,
      rate: 1,
    })
  })
})
