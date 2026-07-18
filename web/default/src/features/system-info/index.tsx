import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'

import { SystemInstancesPanel } from './components/system-instances-panel'
import { SystemTasksPanel } from './components/system-tasks-panel'

export function SystemInfo() {
  const { t } = useTranslation()

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        <span className='inline-flex min-w-0 items-center gap-2'>
          <span className='truncate'>{t('System Info')}</span>
          <Badge variant='outline' className='shrink-0'>
            Root
          </Badge>
        </span>
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          <SystemInstancesPanel />
          <SystemTasksPanel />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
