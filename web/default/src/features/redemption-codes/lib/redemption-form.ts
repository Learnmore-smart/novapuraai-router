import type { TFunction } from 'i18next'
import { z } from 'zod'

import {
  REDEMPTION_VALIDATION,
  getRedemptionFormErrorMessages,
  normalizeRedemptionCurrency,
  type RedemptionCurrency,
} from '../constants'
import { type RedemptionFormData, type Redemption } from '../types'

// ============================================================================
// Form Schema (use getRedemptionFormSchema(t) in components for i18n messages)
// ============================================================================

export function getRedemptionFormSchema(t: TFunction) {
  const msg = getRedemptionFormErrorMessages(t)
  return z
    .object({
      name: z
        .string()
        .min(REDEMPTION_VALIDATION.NAME_MIN_LENGTH, msg.NAME_LENGTH_INVALID)
        .max(REDEMPTION_VALIDATION.NAME_MAX_LENGTH, msg.NAME_LENGTH_INVALID),
      currency: z.enum(['usd', 'cny', 'cad'] as const),
      amount: z
        .number()
        .min(REDEMPTION_VALIDATION.AMOUNT_MIN, msg.AMOUNT_INVALID),
      max_redeems: z
        .number()
        .int()
        .min(REDEMPTION_VALIDATION.MAX_REDEEMS_MIN, msg.MAX_REDEEMS_INVALID)
        .max(REDEMPTION_VALIDATION.MAX_REDEEMS_MAX, msg.MAX_REDEEMS_INVALID),
      expired_time: z.date().optional(),
      count: z
        .number()
        .int()
        .min(REDEMPTION_VALIDATION.COUNT_MIN, msg.COUNT_INVALID)
        .max(REDEMPTION_VALIDATION.COUNT_MAX, msg.COUNT_INVALID)
        .optional(),
      key: z
        .string()
        .trim()
        .optional()
        .refine(
          (val) => !val || /^[A-Za-z0-9_-]{3,64}$/.test(val),
          msg.KEY_INVALID
        ),
    })
    .refine((data) => !data.key?.trim() || (data.count ?? 1) === 1, {
      path: ['key'],
      message: msg.KEY_REQUIRES_SINGLE,
    })
}

export type RedemptionFormValues = {
  name: string
  currency: RedemptionCurrency
  amount: number
  max_redeems: number
  expired_time?: Date
  count?: number
  key?: string
}

// ============================================================================
// Form Defaults
// ============================================================================

export const REDEMPTION_FORM_DEFAULT_VALUES: RedemptionFormValues = {
  name: '',
  currency: 'usd',
  amount: 10,
  max_redeems: 1,
  expired_time: undefined,
  count: 1,
  key: '',
}

// ============================================================================
// Form Data Transformation
// ============================================================================

/**
 * Transform form data to API payload. The backend computes the internal
 * quota from (currency, amount) using current FX rates, so we do not send
 * a `quota` field here.
 */
export function transformFormDataToPayload(
  data: RedemptionFormValues
): RedemptionFormData {
  const payload: RedemptionFormData = {
    name: data.name,
    currency: data.currency,
    amount: data.amount,
    max_redeems: data.max_redeems,
    quota: 0, // 0 tells the backend to derive from currency + amount
    expired_time: data.expired_time
      ? Math.floor(data.expired_time.getTime() / 1000)
      : 0,
    count: data.count || 1,
  }
  // Only send custom key when non-empty and count is 1 (backend rejects
  // custom key with count > 1).
  const customKey = (data.key ?? '').trim()
  if (customKey && (data.count || 1) === 1) {
    payload.key = customKey
  }
  return payload
}

/**
 * Transform redemption data to form defaults. Falls back to USD when the
 * stored currency is missing or unrecognized (legacy rows).
 */
export function transformRedemptionToFormDefaults(
  redemption: Redemption
): RedemptionFormValues {
  return {
    name: redemption.name,
    currency: normalizeRedemptionCurrency(redemption.currency),
    amount:
      redemption.amount > 0
        ? redemption.amount
        : // Legacy rows without amount: derive a USD price from quota.
          redemption.quota / 500000,
    max_redeems:
      redemption.max_redeems && redemption.max_redeems > 0
        ? redemption.max_redeems
        : 1,
    expired_time:
      redemption.expired_time > 0
        ? new Date(redemption.expired_time * 1000)
        : undefined,
    count: 1,
    key: '',
  }
}
