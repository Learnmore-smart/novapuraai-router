import { Tag } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'

import type { CouponValidationResponse } from '../types'

interface CouponConfirmModalProps {
  validation: CouponValidationResponse
  mode: 'auto_renew' | 'prepaid'
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
  onCancel: () => void
}

function currencySymbol(currency: string): string {
  return currency.toUpperCase() === 'CNY' ? '¥' : '$'
}

function formatMoney(amount: number, currency: string): string {
  return `${currencySymbol(currency)}${amount.toFixed(2)}`
}

export function CouponConfirmModal(props: CouponConfirmModalProps) {
  const { t } = useTranslation()
  const v = props.validation

  const durationNote =
    props.mode === 'auto_renew'
      ? t('Discount lasts {{months}} months', { months: v.duration_months })
      : t('Applies to this payment only')

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={
        <>
          <Tag className='h-5 w-5' />
          {t('Coupon applied')}
        </>
      }
      description={t('Please confirm the discounted price before continuing.')}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'
      titleClassName='flex items-center gap-2'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button variant='outline' onClick={props.onCancel}>
            {t('Cancel')}
          </Button>
          <Button onClick={props.onConfirm}>{t('Confirm')}</Button>
        </>
      }
    >
      <div className='space-y-3'>
        <div className='bg-muted/50 space-y-2 rounded-lg border p-3 sm:p-4'>
          <div className='flex items-center justify-between'>
            <span className='text-muted-foreground text-sm'>
              {t('Coupon code')}
            </span>
            <span className='text-sm font-medium'>{v.coupon_name}</span>
          </div>
          <div className='flex items-center justify-between'>
            <span className='text-muted-foreground text-sm'>
              {t('Discount')}
            </span>
            <span className='text-primary text-sm font-semibold'>
              {v.percent_off}% {t('off')}
            </span>
          </div>
        </div>

        <div className='space-y-2'>
          <div className='flex items-center justify-between text-sm'>
            <span className='text-muted-foreground'>{t('Original price')}</span>
            <span className='font-mono tabular-nums'>
              {formatMoney(v.original_price, v.currency)}
            </span>
          </div>
          <div className='flex items-center justify-between text-sm'>
            <span className='text-muted-foreground'>
              {t('Discount amount')}
            </span>
            <span className='text-destructive font-mono tabular-nums'>
              -{formatMoney(v.discount_amount, v.currency)}
            </span>
          </div>
          <Separator />
          <div className='flex items-center justify-between text-sm'>
            <span className='font-medium'>{t('Final amount')}</span>
            <span className='text-primary font-mono text-lg font-bold tabular-nums'>
              {formatMoney(v.final_amount, v.currency)}
            </span>
          </div>
          <div className='flex items-center justify-between text-sm'>
            <span className='text-muted-foreground'>
              {t('Next renewal price')}
            </span>
            <span className='font-mono tabular-nums'>
              {formatMoney(v.next_renewal_price, v.currency)}
            </span>
          </div>
        </div>

        <p className='text-muted-foreground text-xs'>{durationNote}</p>
      </div>
    </Dialog>
  )
}
