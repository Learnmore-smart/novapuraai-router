import type { BulkPricingMaps } from './model-pricing-bulk-json'

export const pricingJsonFields = [
  ['ModelPrice', 'modelPrice'],
  ['ModelRatio', 'modelRatio'],
  ['CacheRatio', 'cacheRatio'],
  ['CreateCacheRatio', 'createCacheRatio'],
  ['CompletionRatio', 'completionRatio'],
  ['ImageRatio', 'imageRatio'],
  ['AudioRatio', 'audioRatio'],
  ['AudioCompletionRatio', 'audioCompletionRatio'],
  ['ModelDiscount', 'modelDiscount'],
] as const

export type PricingFieldName = (typeof pricingJsonFields)[number][0]
export type PricingFieldsDrafts = Record<PricingFieldName, string>

type NumberMap = Record<string, number>
type ParsedNumberMap =
  | { ok: true; value: NumberMap }
  | { ok: false; error: string }

export type ApplyPricingFieldsResult =
  | { ok: true; updates: PricingFieldsDrafts }
  | { ok: false; errors: string[] }

function parseNumberMap(
  value: string,
  field: PricingFieldName
): ParsedNumberMap {
  try {
    const parsed: unknown = JSON.parse(value)
    if (
      parsed === null ||
      typeof parsed !== 'object' ||
      Array.isArray(parsed) ||
      Object.values(parsed).some(
        (entry) => typeof entry !== 'number' || !Number.isFinite(entry)
      )
    ) {
      return {
        ok: false,
        error: `${field}: must be a JSON object containing finite numbers`,
      }
    }
    return { ok: true, value: parsed as NumberMap }
  } catch {
    return { ok: false, error: `${field}: invalid JSON` }
  }
}

export function exportPricingFieldsJson(
  maps: BulkPricingMaps,
  candidateModelNames: string[] = [],
  candidateModelsOnly = false
): PricingFieldsDrafts {
  const allowedNames = new Set(candidateModelNames)
  return Object.fromEntries(
    pricingJsonFields.map(([field, mapKey]) => {
      const source = maps[mapKey]
      if (!candidateModelsOnly) return [field, source]

      const parsed = parseNumberMap(source, field)
      if (!parsed.ok) return [field, '{}']
      return [
        field,
        JSON.stringify(
          Object.fromEntries(
            Object.entries(parsed.value).filter(([name]) =>
              allowedNames.has(name)
            )
          ),
          null,
          2
        ),
      ]
    })
  ) as PricingFieldsDrafts
}

export function applyPricingFieldsJson(
  maps: BulkPricingMaps,
  drafts: PricingFieldsDrafts,
  candidateModelNames: string[] = [],
  candidateModelsOnly = false
): ApplyPricingFieldsResult {
  const allowedNames = new Set(candidateModelNames)
  const parsedDrafts = new Map<PricingFieldName, NumberMap>()
  const currentMaps = new Map<PricingFieldName, NumberMap>()
  const errors: string[] = []

  for (const [field, mapKey] of pricingJsonFields) {
    const parsedDraft = parseNumberMap(drafts[field], field)
    if (!parsedDraft.ok) {
      errors.push(parsedDraft.error)
      continue
    }
    parsedDrafts.set(field, parsedDraft.value)

    if (!candidateModelsOnly) continue
    for (const name of Object.keys(parsedDraft.value)) {
      if (!allowedNames.has(name)) {
        errors.push(
          `${field}: model "${name}" is outside the allowed unset-pricing scope`
        )
      }
    }

    const parsedCurrent = parseNumberMap(maps[mapKey], field)
    if (!parsedCurrent.ok) {
      errors.push(parsedCurrent.error)
      continue
    }
    currentMaps.set(field, parsedCurrent.value)
  }

  if (errors.length > 0) return { ok: false, errors }

  return {
    ok: true,
    updates: Object.fromEntries(
      pricingJsonFields.map(([field]) => {
        const draft = parsedDrafts.get(field) || {}
        if (!candidateModelsOnly) {
          return [field, JSON.stringify(draft, null, 2)]
        }

        const current = currentMaps.get(field) || {}
        const mergedEntries = Object.entries(current).filter(
          ([name]) => !allowedNames.has(name)
        )
        mergedEntries.push(...Object.entries(draft))
        return [
          field,
          JSON.stringify(Object.fromEntries(mergedEntries), null, 2),
        ]
      })
    ) as PricingFieldsDrafts,
  }
}
