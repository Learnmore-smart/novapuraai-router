import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  buildChannelTestTargets,
  getChannelTestResultKey,
  summarizeChannelTestScope,
  summarizeModelTestResults,
} from './channel-test-results.ts'

describe('channel test result matrix', () => {
  test('builds every model and key combination exactly once', () => {
    const targets = buildChannelTestTargets(['model-a', 'model-b'], 3, null)

    assert.deepEqual(targets, [
      { model: 'model-a', keyIndex: 0 },
      { model: 'model-a', keyIndex: 1 },
      { model: 'model-a', keyIndex: 2 },
      { model: 'model-b', keyIndex: 0 },
      { model: 'model-b', keyIndex: 1 },
      { model: 'model-b', keyIndex: 2 },
    ])
  })

  test('limits work to the selected key while preserving every model', () => {
    const targets = buildChannelTestTargets(['model-a', 'model-b'], 42, 17)

    assert.deepEqual(targets, [
      { model: 'model-a', keyIndex: 17 },
      { model: 'model-b', keyIndex: 17 },
    ])
  })

  test('aggregates per-model and total failure rates across keys', () => {
    const results = {
      [getChannelTestResultKey('model-a', 0)]: { status: 'success' as const },
      [getChannelTestResultKey('model-a', 1)]: { status: 'error' as const },
      [getChannelTestResultKey('model-b', 0)]: { status: 'success' as const },
      [getChannelTestResultKey('model-b', 1)]: { status: 'success' as const },
    }

    assert.deepEqual(summarizeModelTestResults(results, 'model-a', [0, 1]), {
      status: 'error',
      tested: 2,
      failed: 1,
      failureRate: 50,
    })
    assert.deepEqual(
      summarizeChannelTestScope(results, ['model-a', 'model-b'], [0, 1]),
      { tested: 4, failed: 1, failureRate: 25 }
    )
  })
})
