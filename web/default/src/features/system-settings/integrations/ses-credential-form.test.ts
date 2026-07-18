import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import { buildSESCredentialUpdate } from './ses-credential-form.ts'

describe('SES credential form payload', () => {
  it('omits blank fields so stored credentials are preserved', () => {
    assert.deepEqual(
      buildSESCredentialUpdate({
        accessKeyId: '',
        secretAccessKey: '',
        sessionToken: '',
        clearSessionToken: false,
      }),
      {}
    )
  })

  it('sends only non-empty credential replacements', () => {
    assert.deepEqual(
      buildSESCredentialUpdate({
        accessKeyId: '  AKIA-DASHBOARD  ',
        secretAccessKey: 'replacement-secret',
        sessionToken: '',
        clearSessionToken: false,
      }),
      {
        access_key_id: 'AKIA-DASHBOARD',
        secret_access_key: 'replacement-secret',
      }
    )
  })

  it('clears the optional session token explicitly', () => {
    assert.deepEqual(
      buildSESCredentialUpdate({
        accessKeyId: '',
        secretAccessKey: '',
        sessionToken: '',
        clearSessionToken: true,
      }),
      { clear_session_token: true }
    )
  })
})
