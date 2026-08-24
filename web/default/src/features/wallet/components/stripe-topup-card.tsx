import { useQuery } from '@tanstack/react-query'
import { CreditCard } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Field,
  FieldDescription,
  FieldLabel,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import type { BillingCurrency } from '@/lib/billing-currency'

import { getBillingTopupQuote } from '../api'
import { useStripeTopup } from '../hooks/use-stripe-topup'
import type { BillingTopupOffer } from '../types'

const CURRENCY_ITEMS: Array<{ label: string; value: BillingCurrency }> = [
  { label: 'CNY', value: 'cny' },
  { label: 'USD', value: 'usd' },
  { label: 'CAD', value: 'cad' },
]

function unavailableMessage(
  offer: BillingTopupOffer,
  translate: (key: string) => string
) {
  switch (offer.unavailable_reason) {
    case 'currency_unavailable':
      return translate('This currency is currently unavailable.')
    default:
      return translate('This top-up amount is currently unavailable.')
  }
}

function TopupCardSkeleton() {
  return (
    <Card aria-live='polite'>
      <CardHeader>
        <Skeleton className='h-5 w-44' />
        <Skeleton className='h-4 w-72 max-w-full' />
      </CardHeader>
      <CardContent className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        {[0, 1, 2, 3].map((item) => (
          <Skeleton key={item} className='h-36 w-full rounded-xl' />
        ))}
      </CardContent>
    </Card>
  )
}

