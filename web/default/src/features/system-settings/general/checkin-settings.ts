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
import type { UpdateOptionRequest } from '../types'

type UpdateOptionResult = {
  success: boolean
  message?: string
}

type CheckinQuotaValues = {
  minQuota: number
  maxQuota: number
}

export function buildCheckinQuotaOptionUpdates(
  displayValues: CheckinQuotaValues,
  persistedQuotaUnits: CheckinQuotaValues,
  toQuotaUnits: (amount: number) => number
): UpdateOptionRequest[] {
  const updates: UpdateOptionRequest[] = []
  const minQuotaUnits = toQuotaUnits(displayValues.minQuota)
  const maxQuotaUnits = toQuotaUnits(displayValues.maxQuota)

  if (minQuotaUnits !== persistedQuotaUnits.minQuota) {
    updates.push({
      key: 'checkin_setting.min_quota',
      value: String(minQuotaUnits),
    })
  }

  if (maxQuotaUnits !== persistedQuotaUnits.maxQuota) {
    updates.push({
      key: 'checkin_setting.max_quota',
      value: String(maxQuotaUnits),
    })
  }

  return updates
}

export async function saveCheckinOptionUpdates(
  updates: UpdateOptionRequest[],
  persist: (update: UpdateOptionRequest) => Promise<UpdateOptionResult>
): Promise<void> {
  const results = await Promise.all(updates.map((update) => persist(update)))
  const failure = results.find((result) => !result.success)
  if (failure) {
    throw new Error(failure.message)
  }
}
