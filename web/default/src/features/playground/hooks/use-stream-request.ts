import { useCallback, useRef, useState } from 'react'
import { SSE } from 'sse.js'

import { getCommonHeaders } from '@/lib/api'

import { API_ENDPOINTS, ERROR_MESSAGES } from '../constants'
import {
  getStreamReadyStateError,
  isStreamClosedReadyState,
  isStreamDoneMessage,
  parseStreamErrorDetails,
  parseStreamMessageUpdates,
} from '../lib'
import type { ChatCompletionRequest } from '../types'

/**
 * Hook for handling streaming chat completion requests
 */
export function useStreamRequest() {
  const sseSourceRef = useRef<SSE | null>(null)
  const isStreamCompleteRef = useRef(false)
  const [isStreaming, setIsStreaming] = useState(false)

  const closeActiveStream = useCallback((source?: SSE) => {
    const streamSource = source ?? sseSourceRef.current
    streamSource?.close()

    if (!source || sseSourceRef.current === source) {
      sseSourceRef.current = null
      setIsStreaming(false)
    }
  }, [])

  const sendStreamRequest = useCallback(
    (
      payload: ChatCompletionRequest,
      onUpdate: (type: 'reasoning' | 'content', chunk: string) => void,
      onComplete: () => void,
      onError: (error: string, errorCode?: string) => void
    ) => {
      sseSourceRef.current?.close()

      const source = new SSE(API_ENDPOINTS.CHAT_COMPLETIONS, {
        headers: getCommonHeaders(),
        method: 'POST',
        payload: JSON.stringify(payload),
      })

      sseSourceRef.current = source
      isStreamCompleteRef.current = false
      setIsStreaming(true)

      const handleError = (errorMessage: string, errorCode?: string) => {
        if (!isStreamCompleteRef.current) {
          onError(errorMessage, errorCode)
          closeActiveStream(source)
        }
      }

      source.addEventListener('message', (e: MessageEvent) => {
        if (isStreamDoneMessage(e.data)) {
          isStreamCompleteRef.current = true
          closeActiveStream(source)
          onComplete()
          return
        }

        try {
          const updates = parseStreamMessageUpdates(e.data)

          for (const update of updates) {
            onUpdate(update.type, update.chunk)
          }
        } catch (error) {
          // eslint-disable-next-line no-console
          console.error('Failed to parse SSE message:', error)
          handleError(ERROR_MESSAGES.PARSE_ERROR)
        }
      })

      source.addEventListener('error', (e: Event & { data?: string }) => {
        // Only handle errors if stream didn't complete normally
        if (!isStreamClosedReadyState(source.readyState)) {
          // eslint-disable-next-line no-console
          console.error('SSE Error:', e)
          const { errorCode, errorMessage } = parseStreamErrorDetails(e.data)
          handleError(errorMessage, errorCode)
        }
      })

      source.addEventListener(
        'readystatechange',
        (e: Event & { readyState?: number }) => {
          const errorMessage = getStreamReadyStateError(e.readyState, source)

          if (errorMessage) {
            handleError(errorMessage)
          }
        }
      )

      try {
        source.stream()
      } catch (error: unknown) {
        // eslint-disable-next-line no-console
        console.error('Failed to start SSE stream:', error)
        onError(ERROR_MESSAGES.STREAM_START_ERROR)
        closeActiveStream(source)
      }
    },
    [closeActiveStream]
  )

  const stopStream = useCallback(() => {
    closeActiveStream()
  }, [closeActiveStream])

  return {
    sendStreamRequest,
    stopStream,
    isStreaming,
  }
}
