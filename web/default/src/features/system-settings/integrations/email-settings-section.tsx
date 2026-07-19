import { useQuery } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { ChevronDown } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
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
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { getTransactionalEmailHealth } from '../api'
import { EmailProviderHealthCard } from './email-provider-health-card'
import { isValidEmailSender } from './email-sender'
import {
  getWriteOnlySecretPlaceholder,
  hasWriteOnlySecretReplacement,
} from '../auth/write-only-secret'

const createEmailSchema = (t: (key: string) => string) =>
  z.object({
    SMTPServer: z.string(),
    SMTPPort: z.string().refine((value) => {
      const trimmed = value.trim()
      if (!trimmed) return true
      return /^\d+$/.test(trimmed)
    }, t('Port must be a positive integer')),
    SMTPAccount: z.string(),
    SMTPFrom: z
      .string()
      .refine(
        isValidEmailSender,
        t('Enter a valid sender address or leave blank')
      ),
    SMTPToken: z.string().optional(),
    SMTPSSLEnabled: z.boolean(),
    SMTPStartTLSEnabled: z.boolean(),
    SMTPInsecureSkipVerify: z.boolean(),
    SMTPForceAuthLogin: z.boolean(),
  })

type EmailFormValues = z.infer<ReturnType<typeof createEmailSchema>>

type EmailSettingsSectionProps = {
  defaultValues: EmailFormValues & {
    SMTPTokenConfigured: boolean
  }
}

type SmtpSecurityMode = 'none' | 'ssl_tls' | 'starttls'

function getSmtpSecurityMode(values: {
  SMTPSSLEnabled: boolean
  SMTPStartTLSEnabled: boolean
}): SmtpSecurityMode {
  if (values.SMTPSSLEnabled) return 'ssl_tls'
  if (values.SMTPStartTLSEnabled) return 'starttls'
  return 'none'
}

