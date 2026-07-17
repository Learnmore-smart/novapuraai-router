/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

/** Static refund policy shell (MVP §1.2 / §13 legal). Admin CMS can replace later. */
export function RefundPolicy() {
  const { t } = useTranslation()
  return (
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
  )
}
