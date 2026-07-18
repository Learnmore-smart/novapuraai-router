import { useQuery } from '@tanstack/react-query'
import { Braces, Code2, Eye, RotateCcw, Save } from 'lucide-react'
import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormLabel,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { getEnabledModels } from '@/features/channels/api'

import {
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import {
  getGlobalDiscountDraft,
  setGlobalDiscountDraft,
} from './model-global-discount'
import {
  getUnsetPricingModelNames,
  type BulkPricingMaps,
} from './model-pricing-bulk-json'
import { ModelPricingBulkJsonDialog } from './model-pricing-bulk-json-dialog'
import {
  ModelPricingFieldsJsonEditor,
  type ModelPricingFieldsJsonEditorHandle,
} from './model-pricing-fields-json-editor'
import {
  ModelPricingUnifiedJsonEditor,
  type ModelPricingUnifiedJsonEditorHandle,
} from './model-pricing-unified-json-editor'
import {
  ModelRatioVisualEditor,
  type ModelRatioVisualEditorHandle,
} from './model-ratio-visual-editor'

type ModelFormValues = {
  ModelPrice: string
  ModelRatio: string
  CacheRatio: string
  CreateCacheRatio: string
  CompletionRatio: string
  ImageRatio: string
  AudioRatio: string
  AudioCompletionRatio: string
  ModelDiscount: string
  ExposeRatioEnabled: boolean
  BillingMode: string
  BillingExpr: string
}

type ModelRatioFormProps = {
  form: UseFormReturn<ModelFormValues>
  savedValues: ModelFormValues
  onSave: (values: ModelFormValues) => Promise<void>
  onReset: () => void
  isSaving: boolean
  isResetting: boolean
  variant?: 'default' | 'unset'
}

export const ModelRatioForm = memo(function ModelRatioForm({
  form,
  savedValues,
  onSave,
  onReset,
  isSaving,
  isResetting,
  variant = 'default',
}: ModelRatioFormProps) {
  const { t } = useTranslation()
  const isUnsetVariant = variant === 'unset'
  const [editMode, setEditMode] = useState<'visual' | 'json'>('visual')
  const [jsonView, setJsonView] = useState<'fields' | 'unified'>('fields')
  const [bulkJsonOpen, setBulkJsonOpen] = useState(false)
  const [globalDiscountRate, setGlobalDiscountRate] = useState('0.8')
  const visualEditorRef = useRef<ModelRatioVisualEditorHandle>(null)
  const fieldsEditorRef = useRef<ModelPricingFieldsJsonEditorHandle>(null)
  const unifiedEditorRef = useRef<ModelPricingUnifiedJsonEditorHandle>(null)

  const enabledModelsQuery = useQuery({
    queryKey: ['enabled-models'],
    queryFn: getEnabledModels,
    enabled: true,
  })

  const enabledModelsError =
    enabledModelsQuery.isError ||
    (enabledModelsQuery.data !== undefined && !enabledModelsQuery.data.success)
  const enabledModelsErrorMessage = enabledModelsQuery.data?.message

  const modelPrice = form.watch('ModelPrice')
  const modelRatio = form.watch('ModelRatio')
  const cacheRatio = form.watch('CacheRatio')
  const createCacheRatio = form.watch('CreateCacheRatio')
  const completionRatio = form.watch('CompletionRatio')
  const imageRatio = form.watch('ImageRatio')
  const audioRatio = form.watch('AudioRatio')
  const audioCompletionRatio = form.watch('AudioCompletionRatio')
  const modelDiscount = form.watch('ModelDiscount')
  const billingMode = form.watch('BillingMode')
  const billingExpr = form.watch('BillingExpr')

  const currentMaps = useMemo<BulkPricingMaps>(
    () => ({
      modelPrice,
      modelRatio,
      cacheRatio,
      createCacheRatio,
      completionRatio,
      imageRatio,
      audioRatio,
      audioCompletionRatio,
      modelDiscount,
      billingMode,
      billingExpr,
    }),
    [
      audioCompletionRatio,
      audioRatio,
      billingExpr,
      billingMode,
      cacheRatio,
      completionRatio,
      createCacheRatio,
      imageRatio,
      modelDiscount,
      modelPrice,
      modelRatio,
    ]
  )

  const savedMaps = useMemo<BulkPricingMaps>(
    () => ({
      modelPrice: savedValues.ModelPrice,
      modelRatio: savedValues.ModelRatio,
      cacheRatio: savedValues.CacheRatio,
      createCacheRatio: savedValues.CreateCacheRatio,
      completionRatio: savedValues.CompletionRatio,
      imageRatio: savedValues.ImageRatio,
      audioRatio: savedValues.AudioRatio,
      audioCompletionRatio: savedValues.AudioCompletionRatio,
      modelDiscount: savedValues.ModelDiscount,
      billingMode: savedValues.BillingMode,
      billingExpr: savedValues.BillingExpr,
    }),
    [
      savedValues.AudioCompletionRatio,
      savedValues.AudioRatio,
      savedValues.BillingExpr,
      savedValues.BillingMode,
      savedValues.CacheRatio,
      savedValues.CompletionRatio,
      savedValues.CreateCacheRatio,
      savedValues.ImageRatio,
      savedValues.ModelDiscount,
      savedValues.ModelPrice,
      savedValues.ModelRatio,
    ]
  )

  const candidateModelNames = useMemo(
    () => enabledModelsQuery.data?.data || [],
    [enabledModelsQuery.data?.data]
  )
  const unifiedModelNames = useMemo(
    () =>
      isUnsetVariant
        ? getUnsetPricingModelNames(savedMaps, candidateModelNames)
        : candidateModelNames,
    [candidateModelNames, isUnsetVariant, savedMaps]
  )

  useEffect(() => {
    if (!enabledModelsError) return
    toast.error(enabledModelsErrorMessage || t('Failed to load enabled models'))
  }, [enabledModelsError, enabledModelsErrorMessage, t])

  const handleFieldChange = useCallback(
    (field: keyof ModelFormValues, value: string) => {
      form.setValue(field, value, {
        shouldValidate: true,
        shouldDirty: true,
      })
    },
    [form]
  )

  const modelDiscountDraft = modelDiscount
  const globalDiscount = getGlobalDiscountDraft(modelDiscountDraft)

  useEffect(() => {
    if (globalDiscount.rate !== null) {
      setGlobalDiscountRate(String(globalDiscount.rate))
    }
  }, [globalDiscount.rate])

  const updateGlobalDiscount = useCallback(
    (enabled: boolean, rateText: string) => {
      const rate = Number(rateText)
      try {
        handleFieldChange(
          'ModelDiscount',
          setGlobalDiscountDraft(form.getValues('ModelDiscount'), enabled, rate)
        )
      } catch {
        toast.error(
          t('Global discount rate must be greater than 0 and at most 1.')
        )
      }
    },
    [form, handleFieldChange, t]
  )

  const commitActiveJsonEditor = useCallback(() => {
    if (jsonView === 'unified') {
      return unifiedEditorRef.current?.commitDraft() === true
    }
    return fieldsEditorRef.current?.commitDraft() === true
  }, [jsonView])

  const toggleEditMode = useCallback(() => {
    if (editMode === 'json' && !commitActiveJsonEditor()) return
    setEditMode((prev) => (prev === 'visual' ? 'json' : 'visual'))
  }, [commitActiveJsonEditor, editMode])

  const handleJsonViewChange = useCallback(
    (nextView: 'fields' | 'unified') => {
      if (nextView === jsonView || !commitActiveJsonEditor()) return
      setJsonView(nextView)
    },
    [commitActiveJsonEditor, jsonView]
  )

  const handleSave = useCallback(async () => {
    if (editMode === 'visual') {
      const committed = await visualEditorRef.current?.commitOpenEditor()
      if (committed === false) return
    } else if (!commitActiveJsonEditor()) return

    await form.handleSubmit(onSave)()
  }, [commitActiveJsonEditor, editMode, form, onSave])

  const handleBulkApply = useCallback(
    (updates: Record<string, string>) => {
      const fieldMap: Record<string, keyof ModelFormValues> = {
        'billing_setting.billing_mode': 'BillingMode',
        'billing_setting.billing_expr': 'BillingExpr',
      }
      for (const [key, value] of Object.entries(updates)) {
        const formField = fieldMap[key] || (key as keyof ModelFormValues)
        handleFieldChange(formField, value)
      }
    },
    [handleFieldChange]
  )

  return (
    <div className='space-y-6'>
      <div className='flex flex-wrap justify-end gap-2'>
        {!isUnsetVariant && (
          <Button
            type='button'
            variant='destructive'
            size='sm'
            onClick={onReset}
            disabled={isResetting}
          >
            <RotateCcw data-icon='inline-start' />
            {t('Reset prices')}
          </Button>
        )}
        {editMode === 'json' && (
          <>
            <div
              role='group'
              aria-label={t('JSON editor layout')}
              className='bg-muted/30 inline-flex items-center rounded-lg border p-0.5'
            >
              <Button
                type='button'
                variant={jsonView === 'fields' ? 'secondary' : 'ghost'}
                size='sm'
                className='h-7 px-2.5 shadow-none'
                aria-pressed={jsonView === 'fields'}
                onClick={() => handleJsonViewChange('fields')}
              >
                {t('Field JSON')}
              </Button>
              <Button
                type='button'
                variant={jsonView === 'unified' ? 'secondary' : 'ghost'}
                size='sm'
                className='h-7 px-2.5 shadow-none'
                aria-pressed={jsonView === 'unified'}
                onClick={() => handleJsonViewChange('unified')}
              >
                {t('Unified JSON')}
              </Button>
            </div>
            <Button
              type='button'
              size='sm'
              onClick={handleSave}
              disabled={isSaving}
            >
              <Save data-icon='inline-start' />
              {isSaving ? t('Saving...') : t('Save model prices')}
            </Button>
          </>
        )}
        {!isUnsetVariant && (
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => setBulkJsonOpen(true)}
            disabled={enabledModelsQuery.isLoading || enabledModelsError}
          >
            <Braces data-icon='inline-start' />
            {t('Bulk edit (JSON)')}
          </Button>
        )}
        <Button variant='outline' size='sm' onClick={toggleEditMode}>
          {editMode === 'visual' ? (
            <>
              <Code2 className='mr-2 h-4 w-4' />
              {t('Switch to JSON')}
            </>
          ) : (
            <>
              <Eye className='mr-2 h-4 w-4' />
              {t('Switch to Visual')}
            </>
          )}
        </Button>
      </div>

      {!isUnsetVariant && (
        <SettingsSwitchItem>
          <SettingsSwitchContent>
            <div className='text-sm font-medium'>
              {t('Apply discount to all models')}
            </div>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Temporarily overrides every model discount without deleting individual rates. Turn it off to restore each model rate.'
              )}
            </p>
          </SettingsSwitchContent>
          <div className='flex items-center gap-3'>
            <Input
              className='w-24'
              inputMode='decimal'
              aria-label={t('Global discount rate')}
              value={globalDiscountRate}
              onChange={(event) => {
                const nextValue = event.target.value
                if (!/^\d*\.?\d*$/.test(nextValue)) return
                setGlobalDiscountRate(nextValue)
                const rate = Number(nextValue)
                if (
                  globalDiscount.enabled &&
                  nextValue !== '' &&
                  rate > 0 &&
                  rate <= 1
                ) {
                  updateGlobalDiscount(true, nextValue)
                }
              }}
              placeholder='0.8'
            />
            <Switch
              checked={globalDiscount.enabled}
              onCheckedChange={(checked) =>
                updateGlobalDiscount(checked, globalDiscountRate)
              }
              aria-label={t('Apply discount to all models')}
            />
          </div>
        </SettingsSwitchItem>
      )}

      <Form {...form}>
        {editMode === 'visual' ? (
          <div className='space-y-6'>
            <ModelRatioVisualEditor
              ref={visualEditorRef}
              savedModelPrice={savedValues.ModelPrice}
              savedModelRatio={savedValues.ModelRatio}
              savedCacheRatio={savedValues.CacheRatio}
              savedCreateCacheRatio={savedValues.CreateCacheRatio}
              savedCompletionRatio={savedValues.CompletionRatio}
              savedImageRatio={savedValues.ImageRatio}
              savedAudioRatio={savedValues.AudioRatio}
              savedAudioCompletionRatio={savedValues.AudioCompletionRatio}
              savedModelDiscount={savedValues.ModelDiscount}
              savedBillingMode={savedValues.BillingMode}
              savedBillingExpr={savedValues.BillingExpr}
              modelPrice={modelPrice}
              modelRatio={modelRatio}
              cacheRatio={cacheRatio}
              createCacheRatio={createCacheRatio}
              completionRatio={completionRatio}
              imageRatio={imageRatio}
              audioRatio={audioRatio}
              audioCompletionRatio={audioCompletionRatio}
              modelDiscount={modelDiscount}
              billingMode={billingMode}
              billingExpr={billingExpr}
              candidateModelNames={enabledModelsQuery.data?.data}
              candidateModelsLoading={enabledModelsQuery.isLoading}
              candidateModelsUnavailable={enabledModelsError}
              filterMode={isUnsetVariant ? 'unset' : 'all'}
              onSave={handleSave}
              isSaving={isSaving}
              onChange={(field, value) => {
                const fieldMap: Record<string, keyof ModelFormValues> = {
                  'billing_setting.billing_mode': 'BillingMode',
                  'billing_setting.billing_expr': 'BillingExpr',
                }
                const formField =
                  fieldMap[field] || (field as keyof ModelFormValues)
                handleFieldChange(formField, value)
              }}
            />

            {!isUnsetVariant && (
              <FormField
                control={form.control}
                name='ExposeRatioEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Expose ratio API')}</FormLabel>
                      <FormDescription>
                        {t(
                          'Allow clients to query configured ratios via `/api/ratio`.'
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />
            )}
          </div>
        ) : (
          <div className='space-y-6'>
            {jsonView === 'fields' ? (
              <ModelPricingFieldsJsonEditor
                ref={fieldsEditorRef}
                maps={currentMaps}
                modelNames={unifiedModelNames}
                candidateModelsOnly={isUnsetVariant}
                onApply={handleBulkApply}
              />
            ) : (
              <ModelPricingUnifiedJsonEditor
                ref={unifiedEditorRef}
                maps={currentMaps}
                modelNames={unifiedModelNames}
                candidateModelsOnly={isUnsetVariant}
                onApply={handleBulkApply}
              />
            )}

            {!isUnsetVariant && (
              <FormField
                control={form.control}
                name='ExposeRatioEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Expose ratio API')}</FormLabel>
                      <FormDescription>
                        {t(
                          'Allow clients to query configured ratios via `/api/ratio`.'
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />
            )}
          </div>
        )}
      </Form>

      <ModelPricingBulkJsonDialog
        open={bulkJsonOpen}
        onOpenChange={setBulkJsonOpen}
        maps={currentMaps}
        modelNames={candidateModelNames}
        onApply={handleBulkApply}
      />
    </div>
  )
})
