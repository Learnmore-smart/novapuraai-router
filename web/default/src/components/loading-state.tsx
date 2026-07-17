/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

interface LoadingStateProps {
  className?: string
  message?: string
  size?: 'sm' | 'md' | 'lg'
  inline?: boolean
}

export function LoadingState(props: LoadingStateProps) {
  const { t } = useTranslation()
  const size = props.size ?? 'md'

  if (props.inline) {
    return (
      <span
        className={cn(
          'text-muted-foreground inline-flex items-center gap-2 text-sm',
          props.className
        )}
      >
        <span
          className={cn(
            'border-border border-t-primary inline-block animate-spin rounded-full border-2',
            size === 'sm' && 'size-3.5',
            size === 'md' && 'size-4',
            size === 'lg' && 'size-5'
          )}
          aria-hidden='true'
        />
        {props.message != null && <span>{props.message}</span>}
      </span>
    )
  }

  return (
    <div
      className={cn(
        'flex min-h-[220px] flex-col items-center justify-center gap-4 px-4',
        props.className
      )}
      role='status'
      aria-live='polite'
      aria-label={props.message ?? t('Loading...')}
    >
      <div className='w-full max-w-xs space-y-2.5'>
        <Skeleton className='h-3 w-1/3 rounded-sm' />
        <Skeleton className='h-3 w-full rounded-sm' />
        <Skeleton className='h-3 w-5/6 rounded-sm' />
        <Skeleton className='h-3 w-2/3 rounded-sm' />
      </div>
      <p className='text-muted-foreground text-sm'>
        {props.message ?? t('Loading...')}
      </p>
    </div>
  )
}
