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

import { cn } from '@/lib/utils'

/**
 * NovaPuraAI monogram — the raster brand mark (rounded copper→gold "N" tile),
 * matched to the favicon / app icon. Uses the small 256px PNG so it stays crisp
 * at chrome sizes without shipping the full 512px asset. object-contain keeps
 * the whole tile visible (no crop), and rounded-md softens the square PNG
 * corners so the monogram matches the rounded custom-logo box everywhere.
 */
export function Logo({
  className,
  alt = 'NovaPuraAI',
  ...props
}: ImgHTMLAttributes<HTMLImageElement>) {
  return (
    <img
      src='/logo-256.png'
      alt={alt}
      decoding='async'
      className={cn('size-6 shrink-0 rounded-md object-contain', className)}
      {...props}
    />
  )
}
