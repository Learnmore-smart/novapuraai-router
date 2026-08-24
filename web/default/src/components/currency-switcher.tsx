'use client'

import { useTranslation } from 'react-i18next'

import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  getUserCurrencyPreference,
  setUserCurrencyPreference,
  useCurrencyPreferenceVersion,
} from '@/lib/currency'
import { cn } from '@/lib/utils'
import {
  useSystemConfigStore,
  type CurrencyDisplayType,
} from '@/stores/system-config-store'

/**
 * The set of currencies a user may switch between locally (per browser).
 * Mirrors `ALLOWED_USER_OVERRIDES` in `lib/currency.ts`. Token / Custom
 * modes are admin-only and intentionally excluded.
 */
const USER_CURRENCY_OPTIONS: ReadonlyArray<{
  value: CurrencyDisplayType
  label: string
  symbol: string
}> = [
  { value: 'USD', label: 'USD', symbol: '$' },
  { value: 'CNY', label: 'CNY', symbol: '¥' },
  { value: 'CAD', label: 'CAD', symbol: 'C$' },
]

interface CurrencySwitcherProps {
  /** Optional className for the trigger. */
  className?: string
  /** Visual size of the trigger. */
  size?: 'sm' | 'default'
}

/**
 * Inline currency switcher. Reads/writes the user's per-browser preference
 * (localStorage) — admin's site-wide setting is the default; the user's
 * choice overrides it only for their own view.
 *
 * On change, bumps `useCurrencyPreferenceStore` so any component that calls
 * `useCurrencyPreferenceVersion()` re-renders and picks up the new currency.
 */
export function CurrencySwitcher({
  className,
  size = 'sm',
}: CurrencySwitcherProps) {
  const { t } = useTranslation()
  // Subscribe to preference changes so the trigger label stays in sync.
  useCurrencyPreferenceVersion()
  const adminDefault = useSystemConfigStore(
    (s) => s.config.currency.quotaDisplayType
  )

  const userPref = getUserCurrencyPreference()
  const fallback: CurrencyDisplayType =
    adminDefault === 'CAD' || adminDefault === 'CNY' || adminDefault === 'USD'
      ? adminDefault
      : 'USD'
  const effective: CurrencyDisplayType = userPref ?? fallback

  const handleChange = (next: string | null) => {
    if (next === 'USD' || next === 'CNY' || next === 'CAD') {
      setUserCurrencyPreference(next)
    }
  }

  return (
    <Select
      value={effective}
      onValueChange={handleChange}
      items={USER_CURRENCY_OPTIONS.map((opt) => ({
        value: opt.value,
        label: opt.label,
      }))}
    >
      <SelectTrigger
        size={size}
        className={cn('font-mono', className)}
        aria-label={t('Switch currency')}
      >
        <SelectValue placeholder={t('Currency')} />
      </SelectTrigger>
      <SelectContent alignItemWithTrigger={false}>
        <SelectGroup>
          {USER_CURRENCY_OPTIONS.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              <span className='font-mono'>{opt.symbol}</span>
              <span className='text-muted-foreground'>{opt.label}</span>
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  )
}
