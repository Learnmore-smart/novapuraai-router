import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { sendChatCompletion } from '../api'
import { ERROR_MESSAGES } from '../constants'
import {
  applyStreamingChunk,
  buildChatCompletionPayload,
  completeAssistantMessage,
  decideAutoResend,
  hasChatCompletionChoice,
  hasPendingAutoResend,
  isAssistantMessageFinal,
  isAssistantMessagePending,
  parseRequestErrorDetails,
  applyChatCompletionResponse,
  resetPendingMessageForRetry,
  updateAssistantMessageWithError,
  updateLastAssistantMessage,
} from '../lib'
import type { Message, PlaygroundConfig, ParameterEnabled } from '../types'
import { useStreamRequest } from './use-stream-request'

interface UseChatHandlerOptions {
  config: PlaygroundConfig
  parameterEnabled: ParameterEnabled
  messages: Message[]
  onMessageUpdate: (updater: (prev: Message[]) => Message[]) => void
}

const KNOWN_ERROR_MESSAGES = new Set<string>(Object.values(ERROR_MESSAGES))
const STREAM_UPDATE_FLUSH_MS = 50

type PendingStreamChunks = {
  content: string
  reasoning: string
}

function mergePendingStreamChunk(
  currentChunk: string,
  nextChunk: string
): string {
  if (!currentChunk || !nextChunk.startsWith(currentChunk)) {
    return currentChunk + nextChunk
  }

  return nextChunk
}

/**
 * Hook for handling chat message sending and receiving
 */
