export const BILLING_CURRENCIES = ['cny', 'usd', 'cad'] as const

export type BillingCurrency = (typeof BILLING_CURRENCIES)[number]

export function isBillingCurrency(value: string): value is BillingCurrency {
  return BILLING_CURRENCIES.includes(value as BillingCurrency)
}

export function formatBillingAmount(
  amount: number,
  currency: BillingCurrency,
  maximumFractionDigits = 6
): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: currency.toUpperCase(),
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: 0,
    maximumFractionDigits,
  }).format(amount)
}
