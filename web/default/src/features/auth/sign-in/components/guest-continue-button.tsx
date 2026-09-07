import { Link } from '@tanstack/react-router'
import { UserRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

export function GuestContinueButton() {
  const { t } = useTranslation()

  return (
    <Button
      className='h-11 w-full justify-center gap-2'
      render={<Link to='/' />}
    >
      <UserRound className='h-4 w-4' aria-hidden='true' />
      {t('Continue as guest')}
    </Button>
  )
}
