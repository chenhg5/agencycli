import { NavLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ChevronRight, PanelLeftClose, PanelLeft } from 'lucide-react'
import {
  projectIdFromPath,
  projectSubNav,
  workspaceNav,
  isNavActive,
} from './nav-config'
import { cn } from '../../lib/cn'
import { useAuth } from '../../lib/auth'

const linkBase =
  'group relative flex items-center gap-2.5 rounded-lg px-2.5 py-[7px] text-[13px] font-medium transition-all duration-150 outline-none select-none'
const linkIdle =
  'text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 dark:text-zinc-400 dark:hover:bg-zinc-800/70 dark:hover:text-zinc-100'
const linkActive =
  'bg-neutral-100 text-neutral-900 dark:bg-zinc-800/70 dark:text-zinc-100'

const linkCollapsed =
  'group relative flex items-center justify-center rounded-md p-2 transition-all duration-150 outline-none select-none'

const subLinkBase =
  'group relative flex items-center gap-2 rounded-lg py-[5px] pl-2.5 pr-2 text-[12.5px] font-medium transition-all duration-150 outline-none select-none'
const subLinkIdle =
  'text-neutral-500 hover:bg-neutral-100 hover:text-neutral-800 dark:text-zinc-500 dark:hover:bg-zinc-800/70 dark:hover:text-zinc-200'
const subActive =
  'bg-neutral-100 text-neutral-900 dark:bg-zinc-800/70 dark:text-zinc-100'

export function Sidebar({ collapsed, onToggle }: { collapsed: boolean; onToggle: () => void }) {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const { user } = useAuth()
  const isAdmin = !user || user.role === 'admin'
  const projectId = projectIdFromPath(pathname)

  return (
    <aside
      className={cn(
        'flex shrink-0 flex-col border-r border-neutral-200/80 bg-white transition-[width] duration-200 dark:border-zinc-700/60 dark:bg-zinc-900',
        collapsed ? 'w-[3.5rem]' : 'w-[14.5rem]',
      )}
    >
      {/* Brand */}
      <div className={cn('flex h-11 items-center border-b border-neutral-200/80 dark:border-zinc-700/60', collapsed ? 'justify-center px-2' : 'px-4')}>
        {collapsed ? (
          <span
            className="text-lg font-bold tracking-tight text-sky-600 dark:text-sky-400"
            style={{ fontFamily: "'Space Grotesk', sans-serif" }}
          >
            A
          </span>
        ) : (
          <span
            className="text-[17px] font-bold tracking-tight text-neutral-900 dark:text-zinc-100"
            style={{ fontFamily: "'Space Grotesk', sans-serif" }}
          >
            Agency<span className="text-sky-600 dark:text-sky-400">Cli</span>
          </span>
        )}
      </div>

      <nav
        className={cn('flex flex-1 flex-col gap-0.5 overflow-y-auto py-3', collapsed ? 'px-1.5' : 'px-2')}
        aria-label={t('aria.mainNavigation')}
      >
        {workspaceNav.filter(item => !item.adminOnly || isAdmin).map(({ to, navKey, icon: Icon, activePrefix }) => {
          const end = to === '/'
          let active = isNavActive(pathname, to, end, activePrefix)
          if (navKey === 'projects' && projectId) {
            active = true
          }
          const showProjectNest =
            !collapsed && navKey === 'projects' && projectId !== null && projectId !== ''

          if (collapsed) {
            return (
              <NavLink
                key={to}
                to={to}
                end={end}
                className={cn(
                  linkCollapsed,
                  active
                    ? 'bg-neutral-100 text-neutral-900 dark:bg-zinc-800/70 dark:text-zinc-100'
                    : 'text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 dark:text-zinc-500 dark:hover:bg-zinc-800/70 dark:hover:text-zinc-100',
                )}
                title={t(`nav.${navKey}`)}
              >
                <Icon className="size-4.5 shrink-0" strokeWidth={1.8} />
              </NavLink>
            )
          }

          return (
            <div key={to}>
              <NavLink
                to={to}
                end={end}
                className={cn(linkBase, active ? linkActive : linkIdle)}
              >
                <Icon className="size-4 shrink-0 opacity-80" strokeWidth={1.8} />
                <span className="flex-1">{t(`nav.${navKey}`)}</span>
                {showProjectNest && (
                  <ChevronRight className="size-3 opacity-40 transition-transform duration-150 rotate-90" strokeWidth={2} />
                )}
              </NavLink>

              {showProjectNest && (
                <div className="mt-0.5 ml-[18px] border-l border-neutral-200/70 pl-2.5 dark:border-zinc-700/60 animate-fade-in">
                  <p className="truncate px-2.5 py-1.5 text-[10.5px] font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
                    {projectId}
                  </p>
                  <div className="space-y-px">
                    {projectSubNav.filter(item => !item.adminOnly || isAdmin).map(({ segment, icon: SubIcon }) => {
                      const subTo = `/projects/${encodeURIComponent(projectId)}/${segment}`
                      const subActiveState =
                        pathname === subTo || pathname.startsWith(`${subTo}/`)
                      return (
                        <NavLink
                          key={segment}
                          to={subTo}
                          className={cn(
                            subLinkBase,
                            subActiveState ? subActive : subLinkIdle,
                          )}
                        >
                          <SubIcon className="size-3.5 shrink-0 opacity-75" strokeWidth={1.8} />
                          <span>{t(`projectNav.${segment}`)}</span>
                        </NavLink>
                      )
                    })}
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </nav>

      {/* Collapse / expand toggle at bottom */}
      <div className={cn('flex h-10 shrink-0 items-center border-t border-neutral-200/60 dark:border-zinc-700/50', collapsed ? 'justify-center px-1.5' : 'px-2')}>
        <button
          type="button"
          onClick={onToggle}
          className={cn(
            'flex w-full items-center rounded-md transition-colors hover:bg-neutral-100 dark:hover:bg-zinc-800',
            collapsed
              ? 'justify-center p-1.5 text-neutral-400 dark:text-zinc-500'
              : 'gap-2.5 px-2.5 py-[5px] text-[13px] font-medium text-neutral-500 dark:text-zinc-500',
          )}
          title={collapsed ? t('sidebar.expand') : t('sidebar.collapse')}
        >
          {collapsed
            ? <PanelLeft className="size-4" strokeWidth={1.8} />
            : <>
                <PanelLeftClose className="size-4 shrink-0 opacity-80" strokeWidth={1.8} />
                <span>{t('sidebar.collapse')}</span>
              </>
          }
        </button>
      </div>
    </aside>
  )
}
