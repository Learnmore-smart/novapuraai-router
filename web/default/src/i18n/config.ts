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
import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import { convertDetectedLanguage } from './languages'
import en from './locales/en.json'
import fr from './locales/fr.json'
import ja from './locales/ja.json'
import ru from './locales/ru.json'
import vi from './locales/vi.json'
import zhCN from './locales/zh.json'
import zhTW from './locales/zh-TW.json'

export const resources = {
  en,
  zhCN,
  fr,
  ru,
  ja,
  vi,
  zhTW,
} as const

type LocaleBundle = { translation?: Record<string, string> }

/** Push current locale modules into i18next (initial load + HMR). */
function applyLocaleBundles() {
  const bundles: Record<string, LocaleBundle> = {
    en,
    zhCN,
    fr,
    ru,
    ja,
    vi,
    zhTW,
  }
  for (const [lng, bundle] of Object.entries(bundles)) {
    const dictionary = bundle.translation ?? (bundle as Record<string, string>)
    i18n.addResourceBundle(lng, 'translation', dictionary, true, true)
  }
}

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    // NovaPura product default: Chinese first; English remains fully supported.
    fallbackLng: 'zhCN',
    supportedLngs: ['en', 'zhCN', 'fr', 'ru', 'ja', 'vi', 'zhTW'],
    load: 'currentOnly',
    nsSeparator: false, // Allow literal colons in keys (e.g., URLs, labels)
    debug: import.meta.env.DEV,
    interpolation: {
      escapeValue: false, // not needed for react as it escapes by default
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
      // Browsers report `zh-CN`/`zh-TW`/`zh`; map them onto our `zhCN`/`zhTW`
      // codes (non-Chinese codes pass through for normal supportedLngs matching).
      convertDetectedLanguage,
    },
  })

// Dev-only: locale JSON is huge and was loaded once into i18next, so plain HMR
// often rebuilt the module without refreshing UI copy. Accept those modules and
// force a languageChanged so useTranslation() re-renders without a server restart.
if (import.meta.env.DEV && import.meta.webpackHot) {
  import.meta.webpackHot.accept(
    [
      './locales/en.json',
      './locales/zh.json',
      './locales/zh-TW.json',
      './locales/fr.json',
      './locales/ja.json',
      './locales/ru.json',
      './locales/vi.json',
    ],
    () => {
      applyLocaleBundles()
      void i18n.changeLanguage(i18n.language)
    }
  )
}

export default i18n
