import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CircleDollarSign } from 'lucide-react'
import { useCallback, useEffect, type MutableRefObject } from 'react'
import { Controller, useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
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
import { Switch } from '@/components/ui/switch'
import type { BillingCurrency } from '@/lib/billing-currency'

import {
  getAdminBillingCurrencies,
  getAdminTopupCampaign,
  updateAdminBillingCurrencies,
} from './topup-promotion-api'
import {
  createTopupPromotionFormValues,
  topupPromotionFormSchema,
  type TopupPromotionFormValues,
} from './topup-promotion-form'

const CURRENCY_ITEMS: Array<{ label: string; value: BillingCurrency }> = [
  { label: 'CNY', value: 'cny' },
  { label: 'USD', value: 'usd' },
  { label: 'CAD', value: 'cad' },
]

const EMPTY_VALUES: TopupPromotionFormValues = {
  defaultCurrency: 'cny',
  autoUpdateFX: true,
  currencies: {
    cny: { enabled: true, fx: 7.3 },
    usd: { enabled: true, fx: 1 },
    cad: { enabled: true, fx: 1.37 },
  },
  campaign: {
    name: '',
    enabled: false,
    startAt: '',
    endAt: '',
    globalBudgetUSD: 0,
    perUserLimit: 0,
    defaultPromoExpiryDays: 0,
  },
}

export type TopupPromotionSettingsHandle = {
  isDirty: () => boolean
  save: () => Promise<boolean>
}

export function TopupPromotionSettings({
  saveHandleRef,
}: {
  saveHandleRef?: MutableRefObject<TopupPromotionSettingsHandle | null>
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const currencyQuery = useQuery({
    queryKey: ['admin-billing-currencies'],
    queryFn: getAdminBillingCurrencies,
  })
  const campaignQuery = useQuery({
    queryKey: ['admin-topup-campaign'],
    queryFn: getAdminTopupCampaign,
  })
  const form = useForm<TopupPromotionFormValues>({
    resolver: zodResolver(
      topupPromotionFormSchema
    ) as Resolver<TopupPromotionFormValues>,
    defaultValues: EMPTY_VALUES,
  })

  useEffect(() => {
    if (!currencyQuery.data || !campaignQuery.data) {
      return
    }
    form.reset(
      createTopupPromotionFormValues(currencyQuery.data, campaignQuery.data)
    )
  }, [campaignQuery.data, currencyQuery.data, form])

  const saveMutation = useMutation({
    mutationFn: async (values: TopupPromotionFormValues) => {
      await updateAdminBillingCurrencies({
        default_currency: values.defaultCurrency,
        auto_update_fx: values.autoUpdateFX,
        fx_source: currencyQuery.data?.fx_source,
        fx_updated_at: currencyQuery.data?.fx_updated_at,
        reference_fx_presentment_per_usd:
          currencyQuery.data?.reference_fx_presentment_per_usd,
        currencies: {
          cny: {
            enabled: values.currencies.cny.enabled,
            fx_presentment_per_usd: values.currencies.cny.fx,
          },
          usd: {
            enabled: values.currencies.usd.enabled,
            fx_presentment_per_usd: values.currencies.usd.fx,
          },
          cad: {
            enabled: values.currencies.cad.enabled,
            fx_presentment_per_usd: values.currencies.cad.fx,
          },
        },
      })
    },
    onSuccess: async (_, values) => {
      form.reset(values)
      toast.success(t('Top-up currency settings saved'))
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['admin-billing-currencies'],
        }),
        queryClient.invalidateQueries({ queryKey: ['billing-topup-config'] }),
        queryClient.invalidateQueries({ queryKey: ['pricing'] }),
      ])
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Unable to save top-up currency settings'))
    },
  })

  const save = useCallback(async () => {
    const isValid = await form.trigger()
    if (!isValid) {
      toast.error(t('Fix the highlighted currency fields'))
      return false
    }
    try {
      await saveMutation.mutateAsync(form.getValues())
      return true
    } catch {
      return false
    }
  }, [form, saveMutation, t])

  useEffect(() => {
    if (!saveHandleRef) {
      return
    }
    saveHandleRef.current = {
      isDirty: () => form.formState.isDirty,
      save: async () => {
        if (!form.formState.isDirty) {
          return false
        }
        return save()
      },
    }
    return () => {
      saveHandleRef.current = null
    }
  }, [form.formState.isDirty, save, saveHandleRef])

  if (currencyQuery.isPending || campaignQuery.isPending) {
    return <Skeleton className='h-72 w-full rounded-xl' />
  }

  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle className='flex items-center gap-2'>
          <CircleDollarSign aria-hidden='true' />
          {t('Top-up currency settings')}
        </CardTitle>
        <CardDescription>
          {t(
            'Manage billing currencies and exchange rates. Top-ups are credited 1:1 and do not include bonus credits.'
          )}
        </CardDescription>
        <CardAction>
          <Badge variant='outline'>
            {form.watch('defaultCurrency').toUpperCase()} {t('default')}
          </Badge>
        </CardAction>
      </CardHeader>
      <CardContent className='pt-5'>
        <FieldGroup>
          <Field>
            <FieldLabel>{t('Default billing currency')}</FieldLabel>
            <Controller
              control={form.control}
              name='defaultCurrency'
              render={({ field }) => (
                <Select
                  items={CURRENCY_ITEMS}
                  value={field.value}
                  onValueChange={(value) => value && field.onChange(value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {CURRENCY_ITEMS.map((item) => (
                        <SelectItem key={item.value} value={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              )}
            />
            <FieldError errors={[form.formState.errors.defaultCurrency]} />
            <FieldDescription>
              {t(
                'CNY is the launch default. Changing it affects future displays only.'
              )}
            </FieldDescription>
          </Field>
          <Controller
            control={form.control}
            name='autoUpdateFX'
            render={({ field }) => (
              <Field orientation='horizontal'>
                <div>
                  <FieldTitle>{t('Automatic exchange rates')}</FieldTitle>
                  <FieldDescription>
                    {t(
                      'Refreshes from Bank of Canada daily exchange rates and keeps the last successful values if the provider is unavailable.'
                    )}
                  </FieldDescription>
                </div>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
              </Field>
            )}
          />
          <div className='grid gap-3 md:grid-cols-3'>
            {CURRENCY_ITEMS.map((item) => (
              <FieldSet key={item.value} className='rounded-xl border p-4'>
                <FieldLegend>{item.label}</FieldLegend>
                <Controller
                  control={form.control}
                  name={`currencies.${item.value}.enabled`}
                  render={({ field }) => (
                    <Field orientation='horizontal'>
                      <FieldTitle>{t('Enabled')}</FieldTitle>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </Field>
                  )}
                />
                <Field>
                  <FieldLabel htmlFor={`${item.value}-fx`}>
                    {form.watch('autoUpdateFX')
                      ? t('Active automatic rate')
                      : t('Custom rate')}
                  </FieldLabel>
                  <Input
                    id={`${item.value}-fx`}
                    type='number'
                    min='0.000001'
                    step='0.000001'
                    disabled={item.value === 'usd' || form.watch('autoUpdateFX')}
                    {...form.register(`currencies.${item.value}.fx`, {
                      valueAsNumber: true,
                    })}
                  />
                  <FieldDescription>
                    {t('Bank of Canada reference')}: 1 USD ={' '}
                    {currencyQuery.data?.reference_fx_presentment_per_usd?.[
                      item.value
                    ] ?? form.watch(`currencies.${item.value}.fx`)}{' '}
                    {item.label}
                  </FieldDescription>
                  <FieldDescription>
                    {t('Top-up range')}: 1.00-500.00 {item.label}
                  </FieldDescription>
                </Field>
              </FieldSet>
            ))}
          </div>
          <FieldDescription>
            {t(
              'Model prices come from canonical model pricing and are converted with these rates.'
            )}
          </FieldDescription>
          {currencyQuery.data?.fx_updated_at ? (
            <FieldDescription>
              {t('Exchange rates last updated')}:{' '}
              {new Date(
                currencyQuery.data.fx_updated_at * 1000
              ).toLocaleString()}{' '}
              ({currencyQuery.data.fx_source || 'Bank of Canada'})
            </FieldDescription>
          ) : null}
        </FieldGroup>
      </CardContent>
      <CardFooter className='justify-end'>
        <Button type='button' disabled={saveMutation.isPending} onClick={save}>
          {saveMutation.isPending ? <Spinner data-icon='inline-start' /> : null}
          {t('Save currency settings')}
        </Button>
      </CardFooter>
    </Card>
  )
}
