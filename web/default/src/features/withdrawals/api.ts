import { api } from '@/lib/api'

import type {
  WithdrawalQueueResponse,
  WithdrawalRequest,
  ApiResponse,
  ProcessWithdrawalPayload,
  CommissionSummary,
} from './types'

// ============================================================================
// Admin Withdrawal APIs
// ============================================================================

/**
 * List all withdrawal requests (admin). Optional status filter.
 */
export async function getWithdrawalQueue(
  page: number,
  pageSize: number,
  status = ''
): Promise<WithdrawalQueueResponse> {
  const params = new URLSearchParams({
    p: String(page),
    page_size: String(pageSize),
  })
  if (status) params.set('status', status)
  const res = await api.get(`/api/commission/queue?${params.toString()}`)
  return res.data
}

/**
 * Process a withdrawal request (admin). Action is "paid" or "rejected".
 */
export async function processWithdrawal(
  id: number,
  payload: ProcessWithdrawalPayload
): Promise<ApiResponse<WithdrawalRequest>> {
  const res = await api.post(
    `/api/commission/withdrawals/${id}/process`,
    payload
  )
  return res.data
}

// ============================================================================
// User Commission APIs
// ============================================================================

/**
 * Get the current user's commission balances (pending, available, total, withdrawn).
 */
export async function getCommissionSummary(): Promise<ApiResponse<CommissionSummary>> {
  const res = await api.get('/api/commission/summary')
  return res.data
}

/**
 * Request a withdrawal (user). Amount is in USD cents.
 */
export async function requestWithdrawal(
  amountCents: number
): Promise<ApiResponse<WithdrawalRequest>> {
  const res = await api.post('/api/commission/withdraw', {
    amount_cents: amountCents,
  })
  return res.data
}

/**
 * List the current user's withdrawal history.
 */
export async function getMyWithdrawals(
  page: number,
  pageSize: number
): Promise<WithdrawalQueueResponse> {
  const params = new URLSearchParams({
    p: String(page),
    page_size: String(pageSize),
  })
  const res = await api.get(`/api/commission/withdrawals?${params.toString()}`)
  return res.data
}
