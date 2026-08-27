import { useId } from 'react'

import { cn } from '@/lib/utils'

/**
 * Filled silhouette of the existing NovaPuraAI monogram. Keeping the mark as
 * path data lets the routing signal use the same shape without relying on a
 * font glyph or raster asset.
 */
export const N_MARK_PATH =
  'M50 188C38 188 29 179 29 167V72C29 54 44 40 62 40C74 40 85 46 93 56L151 132V61C151 43 166 29 184 29C202 29 217 43 217 61V168C217 184 204 197 188 197C176 197 165 191 157 181L95 100V167C95 179 85 188 73 188H50Z'

interface NApertureProps {
  accessibleLabel: string
  className?: string
  isLoading?: boolean
  modelNames: readonly string[]
  routeLabel: string
}

export function NAperture(props: NApertureProps) {
  const rawId = useId()
  const idSuffix = rawId.replaceAll(/[^a-zA-Z0-9_-]/g, '')
  const clipId = `np-aperture-clip-${idSuffix}`
  const fillGradientId = `np-aperture-fill-${idSuffix}`
  const routeGradientId = `np-aperture-route-${idSuffix}`

  return (
    <div
      className={cn('np-aperture', props.className)}
      data-model-count={props.modelNames.length}
    >
      <svg
        className='np-aperture-mark'
        viewBox='0 0 246 226'
        role='img'
        aria-label={props.accessibleLabel}
      >
        <defs>
          <clipPath id={clipId}>
            <path d={N_MARK_PATH} />
          </clipPath>
          <linearGradient
            id={fillGradientId}
            x1='0%'
            y1='0%'
            x2='100%'
            y2='100%'
          >
            <stop offset='0%' stopColor='var(--np-aperture-coral)' />
            <stop offset='38%' stopColor='var(--np-aperture-solar)' />
            <stop offset='70%' stopColor='var(--np-aperture-coral)' />
            <stop offset='100%' stopColor='var(--np-aperture-violet)' />
          </linearGradient>
          <linearGradient
            id={routeGradientId}
            x1='0%'
            y1='0%'
            x2='100%'
            y2='0%'
          >
            <stop offset='0%' stopColor='var(--np-aperture-coral)' />
            <stop offset='52%' stopColor='var(--np-aperture-solar)' />
            <stop offset='100%' stopColor='var(--np-aperture-green)' />
          </linearGradient>
        </defs>

        <path
          d={N_MARK_PATH}
          fill='var(--np-aperture-ink)'
          fillOpacity='0.06'
        />
        <g clipPath={`url(#${clipId})`}>
          <rect
            x='0'
            y='0'
            width='246'
            height='226'
            fill={`url(#${fillGradientId})`}
          />
          <circle
            cx='82'
            cy='54'
            r='76'
            fill='var(--np-aperture-solar)'
            fillOpacity='0.92'
          />
          <circle
            cx='194'
            cy='180'
            r='86'
            fill='var(--np-aperture-violet)'
            fillOpacity='0.82'
          />
          <path
            className='np-aperture-route-line'
            d='M19 162C56 116 86 176 124 116S181 81 229 143'
            pathLength='1'
            fill='none'
            stroke={`url(#${routeGradientId})`}
            strokeWidth='5'
            strokeLinecap='round'
          />
          <circle cx='77' cy='119' r='7' fill='var(--np-aperture-green)' />
          <circle cx='158' cy='142' r='6' fill='var(--np-aperture-solar)' />
        </g>
        <path
          d={N_MARK_PATH}
          fill='none'
          stroke='var(--np-aperture-ink)'
          strokeOpacity='0.1'
          strokeWidth='1'
        />
      </svg>

      <ul className='np-aperture-routes' aria-label={props.accessibleLabel}>
        {props.modelNames.map((modelName) => (
          <li
            key={modelName}
            className='np-aperture-route-chip'
            title={modelName}
          >
            <span className='np-aperture-route-dot' aria-hidden='true' />
            <span className='truncate'>{modelName}</span>
            <span className='np-aperture-route-label'>{props.routeLabel}</span>
          </li>
        ))}
        {props.isLoading && props.modelNames.length === 0 && (
          <li className='np-aperture-route-chip np-aperture-route-chip-muted'>
            {props.routeLabel}
          </li>
        )}
      </ul>
    </div>
  )
}
