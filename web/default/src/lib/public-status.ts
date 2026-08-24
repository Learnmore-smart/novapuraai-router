export interface RegisterPromo {
  amount: number
  currency: string
}

function parsePositiveAmount(value: unknown): number | null {
  let amount = Number.NaN
  if (typeof value === 'number') {
    amount = value
  } else if (typeof value === 'string' && value.trim() !== '') {
    amount = Number(value)
  }
  return Number.isFinite(amount) && amount > 0 ? amount : null
}

/** Reads the public registration offer without inventing client-side state. */
export function getRegisterPromo(
  status: Record<string, unknown> | null | undefined
): RegisterPromo | null {
  if (status?.register_promo_enabled !== true) return null

  const amount = parsePositiveAmount(status.register_promo_amount)
  const currency =
    typeof status.register_promo_currency === 'string'
      ? status.register_promo_currency.trim().toUpperCase()
      : ''
  if (amount === null || !/^[A-Z]{3}$/.test(currency)) return null

  return { amount, currency }
}

export function formatPublicCurrency(
  amount: number,
  currency: string,
  locale?: string
): string | null {
  try {
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency,
      currencyDisplay: 'narrowSymbol',
      maximumFractionDigits: 2,
    }).format(amount)
  } catch {
    return null
  }
}
