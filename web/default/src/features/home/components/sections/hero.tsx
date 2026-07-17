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
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { HeroTerminalDemo } from '../hero-terminal-demo'

const PROOF_POINTS = [
  'OpenAI-compatible',
  'Prepaid balances',
  'Usage per request',
] as const

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()

  return (
    <section className='np-hero-wash relative overflow-hidden px-5 pt-28 pb-16 sm:px-6 md:pt-36 md:pb-24'>
      <div
        className='np-grid pointer-events-none absolute inset-0'
        aria-hidden='true'
      />

      <div className='relative mx-auto max-w-3xl text-center'>
        <Link
          to='/pricing'
          className='np-fade-up border-border bg-card/70 text-muted-foreground hover:text-foreground hover:border-primary/30 group inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium backdrop-blur-sm transition-colors'
        >
          <span className='bg-primary size-1.5 rounded-full' aria-hidden='true' />
          {t('40+ model providers, one endpoint')}
          <ArrowRight className='size-3 transition-transform group-hover:translate-x-0.5' />
        </Link>

        <h1 className='np-fade-up np-fade-up-delay-1 mx-auto mt-6 max-w-2xl text-[clamp(2.4rem,6vw,4rem)] leading-[1.03] font-semibold tracking-[-0.04em]'>
          {t('One compatible API for every model you ship.')}
        </h1>

        <p className='text-muted-foreground np-fade-up np-fade-up-delay-2 mx-auto mt-6 max-w-xl text-base leading-7 sm:text-lg sm:leading-8'>
          {t(
            'A prepaid gateway with health-aware routing and a request ledger you can audit. Point your existing OpenAI-style client at NovaPura and ship.'
          )}
        </p>

        <div className='np-fade-up np-fade-up-delay-3 mt-9 flex flex-wrap items-center justify-center gap-3'>
          <Button
            className='group h-11 rounded-md px-5 font-semibold'
            render={
              <Link to={props.isAuthenticated ? '/dashboard' : '/sign-up'} />
            }
          >
            {props.isAuthenticated
              ? t('Open dashboard')
              : t('Create your account')}
            <ArrowRight className='size-4 transition-transform group-hover:translate-x-0.5' />
          </Button>
          <Button
            variant='outline'
            className='h-11 rounded-md px-4'
            render={<Link to='/pricing' />}
          >
            {t('View pricing')}
          </Button>
        </div>

        <ul className='np-fade-up np-fade-up-delay-3 mt-8 flex flex-wrap items-center justify-center gap-x-5 gap-y-2'>
          {PROOF_POINTS.map((point) => (
            <li
              key={point}
              className='text-muted-foreground flex items-center gap-2 text-sm'
            >
              <span
                className='bg-primary size-1.5 shrink-0 rounded-full'
                aria-hidden='true'
              />
              {t(point)}
            </li>
          ))}
        </ul>
      </div>

      <div className='np-fade-up np-fade-up-delay-3 relative mx-auto mt-16 w-full max-w-4xl'>
        <HeroTerminalDemo />
      </div>
    </section>
  )
}
