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
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { Menu, X } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { LanguageSwitcher } from '@/components/language-switcher'
import { NotificationPopover } from '@/components/notification-popover'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useNotifications } from '@/hooks/use-notifications'
import { useSystemConfig } from '@/hooks/use-system-config'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { isDefaultBrandLogo } from '@/lib/dom-utils'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { defaultTopNavLinks } from '../config/top-nav.config'
import type { TopNavLink } from '../types'
import { HeaderLogo } from './header-logo'

const AUTH_PROMPT_SECONDS = 5

type AuthPromptTarget = {
  title: string
  href: string
}

export interface PublicHeaderProps {
  navLinks?: TopNavLink[]
  mobileLinks?: TopNavLink[]
  navContent?: React.ReactNode
  showThemeSwitch?: boolean
  showLanguageSwitcher?: boolean
  logo?: React.ReactNode
  siteName?: string
  homeUrl?: string
  leftContent?: React.ReactNode
  rightContent?: React.ReactNode
  showNavigation?: boolean
  showAuthButtons?: boolean
  showNotifications?: boolean
  className?: string
}

export function PublicHeader(props: PublicHeaderProps) {
  const {
    navLinks = defaultTopNavLinks,
    showThemeSwitch = true,
    showLanguageSwitcher = true,
    logo: customLogo,
    siteName: customSiteName,
    homeUrl = '/',
    showAuthButtons = true,
    showNotifications = true,
    className,
  } = props

  const { t } = useTranslation()
  const navigate = useNavigate()
  const [scrolled, setScrolled] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [authPromptTarget, setAuthPromptTarget] =
    useState<AuthPromptTarget | null>(null)
  const [authPromptSecondsLeft, setAuthPromptSecondsLeft] =
    useState(AUTH_PROMPT_SECONDS)
  const mobileTriggerRef = useRef<HTMLButtonElement>(null)
  const mobileOverlayRef = useRef<HTMLDivElement>(null)
  const { auth } = useAuthStore()
  const {
    systemName,
    logo: systemLogo,
    loading,
    logoLoaded,
  } = useSystemConfig()
  const dynamicLinks = useTopNavLinks()
  const notifications = useNotifications()
  const routerState = useRouterState()
  const pathname = routerState.location.pathname

  const user = auth.user
  const isAuthenticated = !!user
  const displaySiteName = customSiteName || systemName
  const links = dynamicLinks.length > 0 ? dynamicLinks : navLinks

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8)
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => {
    if (!mobileOpen) return

    const previouslyFocused =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null
    const frameId = window.requestAnimationFrame(() => {
      mobileOverlayRef.current
        ?.querySelector<HTMLElement>('a[href], button:not([disabled])')
        ?.focus()
    })
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        setMobileOpen(false)
        return
      }

      if (event.key !== 'Tab') return
      const overlayFocusable = [
        ...(mobileOverlayRef.current?.querySelectorAll<HTMLElement>(
          'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"])'
        ) ?? []),
      ]
      const focusable = [mobileTriggerRef.current, ...overlayFocusable].filter(
        (element): element is HTMLElement => element !== null
      )
      if (focusable.length === 0) return

      const first = focusable[0]
      const last = focusable.at(-1)
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last?.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }

    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', handleKeyDown)

    return () => {
      window.cancelAnimationFrame(frameId)
      document.body.style.overflow = ''
      document.removeEventListener('keydown', handleKeyDown)
      previouslyFocused?.focus()
    }
  }, [mobileOpen])

  useEffect(() => {
    if (!authPromptTarget) return

    const intervalId = window.setInterval(() => {
      setAuthPromptSecondsLeft((seconds) => Math.max(seconds - 1, 0))
    }, 1000)

    const timeoutId = window.setTimeout(() => {
      const redirect = authPromptTarget.href
      setAuthPromptTarget(null)
      navigate({ to: '/sign-in', search: { redirect } })
    }, AUTH_PROMPT_SECONDS * 1000)

    return () => {
      window.clearInterval(intervalId)
      window.clearTimeout(timeoutId)
    }
  }, [authPromptTarget, navigate])

  const closeAuthPrompt = useCallback(() => {
    setAuthPromptTarget(null)
    setAuthPromptSecondsLeft(AUTH_PROMPT_SECONDS)
  }, [])

  const navigateToSignIn = useCallback(() => {
    const redirect = authPromptTarget?.href || '/'
    setAuthPromptTarget(null)
    navigate({ to: '/sign-in', search: { redirect } })
  }, [authPromptTarget?.href, navigate])

  const handleNavLinkClick = useCallback(
    (
      event: React.MouseEvent<HTMLAnchorElement>,
      link: TopNavLink,
      closeMobile = false
    ) => {
      if (link.disabled) {
        event.preventDefault()
        return
      }

      if (link.requiresAuth) {
        event.preventDefault()
        if (closeMobile) {
          setMobileOpen(false)
        }
        setAuthPromptSecondsLeft(AUTH_PROMPT_SECONDS)
        setAuthPromptTarget({
          title: t(link.title),
          href: link.href,
        })
        return
      }

      if (closeMobile) {
        setMobileOpen(false)
      }
    },
    [t]
  )

  const stockLogo = isDefaultBrandLogo(systemLogo)

  let logoContent: React.ReactNode = (
    <HeaderLogo
      src={systemLogo}
      loading={loading}
      logoLoaded={logoLoaded}
      className='size-full'
    />
  )
  if (loading) {
    logoContent = <Skeleton className='size-full rounded-md' />
  } else if (customLogo) {
    logoContent = customLogo
  }

  // Auth CTA: Login (text) + Sign up (solid primary) for conversion; Dashboard when signed in.
  let desktopAuthControl: React.ReactNode = null
  if (loading) {
    desktopAuthControl = <Skeleton className='h-9 w-24 rounded-md' />
  } else if (isAuthenticated) {
    desktopAuthControl = (
      <div className='flex items-center gap-2'>
        <Button
          size='sm'
          variant='outline'
          className='h-9 rounded-md px-3.5 text-xs font-medium'
          render={<Link to='/dashboard' />}
        >
          {t('Dashboard')}
        </Button>
        <ProfileDropdown />
      </div>
    )
  } else {
    desktopAuthControl = (
      <div className='flex items-center gap-1.5'>
        <Button
          size='sm'
          variant='ghost'
          className='h-9 rounded-md px-3 text-xs font-medium'
          render={<Link to='/sign-in' />}
        >
          {t('Login')}
        </Button>
        <Button
          size='sm'
          className='h-9 rounded-md px-3.5 text-xs font-semibold shadow-none'
          render={<Link to='/sign-up' />}
        >
          {t('Sign up')}
        </Button>
      </div>
    )
  }

  return (
    <>
      <header
        className={cn(
          'fixed inset-x-0 top-0 z-50 border-b transition-[background-color,border-color,box-shadow] duration-200',
          scrolled
            ? 'border-border bg-background/95 shadow-[var(--elevation-1)] backdrop-blur-sm'
            : 'border-transparent bg-background/80 backdrop-blur-sm',
          className
        )}
      >
        <nav className='mx-auto flex h-[var(--app-header-height)] max-w-7xl items-center justify-between gap-4 px-4 sm:px-6'>
          <Link
            to={homeUrl}
            className='group flex min-w-0 shrink-0 items-center gap-2.5'
          >
            {/* Stock monogram already has radius — do not double-round. Custom logos may use contain box. */}
            <div
              className={cn(
                'flex size-8 shrink-0 items-center justify-center',
                !stockLogo && 'overflow-hidden rounded-md'
              )}
            >
              {logoContent}
            </div>
            <span className='truncate text-[0.9375rem] font-semibold tracking-tight'>
              {loading ? <Skeleton className='h-4 w-20' /> : displaySiteName}
            </span>
          </Link>

          <div className='hidden items-center gap-0.5 md:flex'>
            {links.map((link) => {
              const isActive =
                pathname === link.href ||
                (link.href !== '/' && pathname.startsWith(link.href))
              if (link.external) {
                return (
                  <a
                    key={`${link.title}:${link.href}`}
                    href={link.href}
                    target='_blank'
                    rel='noopener noreferrer'
                    aria-disabled={link.disabled}
                    tabIndex={link.disabled ? -1 : undefined}
                    onClick={(event) => handleNavLinkClick(event, link)}
                    className={cn(
                      'text-muted-foreground hover:text-foreground rounded-md px-3 py-2 text-[13px] font-medium transition-colors',
                      link.disabled && 'pointer-events-none opacity-50'
                    )}
                  >
                    {t(link.title)}
                  </a>
                )
              }
              return (
                <Link
                  key={`${link.title}:${link.href}`}
                  to={link.href}
                  disabled={link.disabled}
                  onClick={(event) => handleNavLinkClick(event, link)}
                  className={cn(
                    'rounded-md px-3 py-2 text-[13px] font-medium transition-colors',
                    isActive
                      ? 'bg-muted text-foreground'
                      : 'text-muted-foreground hover:text-foreground hover:bg-muted/60',
                    link.disabled && 'pointer-events-none opacity-50'
                  )}
                >
                  {t(link.title)}
                </Link>
              )
            })}
          </div>

          <div className='hidden items-center gap-1.5 md:flex'>
            {showLanguageSwitcher && <LanguageSwitcher />}
            {showThemeSwitch && <ThemeSwitch />}
            {showNotifications && isAuthenticated && (
              <NotificationPopover
                open={notifications.popoverOpen}
                onOpenChange={notifications.setPopoverOpen}
                unreadCount={notifications.unreadCount}
                activeTab={notifications.activeTab}
                onTabChange={notifications.setActiveTab}
                notice={notifications.notice}
                announcements={notifications.announcements}
                loading={notifications.loading}
              />
            )}
            {showAuthButtons && desktopAuthControl}
          </div>

          <div className='flex items-center gap-1.5 md:hidden'>
            {showThemeSwitch && <ThemeSwitch />}
            {showAuthButtons && !loading && isAuthenticated && (
              <ProfileDropdown />
            )}
            <Button
              ref={mobileTriggerRef}
              type='button'
              variant='ghost'
              size='icon'
              className='size-9 rounded-md'
              onClick={() => setMobileOpen((v) => !v)}
              aria-label={t('Toggle navigation menu')}
              aria-expanded={mobileOpen}
              aria-controls='mobile-navigation-dialog'
            >
              {mobileOpen ? (
                <X className='size-[1.125rem]' aria-hidden='true' />
              ) : (
                <Menu className='size-[1.125rem]' aria-hidden='true' />
              )}
            </Button>
          </div>
        </nav>
      </header>

      {mobileOpen && (
        <div
          id='mobile-navigation-dialog'
          ref={mobileOverlayRef}
          role='dialog'
          aria-modal='true'
          aria-label={t('Toggle navigation menu')}
          className='bg-background fixed inset-0 z-40 md:hidden'
        >
          <div className='flex h-full flex-col justify-between px-6 pt-[calc(var(--app-header-height)+1.5rem)] pb-10'>
            <nav className='border-border flex flex-col gap-1 border-t pt-4'>
              {links.map((link) => {
                const isActive =
                  pathname === link.href ||
                  (link.href !== '/' && pathname.startsWith(link.href))
                const linkClassName = cn(
                  'flex items-center py-3 text-base font-medium tracking-tight transition-colors',
                  isActive ? 'text-foreground' : 'text-muted-foreground',
                  isActive && 'bg-muted -mx-3 rounded-md px-3',
                  link.disabled && 'pointer-events-none opacity-50'
                )
                if (link.external) {
                  return (
                    <a
                      key={`${link.title}:${link.href}`}
                      href={link.href}
                      target='_blank'
                      rel='noopener noreferrer'
                      aria-disabled={link.disabled}
                      tabIndex={link.disabled ? -1 : undefined}
                      onClick={(event) => handleNavLinkClick(event, link, true)}
                      className={linkClassName}
                    >
                      {t(link.title)}
                    </a>
                  )
                }
                return (
                  <Link
                    key={`${link.title}:${link.href}`}
                    to={link.href}
                    disabled={link.disabled}
                    onClick={(event) => handleNavLinkClick(event, link, true)}
                    className={linkClassName}
                  >
                    {t(link.title)}
                  </Link>
                )
              })}
            </nav>

            {showAuthButtons && (
              <div className='border-border flex flex-col gap-2 border-t pt-6'>
                {isAuthenticated ? (
                  <Link
                    to='/dashboard'
                    onClick={() => setMobileOpen(false)}
                    className='bg-primary text-primary-foreground inline-flex h-11 items-center justify-center rounded-md text-sm font-semibold transition-opacity hover:opacity-90'
                  >
                    {t('Dashboard')}
                  </Link>
                ) : (
                  <>
                    <Link
                      to='/sign-up'
                      onClick={() => setMobileOpen(false)}
                      className='bg-primary text-primary-foreground inline-flex h-11 items-center justify-center rounded-md text-sm font-semibold transition-opacity hover:opacity-90'
                    >
                      {t('Sign up')}
                    </Link>
                    <Link
                      to='/sign-in'
                      onClick={() => setMobileOpen(false)}
                      className='border-border text-foreground hover:bg-muted inline-flex h-11 items-center justify-center rounded-md border text-sm font-medium transition-colors'
                    >
                      {t('Login')}
                    </Link>
                  </>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      <Dialog
        open={!!authPromptTarget}
        onOpenChange={(open) => {
          if (!open) {
            closeAuthPrompt()
          }
        }}
        title={t('Sign in required')}
        description={t('Please sign in to view {{module}}.', {
          module: authPromptTarget?.title || '',
        })}
        contentClassName='sm:max-w-md'
        contentHeight='auto'
        footer={
          <>
            <Button variant='outline' onClick={closeAuthPrompt}>
              {t('Cancel')}
            </Button>
            <Button onClick={navigateToSignIn}>{t('Sign in now')}</Button>
          </>
        }
      >
        <div className='bg-muted text-muted-foreground rounded-md px-3 py-2 text-sm'>
          {t('Redirecting to sign in in {{seconds}} seconds.', {
            seconds: authPromptSecondsLeft,
          })}
        </div>
      </Dialog>
    </>
  )
}
