import { Link } from '@tanstack/react-router'
import { BookOpen, Menu } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from '@/components/ui/drawer'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'

import { DOC_NAV_TREE, type DocSectionId } from '../config/nav-tree'

type DocsSidebarProps = {
  activeSection: DocSectionId
  className?: string
}

function SidebarNav(props: {
  activeSection: DocSectionId
  onNavigate?: () => void
}) {
  const { t } = useTranslation()

  return (
    <nav aria-label={t('Documentation')} className='space-y-6 pr-2'>
      {DOC_NAV_TREE.map((group) => (
        <div key={group.titleKey}>
          <p className='text-muted-foreground mb-2 px-2 text-xs font-semibold tracking-wide uppercase'>
            {t(group.titleKey)}
          </p>
          <ul className='space-y-0.5'>
            {group.items.map((item) => {
              const isActive = item.id === props.activeSection
              return (
                <li key={item.id}>
                  <Link
                    to='/docs/$section'
                    params={{ section: item.id }}
                    onClick={props.onNavigate}
                    className={cn(
                      'block rounded-md px-2.5 py-1.5 text-sm transition-colors',
                      isActive
                        ? 'bg-muted text-foreground font-medium'
                        : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                    )}
                    aria-current={isActive ? 'page' : undefined}
                  >
                    {t(item.titleKey)}
                  </Link>
                </li>
              )
            })}
          </ul>
        </div>
      ))}
    </nav>
  )
}

export function DocsSidebar(props: DocsSidebarProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  return (
    <>
      {/* Mobile drawer trigger */}
      <div className='border-border bg-background/80 sticky top-[var(--app-header-height)] z-20 -mx-4 mb-4 border-b px-4 py-2 backdrop-blur lg:hidden'>
        <Drawer open={open} onOpenChange={setOpen}>
          <DrawerTrigger asChild>
            <Button
              variant='outline'
              size='sm'
              className='w-full justify-start gap-2'
            >
              <Menu className='size-4' />
              {t('Browse documentation')}
            </Button>
          </DrawerTrigger>
          <DrawerContent className='max-h-[85vh]'>
            <DrawerHeader>
              <DrawerTitle className='flex items-center gap-2'>
                <BookOpen className='size-4' />
                {t('Documentation')}
              </DrawerTitle>
            </DrawerHeader>
            <ScrollArea className='h-[70vh] px-4 pb-6'>
              <SidebarNav
                activeSection={props.activeSection}
                onNavigate={() => setOpen(false)}
              />
            </ScrollArea>
          </DrawerContent>
        </Drawer>
      </div>

      {/* Desktop sidebar */}
      <aside
        className={cn(
          'border-border sticky top-[calc(var(--app-header-height)+1.5rem)] hidden max-h-[calc(100vh-var(--app-header-height)-3rem)] w-64 shrink-0 overflow-y-auto border-r pr-4 lg:block',
          props.className
        )}
      >
        <div className='mb-4 flex items-center gap-2 px-2'>
          <BookOpen className='text-primary size-4' />
          <span className='text-sm font-semibold'>{t('Documentation')}</span>
        </div>
        <SidebarNav activeSection={props.activeSection} />
      </aside>
    </>
  )
}
