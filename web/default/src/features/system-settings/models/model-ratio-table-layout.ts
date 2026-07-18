const columnWidthClassNames: Record<string, string> = {
  select: 'w-9',
  name: 'w-[300px]',
  billingMode: 'w-[120px]',
  priceSummary: 'w-[300px]',
  actions: 'w-[72px]',
}

export function getModelRatioColumnWidthClassNames(
  columnIds: string[]
): string[] {
  return columnIds.map(
    (columnId) => columnWidthClassNames[columnId] || 'w-auto'
  )
}
