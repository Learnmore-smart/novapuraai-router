import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { useStatus } from '@/hooks/use-status'
import { formatPublicCurrency, getRegisterPromo } from '@/lib/public-status'

import { AuthLayout } from '../auth-layout'
import { TermsFooter } from '../components/terms-footer'
import { SignUpForm } from './components/sign-up-form'

export function SignUp() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const registerPromo = getRegisterPromo(status)

  return (
    <AuthLayout>
      <div className='w-full space-y-7'>
        <div className='space-y-2'>
          <h2 className='text-2xl font-semibold tracking-tight'>
            {t('Create your NovaPura account')}
          </h2>
          <p className='text-muted-foreground text-sm leading-6'>
            {t(
              'Open a prepaid workspace, generate an API key, and start routing models through one compatible endpoint.'
            )}
          </p>
          {registerPromo && (
            <p className='text-muted-foreground text-sm leading-6'>
              {t('New accounts receive {{amount}} in API credits.', {
                amount:
                  formatPublicCurrency(
                    registerPromo.amount,
                    registerPromo.currency
                  ) || '—',
              })}
            </p>
          )}
          <p className='text-muted-foreground text-sm'>
            {t('Already have an account?')}{' '}
            <Link
              to='/sign-in'
              className='text-primary font-medium underline-offset-4 hover:underline'
            >
              {t('Sign in')}
            </Link>
          </p>
        </div>

        <SignUpForm />

        <TermsFooter
          variant='sign-up'
          status={status}
          className='text-center'
        />
      </div>
    </AuthLayout>
  )
}
