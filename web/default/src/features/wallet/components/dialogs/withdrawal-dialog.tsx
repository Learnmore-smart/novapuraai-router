import { Loader2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { formatCurrencyFromUSD } from '@/lib/currency'

interface WithdrawalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (amountCents: number) => Promise<boolean>
  balanceCents: number
  minWithdrawalCents: number
  freezeDays: number
  submitting: boolean
}

export function WithdrawalDialog({
  open,
  onOpenChange,
  onConfirm,
  balanceCents,
  minWithdrawalCents,
  freezeDays,
  submitting,
}: WithdrawalDialogProps) {
  const { t } = useTranslation()
  const [amountUSD, setAmountUSD] = useState<number>(0)

  // Default the input to the minimum withdrawable amount.
  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setAmountUSD(Math.min(minWithdrawalCents, balanceCents) / 100)
    }
  }, [open, minWithdrawalCents, balanceCents])

  const amountCents = useMemo(
    () => Math.round(amountUSD * 100),
    [amountUSD]
  )

  const canSubmit =
    Number.isFinite(amountUSD) &&
    amountCents >= minWithdrawalCents &&
    amountCents <= balanceCents &&
    amountCents > 0

  const handleConfirm = async () => {
    if (!canSubmit) return
    const success = await onConfirm(amountCents)
    if (success) {
      onOpenChange(false)
    }
  }

  const balanceDisplay = formatCurrencyFromUSD(balanceCents / 100)
  const minDisplay = formatCurrencyFromUSD(minWithdrawalCents / 100)

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Withdraw Cash Commission')}
      description={t(
        'Submit a withdrawal request for admin review. Funds are debited now and refunded if rejected.'
      )}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'
      titleClassName='text-xl font-semibold'
      footerClassName='grid grid-cols-2 gap-2 sm:flex'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            {t('Cancel')}
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={submitting || !canSubmit}
          >
            {submitting && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {t('Submit Request')}
          </Button>
        </>
      }
    >
      <div className='space-y-4 py-3 sm:space-y-6 sm:py-4'>
        <div className='space-y-2'>
          <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
            {t('Withdrawable Balance')}
          </Label>
          <div className='text-2xl font-semibold tabular-nums'>
            {balanceDisplay}
          </div>
          <p className='text-muted-foreground text-xs'>
            {t(
              'This is withdrawable cash commission, not API usage quota.'
            )}
          </p>
        </div>

        <div className='space-y-3'>
          <Label
            htmlFor='withdrawal-amount'
            className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
          >
            {t('Withdrawal Amount (USD)')}
          </Label>
          <Input
            id='withdrawal-amount'
            type='number'
            value={amountUSD || ''}
            onChange={(e) => setAmountUSD(Number(e.target.value))}
            min={minWithdrawalCents / 100}
            max={balanceCents / 100}
            step={0.01}
            className='font-mono text-lg'
          />
          <p className='text-muted-foreground text-xs'>
            {t('Minimum:')} {minDisplay}
            {freezeDays > 0 && (
              <>
                {' · '}
                {t(
                  'Commission is frozen for {{days}} days after each recharge before becoming withdrawable.',
                  { days: freezeDays }
                )}
              </>
            )}
          </p>
        </div>
      </div>
    </Dialog>
  )
}
