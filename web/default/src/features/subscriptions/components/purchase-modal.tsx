import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { Crown } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'

import { createSubscriptionCheckout, validateSubscriptionCoupon } from '../api'
import type {
  CouponValidationResponse,
  PlanRecord,
  SubscriptionCheckoutMode,
} from '../types'
import { CouponConfirmModal } from './coupon-confirm-modal'

interface PurchaseModalProps {
  plan: PlanRecord
  currency: 'CNY' | 'USD'
  open: boolean
  onOpenChange: (open: boolean) => void
}

const purchaseFormSchema = z.object({
  coupon_code: z.string().trim(),
})

type PurchaseFormValues = z.infer<typeof purchaseFormSchema>

function currencySymbol(currency: string): string {
  return currency.toUpperCase() === 'CNY' ? '¥' : '$'
}

function formatMoney(amount: number, currency: string): string {
  return `${currencySymbol(currency)}${amount.toFixed(2)}`
}

function parsePrepaidMonths(raw: string | undefined): number[] {
  if (!raw) return [1]
  const months = raw
    .split(',')
    .map((s) => Number.parseInt(s.trim(), 10))
    .filter((n) => Number.isFinite(n) && n > 0)
  return months.length > 0 ? months : [1]
}

export function PurchaseModal(props: PurchaseModalProps) {
  const { t } = useTranslation()
  const plan = props.plan.plan
  const prepaidOptions = useMemo(
    () => parsePrepaidMonths(plan.prepaid_months),
    [plan.prepaid_months]
  )

  const [mode, setMode] = useState<SubscriptionCheckoutMode>(
    plan.auto_renew ? 'auto_renew' : 'prepaid'
  )
  const [prepaidMonths, setPrepaidMonths] = useState<number>(prepaidOptions[0])
  const [confirmedCoupon, setConfirmedCoupon] =
    useState<CouponValidationResponse | null>(null)
  const [pendingValidation, setPendingValidation] =
    useState<CouponValidationResponse | null>(null)
  const [couponConfirmOpen, setCouponConfirmOpen] = useState(false)
  const [couponError, setCouponError] = useState('')
  // isRedirecting stays true from the moment we receive a checkout_url until
  // the browser finishes navigating to Stripe. Without it, the button would
  // briefly re-enable between mutation success (isPending -> false) and the
  // actual page unload, inviting a second click that creates a duplicate
  // Stripe Checkout Session.
  const [isRedirecting, setIsRedirecting] = useState(false)
  // conflictManageUrl is populated when the backend returns 409 with a
  // manage_url (active auto-renew subscription already exists, or a pending
  // checkout is in progress). Rendered as a link so the user can navigate to
  // their existing subscription without guessing where to go.
  const [conflictManageUrl, setConflictManageUrl] = useState('')

  const form = useForm<PurchaseFormValues>({
    resolver: zodResolver(purchaseFormSchema),
    defaultValues: { coupon_code: '' },
  })

  // Reset internal state whenever a different plan/modal session opens.
  useEffect(() => {
    if (props.open) {
      setMode(plan.auto_renew ? 'auto_renew' : 'prepaid')
      setPrepaidMonths(prepaidOptions[0])
      setConfirmedCoupon(null)
      setPendingValidation(null)
      setCouponError('')
      setIsRedirecting(false)
      setConflictManageUrl('')
      form.reset({ coupon_code: '' })
    }
  }, [props.open, plan.auto_renew, prepaidOptions, form])

  // Mode / term changes invalidate any previously confirmed coupon and clear
  // any conflict state (the conflict was scoped to the previous mode/term).
  useEffect(() => {
    setConfirmedCoupon(null)
    setCouponError('')
    setConflictManageUrl('')
  }, [mode, prepaidMonths, props.currency])

  const monthlyPrice =
    props.currency === 'CNY'
      ? (plan.price_amount_cny ?? plan.price_amount)
      : (plan.price_amount_usd ?? plan.price_amount)

  const originalPrice =
    confirmedCoupon?.original_price ??
    monthlyPrice * (mode === 'prepaid' ? prepaidMonths : 1)
  const discountAmount = confirmedCoupon?.discount_amount ?? 0
  const finalAmount = confirmedCoupon?.final_amount ?? originalPrice
  const breakdownCurrency = confirmedCoupon?.currency ?? props.currency

  const validateMutation = useMutation({
    mutationFn: (code: string) =>
      validateSubscriptionCoupon({
        code,
        plan_id: plan.id,
        mode,
        currency: props.currency,
        prepaid_months: mode === 'prepaid' ? prepaidMonths : undefined,
      }),
    onSuccess: (res) => {
      const data = res.data
      if (!data) {
        setCouponError(t('Validation failed'))
        return
      }
      if (data.valid) {
        setCouponError('')
        setPendingValidation(data)
        setCouponConfirmOpen(true)
      } else {
        setCouponError(data.reason || t('Invalid coupon code'))
      }
    },
  })

  const checkoutMutation = useMutation({
    mutationFn: () =>
      createSubscriptionCheckout({
        plan_id: plan.id,
        mode,
        currency: props.currency,
        prepaid_months: mode === 'prepaid' ? prepaidMonths : undefined,
        coupon_code: confirmedCoupon
          ? form.getValues('coupon_code')
          : undefined,
      }),
    onSuccess: (res) => {
      const url = res.data?.checkout_url
      if (!url) {
        toast.error(res.message || t('Checkout failed'))
        return
      }
      // Lock the button against a second click from the moment we receive
      // the checkout_url until the browser actually unloads the page. The
      // global axios interceptor does not toast on 2xx, so a missing-url
      // response is the only soft-failure path here.
      setIsRedirecting(true)
      window.location.assign(url)
    },
    onError: (error: unknown) => {
      // 409 Conflict: the backend reports an active subscription already
      // exists (auto-renew duplicate) or a pending checkout is in progress
      // (rapid double-click). The global interceptor already toasts the
      // message; surface the manage_url as a link so the user can act on
      // it without dismissing the modal first.
      const status =
        (error as { response?: { status?: number } })?.response?.status ?? 0
      if (status === 409) {
        const data = (
          error as { response?: { data?: { manage_url?: string } } }
        )?.response?.data
        if (data?.manage_url) {
          setConflictManageUrl(data.manage_url)
        }
      }
    },
  })

  // The button stays disabled across both phases of the checkout flow: the
  // API call itself (isPending) and the subsequent redirect to Stripe
  // (isRedirecting). A conflict (409) does NOT keep the button disabled —
  // the user can dismiss the conflict and retry after managing their
  // existing subscription.
  const isCheckoutProcessing =
    checkoutMutation.isPending || isRedirecting

  const handleValidateCoupon = form.handleSubmit(async (values) => {
    const code = values.coupon_code.trim()
    if (!code) {
      setCouponError(t('Please enter a coupon code'))
      return
    }
    setCouponError('')
    validateMutation.mutate(code)
  })

  return (
    <>
      <Dialog
        open={props.open}
        onOpenChange={props.onOpenChange}
        title={
          <>
            <Crown className='h-5 w-5' />
            {plan.title}
          </>
        }
        description={t(
          'Unlimited access to included models. Models outside the plan continue to use your account balance.'
        )}
        contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-lg'
        titleClassName='flex items-center gap-2'
        contentHeight='auto'
        bodyClassName='space-y-4'
        footer={
          <div className='flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center'>
            <Button
              className='w-full sm:w-auto'
              onClick={() => checkoutMutation.mutate()}
              disabled={isCheckoutProcessing}
            >
              {isCheckoutProcessing
                ? t('Processing...')
                : t('Go to Stripe Checkout')}
            </Button>
            {conflictManageUrl && (
              <a
                href={conflictManageUrl}
                className='text-primary text-sm font-medium underline underline-offset-4'
              >
                {t('Manage subscription')}
              </a>
            )}
          </div>
        }
      >
        {/* Mode selector */}
        <div className='space-y-2'>
          <Label>{t('Billing mode')}</Label>
          <RadioGroup
            value={mode}
            onValueChange={(v) => v && setMode(v as SubscriptionCheckoutMode)}
            className='grid grid-cols-1 gap-2 sm:grid-cols-2'
          >
            <label className='border-input has-data-[slot=radio-group-item]:checked:border-primary has-data-[slot=radio-group-item]:checked:bg-primary/5 flex cursor-pointer items-start gap-2 rounded-lg border p-3'>
              <RadioGroupItem value='auto_renew' className='mt-0.5' />
              <div className='space-y-0.5'>
                <div className='text-sm font-medium'>{t('Auto-renew')}</div>
                <div className='text-muted-foreground text-xs'>
                  {formatMoney(monthlyPrice, props.currency)}
                  {' / '}
                  {t('mo')} ·{' '}
                  {t('Automatically renews monthly. Cancel anytime.')}
                </div>
              </div>
            </label>
            <label className='border-input has-data-[slot=radio-group-item]:checked:border-primary has-data-[slot=radio-group-item]:checked:bg-primary/5 flex cursor-pointer items-start gap-2 rounded-lg border p-3'>
              <RadioGroupItem value='prepaid' className='mt-0.5' />
              <div className='space-y-0.5'>
                <div className='text-sm font-medium'>{t('Prepaid')}</div>
                <div className='text-muted-foreground text-xs'>
                  {t('One-time payment')}
                </div>
              </div>
            </label>
          </RadioGroup>
        </div>

        {mode === 'prepaid' && (
          <div className='space-y-2'>
            <Label>{t('Prepaid months')}</Label>
            <Select
              value={String(prepaidMonths)}
              onValueChange={(v) =>
                v !== null && setPrepaidMonths(Number.parseInt(v, 10))
              }
              items={prepaidOptions.map((m) => ({
                value: String(m),
                label: String(m),
              }))}
            >
              <SelectTrigger className='w-full'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {prepaidOptions.map((m) => (
                    <SelectItem key={m} value={String(m)}>
                      {m} {t('months')}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        )}

        {/* Coupon */}
        <div className='space-y-2'>
          <Label htmlFor='coupon_code'>{t('Coupon code')}</Label>
          <div className='flex gap-2'>
            <Input
              id='coupon_code'
              autoComplete='off'
              placeholder={t('Enter coupon code')}
              {...form.register('coupon_code')}
            />
            <Button
              type='button'
              variant='outline'
              onClick={handleValidateCoupon}
              disabled={validateMutation.isPending}
            >
              {validateMutation.isPending ? t('Validating...') : t('Validate')}
            </Button>
          </div>
          {couponError && (
            <p className='text-destructive text-xs'>{couponError}</p>
          )}
          {confirmedCoupon && (
            <Alert>
              <AlertDescription className='text-xs'>
                {confirmedCoupon.coupon_name} · {confirmedCoupon.percent_off}%{' '}
                {t('off')} —{' '}
                <button
                  type='button'
                  className='text-primary underline'
                  onClick={() => {
                    setConfirmedCoupon(null)
                    form.reset({ coupon_code: '' })
                  }}
                >
                  {t('Remove')}
                </button>
              </AlertDescription>
            </Alert>
          )}
        </div>

        <Separator />

        {/* Price breakdown */}
        <div className='space-y-2'>
          <div className='flex items-center justify-between text-sm'>
            <span className='text-muted-foreground'>{t('Original price')}</span>
            <span className='font-mono tabular-nums'>
              {formatMoney(originalPrice, breakdownCurrency)}
            </span>
          </div>
          {discountAmount > 0 && (
            <div className='flex items-center justify-between text-sm'>
              <span className='text-muted-foreground'>
                {t('Discount amount')}
              </span>
              <span className='text-destructive font-mono tabular-nums'>
                -{formatMoney(discountAmount, breakdownCurrency)}
              </span>
            </div>
          )}
          <div className='flex items-center justify-between text-sm'>
            <span className='font-medium'>{t('Final amount')}</span>
            <span className='text-primary font-mono text-lg font-bold tabular-nums'>
              {formatMoney(finalAmount, breakdownCurrency)}
            </span>
          </div>
        </div>

        {mode === 'auto_renew' && (
          <p className='text-muted-foreground text-xs'>
            {t(
              'You will be charged {{amount}} {{currency}} monthly until you cancel.',
              {
                amount: formatMoney(finalAmount, breakdownCurrency),
                currency: breakdownCurrency,
              }
            )}
          </p>
        )}
      </Dialog>

      {pendingValidation && (
        <CouponConfirmModal
          validation={pendingValidation}
          mode={mode}
          open={couponConfirmOpen}
          onOpenChange={setCouponConfirmOpen}
          onConfirm={() => {
            setConfirmedCoupon(pendingValidation)
            setCouponConfirmOpen(false)
          }}
          onCancel={() => {
            setCouponConfirmOpen(false)
          }}
        />
      )}
    </>
  )
}
