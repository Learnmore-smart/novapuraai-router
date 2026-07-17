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
import { safeJsonParse } from '../utils/json-parser'
import { formatPricingNumber } from './pricing-format'

/**
 * Bulk JSON pricing editor: converts between the per-field ratio maps stored
 * in system options and a per-model JSON document using USD prices per 1M
 * tokens (the same numbers shown in the visual editor).
 *
 * Document shape:
 *   {
 *     "gpt-4o":   { "input": 2.5, "output": 10, "cache_read": 1.25,
 *                   "cache_write": 3.13, "image_input": 2, "audio_input": 40,
 *                   "audio_output": 80, "discount": 0.9 },
 *     "dall-e-3": { "per_request": 0.04 },
 *     "old-model": null
 *   }
 * `null` removes the model from every pricing map. Models not mentioned are
 * left untouched. Expression-billed models are exported/imported by other
 * tooling and are skipped here.
 */

export type BulkPricingMaps = {
  modelPrice: string
  modelRatio: string
  cacheRatio: string
  createCacheRatio: string
  completionRatio: string
  imageRatio: string
  audioRatio: string
  audioCompletionRatio: string
  modelDiscount: string
  billingMode: string
  billingExpr: string
}

export type BulkApplyResult =
  | {
      ok: true
      updates: Record<string, string>
      applied: number
      removed: number
      skippedTiered: string[]
    }
  | { ok: false; errors: string[] }

const PRICE_FIELDS = [
  'per_request',
  'input',
  'output',
  'cache_read',
  'cache_write',
  'image_input',
  'audio_input',
  'audio_output',
  'discount',
] as const

type NumberMap = Record<string, number>

const parseMap = (value: string): NumberMap =>
  safeJsonParse<NumberMap>(value, { fallback: {}, silent: true })

const parseStringMap = (value: string): Record<string, string> =>
  safeJsonParse<Record<string, string>>(value, { fallback: {}, silent: true })

const round = (value: number): number => Number(formatPricingNumber(value))

export function exportPricingJson(maps: BulkPricingMaps): string {
  const priceMap = parseMap(maps.modelPrice)
  const ratioMap = parseMap(maps.modelRatio)
  const cacheMap = parseMap(maps.cacheRatio)
  const createCacheMap = parseMap(maps.createCacheRatio)
  const completionMap = parseMap(maps.completionRatio)
  const imageMap = parseMap(maps.imageRatio)
  const audioMap = parseMap(maps.audioRatio)
  const audioCompletionMap = parseMap(maps.audioCompletionRatio)
  const discountMap = parseMap(maps.modelDiscount)
  const billingModeMap = parseStringMap(maps.billingMode)

  const names = new Set([
    ...Object.keys(priceMap),
    ...Object.keys(ratioMap),
    ...Object.keys(cacheMap),
    ...Object.keys(createCacheMap),
    ...Object.keys(completionMap),
    ...Object.keys(imageMap),
    ...Object.keys(audioMap),
    ...Object.keys(audioCompletionMap),
    ...Object.keys(discountMap),
  ])

  const document: Record<string, Record<string, number>> = {}
  for (const name of [...names].sort((a, b) => a.localeCompare(b))) {
    if (billingModeMap[name] === 'tiered_expr') continue

    const entry: Record<string, number> = {}
    if (priceMap[name] !== undefined) {
      entry.per_request = priceMap[name]
    } else if (ratioMap[name] !== undefined) {
      const input = ratioMap[name] * 2
      entry.input = round(input)
      if (completionMap[name] !== undefined) {
        entry.output = round(completionMap[name] * input)
      }
      if (cacheMap[name] !== undefined) {
        entry.cache_read = round(cacheMap[name] * input)
      }
      if (createCacheMap[name] !== undefined) {
        entry.cache_write = round(createCacheMap[name] * input)
      }
      if (imageMap[name] !== undefined) {
        entry.image_input = round(imageMap[name] * input)
      }
      if (audioMap[name] !== undefined) {
        const audioInput = audioMap[name] * input
        entry.audio_input = round(audioInput)
        if (audioCompletionMap[name] !== undefined) {
          entry.audio_output = round(audioCompletionMap[name] * audioInput)
        }
      }
    }
    if (discountMap[name] !== undefined) {
      entry.discount = discountMap[name]
    }
    if (Object.keys(entry).length > 0) {
      document[name] = entry
    }
  }

  return JSON.stringify(document, null, 2)
}

const isFiniteNumber = (value: unknown): value is number =>
  typeof value === 'number' && Number.isFinite(value)

