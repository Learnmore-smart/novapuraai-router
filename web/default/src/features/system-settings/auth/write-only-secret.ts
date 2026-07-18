export const WRITE_ONLY_SECRET_MASK = '••••••••••••••••'

export function getWriteOnlySecretPlaceholder(
  configured: boolean,
  emptyPrompt: string
): string {
  return configured ? WRITE_ONLY_SECRET_MASK : emptyPrompt
}

export function hasWriteOnlySecretReplacement(value: string): boolean {
  return value.trim() !== ''
}

export function orderAuthOptionUpdates<T>(
  updates: Array<[string, T]>,
  enabledKey: string,
  enabling: boolean
): Array<[string, T]> {
  const enabledUpdate = updates.find(([key]) => key === enabledKey)
  if (!enabledUpdate) return updates

  const remainingUpdates = updates.filter(([key]) => key !== enabledKey)
  return enabling
    ? [...remainingUpdates, enabledUpdate]
    : [enabledUpdate, ...remainingUpdates]
}
