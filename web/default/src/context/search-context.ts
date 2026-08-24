import {
  createContext,
  useContext,
  type Dispatch,
  type SetStateAction,
} from 'react'

type SearchContextType = {
  open: boolean
  setOpen: Dispatch<SetStateAction<boolean>>
}

export const SearchContext = createContext<SearchContextType | null>(null)

export function useSearch() {
  const searchContext = useContext(SearchContext)

  if (!searchContext) {
    throw new Error('useSearch has to be used within SearchProvider')
  }

  return searchContext
}
