export type TocItem = {
  id: string
  text: string
  level: 2 | 3
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replaceAll(/[^\p{L}\p{N}\s-]/gu, '')
    .replaceAll(/\s+/g, '-')
    .replaceAll(/-+/g, '-')
}

export function extractTocFromMarkdown(markdown: string): TocItem[] {
  const items: TocItem[] = []
  const seen = new Map<string, number>()
  const lines = markdown.split('\n')

  for (const line of lines) {
    const match = /^(#{2,3})\s+(.+)$/.exec(line.trim())
    if (!match) continue
    const marks = match[1]
    const rawText = match[2]
    if (!marks || !rawText) continue
    const level = marks.length as 2 | 3
    const text = rawText.replaceAll(/#+\s*$/g, '').trim()
    if (!text) continue
    let id = slugify(text)
    const count = seen.get(id) ?? 0
    seen.set(id, count + 1)
    if (count > 0) id = `${id}-${count + 1}`
    items.push({ id, text, level })
  }

  return items
}
