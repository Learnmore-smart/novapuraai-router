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
  { name: 'GLM', icon: '/model-icons/glm.png' },
  { name: 'MiniMax', icon: '/model-icons/minimax.png' },
  { name: 'Step', icon: '/model-icons/step.png' },
  { name: 'Qwen', icon: '/model-icons/qwen.png' },
  { name: 'Gemma', icon: '/model-icons/gemma.svg' },
  { name: 'Poolside', icon: '/model-icons/poolside.svg' },
  { name: 'Meta', icon: '/model-icons/meta.svg' },
  { name: 'Mistral', icon: '/model-icons/mistral.svg' },
  { name: 'NVIDIA', icon: '/model-icons/nvidia-logo-horz.svg' },
  { name: 'Sarvam', icon: '/model-icons/sarvam-ai-logo.png' },
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
          <ul className='np-marquee-track flex w-max items-center'>
            {track.map((item) => (
              <li
                key={item.id}
                aria-hidden={item.duplicate}
                className='flex shrink-0 items-center gap-2.5 pr-8 sm:pr-10'
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
