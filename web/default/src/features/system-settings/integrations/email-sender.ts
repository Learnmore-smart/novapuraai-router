const BARE_EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const DISPLAY_EMAIL_PATTERN = /^(?:"(?:[^"\\]|\\.)*"|[^"<>]+)\s*<([^<>]+)>$/

export function isValidEmailSender(value: string): boolean {
  const trimmed = value.trim()
  if (!trimmed) return true
  if (BARE_EMAIL_PATTERN.test(trimmed)) return true
  const displayMatch = DISPLAY_EMAIL_PATTERN.exec(trimmed)
  return Boolean(
    displayMatch?.[1] && BARE_EMAIL_PATTERN.test(displayMatch[1].trim())
  )
}
