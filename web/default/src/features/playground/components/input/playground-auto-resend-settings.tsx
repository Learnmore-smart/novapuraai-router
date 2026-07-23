import { RefreshCwIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { PromptInputButton } from '@/components/ai-elements/prompt-input'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Switch } from '@/components/ui/switch'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { AUTO_RESEND } from '../../constants'
import type { PlaygroundConfig } from '../../types'

type PlaygroundAutoResendSettingsProps = {
  config: PlaygroundConfig
  disabled?: boolean
  onConfigChange: <K extends keyof PlaygroundConfig>(
    key: K,
    value: PlaygroundConfig[K]
  ) => void
}

function clampMaxRetries(value: number): number {
  if (Number.isNaN(value) || value < 0) {
    return 0
  }
  return Math.min(AUTO_RESEND.MAX_RETRIES_LIMIT, Math.trunc(value))
}

export function PlaygroundAutoResendSettings({
  config,
  disabled,
  onConfigChange,
}: PlaygroundAutoResendSettingsProps) {
  const { t } = useTranslation()

  const trigger = (
    <PromptInputButton
      aria-label={t('Auto-resend on reload')}
      className='text-muted-foreground hover:text-foreground hover:bg-muted/70 font-medium'
      disabled={disabled}
      variant='ghost'
    >
      <RefreshCwIcon size={16} />
    </PromptInputButton>
  )

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger render={<PopoverTrigger render={trigger} />} />
        <TooltipContent>
          <p>{t('Auto-resend on reload')}</p>
        </TooltipContent>
      </Tooltip>
      <PopoverContent
        align='start'
        className='w-80 max-w-[calc(100vw-2rem)] gap-3 p-3'
        collisionPadding={8}
        side='top'
        sideOffset={8}
      >
        <div className='space-y-1 px-1'>
          <div className='text-sm font-semibold'>
            {t('Auto-resend on reload')}
          </div>
          <div className='text-muted-foreground text-xs leading-4'>
            {t(
              'When the page is reloaded mid-generation, automatically resend the request. Retries may incur multiple charges.'
            )}
          </div>
        </div>
        <div className='border-border/70 bg-background/60 grid gap-2 rounded-lg border p-3'>
          <div className='flex items-start justify-between gap-3'>
            <div className='min-w-0 space-y-1'>
              <label
                className='truncate text-sm leading-5 font-medium'
                htmlFor='playground-auto-resend-enabled'
              >
                {t('Auto-resend on reload')}
              </label>
              <p className='text-muted-foreground text-xs leading-4'>
                {t('Enable to resend the last request after a reload.')}
              </p>
            </div>
            <Switch
              aria-label={t('Auto-resend on reload')}
              checked={config.autoResendEnabled}
              disabled={disabled}
              id='playground-auto-resend-enabled'
              onCheckedChange={(checked) =>
                onConfigChange('autoResendEnabled', checked)
              }
              size='sm'
            />
          </div>
        </div>
        <div
          className='border-border/70 bg-background/60 grid gap-2 rounded-lg border p-3 transition-opacity'
          style={{ opacity: config.autoResendEnabled ? 1 : 0.55 }}
        >
          <div className='flex items-start justify-between gap-3'>
            <div className='min-w-0 space-y-1'>
              <label
                className='truncate text-sm leading-5 font-medium'
                htmlFor='playground-auto-resend-max-retries'
              >
                {t('Max retries')}
              </label>
              <p className='text-muted-foreground text-xs leading-4'>
                {t('Number of retry attempts before falling back to an error.')}
              </p>
            </div>
            <Input
              className='w-20'
              disabled={disabled || !config.autoResendEnabled}
              id='playground-auto-resend-max-retries'
              inputMode='numeric'
              max={AUTO_RESEND.MAX_RETRIES_LIMIT}
              min={0}
              onChange={(event) =>
                onConfigChange(
                  'autoResendMaxRetries',
                  clampMaxRetries(Number.parseInt(event.target.value, 10))
                )
              }
              step={1}
              type='number'
              value={config.autoResendMaxRetries}
            />
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
