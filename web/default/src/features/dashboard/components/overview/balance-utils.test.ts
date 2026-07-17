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
import test from 'node:test'

import { getBalanceBreakdown } from './balance-utils.ts'

test('uses explicit backend cash and promotional balances', () => {
  assert.deepEqual(getBalanceBreakdown(120, 80, 40), {
    total: 120,
    cash: 80,
    promo: 40,
  })
})

test('derives cash from total minus promo for older user payloads', () => {
  assert.deepEqual(getBalanceBreakdown(120, undefined, 35), {
    total: 120,
    cash: 85,
    promo: 35,
  })
})

test('never exposes negative or non-finite balances', () => {
  assert.deepEqual(getBalanceBreakdown(-10, Number.NaN, 25), {
    total: 0,
    cash: 0,
    promo: 25,
  })
})
