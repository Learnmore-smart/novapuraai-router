import i18next from 'i18next'
import { useCallback, useState } from 'react'
import { toast } from 'sonner'

import { useStatus } from '@/hooks/use-status'

/**
 * Hook for managing Turnstile verification
 */
export function useTurnstile() {
  const { status } = useStatus()
  const [turnstileToken, setTurnstileToken] = useState('')
  const [turnstileResetKey, setTurnstileResetKey] = useState(0)

  const isTurnstileEnabled = !!(
    status?.turnstile_check && status?.turnstile_site_key
  )
  const turnstileSiteKey = status?.turnstile_site_key || ''

  /**
   * Validate if turnstile is ready when required
   */
  const validateTurnstile = (): boolean => {
    if (isTurnstileEnabled && !turnstileToken) {
      toast.info(
        i18next.t('Please wait a moment, human check is initializing...')
      )
      return false
    }
    return true
  }

  const resetTurnstile = useCallback(() => {
    setTurnstileToken('')
    setTurnstileResetKey((current) => current + 1)
  }, [])

  return {
    isTurnstileEnabled,
    turnstileSiteKey,
    turnstileToken,
    turnstileResetKey,
    setTurnstileToken,
    resetTurnstile,
    validateTurnstile,
  }
}
