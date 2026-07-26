import { Loader2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { cn } from '@/lib/utils'

import { PAYOUT_CHANNEL_MANUAL, PAYOUT_CHANNEL_STRIPE_CONNECT } from '@/features/withdrawals/constants'

interface WithdrawalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (amountCents: number, payoutChannel: string) => Promise<boolean>
  balanceCents: number
  minWithdrawalCents: number
  freezeDays: number
  submitting: boolean
  /**
   * True when the user has a Stripe Connect account record on file. When false,
   * the payout-channel selector is hidden and the request falls back to the
   * legacy manual flow — preserving the original behavior for users who never
   * touched Stripe Connect (and for deployments where the feature is off).
   */
  stripeConnectStarted?: boolean
  /** True when the user's Stripe Connect account is fully enabled for payouts. */
  stripeConnectEnabled?: boolean
  /** Jump to the Stripe Connect onboarding card (closes this dialog). */
  onJumpToOnboarding?: () => void
}

export function WithdrawalDialog({
  open,
  onOpenChange,
  onConfirm,
  balanceCents,
  minWithdrawalCents,
  freezeDays,
  submitting,
  stripeConnectStarted = false,
  stripeConnectEnabled = false,
  onJumpToOnboarding,
}: WithdrawalDialogProps) {
  const { t } = useTranslation()
  const [amountUSD, setAmountUSD] = useState<number>(0)
  const [payoutChannel, setPayoutChannel] = useState<string>(
    PAYOUT_CHANNEL_MANUAL
  )

  // Default the input to the minimum withdrawable amount and reset the payout
  // channel to the safe manual default whenever the dialog reopens.
  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setAmountUSD(Math.min(minWithdrawalCents, balanceCents) / 100)
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setPayoutChannel(PAYOUT_CHANNEL_MANUAL)
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
    const success = await onConfirm(amountCents, payoutChannel)
    if (success) {
      onOpenChange(false)
    }
  }

  const handleJumpToOnboarding = () => {
    onOpenChange(false)
    onJumpToOnboarding?.()
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

        {stripeConnectStarted && (
          <div className='space-y-2'>
            <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
              {t('Payout Channel')}
            </Label>
            <RadioGroup
              value={payoutChannel}
              onValueChange={(value) => setPayoutChannel(value as string)}
              className='grid gap-2 sm:grid-cols-2'
            >
              <label
                htmlFor='payout-manual'
                className={cn(
                  'flex cursor-pointer items-center gap-2 rounded-lg border p-2.5 transition-colors hover:bg-muted',
                  payoutChannel === PAYOUT_CHANNEL_MANUAL && 'border-primary'
                )}
              >
                <RadioGroupItem
                  id='payout-manual'
                  value={PAYOUT_CHANNEL_MANUAL}
                />
                <span className='text-sm font-medium'>
                  {t('Manual transfer')}
                </span>
              </label>

              {stripeConnectEnabled ? (
                <label
                  htmlFor='payout-stripe'
                  className={cn(
                    'flex cursor-pointer items-center gap-2 rounded-lg border p-2.5 transition-colors hover:bg-muted',
                    payoutChannel === PAYOUT_CHANNEL_STRIPE_CONNECT &&
                      'border-primary'
                  )}
                >
                  <RadioGroupItem
                    id='payout-stripe'
                    value={PAYOUT_CHANNEL_STRIPE_CONNECT}
                  />
                  <span className='text-sm font-medium'>
                    {t('Stripe Connect')}
                  </span>
                </label>
              ) : (
                <TooltipProvider delay={0}>
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <label
                          htmlFor='payout-stripe'
                          className='flex cursor-not-allowed items-center gap-2 rounded-lg border p-2.5 opacity-60'
                        />
                      }
                    >
                      <RadioGroupItem
                        id='payout-stripe'
                        value={PAYOUT_CHANNEL_STRIPE_CONNECT}
                        disabled
                      />
                      <span className='text-sm font-medium'>
                        {t('Stripe Connect')}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      {t('Complete Stripe onboarding first')}
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )}
            </RadioGroup>

            {!stripeConnectEnabled && (
              <button
                type='button'
                onClick={handleJumpToOnboarding}
                className='text-primary text-xs underline underline-offset-2'
              >
                {t('Complete Stripe onboarding first')}
              </button>
            )}
          </div>
        )}
      </div>
    </Dialog>
  )
}
