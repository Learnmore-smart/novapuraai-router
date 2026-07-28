import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  getFAQEntryTranslation,
  parseFAQBatch,
  toFAQTranslationLanguage,
} from './faq-json.ts'

describe('FAQ JSON batch import', () => {
  it('appends valid entries with collision-free IDs and ignores supplied IDs', () => {
    const result = parseFAQBatch(
      JSON.stringify([
        { id: 999, question: 'First question?', answer: 'First answer.' },
        { question: 'Second question?', answer: 'Second answer.' },
      ]),
      [2, 8]
    )

    assert.deepEqual(result, {
      success: true,
      entries: [
        { id: 9, question: 'First question?', answer: 'First answer.' },
        { id: 10, question: 'Second question?', answer: 'Second answer.' },
      ],
    })
  })

  it('imports one entry with all localized question and answer pairs', () => {
    const translations = {
      en: {
        question: 'Do registrations receive credits?',
        answer: 'Check the current welcome offer.',
      },
      zh: {
        question: '注册会赠送额度吗？',
        answer: '请查看当前的新用户活动说明。',
      },
      'zh-TW': {
        question: '註冊會贈送額度嗎？',
        answer: '請查看目前的新用戶活動說明。',
      },
      fr: {
        question: 'L’inscription donne-t-elle des crédits ?',
        answer: 'Consultez l’offre de bienvenue actuelle.',
      },
      ja: {
        question: '登録するとクレジットは付与されますか？',
        answer: '現在の新規登録特典をご確認ください。',
      },
      ru: {
        question: 'Начисляются ли кредиты за регистрацию?',
        answer: 'Проверьте текущее приветственное предложение.',
      },
      vi: {
        question: 'Đăng ký có được tặng hạn mức không?',
        answer: 'Hãy xem ưu đãi chào mừng hiện tại.',
      },
    }

    assert.deepEqual(
      parseFAQBatch(JSON.stringify([{ id: 999, translations }]), [4]),
      {
        success: true,
        entries: [
          {
            id: 5,
            question: translations.en.question,
            answer: translations.en.answer,
            translations,
          },
        ],
      }
    )
  })

  it('rejects a multilingual entry with unsupported language codes', () => {
    assert.deepEqual(
      parseFAQBatch(
        JSON.stringify([
          {
            translations: {
              de: { question: 'Frage', answer: 'Antwort' },
            },
          },
        ]),
        []
      ),
      { success: false, error: 'Invalid JSON data' }
    )
  })

  it('selects a localized entry and keeps legacy entries in English', () => {
    const localized = {
      id: 1,
      question: 'English question',
      answer: 'English answer',
      translations: {
        en: { question: 'English question', answer: 'English answer' },
        zh: { question: '中文问题', answer: '中文回答' },
      },
    }

    assert.deepEqual(getFAQEntryTranslation(localized, 'zh'), {
      question: '中文问题',
      answer: '中文回答',
    })
    assert.equal(getFAQEntryTranslation(localized, 'fr'), undefined)
    assert.deepEqual(
      getFAQEntryTranslation(
        { id: 2, question: 'Legacy question', answer: 'Legacy answer' },
        'en'
      ),
      { question: 'Legacy question', answer: 'Legacy answer' }
    )
  })

  it('rejects malformed JSON without entries', () => {
    assert.deepEqual(parseFAQBatch('[', []), {
      success: false,
      error: 'Invalid JSON data',
    })
  })

  it('requires a non-empty array', () => {
    assert.deepEqual(parseFAQBatch('{}', []), {
      success: false,
      error: 'FAQ JSON must be a non-empty array',
    })
    assert.deepEqual(parseFAQBatch('[]', []), {
      success: false,
      error: 'FAQ JSON must be a non-empty array',
    })
  })

  it('requires bounded non-empty question and answer strings', () => {
    assert.deepEqual(
      parseFAQBatch('[{"question":" ","answer":"Answer"}]', []),
      {
        success: false,
        error: 'Entry {{index}} requires a question and answer',
        values: { index: 1 },
      }
    )
    assert.deepEqual(
      parseFAQBatch(
        JSON.stringify([{ question: 'q'.repeat(201), answer: 'Answer' }]),
        []
      ),
      {
        success: false,
        error: 'Entry {{index}} question exceeds 200 characters',
        values: { index: 1 },
      }
    )
    assert.deepEqual(
      parseFAQBatch(
        JSON.stringify([{ question: 'Question', answer: 'a'.repeat(1001) }]),
        []
      ),
      {
        success: false,
        error: 'Entry {{index}} answer exceeds 1000 characters',
        values: { index: 1 },
      }
    )
  })
})

describe('toFAQTranslationLanguage', () => {
  it('maps interface language codes to FAQ translation language codes', () => {
    assert.equal(toFAQTranslationLanguage('zhCN'), 'zh')
    assert.equal(toFAQTranslationLanguage('zhTW'), 'zh-TW')
    assert.equal(toFAQTranslationLanguage('en'), 'en')
    assert.equal(toFAQTranslationLanguage('fr'), 'fr')
    assert.equal(toFAQTranslationLanguage('ru'), 'ru')
    assert.equal(toFAQTranslationLanguage('ja'), 'ja')
    assert.equal(toFAQTranslationLanguage('vi'), 'vi')
  })

  it('falls back to English for unknown or missing interface languages', () => {
    assert.equal(toFAQTranslationLanguage('de'), 'en')
    assert.equal(toFAQTranslationLanguage('ko'), 'en')
    assert.equal(toFAQTranslationLanguage(''), 'en')
    assert.equal(toFAQTranslationLanguage(null), 'en')
    assert.equal(toFAQTranslationLanguage(undefined), 'en')
  })
})
