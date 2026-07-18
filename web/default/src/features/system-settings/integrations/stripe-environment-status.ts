export type StripeCredentialField =
  | 'secret'
  | 'publishable'
  | 'webhook'
  | 'account'
  | 'product'

export type StripeEnvironmentConfiguration = {
  secretConfigured: boolean
  publishableConfigured: boolean
  webhookConfigured: boolean
  accountID: string
  productID: string
}

export function getStripeEnvironmentReadiness(
  configuration: StripeEnvironmentConfiguration
) {
  const missing: StripeCredentialField[] = []
  if (!configuration.secretConfigured) missing.push('secret')
  if (!configuration.publishableConfigured) missing.push('publishable')
  if (!configuration.webhookConfigured) missing.push('webhook')
  if (!configuration.accountID.trim()) missing.push('account')
  if (!configuration.productID.trim()) missing.push('product')

  return { ready: missing.length === 0, missing }
}
