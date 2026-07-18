import { LoaderCircle } from 'lucide-react'

import { cn } from '@/lib/utils'

type SpinnerProps = Omit<
  React.ComponentProps<typeof LoaderCircle>,
  'strokeWidth'
> & {
  strokeWidth?: number
}

function Spinner({ className, strokeWidth = 2, ...props }: SpinnerProps) {
  return (
    <LoaderCircle
      strokeWidth={strokeWidth}
      role='status'
      aria-label='Loading'
      className={cn('size-4 animate-spin', className)}
      {...props}
    />
  )
}

export { Spinner }
