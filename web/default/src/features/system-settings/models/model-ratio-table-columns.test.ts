import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { getModelRatioColumnWidthClassNames } from './model-ratio-table-layout.ts'

describe('model pricing table column widths', () => {
  test('keeps semantic widths after the mode column is hidden', () => {
    assert.deepEqual(
      getModelRatioColumnWidthClassNames([
        'select',
        'name',
        'priceSummary',
        'actions',
      ]),
      ['w-9', 'w-[300px]', 'w-[300px]', 'w-[72px]']
    )
  })
})
