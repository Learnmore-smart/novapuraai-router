import { MESSAGE_ROLES, MESSAGE_STATUS } from '../../constants'
import type { Message } from '../../types'
import { getMessageContent, hasMessageContent } from './message-utils'

type MessageActionState = {
  content: string
  hasContent: boolean
  isAssistant: boolean
  isLoading: boolean
  isUser: boolean
}

export function getMessageActionState(message: Message): MessageActionState {
  return {
    content: getMessageContent(message),
    hasContent: hasMessageContent(message),
    isAssistant: message.from === MESSAGE_ROLES.ASSISTANT,
    isUser: message.from === MESSAGE_ROLES.USER,
    isLoading:
      message.status === MESSAGE_STATUS.LOADING ||
      message.status === MESSAGE_STATUS.STREAMING,
  }
}

export function getMessageActionsVisibilityClass(
  alwaysVisible: boolean
): string {
  return alwaysVisible
    ? 'opacity-100'
    : 'opacity-0 group-hover:opacity-100 max-md:opacity-100'
}
