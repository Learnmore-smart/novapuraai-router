import assert from 'node:assert/strict'
import test from 'node:test'

import { hasConfirmedSuccessfulRequest } from './dashboard-status.ts'

test('accepts only an explicit user-scoped success signal', () => {
  assert.equal(hasConfirmedSuccessfulRequest(true), true)
  assert.equal(
    hasConfirmedSuccessfulRequest({ has_successful_request: true }),
    true
  )
})

test('does not infer success from another response shape or global metrics', () => {
  assert.equal(hasConfirmedSuccessfulRequest(false), false)
  assert.equal(hasConfirmedSuccessfulRequest({ success_rate: 100 }), false)
  assert.equal(
    hasConfirmedSuccessfulRequest([{ has_successful_request: true }]),
    false
  )
  assert.equal(hasConfirmedSuccessfulRequest(undefined), false)
})
