import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import type { MultiKeyConfirmAction } from '../../types'

type MultiKeyTableRowActionsProps = {
  keyIndex: number
  status: number
  canDelete: boolean
  onAction: (action: MultiKeyConfirmAction) => void
}

export function MultiKeyTableRowActions({
  keyIndex,
  status,
  canDelete,
  onAction,
}: MultiKeyTableRowActionsProps) {
  const { t } = useTranslation()
  const isEnabled = status === 1

  return (
    <div className='flex justify-end gap-2'>
      {isEnabled ? (
        <Button
          variant='outline'
          size='sm'
          onClick={() => onAction({ type: 'disable', keyIndex })}
        >
          {t('Disable')}
        </Button>
      ) : (
        <Button
          variant='outline'
          size='sm'
          onClick={() => onAction({ type: 'enable', keyIndex })}
        >
          {t('Enable')}
        </Button>
      )}
      <Button
        variant='destructive'
        size='sm'
        onClick={() => {
          if (!canDelete) return
          onAction({ type: 'delete', keyIndex })
        }}
        disabled={!canDelete}
        title={
          canDelete ? undefined : t('No permission to perform this action')
        }
      >
        {t('Delete')}
      </Button>
    </div>
  )
}
