import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import {
  CheckCircle2,
  CircleAlert,
  Code2,
  Eye,
  ShieldAlert,
} from 'lucide-react'
import * as React from 'react'
import { useForm, useWatch, type Control, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'
import { AmountDiscountVisualEditor } from './amount-discount-visual-editor'
import { AmountOptionsVisualEditor } from './amount-options-visual-editor'
import { CreemProductsVisualEditor } from './creem-products-visual-editor'
import { PaymentMethodsVisualEditor } from './payment-methods-visual-editor'
import { getStripeEnvironmentReadiness } from './stripe-environment-status'
import {
  TopupPromotionSettings,
  type TopupPromotionSettingsHandle,
} from './topup-promotion-settings'
import {
  formatJsonForEditor,
  getJsonError,
  normalizeJsonForComparison,
  removeTrailingSlash,
} from './utils'
import { saveWaffoPancakeConfig } from './waffo-pancake-api'
import {
  WaffoPancakeSettingsSection,
  type WaffoPancakeBinding,
  type WaffoPancakeSettingsValues,
} from './waffo-pancake-settings-section'
import {
  type PayMethod,
  WaffoSettingsSection,
  type WaffoSettingsValues,
} from './waffo-settings-section'

function isHttpOriginUrl(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return true

  try {
    const url = new URL(trimmed)
    const isHttpProtocol = url.protocol === 'http:' || url.protocol === 'https:'
    const hasNoPath = url.pathname === '' || url.pathname === '/'
    return isHttpProtocol && hasNoPath && !url.search && !url.hash
  } catch {
    return false
  }
}

const paymentSchema = z.object({
  PayAddress: z.string().refine((value) => {
    const trimmed = value.trim()
    if (!trimmed) return true
    return /^https?:\/\//.test(trimmed)
  }, 'Provide a valid callback URL starting with http:// or https://'),
  EpayId: z.string(),
  EpayKey: z.string(),
  Price: z.coerce.number().min(0),
  MinTopUp: z.coerce.number().min(0),
  CustomCallbackAddress: z
    .string()
    .refine(
      isHttpOriginUrl,
      'Enter only a top-level callback domain, for example https://api.example.com, without any path.'
    ),
  PayMethods: z.string().superRefine((value, ctx) => {
    const error = getJsonError(value)
    if (error) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: error,
      })
    }
  }),
  AmountOptions: z.string().superRefine((value, ctx) => {
    const error = getJsonError(value, (parsed) => Array.isArray(parsed))
    if (error) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: error,
      })
    }
  }),
  AmountDiscount: z.string().superRefine((value, ctx) => {
    const error = getJsonError(
      value,
      (parsed) =>
        !!parsed && typeof parsed === 'object' && !Array.isArray(parsed)
    )
    if (error) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: error,
      })
    }
  }),
  StripePriceId: z.string(),
  StripeTestTopupProductID: z.string(),
  StripeProdTopupProductID: z.string(),
  StripeTestAccountID: z.string(),
  StripeProdAccountID: z.string(),
  StripeTopupEnabled: z.boolean(),
  StripeUnitPrice: z.coerce.number().min(0),
  StripeMinTopUp: z.coerce.number().min(0),
  StripePromotionCodesEnabled: z.boolean(),
  CreemApiKey: z.string(),
  CreemWebhookSecret: z.string(),
  CreemTestMode: z.boolean(),
  CreemProducts: z.string().superRefine((value, ctx) => {
    const error = getJsonError(value, (parsed) => Array.isArray(parsed))
    if (error) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: error,
      })
    }
  }),
  WaffoEnabled: z.boolean(),
  WaffoApiKey: z.string(),
  WaffoPrivateKey: z.string(),
  WaffoPublicCert: z.string(),
  WaffoSandboxPublicCert: z.string(),
  WaffoSandboxApiKey: z.string(),
  WaffoSandboxPrivateKey: z.string(),
  WaffoSandbox: z.boolean(),
  WaffoMerchantId: z.string(),
  WaffoCurrency: z.string(),
  WaffoUnitPrice: z.coerce.number().min(0),
  WaffoMinTopUp: z.coerce.number().min(1),
  WaffoNotifyUrl: z.string(),
  WaffoReturnUrl: z.string(),
  WaffoPancakeMerchantID: z.string(),
  WaffoPancakePrivateKey: z.string(),
  WaffoPancakeReturnURL: z.string(),
})

type PaymentFormValues = z.infer<typeof paymentSchema>
type WaffoFormFieldValues = Omit<WaffoSettingsValues, 'WaffoPayMethods'>
type PaymentBaseFormValues = Omit<
  PaymentFormValues,
  keyof WaffoFormFieldValues | keyof WaffoPancakeSettingsValues
>

const paymentTabContentClassName = 'mt-6 min-w-0'

type PaymentSettingsSectionProps = {
  defaultValues: PaymentBaseFormValues
  stripeCredentialStatus: {
    runtimeEnvironment: 'test' | 'production' | 'disabled'
    test: StripeCredentialStatus
    production: StripeCredentialStatus
  }
  waffoDefaultValues: WaffoSettingsValues
  waffoPancakeDefaultValues: WaffoPancakeSettingsValues
  waffoPancakeProvisionedStoreID?: string
  waffoPancakeProvisionedProductID?: string
}

type StripeCredentialStatus = {
  secretConfigured: boolean
  publishableConfigured: boolean
  webhookConfigured: boolean
}

type StripeEnvironmentCardProps = {
  control: Control<PaymentFormValues>
  environment: 'test' | 'production'
  isActive: boolean
  status: StripeCredentialStatus
  productField: 'StripeTestTopupProductID' | 'StripeProdTopupProductID'
  accountField: 'StripeTestAccountID' | 'StripeProdAccountID'
}

