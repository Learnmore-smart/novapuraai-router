import assert from "node:assert/strict";
import { describe, test } from "node:test";

import {
  getInvalidUpstreamModelName,
  getTokenModelMissingOutput,
} from "./upstream-ratio-sync-helpers.ts";

describe("upstream ratio sync model names", () => {
  test("rejects model names ending in a trailing quote", () => {
    assert.equal(
      getInvalidUpstreamModelName(["valid-model", 'invalid-model"']),
      'invalid-model"',
    );
    assert.equal(getInvalidUpstreamModelName(["valid-model"]), null);
  });
});

describe("upstream ratio sync output pricing", () => {
  test("rejects token input ratios without a positive output ratio", () => {
    assert.equal(getTokenModelMissingOutput({ model: 0.1 }, {}, {}), "model");
    assert.equal(
      getTokenModelMissingOutput({ model: 0.1 }, { model: 3 }, {}),
      null,
    );
    assert.equal(
      getTokenModelMissingOutput({ model: 0.1 }, {}, { model: 0 }),
      null,
    );
  });
});
