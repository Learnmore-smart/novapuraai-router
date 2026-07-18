import { createContext } from 'react'

/**
 * Where channel row-derived controls are being rendered. Card view can tune
 * compact display details while table view keeps the full desktop treatment.
 */
export type ChannelRowActionsLayout = 'table' | 'card'

export const ChannelRowActionsLayoutContext =
  createContext<ChannelRowActionsLayout>('table')
