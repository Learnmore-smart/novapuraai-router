import { Link, useSearch } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { useStatus } from '@/hooks/use-status'

import { AuthLayout } from '../auth-layout'
import { TermsFooter } from '../components/terms-footer'
import { GUEST_ONLY_LOGIN } from '../constants'
import { GuestContinueButton } from './components/guest-continue-button'
import { UserAuthForm } from './components/user-auth-form'

export function SignIn() {
  const { t } = useTranslation()
  const { redirect } = useSearch({ from: '/(auth)/sign-in' })
  const { status } = useStatus()

  return (
    <AuthLayout>
      <div className='w-full space-y-7'>
        <div className='space-y-2'>
          <h2 className='text-2xl font-semibold tracking-tight'>
            {t('Sign in to your account')}
          </h2>
          {!GUEST_ONLY_LOGIN &&
            !status?.self_use_mode_enabled &&
            status?.register_enabled !== false && (
              <p className='text-muted-foreground text-sm'>
                {t("Don't have an account?")}{' '}
                <Link
                  to='/sign-up'
                  className='text-primary font-medium underline-offset-4 hover:underline'
                >
                  {t('Sign up')}
                </Link>
              </p>
            )}
        </div>

        {GUEST_ONLY_LOGIN ? (
          <GuestContinueButton />
        ) : (
          <UserAuthForm redirectTo={redirect} />
        )}

        <TermsFooter
          variant='sign-in'
          status={status}
          className='text-center'
        />
      </div>
    </AuthLayout>
  )
}
