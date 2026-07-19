import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CircleDollarSign, RefreshCw, Sparkles } from 'lucide-react'
import { useCallback, useEffect, useState, type MutableRefObject } from 'react'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { BillingCurrency } from '@/lib/billing-currency'

import {
  getAdminBillingCurrencies,
  getAdminTopupCampaign,
  getAdminTopupPreview,
  updateAdminBillingCurrencies,
  updateAdminTopupCampaign,
} from './topup-promotion-api'
import {
  createTopupPromotionFormValues,
  localInputToUnix,
  topupPromotionFormSchema,
  type TopupPromotionFormValues,
} from './topup-promotion-form'

const CURRENCY_ITEMS: Array<{ label: string; value: BillingCurrency }> = [
  { label: 'CNY', value: 'cny' },
  { label: 'USD', value: 'usd' },
  { label: 'CAD', value: 'cad' },
]

const PROMOTION_BANDS = [
  { payment: '10.00-19.99', bonus: '2x', total: '3x' },
  { payment: '20.00-49.99', bonus: '3x', total: '4x' },
  { payment: '50.00-99.99', bonus: '4x', total: '5x' },
  { payment: '100.00-199.99', bonus: '5x', total: '6x' },
  { payment: '200.00-499.99', bonus: '6x', total: '7x' },
  { payment: '500.00', bonus: '7x', total: '8x' },
] as const

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
    enabled: true,
    startAt: '',
    endAt: '',
    globalBudgetUSD: 0,
    perUserLimit: 0,
    // 0 = promotional top-up lots never expire (preferred launch default).
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
  const [previewCurrency, setPreviewCurrency] = useState<BillingCurrency>('cny')
  const currencyQuery = useQuery({
    queryKey: ['admin-billing-currencies'],
    queryFn: getAdminBillingCurrencies,
  })
  const campaignQuery = useQuery({
    queryKey: ['admin-topup-campaign'],
    queryFn: getAdminTopupCampaign,
  })
  const previewQuery = useQuery({
    queryKey: ['admin-topup-preview', previewCurrency],
    queryFn: () => getAdminTopupPreview(previewCurrency),
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
    setPreviewCurrency(currencyQuery.data.default_currency)
  }, [campaignQuery.data, currencyQuery.data, form])

  const saveMutation = useMutation({
    mutationFn: async (values: TopupPromotionFormValues) => {
      if (!campaignQuery.data) {
        throw new Error(t('Campaign data is unavailable'))
      }
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
      await updateAdminTopupCampaign({
        ...campaignQuery.data,
        name: values.campaign.name,
        enabled: values.campaign.enabled,
        start_at: localInputToUnix(values.campaign.startAt),
        end_at: localInputToUnix(values.campaign.endAt),
        global_budget_micro_usd: Math.round(
          values.campaign.globalBudgetUSD * 1_000_000
        ),
        per_user_limit: values.campaign.perUserLimit,
        default_promo_expiry_days: values.campaign.defaultPromoExpiryDays,
      })
    },
    onSuccess: async (_, values) => {
      form.reset(values)
      toast.success(t('Launch billing settings saved'))
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['admin-billing-currencies'],
        }),
        queryClient.invalidateQueries({ queryKey: ['admin-topup-campaign'] }),
        queryClient.invalidateQueries({ queryKey: ['admin-topup-preview'] }),
        queryClient.invalidateQueries({ queryKey: ['billing-topup-config'] }),
        queryClient.invalidateQueries({ queryKey: ['pricing'] }),
      ])
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Unable to save launch billing settings'))
    },
  })

  const save = useCallback(async () => {
    const isValid = await form.trigger()
    if (!isValid) {
      toast.error(t('Fix the highlighted launch billing fields'))
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

  const isLoading = currencyQuery.isPending || campaignQuery.isPending

  if (isLoading) {
    return <Skeleton className='h-72 w-full rounded-xl' />
  }

  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle className='flex items-center gap-2'>
          <CircleDollarSign aria-hidden='true' />
          {t('Launch billing control room')}
        </CardTitle>
        <CardDescription>
          {t(
            'Manage shared currencies, active and Bank of Canada reference rates, one-unit minimums, promotion limits, and budgets.'
          )}
        </CardDescription>
        <CardAction className='flex gap-2'>
          <Badge
            variant={form.watch('campaign.enabled') ? 'secondary' : 'outline'}
          >
            {form.watch('campaign.enabled')
              ? t('Campaign on')
              : t('Campaign off')}
          </Badge>
          <Badge variant='outline'>
            {form.watch('defaultCurrency').toUpperCase()} {t('default')}
          </Badge>
        </CardAction>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue='currencies'>
          <div className='overflow-x-auto pb-1'>
            <TabsList className='grid min-w-[34rem] grid-cols-4'>
              <TabsTrigger value='currencies'>{t('Currencies')}</TabsTrigger>
              <TabsTrigger value='campaign'>{t('Campaign')}</TabsTrigger>
              <TabsTrigger value='tiers'>{t('Promotion bands')}</TabsTrigger>
              <TabsTrigger value='preview'>{t('Preview')}</TabsTrigger>
            </TabsList>
          </div>

          <TabsContent value='currencies' className='mt-5'>
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
                        disabled={
                          item.value === 'usd' || form.watch('autoUpdateFX')
                        }
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
          </TabsContent>

          <TabsContent value='campaign' className='mt-5'>
            <FieldGroup>
              <Controller
                control={form.control}
                name='campaign.enabled'
                render={({ field }) => (
                  <Field orientation='horizontal'>
                    <div>
                      <FieldTitle>{t('Enable promotion')}</FieldTitle>
                      <FieldDescription>
                        {t('Disabling stops future promotional checkouts.')}
                      </FieldDescription>
                    </div>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </Field>
                )}
              />
              <Field>
                <FieldLabel htmlFor='campaign-name'>
                  {t('Campaign name')}
                </FieldLabel>
                <Input id='campaign-name' {...form.register('campaign.name')} />
              </Field>
              <div className='grid gap-4 md:grid-cols-2'>
                <Field>
                  <FieldLabel htmlFor='campaign-start'>
                    {t('Starts at')}
                  </FieldLabel>
                  <Input
                    id='campaign-start'
                    type='datetime-local'
                    {...form.register('campaign.startAt')}
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor='campaign-end'>{t('Ends at')}</FieldLabel>
                  <Input
                    id='campaign-end'
                    type='datetime-local'
                    {...form.register('campaign.endAt')}
                  />
                </Field>
              </div>
              <div className='grid gap-4 md:grid-cols-3'>
                <Field>
                  <FieldLabel htmlFor='campaign-budget'>
                    {t('Global promo budget (USD credits)')}
                  </FieldLabel>
                  <Input
                    id='campaign-budget'
                    type='number'
                    min='0'
                    step='0.01'
                    {...form.register('campaign.globalBudgetUSD', {
                      valueAsNumber: true,
                    })}
                  />
                  <FieldDescription>{t('0 means unlimited.')}</FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor='campaign-limit'>
                    {t('Per-user redemption limit')}
                  </FieldLabel>
                  <Input
                    id='campaign-limit'
                    type='number'
                    min='0'
                    step='1'
                    {...form.register('campaign.perUserLimit', {
                      valueAsNumber: true,
                    })}
                  />
                  <FieldDescription>{t('0 means unlimited.')}</FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor='campaign-expiry'>
                    {t('Promo expiry days')}
                  </FieldLabel>
                  <Input
                    id='campaign-expiry'
                    type='number'
                    min='0'
                    step='1'
                    {...form.register('campaign.defaultPromoExpiryDays', {
                      valueAsNumber: true,
                    })}
                  />
                  <FieldDescription>
                    {t(
                      'Applies only to promotional bonus lots from future top-ups. 0 means never expire. Paid cash credits never expire. Already-issued lots keep their original expiry.'
                    )}
                  </FieldDescription>
                </Field>
              </div>
              <div className='flex flex-wrap gap-2'>
                <Badge variant='outline'>
                  {t('Reserved')}:{' '}
                  {(
                    (campaignQuery.data?.reserved_promo_micro_usd ?? 0) /
                    1_000_000
                  ).toLocaleString(undefined, {
                    style: 'currency',
                    currency: 'USD',
                  })}
                </Badge>
                <Badge variant='outline'>
                  {t('Issued')}:{' '}
                  {(
                    (campaignQuery.data?.issued_promo_micro_usd ?? 0) /
                    1_000_000
                  ).toLocaleString(undefined, {
                    style: 'currency',
                    currency: 'USD',
                  })}
                </Badge>
              </div>
            </FieldGroup>
          </TabsContent>

          <TabsContent value='tiers' className='mt-5'>
            <div className='space-y-4'>
              <div>
                <h4 className='font-medium'>{t('Shared promotion bands')}</h4>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Configured once for CNY, USD, and CAD. The same number is paid and credited in the selected currency; no visible cross-currency conversion is used.'
                  )}
                </p>
              </div>
              <div className='overflow-hidden rounded-xl border'>
                <div className='bg-muted/50 grid grid-cols-3 gap-3 px-4 py-3 text-xs font-medium uppercase'>
                  <span>{t('Payment')}</span>
                  <span>{t('Bonus')}</span>
                  <span>{t('Total credits')}</span>
                </div>
                {PROMOTION_BANDS.map((band) => (
                  <div
                    key={band.payment}
                    className='grid grid-cols-3 gap-3 border-t px-4 py-3 font-mono text-sm tabular-nums'
                  >
                    <span>{band.payment}</span>
                    <span>{band.bonus}</span>
                    <span>{band.total}</span>
                  </div>
                ))}
              </div>
              <FieldDescription>
                {t(
                  'All calculations use integer minor units. A 19.99 payment is 1999 minor units in one Checkout line item with quantity 1.'
                )}
              </FieldDescription>
            </div>
          </TabsContent>

          <TabsContent value='preview' className='mt-5'>
            <div className='flex flex-col gap-4'>
              <div className='flex flex-wrap items-center justify-between gap-3'>
                <Select
                  items={CURRENCY_ITEMS}
                  value={previewCurrency}
                  onValueChange={(value) => value && setPreviewCurrency(value)}
                >
                  <SelectTrigger aria-label={t('Preview currency')}>
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
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => previewQuery.refetch()}
                  disabled={previewQuery.isFetching}
                >
                  {previewQuery.isFetching ? (
                    <Spinner data-icon='inline-start' />
                  ) : (
                    <RefreshCw data-icon='inline-start' />
                  )}
                  {t('Refresh preview')}
                </Button>
              </div>
              {previewQuery.isPending ? (
                <Skeleton className='h-36 w-full rounded-xl' />
              ) : (
                <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
                  {previewQuery.data?.offers.map((offer) => (
                    <Card key={offer.tier_id} size='sm'>
                      <CardHeader>
                        <CardTitle>{offer.payment_display}</CardTitle>
                        <CardDescription>
                          +{offer.bonus_display} {t('bonus')}
                        </CardDescription>
                        <CardAction>
                          <Badge
                            variant={offer.available ? 'secondary' : 'outline'}
                          >
                            {offer.available
                              ? t('Available')
                              : t('Unavailable')}
                          </Badge>
                        </CardAction>
                      </CardHeader>
                      <CardContent>
                        <p className='font-mono text-lg font-semibold'>
                          {offer.total_display} {t('total')}
                        </p>
                        {offer.unavailable_reason ? (
                          <p className='text-muted-foreground mt-1 text-xs'>
                            {offer.unavailable_reason}
                          </p>
                        ) : null}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </div>
          </TabsContent>
        </Tabs>
      </CardContent>
      <CardFooter className='justify-between gap-3'>
        <p className='text-muted-foreground text-xs'>
          <Sparkles aria-hidden='true' className='mr-1 inline' />
          {t(
            'Changes apply only to future orders; completed order snapshots remain immutable.'
          )}
        </p>
        <Button type='button' disabled={saveMutation.isPending} onClick={save}>
          {saveMutation.isPending ? <Spinner data-icon='inline-start' /> : null}
          {t('Save launch billing')}
        </Button>
      </CardFooter>
    </Card>
  )
}
