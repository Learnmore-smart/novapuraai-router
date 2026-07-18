import { useCallback, useState } from 'react'

import {
  appendUserMessagePair,
  applyMessageEdit,
  createRegeneratedMessages,
  removeMessageByKey,
} from '../lib'
import type { Message } from '../types'

type UsePlaygroundConversationOptions = {
  messages: Message[]
  updateMessages: (
    updater: Message[] | ((prev: Message[]) => Message[])
  ) => void
  sendChat: (messages: Message[]) => void
}

export function usePlaygroundConversation({
  messages,
  updateMessages,
  sendChat,
}: UsePlaygroundConversationOptions) {
  const [editingMessageKey, setEditingMessageKey] = useState<string | null>(
    null
  )

  const handleSendMessage = useCallback(
    (text: string) => {
      const nextMessages = appendUserMessagePair(messages, text)
      updateMessages(nextMessages)
      sendChat(nextMessages)
    },
    [messages, updateMessages, sendChat]
  )

  const handleRegenerateMessage = useCallback(
    (message: Message) => {
      const nextMessages = createRegeneratedMessages(messages, message.key)
      if (!nextMessages) return

      updateMessages(nextMessages)
      sendChat(nextMessages)
    },
    [messages, updateMessages, sendChat]
  )

  const handleEditMessage = useCallback((message: Message) => {
    setEditingMessageKey(message.key)
  }, [])

  const handleEditOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setEditingMessageKey(null)
    }
  }, [])

  const applyEdit = useCallback(
    (newContent: string, shouldSubmit: boolean) => {
      if (!editingMessageKey) return

      const editResult = applyMessageEdit(
        messages,
        editingMessageKey,
        newContent,
        shouldSubmit
      )
      if (!editResult) return

      setEditingMessageKey(null)
      updateMessages(editResult.messages)

      if (editResult.shouldSend) {
        sendChat(editResult.messages)
      }
    },
    [editingMessageKey, messages, updateMessages, sendChat]
  )

  const handleDeleteMessage = useCallback(
    (message: Message) => {
      updateMessages((previousMessages) =>
        removeMessageByKey(previousMessages, message.key)
      )
    },
    [updateMessages]
  )

  return {
    editingMessageKey,
    handleSendMessage,
    handleRegenerateMessage,
    handleEditMessage,
    handleEditOpenChange,
    applyEdit,
    handleDeleteMessage,
  }
}
