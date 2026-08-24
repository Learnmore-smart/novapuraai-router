import * as z from "zod";

import { combineBillingExpr } from "@/features/pricing/lib/billing-expr";

import { formatPricingNumber } from "./pricing-format";

export const createModelPricingSchema = (t: (key: string) => string) =>
  z.object({
    name: z
      .string()
      .min(1, t("Model name is required"))
      .refine((value) => !isInvalidModelName(value), {
        message: t("Invalid model mapping format"),
      }),
    price: z.string().optional(),
    ratio: z.string().optional(),
    cacheRatio: z.string().optional(),
    createCacheRatio: z.string().optional(),
    completionRatio: z.string().optional(),
    imageRatio: z.string().optional(),
    audioRatio: z.string().optional(),
    audioCompletionRatio: z.string().optional(),
    discount: z
      .string()
      .optional()
      .refine((value) => isValidDiscountDraft(value), {
        message: t("Discount rate must be greater than 0 and at most 1."),
      }),
  });

export function isValidDiscountDraft(value: string | undefined): boolean {
  if (value === undefined || value === "") return true;
  const num = Number(value);
  return Number.isFinite(num) && num > 0 && num <= 1;
}

export function isInvalidModelName(value: string): boolean {
  const trimmed = value.trim();
  return trimmed.length === 0 || trimmed !== value || trimmed.endsWith('"');
}

export function hasPositiveTokenOutput(value: unknown): boolean {
  if (typeof value !== "number" && typeof value !== "string") return false;
  const numberValue = Number(value);
  return Number.isFinite(numberValue) && numberValue > 0;
}

// The backend applies a model-specific completion default when a persisted
// completion ratio is absent. Keep exports round-trippable while preserving
// the canonical DeepSeek pricing introduced by the pricing migration.
export function getEffectiveCompletionRatio(
  modelName: string,
  storedRatio: unknown,
): number {
  if (hasPositiveTokenOutput(storedRatio)) return Number(storedRatio);
  if (modelName === "deepseek-v4-flash-0731") return 3;
  if (modelName.endsWith("-all") || modelName.endsWith("-gizmo-*")) return 2;
  if (modelName.startsWith("gpt-4o")) {
    if (modelName === "gpt-4o-2024-05-13") return 3;
    if (modelName.startsWith("gpt-4o-mini-tts")) return 20;
    return 4;
  }
  if (modelName.startsWith("gpt-5")) {
    if (!modelName.includes(".")) return 8;
    if (modelName.startsWith("gpt-5.4-nano")) return 6.25;
    return 6;
  }
  if (modelName.startsWith("gpt-4.5-preview")) return 2;
  if (
    modelName.startsWith("gpt-4-turbo") ||
    modelName.endsWith("gpt-4-1106") ||
    modelName.endsWith("gpt-4-1105")
  ) {
    return 3;
  }
  if (modelName.startsWith("gpt-")) return 2;
  if (modelName.startsWith("o1") || modelName.startsWith("o3")) return 4;
  if (modelName === "chatgpt-4o-latest") return 3;
  if (
    modelName.includes("claude-3") ||
    modelName.includes("claude-sonnet-4") ||
    modelName.includes("claude-opus-4") ||
    modelName.includes("claude-haiku-4")
  ) {
    return 5;
  }
  if (modelName.startsWith("gpt-3.5")) {
    if (modelName === "gpt-3.5-turbo" || modelName.endsWith("0125")) return 3;
    if (modelName.endsWith("1106")) return 2;
    return 4 / 3;
  }
  if (modelName.startsWith("mistral-")) return 3;
  if (modelName.startsWith("gemini-1.5")) return 4;
  if (modelName.startsWith("gemini-2.0")) return 4;
  if (modelName.startsWith("gemini-2.5-pro")) return 8;
  if (modelName.startsWith("gemini-2.5-flash-preview")) {
    return modelName.endsWith("-nothinking") ? 4 : 3.5 / 0.15;
  }
  if (modelName.startsWith("gemini-2.5-flash-lite")) return 4;
  if (modelName.startsWith("gemini-2.5-flash")) return 2.5 / 0.3;
  if (modelName.startsWith("gemini-robotics-er-1.5")) return 2.5 / 0.3;
  if (modelName.startsWith("gemini-3-pro-image")) return 60;
  if (modelName.startsWith("gemini-3-pro")) return 6;
  if (modelName.startsWith("gemini-")) return 4;
  if (modelName.startsWith("command")) {
    if (modelName === "command-r") return 3;
    if (modelName === "command-r-plus") return 5;
    return 4;
  }
  if (
    modelName.startsWith("ERNIE-Speed-") ||
    modelName.startsWith("ERNIE-Lite-") ||
    modelName.startsWith("ERNIE-Character") ||
    modelName.startsWith("ERNIE-Functions")
  ) {
    return 2;
  }
  if (modelName.startsWith("grok-4.5") || modelName.startsWith("grok-4")) {
    return 3;
  }
  if (modelName === "llama2-70b-4096") return 0.8 / 0.64;
  if (modelName === "llama3-8b-8192") return 2;
  if (modelName === "llama3-70b-8192") return 0.79 / 0.59;
  return 1;
}

