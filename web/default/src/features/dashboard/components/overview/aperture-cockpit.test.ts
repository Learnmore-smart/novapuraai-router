import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { formatLifecycleDate } from './aperture-cockpit-helpers.ts'

describe('aperture cockpit lifecycle dates', () => {
  test('does not render missing Unix timestamps as 1969 or 1970 dates', () => {
    assert.equal(formatLifecycleDate(0), null)
    assert.equal(formatLifecycleDate(-1), null)
  })
})
