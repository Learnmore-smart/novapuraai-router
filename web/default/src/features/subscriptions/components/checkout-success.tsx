import { Link } from '@tanstack/react-router'
import { CheckCircle2 } from 'lucide-react'
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

export function CheckoutSuccess() {
  const { t } = useTranslation()

  return (
    <PublicLayout>
      <Card className='mx-auto mt-8 max-w-lg'>
        <CardHeader className='text-center'>
          <div className='bg-primary/10 mx-auto mb-2 flex size-12 items-center justify-center rounded-full'>
            <CheckCircle2 className='text-primary h-6 w-6' />
          </div>
          <CardTitle>{t('Payment successful')}</CardTitle>
          <CardDescription>
            {t(
              'Your subscription is being confirmed. You will receive a confirmation shortly.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className='text-muted-foreground text-center text-sm'>
            {t(
              'If your subscription is not active yet, please wait a moment and refresh this page.'
            )}
          </p>
        </CardContent>
        <CardFooter className='flex justify-center gap-2'>
          <Button render={<Link to='/wallet' />} variant='outline'>
            {t('Back to wallet')}
          </Button>
          <Button render={<Link to='/plans' />}>{t('Back to plans')}</Button>
        </CardFooter>
      </Card>
    </PublicLayout>
  )
}
