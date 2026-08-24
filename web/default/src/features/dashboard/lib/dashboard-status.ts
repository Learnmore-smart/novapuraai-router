export interface SuccessfulRequestStatus {
  has_successful_request?: unknown
}

/** Only the authenticated user's explicit server signal can open the cockpit. */
export function hasConfirmedSuccessfulRequest(value: unknown): boolean {
  if (typeof value === 'boolean') return value
  if (!value || typeof value !== 'object') return false

  const signal = (value as SuccessfulRequestStatus).has_successful_request
  return signal === true
}
