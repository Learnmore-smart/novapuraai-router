export function buildQueryParams(
  params: Record<string, unknown>
): URLSearchParams {
  const queryParams = new URLSearchParams()

  Object.entries(params).forEach(([key, value]) => {
    // Keep 0 as a valid value, only filter out undefined, null, and empty string
    if (value !== undefined && value !== null && value !== '') {
      queryParams.append(key, String(value))
    }
  })

  return queryParams
}
