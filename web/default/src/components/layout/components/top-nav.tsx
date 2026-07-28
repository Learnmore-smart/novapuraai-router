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
import { Link, useRouterState } from '@tanstack/react-router'
import { Menu } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'

import { type TopNavLink } from '../types'

type TopNavProps = React.HTMLAttributes<HTMLElement> & {
  links: TopNavLink[]
}

/**
 * 顶部导航栏组件
 * 在大屏幕显示水平导航，在小屏幕显示下拉菜单
 */
export function TopNav({ className, links, ...props }: TopNavProps) {
  const { t } = useTranslation()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  // 规范化链接，并根据当前路由计算激活态（与落地页导航保持一致）
  const normalizedLinks = useMemo(
    () =>
      links.map((link) => ({
        disabled: false,
        external: false,
        ...link,
        isActive:
          pathname === link.href ||
          (link.href !== '/' && pathname.startsWith(link.href)),
      })),
    [links, pathname]
  )

  return (
    <>
      {/* 移动端下拉菜单 */}
      <div className='lg:hidden'>
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger
            render={<Button size='icon' variant='outline' className='size-7' />}
          >
            <Menu />
          </DropdownMenuTrigger>
          <DropdownMenuContent side='bottom' align='start'>
            {normalizedLinks.map(
              ({ title, href, isActive, disabled, external }) => (
                <DropdownMenuItem
                  key={`${title}-${href}`}
                  render={
                    external ? (
                      <a
                        href={href}
                        target='_blank'
                        rel='noopener noreferrer'
                        className={!isActive ? 'text-muted-foreground' : ''}
                      >
                        {t(title)}
                      </a>
                    ) : (
                      <Link
                        to={href}
                        className={!isActive ? 'text-muted-foreground' : ''}
                        disabled={disabled}
                      >
                        {t(title)}
                      </Link>
                    )
                  }
                ></DropdownMenuItem>
              )
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* 桌面端水平导航 —— 与落地页导航同款胶囊样式 */}
      <nav
        className={cn('hidden items-center gap-0.5 lg:flex', className)}
        {...props}
      >
        {normalizedLinks.map(
          ({ title, href, isActive, disabled, external }) => {
            const linkClassName = cn(
              'rounded-md px-3 py-2 text-[13px] font-medium transition-colors',
              isActive
                ? 'bg-muted text-foreground'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/60',
              disabled && 'pointer-events-none opacity-50'
            )
            return external ? (
              <a
                key={`${title}-${href}`}
                href={href}
                target='_blank'
                rel='noopener noreferrer'
                className={linkClassName}
              >
                {t(title)}
              </a>
            ) : (
              <Link
                key={`${title}-${href}`}
                to={href}
                disabled={disabled}
                className={linkClassName}
              >
                {t(title)}
              </Link>
            )
          }
        )}
      </nav>
    </>
  )
}
