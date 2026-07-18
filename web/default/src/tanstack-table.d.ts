import '@tanstack/react-table'

declare module '@tanstack/react-table' {
  interface ColumnMeta<_TData, _TValue> {
    label?: string
    description?: string
    className?: string
    pinned?: 'left' | 'right'
    // Mobile card list layout hints (used by MobileCardList)
    mobileTitle?: boolean // card title area (left, larger text)
    mobileBadge?: boolean // status badge alongside title (right)
    mobileHidden?: boolean // hide this column on mobile entirely
    mobileOrder?: number // lower values appear first in card field area
  }
}
