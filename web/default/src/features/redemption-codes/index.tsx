import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'

import { RedemptionsDialogs } from './components/redemptions-dialogs'
import { RedemptionsPrimaryButtons } from './components/redemptions-primary-buttons'
import { RedemptionsProvider } from './components/redemptions-provider'
import { RedemptionsTable } from './components/redemptions-table'

export function Redemptions() {
  const { t } = useTranslation()
  return (
    <RedemptionsProvider>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>
          {t('Redemption Codes')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <RedemptionsPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <RedemptionsTable />
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <RedemptionsDialogs />
    </RedemptionsProvider>
  )
}
