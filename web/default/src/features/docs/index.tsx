import { useRef } from 'react'

import { PublicLayout } from '@/components/layout'

import { DocsContent } from './components/docs-content'
import { DocsSidebar } from './components/docs-sidebar'
import { DocsToc } from './components/docs-toc'

type DocsPageProps = {
  section?: string
}

export function DocsPage(props: DocsPageProps) {
  // The content container is shared with DocsToc so the TOC can read rendered
  // headings (h2/h3) from the DOM after markdown is parsed.
  const contentRef = useRef<HTMLDivElement>(null)

  return (
    <PublicLayout showMainContainer={false}>
      <div className='mx-auto max-w-7xl px-4 pt-[calc(var(--app-header-height)+1.25rem)] pb-16 md:px-6'>
        <div className='grid gap-8 lg:grid-cols-[16rem_minmax(0,1fr)] xl:grid-cols-[16rem_minmax(0,1fr)_14rem]'>
          <DocsSidebar />
          <DocsContent
            sectionId={props.section ?? ''}
            contentRef={contentRef}
          />
          <DocsToc containerRef={contentRef} />
        </div>
      </div>
    </PublicLayout>
  )
}