export function applyPricingJson(
  maps: BulkPricingMaps,
  jsonText: string
): BulkApplyResult {
  let parsed: unknown
  try {
    parsed = JSON.parse(jsonText)
  } catch (error) {
    return {
      ok: false,
      errors: [error instanceof Error ? error.message : 'Invalid JSON'],
    }
  }
  if (
    parsed === null ||
    typeof parsed !== 'object' ||
    Array.isArray(parsed)
  ) {
    return { ok: false, errors: ['Expected a JSON object keyed by model name'] }
  }

  const errors: string[] = []
  const entries = Object.entries(parsed as Record<string, unknown>)

  for (const [name, rawEntry] of entries) {
    if (name.trim() === '') {
      errors.push('Empty model name')
      continue
    }
    if (rawEntry === null) continue
    if (typeof rawEntry !== 'object' || Array.isArray(rawEntry)) {
      errors.push(`${name}: entry must be an object or null`)
      continue
    }
    const entry = rawEntry as Record<string, unknown>
    for (const key of Object.keys(entry)) {
      if (!PRICE_FIELDS.includes(key as (typeof PRICE_FIELDS)[number])) {
        errors.push(`${name}: unknown field "${key}"`)
      }
    }
    for (const key of PRICE_FIELDS) {
      if (entry[key] !== undefined && !isFiniteNumber(entry[key])) {
        errors.push(`${name}: "${key}" must be a number`)
      }
    }
    const hasPerRequest = isFiniteNumber(entry.per_request)
    const hasInput = isFiniteNumber(entry.input)
    const laneKeys = [
      'output',
      'cache_read',
      'cache_write',
      'image_input',
      'audio_input',
      'audio_output',
    ]
    const hasLanes = laneKeys.some((key) => entry[key] !== undefined)
    if (hasPerRequest && (hasInput || hasLanes)) {
      errors.push(`${name}: "per_request" cannot be combined with token prices`)
    }
    if (!hasPerRequest && hasLanes && !hasInput) {
      errors.push(`${name}: token prices require "input"`)
    }
    if (
      entry.audio_output !== undefined &&
      entry.audio_input === undefined &&
      !hasPerRequest
    ) {
      errors.push(`${name}: "audio_output" requires "audio_input"`)
    }
    if (entry.discount !== undefined && isFiniteNumber(entry.discount)) {
      const discount = entry.discount
      if (discount <= 0 || discount > 1) {
        errors.push(`${name}: "discount" must be within (0, 1]`)
      }
    }
    const negatives = PRICE_FIELDS.filter(
      (key) => isFiniteNumber(entry[key]) && (entry[key] as number) < 0
    )
    if (negatives.length > 0) {
      errors.push(`${name}: negative prices are not allowed`)
    }
    if (
      !hasPerRequest &&
      !hasInput &&
      !hasLanes &&
      entry.discount === undefined
    ) {
      errors.push(`${name}: entry has no pricing fields`)
    }
  }

  if (errors.length > 0) {
    return { ok: false, errors }
  }

  const priceMap = parseMap(maps.modelPrice)
  const ratioMap = parseMap(maps.modelRatio)
  const cacheMap = parseMap(maps.cacheRatio)
  const createCacheMap = parseMap(maps.createCacheRatio)
  const completionMap = parseMap(maps.completionRatio)
  const imageMap = parseMap(maps.imageRatio)
  const audioMap = parseMap(maps.audioRatio)
  const audioCompletionMap = parseMap(maps.audioCompletionRatio)
  const discountMap = parseMap(maps.modelDiscount)
  const billingModeMap = parseStringMap(maps.billingMode)
  const billingExprMap = parseStringMap(maps.billingExpr)

  let applied = 0
  let removed = 0
  const skippedTiered: string[] = []

  for (const [rawName, rawEntry] of entries) {
    const name = rawName.trim()

    if (rawEntry === null) {
      delete priceMap[name]
      delete ratioMap[name]
      delete cacheMap[name]
      delete createCacheMap[name]
      delete completionMap[name]
      delete imageMap[name]
      delete audioMap[name]
      delete audioCompletionMap[name]
      delete discountMap[name]
      delete billingModeMap[name]
      delete billingExprMap[name]
      removed += 1
      continue
    }

    const entry = rawEntry as Record<string, number>
    const discountOnly =
      entry.per_request === undefined && entry.input === undefined

    if (billingModeMap[name] === 'tiered_expr' && discountOnly) {
      // Discounts do not apply to expression-billed models; leave them as-is
      // instead of silently storing an inert rate.
      skippedTiered.push(name)
      continue
    }

    if (!discountOnly) {
      delete priceMap[name]
      delete ratioMap[name]
      delete cacheMap[name]
      delete createCacheMap[name]
      delete completionMap[name]
      delete imageMap[name]
      delete audioMap[name]
      delete audioCompletionMap[name]
      delete billingModeMap[name]
      delete billingExprMap[name]

      if (entry.per_request !== undefined) {
        priceMap[name] = entry.per_request
      } else {
        const input = entry.input
        ratioMap[name] = round(input / 2)
        if (entry.output !== undefined && input > 0) {
          completionMap[name] = round(entry.output / input)
        }
        if (entry.cache_read !== undefined && input > 0) {
          cacheMap[name] = round(entry.cache_read / input)
        }
        if (entry.cache_write !== undefined && input > 0) {
          createCacheMap[name] = round(entry.cache_write / input)
        }
        if (entry.image_input !== undefined && input > 0) {
          imageMap[name] = round(entry.image_input / input)
        }
        if (entry.audio_input !== undefined && input > 0) {
          audioMap[name] = round(entry.audio_input / input)
          if (entry.audio_output !== undefined && entry.audio_input > 0) {
            audioCompletionMap[name] = round(
              entry.audio_output / entry.audio_input
            )
          }
        }
      }
    }

    delete discountMap[name]
    if (entry.discount !== undefined && entry.discount < 1) {
      discountMap[name] = entry.discount
    }
    applied += 1
  }

  const stringify = (value: NumberMap | Record<string, string>) =>
    JSON.stringify(value, null, 2)

  return {
    ok: true,
    applied,
    removed,
    skippedTiered,
    updates: {
      ModelPrice: stringify(priceMap),
      ModelRatio: stringify(ratioMap),
      CacheRatio: stringify(cacheMap),
      CreateCacheRatio: stringify(createCacheMap),
      CompletionRatio: stringify(completionMap),
      ImageRatio: stringify(imageMap),
      AudioRatio: stringify(audioMap),
      AudioCompletionRatio: stringify(audioCompletionMap),
      ModelDiscount: stringify(discountMap),
      'billing_setting.billing_mode': stringify(billingModeMap),
      'billing_setting.billing_expr': stringify(billingExprMap),
    },
  }
}
