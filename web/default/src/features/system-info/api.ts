import { api } from '@/lib/api'

import type {
  SystemInstanceDeleteResponse,
  SystemInstanceListResponse,
} from './types'

export async function listSystemInstances() {
  const res = await api.get<SystemInstanceListResponse>(
    '/api/system-info/instances'
  )
  return res.data
}

export async function deleteStaleSystemInstances() {
  const res = await api.delete<SystemInstanceDeleteResponse>(
    '/api/system-info/stale-instances'
  )
  return res.data
}

export async function deleteStaleSystemInstance(nodeName: string) {
  const res = await api.delete<SystemInstanceDeleteResponse>(
    `/api/system-info/instances/${encodeURIComponent(nodeName)}`
  )
  return res.data
}
