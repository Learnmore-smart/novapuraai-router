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
import { Link } from '@tanstack/react-router'
import { Fragment, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'
import { cn } from '@/lib/utils'

import { BrandMark } from './brand-mark'

interface FooterLink {
  text: string
  href: string
}

interface FooterColumnProps {
  title: string
  links: FooterLink[]
}

interface FooterProps {
  logo?: string
  name?: string
  columns?: FooterColumnProps[]
  copyright?: string
  className?: string
}

const NEW_API_FOOTER_ATTRIBUTION_KEY = [
  'footer',
  'new' + 'api',
  'projectAttributionSuffix',
].join('.')

function FooterLinkItem(props: { link: FooterLink }) {
  const { t } = useTranslation()
  const isExternal = props.link.href.startsWith('http')
  const label = t(props.link.text)

  if (isExternal) {
    return (
      <a
        href={props.link.href}
        target='_blank'
        rel='noopener noreferrer'
        className='text-muted-foreground hover:text-foreground text-sm transition-colors'
      >
        {label}
      </a>
    )
  }

  return (
    <Link
      to={props.link.href}
      className='text-muted-foreground hover:text-foreground text-sm transition-colors'
    >
      {label}
    </Link>
  )
}

function LegalLinks(props: { leadingSeparator?: boolean }) {
  const { t } = useTranslation()
  // Legal documents ship as frontend i18n content (always available).
  // Status flags remain a soft gate for older backends that omit them.
  const { status } = useStatus()
  const termsEnabled = status?.user_agreement_enabled !== false
  const privacyEnabled = status?.privacy_policy_enabled !== false
  const items: { key: string; label: string; href: string }[] = []
  if (termsEnabled) {
    items.push({
      key: 'user-agreement',
      label: t('Terms and Conditions'),
      href: '/user-agreement',
    })
  }
  if (privacyEnabled) {
    items.push({
      key: 'privacy-policy',
      label: t('Privacy Policy'),
      href: '/privacy-policy',
    })
  }
  if (items.length === 0) {
    return null
  }
  return (
    <>
      {items.map((item, index) => (
        <Fragment key={item.key}>
          {(props.leadingSeparator || index > 0) && (
            <span aria-hidden='true' className='text-muted-foreground/30'>
              ·
            </span>
          )}
          <Link
            to={item.href}
            className='hover:text-foreground transition-colors'
          >
            {item.label}
          </Link>
        </Fragment>
      ))}
    </>
  )
}

function ProjectAttribution(props: { currentYear: number; inline?: boolean }) {
  const { t } = useTranslation()
  const content = (
    <span className='text-muted-foreground/50'>
      &copy; {props.currentYear}{' '}
      <a
        href='https://github.com/QuantumNous/new-api'
        target='_blank'
        rel='noopener noreferrer'
        className='text-foreground/70 hover:text-foreground font-medium transition-colors'
      >
        {t('New API')}
      </a>
      . {t(NEW_API_FOOTER_ATTRIBUTION_KEY)}
    </span>
  )
  if (props.inline) {
    return content
  }
  return (
    <div className='text-muted-foreground/50 text-center text-xs sm:text-right'>
      {content}
    </div>
  )
}

export function Footer(props: FooterProps) {
  const { t } = useTranslation()
  const {
    systemName,
    logo: systemLogo,
    footerHtml,
    demoSiteEnabled,
  } = useSystemConfig()

  const displayLogo = systemLogo || props.logo || '/logo.png'
  const displayName = systemName || props.name || 'NovaPuraAI'
  const isDemoSiteMode = Boolean(demoSiteEnabled)
  const currentYear = new Date().getFullYear()

  const { status } = useStatus()
  const docsLink =
    typeof status?.docs_link === 'string' ? status.docs_link.trim() : ''

  const productColumns = useMemo<FooterColumnProps[]>(() => {
    const columns: FooterColumnProps[] = [
      {
        title: t('Product'),
        links: [
          { text: t('Model Square'), href: '/pricing' },
          { text: t('Rankings'), href: '/rankings' },
          { text: t('Login'), href: '/sign-in' },
        ],
      },
      {
        title: t('Account'),
        links: [
          { text: t('Sign up'), href: '/sign-up' },
          { text: t('Dashboard'), href: '/dashboard' },
          { text: t('Forgot password'), href: '/forgot-password' },
        ],
      },
    ]

    const docsLinks: FooterLink[] = [
      { text: 'Docs', href: '/docs' },
      { text: 'Terms and Conditions', href: '/user-agreement' },
      { text: 'Privacy Policy', href: '/privacy-policy' },
      { text: 'Refund Policy', href: '/refund-policy' },
    ]
    if (docsLink) {
      docsLinks.splice(1, 0, {
        text: 'footer.columns.docs.links.apiDocs',
        href: docsLink,
      })
    }
    columns.push({
      title: 'footer.columns.docs.title',
      links: docsLinks,
    })

    return columns
  }, [docsLink, t])

  const fallbackColumns = useMemo<FooterColumnProps[]>(
    () => [
      {
        title: t('footer.columns.related.title'),
        links: [
          {
            text: t('footer.columns.related.links.oneApi'),
            href: 'https://github.com/songquanpeng/one-api',
          },
          {
            text: t('footer.columns.related.links.midjourney'),
            href: 'https://github.com/novicezk/midjourney-proxy',
          },
          {
            text: t('footer.columns.related.links.newApiKeyTool'),
            href: 'https://github.com/Calcium-Ion/new-api-key-tool',
          },
        ],
      },
    ],
    [t]
  )

  const displayColumns =
    props.columns ?? (isDemoSiteMode ? fallbackColumns : productColumns)

  if (footerHtml) {
    return (
      <footer
        className={cn(
          'relative z-10 border-t border-border bg-muted/30',
          props.className
        )}
      >
        <div className='mx-auto w-full max-w-7xl px-6 py-6'>
          <div className='border-border bg-card flex flex-col items-center justify-between gap-4 border px-4 py-4 sm:flex-row sm:px-5'>
            <div
              className='custom-footer text-muted-foreground min-w-0 text-center text-sm sm:text-left'
              dangerouslySetInnerHTML={{ __html: footerHtml }}
            />
            <div className='border-border text-muted-foreground/50 flex w-full flex-wrap items-center justify-center gap-x-3 gap-y-1 border-t pt-4 text-xs sm:w-auto sm:justify-end sm:border-t-0 sm:border-l sm:pt-0 sm:pl-5'>
              <LegalLinks />
              <ProjectAttribution currentYear={currentYear} inline />
            </div>
          </div>
        </div>
      </footer>
    )
  }

  return (
    <footer
      className={cn(
        'relative z-10 border-t border-border bg-muted/20',
        props.className
      )}
    >
      <div className='mx-auto max-w-7xl px-6 py-14 md:py-16'>
        <div className='flex flex-col justify-between gap-12 md:flex-row md:gap-16'>
          <div className='max-w-sm shrink-0'>
            <Link to='/' className='flex items-center gap-2.5'>
              <BrandMark
                src={displayLogo}
                alt={displayName}
                className='size-8'
              />
              <span className='text-[0.9375rem] font-semibold tracking-tight'>
                {displayName}
              </span>
            </Link>
            <p className='text-muted-foreground mt-4 text-sm leading-relaxed'>
              {t(
                'One OpenAI-compatible endpoint. Prepaid balances. Clear usage per request.'
              )}
            </p>
          </div>

          <div className='grid grid-cols-2 gap-10 sm:grid-cols-3 md:gap-14'>
            {displayColumns.map((column) => (
              <div key={column.title}>
                <p className='np-kicker mb-4'>{t(column.title)}</p>
                <ul className='space-y-3'>
                  {column.links.map((link) => (
                    <li key={`${link.text}:${link.href}`}>
                      <FooterLinkItem link={link} />
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>

        <div className='border-border mt-14 flex flex-col items-center justify-between gap-x-3 gap-y-2 border-t pt-6 sm:flex-row'>
          <div className='text-muted-foreground/50 flex flex-wrap items-center justify-center gap-x-2 gap-y-1 text-xs sm:justify-start'>
            <span>
              &copy; {currentYear} {displayName}.{' '}
              {props.copyright ?? t('footer.defaultCopyright')}
            </span>
            <LegalLinks leadingSeparator />
          </div>
          <ProjectAttribution currentYear={currentYear} />
        </div>
      </div>
    </footer>
  )
}
