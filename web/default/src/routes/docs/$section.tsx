import { createFileRoute, redirect } from '@tanstack/react-router'

import {
  DEFAULT_DOC_SECTION,
  isDocSectionId,
} from '@/features/docs/config/nav-tree'
import { DocsPage } from '@/features/docs'

export const Route = createFileRoute('/docs/$section')({
  beforeLoad: ({ params }) => {
    if (!isDocSectionId(params.section)) {
      throw redirect({
        to: '/docs/$section',
        params: { section: DEFAULT_DOC_SECTION },
      })
    }
  },
  component: DocsSectionRoute,
})

function DocsSectionRoute() {
  const { section } = Route.useParams()
  return <DocsPage section={section} />
}
