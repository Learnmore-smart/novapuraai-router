export function formatLifecycleDate(
  value?: number | string | null
): string | null {
  if (value === null || value === undefined || value === '') return null
  if (typeof value === 'number' && value <= 0) return null

  const date = new Date(
    typeof value === 'number' && value < 10_000_000_000 ? value * 1000 : value
  )
  if (Number.isNaN(date.getTime())) return null

  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(
    date
  )
}
