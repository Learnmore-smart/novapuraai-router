import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { getSignUpTurnstileAction } from './turnstile-action.ts'

describe('signup Turnstile action selection', () => {
  test('uses email verification before the code is sent', () => {
    assert.equal(getSignUpTurnstileAction(true, false), 'email_verification')
  })

  test('uses registration after the code is sent or when email verification is off', () => {
    assert.equal(getSignUpTurnstileAction(true, true), 'register')
    assert.equal(getSignUpTurnstileAction(false, false), 'register')
  })
})
