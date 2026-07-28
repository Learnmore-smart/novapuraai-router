import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { handleServerError } from '@/lib/handle-server-error'

import { createSubscriptionPortal } from '../api'

interface PortalButtonProps {
  /** Already-translated button label. */
  label: string
  variant?: 'primary' | 'outline'
  className?: string
}

// Reusable button that opens the Stripe Customer Portal. The portal handles
// subscription management (cancel / reactivate), invoice viewing, and payment
// method updates — the backend just mints a portal session URL.
export function PortalButton(props: PortalButtonProps) {
  const { t } = useTranslation()

  const portalMutation = useMutation({
    mutationFn: createSubscriptionPortal,
    onSuccess: (res) => {
      const url = res.data?.url
      if (!url) {
        handleServerError(new Error(t('Something went wrong!')))
        return
      }
      // Full-page redirect to Stripe's hosted portal.
      window.location.href = url
    },
    onError: (error: unknown) => {
      handleServerError(error)
    },
  })

  return (
    <Button
      variant={props.variant === 'outline' ? 'outline' : 'default'}
      className={props.className}
      onClick={() => portalMutation.mutate()}
      disabled={portalMutation.isPending}
    >
      {portalMutation.isPending ? t('Loading...') : props.label}
    </Button>
  )
}
