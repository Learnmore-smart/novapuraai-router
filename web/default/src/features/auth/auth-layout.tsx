import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { BrandMark } from '@/components/layout'
import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'
import { isDefaultBrandLogo } from '@/lib/dom-utils'
import { cn } from '@/lib/utils'

type AuthLayoutProps = {
  children: React.ReactNode
}

const VALUE_PROPS = [
  'Keep your existing OpenAI-style client and point it at NovaPura.',
  'Cash and promotional credit stay separate, so spend is explainable.',
  'Every request leaves a ledger entry: model, tokens, latency, cost.',
] as const

export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()
  const stockLogo = isDefaultBrandLogo(logo)

  return (
    <div className='bg-background grid min-h-svh lg:grid-cols-[minmax(0,0.92fr)_minmax(0,1.08fr)]'>
      <aside className='np-hero-wash border-border relative hidden flex-col justify-between overflow-hidden border-r p-10 lg:flex xl:p-12'>
        <div
          className='np-grid pointer-events-none absolute inset-0'
          aria-hidden='true'
        />

        <Link to='/' className='relative flex items-center gap-2.5'>
          <div
            className={cn(
              'relative size-9',
              !stockLogo && 'overflow-hidden rounded-md'
            )}
          >
            {loading ? (
              <Skeleton className='absolute inset-0 rounded-md' />
            ) : (
              <BrandMark src={logo} alt={t('Logo')} className='size-9' />
            )}
          </div>
          {loading ? (
            <Skeleton className='h-5 w-28' />
          ) : (
            <span className='text-lg font-semibold tracking-tight'>
              {systemName}
            </span>
          )}
        </Link>

        <div className='relative max-w-md'>
          <p className='np-kicker'>{t('AI API infrastructure')}</p>
          <h1 className='mt-4 text-3xl leading-tight font-semibold tracking-[-0.03em]'>
            {t('One compatible endpoint. Prepaid control. Clear usage.')}
          </h1>
          <ul className='mt-8 space-y-4'>
            {VALUE_PROPS.map((line) => (
              <li
                key={line}
                className='text-muted-foreground flex gap-3 text-sm leading-6'
              >
                <span
                  className='bg-primary mt-2 size-1.5 shrink-0 rounded-full'
                  aria-hidden='true'
                />
                {t(line)}
              </li>
            ))}
          </ul>

          <div className='np-surface mt-10 overflow-hidden'>
            <div className='border-border flex items-center justify-between border-b px-4 py-2.5'>
              <span className='text-muted-foreground text-[0.6875rem] font-semibold tracking-wide uppercase'>
                {t('Endpoint')}
              </span>
              <span className='text-success bg-success/10 inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[0.625rem] font-semibold'>
                <span className='bg-success size-1.5 animate-pulse rounded-full' />
                {t('Ready')}
              </span>
            </div>
            <p className='px-4 py-3 font-mono text-xs leading-5 break-all'>
              <span className='text-primary'>POST</span> /v1/chat/completions
            </p>
          </div>
        </div>

        <p className='text-muted-foreground relative text-xs'>
          {t('Pay only for what you use. No subscription required to start.')}
        </p>
      </aside>

      <div className='relative flex flex-col'>
        <Link
          to='/'
          className='absolute top-5 left-5 z-10 flex items-center gap-2 lg:hidden'
        >
          <div
            className={cn(
              'relative size-8',
              !stockLogo && 'overflow-hidden rounded-md'
            )}
          >
            {loading ? (
              <Skeleton className='absolute inset-0 rounded-md' />
            ) : (
              <BrandMark src={logo} alt={t('Logo')} className='size-8' />
            )}
          </div>
          {loading ? (
            <Skeleton className='h-5 w-24' />
          ) : (
            <span className='text-base font-semibold tracking-tight'>
              {systemName}
            </span>
          )}
        </Link>

        <div className='flex flex-1 items-center justify-center px-4 py-20 sm:px-8'>
          <div className='w-full max-w-[26rem]'>{children}</div>
        </div>
      </div>
    </div>
  )
}
