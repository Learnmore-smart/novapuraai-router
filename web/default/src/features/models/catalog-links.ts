export const MODEL_CATALOG_PATH = '/models/catalog'
export const MODEL_METADATA_PATH = '/models/metadata'
export const MODEL_DEPLOYMENTS_PATH = '/models/deployments'
export const MODEL_PRICING_EDITOR_PATH =
  '/system-settings/billing/model-pricing'

export type ModelSection = 'catalog' | 'deployments'

export function getModelSectionPath(section: ModelSection) {
  return `/models/${section}`
}

export function getModelProfilePath(modelName: string) {
  return `/pricing/${encodeURIComponent(modelName)}`
}
