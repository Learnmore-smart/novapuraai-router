import assert from "node:assert/strict";
import { describe, test } from "node:test";

import {
  buildPreviewRows,
  createInitialLaneState,
  getEffectiveCompletionRatio,
  hasRequiredTokenOutput,
  hasPositiveTokenOutput,
  isInvalidModelName,
} from "./model-pricing-core.ts";

describe("model pricing core validation", () => {
  test("rejects trailing-quote model names", () => {
    assert.equal(isInvalidModelName('deepseek-v4-flash-0731"'), true);
    assert.equal(isInvalidModelName("deepseek-v4-flash-0731"), false);
    assert.equal(isInvalidModelName("   "), true);
    assert.equal(isInvalidModelName(" gpt-4o"), true);
  });

  test("matches backend completion defaults when a stored output ratio is absent", () => {
    assert.equal(getEffectiveCompletionRatio("gpt-4o", undefined), 4);
    assert.equal(getEffectiveCompletionRatio("gpt-4o", 3), 3);
    assert.equal(
      getEffectiveCompletionRatio("deepseek-v4-flash-0731", undefined),
      3,
    );
  });

  test("requires a positive token output price", () => {
    assert.equal(hasPositiveTokenOutput(undefined), false);
    assert.equal(hasPositiveTokenOutput(0), false);
    assert.equal(hasPositiveTokenOutput(-1), false);
    assert.equal(hasPositiveTokenOutput(0.66), true);
  });

  test("keeps per-request and expression pricing exempt while protecting visual token writes", () => {
    assert.equal(
      hasRequiredTokenOutput({ billingMode: "per-request", price: "0" }),
      true,
    );
    assert.equal(
      hasRequiredTokenOutput({ billingMode: "tiered_expr", price: "" }),
      true,
    );
    assert.equal(
      hasRequiredTokenOutput({
        billingMode: "per-token",
        completionRatio: "",
      }),
      false,
    );
    assert.equal(
      hasRequiredTokenOutput({
        billingMode: "per-token",
        completionRatio: "3",
      }),
      true,
    );
  });

  test("keeps DeepSeek 0.11 ratio and 3 completion ratio in the visual preview", () => {
    const editData = {
      name: "deepseek-v4-flash-0731",
      ratio: "0.11",
      completionRatio: "3",
    };
    const laneState = createInitialLaneState(editData);
    const rows = buildPreviewRows(
      {
        name: editData.name,
        price: "",
        ratio: editData.ratio,
        cacheRatio: "",
        createCacheRatio: "",
        completionRatio: editData.completionRatio,
        imageRatio: "",
        audioRatio: "",
        audioCompletionRatio: "",
        discount: "",
      },
      "per-token",
      "",
      "",
      laneState.promptPrice,
      laneState.prices,
      laneState.enabled,
      (key) => key,
    );

    assert.equal(rows.find((row) => row.key === "inputPrice")?.value, "$0.22");
    assert.equal(rows.find((row) => row.key === "completion")?.value, "$0.66");
  });
});
