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
/**
 * Home page constants for optional CMS-driven feature/stat overrides.
 */
import type { TFunction } from 'i18next'

export const MAIN_BASE_CLASSES = 'bg-background text-foreground w-full'

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
