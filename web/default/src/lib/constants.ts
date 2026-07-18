/**
 * Application-wide constants
 */

// System Configuration Defaults (NovaPuraAI fork of New API)
export const DEFAULT_SYSTEM_NAME = 'NovaPuraAI'
/** Raster brand mark served from web/default/public (sourced from public/logo/). */
export const DEFAULT_LOGO = '/logo.png'
/** Multi-size ICO from public/logo/novapuraai_router_favicon/. */
export const DEFAULT_FAVICON = '/favicon.ico'

// LocalStorage Keys
export const STORAGE_KEYS = {
  SYSTEM_NAME: 'system_name',
  LOGO: 'logo',
  FOOTER_HTML: 'footer_html',
} as const
