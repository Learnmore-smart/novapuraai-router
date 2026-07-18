import { KeyRound } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import {
  SideDrawerSection,
  SideDrawerSectionHeader,
} from '@/components/drawer-layout'

type ChannelApiAccessSectionProps = {
  children: ReactNode
}

export function ChannelApiAccessSection(props: ChannelApiAccessSectionProps) {
  const { t } = useTranslation()

  return (
    <SideDrawerSection>
      <SideDrawerSectionHeader
        title={t('Credentials')}
        description={t('Authentication')}
        icon={<KeyRound className='h-4 w-4' aria-hidden='true' />}
        iconTone='success'
      />
      {props.children}
    </SideDrawerSection>
  )
}
