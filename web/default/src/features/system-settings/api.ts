import { api } from '@/lib/api'

import type {
  FetchUpstreamRatiosRequest,
  LogCleanupTask,
  RetryTransactionalEmailResponse,
  SendTransactionalEmailTestResponse,
  SESCredentialStatusResponse,
  SESCredentialUpdateRequest,
  StripeCredentialEnvironment,
  StripeCredentialStatusResponse,
  StripeCredentialUpdateRequest,
  SwitchTransactionalEmailProviderResponse,
  SystemOptionsResponse,
  SystemTaskListResponse,
  SystemTaskResponse,
  TransactionalEmailHealthResponse,
  TransactionalEmailProvider,
  UpdateOptionRequest,
  UpdateOptionResponse,
  UpstreamChannelsResponse,
  UpstreamRatiosResponse,
} from './types'

export async function getSystemOptions() {
  const res = await api.get<SystemOptionsResponse>('/api/option/')
  return res.data
}

export async function updateSystemOption(request: UpdateOptionRequest) {
  const res = await api.put<UpdateOptionResponse>('/api/option/', request)
  return res.data
}

export async function getTransactionalEmailHealth() {
  const res = await api.get<TransactionalEmailHealthResponse>(
    '/api/option/email-provider/health'
  )
  return res.data
}

export async function switchTransactionalEmailProvider(
  provider: TransactionalEmailProvider
) {
  const res = await api.put<SwitchTransactionalEmailProviderResponse>(
    '/api/option/email-provider',
    { provider }
  )
  return res.data
}

export async function retryTransactionalEmailQueue() {
  const res = await api.post<RetryTransactionalEmailResponse>(
    '/api/option/email-provider/retry-safe'
  )
  return res.data
}

export async function sendTransactionalEmailTest(recipient: string) {
  const res = await api.post<SendTransactionalEmailTestResponse>(
    '/api/option/email-provider/test',
    { recipient }
  )
  return res.data
}

export async function getTransactionalEmailSESCredentials() {
  const res = await api.get<SESCredentialStatusResponse>(
    '/api/option/email-provider/ses/credentials'
  )
  return res.data
}

export async function updateTransactionalEmailSESCredentials(
  request: SESCredentialUpdateRequest
) {
  const res = await api.put<SESCredentialStatusResponse>(
    '/api/option/email-provider/ses/credentials',
    request
  )
  return res.data
}

export async function deleteTransactionalEmailSESCredentials() {
  const res = await api.delete<SESCredentialStatusResponse>(
    '/api/option/email-provider/ses/credentials'
  )
  return res.data
}

export async function updateStripeCredentials(
  environment: StripeCredentialEnvironment,
  request: StripeCredentialUpdateRequest
) {
  const res = await api.put<StripeCredentialStatusResponse>(
    `/api/option/stripe/${environment}/credentials`,
    request
  )
  return res.data
}

export async function deleteStripeCredentials(
  environment: StripeCredentialEnvironment
) {
  const res = await api.delete<StripeCredentialStatusResponse>(
    `/api/option/stripe/${environment}/credentials`
  )
  return res.data
}

export async function startLogCleanupTask(targetTimestamp: number) {
  const res = await api.post<SystemTaskResponse<LogCleanupTask>>(
    '/api/system-task/log-cleanup',
    null,
    {
      params: { target_timestamp: targetTimestamp },
    }
  )
  return res.data
}

export async function getCurrentLogCleanupTask() {
  const res = await api.get<SystemTaskResponse<LogCleanupTask | null>>(
    '/api/system-task/current',
    {
      params: { type: 'log_cleanup' },
    }
  )
  return res.data
}

export async function getSystemTask(taskId: string) {
  const res = await api.get<SystemTaskResponse<LogCleanupTask>>(
    `/api/system-task/${taskId}`
  )
  return res.data
}

export async function listSystemTasks(limit = 20) {
  const res = await api.get<SystemTaskListResponse>('/api/system-task/list', {
    params: { limit },
  })
  return res.data
}

export async function resetModelRatios() {
  const res = await api.post<UpdateOptionResponse>(
    '/api/option/rest_model_ratio'
  )
  return res.data
}

export async function getUpstreamChannels() {
  const res = await api.get<UpstreamChannelsResponse>(
    '/api/ratio_sync/channels'
  )
  return res.data
}

export async function fetchUpstreamRatios(request: FetchUpstreamRatiosRequest) {
  const res = await api.post<UpstreamRatiosResponse>(
    '/api/ratio_sync/fetch',
    request
  )
  return res.data
}
