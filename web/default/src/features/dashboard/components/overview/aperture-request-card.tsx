import { Link } from '@tanstack/react-router'
import { ArrowRight, Copy, TerminalSquare } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import { fetchTokenKey } from '@/features/keys/api'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

export interface ApertureRequestCardProps {
  endpoint: string
  model: string
  keyId?: number
  keyName?: string
  maskedKey: string
  title?: string
}

function buildApertureCurlCommand(args: {
  endpoint: string
  apiKey: string
  model: string
}): string {
  return [
    `curl ${args.endpoint} \\`,
    '  -H "Content-Type: application/json" \\',
    `  -H "Authorization: Bearer ${args.apiKey}" \\`,
    `  -d '{"model":"${args.model}","messages":[{"role":"user","content":"Say hello in one sentence."}]}'`,
  ].join('\n')
}

export function ApertureRequestCard(props: ApertureRequestCardProps) {
  const { t } = useTranslation()
  const [isCopying, setIsCopying] = useState(false)
  const { copyToClipboard } = useCopyToClipboard({ notify: false })
  const previewKey = props.maskedKey || 'sk-...'
  const previewCurl = buildApertureCurlCommand({
    endpoint: props.endpoint,
    apiKey: previewKey,
    model: props.model,
  })

  const handleCopyRequest = async () => {
    if (!props.keyId || isCopying) return

    setIsCopying(true)
    try {
      const result = await fetchTokenKey(props.keyId)
      const key = result.success && result.data?.key ? result.data.key : ''
      if (!key) {
        toast.error(result.message || t('Failed to copy to clipboard'))
        return
      }

      const copied = await copyToClipboard(
        buildApertureCurlCommand({
          endpoint: props.endpoint,
          apiKey: `sk-${key}`,
          model: props.model,
        })
      )
      toast[copied ? 'success' : 'error'](
        copied ? t('Copied to clipboard') : t('Failed to copy to clipboard')
      )
    } catch {
      toast.error(t('Failed to copy to clipboard'))
    } finally {
      setIsCopying(false)
    }
  }

  return (
    <section className='border-border bg-card overflow-hidden rounded-lg border'>
      <div className='border-border flex flex-wrap items-start justify-between gap-3 border-b p-4 sm:p-5'>
        <div className='flex min-w-0 items-start gap-3'>
          <IconBadge tone='info'>
            <TerminalSquare />
          </IconBadge>
          <div className='min-w-0'>
            <p className='editorial-kicker'>{t('N Aperture')}</p>
            <h3 className='mt-1 truncate text-base font-semibold sm:text-lg'>
              {props.title || t('First API request')}
            </h3>
            <p className='text-muted-foreground mt-1 text-xs leading-relaxed'>
              {props.keyName ||
                t('Use a masked key to keep the request boundary clear.')}
            </p>
          </div>
        </div>
        {props.keyId ? (
          <Button
            variant='outline'
            size='sm'
            className='shrink-0 gap-1.5'
            disabled={isCopying}
            onClick={handleCopyRequest}
            aria-label={t('Copy a ready-to-run request')}
          >
            <Copy data-icon='inline-start' />
            {isCopying ? t('Loading') : t('Copy request')}
          </Button>
        ) : (
          <Button
            variant='outline'
            size='sm'
            className='shrink-0'
            render={<Link to='/keys' />}
          >
            {t('Create API Key')}
            <ArrowRight data-icon='inline-end' />
          </Button>
        )}
      </div>

      <div className='bg-muted/60 p-4 sm:p-5'>
        <div className='bg-background border-border overflow-hidden rounded-md border p-3 font-mono text-[11px] leading-6 sm:text-xs'>
          <div className='mb-2 flex items-center gap-1.5'>
            <span
              className='bg-destructive size-2 rounded-full'
              aria-hidden='true'
            />
            <span
              className='bg-warning size-2 rounded-full'
              aria-hidden='true'
            />
            <span
              className='bg-success size-2 rounded-full'
              aria-hidden='true'
            />
          </div>
          <pre className='text-muted-foreground overflow-x-auto break-all whitespace-pre-wrap'>
            {previewCurl}
          </pre>
        </div>
        <div className='text-muted-foreground mt-3 flex flex-wrap items-center justify-between gap-2 text-xs'>
          <span>
            {t('The key stays masked until you choose to copy a request.')}
          </span>
          <Button
            variant='link'
            size='sm'
            className='h-auto shrink-0 p-0'
            render={<Link to='/docs' />}
          >
            {t('Read the docs')}
            <ArrowRight data-icon='inline-end' className='size-3.5' />
          </Button>
        </div>
      </div>
    </section>
  )
}
