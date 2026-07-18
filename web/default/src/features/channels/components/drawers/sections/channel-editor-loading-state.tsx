import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'

export function ChannelEditorLoadingState() {
  const { t } = useTranslation()

  return (
    <div
      className='border-border/60 flex flex-col gap-4 rounded-lg border p-4'
      aria-live='polite'
    >
      <div>
        <p className='text-sm font-medium'>{t('Loading channel details')}</p>
        <p className='text-muted-foreground mt-1 text-xs'>
          {t('Please wait before editing to avoid overwriting saved values.')}
        </p>
      </div>
      <div className='grid gap-4 sm:grid-cols-2'>
        <Skeleton className='h-10 w-full' />
        <Skeleton className='h-10 w-full' />
      </div>
      <Skeleton className='h-24 w-full' />
      <Skeleton className='h-32 w-full' />
    </div>
  )
}
