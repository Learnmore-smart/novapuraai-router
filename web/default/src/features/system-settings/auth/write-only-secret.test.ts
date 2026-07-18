import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  getWriteOnlySecretPlaceholder,
  hasWriteOnlySecretReplacement,
  orderAuthOptionUpdates,
  WRITE_ONLY_SECRET_MASK,
} from './write-only-secret.ts'

describe('write-only auth secrets', () => {
  test('shows only a UI mask when a secret is configured', () => {
    assert.equal(
      getWriteOnlySecretPlaceholder(true, 'Enter a secret'),
      WRITE_ONLY_SECRET_MASK
    )
    assert.match(WRITE_ONLY_SECRET_MASK, /^•+$/)
  })

  test('uses the normal prompt when no secret is configured', () => {
    assert.equal(
      getWriteOnlySecretPlaceholder(false, 'Enter a secret'),
      'Enter a secret'
    )
  })

  test('submits only a non-empty replacement', () => {
    assert.equal(hasWriteOnlySecretReplacement(''), false)
    assert.equal(hasWriteOnlySecretReplacement('   '), false)
    assert.equal(hasWriteOnlySecretReplacement('replacement'), true)
  })

  test('saves credentials before enabling an auth integration', () => {
    const updates = [
      ['Enabled', true],
      ['ClientId', 'client'],
      ['ClientSecret', 'secret'],
    ] as Array<[string, string | boolean]>

    assert.deepEqual(orderAuthOptionUpdates(updates, 'Enabled', true), [
      ['ClientId', 'client'],
      ['ClientSecret', 'secret'],
      ['Enabled', true],
    ])
  })

  test('disables an auth integration before changing its credentials', () => {
    const updates = [
      ['ClientId', 'client'],
      ['Enabled', false],
    ] as Array<[string, string | boolean]>

    assert.deepEqual(orderAuthOptionUpdates(updates, 'Enabled', false), [
      ['Enabled', false],
      ['ClientId', 'client'],
    ])
  })
})
