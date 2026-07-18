import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Switch } from '@/components/ui/switch'

import { resetPlanSubscriptions } from '../../api'
import { useSubscriptions } from '../subscriptions-provider'

export function ResetSubscriptionsDialog() {
  const { t } = useTranslation()
  const { open, setOpen, currentRow, triggerRefresh } = useSubscriptions()
  const [advanceResetTime, setAdvanceResetTime] = useState(true)
  const [resetting, setResetting] = useState(false)
  const isOpen = open === 'reset-subscriptions'
  const plan = currentRow?.plan
  const planLabel = plan?.title || (plan?.id ? `#${plan.id}` : '-')

  useEffect(() => {
    if (isOpen) setAdvanceResetTime(true)
  }, [isOpen])

  const handleConfirm = async () => {
    if (!plan?.id) return
    setResetting(true)
    try {
      const res = await resetPlanSubscriptions(plan.id, {
        advance_reset_time: advanceResetTime,
      })
      if (res.success) {
        toast.success(
          t('Reset {{count}} active subscriptions', {
            count: res.data?.reset_count || 0,
          })
        )
        triggerRefresh()
        setOpen(null)
      }
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setResetting(false)
    }
  }

  return (
    <ConfirmDialog
      open={isOpen}
      onOpenChange={(nextOpen) => !nextOpen && setOpen(null)}
      title={t('Reset subscription quota')}
      desc={t('Reset all active subscriptions under {{plan}}?', {
        plan: planLabel,
      })}
      confirmText={t('Reset quota')}
      handleConfirm={handleConfirm}
      disabled={!plan?.id}
      isLoading={resetting}
    >
      <label className='flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm'>
        <span>{t('Advance next reset time')}</span>
        <Switch
          checked={advanceResetTime}
          onCheckedChange={(checked) => setAdvanceResetTime(!!checked)}
          aria-label={t('Advance next reset time')}
        />
      </label>
    </ConfirmDialog>
  )
}
