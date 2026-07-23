import { nanoid } from 'nanoid'

import { AUTO_RESEND, MESSAGE_ROLES, MESSAGE_STATUS } from '../../constants.ts'
import type { Message } from '../../types'

/**
 * Exponential backoff delay: base * 2^retryCount ms.
 * retryCount 0 -> 1000ms, 1 -> 2000ms, 2 -> 4000ms, etc.
 */
export function computeBackoffDelay(retryCount: number): number {
  return AUTO_RESEND.BASE_DELAY_MS * Math.pow(2, retryCount)
}

type DecideAutoResendInput = {
  isAutoResendMode: boolean
  autoResendEnabled: boolean
  retryCount: number
  maxRetries: number
}

type DecideAutoResendResult = {
  shouldRetry: boolean
  delayMs: number
}

/**
 * Decide whether to schedule another auto-resend attempt.
 * Only retry when in auto-resend mode, feature is enabled,
 * and retryCount has not reached the configured maximum.
 */
export function decideAutoResend(
  input: DecideAutoResendInput
): DecideAutoResendResult {
  if (
    !input.isAutoResendMode ||
    !input.autoResendEnabled ||
    input.retryCount >= input.maxRetries
  ) {
    return { shouldRetry: false, delayMs: 0 }
  }

  return {
    shouldRetry: true,
    delayMs: computeBackoffDelay(input.retryCount),
  }
}

/**
 * Whether the message list ends with a pending auto-resend marker.
 * Used by the Playground component to trigger resend after load.
 */
export function hasPendingAutoResend(messages: Message[]): boolean {
  const last = messages.at(-1)
  return Boolean(
    last?.from === MESSAGE_ROLES.ASSISTANT && last?.pendingAutoResend
  )
}

/**
 * Reset a pending-auto-resend message to a fresh LOADING state,
 * discarding any partial content/reasoning/timing from the previous attempt.
 * Returns the message unchanged if it has no pendingAutoResend flag.
 */
export function resetPendingMessageForRetry(message: Message): Message {
  if (!message.pendingAutoResend) {
    return message
  }

  return {
    key: message.key,
    from: message.from,
    versions: [{ id: nanoid(), content: '' }],
    createdAt: message.createdAt,
    reasoning: undefined,
    isReasoningStreaming: false,
    isReasoningComplete: false,
    isContentComplete: false,
    status: MESSAGE_STATUS.LOADING,
    pendingAutoResend: true,
  }
}
