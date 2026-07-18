import { cn } from '@/lib/utils'

type TableIdProps = {
  className?: string
  value: number | string
}

export function TableId(props: TableIdProps) {
  return (
    <span
      className={cn(
        'text-muted-foreground inline-block font-mono tabular-nums',
        props.className
      )}
    >
      {props.value}
    </span>
  )
}
