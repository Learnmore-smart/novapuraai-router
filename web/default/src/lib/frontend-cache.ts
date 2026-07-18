const FRONTEND_CACHE_VERSION = 'default-v1'
const FRONTEND_CACHE_VERSION_KEY = 'newapi:default:cache-version'
const PRESERVED_LOCAL_STORAGE_KEYS = new Set([
  FRONTEND_CACHE_VERSION_KEY,
  'user',
  'uid',
  'aff',
  'oauth:binding:result',
])

export function initializeFrontendCache(): void {
  if (typeof window === 'undefined') return

  try {
    const currentVersion = window.localStorage.getItem(
      FRONTEND_CACHE_VERSION_KEY
    )
    if (currentVersion === FRONTEND_CACHE_VERSION) return

    clearLocalUiCache()
    window.localStorage.setItem(
      FRONTEND_CACHE_VERSION_KEY,
      FRONTEND_CACHE_VERSION
    )
  } catch {
    // Storage can be unavailable in private mode; the app should still boot.
  }
}

function clearLocalUiCache(): void {
  const keysToRemove: string[] = []
  for (let index = 0; index < window.localStorage.length; index += 1) {
    const key = window.localStorage.key(index)
    if (key && !PRESERVED_LOCAL_STORAGE_KEYS.has(key)) {
      keysToRemove.push(key)
    }
  }

  keysToRemove.forEach((key) => window.localStorage.removeItem(key))
}
