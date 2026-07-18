import { cn } from '@/lib/utils'

import { useSuppressSettingsSectionHeader } from './settings-page-context'

type SettingsSectionProps = {
  title: string
  titleProps?: React.HTMLAttributes<HTMLHeadingElement>
  children: React.ReactNode
  className?: string
}

export function SettingsSection({
  title,
  titleProps,
  children,
  className,
}: SettingsSectionProps) {
  const suppressHeader = useSuppressSettingsSectionHeader()

  return (
    <section className={cn('flex flex-col gap-4', className)}>
      {!suppressHeader && (
        <div className='flex flex-col gap-1'>
          <h3
            {...titleProps}
            className={cn('text-base font-semibold', titleProps?.className)}
          >
            {title}
          </h3>
        </div>
      )}
      {children}
    </section>
  )
}
