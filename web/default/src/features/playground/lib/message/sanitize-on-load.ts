import { MESSAGE_ROLES, MESSAGE_STATUS } from '../../constants.ts'
import type { Message } from '../../types'

export function isAssistantMessagePending(message: Message): boolean {
  return (
    message.status === MESSAGE_STATUS.LOADING ||
    message.status === MESSAGE_STATUS.STREAMING
  )
}

export function isPendingAssistantMessage(message?: Message): boolean {
  return Boolean(
    message?.from === MESSAGE_ROLES.ASSISTANT &&
      isAssistantMessagePending(message)
  )
}

/**
 * Sanitize messages loaded from storage.
 * Stuck loading/streaming assistant messages are marked for auto-resend
 * (pendingAutoResend: true, status kept as LOADING) instead of being
 * converted to ERROR or COMPLETE. The Playground component decides whether
 * to trigger resend or fall back to ERROR based on the autoResend config.
 */
export function sanitizeMessagesOnLoad(messages: Message[]): Message[] {
  let targetIndex = -1

  for (let i = messages.length - 1; i >= 0; i--) {
    const message = messages[i]

    if (isPendingAssistantMessage(message)) {
      targetIndex = i
      break
    }
  }

  if (targetIndex === -1) return messages

  const sanitized: Message = {
    ...messages[targetIndex],
    status: MESSAGE_STATUS.LOADING,
    isReasoningStreaming: false,
    pendingAutoResend: true,
  }

  const result = [...messages]
  result[targetIndex] = sanitized
  return result
}
