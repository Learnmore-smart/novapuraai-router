// ============================================================================
// Withdrawal Types
// ============================================================================

/**
 * All possible withdrawal lifecycle states.
 *
 * Legacy states (`pending`, `paid`, `rejected`) apply to manual payouts.
 * Stripe Connect states apply only when `payout_channel === 'stripe_connect'`:
 *   - `transfer_creating`: admin approved; Transfer being created on Stripe (transient)
 *   - `awaiting_funds`: Transfer created; waiting for the connected account's
 *     available USD balance to settle before the Payout can be created
 *   - `payout_creating`: funds available; Payout being created (transient)
 *   - `processing`: Payout created on Stripe; awaiting `payout.paid` / `payout.failed`
 *   - `action_required`: Payout failed (e.g. bank rejected). Stripe has disabled
 *     the external account. Admin may reverse the Transfer to refund the user.
 *   - `failed`: terminal failure; commission balance refunded
 */
export type WithdrawalStatus =
  | 'pending'
  | 'transfer_creating'
  | 'awaiting_funds'
  | 'payout_creating'
  | 'processing'
  | 'action_required'
  | 'failed'
  | 'paid'
  | 'rejected'

export interface WithdrawalRequest {
  id: number
  user_id: number
  amount_cents: number
  status: WithdrawalStatus
  payout_channel: string
  payout_tx_id: string
  admin_remark: string
  reviewed_by: number
  requested_at: number
  reviewed_at: number
  /**
   * Stripe Connect fields. Only populated when `payout_channel === 'stripe_connect'`.
   */
  stripe_account_id?: string
  stripe_transfer_id?: string
  stripe_transfer_status?: string
  stripe_transfer_amount_reversed?: number
  stripe_payout_id?: string
  stripe_payout_status?: string
  stripe_payout_attempt?: number
  last_reconcile_at?: number
  /** Free-text error from the last reconciliation attempt (Stripe webhook or reversal). */
  last_reconcile_error?: string
  created_at: number
}

export interface WithdrawalQueueResponse {
  success: boolean
  message?: string
  data?: {
    items: WithdrawalRequest[]
    total: number
    page: number
    page_size: number
  }
}

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface ProcessWithdrawalPayload {
  action: 'paid' | 'rejected'
  payout_channel?: string
  payout_tx_id?: string
  admin_remark?: string
}

export interface ReverseWithdrawalPayload {
  reason: string
}

// ============================================================================
// Commission Summary (user-side)
// ============================================================================

export interface CommissionSummary {
  pending_cents: number
  balance_cents: number
  total_cents: number
  withdrawn_cents: number
  min_withdrawal_cents: number
  freeze_days: number
  approved: boolean
}
