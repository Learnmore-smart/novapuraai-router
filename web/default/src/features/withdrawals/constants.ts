import type { WithdrawalStatus } from './types'

// ============================================================================
// Withdrawal Status Configuration
// ============================================================================

export const WITHDRAWAL_STATUS = {
  PENDING: 'pending' as WithdrawalStatus,
  PAID: 'paid' as WithdrawalStatus,
  REJECTED: 'rejected' as WithdrawalStatus,
} as const

export const WITHDRAWAL_STATUSES = {
  [WITHDRAWAL_STATUS.PENDING]: {
    labelKey: 'Pending',
    variant: 'warning' as const,
    value: WITHDRAWAL_STATUS.PENDING,
  },
  [WITHDRAWAL_STATUS.PAID]: {
    labelKey: 'Paid',
    variant: 'success' as const,
    value: WITHDRAWAL_STATUS.PAID,
  },
  [WITHDRAWAL_STATUS.REJECTED]: {
    labelKey: 'Rejected',
    variant: 'danger' as const,
    value: WITHDRAWAL_STATUS.REJECTED,
  },
} as const

export const getWithdrawalStatusOptions = (
  t: (key: string) => string
) => [
  { label: t('All'), value: '' },
  { label: t('Pending'), value: 'pending' },
  { label: t('Paid'), value: 'paid' },
  { label: t('Rejected'), value: 'rejected' },
]

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
