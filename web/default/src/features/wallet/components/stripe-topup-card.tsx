import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api } from '@/lib/api'

type Quote = {
  currency: string
  amount_major: number
  amount_minor: number
  paid_quota: number
  promo_quota: number
  total_quota: number
  paid_credit_micro_usd: number
  promo_credit_micro_usd: number
  display_label: string
  promotion_name?: string
}

type ConfigResp = {
  default_currency: string
  payment_methods_note: string
  sandbox: boolean
  api_balance: {
    total_quota: number
    promo_quota: number
    cash_quota: number
    label: string
    label_zh: string
  }
  config: {
    enabled: boolean
    presets: Record<string, number[]>
    min_max_major: Record<string, [number, number]>
    currencies: string[]
  }
}

const LS_CURRENCY = 'novapura.topup.currency'

export function StripeTopupCard() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [quoting, setQuoting] = useState(false)
  const [checkingOut, setCheckingOut] = useState(false)
  const [cfg, setCfg] = useState<ConfigResp | null>(null)
  const [currency, setCurrency] = useState('usd')
  const [amountMajor, setAmountMajor] = useState(10)
  const [custom, setCustom] = useState('')
  const [quote, setQuote] = useState<Quote | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const res = await api.get('/api/billing/top-up/config')
        if (cancelled) return
        const data = res.data?.data as ConfigResp
        setCfg(data)
        const saved =
          localStorage.getItem(LS_CURRENCY) || data?.default_currency || 'usd'
        setCurrency(saved)
        const presets = data?.config?.presets?.[saved] || [10]
        setAmountMajor(presets[0] ?? 10)
      } catch {
        // Stripe path may be disabled
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const presets = useMemo(
    () => cfg?.config?.presets?.[currency] || [],
    [cfg, currency]
  )
  const minMax = cfg?.config?.min_max_major?.[currency] || [1, 2000]

  const refreshQuote = useCallback(
    async (cur: string, major: number) => {
      setQuoting(true)
      try {
        const res = await api.post('/api/billing/top-up/quote', {
          currency: cur,
          amount_major: major,
        })
        if (res.data?.success) {
          setQuote(res.data.data as Quote)
        } else {
          setQuote(null)
          toast.error(res.data?.message || t('Unable to quote top-up'))
        }
      } catch (e: unknown) {
        setQuote(null)
        const msg =
          e && typeof e === 'object' && 'response' in e
            ? // eslint-disable-next-line @typescript-eslint/no-explicit-any
              (e as any).response?.data?.message
            : null
        toast.error(msg || t('Unable to quote top-up'))
      } finally {
        setQuoting(false)
      }
    },
    [t]
  )

  useEffect(() => {
    if (!cfg?.config?.enabled) return
    if (amountMajor > 0) {
      void refreshQuote(currency, amountMajor)
    }
  }, [cfg, currency, amountMajor, refreshQuote])

  const onCurrency = (v: string) => {
    setCurrency(v)
    localStorage.setItem(LS_CURRENCY, v)
    const p = cfg?.config?.presets?.[v] || [10]
    setAmountMajor(p[0] ?? 10)
    setCustom('')
  }

  const onCheckout = async () => {
    setCheckingOut(true)
    try {
      const res = await api.post('/api/billing/top-up/checkout', {
        currency,
        amount_major: amountMajor,
      })
      if (res.data?.success && res.data?.data?.checkout_url) {
        window.location.href = res.data.data.checkout_url
        return
      }
      toast.error(res.data?.message || t('Checkout failed'))
    } catch (e: unknown) {
      const msg =
        e && typeof e === 'object' && 'response' in e
          ? // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (e as any).response?.data?.message
          : null
      toast.error(msg || t('Checkout failed'))
    } finally {
      setCheckingOut(false)
    }
  }

  if (loading) {
    return null
  }
  if (!cfg?.config?.enabled) {
    return null
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {t('API Credits top-up')}
          {cfg.sandbox ? (
            <span className="ml-2 text-xs font-normal text-amber-600">
              Sandbox
            </span>
          ) : null}
        </CardTitle>
        <CardDescription>
          {t('Prepaid platform API credits. Non-withdrawable and non-transferable.')}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="text-sm text-muted-foreground">
          {t('Current API balance')}:{' '}
          <span className="font-medium text-foreground">
            {cfg.api_balance?.total_quota ?? 0}
          </span>{' '}
          ({cfg.api_balance?.label_zh || cfg.api_balance?.label})
        </div>

        <div className="space-y-2">
          <Label>{t('Currency')}</Label>
          <Select value={currency} onValueChange={onCurrency}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(cfg.config.currencies || ['usd', 'cny', 'cad']).map((c) => (
                <SelectItem key={c} value={c}>
                  {c.toUpperCase()}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-wrap gap-2">
          {presets.map((p) => (
            <Button
              key={p}
              type="button"
              size="sm"
              variant={amountMajor === p && !custom ? 'default' : 'outline'}
              onClick={() => {
                setCustom('')
                setAmountMajor(p)
              }}
            >
              {p}
            </Button>
          ))}
        </div>

        <div className="space-y-2">
          <Label>{t('Custom amount')}</Label>
          <Input
            type="number"
            min={minMax[0]}
            max={minMax[1]}
            value={custom}
            placeholder={`${minMax[0]} – ${minMax[1]}`}
            onChange={(e) => {
              setCustom(e.target.value)
              const n = parseInt(e.target.value, 10)
              if (!Number.isNaN(n)) setAmountMajor(n)
            }}
          />
        </div>

        {quote ? (
          <div className="rounded-md border p-3 text-sm space-y-1">
            <div>
              {t('You pay')}: <strong>{quote.display_label}</strong>
            </div>
            <div>
              {t('Base API credits')}: {quote.paid_quota}
            </div>
            <div>
              {t('Promotional credits')}: {quote.promo_quota}
              {quote.promotion_name ? ` (${quote.promotion_name})` : ''}
            </div>
            <div>
              {t('Total credits after payment')}:{' '}
              <strong>{quote.total_quota}</strong>
            </div>
          </div>
        ) : null}

        <p className="text-xs text-muted-foreground">
          {cfg.payment_methods_note ||
            t(
              'Payment methods shown at secure checkout vary by region and device.'
            )}
        </p>

        <Button
          className="w-full"
          disabled={checkingOut || quoting || !quote}
          onClick={() => void onCheckout()}
        >
          {checkingOut ? t('Redirecting…') : t('Pay with Stripe')}
        </Button>
      </CardContent>
    </Card>
  )
}
