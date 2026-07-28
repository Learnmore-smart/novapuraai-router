import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { MESSAGE_ROLES, MESSAGE_STATUS } from '../../constants.ts'
import type { Message } from '../../types.ts'
import {
  computeBackoffDelay,
  decideAutoResend,
  hasPendingAutoResend,
  resetPendingMessageForRetry,
} from './auto-resend-utils.ts'

function makeAssistantMessage(overrides: Partial<Message> = {}): Message {
  return {
    key: 'msg-1',
    from: MESSAGE_ROLES.ASSISTANT,
    versions: [{ id: 'v1', content: '' }],
    status: MESSAGE_STATUS.LOADING,
    ...overrides,
  }
}

describe('computeBackoffDelay', () => {
  test('returns 1000 * 2^retryCount for the first 5 attempts', () => {
    assert.equal(computeBackoffDelay(0), 1000)
    assert.equal(computeBackoffDelay(1), 2000)
    assert.equal(computeBackoffDelay(2), 4000)
    assert.equal(computeBackoffDelay(3), 8000)
    assert.equal(computeBackoffDelay(4), 16000)
  })
})

describe('decideAutoResend', () => {
  test('returns shouldRetry=true with delay when retryCount under max', () => {
    const result = decideAutoResend({
      isAutoResendMode: true,
      autoResendEnabled: true,
      retryCount: 0,
      maxRetries: 3,
    })
    assert.deepEqual(result, { shouldRetry: true, delayMs: 1000 })
  })

  test('returns shouldRetry=false when retryCount reaches max', () => {
    const result = decideAutoResend({
      isAutoResendMode: true,
      autoResendEnabled: true,
      retryCount: 3,
      maxRetries: 3,
    })
    assert.deepEqual(result, { shouldRetry: false, delayMs: 0 })
  })

  test('returns shouldRetry=false when autoResend disabled', () => {
    const result = decideAutoResend({
      isAutoResendMode: true,
      autoResendEnabled: false,
      retryCount: 0,
      maxRetries: 3,
    })
    assert.deepEqual(result, { shouldRetry: false, delayMs: 0 })
  })

  test('returns shouldRetry=false when not in auto-resend mode', () => {
    const result = decideAutoResend({
      isAutoResendMode: false,
      autoResendEnabled: true,
      retryCount: 0,
      maxRetries: 3,
    })
    assert.deepEqual(result, { shouldRetry: false, delayMs: 0 })
  })
})

describe('hasPendingAutoResend', () => {
  test('returns true when last assistant message carries the flag', () => {
    const messages = [makeAssistantMessage({ pendingAutoResend: true })]
    assert.equal(hasPendingAutoResend(messages), true)
  })

  test('returns false when no message carries the flag', () => {
    const messages = [makeAssistantMessage()]
    assert.equal(hasPendingAutoResend(messages), false)
  })

  test('returns false for empty message list', () => {
    assert.equal(hasPendingAutoResend([]), false)
  })
})

describe('resetPendingMessageForRetry', () => {
  test('clears content/reasoning/timing and keeps pendingAutoResend + LOADING', () => {
    const original = makeAssistantMessage({
      versions: [{ id: 'v1', content: 'partial output' }],
      reasoning: {
        content: 'partial reasoning',
        duration: 2,
        startedAt: 1000,
        completedAt: 2000,
        durationMs: 1000,
      },
      isReasoningStreaming: true,
      isReasoningComplete: true,
      isContentComplete: true,
      startedAt: 1000,
      completedAt: 2000,
      durationMs: 1000,
      pendingAutoResend: true,
      status: MESSAGE_STATUS.STREAMING,
    })

    const result = resetPendingMessageForRetry(original)

    assert.equal(result.pendingAutoResend, true)
    assert.equal(result.status, MESSAGE_STATUS.LOADING)
    assert.equal(result.versions[0].content, '')
    assert.equal(result.reasoning, undefined)
    assert.equal(result.isReasoningStreaming, false)
    assert.equal(result.isReasoningComplete, false)
    assert.equal(result.isContentComplete, false)
    assert.equal(result.startedAt, undefined)
    assert.equal(result.completedAt, undefined)
    assert.equal(result.durationMs, undefined)
  })

  test('returns message unchanged when pendingAutoResend is not set', () => {
    const original = makeAssistantMessage({
      versions: [{ id: 'v1', content: 'keep me' }],
    })
    const result = resetPendingMessageForRetry(original)
    assert.equal(result, original)
  })
})
