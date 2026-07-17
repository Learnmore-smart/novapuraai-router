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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
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

const botProtectionSchema = z.object({
  TurnstileCheckEnabled: z.boolean(),
  TurnstileSiteKey: z.string().optional(),
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

  const formDefaults: BotProtectionFormValues = {
    TurnstileCheckEnabled: defaultValues.TurnstileCheckEnabled,
    TurnstileSiteKey: defaultValues.TurnstileSiteKey,
    TurnstileAllowedHostnames: defaultValues.TurnstileAllowedHostnames,
  }

  const form = useForm<BotProtectionFormValues>({
    resolver: zodResolver(botProtectionSchema),
    defaultValues: formDefaults,
  })

  useEffect(() => {
    form.reset({
      TurnstileCheckEnabled: defaultValues.TurnstileCheckEnabled,
      TurnstileSiteKey: defaultValues.TurnstileSiteKey,
      TurnstileAllowedHostnames: defaultValues.TurnstileAllowedHostnames,
    })
  }, [defaultValues, form])

  const onSubmit = async (data: BotProtectionFormValues) => {
    const enabledChanged =
      data.TurnstileCheckEnabled !== defaultValues.TurnstileCheckEnabled
    const updates = Object.entries(data).filter(
      ([key, value]) =>
        key !== 'TurnstileCheckEnabled' &&
        value !== defaultValues[key as keyof BotProtectionFormValues]
    )

    if (enabledChanged && !data.TurnstileCheckEnabled) {
      await updateOption.mutateAsync({
        key: 'TurnstileCheckEnabled',
        value: false,
      })
    }
    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value: value ?? '' })
    }
    if (enabledChanged && data.TurnstileCheckEnabled) {
      await updateOption.mutateAsync({
        key: 'TurnstileCheckEnabled',
        value: true,
      })
    }
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

          <FormItem>
            <FormLabel>{t('Secret Key')}</FormLabel>
            <Input
              readOnly
              value={
                defaultValues.TurnstileSecretKeyConfigured
                  ? t('Configured through Secret Manager')
                  : t('Not configured in environment')
              }
              className='bg-muted text-muted-foreground'
              autoComplete='off'
            />
            <FormDescription>
              {t(
                'Set TURNSTILE_SECRET_KEY via environment (Google Secret Manager on Cloud Run). It is never stored in the database or returned by the settings API.'
              )}
            </FormDescription>
          </FormItem>

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
