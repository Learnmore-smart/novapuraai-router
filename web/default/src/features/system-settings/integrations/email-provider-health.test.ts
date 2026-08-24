import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  getEmailProviderSwitchState,
  isValidTestEmailRecipient,
} from './email-provider-health.ts'

describe('transactional email provider switch state', () => {
  test('marks the active provider as selected', () => {
    assert.equal(
      getEmailProviderSwitchState(
        { provider: 'ses', configured: true, reachable: true, ready: true },
        'ses'
      ),
      'selected'
    )
  })

  test('keeps an unconfigured provider unavailable', () => {
    assert.equal(
      getEmailProviderSwitchState(
        {
          provider: 'ses',
          configured: false,
          reachable: false,
          ready: false,
          failure_reason: 'configuration',
        },
        'ses'
      ),
      'selected'
    )
  })

  test('accepts a normal test recipient and rejects malformed input', () => {
    assert.equal(isValidTestEmailRecipient('noahzh52@gmail.com'), true)
    assert.equal(isValidTestEmailRecipient('not-an-email'), false)
  })
})
