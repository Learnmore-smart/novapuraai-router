import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import { buildSESCredentialUpdate } from './ses-credential-form.ts'

describe('SES credential form payload', () => {
  it('omits blank credential fields so stored secrets are preserved', () => {
    assert.deepEqual(
      buildSESCredentialUpdate({
        accessKeyId: '',
        secretAccessKey: '',
        sessionToken: '',
        clearSessionToken: false,
        region: 'us-east-2',
        fromAddress: 'noreply@example.com',
        initialRegion: 'us-east-2',
        initialFromAddress: 'noreply@example.com',
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
        region: 'us-east-2',
        fromAddress: 'noreply@example.com',
        initialRegion: 'us-east-2',
        initialFromAddress: 'noreply@example.com',
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
        region: 'us-east-2',
        fromAddress: 'noreply@example.com',
        initialRegion: 'us-east-2',
        initialFromAddress: 'noreply@example.com',
      }),
      { clear_session_token: true }
    )
  })

  it('includes region and verified sender only when they change', () => {
    assert.deepEqual(
      buildSESCredentialUpdate({
        accessKeyId: '',
        secretAccessKey: '',
        sessionToken: '',
        clearSessionToken: false,
        region: '  ap-southeast-1  ',
        fromAddress: 'NovaPuraAI <noreply@novapuraai.com>',
        initialRegion: 'us-east-2',
        initialFromAddress: 'old@example.com',
      }),
      {
        region: 'ap-southeast-1',
        from_address: 'NovaPuraAI <noreply@novapuraai.com>',
      }
    )
  })

  it('allows long-term IAM credentials with a blank session token', () => {
    assert.deepEqual(
      buildSESCredentialUpdate({
        accessKeyId: 'AKIAEXAMPLE',
        secretAccessKey: 'secret',
        sessionToken: '',
        clearSessionToken: false,
        region: 'us-east-2',
        fromAddress: 'noreply@example.com',
        initialRegion: '',
        initialFromAddress: '',
      }),
      {
        access_key_id: 'AKIAEXAMPLE',
        secret_access_key: 'secret',
        region: 'us-east-2',
        from_address: 'noreply@example.com',
      }
    )
  })
})
