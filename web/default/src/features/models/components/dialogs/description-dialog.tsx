import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'

type DescriptionDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  modelName: string
  description: string
}

export function DescriptionDialog({
  open,
  onOpenChange,
  modelName,
  description,
}: DescriptionDialogProps) {
  const { t } = useTranslation()
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={modelName}
      description={t('Model Description')}
      contentClassName='max-w-2xl'
      contentHeight='auto'
      bodyClassName='space-y-4'
    >
      <ScrollArea className='max-h-96'>
        <div className='space-y-2 pr-4'>
          <p className='text-foreground text-sm leading-relaxed break-words whitespace-pre-wrap'>
            {description}
          </p>
        </div>
      </ScrollArea>
    </Dialog>
  )
}
