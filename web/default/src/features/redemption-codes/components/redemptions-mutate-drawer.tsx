import { zodResolver } from '@hookform/resolvers/zod'
import { type FormEvent, useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DateTimePicker } from '@/components/datetime-picker'
import {
  SideDrawerSection,
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { addTimeToDate } from '@/lib/time'

import { createRedemption, updateRedemption, getRedemption } from '../api'
import {
  REDEMPTION_CURRENCIES,
  REDEMPTION_CURRENCY_LABELS,
  REDEMPTION_CURRENCY_SYMBOLS,
  SUCCESS_MESSAGES,
  type RedemptionCurrency,
} from '../constants'
import {
  getRedemptionFormSchema,
  type RedemptionFormValues,
  REDEMPTION_FORM_DEFAULT_VALUES,
  transformFormDataToPayload,
  transformRedemptionToFormDefaults,
} from '../lib'
import type { Redemption } from '../types'
import { useRedemptions } from './redemptions-provider'

type RedemptionsMutateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: Redemption
}

export function RedemptionsMutateDrawer({
  open,
  onOpenChange,
  currentRow,
}: RedemptionsMutateDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow
  const { triggerRefresh } = useRedemptions()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const form = useForm<RedemptionFormValues>({
    resolver: zodResolver(getRedemptionFormSchema(t)),
    defaultValues: REDEMPTION_FORM_DEFAULT_VALUES,
  })

  // Load existing data when updating
  useEffect(() => {
    if (open && isUpdate && currentRow) {
      // For update, fetch fresh data
      getRedemption(currentRow.id)
        .then((result) => {
          if (result.success && result.data) {
            form.reset(transformRedemptionToFormDefaults(result.data))
          }
        })
        .catch(() => {
          // Ignore refresh failures; the drawer keeps its current form state.
        })
    } else if (open && !isUpdate) {
      // For create, reset to defaults
      form.reset(REDEMPTION_FORM_DEFAULT_VALUES)
    }
  }, [open, isUpdate, currentRow, form])

  const onSubmit = async (data: RedemptionFormValues) => {
    setIsSubmitting(true)
    try {
      const basePayload = transformFormDataToPayload(data)

      if (isUpdate && currentRow) {
        const result = await updateRedemption({
          ...basePayload,
          id: currentRow.id,
        })
        if (result.success) {
          toast.success(t(SUCCESS_MESSAGES.REDEMPTION_UPDATED))
          onOpenChange(false)
          triggerRefresh()
        }
      } else {
        // Create mode
        const result = await createRedemption(basePayload)
        if (result.success) {
          const count = result.data?.length || 0
          toast.success(
            count > 1
              ? t('Successfully created {{count}} redemption codes', {
                  count,
                })
              : t(SUCCESS_MESSAGES.REDEMPTION_CREATED)
          )
          onOpenChange(false)
          triggerRefresh()
        }
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    if (!isUpdate) {
      const name = form.getValues('name')
      if (!name?.trim()) {
        // Auto-generate name from price + currency, e.g. "10 USD"
        const amount = form.getValues('amount')
        const currency = form.getValues('currency')
        const symbol = REDEMPTION_CURRENCY_SYMBOLS[currency]
        const label = REDEMPTION_CURRENCY_LABELS[currency]
        const autoName = `${symbol}${amount} ${label}`.slice(0, 20)
        form.setValue('name', autoName, { shouldValidate: true })
      }
    }

    void form.handleSubmit(onSubmit)(event)
  }

  const handleSetExpiry = (months: number, days: number, hours: number) => {
    const newDate = addTimeToDate(months, days, hours)
    form.setValue('expired_time', newDate)
  }

  const selectedCurrency: RedemptionCurrency = form.watch('currency')
  const currencySymbol = REDEMPTION_CURRENCY_SYMBOLS[selectedCurrency]
  const currencyLabel = REDEMPTION_CURRENCY_LABELS[selectedCurrency]
  const amountLabel = t('Amount ({{currency}})', { currency: currencyLabel })
  const count = form.watch('count') ?? 1
  const isCustomCodeDisabled = count > 1

  return (
    <Sheet
      open={open}
      onOpenChange={(v) => {
        onOpenChange(v)
        if (!v) {
          form.reset()
        }
      }}
    >
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[600px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {isUpdate
              ? t('Update Redemption Code')
              : t('Create Redemption Code')}
          </SheetTitle>
          <SheetDescription>
            {isUpdate
              ? t('Update the redemption code by providing necessary info.')
              : t(
                  'Add new redemption code(s) by providing necessary info.'
                )}{' '}
            {t('Click save when you&apos;re done.')}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form
            id='redemption-form'
            onSubmit={handleSubmit}
            className={sideDrawerFormClassName()}
          >
            <SideDrawerSection>
              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Name')}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t('Enter a name')} />
                    </FormControl>
                    <FormDescription>
                      {t('Name for this redemption code (1-20 characters)')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className='grid grid-cols-1 gap-4 sm:grid-cols-[140px_1fr]'>
                <FormField
                  control={form.control}
                  name='currency'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Currency')}</FormLabel>
                      <Select
                        items={REDEMPTION_CURRENCIES.map((c) => ({
                          value: c,
                          label: REDEMPTION_CURRENCY_LABELS[c],
                        }))}
                        onValueChange={(value) =>
                          value !== null &&
                          field.onChange(value as RedemptionCurrency)
                        }
                        value={field.value}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue
                              placeholder={t('Select currency')}
                            />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            {REDEMPTION_CURRENCIES.map((c) => (
                              <SelectItem key={c} value={c}>
                                {REDEMPTION_CURRENCY_LABELS[c]}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormDescription>
                        {t('Currency of the redemption amount')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='amount'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{amountLabel}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          step='0.01'
                          min='0.01'
                          placeholder={t('Enter amount in {{currency}}', {
                            currency: currencyLabel,
                          })}
                          onChange={(e) =>
                            field.onChange(Number.parseFloat(e.target.value) || 0)
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Free quota credited to the user, e.g. {{symbol}}10 = 10 {{currency}} worth of credit',
                          {
                            symbol: currencySymbol,
                            currency: currencyLabel,
                          }
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <FormField
                control={form.control}
                name='max_redeems'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Total Redemption Count')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type='number'
                        min='1'
                        max='100000'
                        placeholder={t('Total times this code can be redeemed')}
                        onChange={(e) =>
                          field.onChange(Number.parseInt(e.target.value, 10) || 1)
                        }
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'How many users can redeem this code in total. Each user can redeem it only once.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='expired_time'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Expiration Time')}</FormLabel>
                    <div className='flex flex-col gap-2'>
                      <FormControl>
                        <DateTimePicker
                          value={field.value}
                          onChange={field.onChange}
                          placeholder={t('Never expires')}
                        />
                      </FormControl>
                      <div className='grid grid-cols-4 gap-1.5 sm:flex sm:gap-2'>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          onClick={() => handleSetExpiry(0, 0, 0)}
                        >
                          {t('Never')}
                        </Button>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          onClick={() => handleSetExpiry(1, 0, 0)}
                        >
                          {t('1M')}
                        </Button>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          onClick={() => handleSetExpiry(0, 7, 0)}
                        >
                          {t('1W')}
                        </Button>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          onClick={() => handleSetExpiry(0, 1, 0)}
                        >
                          {t('1 Day')}
                        </Button>
                      </div>
                    </div>
                    <FormDescription>
                      {t('Leave empty for never expires')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {!isUpdate && (
                <FormField
                  control={form.control}
                  name='count'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Quantity')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          min='1'
                          max='100'
                          placeholder={t('Number of codes to create')}
                          onChange={(e) =>
                            field.onChange(Number.parseInt(e.target.value, 10) || 1)
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Create multiple redemption codes at once (1-100)')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {!isUpdate && (
                <FormField
                  control={form.control}
                  name='key'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Custom Code (optional)')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          value={field.value ?? ''}
                          placeholder={t('e.g. Launch-2026')}
                          disabled={isCustomCodeDisabled}
                        />
                      </FormControl>
                      <FormDescription>
                        {isCustomCodeDisabled
                          ? t(
                              'Custom code is only available when quantity is 1'
                            )
                          : t(
                              '3-64 characters: letters, digits, hyphens, underscores. Leave empty for auto-generated code.'
                            )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </SideDrawerSection>
          </form>
        </Form>
        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Close')}
          </SheetClose>
          <Button form='redemption-form' type='submit' disabled={isSubmitting}>
            {isSubmitting ? t('Saving...') : t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
