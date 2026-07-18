export const GLOBAL_MODEL_DISCOUNT_KEY = '*'

type DiscountMap = Record<string, number>

function parseDiscountMap(value: string): DiscountMap {
  const parsed: unknown = JSON.parse(value)
  if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Expected a JSON object for model discounts')
  }
  return parsed as DiscountMap
}

export function getGlobalDiscountDraft(value: string): {
  enabled: boolean
  rate: number | null
} {
  try {
    const rate = parseDiscountMap(value)[GLOBAL_MODEL_DISCOUNT_KEY]
    if (Number.isFinite(rate) && rate > 0 && rate <= 1) {
      return { enabled: true, rate }
    }
  } catch {
    // The main form owns JSON validation. Keep this reader non-throwing so an
    // unrelated draft syntax error does not crash the settings page.
  }
  return { enabled: false, rate: null }
}

export function setGlobalDiscountDraft(
  value: string,
  enabled: boolean,
  rate: number
): string {
  const discountMap = parseDiscountMap(value)
  if (enabled) {
    if (!Number.isFinite(rate) || rate <= 0 || rate > 1) {
      throw new Error('Global discount must be within (0, 1]')
    }
    discountMap[GLOBAL_MODEL_DISCOUNT_KEY] = rate
  } else {
    delete discountMap[GLOBAL_MODEL_DISCOUNT_KEY]
  }
  return JSON.stringify(discountMap, null, 2)
}
