import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  applyPricingJson,
  exportPricingJson,
  getUnsetPricingModelNames,
  type BulkPricingMaps,
} from './model-pricing-bulk-json.ts'

const emptyMaps: BulkPricingMaps = {
  modelPrice: '{}',
  modelRatio: '{}',
  cacheRatio: '{}',
  createCacheRatio: '{}',
  completionRatio: '{}',
  imageRatio: '{}',
  audioRatio: '{}',
  audioCompletionRatio: '{}',
  modelDiscount: '{}',
  billingMode: '{}',
  billingExpr: '{}',
}

describe('bulk pricing JSON', () => {
  test('applies token prices using the visual editor conversion rules', () => {
    const result = applyPricingJson(
      emptyMaps,
      JSON.stringify({
        'test-model': {
          input: 3,
          output: 15,
          cache_read: 0.3,
          cache_write: 3.75,
          image_input: 2.5,
          audio_input: 3.81,
          audio_output: 15.11,
          discount: 0.8,
        },
      })
    )
    assert.ok(result.ok)
    assert.equal(result.applied, 1)
    const ratio = JSON.parse(result.updates.ModelRatio)
    const completion = JSON.parse(result.updates.CompletionRatio)
    const cache = JSON.parse(result.updates.CacheRatio)
    const createCache = JSON.parse(result.updates.CreateCacheRatio)
    const image = JSON.parse(result.updates.ImageRatio)
    const audio = JSON.parse(result.updates.AudioRatio)
    const audioCompletion = JSON.parse(result.updates.AudioCompletionRatio)
    const discount = JSON.parse(result.updates.ModelDiscount)
    assert.equal(ratio['test-model'], 1.5)
    assert.equal(completion['test-model'], 5)
    assert.equal(cache['test-model'], 0.1)
    assert.equal(createCache['test-model'], 1.25)
    assert.ok(Math.abs(image['test-model'] - 2.5 / 3) < 1e-6)
    assert.equal(audio['test-model'], 1.27)
    assert.ok(Math.abs(audioCompletion['test-model'] - 15.11 / 3.81) < 1e-6)
    assert.equal(discount['test-model'], 0.8)
  })

  test('round-trips through export', () => {
    const applied = applyPricingJson(
      emptyMaps,
      JSON.stringify({
        'token-model': { input: 3, output: 15, discount: 0.9 },
        'fixed-model': { per_request: 0.04 },
      })
    )
    assert.ok(applied.ok)
    const exported = JSON.parse(
      exportPricingJson({
        ...emptyMaps,
        modelRatio: applied.updates.ModelRatio,
        modelPrice: applied.updates.ModelPrice,
        completionRatio: applied.updates.CompletionRatio,
        modelDiscount: applied.updates.ModelDiscount,
      })
    )
    assert.deepEqual(exported['token-model'], {
      input: 3,
      output: 15,
      discount: 0.9,
    })
    assert.deepEqual(exported['fixed-model'], { per_request: 0.04 })
  })

  test('exports every enabled model and omits the global discount control key', () => {
    const exported = JSON.parse(
      exportPricingJson(
        {
          ...emptyMaps,
          modelRatio: '{"configured":1.5}',
          modelDiscount: '{"*":0.8,"configured":0.9}',
        },
        ['unset-model', 'configured']
      )
    )

    assert.deepEqual(exported.configured, {
      input: 3,
      output: 3,
      discount: 0.9,
    })
    assert.equal(exported['unset-model'], null)
    assert.equal(exported['*'], undefined)
  })

  test('defaults legacy zero discounts to 1 in generated JSON', () => {
    const json = exportPricingJson({
      ...emptyMaps,
      modelRatio: '{"no-discount":1.5}',
      completionRatio: '{"no-discount":3}',
      modelDiscount: '{"no-discount":0}',
    })
    const exported = JSON.parse(json)

    assert.equal(exported['no-discount'].discount, 1)

    const applied = applyPricingJson(emptyMaps, json)
    assert.ok(applied.ok)
    assert.deepEqual(JSON.parse(applied.updates.ModelDiscount), {})
  })

  test('candidate-only export does not include configured models outside the requested scope', () => {
    const exported = JSON.parse(
      exportPricingJson(
        {
          ...emptyMaps,
          modelRatio: '{"configured":1.5}',
        },
        ['unset-model'],
        { candidateModelsOnly: true }
      )
    )

    assert.deepEqual(exported, { 'unset-model': null })
  })

  test('selects enabled models whose saved base pricing is unset', () => {
    const maps: BulkPricingMaps = {
      ...emptyMaps,
      modelPrice: '{"fixed":0.04}',
      modelRatio: '{"token":1.5}',
      modelDiscount: '{"discount-only":0.8}',
      billingMode: '{"expression":"tiered_expr"}',
    }

    assert.deepEqual(
      getUnsetPricingModelNames(maps, [
        'token',
        'fixed',
        'expression',
        'discount-only',
        'unset',
        'unset',
      ]),
      ['discount-only', 'unset']
    )
  })

  test('applying an exported unset model is idempotent', () => {
    const result = applyPricingJson(emptyMaps, '{"unset-model":null}')
    assert.ok(result.ok)
    assert.equal(result.removed, 0)
  })

  test('rejects the reserved global discount key even with surrounding whitespace', () => {
    const result = applyPricingJson(
      { ...emptyMaps, modelDiscount: '{"*":0.8}' },
      '{" * ":null}'
    )
    assert.equal(result.ok, false)
  })

  test('null removes a model from every map', () => {
    const maps: BulkPricingMaps = {
      ...emptyMaps,
      modelRatio: '{"doomed":1.5}',
      completionRatio: '{"doomed":5}',
      modelDiscount: '{"doomed":0.8}',
    }
    const result = applyPricingJson(maps, '{"doomed": null}')
    assert.ok(result.ok)
    assert.equal(result.removed, 1)
    assert.deepEqual(JSON.parse(result.updates.ModelRatio), {})
    assert.deepEqual(JSON.parse(result.updates.CompletionRatio), {})
    assert.deepEqual(JSON.parse(result.updates.ModelDiscount), {})
  })

  test('rejects invalid entries with per-model errors', () => {
    const cases: Array<[string, string]> = [
      ['{"m": {"output": 15}}', 'token prices require "input"'],
      ['{"m": {"per_request": 1, "input": 3}}', 'cannot be combined'],
      ['{"m": {"input": 3, "discount": 1.5}}', 'within (0, 1]'],
      ['{"m": {"input": -1}}', 'negative prices'],
      ['{"m": {"inptu": 3}}', 'unknown field'],
      ['{"m": {"input": "3"}}', 'must be a number'],
      ['[1,2]', 'Expected a JSON object'],
    ]
    for (const [doc, needle] of cases) {
      const result = applyPricingJson(emptyMaps, doc)
      assert.ok(!result.ok, doc)
      assert.ok(
        result.errors.some((error) => error.includes(needle)),
        `${doc} -> ${result.errors.join('; ')}`
      )
    }
  })

  test('requires a positive output price for token entries', () => {
    const result = applyPricingJson(emptyMaps, '{"token-model":{"input":3}}')
    assert.equal(result.ok, false)
    assert.match(result.errors.join('\n'), /positive.*output/i)
  })

  test('rejects non-positive token input even when output is positive', () => {
    const result = applyPricingJson(
      emptyMaps,
      JSON.stringify({ 'zero-input-model': { input: 0, output: 5 } })
    )

    assert.equal(result.ok, false)
    assert.match(result.errors.join('\n'), /positive.*input/i)
  })

  test('keeps per-request pricing exempt from the token output requirement', () => {
    const result = applyPricingJson(
      emptyMaps,
      JSON.stringify({ 'fixed-model': { per_request: 0 } })
    )
    assert.ok(result.ok)
    assert.equal(JSON.parse(result.updates.ModelPrice)['fixed-model'], 0)
  })

  test('keeps DeepSeek 0.11 ratio and 3 completion ratio through Unified JSON', () => {
    const exported = JSON.parse(
      exportPricingJson({
        ...emptyMaps,
        modelRatio: '{"deepseek-v4-flash-0731":0.11}',
        completionRatio: '{"deepseek-v4-flash-0731":3}',
      })
    )
    assert.deepEqual(exported['deepseek-v4-flash-0731'], {
      input: 0.22,
      output: 0.66,
    })

    const applied = applyPricingJson(emptyMaps, JSON.stringify(exported))
    assert.ok(applied.ok)
    assert.equal(
      JSON.parse(applied.updates.ModelRatio)['deepseek-v4-flash-0731'],
      0.11
    )
    assert.equal(
      JSON.parse(applied.updates.CompletionRatio)['deepseek-v4-flash-0731'],
      3
    )
  })

  test('rejects invalid model names before any pricing update is produced', () => {
    const result = applyPricingJson(
      emptyMaps,
      JSON.stringify({
        valid: { input: 1, output: 2 },
        'invalid-model"': { input: 1, output: 2 },
      })
    )
    assert.equal(result.ok, false)
    assert.match(result.errors.join('\n'), /invalid.*model.*quote/i)
  })

  test('models omitted from the document stay untouched', () => {
    const maps: BulkPricingMaps = {
      ...emptyMaps,
      modelRatio: '{"kept":2}',
    }
    const result = applyPricingJson(
      maps,
      '{"new-model": {"input": 1, "output": 3}}'
    )
    assert.ok(result.ok)
    const ratio = JSON.parse(result.updates.ModelRatio)
    assert.equal(ratio.kept, 2)
    assert.equal(ratio['new-model'], 0.5)
  })

  test('expression-billed models reject discount-only entries', () => {
    const maps: BulkPricingMaps = {
      ...emptyMaps,
      billingMode: '{"expr-model":"tiered_expr"}',
      billingExpr: '{"expr-model":"tier(\\"base\\", p * 1)"}',
    }
    const result = applyPricingJson(maps, '{"expr-model": {"discount": 0.5}}')
    assert.ok(result.ok)
    assert.deepEqual(result.skippedTiered, ['expr-model'])
    assert.deepEqual(JSON.parse(result.updates.ModelDiscount), {})
    // Providing real prices converts the model off expression billing.
    const converted = applyPricingJson(
      maps,
      '{"expr-model": {"input": 3, "output": 9}}'
    )
    assert.ok(converted.ok)
    assert.deepEqual(
      JSON.parse(converted.updates['billing_setting.billing_mode']),
      {}
    )
    assert.equal(JSON.parse(converted.updates.ModelRatio)['expr-model'], 1.5)
  })
})
