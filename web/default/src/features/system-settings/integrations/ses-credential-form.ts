export type SESCredentialFormValues = {
  accessKeyId: string
  secretAccessKey: string
  sessionToken: string
  clearSessionToken: boolean
}

export type SESCredentialUpdatePayload = {
  access_key_id?: string
  secret_access_key?: string
  session_token?: string
  clear_session_token?: boolean
}

export function buildSESCredentialUpdate(
  values: SESCredentialFormValues
): SESCredentialUpdatePayload {
  const payload: SESCredentialUpdatePayload = {}
  const accessKeyId = values.accessKeyId.trim()
  if (accessKeyId) payload.access_key_id = accessKeyId
  if (values.secretAccessKey) {
    payload.secret_access_key = values.secretAccessKey
  }
  if (values.sessionToken) {
    payload.session_token = values.sessionToken
  } else if (values.clearSessionToken) {
    payload.clear_session_token = true
  }
  return payload
}
