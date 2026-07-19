import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

// Rsbuild already maps `import './x.md'` to a raw string (rsbuild.config.ts).
// `import.meta.glob` lets us lazy-load only the (section, lang) the user opens.
const docModules = import.meta.glob<string>('@/i18n/docs/*/*.md', {
  query: '?raw',
  import: 'default',
  eager: false,
})

// Map i18next language code to docs folder name.
const LANG_FOLDER: Record<string, string> = {
  en: 'en',
  zhCN: 'zh',
  zhTW: 'zh-TW',
  fr: 'fr',
  ru: 'ru',
  ja: 'ja',
  vi: 'vi',
}

function pickFolder(i18nLang: string): string {
  return LANG_FOLDER[i18nLang] ?? 'en'
}

function moduleKey(sectionId: string, folder: string): string {
  return `/src/i18n/docs/${sectionId}/${folder}.md`
}

export interface DocContentResult {
  content: string
  loading: boolean
  notFound: boolean
}

interface LoaderEntry {
  key: string
  loader: () => Promise<string>
}

function resolveLoader(sectionId: string, folder: string): LoaderEntry | null {
  const key = moduleKey(sectionId, folder)
  const loader = docModules[key]
  if (loader) return { key, loader }
  // Fall back to English if the requested language is missing.
  if (folder !== 'en') {
    const enKey = moduleKey(sectionId, 'en')
    const enLoader = docModules[enKey]
    if (enLoader) return { key: enKey, loader: enLoader }
  }
  return null
}

export function useDocContent(sectionId: string): DocContentResult {
  const { i18n } = useTranslation()
  const folder = pickFolder(i18n.language)
  const entry = resolveLoader(sectionId, folder)

  const [state, setState] = useState<{
    content: string
    loading: boolean
    notFound: boolean
  }>(
    entry
      ? { content: '', loading: true, notFound: false }
      : { content: '', loading: false, notFound: true }
  )

  const cacheKey = entry?.key ?? ''
  useEffect(() => {
    if (!entry) {
      setState({ content: '', loading: false, notFound: true })
      return
    }
    let cancelled = false
    setState({ content: '', loading: true, notFound: false })
    entry
      .loader()
      .then((raw) => {
        if (!cancelled) {
          setState({ content: raw ?? '', loading: false, notFound: false })
        }
      })
      .catch(() => {
        if (!cancelled) {
          setState({ content: '', loading: false, notFound: true })
        }
      })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cacheKey])

  return state
}