export function useChatHandler({
  config,
  parameterEnabled,
  messages,
  onMessageUpdate,
}: UseChatHandlerOptions) {
  const { t } = useTranslation()
  const { sendStreamRequest, stopStream, isStreaming } = useStreamRequest()
  const [isRequesting, setIsRequesting] = useState(false)
  const abortControllerRef = useRef<AbortController | null>(null)
  const requestIdRef = useRef(0)
  const pendingStreamChunksRef = useRef<PendingStreamChunks>({
    content: '',
    reasoning: '',
  })
  const streamFlushTimerRef = useRef<number | null>(null)

  const autoResendRetryCountRef = useRef(-1)
  const autoResendTimerRef = useRef<number | null>(null)
  const isAutoResendModeRef = useRef(false)
  const latestMessagesRef = useRef<Message[]>(messages)
  latestMessagesRef.current = messages
  const sendChatRef = useRef<(messages: Message[]) => void>(() => {})

  const flushStreamUpdates = useCallback(() => {
    if (streamFlushTimerRef.current !== null) {
      window.clearTimeout(streamFlushTimerRef.current)
      streamFlushTimerRef.current = null
    }

    const pendingChunks = pendingStreamChunksRef.current
    if (!pendingChunks.reasoning && !pendingChunks.content) {
      return
    }

    pendingStreamChunksRef.current = { content: '', reasoning: '' }
    onMessageUpdate((prev) =>
      updateLastAssistantMessage(prev, (message) => {
        let updatedMessage = message

        if (pendingChunks.reasoning) {
          updatedMessage = applyStreamingChunk(
            updatedMessage,
            'reasoning',
            pendingChunks.reasoning
          )
        }

        if (pendingChunks.content) {
          updatedMessage = applyStreamingChunk(
            updatedMessage,
            'content',
            pendingChunks.content
          )
        }

        return updatedMessage
      })
    )
  }, [onMessageUpdate])

  const scheduleStreamFlush = useCallback(() => {
    if (streamFlushTimerRef.current !== null) {
      return
    }

    streamFlushTimerRef.current = window.setTimeout(
      flushStreamUpdates,
      STREAM_UPDATE_FLUSH_MS
    )
  }, [flushStreamUpdates])

  useEffect(
    () => () => {
      if (streamFlushTimerRef.current !== null) {
        window.clearTimeout(streamFlushTimerRef.current)
      }
    },
    []
  )

  const clearAutoResendTimer = useCallback(() => {
    if (autoResendTimerRef.current !== null) {
      window.clearTimeout(autoResendTimerRef.current)
      autoResendTimerRef.current = null
    }
  }, [])

  const exitAutoResendMode = useCallback(() => {
    clearAutoResendTimer()
    isAutoResendModeRef.current = false
    autoResendRetryCountRef.current = -1
  }, [clearAutoResendTimer])

  useEffect(
    () => () => {
      clearAutoResendTimer()
    },
    [clearAutoResendTimer]
  )

  const getDisplayError = useCallback(
    (error: string) => {
      if (KNOWN_ERROR_MESSAGES.has(error)) {
        return t(error)
      }

      const connectionClosedSuffix = `: ${ERROR_MESSAGES.CONNECTION_CLOSED}`
      if (error.endsWith(connectionClosedSuffix)) {
        return `${error.slice(0, -ERROR_MESSAGES.CONNECTION_CLOSED.length)}${t(
          ERROR_MESSAGES.CONNECTION_CLOSED
        )}`
      }

      return error
    },
    [t]
  )

  // Handle stream update
  const handleStreamUpdate = useCallback(
    (type: 'reasoning' | 'content', chunk: string) => {
      pendingStreamChunksRef.current[type] = mergePendingStreamChunk(
        pendingStreamChunksRef.current[type],
        chunk
      )
      scheduleStreamFlush()
    },
    [scheduleStreamFlush]
  )

  // Handle stream complete
  const handleStreamComplete = useCallback(() => {
    flushStreamUpdates()
    exitAutoResendMode()
    setIsRequesting(false)
    onMessageUpdate((prev) =>
      updateLastAssistantMessage(prev, (message) => {
        if (isAssistantMessageFinal(message) && !message.pendingAutoResend) {
          return message
        }
        const completed = completeAssistantMessage(message)
        return { ...completed, pendingAutoResend: undefined }
      })
    )
  }, [flushStreamUpdates, exitAutoResendMode, onMessageUpdate])

  // Handle stream error
  const handleStreamError = useCallback(
    (error: string, errorCode?: string) => {
      flushStreamUpdates()

      const decision = decideAutoResend({
        isAutoResendMode: isAutoResendModeRef.current,
        autoResendEnabled: config.autoResendEnabled,
        retryCount: autoResendRetryCountRef.current,
        maxRetries: config.autoResendMaxRetries,
      })

      if (decision.shouldRetry) {
        const nextRetryCount = autoResendRetryCountRef.current + 1
        autoResendRetryCountRef.current = nextRetryCount
        toast.info(
          t('Retrying ({{n}}/{{max}})...', {
            n: nextRetryCount,
            max: config.autoResendMaxRetries,
          })
        )
        autoResendTimerRef.current = window.setTimeout(() => {
          autoResendTimerRef.current = null
          const resetMessages = latestMessagesRef.current.map((message) =>
            message.pendingAutoResend
              ? resetPendingMessageForRetry(message)
              : message
          )
          onMessageUpdate(() => resetMessages)
          sendChatRef.current(resetMessages)
        }, decision.delayMs)
        return
      }

      exitAutoResendMode()
      setIsRequesting(false)
      const displayError = getDisplayError(error)
      toast.error(displayError)
      const errorTitle = t(ERROR_MESSAGES.API_REQUEST_ERROR)
      onMessageUpdate((prev) =>
        updateAssistantMessageWithError(
          prev,
          displayError,
          errorCode,
          errorTitle
        )
      )
    },
    [
      flushStreamUpdates,
      exitAutoResendMode,
      getDisplayError,
      onMessageUpdate,
      t,
      config.autoResendEnabled,
      config.autoResendMaxRetries,
    ]
  )

  // Send streaming chat request
  const sendStreamingChat = useCallback(
    (messages: Message[]) => {
      setIsRequesting(true)
      const payload = buildChatCompletionPayload(
        messages,
        config,
        parameterEnabled
      )
      sendStreamRequest(
        payload,
        handleStreamUpdate,
        handleStreamComplete,
        handleStreamError
      )
    },
    [
      config,
      parameterEnabled,
      sendStreamRequest,
      handleStreamUpdate,
      handleStreamComplete,
      handleStreamError,
    ]
  )

  // Send non-streaming chat request
  const sendNonStreamingChat = useCallback(
    async (messages: Message[]) => {
      const payload = buildChatCompletionPayload(
        messages,
        config,
        parameterEnabled
      )
      const requestId = requestIdRef.current + 1
      const abortController = new AbortController()

      requestIdRef.current = requestId
      abortControllerRef.current = abortController

      try {
        setIsRequesting(true)
        const response = await sendChatCompletion(
          payload,
          abortController.signal
        )
        if (abortController.signal.aborted) return

        if (!hasChatCompletionChoice(response)) {
          handleStreamError(ERROR_MESSAGES.API_REQUEST_ERROR)
          return
        }

        onMessageUpdate((prev) =>
          updateLastAssistantMessage(prev, (message) => {
            const updatedMessage = applyChatCompletionResponse(
              message,
              response
            )

            return updatedMessage ?? message
          })
        )
      } catch (error: unknown) {
        if (abortController.signal.aborted) return

        const { errorCode, errorMessage } = parseRequestErrorDetails(error)
        handleStreamError(errorMessage, errorCode)
      } finally {
        if (requestIdRef.current === requestId) {
          abortControllerRef.current = null
          setIsRequesting(false)
        }
      }
    },
    [config, parameterEnabled, onMessageUpdate, handleStreamError]
  )

  // Send chat request (stream or non-stream based on config)
  const sendChat = useCallback(
    (messages: Message[]) => {
      clearAutoResendTimer()
      if (!messages.some((m) => m.pendingAutoResend)) {
        // User-initiated send (not an auto-resend): exit auto-resend mode so a
        // later failure does not get mistaken for an auto-resend retry.
        isAutoResendModeRef.current = false
        autoResendRetryCountRef.current = -1
      }
      if (config.stream) {
        sendStreamingChat(messages)
      } else {
        sendNonStreamingChat(messages)
      }
    },
    [
      config.stream,
      sendStreamingChat,
      sendNonStreamingChat,
      clearAutoResendTimer,
    ]
  )
  sendChatRef.current = sendChat

  // Stop generation
  const stopGeneration = useCallback(() => {
    stopStream()
    exitAutoResendMode()
    flushStreamUpdates()
    abortControllerRef.current?.abort()
    abortControllerRef.current = null
    setIsRequesting(false)
    onMessageUpdate((prev) =>
      updateLastAssistantMessage(prev, (message) => {
        if (!isAssistantMessagePending(message)) {
          return message
        }
        const completed = completeAssistantMessage(message)
        return { ...completed, pendingAutoResend: undefined }
      })
    )
  }, [stopStream, exitAutoResendMode, flushStreamUpdates, onMessageUpdate])

  // Trigger the first auto-resend attempt after a page reload that recovered
  // a stuck message. Resets the pending message to a fresh LOADING state,
  // enters auto-resend mode, and calls sendChat.
  const triggerAutoResend = useCallback(
    (messages: Message[]) => {
      if (!hasPendingAutoResend(messages)) {
        return
      }

      const resetMessages = messages.map((message) =>
        message.pendingAutoResend
          ? resetPendingMessageForRetry(message)
          : message
      )

      onMessageUpdate(() => resetMessages)
      isAutoResendModeRef.current = true
      autoResendRetryCountRef.current = 0
      sendChat(resetMessages)
    },
    [onMessageUpdate, sendChat]
  )

  return {
    sendChat,
    stopGeneration,
    triggerAutoResend,
    isGenerating: isStreaming || isRequesting,
  }
}
