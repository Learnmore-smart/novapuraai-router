import { api } from '@/lib/api'

import type {
  WithdrawalQueueResponse,
  WithdrawalRequest,
  ApiResponse,
  ProcessWithdrawalPayload,
  ReverseWithdrawalPayload,
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
 *
 * When action=paid and payout_channel=stripe_connect, the backend delegates to
 * the synchronous Stripe Connect Transfer flow (service.ApproveStripeConnectWithdrawal)
 * instead of the manual paid path — the withdrawal transitions through
 * transfer_creating → awaiting_funds → payout_creating → processing → paid.
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

/**
 * Manually reverse the Stripe Transfer for an `action_required` withdrawal
 * (admin). Refunds the user's commission balance by the reversed amount and
 * transitions the withdrawal to `failed`. Body: `{ reason: "..." }`.
 *
 * Returns the updated withdrawal on success. Errors (e.g. Stripe
 * insufficient_funds, already-terminal) are surfaced via the standard
 * `success: false` envelope.
 */
export async function reverseWithdrawal(
  id: number,
  payload: ReverseWithdrawalPayload
): Promise<ApiResponse<WithdrawalRequest>> {
  const res = await api.post(
    `/api/user/withdrawal/${id}/reverse`,
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
 *
 * `payoutChannel` ("manual" | "stripe_connect") is forwarded so the backend can
 * route the request to the chosen payout flow. The legacy manual-only contract
 * is preserved when the channel is omitted.
 */
export async function requestWithdrawal(
  amountCents: number,
  payoutChannel?: string
): Promise<ApiResponse<WithdrawalRequest>> {
  const body: Record<string, unknown> = { amount_cents: amountCents }
  if (payoutChannel) {
    body.payout_channel = payoutChannel
  }
  const res = await api.post('/api/commission/withdraw', body)
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
