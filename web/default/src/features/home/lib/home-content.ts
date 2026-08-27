import type { FAQItem } from '@/features/dashboard/types'

export { getRegisterPromo } from '@/lib/public-status'

export type HomeContentMode =
  | 'loading'
  | 'url'
  | 'html'
  | 'markdown'
  | 'built-in'

export function getHomeContentMode(input: {
  isLoaded: boolean
  content: string
  isUrl: boolean
}): HomeContentMode {
  if (!input.isLoaded) return 'loading'
  if (!input.content) return 'built-in'
  if (input.isUrl) return 'url'
  if (/^\s*</.test(input.content)) return 'html'
  return 'markdown'
}

export const HOME_FEATURED_MODELS = [
  'kimi-k3',
  'deepseek-v4-flash-0731',
  'deepseek-v4-pro-0831',
] as const

export function getHomeRouteModelNames(
  models: ReadonlyArray<{ model_name?: unknown }>
): string[] {
  const catalogByKey = new Map<string, string>()

  for (const model of models) {
    if (typeof model.model_name !== 'string') continue

    const name = model.model_name.trim()
    if (!name) continue

    const slash = name.lastIndexOf('/')
    const key = (slash >= 0 ? name.slice(slash + 1) : name).toLowerCase()
    if (!catalogByKey.has(key)) catalogByKey.set(key, name)
  }

  return HOME_FEATURED_MODELS.map(
    (featured) => catalogByKey.get(featured.toLowerCase()) ?? featured
  )
}

export function getPublicModelNames(
  models: ReadonlyArray<{ model_name?: unknown }>,
  limit = 4
): string[] {
  if (limit <= 0) return []

  const names: string[] = []
  const seen = new Set<string>()

  for (const model of models) {
    if (names.length >= limit) break
    if (typeof model.model_name !== 'string') continue

    const name = model.model_name.trim()
    const normalizedName = name.toLowerCase()
    if (!name || seen.has(normalizedName)) continue

    seen.add(normalizedName)
    names.push(name)
  }

  return names
}

function normalizeFingerprintPart(value: string): string {
  return value.trim().replaceAll(/\s+/g, ' ').toLocaleLowerCase()
}

export function dedupeFAQItems(items: FAQItem[]): FAQItem[] {
  const seen = new Set<string>()
  return items.filter((item) => {
    const fingerprint = [
      normalizeFingerprintPart(item.question),
      normalizeFingerprintPart(item.answer),
    ].join('\u0000')
    if (seen.has(fingerprint)) return false
    seen.add(fingerprint)
    return true
  })
}