export type ModelPricingFormValues = z.infer<
  ReturnType<typeof createModelPricingSchema>
>;

export type PricingMode = "per-token" | "per-request" | "tiered_expr";

export type LaneKey =
  | "completion"
  | "cache"
  | "createCache"
  | "image"
  | "audioInput"
  | "audioOutput";

export type ModelRatioData = {
  name: string;
  price?: string;
  ratio?: string;
  cacheRatio?: string;
  createCacheRatio?: string;
  completionRatio?: string;
  imageRatio?: string;
  audioRatio?: string;
  audioCompletionRatio?: string;
  discount?: string;
  billingMode?: PricingMode;
  billingExpr?: string;
  requestRuleExpr?: string;
};

export function hasRequiredTokenOutput(
  data: Pick<ModelRatioData, "billingMode" | "price" | "completionRatio">,
): boolean {
  if (
    data.billingMode === "per-request" ||
    data.billingMode === "tiered_expr"
  ) {
    return true;
  }

  // Older snapshots represent per-request pricing with no billing mode but a
  // ModelPrice value. Keep those writes exempt from token-output validation.
  if (data.billingMode === undefined && hasValue(data.price)) return true;

  return hasPositiveTokenOutput(data.completionRatio);
}

export type PreviewRow = {
  key: string;
  label: string;
  value: string;
  multiline?: boolean;
};

export const numericDraftRegex = /^(\d+(\.\d*)?|\.\d*)?$/;

export const EMPTY_LANE_PRICES: Record<LaneKey, string> = {
  completion: "",
  cache: "",
  createCache: "",
  image: "",
  audioInput: "",
  audioOutput: "",
};

export const EMPTY_LANE_ENABLED: Record<LaneKey, boolean> = {
  completion: false,
  cache: false,
  createCache: false,
  image: false,
  audioInput: false,
  audioOutput: false,
};

export const ratioFieldByLane: Record<LaneKey, keyof ModelPricingFormValues> = {
  completion: "completionRatio",
  cache: "cacheRatio",
  createCache: "createCacheRatio",
  image: "imageRatio",
  audioInput: "audioRatio",
  audioOutput: "audioCompletionRatio",
};

export const laneConfigs: Array<{
  key: LaneKey;
  titleKey: string;
  descriptionKey: string;
  placeholder: string;
}> = [
  {
    key: "completion",
    titleKey: "Completion price",
    descriptionKey: "Output token price for generated tokens.",
    placeholder: "15",
  },
  {
    key: "cache",
    titleKey: "Cache read price",
    descriptionKey: "Token price for cache reads.",
    placeholder: "0.3",
  },
  {
    key: "createCache",
    titleKey: "Cache write price",
    descriptionKey: "Token price for creating cache entries.",
    placeholder: "3.75",
  },
  {
    key: "image",
    titleKey: "Image input price",
    descriptionKey: "Token price for image input.",
    placeholder: "2.5",
  },
  {
    key: "audioInput",
    titleKey: "Audio input price",
    descriptionKey: "Token price for audio input.",
    placeholder: "3.81",
  },
  {
    key: "audioOutput",
    titleKey: "Audio output price",
    descriptionKey: "Token price for audio output.",
    placeholder: "15.11",
  },
];

export function hasValue(value: unknown): boolean {
  return (
    value !== "" && value !== null && value !== undefined && value !== false
  );
}

export function toNumberOrNull(value: unknown): number | null {
  if (!hasValue(value) && value !== 0) return null;
  const num = Number(value);
  return Number.isFinite(num) ? num : null;
}

function ratioToBasePrice(ratio: unknown): string {
  const num = toNumberOrNull(ratio);
  if (num === null) return "";
  return formatPricingNumber(num * 2);
}

function deriveLanePrice(
  ratio: unknown,
  denominator: unknown,
  fallback = "",
): string {
  const ratioNumber = toNumberOrNull(ratio);
  const denominatorNumber = toNumberOrNull(denominator);
  if (ratioNumber === null || denominatorNumber === null) return fallback;
  return formatPricingNumber(ratioNumber * denominatorNumber);
}

