import { MESSAGE_STATUS } from '../../constants'
import type { ChatCompletionResponse, Message } from '../../types'
import { parseThinkTags } from './message-reasoning-utils'
import {
  completeAssistantTiming,
  completeReasoningTiming,
  startReasoningTiming,
} from './message-timing-utils'
import {
  getCurrentVersion,
  updateCurrentVersionContent,
} from './message-utils'

export {
  isAssistantMessagePending,
  isPendingAssistantMessage,
  sanitizeMessagesOnLoad,
} from './sanitize-on-load'

/**
 * Process content chunk during streaming.
 * Separates <think> reasoning from visible content in real-time.
 * Note: versions[0].content keeps the full raw content with tags during streaming.
 */
export function processStreamingContent(
  message: Message,
  contentChunk?: string
): Message {
  const currentVersion = getCurrentVersion(message)
  const fullContent = contentChunk
    ? currentVersion.content + contentChunk
    : currentVersion.content

  if (!message.reasoning && !fullContent.includes('<think>')) {
    return {
      ...updateCurrentVersionContent(message, fullContent),
      isReasoningStreaming: false,
    }
  }

  const { reasoning, hasUnclosedTag } = parseThinkTags(fullContent)
  const finalReasoning = reasoning
    ? {
        ...startReasoningTiming(message),
        content: reasoning,
      }
    : message.reasoning

  return {
    ...updateCurrentVersionContent(message, fullContent),
    reasoning: finalReasoning,
    isReasoningStreaming: hasUnclosedTag,
  }
}

export type StreamChunkType = 'reasoning' | 'content'

function getAppendableChunk(currentContent: string, chunk: string): string {
  if (!currentContent || !chunk.startsWith(currentContent)) {
    return chunk
  }

  return chunk.slice(currentContent.length)
}

export function applyStreamingChunk(
  message: Message,
  type: StreamChunkType,
  chunk: string
): Message {
  if (message.status === MESSAGE_STATUS.ERROR) {
    return message
  }

  if (type === 'reasoning') {
    const reasoning = startReasoningTiming(message)
    const appendableChunk = getAppendableChunk(reasoning.content, chunk)

    return {
      ...message,
      reasoning: {
        ...reasoning,
        content: reasoning.content + appendableChunk,
      },
      isReasoningStreaming: true,
      status: MESSAGE_STATUS.STREAMING,
    }
  }

  const currentVersion = getCurrentVersion(message)
  const appendableChunk = getAppendableChunk(currentVersion.content, chunk)
  const contentMessage = processStreamingContent(message, appendableChunk)

  return {
    ...(contentMessage.isReasoningStreaming
      ? contentMessage
      : completeReasoningTiming(contentMessage)),
    status: MESSAGE_STATUS.STREAMING,
  }
}

/**
 * Finalize message after streaming completes.
 * Cleans content and consolidates reasoning from all sources.
 */
export function finalizeMessage(
  message: Message,
  apiReasoningContent?: string
): Message {
  const currentVersion = getCurrentVersion(message)
  const parsedThinkTags = currentVersion.content.includes('<think>')
    ? parseThinkTags(currentVersion.content)
    : undefined
  const visibleContent =
    parsedThinkTags?.visibleContent ?? currentVersion.content
  const finalReasoning =
    apiReasoningContent ||
    message.reasoning?.content ||
    parsedThinkTags?.reasoning ||
    ''

  const finalized = {
    ...updateCurrentVersionContent(message, visibleContent),
    reasoning: finalReasoning
      ? {
          ...startReasoningTiming(message),
          content: finalReasoning,
        }
      : undefined,
    isReasoningStreaming: false,
  }

  return completeReasoningTiming(finalized)
}

export function completeAssistantMessage(message: Message): Message {
  return completeAssistantTiming({
    ...finalizeMessage(message),
    status: MESSAGE_STATUS.COMPLETE,
  })
}

export function isAssistantMessageFinal(message: Message): boolean {
  return (
    message.status === MESSAGE_STATUS.COMPLETE ||
    message.status === MESSAGE_STATUS.ERROR
  )
}

type ChatCompletionChoice = ChatCompletionResponse['choices'][number]

export function hasChatCompletionChoice(
  response: ChatCompletionResponse
): boolean {
  return Boolean(response.choices?.[0])
}

export function applyChatCompletionChoice(
  message: Message,
  choice: ChatCompletionChoice
): Message {
  return completeAssistantTiming({
    ...finalizeMessage(
      updateCurrentVersionContent(message, choice.message?.content || ''),
      choice.message?.reasoning_content
    ),
    status: MESSAGE_STATUS.COMPLETE,
  })
}

export function applyChatCompletionResponse(
  message: Message,
  response: ChatCompletionResponse
): Message | null {
  const choice = response.choices?.[0]

  if (!choice) {
    return null
  }

  return applyChatCompletionChoice(message, choice)
}
