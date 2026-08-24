import { Skeleton } from '@/components/ui/skeleton'

import { VIEW_MODES, type ViewMode } from '../constants'

export interface LoadingSkeletonProps {
  viewMode?: ViewMode
}

const CARD_SKELETONS = [
  'card-one',
  'card-two',
  'card-three',
  'card-four',
  'card-five',
  'card-six',
  'card-seven',
  'card-eight',
  'card-nine',
] as const

const FILTER_SKELETONS = [
  { id: 'pricing-type', width: 80 },
  { id: 'endpoint-type', width: 90 },
  { id: 'vendor', width: 75 },
  { id: 'group', width: 85 },
  { id: 'tag', width: 70 },
] as const

const TABLE_COLUMNS = [
  { id: 'model', width: 200 },
  { id: 'input', width: 100 },
  { id: 'output', width: 100 },
  { id: 'cache-read', width: 100 },
  { id: 'cache-write', width: 80 },
  { id: 'actions', width: 100 },
] as const

const TABLE_ROW_SKELETONS = [
  'row-one',
  'row-two',
  'row-three',
  'row-four',
  'row-five',
  'row-six',
  'row-seven',
  'row-eight',
  'row-nine',
  'row-ten',
] as const

const PAGINATION_SKELETONS = [
  'page-one',
  'page-two',
  'page-three',
  'page-four',
] as const

export function LoadingSkeleton(props: LoadingSkeletonProps) {
  const viewMode = props.viewMode ?? VIEW_MODES.CARD

  return (
    <div className='space-y-5'>
      <div className='space-y-1.5'>
        <Skeleton className='h-8 w-40' />
        <Skeleton className='h-4 w-52' />
      </div>
      <Skeleton className='h-10 w-full rounded-lg' />
      <FilterBarSkeleton />
      {viewMode === VIEW_MODES.TABLE ? (
        <TableContentSkeleton />
      ) : (
        <CardContentSkeleton />
      )}
    </div>
  )
}

function CardContentSkeleton() {
  return (
    <div className='grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3'>
      {CARD_SKELETONS.map((skeletonId) => (
        <div key={skeletonId} className='rounded-xl border p-5'>
          <div className='flex items-start justify-between gap-3'>
            <div className='flex min-w-0 items-start gap-3'>
              <Skeleton className='size-10 shrink-0 rounded-xl' />
              <div className='min-w-0 flex-1 space-y-2'>
                <Skeleton className='h-5 w-36' />
                <Skeleton className='h-3.5 w-48' />
              </div>
            </div>
            <Skeleton className='h-8 w-16 rounded-md' />
          </div>
          <div className='mt-4 space-y-2'>
            <Skeleton className='h-3.5 w-full' />
            <Skeleton className='h-3.5 w-4/5' />
          </div>
          <div className='mt-4 flex items-center gap-2'>
            <Skeleton className='h-4 w-24' />
            <Skeleton className='h-4 w-16' />
          </div>
          <div className='mt-2 flex items-center gap-3'>
            <Skeleton className='h-3.5 w-14' />
            <Skeleton className='h-3.5 w-14' />
            <Skeleton className='h-3.5 w-8' />
          </div>
        </div>
      ))}
    </div>
  )
}

function FilterBarSkeleton() {
  return (
    <div className='space-y-3'>
      <div className='flex items-center gap-3'>
        <div className='flex flex-1 flex-wrap items-center gap-2'>
          {FILTER_SKELETONS.map((skeleton) => (
            <Skeleton
              key={skeleton.id}
              className='h-8 rounded-lg'
              style={{ width: `${skeleton.width}px` }}
            />
          ))}
        </div>
        <div className='flex items-center gap-2'>
          <Skeleton className='h-8 w-24 rounded-lg' />
          <Skeleton className='h-8 w-20 rounded-lg' />
          <Skeleton className='h-8 w-24' />
          <Skeleton className='h-8 w-20 rounded-lg' />
        </div>
      </div>
      <Skeleton className='h-5 w-24' />
    </div>
  )
}

function TableContentSkeleton() {
  return (
    <div className='space-y-4'>
      <div className='overflow-hidden rounded-lg border'>
        <div className='bg-muted/30 border-b px-4 py-3'>
          <div className='flex items-center gap-4'>
            {TABLE_COLUMNS.map((column) => (
              <Skeleton
                key={column.id}
                className='h-4'
                style={{ width: `${column.width}px` }}
              />
            ))}
          </div>
        </div>
        {TABLE_ROW_SKELETONS.map((rowId) => (
          <div
            key={rowId}
            className='flex items-center gap-4 border-b px-4 py-3 last:border-b-0'
          >
            {TABLE_COLUMNS.map((column) => (
              <Skeleton
                key={`${rowId}-${column.id}`}
                className='h-5'
                style={{ width: `${column.width}px` }}
              />
            ))}
          </div>
        ))}
      </div>
      <div className='flex items-center justify-between'>
        <Skeleton className='h-5 w-32' />
        <div className='flex items-center gap-2'>
          {PAGINATION_SKELETONS.map((skeletonId) => (
            <Skeleton key={skeletonId} className='size-8' />
          ))}
        </div>
      </div>
    </div>
  )
}