export function createInitialLaneState(data?: ModelRatioData | null) {
  if (!data) {
    return {
      promptPrice: "",
      prices: { ...EMPTY_LANE_PRICES },
      enabled: { ...EMPTY_LANE_ENABLED },
    };
  }

  const promptPrice = ratioToBasePrice(data.ratio);
  const effectiveCompletionRatio =
    hasValue(data.ratio) && !hasValue(data.price)
      ? getEffectiveCompletionRatio(data.name, data.completionRatio)
      : data.completionRatio;
  const audioInputPrice = deriveLanePrice(data.audioRatio, promptPrice);
  const prices: Record<LaneKey, string> = {
    completion: deriveLanePrice(effectiveCompletionRatio, promptPrice),
    cache: deriveLanePrice(data.cacheRatio, promptPrice),
    createCache: deriveLanePrice(data.createCacheRatio, promptPrice),
    image: deriveLanePrice(data.imageRatio, promptPrice),
    audioInput: audioInputPrice,
    audioOutput: deriveLanePrice(data.audioCompletionRatio, audioInputPrice),
  };

  return {
    promptPrice,
    prices,
    enabled: {
      completion: hasValue(effectiveCompletionRatio),
      cache: hasValue(data.cacheRatio),
      createCache: hasValue(data.createCacheRatio),
      image: hasValue(data.imageRatio),
      audioInput: hasValue(data.audioRatio),
      audioOutput: hasValue(data.audioCompletionRatio),
    },
  };
}

export function buildPreviewRows(
  values: ModelPricingFormValues,
  mode: PricingMode,
  billingExpr: string,
  requestRuleExpr: string,
  promptPrice: string,
  lanePrices: Record<LaneKey, string>,
  laneEnabled: Record<LaneKey, boolean>,
  t: (key: string) => string,
): PreviewRow[] {
  if (mode === "tiered_expr") {
    const effectiveExpr = combineBillingExpr(billingExpr, requestRuleExpr);
    return [
      { key: "mode", label: "BillingMode", value: "tiered_expr" },
      {
        key: "expr",
        label: t("Expression"),
        value: effectiveExpr || t("Empty"),
        multiline: true,
      },
    ];
  }

  const discountNumber = toNumberOrNull(values.discount);
  const activeDiscount =
    discountNumber !== null && discountNumber > 0 && discountNumber < 1
      ? discountNumber
      : null;
  const discountRows = (basePrice: string): PreviewRow[] => {
    if (activeDiscount === null) return [];
    const rows: PreviewRow[] = [
      {
        key: "discount",
        label: t("Discount rate"),
        value: `×${values.discount} (-${Math.round((1 - activeDiscount) * 100)}%)`,
      },
    ];
    const baseNumber = toNumberOrNull(basePrice);
    if (baseNumber !== null) {
      rows.push({
        key: "discountedPrice",
        label: t("Discounted price"),
        value: `$${formatPricingNumber(baseNumber * activeDiscount)}`,
      });
    }
    return rows;
  };

  if (mode === "per-request") {
    return [
      {
        key: "price",
        label: "ModelPrice",
        value: values.price || t("Empty"),
      },
      ...discountRows(values.price || ""),
    ];
  }

  return [
    {
      key: "inputPrice",
      label: t("Input price"),
      value: promptPrice ? `$${promptPrice}` : t("Empty"),
    },
    ...discountRows(promptPrice),
    {
      key: "completion",
      label: t("Completion price"),
      value:
        laneEnabled.completion && lanePrices.completion
          ? `$${lanePrices.completion}`
          : t("Empty"),
    },
    {
      key: "cache",
      label: t("Cache read price"),
      value:
        laneEnabled.cache && lanePrices.cache
          ? `$${lanePrices.cache}`
          : t("Empty"),
    },
    {
      key: "createCache",
      label: t("Cache write price"),
      value:
        laneEnabled.createCache && lanePrices.createCache
          ? `$${lanePrices.createCache}`
          : t("Empty"),
    },
    {
      key: "image",
      label: t("Image input price"),
      value:
        laneEnabled.image && lanePrices.image
          ? `$${lanePrices.image}`
          : t("Empty"),
    },
    {
      key: "audio",
      label: t("Audio input price"),
      value:
        laneEnabled.audioInput && lanePrices.audioInput
          ? `$${lanePrices.audioInput}`
          : t("Empty"),
    },
    {
      key: "audioCompletion",
      label: t("Audio output price"),
      value:
        laneEnabled.audioOutput && lanePrices.audioOutput
          ? `$${lanePrices.audioOutput}`
          : t("Empty"),
    },
  ];
}
