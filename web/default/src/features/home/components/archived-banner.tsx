/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { Archive, ArrowUpRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { GitHubIcon } from '@/components/icons/github-icon'

const GITHUB_REPO_URL = 'https://github.com/Learnmore-smart/novapuraai-router'

export function ArchivedBanner() {
  const { t } = useTranslation()

  return (
    <aside
      aria-label={t('Archived')}
      className='border-amber-500/25 bg-amber-500/10 text-amber-950 dark:border-amber-500/30 dark:bg-amber-950/40 dark:text-amber-200 relative z-30 border-b pt-[var(--app-header-height)]'
    >
      <div className='mx-auto flex max-w-7xl flex-col items-center justify-between gap-3 px-4 py-3 sm:flex-row sm:px-6 sm:py-3.5'>
        <div className='flex flex-wrap items-center justify-center gap-2.5 text-center sm:justify-start sm:text-left'>
          <span className='bg-amber-500/20 text-amber-900 dark:bg-amber-500/30 dark:text-amber-200 inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold uppercase tracking-wider'>
            <Archive className='size-3.5' aria-hidden='true' />
            {t('Archived')}
          </span>
          <p className='text-xs font-medium sm:text-sm'>
            {t(
              'This project is now archived. The service has stopped working.'
            )}
          </p>
        </div>

        <a
          href={GITHUB_REPO_URL}
          target='_blank'
          rel='noopener noreferrer'
          className='border-amber-500/30 bg-amber-500/15 text-amber-950 hover:bg-amber-500/25 dark:border-amber-500/40 dark:bg-amber-400/15 dark:text-amber-100 dark:hover:bg-amber-400/25 inline-flex shrink-0 items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium shadow-xs transition-colors'
        >
          <GitHubIcon className='size-3.5' />
          <span>{t('View on GitHub')}</span>
          <ArrowUpRight className='size-3.5' aria-hidden='true' />
        </a>
      </div>
    </aside>
  )
}
