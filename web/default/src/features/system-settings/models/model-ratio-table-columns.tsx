import type { ColumnDef } from '@tanstack/react-table'

import { DataTableColumnHeader } from '@/components/data-table/core/column-header'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import { StatusBadge } from '@/components/status-badge'
import { Checkbox } from '@/components/ui/checkbox'

import {
  getModeLabel,
  getModeVariant,
  getPriceDetail,
  getPriceSummary,
  type ModelRow,
} from './model-pricing-snapshots'

const filterBySelectedValues = (
  rowValue: unknown,
  filterValue: unknown
): boolean => {
  if (!Array.isArray(filterValue) || filterValue.length === 0) return true
  return filterValue.includes(String(rowValue))
}

type BuildModelRatioColumnsOptions = {
  onDelete: (name: string) => void
  onEdit: (model: ModelRow) => void
  deleteDisabled?: boolean
  t: (key: string) => string
}

export function buildModelRatioColumns({
  onDelete,
  onEdit,
  deleteDisabled,
  t,
}: BuildModelRatioColumnsOptions): ColumnDef<ModelRow>[] {
  return [
    {
      id: 'select',
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          indeterminate={table.getIsSomePageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label={t('Select all')}
          className='translate-y-[2px]'
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label={t('Select row')}
          className='translate-y-[2px]'
        />
      ),
      enableSorting: false,
      enableHiding: false,
      size: 40,
    },
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Model name')} />
      ),
      cell: ({ row }) => (
        <div className='flex min-w-0 items-center gap-2 font-medium'>
          <span className='min-w-0 truncate'>{row.getValue('name')}</span>
          {row.original.billingMode === 'tiered_expr' && (
            <StatusBadge
              label={t('Tiered')}
              variant='info'
              copyable={false}
              className='shrink-0'
            />
          )}
          {row.original.hasConflict && (
            <StatusBadge
              label={t('Conflict')}
              variant='danger'
              copyable={false}
              className='shrink-0'
            />
          )}
        </div>
      ),
      enableHiding: false,
    },
    {
      accessorKey: 'billingMode',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Mode')} />
      ),
      cell: ({ row }) => (
        <StatusBadge
          label={t(getModeLabel(row.original.billingMode))}
          variant={getModeVariant(row.original.billingMode)}
          copyable={false}
          showDot={false}
          className='-ml-1.5 px-0'
        />
      ),
      filterFn: (row, id, value) =>
        filterBySelectedValues(row.getValue(id), value),
      meta: { label: t('Mode') },
    },
    {
      id: 'priceSummary',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Price summary')} />
      ),
      cell: ({ row }) => (
        <div className='flex min-w-0 flex-col gap-1'>
          <span className='truncate font-medium'>
            {getPriceSummary(row.original, t)}
          </span>
          <span className='text-muted-foreground truncate text-xs'>
            {getPriceDetail(row.original, t)}
          </span>
        </div>
      ),
      sortingFn: (rowA, rowB) =>
        getPriceSummary(rowA.original, t).localeCompare(
          getPriceSummary(rowB.original, t)
        ),
      meta: { label: t('Price summary') },
    },
    {
      id: 'actions',
      header: () => <div>{t('Actions')}</div>,
      cell: ({ row }) => (
        <StaticRowActions
          editLabel={t('Edit')}
          deleteLabel={t('Delete')}
          menuLabel={t('Open menu')}
          onEdit={() => onEdit(row.original)}
          onDelete={() => onDelete(row.original.name)}
          deleteDisabled={deleteDisabled}
        />
      ),
      enableHiding: false,
    },
  ]
}
