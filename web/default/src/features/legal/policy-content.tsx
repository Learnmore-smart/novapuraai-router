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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { cn } from '@/lib/utils'

import privacyEn from '@/i18n/policies/privacy/en.json'
import privacyZh from '@/i18n/policies/privacy/zh.json'
import termsEn from '@/i18n/policies/terms/en.json'
import termsZh from '@/i18n/policies/terms/zh.json'

type PolicyKind = 'privacy' | 'terms'

type PolicyBundle = {
  title?: string
  lastUpdated?: string
  inShortLabel?: string
  intro?: Record<string, string>
  summary?: Record<string, string>
  toc?: Record<string, string>
  sections?: Record<string, Record<string, string>>
}

const PRIVACY_BY_LANG: Record<string, { privacy: PolicyBundle }> = {
  en: privacyEn as { privacy: PolicyBundle },
  zh: privacyZh as { privacy: PolicyBundle },
  'zh-CN': privacyZh as { privacy: PolicyBundle },
  'zh-TW': privacyZh as { privacy: PolicyBundle },
}

const TERMS_BY_LANG: Record<string, { terms: PolicyBundle }> = {
  en: termsEn as { terms: PolicyBundle },
  zh: termsZh as { terms: PolicyBundle },
  'zh-CN': termsZh as { terms: PolicyBundle },
  'zh-TW': termsZh as { terms: PolicyBundle },
}

function resolveLang(language: string | undefined): string {
  if (!language) return 'en'
  if (language.startsWith('zh')) return 'zh'
  return language.split('-')[0] || 'en'
}

function getPolicy(kind: PolicyKind, language: string | undefined): PolicyBundle {
  const lang = resolveLang(language)
  if (kind === 'privacy') {
    return (PRIVACY_BY_LANG[lang] ?? PRIVACY_BY_LANG.en).privacy
  }
  return (TERMS_BY_LANG[lang] ?? TERMS_BY_LANG.en).terms
}

function websiteUrl(): string {
  if (
    typeof window !== 'undefined' &&
    window.location?.origin &&
    (window.location.hostname === 'localhost' ||
      window.location.hostname === '127.0.0.1')
  ) {
    return window.location.origin
  }
  return 'https://www.novapuraai.com'
}

function interpolate(text: string, vars: Record<string, string>): string {
  return text
    .replaceAll(/\{\s*websiteUrl\s*\}/g, vars.websiteUrl)
    .replaceAll(/\{\s*getPrivacyPolicyUrl\s*\}/g, vars.privacyUrl)
    .replaceAll(/\{\s*url\}\/contact-us/g, `${vars.websiteUrl}/contact-us`)
    .replaceAll(/\{\s*url\s*\}/g, vars.websiteUrl)
    .replaceAll('&apos;', "'")
}

function sortPolicyKeys(keys: string[]): string[] {
  return [...keys].sort((a, b) => {
    const rank = (key: string) => {
      if (key === 'title') return 0
      if (key.startsWith('subtitle')) return 1
      if (key.startsWith('inShort')) return 2
      if (key.startsWith('q')) return 3
      if (key.startsWith('a')) return 4
      if (key.startsWith('p')) return 5
      if (key.startsWith('li')) return 6
      if (key === 'footer') return 99
      return 50
    }
    const ra = rank(a)
    const rb = rank(b)
    if (ra !== rb) return ra - rb
    return a.localeCompare(b, undefined, { numeric: true })
  })
}

