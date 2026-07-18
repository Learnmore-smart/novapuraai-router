import * as z from 'zod'

import type {
  AdminBillingCurrencyConfig,
  AdminTopupCampaign,
} from './topup-promotion-api'

const currencyDefinitionSchema = z.object({
  enabled: z.boolean(),
  fx: z.coerce.number().positive(),
})

export const topupPromotionFormSchema = z
  .object({
    defaultCurrency: z.enum(['cny', 'usd', 'cad']),
    autoUpdateFX: z.boolean(),
    currencies: z.object({
      cny: currencyDefinitionSchema,
      usd: currencyDefinitionSchema,
      cad: currencyDefinitionSchema,
    }),
    campaign: z.object({
      name: z.string().trim().min(1),
      enabled: z.boolean(),
      startAt: z.string(),
      endAt: z.string(),
      globalBudgetUSD: z.coerce.number().nonnegative(),
      perUserLimit: z.coerce.number().int().nonnegative(),
      defaultPromoExpiryDays: z.coerce.number().int().min(0).max(3650),
    }),
  })
  .refine((values) => values.currencies[values.defaultCurrency].enabled, {
    message: 'The default currency must be enabled',
    path: ['defaultCurrency'],
  })

export type TopupPromotionFormValues = z.infer<typeof topupPromotionFormSchema>

export function unixToLocalInput(timestamp: number): string {
  if (!timestamp) return ''
  const date = new Date(timestamp * 1000)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

export function localInputToUnix(value: string): number {
  if (!value) return 0
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? Math.floor(timestamp / 1000) : 0
}

export function createTopupPromotionFormValues(
  currencyConfig: AdminBillingCurrencyConfig,
  campaign: AdminTopupCampaign
): TopupPromotionFormValues {
  return {
    defaultCurrency: currencyConfig.default_currency,
    autoUpdateFX: currencyConfig.auto_update_fx,
    currencies: {
      cny: {
        enabled: currencyConfig.currencies.cny.enabled,
        fx: currencyConfig.currencies.cny.fx_presentment_per_usd,
      },
      usd: {
        enabled: currencyConfig.currencies.usd.enabled,
        fx: currencyConfig.currencies.usd.fx_presentment_per_usd,
      },
      cad: {
        enabled: currencyConfig.currencies.cad.enabled,
        fx: currencyConfig.currencies.cad.fx_presentment_per_usd,
      },
    },
    campaign: {
      name: campaign.name,
      enabled: campaign.enabled,
      startAt: unixToLocalInput(campaign.start_at),
      endAt: unixToLocalInput(campaign.end_at),
      globalBudgetUSD: campaign.global_budget_micro_usd / 1_000_000,
      perUserLimit: campaign.per_user_limit,
      defaultPromoExpiryDays: campaign.default_promo_expiry_days,
    },
  }
}
