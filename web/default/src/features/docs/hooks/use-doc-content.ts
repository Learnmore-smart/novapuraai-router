import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  DEFAULT_DOC_SECTION,
  type DocSectionId,
  isDocSectionId,
} from '../config/nav-tree'

export type DocLangCode = 'en' | 'zh' | 'zh-TW' | 'fr' | 'ru' | 'ja' | 'vi'

const DOC_LANGS: DocLangCode[] = [
  'en',
  'zh',
  'zh-TW',
  'fr',
  'ru',
  'ja',
  'vi',
]

/**
 * Map i18next language codes onto docs markdown file language codes.
 */
export function resolveDocLang(language: string | undefined): DocLangCode {
  if (!language) return 'en'
  const normalized = language.trim().replaceAll('_', '-')
  const lower = normalized.toLowerCase()

  if (
    lower === 'zhtw' ||
    lower === 'zh-tw' ||
    lower === 'zh-hk' ||
    lower === 'zh-mo' ||
    lower.startsWith('zh-hant')
  ) {
    return 'zh-TW'
  }
  if (lower === 'zhcn' || lower === 'zh-cn' || lower.startsWith('zh')) {
    return 'zh'
  }
  if (lower.startsWith('fr')) return 'fr'
  if (lower.startsWith('ru')) return 'ru'
  if (lower.startsWith('ja')) return 'ja'
  if (lower.startsWith('vi')) return 'vi'
  if (lower.startsWith('en')) return 'en'
  return 'en'
}

// Markdown is loaded as raw strings via rspack `asset/source` (see rsbuild.config.ts).
const docModules = import.meta.glob('../../../i18n/docs/*/*.md') as Record<
  string,
  () => Promise<string | { default: string }>
>

function buildModuleKey(section: DocSectionId, lang: DocLangCode): string {
  return `../../../i18n/docs/${section}/${lang}.md`
}

async function resolveModuleText(
  loader: () => Promise<string | { default: string }>
): Promise<string> {
  const mod = await loader()
  if (typeof mod === 'string') return mod
  if (mod && typeof mod === 'object' && typeof mod.default === 'string') {
    return mod.default
  }
  return ''
}

async function loadDocMarkdown(
  section: DocSectionId,
  lang: DocLangCode
): Promise<{ markdown: string; resolvedLang: DocLangCode }> {
  const preferredKey = buildModuleKey(section, lang)
  const preferredLoader = docModules[preferredKey]
  if (preferredLoader) {
    const markdown = await resolveModuleText(preferredLoader)
    if (markdown.trim()) {
      return { markdown, resolvedLang: lang }
    }
  }

  // Fallback chain: en → zh → first available for section
  for (const fallback of DOC_LANGS) {
    if (fallback === lang) continue
    const key = buildModuleKey(section, fallback)
    const loader = docModules[key]
    if (!loader) continue
    const markdown = await resolveModuleText(loader)
    if (markdown.trim()) {
      return { markdown, resolvedLang: fallback }
    }
  }

  return { markdown: '', resolvedLang: lang }
}

export function useDocContent(sectionParam: string | undefined) {
  const { i18n } = useTranslation()
  const section: DocSectionId = isDocSectionId(sectionParam ?? '')
    ? (sectionParam as DocSectionId)
    : DEFAULT_DOC_SECTION
  const requestedLang = resolveDocLang(i18n.language)

  const [markdown, setMarkdown] = useState('')
  const [resolvedLang, setResolvedLang] = useState<DocLangCode>(requestedLang)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)

    void loadDocMarkdown(section, requestedLang)
      .then((result) => {
        if (cancelled) return
        setMarkdown(result.markdown)
        setResolvedLang(result.resolvedLang)
        if (!result.markdown.trim()) {
          setError('missing')
        }
      })
      .catch(() => {
        if (cancelled) return
        setMarkdown('')
        setError('load_failed')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [section, requestedLang])

  return {
    section,
    markdown,
    loading,
    error,
    requestedLang,
    resolvedLang,
    usedFallback: resolvedLang !== requestedLang,
  }
}
