import { forwardRef, useEffect, useImperativeHandle, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { JsonCodeEditor } from '@/components/json-code-editor'
import { Alert, AlertDescription } from '@/components/ui/alert'

import type { BulkPricingMaps } from './model-pricing-bulk-json'
import {
  applyPricingFieldsJson,
  exportPricingFieldsJson,
  type PricingFieldName,
  type PricingFieldsDrafts,
} from './model-pricing-fields-json'

const fieldConfigs: Array<{
  name: PricingFieldName
  labelKey: string
  descriptionKey: string
}> = [
  {
    name: 'ModelPrice',
    labelKey: 'Model fixed pricing',
    descriptionKey:
      'JSON map of model → USD cost per request. Takes precedence over ratio based billing.',
  },
  {
    name: 'ModelRatio',
    labelKey: 'Model ratio',
    descriptionKey: 'JSON map of model → multiplier applied to quota billing.',
  },
  {
    name: 'CacheRatio',
    labelKey: 'Prompt cache ratio',
    descriptionKey: 'Optional ratio used when upstream cache hits occur.',
  },
  {
    name: 'CreateCacheRatio',
    labelKey: 'Create cache ratio',
    descriptionKey:
      'Ratio applied when creating cache entries for supported models.',
  },
  {
    name: 'CompletionRatio',
    labelKey: 'Completion ratio',
    descriptionKey:
      'Applies to custom completion endpoints. JSON map of model → ratio.',
  },
  {
    name: 'ImageRatio',
    labelKey: 'Image ratio',
    descriptionKey: 'Configure per-model ratio for image inputs or outputs.',
  },
  {
    name: 'AudioRatio',
    labelKey: 'Audio ratio',
    descriptionKey:
      'Ratio applied to audio inputs where supported by the upstream model.',
  },
  {
    name: 'AudioCompletionRatio',
    labelKey: 'Audio completion ratio',
    descriptionKey: 'Ratio applied to audio completions for streaming models.',
  },
  {
    name: 'ModelDiscount',
    labelKey: 'Model discount',
    descriptionKey:
      'JSON map of model → discount rate in (0, 1]. The billed price is the configured price multiplied by this rate.',
  },
]

type ModelPricingFieldsJsonEditorProps = {
  maps: BulkPricingMaps
  modelNames: string[]
  candidateModelsOnly?: boolean
  onApply: (updates: Record<string, string>) => void
}

export type ModelPricingFieldsJsonEditorHandle = {
  commitDraft: () => boolean
}

export const ModelPricingFieldsJsonEditor = forwardRef<
  ModelPricingFieldsJsonEditorHandle,
  ModelPricingFieldsJsonEditorProps
>(function ModelPricingFieldsJsonEditor(
  { maps, modelNames, candidateModelsOnly = false, onApply },
  ref
) {
  const { t } = useTranslation()
  const [drafts, setDrafts] = useState<PricingFieldsDrafts>(() =>
    exportPricingFieldsJson(maps, modelNames, candidateModelsOnly)
  )
  const [errors, setErrors] = useState<string[]>([])
  const [isDirty, setIsDirty] = useState(false)

  useEffect(() => {
    if (isDirty) return
    setDrafts(exportPricingFieldsJson(maps, modelNames, candidateModelsOnly))
  }, [candidateModelsOnly, isDirty, maps, modelNames])

  useImperativeHandle(
    ref,
    () => ({
      commitDraft: () => {
        const result = applyPricingFieldsJson(
          maps,
          drafts,
          modelNames,
          candidateModelsOnly
        )
        if (!result.ok) {
          setErrors(result.errors)
          return false
        }
        setErrors([])
        onApply(result.updates)
        setIsDirty(false)
        return true
      },
    }),
    [candidateModelsOnly, drafts, maps, modelNames, onApply]
  )

  return (
    <div className='space-y-6'>
      <div className='grid min-w-0 gap-x-5 gap-y-8 lg:grid-cols-2 2xl:grid-cols-3'>
        {fieldConfigs.map((config) => {
          const headingId = `pricing-json-${config.name}`
          return (
            <section
              key={config.name}
              aria-labelledby={headingId}
              className='flex min-w-0 flex-col gap-2'
            >
              <h3 id={headingId} className='text-sm font-medium'>
                {t(config.labelKey)}
              </h3>
              <JsonCodeEditor
                value={drafts[config.name]}
                onChange={(value) => {
                  setDrafts((current) => ({
                    ...current,
                    [config.name]: value,
                  }))
                  setIsDirty(true)
                  if (errors.length > 0) setErrors([])
                }}
              />
              <p className='text-muted-foreground text-xs leading-5'>
                {t(config.descriptionKey)}
              </p>
            </section>
          )
        })}
      </div>

      {errors.length > 0 && (
        <Alert variant='destructive'>
          <AlertDescription>
            <div className='flex max-h-40 flex-col gap-1 overflow-y-auto'>
              {errors.map((error) => (
                <span key={error}>{error}</span>
              ))}
            </div>
          </AlertDescription>
        </Alert>
      )}
    </div>
  )
})
