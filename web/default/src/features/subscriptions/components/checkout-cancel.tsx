import { Link } from '@tanstack/react-router'
import { XCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

export function CheckoutCancel() {
  const { t } = useTranslation()

  return (
    <PublicLayout>
      <Card className='mx-auto mt-8 max-w-lg'>
        <CardHeader className='text-center'>
          <div className='bg-muted mx-auto mb-2 flex size-12 items-center justify-center rounded-full'>
            <XCircle className='text-muted-foreground h-6 w-6' />
          </div>
          <CardTitle>{t('Payment cancelled')}</CardTitle>
          <CardDescription>
            {t('Your checkout session was cancelled. No charge was made.')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className='text-muted-foreground text-center text-sm'>
            {t('You can try subscribing again at any time.')}
          </p>
        </CardContent>
        <CardFooter className='flex justify-center gap-2'>
          <Button render={<Link to='/plans' />}>{t('Back to plans')}</Button>
        </CardFooter>
      </Card>
    </PublicLayout>
  )
}
