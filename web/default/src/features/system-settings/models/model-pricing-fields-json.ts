import type { BulkPricingMaps } from "./model-pricing-bulk-json";
import {
  getEffectiveCompletionRatio,
  hasPositiveTokenOutput,
  isInvalidModelName,
} from "./model-pricing-core";

export const pricingJsonFields = [
  ["ModelPrice", "modelPrice"],
  ["ModelRatio", "modelRatio"],
  ["CacheRatio", "cacheRatio"],
  ["CreateCacheRatio", "createCacheRatio"],
  ["CompletionRatio", "completionRatio"],
  ["ImageRatio", "imageRatio"],
  ["AudioRatio", "audioRatio"],
  ["AudioCompletionRatio", "audioCompletionRatio"],
  ["ModelDiscount", "modelDiscount"],
] as const;

export type PricingFieldName = (typeof pricingJsonFields)[number][0];
export type PricingFieldsDrafts = Record<PricingFieldName, string>;

type NumberMap = Record<string, number>;
type ParsedNumberMap =
  | { ok: true; value: NumberMap }
  | { ok: false; error: string };

export type ApplyPricingFieldsResult =
  | { ok: true; updates: PricingFieldsDrafts }
  | { ok: false; errors: string[] };

function parseNumberMap(
  value: string,
  field: PricingFieldName,
): ParsedNumberMap {
  try {
    const parsed: unknown = JSON.parse(value);
    if (
      parsed === null ||
      typeof parsed !== "object" ||
      Array.isArray(parsed) ||
      Object.values(parsed).some(
        (entry) => typeof entry !== "number" || !Number.isFinite(entry),
      )
    ) {
      return {
        ok: false,
        error: `${field}: must be a JSON object containing finite numbers`,
      };
    }
    return { ok: true, value: parsed as NumberMap };
  } catch {
    return { ok: false, error: `${field}: invalid JSON` };
  }
}

export function exportPricingFieldsJson(
  maps: BulkPricingMaps,
  candidateModelNames: string[] = [],
  candidateModelsOnly = false,
): PricingFieldsDrafts {
  const allowedNames = new Set(candidateModelNames);
  const modelPrice = parseNumberMap(maps.modelPrice, "ModelPrice");
  const modelRatio = parseNumberMap(maps.modelRatio, "ModelRatio");
  const completionRatio = parseNumberMap(
    maps.completionRatio,
    "CompletionRatio",
  );
  let effectiveCompletionSource = maps.completionRatio;
  if (modelPrice.ok && modelRatio.ok && completionRatio.ok) {
    const effectiveCompletionValues = { ...completionRatio.value };
    for (const name of Object.keys(modelRatio.value)) {
      if (Object.hasOwn(modelPrice.value, name)) continue;
      effectiveCompletionValues[name] = getEffectiveCompletionRatio(
        name,
        effectiveCompletionValues[name],
      );
    }
    effectiveCompletionSource = JSON.stringify(
      effectiveCompletionValues,
      null,
      2,
    );
  }

  return Object.fromEntries(
    pricingJsonFields.map(([field, mapKey]) => {
      const source =
        mapKey === "completionRatio" ? effectiveCompletionSource : maps[mapKey];
      if (!candidateModelsOnly) return [field, source];

      const parsed = parseNumberMap(source, field);
      if (!parsed.ok) return [field, "{}"];
      return [
        field,
        JSON.stringify(
          Object.fromEntries(
            Object.entries(parsed.value).filter(([name]) =>
              allowedNames.has(name),
            ),
          ),
          null,
          2,
        ),
      ];
    }),
  ) as PricingFieldsDrafts;
}

export function applyPricingFieldsJson(
  maps: BulkPricingMaps,
  drafts: PricingFieldsDrafts,
  candidateModelNames: string[] = [],
  candidateModelsOnly = false,
): ApplyPricingFieldsResult {
  const allowedNames = new Set(candidateModelNames);
  const parsedDrafts = new Map<PricingFieldName, NumberMap>();
  const currentMaps = new Map<PricingFieldName, NumberMap>();
  const errors: string[] = [];

  for (const [field, mapKey] of pricingJsonFields) {
    const parsedDraft = parseNumberMap(drafts[field], field);
    if (!parsedDraft.ok) {
      errors.push(parsedDraft.error);
      continue;
    }
    parsedDrafts.set(field, parsedDraft.value);

    if (!candidateModelsOnly) continue;
    for (const name of Object.keys(parsedDraft.value)) {
      if (isInvalidModelName(name)) {
        errors.push(
          `${field}: invalid model name; trailing quote is not allowed`,
        );
        continue;
      }
      if (!allowedNames.has(name)) {
        errors.push(
          `${field}: model "${name}" is outside the allowed unset-pricing scope`,
        );
      }
    }

    const parsedCurrent = parseNumberMap(maps[mapKey], field);
    if (!parsedCurrent.ok) {
      errors.push(parsedCurrent.error);
      continue;
    }
    currentMaps.set(field, parsedCurrent.value);
  }

  for (const [field, parsed] of parsedDrafts) {
    if (!candidateModelsOnly) {
      for (const name of Object.keys(parsed)) {
        if (isInvalidModelName(name)) {
          errors.push(
            `${field}: invalid model name; trailing quote is not allowed`,
          );
        }
      }
    }
  }

  const modelPrice = parsedDrafts.get("ModelPrice") || {};
  const modelRatio = parsedDrafts.get("ModelRatio") || {};
  const completionRatio = parsedDrafts.get("CompletionRatio") || {};
  const tokenPricingNames = new Set([
    ...Object.keys(modelRatio),
    ...Object.keys(completionRatio),
  ]);
  for (const name of tokenPricingNames) {
    if (Object.hasOwn(modelPrice, name)) continue;
    if (
      !Object.hasOwn(modelRatio, name) ||
      !hasPositiveTokenOutput(modelRatio[name])
    ) {
      errors.push(
        `${name}: completion ratio requires a matching ModelRatio input`,
      );
      continue;
    }
    if (hasPositiveTokenOutput(completionRatio[name])) continue;
    errors.push(
      `${name}: positive completion output price is required for token pricing`,
    );
  }

  if (errors.length > 0) return { ok: false, errors };

  return {
    ok: true,
    updates: Object.fromEntries(
      pricingJsonFields.map(([field]) => {
        const draft = parsedDrafts.get(field) || {};
        if (!candidateModelsOnly) {
          return [field, JSON.stringify(draft, null, 2)];
        }

        const current = currentMaps.get(field) || {};
        const mergedEntries = Object.entries(current).filter(
          ([name]) => !allowedNames.has(name),
        );
        mergedEntries.push(...Object.entries(draft));
        return [
          field,
          JSON.stringify(Object.fromEntries(mergedEntries), null, 2),
        ];
      }),
    ) as PricingFieldsDrafts,
  };
}
