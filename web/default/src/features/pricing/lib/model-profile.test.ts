import assert from 'node:assert/strict'
import test from 'node:test'

import { getModelProfilePath } from './model-profile.ts'

test('builds the canonical public model profile path', () => {
  assert.equal(
    getModelProfilePath('deepseek-v4-flash-0731'),
    '/pricing/deepseek-v4-flash-0731'
  )
  assert.equal(
    getModelProfilePath('model with spaces'),
    '/pricing/model%20with%20spaces'
  )
})
