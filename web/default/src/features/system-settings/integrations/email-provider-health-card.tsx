import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CheckCircle2,
  CircleAlert,
  Loader2,
  RefreshCw,
  ShieldCheck,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import {
  getTransactionalEmailHealth,
  retryTransactionalEmailQueue,
  sendTransactionalEmailTest,
  switchTransactionalEmailProvider,
} from '../api'
import type {
  EmailProviderHealth,
  TransactionalEmailProvider,
} from '../types'
import {
  getEmailProviderSwitchState,
  isValidTestEmailRecipient,
} from './email-provider-health'
import { SESCredentialPanel } from './ses-credential-panel'

const PROVIDER_ORDER: TransactionalEmailProvider[] = ['ses']

function yesNo(value: boolean | undefined, t: (key: string) => string) {
  if (value === true) return t('Yes')
  if (value === false) return t('No')
  return t('Unknown')
}

export function EmailProviderHealthCard() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [testRecipient, setTestRecipient] = useState('noahzh52@gmail.com')
  const healthQuery = useQuery({
    queryKey: ['transactional-email-health'],
    queryFn: getTransactionalEmailHealth,
    refetchInterval: 60_000,
  })

  const switchMutation = useMutation({
    mutationFn: switchTransactionalEmailProvider,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message)
        return
      }
      toast.success(t('Transactional email provider switched'))
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['transactional-email-health'],
        }),
        queryClient.invalidateQueries({ queryKey: ['system-options'] }),
      ])
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to switch email provider'))
    },
  })

  const retryMutation = useMutation({
    mutationFn: retryTransactionalEmailQueue,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message)
        return
      }
      toast.success(
        t(
          'Safe retry complete: {{sent}} sent, {{queued}} queued, {{failed}} failed',
          {
            sent: response.data.sent,
            queued: response.data.queued,
            failed: response.data.failed,
          }
        )
      )
      await queryClient.invalidateQueries({
        queryKey: ['transactional-email-health'],
      })
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to retry queued emails'))
    },
  })

  const testEmailMutation = useMutation({
    mutationFn: sendTransactionalEmailTest,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message)
        return
      }
      toast.success(t('Test email sent'))
      await queryClient.invalidateQueries({
        queryKey: ['transactional-email-health'],
      })
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to send test email'))
    },
  })

  if (healthQuery.isPending) {
    return (
      <div
        className='border-border bg-muted/20 rounded-xl border p-4'
        aria-live='polite'
      >
        <div className='flex items-center gap-2 text-sm font-medium'>
          <Loader2 className='size-4 animate-spin' aria-hidden='true' />
          {t('Checking transactional email health...')}
        </div>
        <div className='mt-4 grid gap-3 md:grid-cols-2'>
          <Skeleton className='h-28' />
          <Skeleton className='h-28' />
        </div>
      </div>
    )
  }

  if (healthQuery.isError || !healthQuery.data?.success) {
    return (
      <div className='border-destructive/30 bg-destructive/5 rounded-xl border p-4'>
        <div className='text-destructive flex items-center gap-2 text-sm font-medium'>
          <CircleAlert className='size-4' aria-hidden='true' />
          {t('Transactional email health is unavailable')}
        </div>
        <Button
          type='button'
          variant='outline'
          size='sm'
          className='mt-3'
          onClick={() => healthQuery.refetch()}
          disabled={healthQuery.isFetching}
        >
          {healthQuery.isFetching && (
            <Loader2 className='animate-spin' aria-hidden='true' />
          )}
          {t('Try again')}
        </Button>
      </div>
    )
  }

  const report = healthQuery.data.data
  const selectedHealth = report.providers.find(
    (provider) => provider.provider === report.selected_provider
  )

  const providerLabel = (_provider: TransactionalEmailProvider) =>
    t('Amazon SES API')

  const failureLabel = (reason?: string) => {
    switch (reason) {
      case 'configuration':
        return t('One or more SES settings are incomplete')
      case 'authentication':
        return t('Provider authentication failed')
      case 'production_access_required':
        return t('SES production access is not approved')
      case 'provider_unavailable':
        return t('Provider is currently unavailable')
      default:
        return t('Provider is not ready')
    }
  }

  const renderSESDetails = (provider: EmailProviderHealth) => {
    let productionAccessLabel = t('Unknown')
    if (provider.production_access) {
      productionAccessLabel = t('Approved')
    } else if (provider.reachable) {
      productionAccessLabel = t('Pending')
    }

    let sandboxRestrictionLabel = t('Unknown')
    if (provider.sandbox_restricted) {
      sandboxRestrictionLabel = t('Restricted (sandbox)')
    } else if (provider.reachable && provider.production_access) {
      sandboxRestrictionLabel = t('None')
    }

    return (
      <div className='text-muted-foreground mt-2 grid gap-1 text-xs sm:grid-cols-2'>
      <span>
        {t('Credentials configured')}:{' '}
        {yesNo(provider.credentials_configured, t)}
      </span>
      <span>
        {t('Region configured')}: {yesNo(provider.region_configured, t)}
      </span>
      <span>
        {t('Sender configured')}: {yesNo(provider.sender_configured, t)}
      </span>
      <span>
        {t('Sending enabled')}: {yesNo(provider.sending_enabled, t)}
      </span>
      <span>
        {t('Production access')}:{' '}
        {productionAccessLabel}
      </span>
      <span>
        {t('Sandbox restrictions')}:{' '}
        {sandboxRestrictionLabel}
      </span>
      </div>
    )
  }

  const renderProvider = (name: TransactionalEmailProvider) => {
    const provider = report.providers.find((item) => item.provider === name)
    if (!provider) return null
    const switchState = getEmailProviderSwitchState(
      provider,
      report.selected_provider
    )
    const isSwitching =
      switchMutation.isPending && switchMutation.variables === provider.provider
    let switchLabel = t('Use provider')
    if (isSwitching) {
      switchLabel = t('Switching...')
    } else if (switchState === 'selected') {
      switchLabel = t('Active')
    }

    return (
      <div
        key={provider.provider}
        className={cn(
          'rounded-lg border p-3',
          switchState === 'selected'
            ? 'border-primary/40 bg-primary/5'
            : 'border-border bg-background'
        )}
      >
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0 flex-1'>
            <div className='flex flex-wrap items-center gap-2'>
              <span className='text-sm font-semibold'>
                {providerLabel(provider.provider)}
              </span>
              {switchState === 'selected' && (
                <Badge variant='secondary'>{t('Selected')}</Badge>
              )}
              <Badge variant={provider.ready ? 'outline' : 'destructive'}>
                {provider.ready ? t('Healthy') : t('Not ready')}
              </Badge>
            </div>
            <p className='text-muted-foreground mt-1 text-xs'>
              {provider.ready
                ? t('Configured and reachable')
                : failureLabel(provider.failure_reason)}
            </p>
            {provider.provider === 'ses' && renderSESDetails(provider)}
          </div>
          <Button
            type='button'
            variant={switchState === 'available' ? 'default' : 'outline'}
            size='sm'
            disabled={switchState !== 'available' || switchMutation.isPending}
            onClick={() => switchMutation.mutate(provider.provider)}
          >
            {isSwitching && (
              <Loader2 className='animate-spin' aria-hidden='true' />
            )}
            {switchLabel}
          </Button>
        </div>
        {provider.provider === 'ses' && <SESCredentialPanel />}
      </div>
    )
  }

  return (
    <div className='border-border bg-muted/20 rounded-xl border p-4'>
      <div className='flex items-start justify-between gap-4'>
        <div>
          <div className='flex items-center gap-2'>
            {selectedHealth?.ready ? (
              <ShieldCheck
                className='size-4 text-emerald-600'
                aria-hidden='true'
              />
            ) : (
              <CircleAlert
                className='text-destructive size-4'
                aria-hidden='true'
              />
            )}
            <h4 className='text-sm font-semibold'>
              {t('Transactional email provider')}
            </h4>
            <Badge variant={selectedHealth?.ready ? 'outline' : 'destructive'}>
              {selectedHealth?.ready ? t('Healthy') : t('Attention required')}
            </Badge>
          </div>
          <p className='text-muted-foreground mt-1 max-w-2xl text-xs'>
            {t(
              'Amazon SES API is configured for transactional verification, reset, receipt, and notification emails.'
            )}
          </p>
        </div>
        <Button
          type='button'
          variant='ghost'
          size='icon-sm'
          aria-label={t('Refresh provider health')}
          onClick={() => healthQuery.refetch()}
          disabled={healthQuery.isFetching}
        >
          <RefreshCw
            className={cn(healthQuery.isFetching && 'animate-spin')}
            aria-hidden='true'
          />
        </Button>
      </div>

      <div className='mt-4 grid gap-3 md:grid-cols-2'>
        {PROVIDER_ORDER.map(renderProvider)}
      </div>

      <div className='border-border mt-4 flex flex-wrap items-center justify-between gap-3 border-t pt-3'>
        <div className='text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs'>
          <span>
            {t('Safe retry queue')}: {report.safe_retry_count}
          </span>
          <span
            className={cn(
              report.manual_review_count > 0 && 'font-medium text-amber-700'
            )}
          >
            {t('Manual review')}: {report.manual_review_count}
          </span>
        </div>
        {report.safe_retry_count > 0 && (
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => retryMutation.mutate()}
            disabled={retryMutation.isPending}
          >
            {retryMutation.isPending ? (
              <Loader2 className='animate-spin' aria-hidden='true' />
            ) : (
              <CheckCircle2 aria-hidden='true' />
            )}
            {retryMutation.isPending
              ? t('Retrying queued emails...')
              : t('Retry safe queue')}
          </Button>
        )}
      </div>

      <div className='border-border mt-3 flex flex-wrap items-end gap-2 border-t pt-3'>
        <label className='min-w-52 flex-1 space-y-1 text-xs'>
          <span className='text-muted-foreground'>{t('Test recipient')}</span>
          <Input
            type='email'
            value={testRecipient}
            onChange={(event) => setTestRecipient(event.target.value)}
            placeholder='name@example.com'
            disabled={testEmailMutation.isPending}
          />
        </label>
        <Button
          type='button'
          variant='outline'
          disabled={
            testEmailMutation.isPending ||
            !isValidTestEmailRecipient(testRecipient)
          }
          onClick={() => testEmailMutation.mutate(testRecipient.trim())}
        >
          {testEmailMutation.isPending && (
            <Loader2 className='animate-spin' aria-hidden='true' />
          )}
          {testEmailMutation.isPending
            ? t('Sending test email...')
            : t('Send test email')}
        </Button>
        <span className='text-muted-foreground w-full text-xs'>
          {t('The test uses the selected transactional email provider.')}
        </span>
      </div>

      {report.manual_review_count > 0 && (
        <p className='mt-3 text-xs text-amber-700' role='status'>
          {t(
            'Ambiguous or stale deliveries are held for manual review and are never retried through another provider.'
          )}
        </p>
      )}
    </div>
  )
}
