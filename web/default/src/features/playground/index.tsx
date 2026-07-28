import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

import { PlaygroundChat } from './components/chat/playground-chat'
import { PlaygroundInput } from './components/input/playground-input'
import { ERROR_MESSAGES, MESSAGE_STATUS } from './constants'
import {
  useChatHandler,
  usePlaygroundConversation,
  usePlaygroundOptions,
  usePlaygroundState,
} from './hooks'
import { hasPendingAutoResend } from './lib'

export function Playground() {
  const { t } = useTranslation()
  const {
    config,
    parameterEnabled,
    messages,
    isLoadingMessages,
    models,
    groups,
    updateMessages,
    setModels,
    setGroups,
    updateConfig,
    updateParameterEnabled,
    clearMessages,
  } = usePlaygroundState()

  const { sendChat, stopGeneration, triggerAutoResend, isGenerating } =
    useChatHandler({
      config,
      parameterEnabled,
      messages,
      onMessageUpdate: updateMessages,
    })

  const {
    editingMessageKey,
    handleSendMessage,
    handleRegenerateMessage,
    handleEditMessage,
    handleEditOpenChange,
    applyEdit,
    handleDeleteMessage,
  } = usePlaygroundConversation({
    messages,
    updateMessages,
    sendChat,
  })

  const handleClearMessages = () => {
    handleEditOpenChange(false)
    clearMessages()
  }

  const { isLoadingModels } = usePlaygroundOptions({
    currentGroup: config.group,
    currentModel: config.model,
    setGroups,
    setModels,
    updateConfig,
  })

  const hasTriggeredAutoResendRef = useRef(false)

  useEffect(() => {
    if (isLoadingMessages || hasTriggeredAutoResendRef.current) {
      return
    }

    if (!hasPendingAutoResend(messages)) {
      return
    }

    hasTriggeredAutoResendRef.current = true

    if (!config.autoResendEnabled) {
      // Feature disabled: fall back to ERROR + Retry button (legacy behavior).
      updateMessages((prev) =>
        prev.map((message, index) =>
          index === prev.length - 1 && message.pendingAutoResend
            ? {
                ...message,
                pendingAutoResend: undefined,
                status: MESSAGE_STATUS.ERROR,
                versions: [
                  {
                    ...message.versions[0],
                    content: `${t(ERROR_MESSAGES.API_REQUEST_ERROR)}: ${t(
                      ERROR_MESSAGES.INTERRUPTED
                    )}`,
                  },
                ],
              }
            : message
        )
      )
      return
    }

    triggerAutoResend(messages)
  }, [
    isLoadingMessages,
    messages,
    config.autoResendEnabled,
    updateMessages,
    triggerAutoResend,
    t,
  ])

  return (
    <div className='relative flex size-full min-h-0 flex-col overflow-hidden'>
      {/* Full-width scroll container: scrolling works even over side whitespace */}
      <div className='flex min-h-0 flex-1 flex-col overflow-hidden'>
        <PlaygroundChat
          messages={messages}
          isLoadingMessages={isLoadingMessages}
          onRegenerateMessage={handleRegenerateMessage}
          onEditMessage={handleEditMessage}
          onDeleteMessage={handleDeleteMessage}
          onSelectPrompt={handleSendMessage}
          isGenerating={isGenerating}
          editingKey={editingMessageKey}
          onCancelEdit={handleEditOpenChange}
          onSaveEdit={(newContent) => applyEdit(newContent, false)}
          onSaveEditAndSubmit={(newContent) => applyEdit(newContent, true)}
        />
      </div>

      {/* Input area: center content and constrain to the same container width */}
      <div className='mx-auto w-full max-w-4xl'>
        <PlaygroundInput
          config={config}
          disabled={isGenerating}
          groups={groups}
          groupValue={config.group}
          isGenerating={isGenerating}
          isModelLoading={isLoadingModels}
          modelValue={config.model}
          models={models}
          onGroupChange={(value) => updateConfig('group', value)}
          onConfigChange={updateConfig}
          onClearMessages={handleClearMessages}
          onModelChange={(value) => updateConfig('model', value)}
          onParameterEnabledChange={updateParameterEnabled}
          onStop={stopGeneration}
          onSubmit={handleSendMessage}
          parameterEnabled={parameterEnabled}
          hasMessages={messages.length > 0}
        />
      </div>
    </div>
  )
}
