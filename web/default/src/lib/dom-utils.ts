/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { DEFAULT_FAVICON, DEFAULT_LOGO } from './constants.ts'

const STOCK_BRAND_LOGO_PATHS = new Set([
  DEFAULT_LOGO,
  '/logo-256.png',
  '/logo.webp',
  '/logo-256.webp',
])

export function isDefaultBrandLogo(url?: string | null): boolean {
  if (!url) return true

  const value = url.trim()
  if (/^(?:[a-z][a-z\d+.-]*:)?\/\//i.test(value)) return false

  try {
    const pathname = new URL(value, 'https://novapura.local').pathname
    return STOCK_BRAND_LOGO_PATHS.has(pathname)
  } catch {
    return false
  }
}

export function resolveFaviconUrl(logoUrl?: string | null): string {
  return isDefaultBrandLogo(logoUrl)
    ? DEFAULT_FAVICON
    : logoUrl || DEFAULT_FAVICON
}

export function applyFaviconToDom(logoUrl?: string | null) {
  if (typeof document === 'undefined') return

  const runtimeFavicon = document.querySelector<HTMLLinkElement>(
    'link[data-runtime-favicon="true"]'
  )

  if (isDefaultBrandLogo(logoUrl)) {
    runtimeFavicon?.remove()
    return
  }

  try {
    const url = resolveFaviconUrl(logoUrl)
    const next = new URL(url, window.location.href).href
    if (runtimeFavicon?.href === next) return

    const link = runtimeFavicon ?? document.createElement('link')
    link.rel = 'icon'
    link.dataset.runtimeFavicon = 'true'
    link.href = url
    if (!runtimeFavicon) document.head.appendChild(link)
  } catch {
    // Ignore malformed URLs
  }
}
