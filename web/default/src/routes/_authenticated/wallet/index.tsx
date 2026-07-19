import { createFileRoute } from '@tanstack/react-router'

import { Wallet } from '@/features/wallet'

type WalletSearch = {
  show_history?: boolean
}

function parseShowHistory(value: unknown): boolean {
  return value === true || value === 'true' || value === 1 || value === '1'
}

export const Route = createFileRoute('/_authenticated/wallet/')({
  component: RouteComponent,
  // Query strings arrive as strings; strict z.boolean() throws on reload/deep-link.
  // Keep show_history optional so plain <Link to="/wallet" /> stays valid.
  validateSearch: (search: Record<string, unknown>): WalletSearch => {
    if (
      search.show_history === undefined ||
      search.show_history === null ||
      search.show_history === ''
    ) {
      return {}
    }
    return { show_history: parseShowHistory(search.show_history) }
  },
})

function RouteComponent() {
  const { show_history } = Route.useSearch()
  return <Wallet initialShowHistory={!!show_history} />
}
