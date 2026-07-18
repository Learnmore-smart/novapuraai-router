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
