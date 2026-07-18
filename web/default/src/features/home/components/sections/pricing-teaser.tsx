import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

const SAMPLE_MODELS = [
  { name: 'GPT-class models', note: 'Pay per token' },
  { name: 'Claude-class models', note: 'Pay per token' },
  { name: 'Gemini-class models', note: 'Pay per token' },
  { name: 'Open & regional models', note: 'Listed in catalogue' },
] as const

export function PricingTeaser() {
  const { t } = useTranslation()

  return (
    <section className='px-5 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-7xl'>
        <div className='flex flex-col justify-between gap-6 md:flex-row md:items-end'>
          <div className='max-w-xl'>
            <p className='np-kicker'>{t('Transparent pricing')}</p>
            <h2 className='mt-4 text-3xl leading-tight font-semibold tracking-[-0.03em] sm:text-4xl'>
              {t('See models and rates before you commit.')}
            </h2>
            <p className='text-muted-foreground mt-4 text-sm leading-7 sm:text-base'>
              {t(
                'Browse the model catalogue for live rates. Top up when you need capacity — no mandatory subscription tier to start.'
              )}
            </p>
          </div>
          <Button
            variant='outline'
            className='group h-11 w-fit shrink-0 rounded-md px-4'
            render={<Link to='/pricing' />}
          >
            {t('Open model catalogue')}
            <ArrowRight className='size-4 transition-transform group-hover:translate-x-0.5' />
          </Button>
        </div>

        <div className='np-surface mt-10 overflow-hidden'>
          <div className='border-border text-muted-foreground grid grid-cols-[1fr_auto] gap-4 border-b px-5 py-3 text-[0.6875rem] font-semibold tracking-wide uppercase sm:px-6'>
            <span>{t('Model family')}</span>
            <span>{t('Billing')}</span>
          </div>
          <ul>
            {SAMPLE_MODELS.map((row, index) => (
              <li
                key={row.name}
                className={
                  index < SAMPLE_MODELS.length - 1
                    ? 'border-border hover:bg-muted/40 grid grid-cols-[1fr_auto] items-center gap-4 border-b px-5 py-4 transition-colors sm:px-6'
                    : 'hover:bg-muted/40 grid grid-cols-[1fr_auto] items-center gap-4 px-5 py-4 transition-colors sm:px-6'
                }
              >
                <span className='flex items-center gap-3 text-sm font-medium'>
                  <span
                    className='bg-primary/70 size-1.5 shrink-0 rounded-full'
                    aria-hidden='true'
                  />
                  {t(row.name)}
                </span>
                <span className='text-muted-foreground font-mono text-xs'>
                  {t(row.note)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  )
}
