import { useState } from 'react'

import { PublicLayout } from '@/components/layout'

import { DocsContent } from './components/docs-content'
import { DocsSidebar } from './components/docs-sidebar'
import { DocsToc, type TocItem } from './components/docs-toc'
import { useDocContent } from './hooks/use-doc-content'

type DocsPageProps = {
  section?: string
}

export function DocsPage(props: DocsPageProps) {
  const content = useDocContent(props.section)
  const [toc, setToc] = useState<TocItem[]>([])

  return (
    <PublicLayout showMainContainer={false}>
      <div className='mx-auto max-w-7xl px-4 pt-[calc(var(--app-header-height)+1.25rem)] pb-16 md:px-6'>
        <div className='grid gap-8 lg:grid-cols-[16rem_minmax(0,1fr)] xl:grid-cols-[16rem_minmax(0,1fr)_14rem]'>
          <DocsSidebar activeSection={content.section} />
          <DocsContent
            section={content.section}
            markdown={content.markdown}
            loading={content.loading}
            error={content.error}
            usedFallback={content.usedFallback}
            onTocChange={setToc}
          />
          <DocsToc items={toc} />
        </div>
      </div>
    </PublicLayout>
  )
}


