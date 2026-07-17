/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
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
