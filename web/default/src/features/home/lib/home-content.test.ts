import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { FAQItem } from '@/features/dashboard/types'

import {
  dedupeFAQItems,
  getHomeContentMode,
  getHomeRouteModelNames,
  getPublicModelNames,
  getRegisterPromo,
} from './home-content'

describe('built-in home content boundaries', () => {
  test('keeps custom content modes separate from the built-in page', () => {
    assert.equal(
      getHomeContentMode({ isLoaded: false, content: '', isUrl: false }),
      'loading'
    )
    assert.equal(
      getHomeContentMode({
        isLoaded: true,
        content: 'https://example.com',
        isUrl: true,
      }),
      'url'
    )
    assert.equal(
      getHomeContentMode({
        isLoaded: true,
        content: '<h1>Custom</h1>',
        isUrl: false,
      }),
      'html'
    )
    assert.equal(
      getHomeContentMode({ isLoaded: true, content: '# Custom', isUrl: false }),
      'markdown'
    )
    assert.equal(
      getHomeContentMode({ isLoaded: true, content: '', isUrl: false }),
      'built-in'
    )
  })

  test('deduplicates exact FAQ content without collapsing different answers', () => {
    const items: FAQItem[] = [
      { id: 1, question: 'How is usage billed?', answer: 'Per request.' },
      { id: 2, question: '  HOW IS USAGE BILLED? ', answer: 'Per   request.' },
      {
        id: 3,
        question: 'How is usage billed?',
        answer: 'At the end of the month.',
      },
    ]

    assert.deepEqual(dedupeFAQItems(items), [items[0], items[2]])
  })

  test('accepts only the enabled canonical registration promo', () => {
    assert.deepEqual(
      getRegisterPromo({
        register_promo_enabled: true,
        register_promo_amount: '10',
        register_promo_currency: 'CNY',
      }),
      { amount: 10, currency: 'CNY' }
    )
    assert.equal(
      getRegisterPromo({ register_promo_cny_yuan: '10' }),
      null
    )
  })

  test('does not synthesize model labels when the server catalogue is empty', () => {
    assert.deepEqual(getPublicModelNames([{ model_name: null }]), [])
  })

  test('pins homepage routes to featured models and prefers catalogue spelling', () => {
    assert.deepEqual(getHomeRouteModelNames([]), [
      'kimi-k3',
      'deepseek-v4-flash-0731',
      'deepseek-v4-pro-0831',
    ])
    assert.deepEqual(
      getHomeRouteModelNames([
        { model_name: 'inkling' },
        { model_name: ' moonshotai/kimi-k3 ' },
        { model_name: 'deepseek-v4-flash-0731' },
        { model_name: 'laguna-xs-2.1' },
      ]),
      ['moonshotai/kimi-k3', 'deepseek-v4-flash-0731', 'deepseek-v4-pro-0831']
    )
  })
})
