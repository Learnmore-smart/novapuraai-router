import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { getEmailProviderSwitchState } from './email-provider-health.ts'

describe('transactional email provider switch state', () => {
  test('marks the active provider as selected', () => {
    assert.equal(
      getEmailProviderSwitchState(
        { provider: 'brevo', configured: true, reachable: true, ready: true },
        'brevo'
      ),
      'selected'
    )
  })

  test('allows only ready inactive providers to be selected', () => {
    assert.equal(
      getEmailProviderSwitchState(
        {
          provider: 'ses',
          configured: true,
          reachable: true,
          ready: true,
          sending_enabled: true,
          production_access: true,
        },
        'brevo'
      ),
      'available'
    )
  })

  test('allows a configured SES provider to be selected before production access', () => {
    assert.equal(
      getEmailProviderSwitchState(
        {
          provider: 'ses',
          configured: true,
          reachable: true,
          ready: false,
          sending_enabled: true,
          production_access: false,
          failure_reason: 'production_access_required',
        },
        'brevo'
      ),
      'available'
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
        'brevo'
      ),
      'unavailable'
    )
  })
})
