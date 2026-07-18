export type StripeCredentialFormValues = {
  secretKey: string
  publishableKey: string
  webhookSecret: string
}

export type StripeCredentialUpdatePayload = {
  secret_key?: string
  publishable_key?: string
  webhook_secret?: string
}

export function buildStripeCredentialUpdate(
  values: StripeCredentialFormValues
): StripeCredentialUpdatePayload {
  const payload: StripeCredentialUpdatePayload = {}
  const secretKey = values.secretKey.trim()
  const publishableKey = values.publishableKey.trim()
  const webhookSecret = values.webhookSecret.trim()
  if (secretKey) payload.secret_key = secretKey
  if (publishableKey) payload.publishable_key = publishableKey
  if (webhookSecret) payload.webhook_secret = webhookSecret
  return payload
}
