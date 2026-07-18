import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { StatusBadge, type StatusBadgeProps } from './status-badge'

type ProviderBadgeProps = Omit<StatusBadgeProps, 'children' | 'label'> & {
  iconKey?: string | null
  iconSize?: number
  label: string
  /** Color the label text by provider name. Set false for a neutral label. */
  colorText?: boolean
}

export function ProviderBadge({
  className,
  iconKey,
  iconSize = 14,
  label,
  colorText = true,
  ...badgeProps
}: ProviderBadgeProps) {
  const icon = iconKey ? getLobeIcon(iconKey, iconSize) : null

  return (
    <div
      data-slot='provider-badge'
      className={cn('flex max-w-full min-w-0 items-center gap-1.5', className)}
    >
      {icon && <span className='flex shrink-0 items-center'>{icon}</span>}
      <StatusBadge
        label={label}
        autoColor={colorText ? label : undefined}
        variant={colorText ? undefined : 'neutral'}
        size='sm'
        className={cn('min-w-0 shrink overflow-hidden', !icon && 'pl-0')}
        {...badgeProps}
      />
    </div>
  )
}
