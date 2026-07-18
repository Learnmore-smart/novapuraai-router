import { KeyRound } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

type ChannelAuthSectionProps = {
  children: ReactNode
}

export function ChannelAuthSection(props: ChannelAuthSectionProps) {
  const { t } = useTranslation()

  return (
    <div className='border-border/60 flex flex-col gap-3 border-t pt-4'>
      <div className='flex items-center gap-2'>
        <KeyRound
          className='text-muted-foreground h-3.5 w-3.5'
          aria-hidden='true'
        />
        <h4 className='text-muted-foreground text-xs font-medium tracking-wide uppercase'>
          {t('Authentication')}
        </h4>
      </div>
      {props.children}
    </div>
  )
}
