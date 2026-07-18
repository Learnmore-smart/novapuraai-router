import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  CHANNEL_FORM_DEFAULT_VALUES,
  transformFormDataToUpdatePayload,
} from './channel-form'

describe('channel update credential pools', () => {
  test('sends append mode with pasted keys for an existing channel', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        key: 'key-two\nkey-three',
        key_mode: 'append',
      },
      42
    )

    assert.equal(payload.key, 'key-two\nkey-three')
    assert.equal(payload.key_mode, 'append')
  })

  test('omits key mode when the write-only key field is blank', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        key: '',
        key_mode: 'replace',
      },
      42
    )

    assert.equal(payload.key, undefined)
    assert.equal(payload.key_mode, undefined)
  })
})
