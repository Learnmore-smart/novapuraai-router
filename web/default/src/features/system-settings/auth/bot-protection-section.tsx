import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

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

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  getWriteOnlySecretPlaceholder,
  hasWriteOnlySecretReplacement,
  orderAuthOptionUpdates,
} from './write-only-secret'

const botProtectionSchema = z.object({
  TurnstileCheckEnabled: z.boolean(),
  TurnstileSiteKey: z.string().optional(),
  TurnstileSecretKey: z.string().optional(),
  TurnstileAllowedHostnames: z.string().optional(),
})

type BotProtectionFormValues = z.infer<typeof botProtectionSchema>

type BotProtectionSectionProps = {
  defaultValues: BotProtectionFormValues & {
    TurnstileSecretKeyConfigured: boolean
  }
}

export function BotProtectionSection({
  defaultValues,
}: BotProtectionSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [secretConfigured, setSecretConfigured] = useState(
    defaultValues.TurnstileSecretKeyConfigured
  )

  const formDefaults: BotProtectionFormValues = {
    TurnstileCheckEnabled: defaultValues.TurnstileCheckEnabled,
    TurnstileSiteKey: defaultValues.TurnstileSiteKey,
    TurnstileSecretKey: '',
    TurnstileAllowedHostnames: defaultValues.TurnstileAllowedHostnames,
  }

  const form = useForm<BotProtectionFormValues>({
    resolver: zodResolver(botProtectionSchema),
    defaultValues: formDefaults,
  })

  useEffect(() => {
    setSecretConfigured(defaultValues.TurnstileSecretKeyConfigured)
    form.reset({
      TurnstileCheckEnabled: defaultValues.TurnstileCheckEnabled,
      TurnstileSiteKey: defaultValues.TurnstileSiteKey,
      TurnstileSecretKey: '',
      TurnstileAllowedHostnames: defaultValues.TurnstileAllowedHostnames,
    })
  }, [defaultValues, form])

  const onSubmit = async (data: BotProtectionFormValues) => {
    const updates = Object.entries(data).filter(([key, value]) => {
      if (key === 'TurnstileSecretKey') {
        return typeof value === 'string' && hasWriteOnlySecretReplacement(value)
      }
      return value !== defaultValues[key as keyof BotProtectionFormValues]
    })

    const orderedUpdates = orderAuthOptionUpdates(
      updates,
      'TurnstileCheckEnabled',
      data.TurnstileCheckEnabled
    )
    for (const [key, value] of orderedUpdates) {
      const result = await updateOption.mutateAsync({
        key,
        value: value ?? '',
      })
      if (!result.success) return
    }

    if (hasWriteOnlySecretReplacement(data.TurnstileSecretKey ?? '')) {
      setSecretConfigured(true)
    }
    form.reset({ ...data, TurnstileSecretKey: '' })
  }

  return (
    <SettingsSection title={t('Bot Protection')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='TurnstileCheckEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable Turnstile')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Protect login and registration with Cloudflare Turnstile'
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

          <FormField
            control={form.control}
            name='TurnstileSiteKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Site Key')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Your Turnstile site key')}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='TurnstileSecretKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Secret Key')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    placeholder={getWriteOnlySecretPlaceholder(
                      secretConfigured,
                      t('Your Turnstile secret key')
                    )}
                    autoComplete='new-password'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Leave blank to keep the current secret. The value is never returned by the settings API.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='TurnstileAllowedHostnames'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Allowed hostnames')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('novapuraai.com, localhost')}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Comma-separated hostnames accepted from Turnstile Siteverify. Do not include a scheme or path.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
