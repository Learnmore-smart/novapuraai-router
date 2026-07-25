import { useCallback, useEffect, useState } from 'react'

import { api } from '@/lib/api'

// ============================================================================
// Stripe Connect Onboarding Types
// ============================================================================

export type StripeConnectOnboardingState =
  | 'created'
  | 'onboarding'
  | 'enabled'
  | 'restricted'
  | 'rejected'

export interface StripeConnectAccount {
  id: number
  user_id: number
  stripe_account_id: string
  email: string
  country: string
  payouts_enabled: boolean
  details_submitted: boolean
  payout_schedule_interval: string
  /** JSON-encoded string array of currently-due Stripe requirements. */
  currently_due: string
  eventually_due: string
  onboarding_state: StripeConnectOnboardingState
  created_at: number
  updated_at: number
}

export interface StripeConnectStatus {
  started: boolean
  account?: StripeConnectAccount
}

// ============================================================================
// Hook
// ============================================================================

/**
 * Manages the user's Stripe Connect onboarding state.
 *
 * `enabled` should be the user's `commission_approved` flag — the backend
 * rejects onboarding otherwise, so we skip the status query entirely when the
 * user is not an approved commission member.
 */
export function useStripeConnect(enabled: boolean) {
  const [status, setStatus] = useState<StripeConnectStatus | null>(null)
  const [loading, setLoading] = useState(false)
  const [starting, setStarting] = useState(false)

  const fetchStatus = useCallback(async () => {
    if (!enabled) {
      setStatus(null)
      return
    }
    try {
      setLoading(true)
      const res = await api.get('/api/user/stripe_connect/status')
      setStatus((res.data?.data ?? null) as StripeConnectStatus | null)
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch stripe connect status:', error)
    } finally {
      setLoading(false)
    }
  }, [enabled])

  const start = useCallback(async () => {
    if (!enabled) return
    try {
      setStarting(true)
      const res = await api.post('/api/user/stripe_connect/start')
      const url = res.data?.data?.url
      if (typeof url === 'string' && url) {
        window.location.assign(url)
      }
    } catch (error) {
      // axios interceptor surfaces business/HTTP errors as toasts.
      // eslint-disable-next-line no-console
      console.error('Failed to start stripe connect onboarding:', error)
    } finally {
      setStarting(false)
    }
  }, [enabled])

  useEffect(() => {
    fetchStatus()
  }, [fetchStatus])

  return {
    status,
    loading,
    starting,
    start,
    refresh: fetchStatus,
  }
}
