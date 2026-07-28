import { HtmlContent, type HtmlContentVariant } from '@/components/html-content'
import { Markdown } from '@/components/ui/markdown'

type RichContentMode = 'markdown' | 'html'

interface RichContentProps {
  content: string
  mode?: RichContentMode
  breaks?: boolean
  className?: string
  htmlVariant?: HtmlContentVariant
}

export function RichContent(props: RichContentProps) {
  if (props.mode === 'html') {
    return (
      <HtmlContent
        content={props.content}
        className={props.className}
        variant={props.htmlVariant}
      />
    )
  }

  return (
    <Markdown breaks={props.breaks} className={props.className}>
      {props.content}
    </Markdown>
  )
}
