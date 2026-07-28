import { KeyRound, Send, UserPlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'

const STEPS = [
  {
    icon: UserPlus,
    title: 'Create your account',
    description:
      'Sign up, complete verification, and open a prepaid workspace. Cash and promotional credit stay separate.',
  },
  {
    icon: KeyRound,
    title: 'Generate an API key',
    description:
      'Create a token, pick a model from the catalogue, and keep spend limits under your control.',
  },
  {
    icon: Send,
    title: 'Send your first request',
    description:
      'Point your OpenAI-compatible client at NovaPura. Inspect model, tokens, latency, and cost in the ledger.',
  },
] as const

export function HowItWorks() {
  const { t } = useTranslation()

  return (
    <section className='border-border bg-muted/25 border-y px-5 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-7xl'>
        <div className='mx-auto max-w-2xl text-center'>
          <p className='np-kicker'>{t('From signup to first response')}</p>
          <h2 className='mt-4 text-3xl leading-tight font-semibold tracking-[-0.03em] sm:text-4xl'>
            {t('Three steps. One endpoint. Clear billing.')}
          </h2>
          <p className='text-muted-foreground mt-4 text-sm leading-7 sm:text-base'>
            {t(
              'The shortest path through NovaPura follows the product: account, key, request.'
            )}
          </p>
        </div>

        <ol className='mt-14 grid gap-5 md:grid-cols-3'>
          {STEPS.map((step, index) => {
            const Icon = step.icon
            return (
              <li
                key={step.title}
                className='np-surface relative flex flex-col p-7 sm:p-8'
              >
                <div className='flex items-center justify-between gap-3'>
                  <span className='bg-background text-primary border-border flex size-11 items-center justify-center rounded-md border'>
                    <Icon className='size-5' aria-hidden='true' />
                  </span>
                  <span className='text-muted-foreground/40 font-mono text-3xl font-semibold tabular-nums'>
                    0{index + 1}
                  </span>
                </div>
                <h3 className='mt-6 text-base font-semibold'>
                  {t(step.title)}
                </h3>
                <p className='text-muted-foreground mt-2 text-sm leading-6'>
                  {t(step.description)}
                </p>
              </li>
            )
          })}
        </ol>
      </div>
    </section>
  )
}
