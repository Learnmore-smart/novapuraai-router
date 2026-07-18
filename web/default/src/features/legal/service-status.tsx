import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { api } from '@/lib/api'

/** Public service status page (MVP section 1.2). Uses /api/status when available. */
export function ServiceStatusPage() {
  const { t } = useTranslation()
  const { data, isLoading, isError } = useQuery({
    queryKey: ['public-service-status'],
    queryFn: async () => {
      const res = await api.get('/api/status')
      return res.data
    },
    refetchInterval: 60_000,
  })

  const ok = data?.success === true

  return (
    <div className='mx-auto max-w-3xl px-4 py-10'>
      <Card>
        <CardHeader>
          <CardTitle>{t('Service Status')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-3 text-sm'>
          {isLoading && <p>{t('Checking status...')}</p>}
          {isError && (
            <p className='text-destructive'>
              {t('Unable to reach the API status endpoint.')}
            </p>
          )}
          {!isLoading && !isError && (
            <p>
              {ok
                ? t('All systems operational.')
                : t('The API reported an unhealthy status.')}
            </p>
          )}
          <p className='text-muted-foreground'>
            {t(
              'For detailed uptime history, configure Uptime Kuma integration in system settings.'
            )}
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
