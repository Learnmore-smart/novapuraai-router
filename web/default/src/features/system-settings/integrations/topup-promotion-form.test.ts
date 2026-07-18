import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { createTopupPromotionFormValues } from './topup-promotion-form.ts'

describe('launch billing form mapping', () => {
  test('keeps automatic FX policy without a duplicate model-price catalog', () => {
    const values = createTopupPromotionFormValues(
      {
        default_currency: 'cny',
        auto_update_fx: true,
        fx_source: 'ecb',
        fx_updated_at: 1_752_796_800,
        reference_fx_presentment_per_usd: { cny: 6.9, usd: 1, cad: 1.36 },
        currencies: {
          cny: { enabled: true, fx_presentment_per_usd: 6.9 },
          usd: { enabled: true, fx_presentment_per_usd: 1 },
          cad: { enabled: true, fx_presentment_per_usd: 1.36 },
        },
      },
      {
        id: 1,
        name: 'Launch',
        enabled: true,
        start_at: 0,
        end_at: 0,
        global_budget_micro_usd: 0,
        reserved_promo_micro_usd: 0,
        issued_promo_micro_usd: 0,
        per_user_limit: 0,
        default_promo_expiry_days: 30,
        created_at: 0,
        updated_at: 0,
      }
    )

    assert.equal(values.autoUpdateFX, true)
    assert.equal('modelPricesJSON' in values, false)
  })
})
