import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { isValidEmailSender } from './email-sender.ts'

describe('SMTP sender validation', () => {
  test('accepts a display name with a mailbox address', () => {
    assert.equal(
      isValidEmailSender('NovaPuraAI <noreply@novapuraai.com>'),
      true
    )
  })

  test('accepts a bare mailbox or an empty compatibility value', () => {
    assert.equal(isValidEmailSender('noreply@novapuraai.com'), true)
    assert.equal(isValidEmailSender(''), true)
  })

  test('rejects malformed sender values', () => {
    assert.equal(isValidEmailSender('NovaPuraAI noreply@novapuraai.com'), false)
    assert.equal(isValidEmailSender('NovaPuraAI <not-an-email>'), false)
  })
})
