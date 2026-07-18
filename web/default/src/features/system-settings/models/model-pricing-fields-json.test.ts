import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { BulkPricingMaps } from './model-pricing-bulk-json.ts'
import {
  applyPricingFieldsJson,
  exportPricingFieldsJson,
  type PricingFieldsDrafts,
} from './model-pricing-fields-json.ts'

const maps: BulkPricingMaps = {
  modelPrice: '{"fixed":0.04}',
  modelRatio: '{"configured":1.5,"allowed":0.2}',
  cacheRatio: '{"configured":0.5,"allowed":0.25}',
  createCacheRatio: '{}',
  completionRatio: '{"configured":3,"allowed":2}',
  imageRatio: '{}',
  audioRatio: '{}',
  audioCompletionRatio: '{}',
  modelDiscount: '{"*":0.8,"configured":0.9,"allowed":0.7}',
  billingMode: '{}',
  billingExpr: '{}',
}

const emptyDrafts: PricingFieldsDrafts = {
  ModelPrice: '{}',
  ModelRatio: '{}',
  CacheRatio: '{}',
  CreateCacheRatio: '{}',
  CompletionRatio: '{}',
  ImageRatio: '{}',
  AudioRatio: '{}',
  AudioCompletionRatio: '{}',
  ModelDiscount: '{}',
}

describe('model pricing field JSON', () => {
  test('exports only allowed model entries for the unset view', () => {
    const drafts = exportPricingFieldsJson(maps, ['allowed'], true)

    assert.deepEqual(JSON.parse(drafts.ModelRatio), { allowed: 0.2 })
    assert.deepEqual(JSON.parse(drafts.CacheRatio), { allowed: 0.25 })
    assert.deepEqual(JSON.parse(drafts.ModelDiscount), { allowed: 0.7 })
    assert.deepEqual(JSON.parse(drafts.ModelPrice), {})
  })

  test('merges an unset-view draft without changing hidden configured entries', () => {
    const result = applyPricingFieldsJson(
      maps,
      {
        ...emptyDrafts,
        ModelRatio: '{"allowed":0.4}',
        CompletionRatio: '{"allowed":2.5}',
      },
      ['allowed'],
      true
    )

    assert.equal(result.ok, true)
    assert.deepEqual(JSON.parse(result.updates.ModelRatio), {
      configured: 1.5,
      allowed: 0.4,
    })
    assert.deepEqual(JSON.parse(result.updates.CompletionRatio), {
      configured: 3,
      allowed: 2.5,
    })
    assert.deepEqual(JSON.parse(result.updates.ModelDiscount), {
      '*': 0.8,
      configured: 0.9,
    })
  })

  test('rejects model names outside the unset-view scope', () => {
    const result = applyPricingFieldsJson(
      maps,
      { ...emptyDrafts, ModelRatio: '{"configured":2}' },
      ['allowed'],
      true
    )

    assert.equal(result.ok, false)
    assert.match(result.errors.join('\n'), /configured.*outside/i)
  })
})
