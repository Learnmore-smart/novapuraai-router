/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { forwardRef, useEffect, useImperativeHandle, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { JsonCodeEditor } from '@/components/json-code-editor'
import { Alert, AlertDescription } from '@/components/ui/alert'

import {
  applyPricingJson,
  exportPricingJson,
  type BulkPricingMaps,
} from './model-pricing-bulk-json'

type ModelPricingUnifiedJsonEditorProps = {
  maps: BulkPricingMaps
  modelNames: string[]
  candidateModelsOnly?: boolean
  onApply: (updates: Record<string, string>) => void
}

export type ModelPricingUnifiedJsonEditorHandle = {
  commitDraft: () => boolean
}

export const ModelPricingUnifiedJsonEditor = forwardRef<
  ModelPricingUnifiedJsonEditorHandle,
  ModelPricingUnifiedJsonEditorProps
>(function ModelPricingUnifiedJsonEditor(props, ref) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState(() =>
    exportPricingJson(props.maps, props.modelNames, {
      candidateModelsOnly: props.candidateModelsOnly,
    })
  )
  const [errors, setErrors] = useState<string[]>([])
  const [isDirty, setIsDirty] = useState(false)

  useEffect(() => {
    if (isDirty) return
    setDraft(
      exportPricingJson(props.maps, props.modelNames, {
        candidateModelsOnly: props.candidateModelsOnly,
      })
    )
  }, [isDirty, props.candidateModelsOnly, props.maps, props.modelNames])

  useImperativeHandle(
    ref,
    () => ({
      commitDraft: () => {
        const result = applyPricingJson(props.maps, draft)
        if (!result.ok) {
          setErrors(result.errors)
          return false
        }
        setErrors([])
        props.onApply(result.updates)
        setIsDirty(false)
        return true
      },
    }),
    [draft, props]
  )

  return (
    <section className='space-y-4'>
      <div className='space-y-1.5'>
        <h3 className='text-sm font-medium'>{t('Unified JSON')}</h3>
        <p className='text-muted-foreground text-sm leading-6'>
          {t(
            'One entry per model with USD prices per 1M tokens: input, output, cache_read, cache_write, image_input, audio_input, audio_output, discount (0–1), or per_request for fixed pricing. Set a model to null to remove it; models not listed stay unchanged.'
          )}
        </p>
      </div>

      <JsonCodeEditor
        value={draft}
        onChange={(value) => {
          setDraft(value)
          setIsDirty(true)
          if (errors.length > 0) setErrors([])
        }}
        heightClassName='h-[clamp(32rem,65vh,52rem)] min-h-[32rem] max-h-[52rem]'
      />

      {errors.length > 0 && (
        <Alert variant='destructive'>
          <AlertDescription>
            <div className='flex max-h-40 flex-col gap-1 overflow-y-auto'>
              {errors.slice(0, 20).map((error) => (
                <span key={error}>{error}</span>
              ))}
              {errors.length > 20 && (
                <span>
                  {t('…and {{count}} more errors', {
                    count: errors.length - 20,
                  })}
                </span>
              )}
            </div>
          </AlertDescription>
        </Alert>
      )}
    </section>
  )
})
