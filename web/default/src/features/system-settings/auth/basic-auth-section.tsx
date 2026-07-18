import { zodResolver } from '@hookform/resolvers/zod'
import { useMemo } from 'react'
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
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

const basicAuthSchema = z.object({
  PasswordLoginEnabled: z.boolean(),
  PasswordRegisterEnabled: z.boolean(),
  EmailVerificationEnabled: z.boolean(),
  RegisterEnabled: z.boolean(),
  EmailDomainRestrictionEnabled: z.boolean(),
  EmailAliasRestrictionEnabled: z.boolean(),
  EmailDomainWhitelist: z.string(),
})

type BasicAuthFormValues = z.infer<typeof basicAuthSchema>

type BasicAuthSectionProps = {
  defaultValues: BasicAuthFormValues
}

export function BasicAuthSection({ defaultValues }: BasicAuthSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const formDefaults = useMemo<BasicAuthFormValues>(
    () => ({
      ...defaultValues,
      EmailDomainWhitelist: defaultValues.EmailDomainWhitelist.split(',')
        .map((domain) => domain.trim())
        .filter(Boolean)
        .join('\n'),
    }),
    [defaultValues]
  )

  const form = useForm<BasicAuthFormValues>({
    resolver: zodResolver(basicAuthSchema),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  const onSubmit = async (data: BasicAuthFormValues) => {
    const updates: Array<{ key: string; value: string | boolean }> = []

    Object.entries(data).forEach(([key, value]) => {
      if (key === 'EmailDomainWhitelist') {
        if (typeof value !== 'string') return
        const domains = value
          .split('\n')
          .map((domain) => domain.trim())
          .filter(Boolean)
          .join(',')
        if (domains !== defaultValues.EmailDomainWhitelist) {
          updates.push({ key, value: domains })
        }
      } else if (value !== defaultValues[key as keyof typeof defaultValues]) {
        updates.push({ key, value })
      }
    })

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  return (
    <SettingsSection title={t('Basic Authentication')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='PasswordLoginEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Password Login')}</FormLabel>
                  <FormDescription>
                    {t('Allow users to log in with password')}
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
            name='RegisterEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Registration Enabled')}</FormLabel>
                  <FormDescription>
                    {t('Allow new users to register')}
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
            name='PasswordRegisterEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Password Registration')}</FormLabel>
                  <FormDescription>
                    {t('Allow registration with password')}
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
            name='EmailVerificationEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Email Verification')}</FormLabel>
                  <FormDescription>
                    {t('Require email verification for new accounts')}
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
            name='EmailDomainRestrictionEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Email Domain Restriction')}</FormLabel>
                  <FormDescription>
                    {t('Only allow specific email domains')}
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
            name='EmailAliasRestrictionEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Email Alias Restriction')}</FormLabel>
                  <FormDescription>
                    {t('Block email aliases (e.g., user+alias@domain.com)')}
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
            name='EmailDomainWhitelist'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Email Domain Whitelist')}</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder={t('example.com&#10;company.com')}
                    rows={4}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'One domain per line (only used when domain restriction is enabled)'
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
