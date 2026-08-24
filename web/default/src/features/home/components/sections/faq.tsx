import { HelpCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Markdown } from '@/components/ui/markdown'
import { Skeleton } from '@/components/ui/skeleton'
import { useFAQ } from '@/features/dashboard/hooks/use-status-data'
import type { FAQItem } from '@/features/dashboard/types'
import { toFAQTranslationLanguage } from '@/features/system-settings/content/faq-json'
import { formatPublicCurrency, getRegisterPromo } from '@/lib/public-status'
import { useStatus } from '@/hooks/use-status'

import { dedupeFAQItems } from '../../lib/home-content'

export function FAQ() {
  const { t, i18n } = useTranslation()
  const { items, loading } = useFAQ()
  const { status } = useStatus()
  const registerPromo = getRegisterPromo(status)
  const faqLanguage = toFAQTranslationLanguage(i18n.language)
  const uniqueItems = dedupeFAQItems(items)

  if (!loading && uniqueItems.length === 0) return null

  return (
    <section className='np-aperture-faq px-5 sm:px-6'>
      <div className='np-aperture-container grid gap-10 lg:grid-cols-[minmax(16rem,0.65fr)_minmax(0,1.35fr)] lg:gap-16'>
        <div className='lg:sticky lg:top-24 lg:self-start'>
          <span className='np-aperture-faq-icon'>
            <HelpCircle className='size-5' aria-hidden='true' />
          </span>
          <p className='np-kicker mt-5'>{t('FAQ')}</p>
          <h2 className='mt-4 text-3xl leading-tight font-semibold tracking-[-0.03em] sm:text-4xl'>
            {t('Questions, answered clearly.')}
          </h2>
          <p className='text-muted-foreground mt-4 max-w-md text-sm leading-7 sm:text-base'>
            {t(
              'Everything you need to know before creating a key, choosing a model, or adding credits.'
            )}
          </p>
          {registerPromo && (
            <p className='text-muted-foreground mt-4 max-w-md text-sm leading-7'>
              {t('New accounts receive {{amount}} in API credits.', {
                amount:
                  formatPublicCurrency(
                    registerPromo.amount,
                    registerPromo.currency
                  ) || '—',
              })}
            </p>
          )}
        </div>

        <div className='np-aperture-faq-panel overflow-hidden'>
          {loading ? (
            <div className='space-y-3 p-5 sm:p-6' aria-label={t('Loading FAQ')}>
              {[0, 1, 2].map((item) => (
                <Skeleton key={item} className='h-14 w-full rounded-md' />
              ))}
            </div>
          ) : (
            <Accordion className='w-full px-5 sm:px-6'>
              {uniqueItems.map((item: FAQItem, index: number) => {
                const key = item.id ?? `faq-${index}`
                const translation = item.translations?.[faqLanguage]
                const question = translation?.question ?? item.question
                const answer = translation?.answer ?? item.answer
                return (
                  <AccordionItem key={key} value={`item-${key}`}>
                    <AccordionTrigger className='py-5 text-start text-base font-semibold hover:no-underline'>
                      <Markdown>{question}</Markdown>
                    </AccordionTrigger>
                    <AccordionContent className='pb-5'>
                      <Markdown className='text-muted-foreground text-sm leading-7'>
                        {answer}
                      </Markdown>
                    </AccordionContent>
                  </AccordionItem>
                )
              })}
            </Accordion>
          )}
        </div>
      </div>
    </section>
  )
}
