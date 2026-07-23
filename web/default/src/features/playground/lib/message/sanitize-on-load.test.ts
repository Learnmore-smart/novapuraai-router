import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { MESSAGE_ROLES, MESSAGE_STATUS } from '../../constants.ts'
import type { Message } from '../../types'
import { sanitizeMessagesOnLoad } from './sanitize-on-load.ts'

function makePendingAssistant(
  overrides: Partial<Message> = {}
): Message {
  return {
    key: 'msg-1',
    from: MESSAGE_ROLES.ASSISTANT,
    versions: [{ id: 'v1', content: '' }],
    status: MESSAGE_STATUS.LOADING,
    ...overrides,
  }
}

describe('sanitizeMessagesOnLoad', () => {
  test('marks zero-content stuck assistant with pendingAutoResend and keeps LOADING', () => {
    const messages = [makePendingAssistant()]
    const result = sanitizeMessagesOnLoad(messages)

    assert.equal(result[0].status, MESSAGE_STATUS.LOADING)
    assert.equal(result[0].pendingAutoResend, true)
  })

  test('marks partial-content stuck assistant with pendingAutoResend and keeps LOADING', () => {
    const messages = [
      makePendingAssistant({
        versions: [{ id: 'v1', content: 'partial output' }],
        status: MESSAGE_STATUS.STREAMING,
      }),
    ]
    const result = sanitizeMessagesOnLoad(messages)

    assert.equal(result[0].status, MESSAGE_STATUS.LOADING)
    assert.equal(result[0].pendingAutoResend, true)
  })

  test('returns messages unchanged when no stuck assistant exists', () => {
    const messages = [
      makePendingAssistant({ status: MESSAGE_STATUS.COMPLETE }),
    ]
    const result = sanitizeMessagesOnLoad(messages)
    assert.equal(result, messages)
  })

  test('returns empty list unchanged', () => {
    assert.deepEqual(sanitizeMessagesOnLoad([]), [])
  })

  test('only sanitizes the last pending assistant, not earlier ones', () => {
    const earlier = makePendingAssistant({
      key: 'earlier',
      status: MESSAGE_STATUS.STREAMING,
    })
    const last = makePendingAssistant({ key: 'last' })
    const result = sanitizeMessagesOnLoad([earlier, last])

    assert.equal(result[0].pendingAutoResend, undefined)
    assert.equal(result[1].pendingAutoResend, true)
  })
})
