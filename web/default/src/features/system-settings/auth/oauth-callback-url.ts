export function resolveOAuthSiteUrl(
  serverAddress: string,
  fallback: string
): string {
  const normalized = serverAddress.trim().replace(/\/+$/, '')
  return normalized || fallback
}

export function buildOAuthCallbackUrl(
  serverAddress: string,
  callbackPath: string,
  fallback: string
): string {
  const siteUrl = resolveOAuthSiteUrl(serverAddress, fallback)
  return `${siteUrl}/oauth/${callbackPath.replace(/^\/+/, '')}`
}
