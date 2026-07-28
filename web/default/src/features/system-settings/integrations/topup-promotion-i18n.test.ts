import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, test } from 'node:test'

const localeDir = resolve(import.meta.dirname, '../../../i18n/locales')
const promotionKeys = [
  'Manage shared currencies, active and Bank of Canada reference rates, one-unit minimums, promotion limits, and budgets.',
  'Promotion bands',
  'Shared promotion bands',
  'All calculations use integer minor units. A 19.99 payment is 1999 minor units in one Checkout line item with quantity 1.',
] as const

function loadTranslations(locale: string): Record<string, string> {
  const source = readFileSync(resolve(localeDir, `${locale}.json`), 'utf8')
  return JSON.parse(source).translation as Record<string, string>
}

describe('launch billing promotion translations', () => {
  test('localizes the promotion panel in every shipped non-English locale', () => {
    for (const locale of ['zh', 'zh-TW', 'fr', 'ja', 'ru', 'vi']) {
      const translations = loadTranslations(locale)
      for (const key of promotionKeys) {
        assert.equal(typeof translations[key], 'string', `${locale}: ${key}`)
        assert.notEqual(translations[key], key, `${locale}: ${key}`)
      }
    }
  })

  test('uses the intended Simplified Chinese promotion labels', () => {
    const translations = loadTranslations('zh')

    assert.equal(translations['Promotion bands'], '优惠档位')
    assert.equal(translations['Shared promotion bands'], '共享优惠档位')
  })
})
