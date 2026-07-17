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
import { useTranslation } from 'react-i18next'

/**
 * Upstream model families NovaPura routes to.
 * Icons live in public/model-icons/ (lobehub icon pack).
 */
const PROVIDERS = [
  { name: 'OpenAI', icon: '/model-icons/openai.png' },
  { name: 'Claude', icon: '/model-icons/claude.png' },
  { name: 'Gemini', icon: '/model-icons/gemini.png' },
  { name: 'Grok', icon: '/model-icons/grok.png' },
  { name: 'DeepSeek', icon: '/model-icons/deepseek.png' },
  { name: 'Kimi', icon: '/model-icons/kimi.png' },
  { name: 'GLM', icon: '/model-icons/glm.png' },
  { name: 'MiniMax', icon: '/model-icons/minimax.png' },
  { name: 'Step', icon: '/model-icons/step.png' },
  { name: 'Qwen', icon: '/model-icons/qwen.png' },
  { name: 'Gemma', icon: '/model-icons/gemma.svg' },
  { name: 'Poolside', icon: '/model-icons/poolside.svg' },
  { name: 'Meta', icon: '/model-icons/meta.svg' },
  { name: 'Mistral', icon: '/model-icons/mistral.svg' },
  { name: 'Thinking Machines', icon: '/model-icons/thinkingmachine.png' },
] as const

export function ProviderStrip() {
  const { t } = useTranslation()
  // Two labelled copies so the -50% translate loops seamlessly; the second copy
  // is aria-hidden and carries a stable, unique key suffix.
  const track = [
    ...PROVIDERS.map((p) => ({ ...p, id: p.name, duplicate: false })),
    ...PROVIDERS.map((p) => ({ ...p, id: `${p.name}::dup`, duplicate: true })),
  ]

  return (
    <section className='border-y border-border px-5 py-10 sm:px-6 md:py-12'>
      <div className='mx-auto max-w-7xl'>
        <p className='np-kicker text-center'>
          {t('Route to the providers you already use')}
        </p>
        <div className='np-marquee-viewport mt-8 overflow-hidden'>
          <ul className='np-marquee-track flex w-max items-center gap-x-8 sm:gap-x-10'>
            {track.map((item) => (
              <li
                key={item.id}
                aria-hidden={item.duplicate}
                className='flex shrink-0 items-center gap-2.5'
              >
                <img
                  src={item.icon}
                  alt={item.duplicate ? '' : item.name}
                  width={28}
                  height={28}
                  decoding='async'
                  className='size-7 object-contain'
                />
                <span className='text-muted-foreground/90 text-sm font-semibold tracking-tight whitespace-nowrap'>
                  {item.name}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  )
}
