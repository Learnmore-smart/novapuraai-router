import type { TFunction } from 'i18next'

import type { StatusBadgeProps } from '@/components/status-badge'

// ============================================================================
// Redemption Status Configuration
// ============================================================================

export const REDEMPTION_STATUS = {
  ENABLED: 1,
  DISABLED: 2,
  USED: 3,
} as const

export const REDEMPTION_STATUS_VALUES = Object.values(REDEMPTION_STATUS).map(
  (value) => String(value)
) as `${number}`[]

// labelKey values are i18n keys; use t(config.labelKey) in components
export const REDEMPTION_STATUSES: Record<
  number,
  Pick<StatusBadgeProps, 'variant'> & {
    labelKey: string
    value: number
  }
> = {
  [REDEMPTION_STATUS.ENABLED]: {
    labelKey: 'Unused',
    variant: 'success',
    value: REDEMPTION_STATUS.ENABLED,
  },
  [REDEMPTION_STATUS.DISABLED]: {
    labelKey: 'Disabled',
    variant: 'neutral',
    value: REDEMPTION_STATUS.DISABLED,
  },
  [REDEMPTION_STATUS.USED]: {
    labelKey: 'Used',
    variant: 'neutral',
    value: REDEMPTION_STATUS.USED,
  },
} as const

// Virtual status filter value for expired redemption codes
// Note: "Expired" is not a real DB status, it's computed from expired_time
export const REDEMPTION_FILTER_EXPIRED = 'expired'

export const REDEMPTION_FILTER_VALUES = [
  String(REDEMPTION_STATUS.ENABLED),
  String(REDEMPTION_STATUS.DISABLED),
  String(REDEMPTION_STATUS.USED),
  REDEMPTION_FILTER_EXPIRED,
] as const

// ============================================================================
// Currency Options (matches setting.BillingCurrencyCNY/USD/CAD on the backend)
// ============================================================================

export type RedemptionCurrency = 'usd' | 'cny' | 'cad'

export const REDEMPTION_CURRENCIES: RedemptionCurrency[] = ['usd', 'cny', 'cad']

export const REDEMPTION_CURRENCY_LABELS: Record<RedemptionCurrency, string> = {
  usd: 'USD',
  cny: 'CNY',
  cad: 'CAD',
}

export const REDEMPTION_CURRENCY_SYMBOLS: Record<RedemptionCurrency, string> = {
  usd: '$',
  cny: '¥',
  cad: 'C$',
}

export function isRedemptionCurrency(
  value: string
): value is RedemptionCurrency {
  return (REDEMPTION_CURRENCIES as string[]).includes(value)
}

export function normalizeRedemptionCurrency(
  value: string | undefined | null
): RedemptionCurrency {
  if (value && isRedemptionCurrency(value.toLowerCase())) {
    return value.toLowerCase() as RedemptionCurrency
  }
  return 'usd'
}

export function getRedemptionStatusOptions(t: TFunction) {
  return [
    ...Object.values(REDEMPTION_STATUSES).map((config) => ({
      label: t(config.labelKey),
      value: String(config.value),
    })),
    {
      label: t('Expired'),
      value: REDEMPTION_FILTER_EXPIRED,
    },
  ]
}

// ============================================================================
// Validation Constants
// ============================================================================

export const REDEMPTION_VALIDATION = {
  NAME_MIN_LENGTH: 1,
  NAME_MAX_LENGTH: 20,
  COUNT_MIN: 1,
  COUNT_MAX: 100,
  AMOUNT_MIN: 0.01,
  MAX_REDEEMS_MIN: 1,
  MAX_REDEEMS_MAX: 100000,
  KEY_MIN_LENGTH: 3,
  KEY_MAX_LENGTH: 64,
} as const

// ============================================================================
// Error Messages
// ============================================================================

// i18n keys; use t(ERROR_MESSAGES.xxx) when displaying. For form schema with interpolation use getRedemptionFormErrorMessages(t).
export const ERROR_MESSAGES = {
  UNEXPECTED: 'An unexpected error occurred',
  LOAD_FAILED: 'Failed to load redemption codes',
  SEARCH_FAILED: 'Failed to search redemption codes',
  CREATE_FAILED: 'Failed to create redemption code',
  UPDATE_FAILED: 'Failed to update redemption code',
  DELETE_FAILED: 'Failed to delete redemption code',
  DELETE_INVALID_FAILED: 'Failed to delete invalid redemption codes',
  STATUS_UPDATE_FAILED: 'Failed to update redemption code status',
  NAME_LENGTH_INVALID: 'Name must be between {{min}} and {{max}} characters',
  COUNT_INVALID: 'Count must be between {{min}} and {{max}}',
  EXPIRED_TIME_INVALID: 'Expired time cannot be earlier than current time',
  AMOUNT_INVALID: 'Amount must be greater than {{min}}',
  MAX_REDEEMS_INVALID:
    'Max redemption count must be between {{min}} and {{max}}',
  KEY_INVALID:
    'Custom code must be {{min}}-{{max}} characters: letters, digits, hyphens, underscores only',
  KEY_REQUIRES_SINGLE: 'Custom code can only be used when quantity is 1',
} as const

/** For form schema only: returns translated messages with interpolation. */
export function getRedemptionFormErrorMessages(t: TFunction) {
  return {
    NAME_LENGTH_INVALID: t(ERROR_MESSAGES.NAME_LENGTH_INVALID, {
      min: REDEMPTION_VALIDATION.NAME_MIN_LENGTH,
      max: REDEMPTION_VALIDATION.NAME_MAX_LENGTH,
    }),
    COUNT_INVALID: t(ERROR_MESSAGES.COUNT_INVALID, {
      min: REDEMPTION_VALIDATION.COUNT_MIN,
      max: REDEMPTION_VALIDATION.COUNT_MAX,
    }),
    EXPIRED_TIME_INVALID: t(ERROR_MESSAGES.EXPIRED_TIME_INVALID),
    AMOUNT_INVALID: t(ERROR_MESSAGES.AMOUNT_INVALID, {
      min: REDEMPTION_VALIDATION.AMOUNT_MIN,
    }),
    MAX_REDEEMS_INVALID: t(ERROR_MESSAGES.MAX_REDEEMS_INVALID, {
      min: REDEMPTION_VALIDATION.MAX_REDEEMS_MIN,
      max: REDEMPTION_VALIDATION.MAX_REDEEMS_MAX,
    }),
    KEY_INVALID: t(ERROR_MESSAGES.KEY_INVALID, {
      min: REDEMPTION_VALIDATION.KEY_MIN_LENGTH,
      max: REDEMPTION_VALIDATION.KEY_MAX_LENGTH,
    }),
    KEY_REQUIRES_SINGLE: t(ERROR_MESSAGES.KEY_REQUIRES_SINGLE),
  } as const
}

// ============================================================================
// Success Messages (i18n keys; use t(SUCCESS_MESSAGES.xxx) when displaying)
// ============================================================================

export const SUCCESS_MESSAGES = {
  REDEMPTION_CREATED: 'Redemption code(s) created successfully',
  REDEMPTION_UPDATED: 'Redemption code updated successfully',
  REDEMPTION_DELETED: 'Redemption code deleted successfully',
  REDEMPTION_ENABLED: 'Redemption code enabled successfully',
  REDEMPTION_DISABLED: 'Redemption code disabled successfully',
  COPY_SUCCESS: 'Copied to clipboard',
} as const
