import assert from 'node:assert/strict'
import test from 'node:test'

import { buildStripeCredentialUpdate } from './stripe-credential-form.ts'

test('blank Stripe credential fields preserve existing encrypted values', () => {
  assert.deepEqual(
    buildStripeCredentialUpdate({
      secretKey: '',
      publishableKey: '  pk_test_replacement  ',
      webhookSecret: '   ',
    }),
    { publishable_key: 'pk_test_replacement' }
  )
})

test('Stripe credential update includes all non-blank replacements', () => {
  assert.deepEqual(
    buildStripeCredentialUpdate({
      secretKey: 'rk_live_secret',
      publishableKey: 'pk_live_public',
      webhookSecret: 'whsec_live',
    }),
    {
      secret_key: 'rk_live_secret',
      publishable_key: 'pk_live_public',
      webhook_secret: 'whsec_live',
    }
  )
})
