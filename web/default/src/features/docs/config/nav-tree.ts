export type DocSectionId =
  | 'quickstart'
  | 'authentication'
  | 'first-request'
  | 'base-url'
  | 'routing'
  | 'billing'
  | 'rate-limits'
  | 'api-chat'
  | 'api-messages'
  | 'api-gemini'
  | 'api-embeddings'
  | 'api-media'
  | 'api-models'
  | 'api-errors'
  | 'sdk-python'
  | 'sdk-node'
  | 'sdk-go'
  | 'sdk-curl'
  | 'integration-cursor'
  | 'integration-nextchat'
  | 'integration-openwebui'
  | 'integration-dify'
  | 'integration-langchain'
  | 'faq'

export type DocNavItem = {
  id: DocSectionId
  /** i18n key (English source string) */
  titleKey: string
}

export type DocNavGroup = {
  /** i18n key (English source string) */
  titleKey: string
  items: DocNavItem[]
}

/**
 * Official docs sidebar tree.
 * Order is intentional: onboarding → concepts → API → SDKs → integrations → FAQ.
 */
export const DOC_NAV_TREE: DocNavGroup[] = [
  {
    titleKey: 'Getting Started',
    items: [
      { id: 'quickstart', titleKey: 'Quickstart' },
      { id: 'authentication', titleKey: 'Authentication' },
      { id: 'first-request', titleKey: 'Your First Request' },
    ],
  },
  {
    titleKey: 'Core Concepts',
    items: [
      { id: 'base-url', titleKey: 'Base URL & Endpoints' },
      { id: 'routing', titleKey: 'Models & Routing' },
      { id: 'billing', titleKey: 'Billing & Quota' },
      { id: 'rate-limits', titleKey: 'Rate Limits' },
    ],
  },
  {
    titleKey: 'API Reference',
    items: [
      { id: 'api-chat', titleKey: 'Chat Completions' },
      { id: 'api-messages', titleKey: 'Messages (Claude)' },
      { id: 'api-gemini', titleKey: 'Gemini' },
      { id: 'api-embeddings', titleKey: 'Embeddings' },
      { id: 'api-media', titleKey: 'Images / Audio / Rerank' },
      { id: 'api-models', titleKey: 'Models List' },
      { id: 'api-errors', titleKey: 'Errors' },
    ],
  },
  {
    titleKey: 'SDKs & Examples',
    items: [
      { id: 'sdk-python', titleKey: 'Python' },
      { id: 'sdk-node', titleKey: 'Node.js' },
      { id: 'sdk-go', titleKey: 'Go' },
      { id: 'sdk-curl', titleKey: 'curl' },
    ],
  },
  {
    titleKey: 'Integrations',
    items: [
      { id: 'integration-cursor', titleKey: 'Cursor' },
      { id: 'integration-nextchat', titleKey: 'NextChat' },
      { id: 'integration-openwebui', titleKey: 'OpenWebUI' },
      { id: 'integration-dify', titleKey: 'Dify' },
      { id: 'integration-langchain', titleKey: 'LangChain / LlamaIndex' },
    ],
  },
  {
    titleKey: 'FAQ',
    items: [{ id: 'faq', titleKey: 'FAQ' }],
  },
]

export const DOC_SECTIONS: DocSectionId[] = DOC_NAV_TREE.flatMap((group) =>
  group.items.map((item) => item.id)
)

export const DEFAULT_DOC_SECTION: DocSectionId = 'quickstart'

export function isDocSectionId(value: string): value is DocSectionId {
  return DOC_SECTIONS.includes(value as DocSectionId)
}

export function getDocSectionMeta(id: DocSectionId): {
  titleKey: string
  groupTitleKey: string
  index: number
} {
  let index = 0
  for (const group of DOC_NAV_TREE) {
    for (const item of group.items) {
      if (item.id === id) {
        return {
          titleKey: item.titleKey,
          groupTitleKey: group.titleKey,
          index,
        }
      }
      index += 1
    }
  }
  return {
    titleKey: id,
    groupTitleKey: 'Docs',
    index: 0,
  }
}

export function getAdjacentSections(id: DocSectionId): {
  prev: DocSectionId | null
  next: DocSectionId | null
} {
  const index = DOC_SECTIONS.indexOf(id)
  if (index < 0) return { prev: null, next: null }
  return {
    prev: index > 0 ? (DOC_SECTIONS[index - 1] ?? null) : null,
    next:
      index < DOC_SECTIONS.length - 1
        ? (DOC_SECTIONS[index + 1] ?? null)
        : null,
  }
}
