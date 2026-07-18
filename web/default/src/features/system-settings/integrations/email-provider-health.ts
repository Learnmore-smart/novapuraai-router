import type { EmailProviderHealth, TransactionalEmailProvider } from '../types'

export type EmailProviderSwitchState = 'selected' | 'available' | 'unavailable'

export function getEmailProviderSwitchState(
  provider: EmailProviderHealth,
  selectedProvider: TransactionalEmailProvider
): EmailProviderSwitchState {
  if (provider.provider === selectedProvider) return 'selected'
  if (provider.configured) return 'available'
  return 'unavailable'
}

export function isValidTestEmailRecipient(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim())
}
