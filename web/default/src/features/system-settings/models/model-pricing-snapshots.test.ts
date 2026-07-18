import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { resolveVisibleModelNames } from './model-pricing-snapshots.ts'

describe('resolveVisibleModelNames', () => {
  test('all mode uses the union of saved and draft pricing models', () => {
    assert.deepEqual(
      resolveVisibleModelNames({
        savedModelNames: ['legacy-model', 'shared-model'],
        draftModelNames: ['draft-model', 'shared-model'],
        candidateModelNames: ['api-model'],
        filterMode: 'all',
      }),
      ['draft-model', 'legacy-model', 'shared-model']
    )
  })

  for (const filterMode of ['configured', 'unset'] as const) {
    test(`${filterMode} mode uses models exposed by configured API channels`, () => {
      assert.deepEqual(
        resolveVisibleModelNames({
          savedModelNames: ['legacy-model'],
          draftModelNames: ['draft-model'],
          candidateModelNames: ['z-api-model', 'a-api-model', 'a-api-model'],
          filterMode,
        }),
        ['a-api-model', 'z-api-model']
      )
    })
  }
})