export function EmailSettingsSection({
  defaultValues,
}: EmailSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const emailSchema = createEmailSchema(t)
  const [tokenConfigured, setTokenConfigured] = useState(
    defaultValues.SMTPTokenConfigured
  )
  const [legacySmtpOpen, setLegacySmtpOpen] = useState(false)

  const healthQuery = useQuery({
    queryKey: ['transactional-email-health'],
    queryFn: getTransactionalEmailHealth,
    staleTime: 30_000,
  })
  const sesSelected =
    healthQuery.data?.success &&
    healthQuery.data.data.selected_provider === 'ses'

  const formDefaults: EmailFormValues = {
    SMTPServer: defaultValues.SMTPServer,
    SMTPPort: defaultValues.SMTPPort,
    SMTPAccount: defaultValues.SMTPAccount,
    SMTPFrom: defaultValues.SMTPFrom,
    SMTPToken: '',
    SMTPSSLEnabled: defaultValues.SMTPSSLEnabled,
    SMTPStartTLSEnabled: defaultValues.SMTPStartTLSEnabled,
    SMTPInsecureSkipVerify: defaultValues.SMTPInsecureSkipVerify,
    SMTPForceAuthLogin: defaultValues.SMTPForceAuthLogin,
  }

  const form = useForm<EmailFormValues>({
    resolver: zodResolver(emailSchema),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  useEffect(() => {
    setTokenConfigured(defaultValues.SMTPTokenConfigured)
  }, [defaultValues.SMTPTokenConfigured])

  const onSubmit = async (values: EmailFormValues) => {
    const securityMode = getSmtpSecurityMode(values)
    const sanitized = {
      SMTPServer: values.SMTPServer.trim(),
      SMTPPort: values.SMTPPort.trim(),
      SMTPAccount: values.SMTPAccount.trim(),
      SMTPFrom: values.SMTPFrom.trim(),
      SMTPSSLEnabled: securityMode === 'ssl_tls',
      SMTPStartTLSEnabled: securityMode === 'starttls',
      SMTPInsecureSkipVerify: values.SMTPInsecureSkipVerify,
      SMTPForceAuthLogin: values.SMTPForceAuthLogin,
    }

    const initial = {
      SMTPServer: defaultValues.SMTPServer.trim(),
      SMTPPort: defaultValues.SMTPPort.trim(),
      SMTPAccount: defaultValues.SMTPAccount.trim(),
      SMTPFrom: defaultValues.SMTPFrom.trim(),
      SMTPSSLEnabled: defaultValues.SMTPSSLEnabled,
      SMTPStartTLSEnabled: defaultValues.SMTPStartTLSEnabled,
      SMTPInsecureSkipVerify: defaultValues.SMTPInsecureSkipVerify,
      SMTPForceAuthLogin: defaultValues.SMTPForceAuthLogin,
    }

    const updates: Array<{ key: string; value: string | boolean }> = []

    if (sanitized.SMTPServer !== initial.SMTPServer) {
      updates.push({ key: 'SMTPServer', value: sanitized.SMTPServer })
    }

    if (sanitized.SMTPPort !== initial.SMTPPort) {
      updates.push({ key: 'SMTPPort', value: sanitized.SMTPPort })
    }

    if (sanitized.SMTPAccount !== initial.SMTPAccount) {
      updates.push({ key: 'SMTPAccount', value: sanitized.SMTPAccount })
    }

    if (sanitized.SMTPFrom !== initial.SMTPFrom) {
      updates.push({ key: 'SMTPFrom', value: sanitized.SMTPFrom })
    }

    if (sanitized.SMTPSSLEnabled !== initial.SMTPSSLEnabled) {
      updates.push({
        key: 'SMTPSSLEnabled',
        value: sanitized.SMTPSSLEnabled,
      })
    }

    if (sanitized.SMTPStartTLSEnabled !== initial.SMTPStartTLSEnabled) {
      updates.push({
        key: 'SMTPStartTLSEnabled',
        value: sanitized.SMTPStartTLSEnabled,
      })
    }

    if (sanitized.SMTPInsecureSkipVerify !== initial.SMTPInsecureSkipVerify) {
      updates.push({
        key: 'SMTPInsecureSkipVerify',
        value: sanitized.SMTPInsecureSkipVerify,
      })
    }

    if (sanitized.SMTPForceAuthLogin !== initial.SMTPForceAuthLogin) {
      updates.push({
        key: 'SMTPForceAuthLogin',
        value: sanitized.SMTPForceAuthLogin,
      })
    }

    if (hasWriteOnlySecretReplacement(values.SMTPToken ?? '')) {
      updates.push({ key: 'SMTPToken', value: values.SMTPToken ?? '' })
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }

    if (hasWriteOnlySecretReplacement(values.SMTPToken ?? '')) {
      setTokenConfigured(true)
    }
    form.reset({ ...values, SMTPToken: '' })
  }

  return (
    <SettingsSection title={t('Transactional Email')}>
      <EmailProviderHealthCard />

      <Collapsible
        open={legacySmtpOpen}
        onOpenChange={setLegacySmtpOpen}
        className='border-border mt-4 rounded-xl border'
      >
        <CollapsibleTrigger className='hover:bg-muted/40 flex w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors'>
          <div className='min-w-0'>
            <h4 className='text-sm font-semibold'>
              {t(
                'Legacy SMTP compatibility settings (only used when the SMTP provider is selected)'
              )}
            </h4>
            <p className='text-muted-foreground mt-1 text-xs'>
              {sesSelected
                ? t(
                    'Amazon SES API is selected. These legacy SMTP host, port, username, and password fields are not required and are not used for SES API delivery.'
                  )
                : t(
                    'These settings remain available for legacy SMTP compatibility. Prefer the transactional provider cards above for Brevo and Amazon SES API.'
                  )}
            </p>
          </div>
          <ChevronDown
            className={cn(
              'text-muted-foreground size-4 shrink-0 transition-transform',
              legacySmtpOpen && 'rotate-180'
            )}
            aria-hidden='true'
          />
        </CollapsibleTrigger>

        <CollapsibleContent className='border-border border-t px-4 py-4'>
          <Form {...form}>
            <SettingsForm
              onSubmit={form.handleSubmit(onSubmit)}
              autoComplete='off'
            >
              {/*
                When SES API is selected, keep Save SMTP out of the page-level
                primary action portal. Place it only inside this collapsed section.
              */}
              {!sesSelected && (
                <SettingsPageFormActions
                  onSave={form.handleSubmit(onSubmit)}
                  isSaving={updateOption.isPending}
                  saveLabel='Save SMTP settings'
                />
              )}

              <FormField
                control={form.control}
                name='SMTPServer'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('SMTP Host')}</FormLabel>
                    <FormControl>
                      <Input
                        autoComplete='off'
                        placeholder={t('smtp.example.com')}
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Hostname or IP of your SMTP provider')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className='grid gap-6 md:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='SMTPPort'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Port')}</FormLabel>
                      <FormControl>
                        <Input
                          autoComplete='off'
                          type='number'
                          placeholder='587'
                          {...field}
                          onChange={(event) =>
                            field.onChange(event.target.value)
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Common ports include 25, 465, and 587')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormItem>
                  <FormLabel>{t('SMTP encryption')}</FormLabel>
                  <FormControl>
                    <RadioGroup
                      value={getSmtpSecurityMode({
                        SMTPSSLEnabled: form.watch('SMTPSSLEnabled'),
                        SMTPStartTLSEnabled: form.watch('SMTPStartTLSEnabled'),
                      })}
                      onValueChange={(value) => {
                        const mode = value as SmtpSecurityMode
                        form.setValue('SMTPSSLEnabled', mode === 'ssl_tls', {
                          shouldDirty: true,
                        })
                        form.setValue(
                          'SMTPStartTLSEnabled',
                          mode === 'starttls',
                          {
                            shouldDirty: true,
                          }
                        )
                      }}
                      className='gap-3'
                    >
                      <div className='flex items-center gap-2'>
                        <RadioGroupItem value='none' id='smtp-security-none' />
                        <Label
                          htmlFor='smtp-security-none'
                          className='cursor-pointer font-normal'
                        >
                          {t('No encryption')}
                        </Label>
                      </div>
                      <div className='flex items-center gap-2'>
                        <RadioGroupItem
                          value='ssl_tls'
                          id='smtp-security-ssl-tls'
                        />
                        <Label
                          htmlFor='smtp-security-ssl-tls'
                          className='cursor-pointer font-normal'
                        >
                          {t('SSL/TLS')}
                        </Label>
                      </div>
                      <div className='flex items-center gap-2'>
                        <RadioGroupItem
                          value='starttls'
                          id='smtp-security-starttls'
                        />
                        <Label
                          htmlFor='smtp-security-starttls'
                          className='cursor-pointer font-normal'
                        >
                          {t('STARTTLS')}
                        </Label>
                      </div>
                    </RadioGroup>
                  </FormControl>
                  <FormDescription>
                    {t('Choose one SMTP transport security mode')}
                  </FormDescription>
                </FormItem>

                <FormField
                  control={form.control}
                  name='SMTPInsecureSkipVerify'
                  render={({ field }) => (
                    <SettingsSwitchItem>
                      <SettingsSwitchContent>
                        <FormLabel>
                          {t('Skip SMTP TLS certificate verification')}
                        </FormLabel>
                        <FormDescription>
                          {t(
                            'Allow self-signed or hostname-mismatched SMTP certificates'
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
                  name='SMTPForceAuthLogin'
                  render={({ field }) => (
                    <SettingsSwitchItem>
                      <SettingsSwitchContent>
                        <FormLabel>{t('Force AUTH LOGIN')}</FormLabel>
                        <FormDescription>
                          {t(
                            'Force SMTP authentication using AUTH LOGIN method'
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

              <FormField
                control={form.control}
                name='SMTPAccount'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Username')}</FormLabel>
                    <FormControl>
                      <Input
                        autoComplete='off'
                        placeholder={t('noreply@example.com')}
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Login name issued by your SMTP provider. For Amazon SES SMTP, use the generated SMTP username, not an AWS access key ID.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='SMTPFrom'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('From Address')}</FormLabel>
                    <FormControl>
                      <Input
                        autoComplete='off'
                        placeholder={t('New API &lt;noreply@example.com&gt;')}
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Email used in outgoing messages. You may enter noreply@example.com or a display form such as NovaPuraAI <noreply@novapuraai.com>.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='SMTPToken'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Password / Access Token')}</FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        placeholder={getWriteOnlySecretPlaceholder(
                          tokenConfigured,
                          t('Your AWS SES SMTP password')
                        )}
                        autoComplete='new-password'
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Leave blank to keep the current password. The value is never returned by the settings API. When SMTP_TOKEN is set in the environment (e.g. Cloud Run + Secret Manager), it overrides the value saved here.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {sesSelected && (
                <div className='flex justify-end pt-2'>
                  <Button
                    type='submit'
                    size='sm'
                    variant='outline'
                    disabled={updateOption.isPending}
                  >
                    {updateOption.isPending
                      ? t('Saving...')
                      : t('Save SMTP settings')}
                  </Button>
                </div>
              )}
            </SettingsForm>
          </Form>
        </CollapsibleContent>
      </Collapsible>
    </SettingsSection>
  )
}
