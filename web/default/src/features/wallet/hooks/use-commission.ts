import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import { toast } from 'sonner'

import {
  getCommissionSummary,
  getMyWithdrawals,
  requestWithdrawal,
} from '@/features/withdrawals/api'
import type {
  CommissionSummary,
  WithdrawalRequest,
} from '@/features/withdrawals/types'

// ============================================================================
// Commission Hook
// ============================================================================

const COMMISSION_PAGE_SIZE = 20

export function useCommission(enabled: boolean) {
  const [summary, setSummary] = useState<CommissionSummary | null>(null)
  const [loading, setLoading] = useState(false)
  const [withdrawing, setWithdrawing] = useState(false)
  const [history, setHistory] = useState<WithdrawalRequest[]>([])
  const [historyTotal, setHistoryTotal] = useState(0)
  const [historyPage, setHistoryPage] = useState(1)
  const [historyLoading, setHistoryLoading] = useState(false)

  const fetchSummary = useCallback(async () => {
    if (!enabled) {
      setSummary(null)
      return
    }
    try {
      setLoading(true)
      const response = await getCommissionSummary()
      if (response.success && response.data) {
        setSummary(response.data)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch commission summary:', error)
    } finally {
      setLoading(false)
    }
  }, [enabled])

  const fetchHistory = useCallback(
    async (page: number) => {
      if (!enabled) {
        setHistory([])
        setHistoryTotal(0)
        return
      }
      try {
        setHistoryLoading(true)
        const response = await getMyWithdrawals(page, COMMISSION_PAGE_SIZE)
        if (response.success && response.data) {
          setHistory(response.data.items ?? [])
          setHistoryTotal(response.data.total ?? 0)
        }
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error('Failed to fetch withdrawal history:', error)
      } finally {
        setHistoryLoading(false)
      }
    },
    [enabled]
  )

  const withdraw = useCallback(
    async (amountCents: number, payoutChannel?: string): Promise<boolean> => {
      if (!enabled) return false
      try {
        setWithdrawing(true)
        const response = await requestWithdrawal(amountCents, payoutChannel)
        if (response.success) {
          toast.success(
            response.message || i18next.t('Withdrawal request submitted')
          )
          await fetchSummary()
          await fetchHistory(1)
          setHistoryPage(1)
          return true
        }
        toast.error(response.message || i18next.t('Withdrawal request failed'))
        return false
      } catch {
        toast.error(i18next.t('Withdrawal request failed'))
        return false
      } finally {
        setWithdrawing(false)
      }
    },
    [enabled, fetchSummary, fetchHistory]
  )

  useEffect(() => {
    fetchSummary()
  }, [fetchSummary])

  useEffect(() => {
    fetchHistory(historyPage)
  }, [fetchHistory, historyPage])

  return {
    summary,
    loading,
    withdrawing,
    history,
    historyTotal,
    historyPage,
    historyLoading,
    setHistoryPage,
    fetchSummary,
    fetchHistory,
    withdraw,
  }
}
