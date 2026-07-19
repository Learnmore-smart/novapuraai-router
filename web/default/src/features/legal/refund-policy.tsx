import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

/** Static refund policy shell (MVP §1.2 / §13 legal). Admin CMS can replace later. */
export function RefundPolicy() {
  const { t } = useTranslation()
  return (
    <PublicLayout>
      <div className='mx-auto max-w-3xl px-4 py-10'>
        <Card>
          <CardHeader>
            <CardTitle>{t('Refund Policy')}</CardTitle>
          </CardHeader>
          <CardContent className='prose dark:prose-invert max-w-none space-y-3 text-sm leading-relaxed'>
            <p>
              {t(
                'Cash top-ups may be refunded only when required by applicable law or when an order fails after payment. Gift (promo) balance is non-withdrawable and non-transferable.'
              )}
            </p>
            <p>
              {t(
                'Successful API usage that has already been billed is non-refundable. Contact support@novapuraai.com for refund requests with your order ID.'
              )}
            </p>
            <p>
              {t(
                'This page is a product default. Site operators should customize the final legal text before public launch.'
              )}
            </p>
          </CardContent>
        </Card>
      </div>
    </PublicLayout>
  )
}
