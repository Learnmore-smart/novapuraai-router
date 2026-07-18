import { Link, useSearch } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { api } from '@/lib/api'

type OrderStatus = {
  order_id: string
  status: string
  presentment_currency: string
  paid_display: string
  promo_display: string
  total_display: string
  promo_expires_at: number
  failure_reason?: string
}

export function StripeTopupSuccess() {
  const { t } = useTranslation()
  const search = useSearch({ strict: false }) as { order_id?: string }
  const orderId = search.order_id || ''
  const [order, setOrder] = useState<OrderStatus | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!orderId) {
      setError(t('Missing order ID'))
      return
    }
    let stopped = false
    let tries = 0
    const tick = async () => {
      try {
        const res = await api.get(`/api/billing/top-up/orders/${orderId}`)
        if (stopped) return
        if (res.data?.success) {
          const data = res.data.data as OrderStatus
          setOrder(data)
          if (
            [
              'credited',
              'failed',
              'expired',
              'manual_review',
              'refunded',
            ].includes(data.status)
          ) {
            return
          }
        }
      } catch {
        // Stripe redirects can arrive just before webhook fulfillment.
      }
      tries += 1
      if (!stopped && tries < 40) {
        window.setTimeout(() => void tick(), 1500)
      }
    }
    void tick()
    return () => {
      stopped = true
    }
  }, [orderId, t])

  const pending =
    !error &&
    (!order || ['pending', 'checkout_created', 'paid'].includes(order.status))

  return (
    <Card className='mx-auto mt-8 max-w-lg'>
      <CardHeader>
        <CardTitle>{t('Top-up status')}</CardTitle>
        <CardDescription>
          {t('The server is confirming the signed payment result.')}
        </CardDescription>
      </CardHeader>
      <CardContent className='flex flex-col gap-4' aria-live='polite'>
        {error ? <p className='text-destructive text-sm'>{error}</p> : null}
        {pending ? (
          <div className='flex items-center gap-2 text-sm'>
            <Spinner />
            {t('Confirming payment')}
          </div>
        ) : null}
        {order?.status === 'credited' ? (
          <div className='bg-muted/40 flex flex-col gap-3 rounded-xl p-4'>
            <div className='flex items-center justify-between gap-3'>
              <p className='font-medium'>{t('Credits added')}</p>
              <Badge>{order.presentment_currency.toUpperCase()}</Badge>
            </div>
            <dl className='grid grid-cols-[1fr_auto] gap-x-4 gap-y-2 text-sm'>
              <dt className='text-muted-foreground'>{t('Paid credits')}</dt>
              <dd className='font-mono tabular-nums'>{order.paid_display}</dd>
              <dt className='text-muted-foreground'>
                {t('Promotional bonus')}
              </dt>
              <dd className='font-mono tabular-nums'>+{order.promo_display}</dd>
              <dt className='font-medium'>{t('Total credits')}</dt>
              <dd className='font-mono font-semibold tabular-nums'>
                {order.total_display}
              </dd>
            </dl>
            <p className='text-muted-foreground text-xs'>
              {order.promo_expires_at > 0
                ? t('Promotional credits expire on {{date}}.', {
                    date: new Date(
                      order.promo_expires_at * 1000
                    ).toLocaleDateString(),
                  })
                : t('Promotional credits do not expire.')}{' '}
              {t('They cannot be withdrawn or transferred.')}
            </p>
          </div>
        ) : null}
        {order &&
        ['failed', 'expired', 'manual_review'].includes(order.status) ? (
          <p className='text-destructive text-sm'>
            {order.failure_reason || t('Payment was not credited.')}
          </p>
        ) : null}
        {order?.status === 'refunded' ? (
          <p className='text-muted-foreground text-sm'>
            {t(
              'This payment was refunded and its remaining source credits were reversed.'
            )}
          </p>
        ) : null}
      </CardContent>
      <CardFooter>
        <Button
          render={<Link to='/wallet' />}
          nativeButton={false}
          variant='outline'
        >
          {t('Back to wallet')}
        </Button>
      </CardFooter>
    </Card>
  )
}
