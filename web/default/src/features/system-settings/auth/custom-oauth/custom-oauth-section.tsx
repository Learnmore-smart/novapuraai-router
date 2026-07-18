import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import { SettingsSection } from '../../components/settings-section'
import { buildOAuthCallbackUrl } from '../oauth-callback-url'
import { ProviderFormDialog } from './components/provider-form-dialog'
import { ProviderTable } from './components/provider-table'
import { useCustomOAuthProviders } from './hooks/use-custom-oauth-providers'
import type { CustomOAuthProvider } from './types'

type CustomOAuthSectionProps = {
  serverAddress: string
}

export function CustomOAuthSection(props: CustomOAuthSectionProps) {
  const { t } = useTranslation()
  const { data: providers = [], isLoading } = useCustomOAuthProviders()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingProvider, setEditingProvider] =
    useState<CustomOAuthProvider | null>(null)
  const callbackFormat = buildOAuthCallbackUrl(
    props.serverAddress,
    '{slug}',
    t('Site URL')
  )

  const handleCreate = () => {
    setEditingProvider(null)
    setDialogOpen(true)
  }

  const handleEdit = (provider: CustomOAuthProvider) => {
    setEditingProvider(provider)
    setDialogOpen(true)
  }

  const handleDialogChange = (open: boolean) => {
    setDialogOpen(open)
    if (!open) {
      setEditingProvider(null)
    }
  }

  if (isLoading) {
    return (
      <SettingsSection title={t('Custom OAuth Providers')}>
        <div className='text-muted-foreground py-8 text-center text-sm'>
          {t('Loading...')}
        </div>
      </SettingsSection>
    )
  }

  return (
    <SettingsSection title={t('Custom OAuth Providers')}>
      <Alert>
        <AlertTitle>{t('Callback URL format')}</AlertTitle>
        <AlertDescription className='space-y-3 text-sm'>
          <p>
            {t(
              'Use this callback URL pattern when registering a custom OAuth provider.'
            )}
          </p>
          <div className='flex min-w-0 flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between'>
            <span className='text-muted-foreground shrink-0'>
              {t('OAuth callback URL')}
            </span>
            <span className='flex min-w-0 items-center gap-2'>
              <code className='bg-muted text-foreground min-w-0 rounded px-1.5 py-0.5 text-xs break-all'>
                {callbackFormat}
              </code>
              <CopyButton
                value={callbackFormat}
                size='icon'
                className='size-7'
                tooltip={t('Copy callback URL')}
                aria-label={t('Copy callback URL')}
              />
            </span>
          </div>
        </AlertDescription>
      </Alert>

      <ProviderTable
        providers={providers}
        onEdit={handleEdit}
        onCreate={handleCreate}
      />

      <ProviderFormDialog
        open={dialogOpen}
        onOpenChange={handleDialogChange}
        provider={editingProvider}
        serverAddress={props.serverAddress}
      />
    </SettingsSection>
  )
}
