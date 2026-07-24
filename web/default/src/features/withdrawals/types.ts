// ============================================================================
// Withdrawal Types
// ============================================================================

export type WithdrawalStatus = 'pending' | 'paid' | 'rejected'

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
