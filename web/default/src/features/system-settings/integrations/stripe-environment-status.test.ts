import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import { getStripeEnvironmentReadiness } from './stripe-environment-status.ts'

describe('getStripeEnvironmentReadiness', () => {
  it('reports ready only when credentials and identifiers are complete', () => {
    assert.deepEqual(
      getStripeEnvironmentReadiness({
        secretConfigured: true,
        publishableConfigured: true,
        webhookConfigured: true,
        accountID: 'acct_123',
        productID: 'prod_123',
      }),
      { ready: true, missing: [] }
    )
  })

  it('reports every missing field', () => {
    assert.deepEqual(
      getStripeEnvironmentReadiness({
        secretConfigured: false,
        publishableConfigured: true,
        webhookConfigured: false,
        accountID: ' ',
        productID: '',
      }),
      {
        ready: false,
        missing: ['secret', 'webhook', 'account', 'product'],
      }
    )
  })
})
