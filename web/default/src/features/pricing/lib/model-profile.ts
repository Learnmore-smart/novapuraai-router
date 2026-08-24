export function getModelProfilePath(modelName: string): string {
  return `/pricing/${encodeURIComponent(modelName)}`
}
