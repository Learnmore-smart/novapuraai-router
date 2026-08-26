import { useTranslation } from 'react-i18next'

import { StatusBadge, type StatusVariant } from '@/components/status-badge'

import { isDynamicPricingModel } from '../lib/dynamic-price'
import { isPriceUnset, isTokenBasedModel } from '../lib/model-helpers'
import type { PricingModel } from '../types'

interface ModelBillingModeBadgeProps {
  model: PricingModel
  className?: string
}

export function ModelBillingModeBadge(props: ModelBillingModeBadgeProps) {
  const { t } = useTranslation()
  let label = t('Per Request')
  let variant: StatusVariant = 'purple'

  if (isPriceUnset(props.model)) {
    label = t('Unset price')
    variant = 'neutral'
  } else if (isDynamicPricingModel(props.model)) {
    label = t('Dynamic Pricing')
    variant = 'warning'
  } else if (isTokenBasedModel(props.model)) {
    label = t('Token-based')
    variant = 'info'
  }

  return (
    <StatusBadge
      label={label}
      variant={variant}
      copyable={false}
      size='sm'
      className={props.className}
    />
  )
}
