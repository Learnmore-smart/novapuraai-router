import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useSearch } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { api } from '@/lib/api'

type OrderStatus = {
  order_id: string
  status: string
  paid_quota: number
  promo_quota: number
  total_quota: number
  failure_reason?: string
}

/**
 * Success redirect does NOT prove payment. Poll server until credited/failed.
 */
export function StripeTopupSuccess() {
  const { t } = useTranslation()
  const search = useSearch({ strict: false }) as { order_id?: string }
  const orderId = search.order_id || ''
  const [order, setOrder] = useState<OrderStatus | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!orderId) {
      setError('missing order_id')
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
            data.status === 'credited' ||
            data.status === 'failed' ||
            data.status === 'expired' ||
            data.status === 'manual_review' ||
            data.status === 'refunded'
          ) {
            return
          }
        }
      } catch {
        // keep polling briefly
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
  }, [orderId])

  const pending =
    !error &&
    (!order ||
      order.status === 'pending' ||
      order.status === 'checkout_created' ||
      order.status === 'paid')

  return (
    <Card className="max-w-lg mx-auto mt-8">
      <CardHeader>
        <CardTitle>{t('Top-up status')}</CardTitle>
        <CardDescription>
          {t('Confirming payment with the server. Do not close this page.')}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {error ? <p className="text-destructive text-sm">{error}</p> : null}
        {pending ? (
          <p className="text-sm">{t('Confirming payment…')}</p>
        ) : null}
        {order?.status === 'credited' ? (
          <div className="text-sm space-y-1">
            <p className="text-green-600 font-medium">{t('Credits added')}</p>
            <p>
              {t('Total credits')}: {order.total_quota}
            </p>
            <p>
              {t('Paid')}: {order.paid_quota} / {t('Promo')}: {order.promo_quota}
            </p>
          </div>
        ) : null}
        {order &&
        ['failed', 'expired', 'manual_review'].includes(order.status) ? (
          <p className="text-sm text-destructive">
            {order.failure_reason || order.status}
          </p>
        ) : null}
        <Button asChild variant="outline">
          <Link to="/wallet">{t('Back to wallet')}</Link>
        </Button>
      </CardContent>
    </Card>
  )
}
