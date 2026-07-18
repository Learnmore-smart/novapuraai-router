import { createFileRoute, redirect } from '@tanstack/react-router'

import { Main } from '@/components/layout'
import { Playground } from '@/features/playground'
import { isSidebarModuleEnabled } from '@/lib/nav-modules'

export const Route = createFileRoute('/_authenticated/playground/')({
  beforeLoad: () => {
    if (!isSidebarModuleEnabled('chat', 'playground')) {
      throw redirect({ to: '/dashboard' })
    }
  },
  component: PlaygroundPage,
})

function PlaygroundPage() {
  return (
    <Main className='p-0'>
      <Playground />
    </Main>
  )
}
