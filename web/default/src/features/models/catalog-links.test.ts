import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  MODEL_CATALOG_PATH,
  MODEL_DEPLOYMENTS_PATH,
  MODEL_METADATA_PATH,
  MODEL_PRICING_EDITOR_PATH,
  getModelProfilePath,
  getModelSectionPath,
} from './catalog-links'

describe('model catalog links', () => {
  it('uses the catalog as the canonical admin section while preserving deployments', () => {
    assert.equal(MODEL_CATALOG_PATH, '/models/catalog')
    assert.equal(MODEL_METADATA_PATH, '/models/metadata')
    assert.equal(MODEL_DEPLOYMENTS_PATH, '/models/deployments')
    assert.equal(getModelSectionPath('catalog'), MODEL_CATALOG_PATH)
    assert.equal(getModelSectionPath('deployments'), MODEL_DEPLOYMENTS_PATH)
  })

  it('encodes model names in public profile paths', () => {
    assert.equal(
      getModelProfilePath('claude 3.5/sonnet'),
      '/pricing/claude%203.5%2Fsonnet'
    )
  })

  it('points administrators to the model pricing editor', () => {
    assert.equal(
      MODEL_PRICING_EDITOR_PATH,
      '/system-settings/billing/model-pricing'
    )
  })
})