function StripeEnvironmentCard({
  control,
  environment,
  isActive,
  status,
  productField,
  accountField,
}: StripeEnvironmentCardProps) {
  const { t } = useTranslation()
  const productID = useWatch({ control, name: productField })
  const accountID = useWatch({ control, name: accountField })
  const readiness = getStripeEnvironmentReadiness({
    ...status,
    productID,
    accountID,
  })
  const label = environment === 'test' ? t('Test') : t('Production')
  const credentialRows = [
    [t('Secret key'), status.secretConfigured],
    [t('Publishable key'), status.publishableConfigured],
    [t('Webhook signing secret'), status.webhookConfigured],
  ] as const

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {t('{{environment}} credentials', { environment: label })}
        </CardTitle>
        <CardDescription>
          {environment === 'test'
            ? t('Used automatically on localhost and non-release deployments.')
            : t('Used automatically only on the trusted production domain.')}
        </CardDescription>
        <CardAction className='flex gap-2'>
          {isActive && <Badge>{t('Active')}</Badge>}
          <Badge variant={readiness.ready ? 'secondary' : 'destructive'}>
            {readiness.ready ? t('Ready') : t('Incomplete')}
          </Badge>
        </CardAction>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div
          className='space-y-2'
          aria-label={t('{{environment}} credential status', {
            environment: label,
          })}
        >
          {credentialRows.map(([credentialLabel, configured]) => (
            <div
              key={credentialLabel}
              className='bg-muted/50 flex items-center justify-between rounded-md px-3 py-2'
            >
              <span className='text-sm font-medium'>{credentialLabel}</span>
              <span className='flex items-center gap-2'>
                <span
                  className='font-mono text-xs tracking-wider select-none'
                  aria-hidden='true'
                  onCopy={(event) => event.preventDefault()}
                >
                  {configured ? '••••••••••••' : '—'}
                </span>
                {configured ? (
                  <CheckCircle2
                    className='size-4 text-emerald-600'
                    aria-label={t('Configured')}
                  />
                ) : (
                  <CircleAlert
                    className='text-destructive size-4'
                    aria-label={t('Missing')}
                  />
                )}
              </span>
            </div>
          ))}
        </div>

        <div className='grid gap-4 sm:grid-cols-2'>
          <FormField
            control={control}
            name={accountField}
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Account ID')}</FormLabel>
                <FormControl>
                  <Input placeholder='acct_xxx' {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={control}
            name={productField}
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Wallet top-up product ID')}</FormLabel>
                <FormControl>
                  <Input placeholder='prod_xxx' {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function parseWaffoPayMethods(value: string): PayMethod[] {
  try {
    const parsed = JSON.parse(value || '[]')
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

export function PaymentSettingsSection({
  defaultValues,
  stripeCredentialStatus,
  waffoDefaultValues,
  waffoPancakeDefaultValues,
  waffoPancakeProvisionedStoreID,
  waffoPancakeProvisionedProductID,
}: PaymentSettingsSectionProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const updateOption = useUpdateOption()
  const initialFormValues = React.useMemo<PaymentFormValues>(
    () => ({
      ...defaultValues,
      ...waffoDefaultValues,
      ...waffoPancakeDefaultValues,
    }),
    [defaultValues, waffoDefaultValues, waffoPancakeDefaultValues]
  )
  const initialRef = React.useRef(initialFormValues)
  const topupPromotionSettingsRef =
    React.useRef<TopupPromotionSettingsHandle>(null)
  const defaultsSignature = React.useMemo(
    () => JSON.stringify(initialFormValues),
    [initialFormValues]
  )

  const [payMethodsVisualMode, setPayMethodsVisualMode] = React.useState(true)
  const [amountOptionsVisualMode, setAmountOptionsVisualMode] =
    React.useState(true)
  const [amountDiscountVisualMode, setAmountDiscountVisualMode] =
    React.useState(true)
  const [creemProductsVisualMode, setCreemProductsVisualMode] =
    React.useState(true)
  const [waffoPayMethods, setWaffoPayMethods] = React.useState<PayMethod[]>(
    () => parseWaffoPayMethods(waffoDefaultValues.WaffoPayMethods)
  )
  const [waffoPancakeSelection, setWaffoPancakeSelection] =
    React.useState<WaffoPancakeBinding>({
      storeID: waffoPancakeProvisionedStoreID ?? '',
      productID: waffoPancakeProvisionedProductID ?? '',
    })
  const [waffoPancakeSavedBinding, setWaffoPancakeSavedBinding] =
    React.useState<WaffoPancakeBinding>({
      storeID: waffoPancakeProvisionedStoreID ?? '',
      productID: waffoPancakeProvisionedProductID ?? '',
    })

  React.useEffect(() => {
    setWaffoPayMethods(parseWaffoPayMethods(waffoDefaultValues.WaffoPayMethods))
  }, [waffoDefaultValues.WaffoPayMethods])

  React.useEffect(() => {
    const nextBinding = {
      storeID: waffoPancakeProvisionedStoreID ?? '',
      productID: waffoPancakeProvisionedProductID ?? '',
    }
    setWaffoPancakeSelection(nextBinding)
    setWaffoPancakeSavedBinding(nextBinding)
  }, [waffoPancakeProvisionedProductID, waffoPancakeProvisionedStoreID])

  const form = useForm<PaymentFormValues>({
    resolver: zodResolver(paymentSchema) as Resolver<PaymentFormValues>,
    mode: 'onChange', // Enable real-time validation
    defaultValues: {
      ...initialFormValues,
      PayMethods: formatJsonForEditor(initialFormValues.PayMethods),
      AmountOptions: formatJsonForEditor(initialFormValues.AmountOptions),
      AmountDiscount: formatJsonForEditor(initialFormValues.AmountDiscount),
      CreemProducts: formatJsonForEditor(initialFormValues.CreemProducts),
    },
  })

  const { isSubmitting } = form.formState

  const setPaymentValue = React.useCallback(
    (
      key: keyof PaymentFormValues,
      value: PaymentFormValues[keyof PaymentFormValues]
    ) => {
      form.setValue(
        key as Parameters<typeof form.setValue>[0],
        value as Parameters<typeof form.setValue>[1],
        {
          shouldDirty: true,
          shouldValidate: true,
        }
      )
    },
    [form]
  )

  const setWaffoValue = React.useCallback(
    <K extends keyof WaffoFormFieldValues>(
      key: K,
      value: WaffoFormFieldValues[K]
    ) => {
      setPaymentValue(
        key as keyof PaymentFormValues,
        value as PaymentFormValues[keyof PaymentFormValues]
      )
    },
    [setPaymentValue]
  )

  const setWaffoPancakeValue = React.useCallback(
    <K extends keyof WaffoPancakeSettingsValues>(
      key: K,
      value: WaffoPancakeSettingsValues[K]
    ) => {
      setPaymentValue(
        key as keyof PaymentFormValues,
        value as PaymentFormValues[keyof PaymentFormValues]
      )
    },
    [setPaymentValue]
  )

  React.useEffect(() => {
    const parsedDefaults = JSON.parse(defaultsSignature) as PaymentFormValues
    initialRef.current = parsedDefaults
    form.reset({
      ...parsedDefaults,
      PayMethods: formatJsonForEditor(parsedDefaults.PayMethods),
      AmountOptions: formatJsonForEditor(parsedDefaults.AmountOptions),
      AmountDiscount: formatJsonForEditor(parsedDefaults.AmountDiscount),
      CreemProducts: formatJsonForEditor(parsedDefaults.CreemProducts),
    })
  }, [defaultsSignature, form])

  const onSubmit = async (values: PaymentFormValues) => {
    const sanitized = {
      PayAddress: removeTrailingSlash(values.PayAddress),
      EpayId: values.EpayId.trim(),
      EpayKey: values.EpayKey.trim(),
      Price: values.Price,
      MinTopUp: values.MinTopUp,
      CustomCallbackAddress: removeTrailingSlash(values.CustomCallbackAddress),
      PayMethods: values.PayMethods.trim(),
      AmountOptions: values.AmountOptions.trim(),
      AmountDiscount: values.AmountDiscount.trim(),
      StripePriceId: values.StripePriceId.trim(),
      StripeTestTopupProductID: values.StripeTestTopupProductID.trim(),
      StripeProdTopupProductID: values.StripeProdTopupProductID.trim(),
      StripeTestAccountID: values.StripeTestAccountID.trim(),
      StripeProdAccountID: values.StripeProdAccountID.trim(),
      StripeTopupEnabled: values.StripeTopupEnabled,
      StripeUnitPrice: values.StripeUnitPrice,
      StripeMinTopUp: values.StripeMinTopUp,
      StripePromotionCodesEnabled: values.StripePromotionCodesEnabled,
      CreemApiKey: values.CreemApiKey.trim(),
      CreemWebhookSecret: values.CreemWebhookSecret.trim(),
      CreemTestMode: values.CreemTestMode,
      CreemProducts: values.CreemProducts.trim(),
      WaffoEnabled: values.WaffoEnabled,
      WaffoSandbox: values.WaffoSandbox,
      WaffoMerchantId: values.WaffoMerchantId.trim(),
      WaffoCurrency: values.WaffoCurrency.trim() || 'USD',
      WaffoUnitPrice: values.WaffoUnitPrice,
      WaffoMinTopUp: values.WaffoMinTopUp,
      WaffoNotifyUrl: values.WaffoNotifyUrl.trim(),
      WaffoReturnUrl: values.WaffoReturnUrl.trim(),
      WaffoPublicCert: values.WaffoPublicCert.trim(),
      WaffoSandboxPublicCert: values.WaffoSandboxPublicCert.trim(),
      WaffoApiKey: values.WaffoApiKey.trim(),
      WaffoPrivateKey: values.WaffoPrivateKey.trim(),
      WaffoSandboxApiKey: values.WaffoSandboxApiKey.trim(),
      WaffoSandboxPrivateKey: values.WaffoSandboxPrivateKey.trim(),
      WaffoPayMethods: JSON.stringify(waffoPayMethods),
      WaffoPancakeMerchantID: values.WaffoPancakeMerchantID.trim(),
      WaffoPancakePrivateKey: values.WaffoPancakePrivateKey.trim(),
      WaffoPancakeReturnURL: removeTrailingSlash(
        values.WaffoPancakeReturnURL.trim()
      ),
    }

    const initial = {
      PayAddress: removeTrailingSlash(initialRef.current.PayAddress),
      EpayId: initialRef.current.EpayId.trim(),
      EpayKey: initialRef.current.EpayKey.trim(),
      Price: initialRef.current.Price,
      MinTopUp: initialRef.current.MinTopUp,
      CustomCallbackAddress: removeTrailingSlash(
        initialRef.current.CustomCallbackAddress
      ),
      PayMethods: initialRef.current.PayMethods.trim(),
      AmountOptions: initialRef.current.AmountOptions.trim(),
      AmountDiscount: initialRef.current.AmountDiscount.trim(),
      StripePriceId: initialRef.current.StripePriceId.trim(),
      StripeTestTopupProductID:
        initialRef.current.StripeTestTopupProductID.trim(),
      StripeProdTopupProductID:
        initialRef.current.StripeProdTopupProductID.trim(),
      StripeTestAccountID: initialRef.current.StripeTestAccountID.trim(),
      StripeProdAccountID: initialRef.current.StripeProdAccountID.trim(),
      StripeTopupEnabled: initialRef.current.StripeTopupEnabled,
      StripeUnitPrice: initialRef.current.StripeUnitPrice,
      StripeMinTopUp: initialRef.current.StripeMinTopUp,
      StripePromotionCodesEnabled:
        initialRef.current.StripePromotionCodesEnabled,
      CreemApiKey: initialRef.current.CreemApiKey.trim(),
      CreemWebhookSecret: initialRef.current.CreemWebhookSecret.trim(),
      CreemTestMode: initialRef.current.CreemTestMode,
      CreemProducts: initialRef.current.CreemProducts.trim(),
      WaffoEnabled: initialRef.current.WaffoEnabled,
      WaffoSandbox: initialRef.current.WaffoSandbox,
      WaffoMerchantId: initialRef.current.WaffoMerchantId.trim(),
      WaffoCurrency: initialRef.current.WaffoCurrency.trim() || 'USD',
      WaffoUnitPrice: initialRef.current.WaffoUnitPrice,
      WaffoMinTopUp: initialRef.current.WaffoMinTopUp,
      WaffoNotifyUrl: initialRef.current.WaffoNotifyUrl.trim(),
      WaffoReturnUrl: initialRef.current.WaffoReturnUrl.trim(),
      WaffoPublicCert: initialRef.current.WaffoPublicCert.trim(),
      WaffoSandboxPublicCert: initialRef.current.WaffoSandboxPublicCert.trim(),
      WaffoApiKey: initialRef.current.WaffoApiKey.trim(),
      WaffoPrivateKey: initialRef.current.WaffoPrivateKey.trim(),
      WaffoSandboxApiKey: initialRef.current.WaffoSandboxApiKey.trim(),
      WaffoSandboxPrivateKey: initialRef.current.WaffoSandboxPrivateKey.trim(),
      WaffoPayMethods: JSON.stringify(
        parseWaffoPayMethods(waffoDefaultValues.WaffoPayMethods)
      ),
      WaffoPancakeMerchantID: initialRef.current.WaffoPancakeMerchantID.trim(),
      WaffoPancakePrivateKey: initialRef.current.WaffoPancakePrivateKey.trim(),
      WaffoPancakeReturnURL: removeTrailingSlash(
        initialRef.current.WaffoPancakeReturnURL.trim()
      ),
    }

    const updates: Array<{ key: string; value: string | number | boolean }> = []

    if (sanitized.PayAddress !== initial.PayAddress) {
      updates.push({ key: 'PayAddress', value: sanitized.PayAddress })
    }

    if (sanitized.EpayId !== initial.EpayId) {
      updates.push({ key: 'EpayId', value: sanitized.EpayId })
    }

    if (sanitized.EpayKey && sanitized.EpayKey !== initial.EpayKey) {
      updates.push({ key: 'EpayKey', value: sanitized.EpayKey })
    }

    if (sanitized.Price !== initial.Price) {
      updates.push({ key: 'Price', value: sanitized.Price })
    }

    if (sanitized.MinTopUp !== initial.MinTopUp) {
      updates.push({ key: 'MinTopUp', value: sanitized.MinTopUp })
    }

    if (sanitized.CustomCallbackAddress !== initial.CustomCallbackAddress) {
      updates.push({
        key: 'CustomCallbackAddress',
        value: sanitized.CustomCallbackAddress,
      })
    }

    if (
      normalizeJsonForComparison(sanitized.PayMethods) !==
      normalizeJsonForComparison(initial.PayMethods)
    ) {
      updates.push({ key: 'PayMethods', value: sanitized.PayMethods })
    }

    if (
      normalizeJsonForComparison(sanitized.AmountOptions) !==
      normalizeJsonForComparison(initial.AmountOptions)
    ) {
      updates.push({
        key: 'payment_setting.amount_options',
        value: sanitized.AmountOptions,
      })
    }

    if (
      normalizeJsonForComparison(sanitized.AmountDiscount) !==
      normalizeJsonForComparison(initial.AmountDiscount)
    ) {
      updates.push({
        key: 'payment_setting.amount_discount',
        value: sanitized.AmountDiscount,
      })
    }

    if (sanitized.StripePriceId !== initial.StripePriceId) {
      updates.push({ key: 'StripePriceId', value: sanitized.StripePriceId })
    }

    if (
      sanitized.StripeTestTopupProductID !== initial.StripeTestTopupProductID
    ) {
      updates.push({
        key: 'StripeTestTopupProductID',
        value: sanitized.StripeTestTopupProductID,
      })
    }

    if (
      sanitized.StripeProdTopupProductID !== initial.StripeProdTopupProductID
    ) {
      updates.push({
        key: 'StripeProdTopupProductID',
        value: sanitized.StripeProdTopupProductID,
      })
    }

    if (sanitized.StripeTestAccountID !== initial.StripeTestAccountID) {
      updates.push({
        key: 'StripeTestAccountID',
        value: sanitized.StripeTestAccountID,
      })
    }

    if (sanitized.StripeProdAccountID !== initial.StripeProdAccountID) {
      updates.push({
        key: 'StripeProdAccountID',
        value: sanitized.StripeProdAccountID,
      })
    }

    if (sanitized.StripeTopupEnabled !== initial.StripeTopupEnabled) {
      updates.push({
        key: 'StripeTopupEnabled',
        value: sanitized.StripeTopupEnabled,
      })
    }

    if (sanitized.StripeUnitPrice !== initial.StripeUnitPrice) {
      updates.push({ key: 'StripeUnitPrice', value: sanitized.StripeUnitPrice })
    }

    if (sanitized.StripeMinTopUp !== initial.StripeMinTopUp) {
      updates.push({ key: 'StripeMinTopUp', value: sanitized.StripeMinTopUp })
    }

    if (
      sanitized.StripePromotionCodesEnabled !==
      initial.StripePromotionCodesEnabled
    ) {
      updates.push({
        key: 'StripePromotionCodesEnabled',
        value: sanitized.StripePromotionCodesEnabled,
      })
    }

    if (
      sanitized.CreemApiKey &&
      sanitized.CreemApiKey !== initial.CreemApiKey
    ) {
      updates.push({ key: 'CreemApiKey', value: sanitized.CreemApiKey })
    }

    if (
      sanitized.CreemWebhookSecret &&
      sanitized.CreemWebhookSecret !== initial.CreemWebhookSecret
    ) {
      updates.push({
        key: 'CreemWebhookSecret',
        value: sanitized.CreemWebhookSecret,
      })
    }

    if (sanitized.CreemTestMode !== initial.CreemTestMode) {
      updates.push({ key: 'CreemTestMode', value: sanitized.CreemTestMode })
    }

    if (
      normalizeJsonForComparison(sanitized.CreemProducts) !==
      normalizeJsonForComparison(initial.CreemProducts)
    ) {
      updates.push({ key: 'CreemProducts', value: sanitized.CreemProducts })
    }

    if (sanitized.WaffoEnabled !== initial.WaffoEnabled) {
      updates.push({ key: 'WaffoEnabled', value: sanitized.WaffoEnabled })
    }

    if (sanitized.WaffoSandbox !== initial.WaffoSandbox) {
      updates.push({ key: 'WaffoSandbox', value: sanitized.WaffoSandbox })
    }

    if (sanitized.WaffoMerchantId !== initial.WaffoMerchantId) {
      updates.push({ key: 'WaffoMerchantId', value: sanitized.WaffoMerchantId })
    }

    if (sanitized.WaffoCurrency !== initial.WaffoCurrency) {
      updates.push({ key: 'WaffoCurrency', value: sanitized.WaffoCurrency })
    }

    if (sanitized.WaffoUnitPrice !== initial.WaffoUnitPrice) {
      updates.push({ key: 'WaffoUnitPrice', value: sanitized.WaffoUnitPrice })
    }

    if (sanitized.WaffoMinTopUp !== initial.WaffoMinTopUp) {
      updates.push({ key: 'WaffoMinTopUp', value: sanitized.WaffoMinTopUp })
    }

    if (sanitized.WaffoNotifyUrl !== initial.WaffoNotifyUrl) {
      updates.push({ key: 'WaffoNotifyUrl', value: sanitized.WaffoNotifyUrl })
    }

    if (sanitized.WaffoReturnUrl !== initial.WaffoReturnUrl) {
      updates.push({ key: 'WaffoReturnUrl', value: sanitized.WaffoReturnUrl })
    }

    if (sanitized.WaffoPublicCert !== initial.WaffoPublicCert) {
      updates.push({ key: 'WaffoPublicCert', value: sanitized.WaffoPublicCert })
    }

    if (sanitized.WaffoSandboxPublicCert !== initial.WaffoSandboxPublicCert) {
      updates.push({
        key: 'WaffoSandboxPublicCert',
        value: sanitized.WaffoSandboxPublicCert,
      })
    }

    if (sanitized.WaffoApiKey) {
      updates.push({ key: 'WaffoApiKey', value: sanitized.WaffoApiKey })
    }

    if (sanitized.WaffoPrivateKey) {
      updates.push({ key: 'WaffoPrivateKey', value: sanitized.WaffoPrivateKey })
    }

    if (sanitized.WaffoSandboxApiKey) {
      updates.push({
        key: 'WaffoSandboxApiKey',
        value: sanitized.WaffoSandboxApiKey,
      })
    }

    if (sanitized.WaffoSandboxPrivateKey) {
      updates.push({
        key: 'WaffoSandboxPrivateKey',
        value: sanitized.WaffoSandboxPrivateKey,
      })
    }

    if (
      normalizeJsonForComparison(sanitized.WaffoPayMethods) !==
      normalizeJsonForComparison(initial.WaffoPayMethods)
    ) {
      updates.push({ key: 'WaffoPayMethods', value: sanitized.WaffoPayMethods })
    }

    const hasWaffoPancakeChanges =
      sanitized.WaffoPancakeMerchantID !== initial.WaffoPancakeMerchantID ||
      sanitized.WaffoPancakePrivateKey.length > 0 ||
      sanitized.WaffoPancakeReturnURL !== initial.WaffoPancakeReturnURL ||
      waffoPancakeSelection.storeID !== waffoPancakeSavedBinding.storeID ||
      waffoPancakeSelection.productID !== waffoPancakeSavedBinding.productID

    const hasTopupPromotionChanges =
      topupPromotionSettingsRef.current?.isDirty() ?? false

    if (
      updates.length === 0 &&
      !hasWaffoPancakeChanges &&
      !hasTopupPromotionChanges
    ) {
      toast.info(t('No changes to save'))
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }

    if (hasTopupPromotionChanges) {
      const saved = await topupPromotionSettingsRef.current?.save()
      if (!saved) {
        return
      }
    }

    if (!hasWaffoPancakeChanges) {
      return
    }

    if (!sanitized.WaffoPancakeMerchantID) {
      toast.error(t('Merchant ID is required'))
      return
    }

    if (!waffoPancakeSelection.storeID || !waffoPancakeSelection.productID) {
      toast.error(t('Pick or create both a store and a product before saving.'))
      return
    }

    try {
      const body = await saveWaffoPancakeConfig({
        merchantID: sanitized.WaffoPancakeMerchantID,
        privateKey: sanitized.WaffoPancakePrivateKey,
        returnURL: sanitized.WaffoPancakeReturnURL,
        storeID: waffoPancakeSelection.storeID,
        productID: waffoPancakeSelection.productID,
      })

      if (
        body?.message === 'success' &&
        typeof body.data === 'object' &&
        body.data
      ) {
        const saved = body.data as { product_id: string; store_id: string }
        const savedBinding = {
          storeID: saved.store_id,
          productID: saved.product_id,
        }
        setWaffoPancakeSavedBinding(savedBinding)
        setWaffoPancakeSelection(savedBinding)
        queryClient.invalidateQueries({ queryKey: ['system-options'] })
        toast.success(t('Waffo Pancake settings saved'))
        return
      }

      const reason = typeof body?.data === 'string' ? body.data : undefined
      toast.error(
        reason
          ? `${t('Waffo Pancake save failed')}: ${reason}`
          : t('Waffo Pancake save failed')
      )
    } catch (error) {
      toast.error(
        `${t('Waffo Pancake save failed')}: ${
          error instanceof Error ? error.message : String(error)
        }`
      )
    }
  }

  const currentFormValues = form.watch()
  const waffoValues: WaffoSettingsValues = {
    WaffoEnabled: currentFormValues.WaffoEnabled,
    WaffoApiKey: currentFormValues.WaffoApiKey,
    WaffoPrivateKey: currentFormValues.WaffoPrivateKey,
    WaffoPublicCert: currentFormValues.WaffoPublicCert,
    WaffoSandboxPublicCert: currentFormValues.WaffoSandboxPublicCert,
    WaffoSandboxApiKey: currentFormValues.WaffoSandboxApiKey,
    WaffoSandboxPrivateKey: currentFormValues.WaffoSandboxPrivateKey,
    WaffoSandbox: currentFormValues.WaffoSandbox,
    WaffoMerchantId: currentFormValues.WaffoMerchantId,
    WaffoCurrency: currentFormValues.WaffoCurrency,
    WaffoUnitPrice: currentFormValues.WaffoUnitPrice,
    WaffoMinTopUp: currentFormValues.WaffoMinTopUp,
    WaffoNotifyUrl: currentFormValues.WaffoNotifyUrl,
    WaffoReturnUrl: currentFormValues.WaffoReturnUrl,
    WaffoPayMethods: JSON.stringify(waffoPayMethods),
  }
  const waffoPancakeValues: WaffoPancakeSettingsValues = {
    WaffoPancakeMerchantID: currentFormValues.WaffoPancakeMerchantID,
    WaffoPancakePrivateKey: currentFormValues.WaffoPancakePrivateKey,
    WaffoPancakeReturnURL: currentFormValues.WaffoPancakeReturnURL,
  }

  return (
    <SettingsSection title={t('Payment Gateway')}>
      <Form {...form}>
        <SettingsForm
          onSubmit={form.handleSubmit(onSubmit)}
          className='gap-y-8'
          data-no-autosubmit='true'
        >
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending || isSubmitting}
            saveLabel='Save all settings'
          />
          <Tabs defaultValue='general' className='min-w-0'>
            <div className='overflow-x-auto pb-1'>
              <TabsList className='grid min-w-[44rem] grid-cols-6'>
                <TabsTrigger value='general'>{t('General')}</TabsTrigger>
                <TabsTrigger value='epay'>Epay</TabsTrigger>
                <TabsTrigger value='stripe'>{t('Stripe')}</TabsTrigger>
                <TabsTrigger value='creem'>Creem</TabsTrigger>
                <TabsTrigger value='waffo-pancake'>Waffo Pancake</TabsTrigger>
                <TabsTrigger value='waffo'>Waffo</TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value='general' className={paymentTabContentClassName}>
              <div className='space-y-4'>
                <div>
                  <h3 className='text-lg font-medium'>
                    {t('General Settings')}
                  </h3>
                  <p className='text-muted-foreground text-sm'>
                    {t('Shared configuration for all payment gateways')}
                  </p>
                </div>

                <TopupPromotionSettings
                  saveHandleRef={topupPromotionSettingsRef}
                />

                <FormField
                  control={form.control}
                  name='PayMethods'
                  render={({ field }) => (
                    <FormItem>
                      <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                        <FormLabel>{t('Payment methods')}</FormLabel>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          onClick={() =>
                            setPayMethodsVisualMode(!payMethodsVisualMode)
                          }
                          className='w-full sm:w-auto'
                        >
                          {payMethodsVisualMode ? (
                            <>
                              <Code2 className='mr-2 h-3 w-3' />
                              {t('JSON Editor')}
                            </>
                          ) : (
                            <>
                              <Eye className='mr-2 h-3 w-3' />
                              {t('Visual Editor')}
                            </>
                          )}
                        </Button>
                      </div>
                      <FormControl>
                        {payMethodsVisualMode ? (
                          <PaymentMethodsVisualEditor
                            value={field.value}
                            onChange={field.onChange}
                          />
                        ) : (
                          <Textarea
                            rows={4}
                            placeholder={t(
                              '[{"name":"支付宝","type":"alipay","icon":"SiAlipay"}]'
                            )}
                            {...field}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          />
                        )}
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Configured as PayMethods JSON. The type value decides which payment flow is used: stripe for Stripe, waffo_pancake for Waffo Pancake, and other values are sent to Epay as the type parameter.'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <div className='grid gap-6 md:grid-cols-2 md:items-start'>
                  <FormField
                    control={form.control}
                    name='AmountOptions'
                    render={({ field }) => (
                      <FormItem>
                        <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                          <FormLabel>{t('Top-up amount options')}</FormLabel>
                          <Button
                            type='button'
                            variant='outline'
                            size='sm'
                            onClick={() =>
                              setAmountOptionsVisualMode(
                                !amountOptionsVisualMode
                              )
                            }
                            className='w-full sm:w-auto'
                          >
                            {amountOptionsVisualMode ? (
                              <>
                                <Code2 className='mr-2 h-3 w-3' />
                                {t('JSON Editor')}
                              </>
                            ) : (
                              <>
                                <Eye className='mr-2 h-3 w-3' />
                                {t('Visual Editor')}
                              </>
                            )}
                          </Button>
                        </div>
                        <FormControl>
                          {amountOptionsVisualMode ? (
                            <AmountOptionsVisualEditor
                              value={field.value}
                              onChange={field.onChange}
                            />
                          ) : (
                            <Textarea
                              rows={4}
                              placeholder='[10, 20, 50, 100]'
                              {...field}
                              onChange={(event) =>
                                field.onChange(event.target.value)
                              }
                            />
                          )}
                        </FormControl>
                        <FormDescription>
                          {t('Preset recharge amounts (JSON array)')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='AmountDiscount'
                    render={({ field }) => (
                      <FormItem>
                        <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                          <FormLabel>{t('Amount discount')}</FormLabel>
                          <Button
                            type='button'
                            variant='outline'
                            size='sm'
                            onClick={() =>
                              setAmountDiscountVisualMode(
                                !amountDiscountVisualMode
                              )
                            }
                            className='w-full sm:w-auto'
                          >
                            {amountDiscountVisualMode ? (
                              <>
                                <Code2 className='mr-2 h-3 w-3' />
                                {t('JSON Editor')}
                              </>
                            ) : (
                              <>
                                <Eye className='mr-2 h-3 w-3' />
                                {t('Visual Editor')}
                              </>
                            )}
                          </Button>
                        </div>
                        <FormControl>
                          {amountDiscountVisualMode ? (
                            <AmountDiscountVisualEditor
                              value={field.value}
                              onChange={field.onChange}
                            />
                          ) : (
                            <Textarea
                              rows={4}
                              placeholder='{"100":0.95,"200":0.9}'
                              {...field}
                              onChange={(event) =>
                                field.onChange(event.target.value)
                              }
                            />
                          )}
                        </FormControl>
                        <FormDescription>
                          {t('Discount map by recharge amount (JSON object)')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>
            </TabsContent>

            <TabsContent value='epay' className={paymentTabContentClassName}>
              <div className='space-y-4'>
                <div>
                  <h3 className='text-lg font-medium'>{t('Epay Gateway')}</h3>
                  <p className='text-muted-foreground text-sm'>
                    {t('Configuration for Epay payment integration')}
                  </p>
                </div>

                <Alert>
                  <ShieldAlert className='h-4 w-4' />
                  <AlertTitle>{t('Epay safety reminder')}</AlertTitle>
                  <AlertDescription>
                    {t(
                      'Epay is a payment protocol, not a specific official website. Verify the provider yourself and do not trust random third-party Epay deployments.'
                    )}
                  </AlertDescription>
                </Alert>

                <div className='grid gap-6 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='PayAddress'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Epay endpoint')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder={t('https://pay.example.com')}
                            {...field}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Base address provided by your Epay service')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='CustomCallbackAddress'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Callback address')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder={t('https://gateway.example.com')}
                            {...field}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Only enter the site origin, for example https://api.example.com. Do not include any path such as /api/user/epay/notify. Leave blank to use the server address.'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <div className='grid gap-6 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='Price'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Epay charge per USD credit')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            step='0.01'
                            min={0}
                            {...safeNumberFieldProps(field)}
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Legacy Epay conversion only. It is not used by Stripe Product Checkout or billing-currency FX.'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name='MinTopUp'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Epay minimum top-up (USD)')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            step='0.01'
                            min={0}
                            {...safeNumberFieldProps(field)}
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Legacy Epay minimum only. Stripe uses the per-currency minimums in General.'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <div className='grid gap-6 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='EpayId'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Epay merchant ID')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder='10001'
                            autoComplete='off'
                            {...field}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='EpayKey'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Epay secret key')}</FormLabel>
                        <FormControl>
                          <Input
                            type='password'
                            placeholder={t('Enter new key to update')}
                            autoComplete='new-password'
                            {...field}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Leave blank unless rotating the secret')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>
            </TabsContent>

            <TabsContent value='stripe' className={paymentTabContentClassName}>
              <div className='space-y-4'>
                <div>
                  <h3 className='text-lg font-medium'>{t('Stripe Gateway')}</h3>
                  <p className='text-muted-foreground text-sm'>
                    {t('Configuration for Stripe payment integration')}
                  </p>
                </div>

                <div className='rounded-md bg-blue-50 p-4 text-sm text-blue-900 dark:bg-blue-950 dark:text-blue-100'>
                  <p className='mb-2 font-medium'>
                    {t('Webhook Configuration:')}
                  </p>
                  <ul className='list-inside list-disc space-y-1'>
                    <li>
                      {t('Webhook URL:')}{' '}
                      <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
                        https://www.novapuraai.com/api/stripe/webhook
                      </code>
                    </li>
                    <li>
                      {t('Local test forwarding:')}{' '}
                      <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
                        stripe listen --forward-to
                        http://localhost:3000/api/stripe/webhook
                      </code>
                    </li>
                    <li>
                      {t('Required events:')}{' '}
                      <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
                        {t('checkout.session.completed')}
                      </code>{' '}
                      {t('and')}{' '}
                      <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
                        {t('checkout.session.expired')}
                      </code>
                    </li>
                    <li>
                      {t('Configure at:')}{' '}
                      <a
                        href='https://dashboard.stripe.com/developers'
                        target='_blank'
                        rel='noreferrer'
                        className='underline hover:no-underline'
                      >
                        {t('Stripe Dashboard')}
                      </a>
                    </li>
                  </ul>
                </div>

                <Alert>
                  <ShieldAlert />
                  <AlertTitle>
                    {t('Stripe secrets are managed securely')}
                  </AlertTitle>
                  <AlertDescription>
                    {t(
                      'Update Stripe keys in Secret Manager or environment variables. This dashboard shows only whether each value is configured and never returns the secret.'
                    )}
                  </AlertDescription>
                </Alert>

                <div className='grid gap-4 xl:grid-cols-2'>
                  <StripeEnvironmentCard
                    control={form.control}
                    environment='test'
                    isActive={
                      stripeCredentialStatus.runtimeEnvironment === 'test'
                    }
                    status={stripeCredentialStatus.test}
                    accountField='StripeTestAccountID'
                    productField='StripeTestTopupProductID'
                  />
                  <StripeEnvironmentCard
                    control={form.control}
                    environment='production'
                    isActive={
                      stripeCredentialStatus.runtimeEnvironment === 'production'
                    }
                    status={stripeCredentialStatus.production}
                    accountField='StripeProdAccountID'
                    productField='StripeProdTopupProductID'
                  />
                </div>

                <div className='grid gap-6 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='StripeTopupEnabled'
                    render={({ field }) => (
                      <SettingsSwitchItem>
                        <SettingsSwitchContent>
                          <FormLabel>{t('Enable wallet top-ups')}</FormLabel>
                          <FormDescription>
                            {t(
                              'Allow the multi-currency Stripe Checkout wallet top-up flow.'
                            )}
                          </FormDescription>
                        </SettingsSwitchContent>
                        <FormControl>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                      </SettingsSwitchItem>
                    )}
                  />
                </div>

                <div className='grid gap-6 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='StripePromotionCodesEnabled'
                    render={({ field }) => (
                      <SettingsSwitchItem>
                        <SettingsSwitchContent>
                          <FormLabel>{t('Promotion codes')}</FormLabel>
                          <FormDescription>
                            {t('Allow users to enter promo codes')}
                          </FormDescription>
                        </SettingsSwitchContent>
                        <FormControl>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                      </SettingsSwitchItem>
                    )}
                  />
                </div>
              </div>
            </TabsContent>

            <TabsContent value='creem' className={paymentTabContentClassName}>
              <div className='space-y-4'>
                <div>
                  <h3 className='text-lg font-medium'>{t('Creem Gateway')}</h3>
                  <p className='text-muted-foreground text-sm'>
                    {t('Configuration for Creem payment integration')}
                  </p>
                </div>

                <div className='rounded-md bg-blue-50 p-4 text-sm text-blue-900 dark:bg-blue-950 dark:text-blue-100'>
                  <p className='mb-2 font-medium'>
                    {t('Webhook Configuration:')}
                  </p>
                  <ul className='list-inside list-disc space-y-1'>
                    <li>
                      {t('Webhook URL:')}{' '}
                      <code className='rounded bg-blue-100 px-1 py-0.5 text-xs dark:bg-blue-900'>
                        {'<ServerAddress>/api/creem/webhook'}
                      </code>
                    </li>
                    <li>{t('Configure in your Creem dashboard')}</li>
                  </ul>
                </div>

                <div className='grid gap-6 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='CreemApiKey'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('API Key')}</FormLabel>
                        <FormControl>
                          <Input
                            type='password'
                            placeholder={t('Enter Creem API key')}
                            autoComplete='new-password'
                            {...field}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Creem API key (leave blank unless updating)')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='CreemWebhookSecret'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Webhook Secret')}</FormLabel>
                        <FormControl>
                          <Input
                            type='password'
                            placeholder={t('Enter webhook secret')}
                            autoComplete='new-password'
                            {...field}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Webhook signing secret (leave blank unless updating)'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <FormField
                  control={form.control}
                  name='CreemTestMode'
                  render={({ field }) => (
                    <SettingsSwitchItem>
                      <SettingsSwitchContent>
                        <FormLabel>{t('Test Mode')}</FormLabel>
                        <FormDescription>
                          {t('Enable test mode for Creem payments')}
                        </FormDescription>
                      </SettingsSwitchContent>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </SettingsSwitchItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='CreemProducts'
                  render={({ field }) => (
                    <FormItem>
                      <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                        <FormLabel>{t('Products')}</FormLabel>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          onClick={() =>
                            setCreemProductsVisualMode(!creemProductsVisualMode)
                          }
                          className='w-full sm:w-auto'
                        >
                          {creemProductsVisualMode ? (
                            <>
                              <Code2 className='mr-2 h-3 w-3' />
                              {t('JSON Editor')}
                            </>
                          ) : (
                            <>
                              <Eye className='mr-2 h-3 w-3' />
                              {t('Visual Editor')}
                            </>
                          )}
                        </Button>
                      </div>
                      <FormControl>
                        {creemProductsVisualMode ? (
                          <CreemProductsVisualEditor
                            value={field.value}
                            onChange={field.onChange}
                          />
                        ) : (
                          <Textarea
                            rows={4}
                            placeholder='[{"name":"Basic","productId":"prod_xxx","price":10,"quota":500000,"currency":"USD"}]'
                            {...field}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          />
                        )}
                      </FormControl>
                      <FormDescription>
                        {t('Configure Creem products. Provide a JSON array.')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </TabsContent>

            <TabsContent
              value='waffo-pancake'
              className={paymentTabContentClassName}
            >
              <WaffoPancakeSettingsSection
                defaultValues={waffoPancakeDefaultValues}
                values={waffoPancakeValues}
                onValueChange={setWaffoPancakeValue}
                selectedBinding={waffoPancakeSelection}
                savedBinding={waffoPancakeSavedBinding}
                onSelectedBindingChange={setWaffoPancakeSelection}
              />
            </TabsContent>

            <TabsContent value='waffo' className={paymentTabContentClassName}>
              <WaffoSettingsSection
                values={waffoValues}
                onValueChange={setWaffoValue}
                payMethods={waffoPayMethods}
                onPayMethodsChange={setWaffoPayMethods}
              />
            </TabsContent>
          </Tabs>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
