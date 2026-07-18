import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { JsonCodeEditor } from '@/components/json-code-editor'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import {
  applyPricingJson,
  exportPricingJson,
  type BulkPricingMaps,
} from './model-pricing-bulk-json'

type ModelPricingBulkJsonDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  maps: BulkPricingMaps
  modelNames: string[]
  onApply: (updates: Record<string, string>) => void
}

export function ModelPricingBulkJsonDialog({
  open,
  onOpenChange,
  maps,
  modelNames,
  onApply,
}: ModelPricingBulkJsonDialogProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState('')
  const [errors, setErrors] = useState<string[]>([])

  useEffect(() => {
    if (open) {
      setDraft(exportPricingJson(maps, modelNames))
      setErrors([])
    }
    // Re-export only when the dialog opens; edits must not be clobbered.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const handleApply = () => {
    const result = applyPricingJson(maps, draft)
    if (!result.ok) {
      setErrors(result.errors)
      return
    }
    setErrors([])
    onApply(result.updates)
    onOpenChange(false)
    let message = t('Updated pricing for {{count}} models', {
      count: result.applied,
    })
    if (result.removed > 0) {
      message += ` · ${t('{{count}} removed', { count: result.removed })}`
    }
    toast.success(message)
    if (result.skippedTiered.length > 0) {
      toast.info(
        t('Skipped expression-billed models: {{models}}', {
          models: result.skippedTiered.join(', '),
        })
      )
    }
    toast.info(t('Review the changes, then save to persist them.'))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>{t('Bulk edit pricing as JSON')}</DialogTitle>
          <DialogDescription>
            {t(
              'One entry per model with USD prices per 1M tokens: input, output, cache_read, cache_write, image_input, audio_input, audio_output, discount (0–1), or per_request for fixed pricing. Set a model to null to remove it; models not listed stay unchanged.'
            )}
          </DialogDescription>
        </DialogHeader>

        <JsonCodeEditor
          value={draft}
          onChange={setDraft}
          heightClassName='h-96 min-h-96 max-h-96'
        />

        {errors.length > 0 && (
          <Alert variant='destructive'>
            <AlertDescription>
              <div className='flex max-h-32 flex-col gap-1 overflow-y-auto'>
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

        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleApply}>{t('Apply to draft')}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
