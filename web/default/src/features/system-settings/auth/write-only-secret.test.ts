/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
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