function PolicyBlock(props: {
  text: string
  keyName: string
  inShortLabel?: string
  className?: string
}) {
  const { text, keyName, inShortLabel, className } = props
  if (keyName === 'title') {
    return (
      <h2 className={cn('mt-8 text-xl font-semibold tracking-tight', className)}>
        {text}
      </h2>
    )
  }
  if (keyName.startsWith('subtitle')) {
    return (
      <h3 className={cn('mt-5 text-lg font-medium tracking-tight', className)}>
        {text}
      </h3>
    )
  }
  if (keyName.startsWith('inShort')) {
    return (
      <p className={cn('text-muted-foreground text-sm italic', className)}>
        {inShortLabel ? (
          <>
            <span className='font-semibold not-italic'>{inShortLabel}</span>{' '}
          </>
        ) : null}
        {text}
      </p>
    )
  }
  if (keyName.startsWith('q')) {
    return (
      <p className={cn('font-semibold', className)}>
        {text}
      </p>
    )
  }
  if (keyName.startsWith('li')) {
    return <li className={cn('ml-5 list-disc', className)}>{text}</li>
  }
  return <p className={cn('leading-relaxed', className)}>{text}</p>
}

type PolicyContentProps = {
  kind: PolicyKind
}

export function PolicyContent({ kind }: PolicyContentProps) {
  const { i18n, t } = useTranslation()
  const site = websiteUrl()
  const vars = useMemo(
    () => ({
      websiteUrl: site,
      privacyUrl: `${site}/privacy-policy`,
    }),
    [site]
  )

  const policy = useMemo(
    () => getPolicy(kind, i18n.language),
    [kind, i18n.language]
  )

  const pageTitle =
    policy.title ||
    (kind === 'privacy' ? t('Privacy Policy') : t('Terms and Conditions'))

  return (
    <PublicLayout>
      <article className='mx-auto max-w-4xl space-y-6 py-12'>
        <header className='space-y-2 border-b pb-6'>
          <h1 className='text-3xl font-semibold tracking-tight'>{pageTitle}</h1>
          {policy.lastUpdated ? (
            <p className='text-muted-foreground text-sm'>{policy.lastUpdated}</p>
          ) : null}
        </header>

        {policy.intro ? (
          <section className='space-y-3'>
            {sortPolicyKeys(Object.keys(policy.intro)).map((key) => {
              const intro = policy.intro ?? {}
              const value = intro[key]
              if (!value) return null
              return (
                <PolicyBlock
                  key={`intro-${key}`}
                  keyName={key}
                  text={interpolate(value, vars)}
                  inShortLabel={policy.inShortLabel}
                />
              )
            })}
          </section>
        ) : null}

        {policy.summary ? (
          <section className='space-y-3'>
            {sortPolicyKeys(Object.keys(policy.summary)).map((key) => {
              const summary = policy.summary ?? {}
              const value = summary[key]
              if (!value) return null
              return (
                <PolicyBlock
                  key={`summary-${key}`}
                  keyName={key === 'title' ? 'title' : key}
                  text={interpolate(value, vars)}
                  inShortLabel={policy.inShortLabel}
                />
              )
            })}
          </section>
        ) : null}

        {policy.toc ? (
          <section className='space-y-3'>
            {policy.toc.title ? (
              <h2 className='mt-8 text-xl font-semibold tracking-tight'>
                {policy.toc.title}
              </h2>
            ) : null}
            <ul className='space-y-1'>
              {sortPolicyKeys(Object.keys(policy.toc))
                .filter((key) => key !== 'title')
                .map((key) => {
                  const toc = policy.toc ?? {}
                  const value = toc[key]
                  if (!value) return null
                  return (
                    <li key={`toc-${key}`} className='ml-5 list-disc'>
                      {value}
                    </li>
                  )
                })}
            </ul>
          </section>
        ) : null}

        {policy.sections
          ? sortPolicyKeys(Object.keys(policy.sections)).map((sectionKey) => {
              const sections = policy.sections ?? {}
              const section = sections[sectionKey]
              if (!section) return null
              return (
                <section key={sectionKey} className='space-y-3'>
                  {sortPolicyKeys(Object.keys(section)).map((key) => {
                    const value = section[key]
                    if (!value) return null
                    return (
                      <PolicyBlock
                        key={`${sectionKey}-${key}`}
                        keyName={key}
                        text={interpolate(value, vars)}
                        inShortLabel={policy.inShortLabel}
                      />
                    )
                  })}
                </section>
              )
            })
          : null}
      </article>
    </PublicLayout>
  )
}
