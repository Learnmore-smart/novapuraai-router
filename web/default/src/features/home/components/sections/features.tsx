import { CircleDollarSign, Route } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface FeaturesProps {
  className?: string
}

const SUPPORTING = [
  {
    icon: Route,
    title: 'Health-aware routing',
    description:
      'Point at one model pool. NovaPura selects a healthy channel and surfaces failures when a route cannot complete.',
  },
  {
    icon: CircleDollarSign,
    title: 'Prepaid cost control',
    description:
      'Spend from balances you control. Promotional credit applies first, cash covers the rest. No subscription maze.',
  },
] as const

export function Features(_props: FeaturesProps) {
  const { t } = useTranslation()

  return (
    <section className='px-5 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-7xl'>
        <div className='mx-auto max-w-2xl text-center'>
          <p className='np-kicker'>{t('Why teams choose NovaPura')}</p>
          <h2 className='mt-4 text-3xl leading-tight font-semibold tracking-[-0.03em] sm:text-4xl'>
            {t('Infrastructure clarity without the integration tax.')}
          </h2>
          <p className='text-muted-foreground mt-4 text-sm leading-7 sm:text-base'>
            {t(
              'One gateway for routing, prepaid spend, and request-level visibility — so product teams ship models instead of plumbing.'
            )}
          </p>
        </div>

        <div className='mt-14 grid gap-5 lg:grid-cols-2'>
          {/* Primary — drop-in compatibility, with a concrete config snippet */}
          <article className='np-surface flex flex-col justify-between overflow-hidden p-7 sm:p-9'>
            <div>
              <p className='np-kicker'>{t('Drop-in compatibility')}</p>
              <h3 className='mt-3 text-xl font-semibold tracking-[-0.02em] sm:text-2xl'>
                {t('Keep your client. Change one base URL.')}
              </h3>
              <p className='text-muted-foreground mt-3 max-w-md text-sm leading-6'>
                {t(
                  'A familiar base URL, bearer token, and chat-completions route. Your existing OpenAI SDKs and tools keep working.'
                )}
              </p>
            </div>
            <div className='bg-muted/50 border-border mt-7 overflow-hidden rounded-md border font-mono text-xs'>
              <div className='text-muted-foreground flex items-center gap-2 px-4 py-2.5'>
                <span className='text-primary'>base_url</span>
                <span className='text-foreground/80 truncate'>
                  = "https://novapura.ai/v1"
                </span>
              </div>
              <div className='border-border text-muted-foreground border-t px-4 py-2.5'>
                <span className='text-primary'>api_key</span>{' '}
                <span className='text-foreground/80'>= "sk-••••••••"</span>
              </div>
            </div>
          </article>

          {/* Supporting — routing + cost */}
          <div className='grid gap-5 sm:grid-cols-2 lg:grid-cols-1'>
            {SUPPORTING.map((feature) => {
              const Icon = feature.icon
              return (
                <article
                  key={feature.title}
                  className='np-surface group flex flex-col p-7 transition-[border-color,box-shadow] hover:border-primary/25 hover:shadow-[var(--elevation-2)] sm:p-8'
                >
                  <span className='bg-muted text-foreground border-border flex size-10 items-center justify-center rounded-md border'>
                    <Icon className='size-5' aria-hidden='true' />
                  </span>
                  <h3 className='mt-5 text-lg font-semibold tracking-[-0.02em]'>
                    {t(feature.title)}
                  </h3>
                  <p className='text-muted-foreground mt-2.5 text-sm leading-6'>
                    {t(feature.description)}
                  </p>
                </article>
              )
            })}
          </div>
        </div>
      </div>
    </section>
  )
}
