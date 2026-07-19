import { createFileRoute, redirect } from '@tanstack/react-router'

import { DEFAULT_DOC_SECTION } from '@/features/docs/config/nav-tree'

export const Route = createFileRoute('/docs/')({
  beforeLoad: () => {
    throw redirect({
      to: '/docs/$section',
      params: { section: DEFAULT_DOC_SECTION },
    })
  },
})
