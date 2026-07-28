import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AuthLayout } from '../auth-layout'
import { ForgotPasswordForm } from './components/forgot-password-form'

export function ForgotPassword() {
  const { t } = useTranslation()
  return (
    <AuthLayout>
      <div className='w-full space-y-8'>
        <div className='border-border space-y-3 border-b pb-6'>
          <p className='editorial-kicker'>{t('Account recovery')}</p>
          <h2 className='text-2xl font-semibold tracking-tight'>
            {t('Reset your password')}
          </h2>
          <p className='text-muted-foreground text-sm leading-6'>
            {t(
              'Enter the email on your NovaPura account. If it is registered, we will send a secure link to choose a new password.'
            )}
          </p>
          <p className='text-muted-foreground text-sm'>
            {t('Remembered it?')}{' '}
            <Link
              to='/sign-in'
              className='text-foreground font-medium underline underline-offset-4'
            >
              {t('Sign in')}
            </Link>
          </p>
        </div>

        <ForgotPasswordForm className='space-y-0' />
      </div>
    </AuthLayout>
  )
}
