export type SignUpTurnstileAction = 'email_verification' | 'register'

export function getSignUpTurnstileAction(
  emailVerificationRequired: boolean,
  emailVerificationSent: boolean
): SignUpTurnstileAction {
  return emailVerificationRequired && !emailVerificationSent
    ? 'email_verification'
    : 'register'
}
