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

import { resolveFaviconUrl } from './dom-utils.ts'

test('stock brand logo paths resolve to the dedicated browser favicon', () => {
  assert.equal(resolveFaviconUrl(), '/favicon.ico')
  assert.equal(resolveFaviconUrl('/logo.png'), '/favicon.ico')
  assert.equal(resolveFaviconUrl('/logo-256.webp'), '/favicon.ico')
  assert.equal(
    resolveFaviconUrl('https://example.com/logo.png'),
    'https://example.com/logo.png'
  )
})

test('custom admin brand URLs continue to control the browser favicon', () => {
  assert.equal(
    resolveFaviconUrl('/custom/acme-mark.png'),
    '/custom/acme-mark.png'
  )
  assert.equal(
    resolveFaviconUrl('https://cdn.example.com/acme.svg'),
    'https://cdn.example.com/acme.svg'
  )
})
