export interface BalanceBreakdown {
  total: number
  cash: number
  promo: number
}

function nonNegativeFinite(value: number | null | undefined): number {
  const number = Number(value ?? 0)
  return Number.isFinite(number) ? Math.max(0, number) : 0
}

export function getBalanceBreakdown(
  totalValue: number | null | undefined,
  cashValue: number | null | undefined,
  promoValue: number | null | undefined
): BalanceBreakdown {
  const total = nonNegativeFinite(totalValue)
  const promo = nonNegativeFinite(promoValue)
  const explicitCash = cashValue == null ? null : Number(cashValue)
  const cash =
    explicitCash !== null && Number.isFinite(explicitCash)
      ? Math.max(0, explicitCash)
      : Math.max(0, total - promo)

  return { total, cash, promo }
}
