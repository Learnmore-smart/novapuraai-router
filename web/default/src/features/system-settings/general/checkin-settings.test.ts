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
  buildCheckinQuotaOptionUpdates,
  saveCheckinOptionUpdates,
} from './checkin-settings.ts'

type Response = { success: boolean; message?: string }

describe('check-in option persistence', () => {
  test('converts a 5-50 display-currency range to internal quota units', () => {
    assert.deepEqual(
      buildCheckinQuotaOptionUpdates(
        { minQuota: 5, maxQuota: 50 },
        { minQuota: 1000, maxQuota: 10000 },
        (amount) => amount * 500000
      ),
      [
        { key: 'checkin_setting.min_quota', value: '2500000' },
        { key: 'checkin_setting.max_quota', value: '25000000' },
      ]
    )
  })

  test('starts every changed option before completing the save', async () => {
    const started: string[] = []
    const resolvers = new Map<string, (response: Response) => void>()

    const savePromise = saveCheckinOptionUpdates(
      [
        { key: 'checkin_setting.min_quota', value: '5' },
        { key: 'checkin_setting.max_quota', value: '50' },
      ],
      (update) => {
        started.push(update.key)
        return new Promise<Response>((resolve) => {
          resolvers.set(update.key, resolve)
        })
      }
    )

    assert.deepEqual(started, [
      'checkin_setting.min_quota',
      'checkin_setting.max_quota',
    ])

    let completed = false
    void savePromise.then(() => {
      completed = true
    })

    resolvers.get('checkin_setting.min_quota')?.({ success: true })
    await Promise.resolve()
    assert.equal(completed, false)

    resolvers.get('checkin_setting.max_quota')?.({ success: true })
    await savePromise
    assert.equal(completed, true)
  })

  test('rejects when any option response reports failure', async () => {
    await assert.rejects(
      saveCheckinOptionUpdates(
        [{ key: 'checkin_setting.max_quota', value: '50' }],
        async () => ({ success: false, message: 'save rejected' })
      ),
      /save rejected/
    )
  })
})
