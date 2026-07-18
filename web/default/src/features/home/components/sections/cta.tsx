import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()

  if (props.isAuthenticated) return null

  return (
    <section className='px-5 pb-20 sm:px-6 md:pb-28'>
      <div className='np-surface-lg relative mx-auto max-w-5xl overflow-hidden px-6 py-14 text-center sm:px-10 md:py-20'>
        <div
          className='pointer-events-none absolute inset-0'
          style={{ backgroundImage: 'var(--np-hero-wash)' }}
          aria-hidden='true'
        />
        <div
          className='np-grid pointer-events-none absolute inset-0'
          aria-hidden='true'
        />
        <div className='relative mx-auto max-w-xl'>
          <p className='np-kicker'>{t('Start building')}</p>
          <h2 className='mt-3 text-3xl leading-tight font-semibold tracking-[-0.03em] sm:text-4xl'>
            {t('Create an account. Generate a key. Ship the next request.')}
          </h2>
          <p className='text-muted-foreground mx-auto mt-4 max-w-md text-sm leading-7 sm:text-base'>
            {t(
              'Pay only for what you use. Your balance and request ledger stay close at hand.'
            )}
          </p>
          <div className='mt-8 flex flex-wrap items-center justify-center gap-3'>
            <Button
              className='group h-11 rounded-md px-5 font-semibold'
              render={<Link to='/sign-up' />}
            >
              {t('Create your account')}
              <ArrowRight className='size-4 transition-transform group-hover:translate-x-0.5' />
            </Button>
            <Button
              variant='outline'
              className='h-11 rounded-md px-5'
              render={<Link to='/sign-in' />}
            >
              {t('Sign in')}
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
