import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

// Maps a raw subscription status to a colored tone. The backend's
// display_status string is the i18n key, but the *tone* (success/warning/etc.)
// is derived from the raw status so we don't depend on string-matching the
// localized label.
type StatusTone = 'success' | 'warning' | 'muted' | 'danger'

const toneByStatus: Record<string, StatusTone> = {
  active: 'success',
  prepaid_active: 'success',
  canceling: 'warning',
  canceled: 'muted',
  expired: 'muted',
  past_due: 'danger',
  payment_failed: 'danger',
}

const toneClasses: Record<StatusTone, string> = {
  success: 'bg-success/10 text-success',
  warning: 'bg-warning/10 text-warning',
  muted: 'bg-muted text-muted-foreground',
  danger: 'bg-destructive/10 text-destructive',
}

interface SubscriptionStatusBadgeProps {
  /** Raw status value from the subscription record (e.g. "active"). */
  status: string
  /** Display label from the backend (already an i18n key, e.g. "Active"). */
  displayStatus?: string
  className?: string
}

export function SubscriptionStatusBadge(
  props: SubscriptionStatusBadgeProps
) {
  const { t } = useTranslation()
  const tone = toneByStatus[props.status] ?? 'muted'
  // display_status is the English key from the backend; fall back to the raw
  // status so we always show something.
  const label = props.displayStatus ? t(props.displayStatus) : props.status

  return (
    <span
      className={cn(
        'inline-flex h-5 w-fit items-center gap-1 rounded-4xl px-2 text-xs font-medium whitespace-nowrap',
        toneClasses[tone],
        props.className
      )}
    >
      {label}
    </span>
  )
}
