import { Check, Copy } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useStatus } from '@/hooks/use-status'
import { cn } from '@/lib/utils'

/** Ledger rows shown on the response side — the product's core promise. */
const LEDGER_ROWS = [
  { label: 'Route', value: 'model-pool · healthy channel' },
  { label: 'Tokens', value: '1,024 in · 512 out' },
  { label: 'Latency', value: '840 ms' },
  { label: 'Cost', value: '$0.0032' },
] as const

interface HeroTerminalDemoProps {
  className?: string
}

/**
 * Hero product visual — a self-contained “API console”: the request you send on
 * the left, the routed response + ledger entry on the right. Communicates
 * compatibility, routing, and per-request accounting without a heavy animation.
 */
export function HeroTerminalDemo(props: HeroTerminalDemoProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const [copied, setCopied] = useState(false)
  const configuredAddress =
    typeof status?.server_address === 'string'
      ? status.server_address.trim()
      : ''
  const baseAddress = (configuredAddress || window.location.origin).replace(
    /\/+$/,
    ''
  )
  const endpoint = baseAddress.endsWith('/v1')
    ? `${baseAddress}/chat/completions`
    : `${baseAddress}/v1/chat/completions`
  const requestExample = [
    `curl ${endpoint} \\`,
    '  -H "Authorization: Bearer sk-••••••••" \\',
    '  -H "Content-Type: application/json" \\',
    "  -d '{",
    '    "model": "your-model",',
    '    "messages": [',
    '      { "role": "user", "content": "Hello" }',
    '    ]',
    "  }'",
  ].join('\n')

  const copyRequest = async () => {
    try {
      await navigator.clipboard.writeText(requestExample)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1800)
    } catch {
      setCopied(false)
    }
  }

  return (
    <div
      className={cn(
        'np-surface-lg relative w-full overflow-hidden',
        props.className
      )}
    >
      {/* Window chrome */}
      <div className='border-border flex items-center gap-3 border-b px-4 py-3 sm:px-5'>
        <div className='flex items-center gap-1.5' aria-hidden='true'>
          <span className='size-2.5 rounded-full bg-[#ef6a6a]' />
          <span className='size-2.5 rounded-full bg-[#f2b53d]' />
          <span className='size-2.5 rounded-full bg-[#57c07f]' />
        </div>
        <div className='bg-muted text-muted-foreground ml-1 flex min-w-0 items-center gap-2 rounded-md px-2.5 py-1'>
          <span className='border-border text-foreground bg-background rounded border px-1.5 py-0.5 font-mono text-[0.625rem] font-bold'>
            POST
          </span>
          <span className='truncate font-mono text-[0.6875rem]'>
            /v1/chat/completions
          </span>
        </div>
        <span className='text-success bg-success/10 ml-auto inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-[0.6875rem] font-semibold'>
          <span className='bg-success size-1.5 animate-pulse rounded-full' />
          {t('200 OK')}
        </span>
      </div>

      {/* Two-pane: request → response */}
      <div className='grid md:grid-cols-2'>
        {/* Request */}
        <div className='border-border relative md:border-r'>
          <div className='text-muted-foreground border-border flex items-center justify-between border-b px-4 py-2.5 sm:px-5'>
            <span className='text-[0.6875rem] font-semibold tracking-wide uppercase'>
              {t('Request')}
            </span>
            <button
              type='button'
              onClick={copyRequest}
              className='hover:text-foreground focus-visible:ring-ring rounded-md p-1 transition-colors focus-visible:ring-2 focus-visible:outline-none'
              aria-label={t('Copy example request')}
            >
              {copied ? (
                <Check className='text-success size-3.5' />
              ) : (
                <Copy className='size-3.5' />
              )}
            </button>
          </div>
          <pre className='text-foreground/80 overflow-x-auto px-4 py-4 text-[0.6875rem] leading-relaxed sm:px-5'>
            <code>{requestExample}</code>
          </pre>
        </div>

        {/* Response + ledger */}
        <div className='bg-muted/30'>
          <div className='text-muted-foreground border-border flex items-center justify-between border-b px-4 py-2.5 sm:px-5'>
            <span className='text-[0.6875rem] font-semibold tracking-wide uppercase'>
              {t('Ledger entry')}
            </span>
            <span className='font-mono text-[0.625rem]'>{t('recorded')}</span>
          </div>
          <dl className='divide-border divide-y'>
            {LEDGER_ROWS.map((row) => (
              <div
                key={row.label}
                className='flex items-center justify-between gap-3 px-4 py-[0.6875rem] sm:px-5'
              >
                <dt className='text-muted-foreground text-xs font-medium'>
                  {t(row.label)}
                </dt>
                <dd className='text-foreground font-mono text-xs tabular-nums'>
                  {row.value}
                </dd>
              </div>
            ))}
          </dl>
          <p className='text-muted-foreground px-4 pt-3 pb-4 text-[0.6875rem] leading-5 sm:px-5'>
            {t(
              'Promotional balance is applied before cash — every call is auditable.'
            )}
          </p>
        </div>
      </div>
    </div>
  )
}
