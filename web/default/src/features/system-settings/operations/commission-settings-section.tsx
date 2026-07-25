import { zodResolver } from '@hookform/resolvers/zod'
import type { ChangeEvent } from 'react'
import type { Resolver } from 'react-hook-form'
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

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const commissionSchema = z.object({
  AffCommissionRate: z.coerce.number().min(0).max(1),
  MinWithdrawalCents: z.coerce.number().int().min(0),
  CommissionFreezeDays: z.coerce.number().int().min(0),
})

type CommissionFormValues = z.infer<typeof commissionSchema>
type NumberInputValue = number | ''

type CommissionSettingsSectionProps = {
  defaultValues: CommissionFormValues
}

export function CommissionSettingsSection({
  defaultValues,
}: CommissionSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const handleNumberChange =
    (onChange: (value: NumberInputValue) => void) =>
    (event: ChangeEvent<HTMLInputElement>) => {
      const value = event.currentTarget.valueAsNumber
      onChange(Number.isNaN(value) ? '' : value)
    }

  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<CommissionFormValues>({
      resolver: zodResolver(commissionSchema) as Resolver<
        CommissionFormValues,
        unknown,
        CommissionFormValues
      >,
      defaultValues,
      onSubmit: async (_data, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          await updateOption.mutateAsync({
            key,
            value: value as string | number | boolean,
          })
        }
      },
    })

  const minWithdrawalUSD =
    (form.watch('MinWithdrawalCents') as number) / 100

  return (
    <SettingsSection title={t('Affiliate Commission')}>
      <FormNavigationGuard when={isDirty} />

      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            isSaving={updateOption.isPending || isSubmitting}
          />
          <FormDirtyIndicator isDirty={isDirty} />
          <SettingsFormGrid>
            <FormField
              control={form.control}
              name='AffCommissionRate'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('Invitee Payment Commission Rate')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      step={0.01}
                      min={0}
                      max={1}
                      value={field.value ?? ''}
                      onChange={handleNumberChange(field.onChange)}
                      name={field.name}
                      onBlur={field.onBlur}
                      ref={field.ref}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Fraction of each approved-affiliate invitee payment credited as cash commission. 0.25 = 25%. Clamped to [0, 1].'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='MinWithdrawalCents'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('Minimum Withdrawal (USD cents)')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      step={100}
                      min={0}
                      value={field.value ?? ''}
                      onChange={handleNumberChange(field.onChange)}
                      name={field.name}
                      onBlur={field.onBlur}
                      ref={field.ref}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Minimum cash commission a user can withdraw in one request. {{amount}} USD.',
                      {
                        amount: Number.isFinite(minWithdrawalUSD)
                          ? minWithdrawalUSD.toFixed(2)
                          : '0.00',
                      }
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <SettingsFormGridItem span='full'>
              <FormField
                control={form.control}
                name='CommissionFreezeDays'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Commission Freeze Days')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        step={1}
                        min={0}
                        value={field.value ?? ''}
                        onChange={handleNumberChange(field.onChange)}
                        name={field.name}
                        onBlur={field.onBlur}
                        ref={field.ref}
                        className='max-w-xs'
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Newly earned commission enters Pending (frozen) state for this many days before becoming withdrawable. Set 0 to skip the freeze. Mitigates refund and chargeback risk.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGridItem>
          </SettingsFormGrid>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
