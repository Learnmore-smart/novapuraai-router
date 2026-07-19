import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, KeyRound, Loader2, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import {
  deleteTransactionalEmailSESCredentials,
  getTransactionalEmailSESCredentials,
  updateTransactionalEmailSESCredentials,
} from '../api'
import { buildSESCredentialUpdate } from './ses-credential-form'
import { isValidEmailSender } from './email-sender'

const MASK = '••••••••••••'

export function SESCredentialPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(true)
  const [removeOpen, setRemoveOpen] = useState(false)
  const [accessKeyId, setAccessKeyId] = useState('')
  const [secretAccessKey, setSecretAccessKey] = useState('')
  const [sessionToken, setSessionToken] = useState('')
  const [clearSessionToken, setClearSessionToken] = useState(false)
  const [region, setRegion] = useState('')
  const [fromAddress, setFromAddress] = useState('')
  const [initialRegion, setInitialRegion] = useState('')
  const [initialFromAddress, setInitialFromAddress] = useState('')

  const statusQuery = useQuery({
    queryKey: ['transactional-email-ses-credentials'],
    queryFn: getTransactionalEmailSESCredentials,
  })

  useEffect(() => {
    if (!statusQuery.data?.success || !statusQuery.data.data) return
    const nextRegion = statusQuery.data.data.region ?? ''
    const nextFrom = statusQuery.data.data.from_address ?? ''
    setRegion(nextRegion)
    setFromAddress(nextFrom)
    setInitialRegion(nextRegion)
    setInitialFromAddress(nextFrom)
  }, [statusQuery.data])

  const refreshEmailState = async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: ['transactional-email-ses-credentials'],
      }),
      queryClient.invalidateQueries({
        queryKey: ['transactional-email-health'],
      }),
      queryClient.invalidateQueries({ queryKey: ['system-options'] }),
    ])
  }

  const saveMutation = useMutation({
    mutationFn: updateTransactionalEmailSESCredentials,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message)
        return
      }
      setAccessKeyId('')
      setSecretAccessKey('')
      setSessionToken('')
      setClearSessionToken(false)
      toast.success(t('SES settings saved securely'))
      await refreshEmailState()
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to save SES settings'))
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteTransactionalEmailSESCredentials,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message)
        return
      }
      setRemoveOpen(false)
      toast.success(t('Dashboard SES credentials removed'))
      await refreshEmailState()
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to remove SES credentials'))
    },
  })

  const status = statusQuery.data?.success ? statusQuery.data.data : undefined
  const environmentManaged = status?.source === 'environment'
  const statusUnavailable = Boolean(
    statusQuery.isError || (statusQuery.data && !statusQuery.data.success)
  )
  let sourceLabel = t('None')
  if (environmentManaged) {
    sourceLabel = t('Environment')
  } else if (status?.source === 'database') {
    sourceLabel = t('Encrypted database')
  }

  const regionChanged = region.trim() !== initialRegion.trim()
  const fromChanged = fromAddress.trim() !== initialFromAddress.trim()
  const hasCredentialChanges = Boolean(
    accessKeyId.trim() || secretAccessKey || sessionToken || clearSessionToken
  )
  const hasChanges = hasCredentialChanges || regionChanged || fromChanged
  const fromValid =
    fromAddress.trim() === '' || isValidEmailSender(fromAddress.trim())

  const submitCredentials = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!fromValid) {
      toast.error(t('Enter a valid verified sender address or leave blank'))
      return
    }
    const payload = buildSESCredentialUpdate({
      accessKeyId,
      secretAccessKey,
      sessionToken,
      clearSessionToken,
      region,
      fromAddress,
      initialRegion,
      initialFromAddress,
    })
    if (Object.keys(payload).length === 0) {
      toast.error(t('Enter at least one SES setting change'))
      return
    }
    saveMutation.mutate(payload)
  }

  return (
    <Collapsible open={open} onOpenChange={setOpen} className='mt-3'>
      <CollapsibleTrigger className='border-border bg-muted/30 hover:bg-muted/60 flex w-full items-center justify-between gap-3 rounded-lg border px-3 py-2 text-left transition-colors'>
        <span className='flex min-w-0 items-center gap-2'>
          <KeyRound
            className='text-muted-foreground size-4'
            aria-hidden='true'
          />
          <span className='text-xs font-medium'>
            {t('Configure Amazon SES API')}
          </span>
          {statusQuery.isPending && <Skeleton className='h-5 w-16' />}
          {!statusQuery.isPending && status?.configured && (
            <Badge variant='outline'>{t('Credentials configured')}</Badge>
          )}
          {!statusQuery.isPending && !status?.configured && (
            <Badge variant='secondary'>{t('Credentials not configured')}</Badge>
          )}
        </span>
        <ChevronDown
          className={cn('size-4 transition-transform', open && 'rotate-180')}
          aria-hidden='true'
        />
      </CollapsibleTrigger>

      <CollapsibleContent className='border-border mt-2 rounded-lg border p-3'>
        {statusUnavailable && (
          <div className='text-destructive text-xs'>
            {t('SES credential status is unavailable')}
          </div>
        )}
        {!statusUnavailable && statusQuery.isPending && (
          <div className='space-y-3' aria-live='polite'>
            <Skeleton className='h-8 w-full' />
            <Skeleton className='h-8 w-full' />
            <Skeleton className='h-8 w-full' />
          </div>
        )}
        {!statusUnavailable && !statusQuery.isPending && (
          <>
            <div className='mb-3 flex flex-wrap items-center gap-2 text-xs'>
              <span className='text-muted-foreground'>
                {t('Credential source')}:
              </span>
              <Badge variant='secondary'>{sourceLabel}</Badge>
              {status?.configured && (
                <span
                  className='text-muted-foreground font-mono tracking-wider select-none'
                  aria-label={t('Credentials are configured')}
                >
                  {MASK}
                </span>
              )}
            </div>

            <form className='space-y-3' onSubmit={submitCredentials}>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Amazon SES API uses AWS SDK credentials, region, and a verified sender. Session token is optional and should stay blank for long-term IAM user keys.'
                )}
              </p>

              <div className='grid gap-3 md:grid-cols-2'>
                <div className='space-y-1.5'>
                  <Label htmlFor='ses-access-key-id'>
                    {t('AWS access key ID')}
                  </Label>
                  <Input
                    id='ses-access-key-id'
                    type='password'
                    autoComplete='new-password'
                    spellCheck={false}
                    maxLength={4096}
                    value={accessKeyId}
                    onChange={(event) => setAccessKeyId(event.target.value)}
                    placeholder={status?.configured ? MASK : 'AKIA...'}
                  />
                </div>
                <div className='space-y-1.5'>
                  <Label htmlFor='ses-secret-access-key'>
                    {t('AWS secret access key')}
                  </Label>
                  <Input
                    id='ses-secret-access-key'
                    type='password'
                    autoComplete='new-password'
                    spellCheck={false}
                    maxLength={4096}
                    value={secretAccessKey}
                    onChange={(event) =>
                      setSecretAccessKey(event.target.value)
                    }
                    placeholder={
                      status?.configured ? MASK : t('Enter secret access key')
                    }
                  />
                </div>
              </div>

              <div className='space-y-1.5'>
                <Label htmlFor='ses-session-token'>
                  {t('AWS session token (optional)')}
                </Label>
                <Input
                  id='ses-session-token'
                  type='password'
                  autoComplete='new-password'
                  spellCheck={false}
                  maxLength={4096}
                  value={sessionToken}
                  onChange={(event) => {
                    setSessionToken(event.target.value)
                    if (event.target.value) setClearSessionToken(false)
                  }}
                  placeholder={
                    status?.has_session_token
                      ? MASK
                      : t('Leave blank for long-term IAM credentials')
                  }
                />
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Leave blank for long-term IAM user access keys. Only temporary STS credentials need a session token.'
                  )}
                </p>
              </div>

              {status?.has_session_token && (
                <div className='flex items-center gap-2'>
                  <Checkbox
                    id='ses-clear-session-token'
                    checked={clearSessionToken}
                    disabled={Boolean(sessionToken)}
                    onCheckedChange={setClearSessionToken}
                  />
                  <Label
                    htmlFor='ses-clear-session-token'
                    className='text-xs font-normal'
                  >
                    {t('Remove the saved session token')}
                  </Label>
                </div>
              )}

              <div className='grid gap-3 md:grid-cols-2'>
                <div className='space-y-1.5'>
                  <Label htmlFor='ses-region'>{t('AWS region')}</Label>
                  <Input
                    id='ses-region'
                    autoComplete='off'
                    spellCheck={false}
                    value={region}
                    onChange={(event) => setRegion(event.target.value)}
                    placeholder='us-east-2'
                  />
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'SES API region, for example us-east-2 or ap-southeast-1. Environment AWS_SES_REGION overrides the saved value when set.'
                    )}
                  </p>
                </div>
                <div className='space-y-1.5'>
                  <Label htmlFor='ses-from-address'>
                    {t('Verified sender address')}
                  </Label>
                  <Input
                    id='ses-from-address'
                    autoComplete='off'
                    value={fromAddress}
                    onChange={(event) => setFromAddress(event.target.value)}
                    placeholder={t('NovaPuraAI <noreply@example.com>')}
                    aria-invalid={!fromValid}
                  />
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'Must be a verified SES identity. Display form is allowed, for example NovaPuraAI <noreply@novapuraai.com>.'
                    )}
                  </p>
                </div>
              </div>

              <div className='flex flex-wrap items-center justify-between gap-2 pt-1'>
                <div>
                  {status?.source === 'database' && (
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      className='text-destructive hover:text-destructive'
                      onClick={() => setRemoveOpen(true)}
                      disabled={
                        saveMutation.isPending || deleteMutation.isPending
                      }
                    >
                      <Trash2 aria-hidden='true' />
                      {t('Remove saved credentials')}
                    </Button>
                  )}
                </div>
                <Button
                  type='submit'
                  size='sm'
                  disabled={
                    !hasChanges ||
                    !fromValid ||
                    saveMutation.isPending ||
                    deleteMutation.isPending
                  }
                >
                  {saveMutation.isPending && (
                    <Loader2 className='animate-spin' aria-hidden='true' />
                  )}
                  {saveMutation.isPending
                    ? t('Saving SES settings...')
                    : t('Save SES settings')}
                </Button>
              </div>
            </form>
          </>
        )}
      </CollapsibleContent>

      <ConfirmDialog
        open={removeOpen}
        onOpenChange={setRemoveOpen}
        title={t('Remove Dashboard SES credentials?')}
        desc={t(
          'Amazon SES will become unavailable unless complete environment credentials are configured.'
        )}
        confirmText={
          deleteMutation.isPending ? t('Removing...') : t('Remove credentials')
        }
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => deleteMutation.mutate()}
      />
    </Collapsible>
  )
}
