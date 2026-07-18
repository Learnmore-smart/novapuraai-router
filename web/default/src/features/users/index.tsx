import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'

import { UsersDeleteDialog } from './components/users-delete-dialog'
import { ShareReviewDialog } from './components/share-review-dialog'
import { UsersMutateDrawer } from './components/users-mutate-drawer'
import { UsersPrimaryButtons } from './components/users-primary-buttons'
import { UsersProvider, useUsers } from './components/users-provider'
import { UsersTable } from './components/users-table'

function UsersContent() {
  const { t } = useTranslation()
  const { open, setOpen, currentRow } = useUsers()

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('Users')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <UsersPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <UsersTable />
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <UsersMutateDrawer
        open={open === 'create' || open === 'update'}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
        currentRow={open === 'update' ? currentRow || undefined : undefined}
      />
      <UsersDeleteDialog />
      <ShareReviewDialog
        open={open === 'share-review'}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
      />
    </>
  )
}

export function Users() {
  return (
    <UsersProvider>
      <UsersContent />
    </UsersProvider>
  )
}
