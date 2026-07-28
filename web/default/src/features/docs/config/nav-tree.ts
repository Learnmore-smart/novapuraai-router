import {
  Compass,
  CreditCard,
  FileQuestion,
  Gauge,
  KeyRound,
  Layers,
  Boxes,
  Plug,
  Code2,
  Terminal,
  AlertTriangle,
  Image,
  ListOrdered,
  Route,
  Send,
  Sparkles,
  Workflow,
  type LucideIcon,
} from 'lucide-react'

export interface DocNavItem {
  id: string
  titleKey: string
  href: string
  icon?: LucideIcon
}

export interface DocNavGroup {
  id: string
  titleKey: string
  items: DocNavItem[]
}

// Order is used for prev/next navigation.
// Sections must match `i18n/docs/<id>/<lang>.md` files.
export const DOC_NAV_GROUPS: DocNavGroup[] = [
  {
    id: 'getting-started',
    titleKey: 'Getting Started',
    items: [
      {
        id: 'quickstart',
        titleKey: 'Quickstart',
        href: '/docs/quickstart',
        icon: Compass,
      },
      {
        id: 'authentication',
        titleKey: 'Authentication',
        href: '/docs/authentication',
        icon: KeyRound,
      },
      {
        id: 'first-request',
        titleKey: 'Your First Request',
        href: '/docs/first-request',
        icon: Send,
      },
    ],
  },
  {
    id: 'core-concepts',
    titleKey: 'Core Concepts',
    items: [
      {
        id: 'base-url',
        titleKey: 'Base URL & Endpoints',
        href: '/docs/base-url',
        icon: Route,
      },
      {
        id: 'routing',
        titleKey: 'Models & Routing',
        href: '/docs/routing',
        icon: Workflow,
      },
      {
        id: 'billing',
        titleKey: 'Billing & Quota',
        href: '/docs/billing',
        icon: CreditCard,
      },
      {
        id: 'rate-limits',
        titleKey: 'Rate Limits',
        href: '/docs/rate-limits',
        icon: Gauge,
      },
    ],
  },
  {
    id: 'api-reference',
    titleKey: 'API Reference',
    items: [
      {
        id: 'api-chat',
        titleKey: 'Chat Completions',
        href: '/docs/api-chat',
        icon: Sparkles,
      },
      {
        id: 'api-messages',
        titleKey: 'Messages (Claude)',
        href: '/docs/api-messages',
        icon: Layers,
      },
      {
        id: 'api-gemini',
        titleKey: 'Gemini',
        href: '/docs/api-gemini',
        icon: Sparkles,
      },
      {
        id: 'api-embeddings',
        titleKey: 'Embeddings',
        href: '/docs/api-embeddings',
        icon: Layers,
      },
      {
        id: 'api-media',
        titleKey: 'Images / Audio / Rerank',
        href: '/docs/api-media',
        icon: Image,
      },
      {
        id: 'api-models',
        titleKey: 'Models List',
        href: '/docs/api-models',
        icon: ListOrdered,
      },
      {
        id: 'api-errors',
        titleKey: 'Errors',
        href: '/docs/api-errors',
        icon: AlertTriangle,
      },
    ],
  },
  {
    id: 'sdks',
    titleKey: 'SDKs & Examples',
    items: [
      {
        id: 'sdk-python',
        titleKey: 'Python',
        href: '/docs/sdk-python',
        icon: Code2,
      },
      {
        id: 'sdk-node',
        titleKey: 'Node.js',
        href: '/docs/sdk-node',
        icon: Code2,
      },
      {
        id: 'sdk-go',
        titleKey: 'Go',
        href: '/docs/sdk-go',
        icon: Code2,
      },
      {
        id: 'sdk-curl',
        titleKey: 'curl',
        href: '/docs/sdk-curl',
        icon: Terminal,
      },
    ],
  },
  {
    id: 'integrations',
    titleKey: 'Integrations',
    items: [
      {
        id: 'integration-cursor',
        titleKey: 'Cursor',
        href: '/docs/integration-cursor',
        icon: Plug,
      },
      {
        id: 'integration-nextchat',
        titleKey: 'NextChat',
        href: '/docs/integration-nextchat',
        icon: Plug,
      },
      {
        id: 'integration-openwebui',
        titleKey: 'OpenWebUI',
        href: '/docs/integration-openwebui',
        icon: Plug,
      },
      {
        id: 'integration-dify',
        titleKey: 'Dify',
        href: '/docs/integration-dify',
        icon: Plug,
      },
      {
        id: 'integration-langchain',
        titleKey: 'LangChain / LlamaIndex',
        href: '/docs/integration-langchain',
        icon: Boxes,
      },
    ],
  },
  {
    id: 'faq',
    titleKey: 'FAQ',
    items: [
      {
        id: 'faq',
        titleKey: 'FAQ',
        href: '/docs/faq',
        icon: FileQuestion,
      },
    ],
  },
]

// Flat ordered list for prev/next navigation across all groups.
export const DOC_FLAT_ITEMS: DocNavItem[] = DOC_NAV_GROUPS.flatMap(
  (group) => group.items
)

// Default landing section for `/docs`.
export const DEFAULT_DOC_SECTION = 'quickstart'

// Back-compat alias (older code may import DOC_DEFAULT_SECTION).
export const DOC_DEFAULT_SECTION = DEFAULT_DOC_SECTION

const DOC_SECTION_IDS: ReadonlySet<string> = new Set(
  DOC_FLAT_ITEMS.map((item) => item.id)
)

export function isDocSectionId(
  sectionId: string | undefined
): sectionId is string {
  return typeof sectionId === 'string' && DOC_SECTION_IDS.has(sectionId)
}

export function findDocItem(sectionId: string): DocNavItem | undefined {
  return DOC_FLAT_ITEMS.find((item) => item.id === sectionId)
}

export function findDocNeighbor(sectionId: string): {
  prev?: DocNavItem
  next?: DocNavItem
} {
  const index = DOC_FLAT_ITEMS.findIndex((item) => item.id === sectionId)
  if (index === -1) return {}
  return {
    prev: index > 0 ? DOC_FLAT_ITEMS[index - 1] : undefined,
    next:
      index < DOC_FLAT_ITEMS.length - 1 ? DOC_FLAT_ITEMS[index + 1] : undefined,
  }
}
