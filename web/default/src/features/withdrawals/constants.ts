import type { StatusVariant } from '@/components/status-badge'

import type { WithdrawalStatus } from './types'

// ============================================================================
// Withdrawal Status Configuration
// ============================================================================

export const WITHDRAWAL_STATUS = {
  PENDING: 'pending' as WithdrawalStatus,
  TRANSFER_CREATING: 'transfer_creating' as WithdrawalStatus,
  AWAITING_FUNDS: 'awaiting_funds' as WithdrawalStatus,
  PAYOUT_CREATING: 'payout_creating' as WithdrawalStatus,
  PROCESSING: 'processing' as WithdrawalStatus,
  ACTION_REQUIRED: 'action_required' as WithdrawalStatus,
  FAILED: 'failed' as WithdrawalStatus,
  PAID: 'paid' as WithdrawalStatus,
  REJECTED: 'rejected' as WithdrawalStatus,
} as const

interface WithdrawalStatusConfig {
  /** i18n key (English source string) used as both the lookup key and the fallback. */
  labelKey: string
  variant: StatusVariant
  value: WithdrawalStatus
}

export const WITHDRAWAL_STATUSES: Record<
  WithdrawalStatus,
  WithdrawalStatusConfig
> = {
  pending: {
    labelKey: 'Pending',
    variant: 'warning',
    value: WITHDRAWAL_STATUS.PENDING,
  },
  transfer_creating: {
    labelKey: 'Transfer Creating',
    variant: 'info',
    value: WITHDRAWAL_STATUS.TRANSFER_CREATING,
  },
  awaiting_funds: {
    labelKey: 'Awaiting Funds',
    variant: 'info',
    value: WITHDRAWAL_STATUS.AWAITING_FUNDS,
  },
  payout_creating: {
    labelKey: 'Payout Creating',
    variant: 'info',
    value: WITHDRAWAL_STATUS.PAYOUT_CREATING,
  },
  processing: {
    labelKey: 'Processing',
    variant: 'info',
    value: WITHDRAWAL_STATUS.PROCESSING,
  },
  action_required: {
    labelKey: 'Action Required',
    variant: 'danger',
    value: WITHDRAWAL_STATUS.ACTION_REQUIRED,
  },
  failed: {
    labelKey: 'Failed',
    variant: 'danger',
    value: WITHDRAWAL_STATUS.FAILED,
  },
  paid: {
    labelKey: 'Paid',
    variant: 'success',
    value: WITHDRAWAL_STATUS.PAID,
  },
  rejected: {
    labelKey: 'Rejected',
    variant: 'neutral',
    value: WITHDRAWAL_STATUS.REJECTED,
  },
}

export const getWithdrawalStatusOptions = (t: (key: string) => string) => [
  { label: t('All'), value: '' },
  { label: t('Pending'), value: 'pending' },
  { label: t('Transfer Creating'), value: 'transfer_creating' },
  { label: t('Awaiting Funds'), value: 'awaiting_funds' },
  { label: t('Payout Creating'), value: 'payout_creating' },
  { label: t('Processing'), value: 'processing' },
  { label: t('Action Required'), value: 'action_required' },
  { label: t('Failed'), value: 'failed' },
  { label: t('Paid'), value: 'paid' },
  { label: t('Rejected'), value: 'rejected' },
]

/**
 * Non-terminal Stripe Connect statuses. While any of these are present in the
 * list, the admin review page auto-refreshes every 30 seconds so the admin
 * sees webhook-driven state transitions without manually clicking refresh.
 */
export const NON_TERMINAL_STATUSES: ReadonlySet<WithdrawalStatus> = new Set([
  WITHDRAWAL_STATUS.TRANSFER_CREATING,
  WITHDRAWAL_STATUS.AWAITING_FUNDS,
  WITHDRAWAL_STATUS.PAYOUT_CREATING,
  WITHDRAWAL_STATUS.PROCESSING,
])

/**
 * Terminal statuses — no further admin action is expected, no auto-refresh needed.
 */
export const TERMINAL_STATUSES: ReadonlySet<WithdrawalStatus> = new Set([
  WITHDRAWAL_STATUS.PAID,
  WITHDRAWAL_STATUS.REJECTED,
  WITHDRAWAL_STATUS.FAILED,
])

export function isTerminalStatus(status: WithdrawalStatus): boolean {
  return TERMINAL_STATUSES.has(status)
}

export function hasNonTerminalItems(
  items: { status: WithdrawalStatus }[]
): boolean {
  return items.some((item) => NON_TERMINAL_STATUSES.has(item.status))
}

// ============================================================================
// Payout Channels
// ============================================================================

export const PAYOUT_CHANNEL_MANUAL = 'manual'
export const PAYOUT_CHANNEL_STRIPE_CONNECT = 'stripe_connect'

export const PAYOUT_CHANNEL_OPTIONS = [
  { label: 'Manual', value: PAYOUT_CHANNEL_MANUAL },
  { label: 'Stripe Connect', value: PAYOUT_CHANNEL_STRIPE_CONNECT },
]

// ============================================================================
// Pagination
// ============================================================================

export const WITHDRAWAL_PAGE_SIZE = 20

/** Auto-refresh interval (ms) for the admin review queue when non-terminal rows exist. */
export const WITHDRAWAL_AUTO_REFRESH_MS = 30_000