export function StripeTopupCard() {
  const { t } = useTranslation()
  const topup = useStripeTopup()
  const [customAmount, setCustomAmount] = useState('')
  const [customSelected, setCustomSelected] = useState(false)
  const [quotedAmountMinor, setQuotedAmountMinor] = useState<number | null>(
    null
  )

  useEffect(() => {
    setCustomAmount('')
    setCustomSelected(false)
    setQuotedAmountMinor(null)
  }, [topup.selectedCurrency])

  const enabledCurrencyItems = CURRENCY_ITEMS.filter((item) =>
    topup.config?.config?.currencies?.includes(item.value)
  )
  const selectedOffer = topup.offers.find(
    (offer) => offer.payment_amount_minor === topup.selectedAmountMinor
  )
  const [minimumMinor, maximumMinor] = topup.config?.config?.min_max_minor?.[
    topup.selectedCurrency
  ] ?? [0, 0]
  const minimumDisplay = (minimumMinor / 100).toFixed(2)
  const maximumDisplay = (maximumMinor / 100).toFixed(2)
  const customAmountMatch = customAmount.trim().match(/^(\d+)(?:\.(\d{1,2}))?$/)
  let customAmountMinor: number | null = null
  if (customAmountMatch) {
    const whole = Number(customAmountMatch[1])
    const fraction = Number((customAmountMatch[2] ?? '').padEnd(2, '0'))
    if (Number.isSafeInteger(whole) && Number.isSafeInteger(fraction)) {
      const minor = whole * 100 + fraction
      if (Number.isSafeInteger(minor)) customAmountMinor = minor
    }
  }
  const customAmountValid =
    customAmountMinor != null &&
    customAmountMinor >= minimumMinor &&
    customAmountMinor <= maximumMinor
  useEffect(() => {
    if (!customSelected || !customAmountValid || customAmountMinor == null) {
      setQuotedAmountMinor(null)
      return
    }
    const timer = window.setTimeout(
      () => setQuotedAmountMinor(customAmountMinor),
      300
    )
    return () => window.clearTimeout(timer)
  }, [customAmountMinor, customAmountValid, customSelected])
  const customQuoteQuery = useQuery({
    queryKey: [
      'billing-topup-quote',
      topup.selectedCurrency,
      quotedAmountMinor,
    ],
    queryFn: () =>
      getBillingTopupQuote({
        currency: topup.selectedCurrency,
        amount_minor: quotedAmountMinor as number,
      }),
    enabled: quotedAmountMinor != null,
    staleTime: 30_000,
  })
  const customQuote = customQuoteQuery.data?.data
  if (topup.isLoading) return <TopupCardSkeleton />
  if (!topup.config?.config.enabled) return null
  const checkoutAmountMinor = customSelected
    ? customAmountMinor
    : topup.selectedAmountMinor
  let selectionSummary: ReactNode = t('Select an available offer')
  if (customSelected) {
    if (customQuote) {
      selectionSummary = (
        <>
          {t('Pay')} {customQuote.payment_display} · {t('Receive')}{' '}
          <strong>{customQuote.total_display}</strong>
        </>
      )
    } else if (customAmountValid) {
      selectionSummary = t('Calculating top-up')
    } else {
      selectionSummary = t('Please enter a valid number')
    }
  } else if (selectedOffer) {
    selectionSummary = (
      <>
        {t('Pay')} {selectedOffer.payment_display} · {t('Receive')}{' '}
        <strong>{selectedOffer.total_display}</strong>
      </>
    )
  }

  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle className='flex flex-wrap items-center gap-2'>
          <CreditCard aria-hidden='true' />
          {t('Add API credits')}
          <Badge variant='secondary'>{t('1:1 credit')}</Badge>
          {topup.config.sandbox ? (
            <Badge variant='outline'>{t('Sandbox')}</Badge>
          ) : null}
        </CardTitle>
        <CardDescription>
          {t('Choose an amount and payment method')}
        </CardDescription>
        <CardAction>
          <Select
            items={enabledCurrencyItems}
            value={topup.selectedCurrency}
            onValueChange={(value) => {
              if (value) topup.changeCurrency(value)
            }}
            disabled={topup.isCurrencyChanging}
          >
            <SelectTrigger aria-label={t('Billing currency')}>
              {topup.isCurrencyChanging ? (
                <Spinner data-icon='inline-start' />
              ) : null}
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {enabledCurrencyItems.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </CardAction>
      </CardHeader>

      <CardContent className='flex flex-col gap-4'>
        {/* Balance lives only in WalletStatsCard — do not re-display with presentment FX. */}
        <p className='text-muted-foreground text-sm'>
          {t(
            'Your spendable API balance is shown at the top of this page. Amounts below are what you pay and receive for this top-up only.'
          )}
        </p>

        {topup.offers.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <CreditCard aria-hidden='true' />
              </EmptyMedia>
              <EmptyTitle>{t('No offers in this currency')}</EmptyTitle>
              <EmptyDescription>
                {t(
                  'Select another enabled billing currency or try again later.'
                )}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <FieldSet>
            <FieldLegend>{t('Choose your top-up')}</FieldLegend>
            <FieldDescription>
              {t('Top-ups are credited 1:1 with the amount paid.')}
            </FieldDescription>
            <RadioGroup
              value={
                customSelected
                  ? ''
                  : (topup.selectedAmountMinor?.toString() ?? '')
              }
              onValueChange={(value) => {
                setCustomSelected(false)
                topup.setSelectedAmountMinor(Number(value))
              }}
              className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'
            >
              {topup.offers.map((offer) => (
                <FieldLabel
                  key={offer.code}
                  htmlFor={`topup-${offer.payment_amount_minor}`}
                >
                  <Field data-disabled={!offer.available} className='min-h-36'>
                    <div className='flex items-start justify-between gap-2'>
                      <FieldTitle>{offer.payment_display}</FieldTitle>
                      <RadioGroupItem
                        id={`topup-${offer.payment_amount_minor}`}
                        value={offer.payment_amount_minor.toString()}
                        disabled={!offer.available}
                      />
                    </div>
                    <div className='mt-2 flex flex-col gap-1 font-mono tabular-nums'>
                      <span className='text-lg font-semibold'>
                        {t('Receive')} {offer.total_display}
                      </span>
                    </div>
                    {!offer.available ? (
                      <FieldDescription>
                        {unavailableMessage(offer, t)}
                      </FieldDescription>
                    ) : null}
                  </Field>
                </FieldLabel>
              ))}
            </RadioGroup>
          </FieldSet>
        )}

        <Field>
          <FieldLabel htmlFor='stripe-custom-topup'>
            {t('Custom Amount')}
          </FieldLabel>
          <div className='flex flex-col gap-2 sm:flex-row sm:items-center'>
            <Input
              id='stripe-custom-topup'
              type='number'
              inputMode='decimal'
              min={minimumDisplay}
              max={maximumDisplay}
              step='0.01'
              value={customAmount}
              onFocus={() => setCustomSelected(true)}
              onChange={(event) => {
                setCustomSelected(true)
                setCustomAmount(event.target.value)
              }}
              placeholder={`${minimumDisplay}-${maximumDisplay} ${topup.selectedCurrency.toUpperCase()}`}
              aria-invalid={customSelected && !customAmountValid}
            />
            <span className='text-muted-foreground text-xs'>
              {t('Minimum:')} {minimumDisplay}{' '}
              {topup.selectedCurrency.toUpperCase()} · {maximumDisplay}{' '}
              {topup.selectedCurrency.toUpperCase()}
            </span>
          </div>
        </Field>

        <p className='text-muted-foreground text-xs leading-relaxed'>
          {t('Top-ups do not include bonus credits.')}
        </p>
      </CardContent>

      <CardFooter className='justify-between gap-3'>
        <div className='text-sm'>{selectionSummary}</div>
        <Button
          disabled={
            topup.isCheckingOut ||
            (customSelected ? !customAmountValid : !selectedOffer?.available)
          }
          onClick={() => {
            if (checkoutAmountMinor != null) topup.checkout(checkoutAmountMinor)
          }}
        >
          {topup.isCheckingOut ? <Spinner data-icon='inline-start' /> : null}
          {topup.isCheckingOut
            ? t('Opening secure checkout')
            : t('Pay with Stripe')}
        </Button>
      </CardFooter>
    </Card>
  )
}
