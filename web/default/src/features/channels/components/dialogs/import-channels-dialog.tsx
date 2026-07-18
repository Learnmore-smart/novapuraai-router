import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { importChannels } from '../../api'
import { channelsQueryKeys } from '../../lib'

type ImportChannelsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const SAMPLE = `name,type,key,models,base_url,group,weight,priority,remark
provider-1,1,sk-example-key-1,gpt-4o-mini,https://api.example.com,default,1,10,batch-a
provider-2,1,sk-example-key-2,gpt-4o-mini,https://api.example.com,default,1,10,batch-a`

export function ImportChannelsDialog({
  open,
  onOpenChange,
}: ImportChannelsDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [csv, setCsv] = useState(SAMPLE)
  const [loading, setLoading] = useState(false)
  const [resultText, setResultText] = useState('')

  const onSubmit = async () => {
    setLoading(true)
    setResultText('')
    try {
      const res = await importChannels({ csv, default_type: 1 })
      if (!res.success) {
        toast.error(res.message || t('Import failed'))
        return
      }
      const d = res.data
      const summary = t(
        'Imported {{success}}, duplicates {{duplicate}}, failed {{failed}}',
        {
          success: d?.success ?? 0,
          duplicate: d?.duplicate ?? 0,
          failed: d?.failed ?? 0,
        }
      )
      toast.success(summary)
      setResultText([summary, ...(d?.errors ?? []).slice(0, 20)].join('\n'))
      await queryClient.invalidateQueries({ queryKey: channelsQueryKeys.all })
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('Import failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Import Channels (CSV)')}
      description={t(
        'Paste CSV with header: name,type,key,models,base_url,group,weight,priority,remark. Keys are stored encrypted and never logged in full.'
      )}
      contentHeight='auto'
      bodyClassName='space-y-3'
      footer={
        <>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Close')}
          </Button>
          <Button onClick={onSubmit} disabled={loading || !csv.trim()}>
            {loading && <Loader2 className='mr-2 size-4 animate-spin' />}
            {t('Import')}
          </Button>
        </>
      }
    >
      <div className='space-y-3 py-2'>
        <div className='space-y-1.5'>
          <Label htmlFor='channel-import-csv'>{t('CSV content')}</Label>
          <Textarea
            id='channel-import-csv'
            value={csv}
            onChange={(e) => setCsv(e.target.value)}
            className='min-h-48 font-mono text-xs'
          />
        </div>
        {resultText ? (
          <pre className='bg-muted max-h-40 overflow-auto rounded-md p-3 text-xs whitespace-pre-wrap'>
            {resultText}
          </pre>
        ) : null}
      </div>
    </Dialog>
  )
}
