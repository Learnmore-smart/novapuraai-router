import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatTimestampRelative, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

interface ApiKeyTimestampCellProps {
  timestamp: number
  now: number
  locale?: string
  justNowLabel: string
  className?: string
}

export function ApiKeyTimestampCell(props: ApiKeyTimestampCellProps) {
  if (!props.timestamp || props.timestamp === -1) {
    return <span className='text-muted-foreground text-xs'>-</span>
  }

  const timestampMs = props.timestamp * 1000
  const isJustNow = timestampMs <= props.now && props.now - timestampMs < 60_000
  const relativeTime = isJustNow
    ? props.justNowLabel
    : formatTimestampRelative(props.timestamp, 'seconds', props.locale)
  const absoluteTime = formatTimestampToDate(props.timestamp)

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <time
            dateTime={new Date(timestampMs).toISOString()}
            tabIndex={0}
            className={cn(
              'block truncate font-mono text-xs tabular-nums',
              props.className
            )}
          />
        }
      >
        {relativeTime}
      </TooltipTrigger>
      <TooltipContent>
        <span className='font-mono tabular-nums'>{absoluteTime}</span>
      </TooltipContent>
    </Tooltip>
  )
}
