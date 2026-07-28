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
import type { ImgHTMLAttributes } from 'react'

import { Logo } from '@/assets/logo'
import { DEFAULT_LOGO } from '@/lib/constants'
import { isDefaultBrandLogo } from '@/lib/dom-utils'
import { cn } from '@/lib/utils'

export type BrandMarkProps = {
  /** Configured logo URL from system settings; default monogram when empty/default */
  src?: string | null
  alt?: string
  className?: string
} & Omit<ImgHTMLAttributes<HTMLImageElement>, 'src' | 'alt' | 'className'>

/**
 * Chrome brand glyph used in header / sidebar / footer / auth.
 * Stock NovaPura paths render the SVG monogram (full-bleed, single radius).
 * Custom admin URLs still render as <img> with object-contain (no crop).
 */
export function BrandMark({
  src,
  alt = 'NovaPuraAI',
  className,
  ...imgProps
}: BrandMarkProps) {
  if (isDefaultBrandLogo(src)) {
    return (
      <Logo aria-label={alt} className={cn('size-full shrink-0', className)} />
    )
  }

  const resolvedSrc = src ?? DEFAULT_LOGO

  return (
    <img
      src={resolvedSrc}
      alt={alt}
      className={cn('size-full object-contain', className)}
      decoding='async'
      {...imgProps}
    />
  )
}
