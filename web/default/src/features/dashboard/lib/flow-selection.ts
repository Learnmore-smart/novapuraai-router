import type {
  FlowQuotaDataItem,
  FlowUserFilterOption,
} from '@/features/dashboard/types'

export type FlowDisplayState = 'loading' | 'error' | 'empty' | 'chart'

export interface FlowResponse {
  success: boolean
  data?: FlowQuotaDataItem[]
  message?: string
}

export function requireSuccessfulFlowRows(
  response: FlowResponse,
  fallbackMessage: string
): FlowQuotaDataItem[] {
  if (!response.success) {
    throw new Error(response.message || fallbackMessage)
  }
  return response.data ?? []
}

export function flowDisplayState(options: {
  isLoading: boolean
  isError: boolean
  linkCount: number
  themeReady: boolean
}): FlowDisplayState {
  if (options.isLoading) return 'loading'
  if (options.isError) return 'error'
  if (options.linkCount === 0) return 'empty'
  if (!options.themeReady) return 'loading'
  return 'chart'
}

export function compactFlowSelectionLabel(count: number): string {
  return count > 0 ? String(count) : '*'
}

export function visibleFlowUsers(
  users: FlowUserFilterOption[],
  selectedUsers: string[]
): FlowUserFilterOption[] {
  if (selectedUsers.length === 0) return users
  const selected = new Set(selectedUsers)
  return users.filter((user) => selected.has(user.value))
}
