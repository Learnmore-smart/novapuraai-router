/**
 * Home page constants for optional CMS-driven feature/stat overrides.
 */
import type { TFunction } from 'i18next'

export const MAIN_BASE_CLASSES = 'bg-background text-foreground w-full'

export const LANDING_MODEL_ROWS = [
  { name: 'GLM 5.2', note: 'Pay per token' },
  { name: 'DeepSeek V4 Pro', note: 'Pay per token' },
  { name: 'Kimi K2.6', note: 'Pay per token' },
  { name: 'Nemotron 3 Ultra', note: 'Pay per token' },
] as const

/** Used by legacy GatewayCard if still mounted somewhere. */
export const GATEWAY_FEATURES = [
  'Compatible API',
  'Channel Pools',
  'Usage Ledger',
  'Prepaid Balance',
  'Promotional Credit',
  'Model Pricing',
  'Health-aware Routing',
  'API Key Controls',
] as const

export function getGatewayFeatures(t: TFunction) {
  return GATEWAY_FEATURES.map((feature) => t(feature))
}

export const DEFAULT_STATS = [
  {
    value: '1',
    suffix: '',
    description: 'Compatible endpoint',
  },
  {
    value: '2',
    suffix: '',
    description: 'Separate balances',
  },
  {
    value: '1',
    suffix: '',
    description: 'Request ledger',
  },
  {
    value: '0',
    suffix: '',
    description: 'Subscription required',
  },
] as const

export const DEFAULT_FEATURES = [
  {
    title: 'Drop-in compatibility',
    description:
      'Use a familiar base URL, bearer token, and chat-completions route. Keep your existing clients and SDKs.',
    iconName: 'Braces',
  },
  {
    title: 'Resilient model routing',
    description:
      'Point at one model pool. NovaPura selects a healthy channel and surfaces failures when a route cannot complete.',
    iconName: 'Route',
  },
  {
    title: 'Prepaid cost control',
    description:
      'Spend from balances you control. Promotional credit applies first; cash covers the rest. No subscription maze.',
    iconName: 'CircleDollarSign',
  },
] as const

export function getDefaultStats(t: TFunction) {
  return DEFAULT_STATS.map((stat) => ({
    ...stat,
    description: stat.description ? t(stat.description) : undefined,
  }))
}

export function getDefaultFeatures(t: TFunction) {
  return DEFAULT_FEATURES.map((feature) => ({
    ...feature,
    title: t(feature.title),
    description: t(feature.description),
  }))
}
