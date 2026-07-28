export const FAQ_TRANSLATION_LANGUAGES = [
  'en',
  'zh',
  'zh-TW',
  'fr',
  'ja',
  'ru',
  'vi',
] as const

export type FAQTranslationLanguage = (typeof FAQ_TRANSLATION_LANGUAGES)[number]

export type FAQTranslation = {
  question: string
  answer: string
}

export type FAQEntry = {
  id: number
  question: string
  answer: string
  translations?: Partial<Record<FAQTranslationLanguage, FAQTranslation>>
}

export function getFAQEntryTranslation(
  entry: FAQEntry,
  language: FAQTranslationLanguage
): FAQTranslation | undefined {
  if (entry.translations) {
    return entry.translations[language]
  }
  if (language === 'en') {
    return { question: entry.question, answer: entry.answer }
  }
  return undefined
}

/**
 * Convert an i18next interface language code (such as `zhCN` / `zhTW`) into the
 * FAQ translation language code used by the backend FAQ payload (`zh` /
 * `zh-TW`). Any unknown value falls back to `en`, matching the top-level
 * `question` / `answer` fallback already stored on every FAQ entry.
 */
export function toFAQTranslationLanguage(
  interfaceLanguage: string | null | undefined
): FAQTranslationLanguage {
  if (!interfaceLanguage) return 'en'
  if (interfaceLanguage === 'zhCN') return 'zh'
  if (interfaceLanguage === 'zhTW') return 'zh-TW'
  return FAQ_TRANSLATION_LANGUAGES.includes(
    interfaceLanguage as FAQTranslationLanguage
  )
    ? (interfaceLanguage as FAQTranslationLanguage)
    : 'en'
}

type FAQBatchResult =
  | { success: true; entries: FAQEntry[] }
  | {
      success: false
      error: string
      values?: { index: number }
    }

export function parseFAQBatch(
  source: string,
  existingIds: number[]
): FAQBatchResult {
  let parsed: unknown
  try {
    parsed = JSON.parse(source)
  } catch {
    return { success: false, error: 'Invalid JSON data' }
  }

  if (!Array.isArray(parsed) || parsed.length === 0) {
    return { success: false, error: 'FAQ JSON must be a non-empty array' }
  }

  const firstId = Math.max(...existingIds, 0) + 1
  const entries: FAQEntry[] = []
  for (const [index, item] of parsed.entries()) {
    if (typeof item !== 'object' || item === null) {
      return {
        success: false,
        error: 'Entry {{index}} requires a question and answer',
        values: { index: index + 1 },
      }
    }

    const candidate = item as Record<string, unknown>
    if (candidate.translations !== undefined) {
      if (
        typeof candidate.translations !== 'object' ||
        candidate.translations === null ||
        Array.isArray(candidate.translations)
      ) {
        return { success: false, error: 'Invalid JSON data' }
      }

      const translations: Partial<
        Record<FAQTranslationLanguage, FAQTranslation>
      > = {}
      for (const [language, translation] of Object.entries(
        candidate.translations
      )) {
        if (
          !FAQ_TRANSLATION_LANGUAGES.includes(
            language as FAQTranslationLanguage
          ) ||
          typeof translation !== 'object' ||
          translation === null ||
          Array.isArray(translation)
        ) {
          return { success: false, error: 'Invalid JSON data' }
        }

        const localized = translation as Record<string, unknown>
        const question =
          typeof localized.question === 'string'
            ? localized.question.trim()
            : ''
        const answer =
          typeof localized.answer === 'string' ? localized.answer.trim() : ''
        if (!question || !answer) {
          return {
            success: false,
            error: 'Entry {{index}} requires a question and answer',
            values: { index: index + 1 },
          }
        }
        if (question.length > 200) {
          return {
            success: false,
            error: 'Entry {{index}} question exceeds 200 characters',
            values: { index: index + 1 },
          }
        }
        if (answer.length > 1000) {
          return {
            success: false,
            error: 'Entry {{index}} answer exceeds 1000 characters',
            values: { index: index + 1 },
          }
        }
        translations[language as FAQTranslationLanguage] = { question, answer }
      }

      const fallback =
        translations.en ??
        FAQ_TRANSLATION_LANGUAGES.map(
          (language) => translations[language]
        ).find(Boolean)
      if (!fallback) {
        return { success: false, error: 'Invalid JSON data' }
      }
      entries.push({
        id: firstId + index,
        question: fallback.question,
        answer: fallback.answer,
        translations,
      })
      continue
    }

    const question =
      typeof candidate.question === 'string' ? candidate.question.trim() : ''
    const answer =
      typeof candidate.answer === 'string' ? candidate.answer.trim() : ''
    if (!question || !answer) {
      return {
        success: false,
        error: 'Entry {{index}} requires a question and answer',
        values: { index: index + 1 },
      }
    }
    if (question.length > 200) {
      return {
        success: false,
        error: 'Entry {{index}} question exceeds 200 characters',
        values: { index: index + 1 },
      }
    }
    if (answer.length > 1000) {
      return {
        success: false,
        error: 'Entry {{index}} answer exceeds 1000 characters',
        values: { index: index + 1 },
      }
    }
    entries.push({ id: firstId + index, question, answer })
  }

  return { success: true, entries }
}
