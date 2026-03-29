import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ExternalLink } from 'lucide-react'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { CommandPalette } from './CommandPalette'
import { recordVisit } from '../../lib/recent-visits'
import {
  navKeyFromPath,
  projectIdFromPath,
  projectNavKeyFromPath,
} from './nav-config'

export type BreadcrumbSegment = {
  label: string
  to?: string
}

function useBreadcrumbs(): BreadcrumbSegment[] {
  const { pathname } = useLocation()
  const { t } = useTranslation()
  const pid = projectIdFromPath(pathname)
  const pseg = projectNavKeyFromPath(pathname)

  if (pid && pseg) {
    return [
      { label: t('nav.projects'), to: '/projects' },
      { label: pid, to: `/projects/${encodeURIComponent(pid)}/tasks` },
      { label: t(`projectNav.${pseg}`) },
    ]
  }

  if (pid) {
    return [
      { label: t('nav.projects'), to: '/projects' },
      { label: pid },
    ]
  }

  if (pathname.startsWith('/teams/') && pathname !== '/teams') {
    const id = decodeURIComponent(pathname.split('/')[2] ?? '')
    return [
      { label: t('nav.teams'), to: '/teams' },
      { label: id },
    ]
  }

  const key = navKeyFromPath(pathname)
  return [{ label: t(`nav.${key}`) }]
}

const SIDEBAR_KEY = 'sidebar-collapsed'

export function AppShell() {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const crumbs = useBreadcrumbs()
  const [searchOpen, setSearchOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(SIDEBAR_KEY) === '1')

  function toggleSidebar() {
    setCollapsed((v) => {
      const next = !v
      localStorage.setItem(SIDEBAR_KEY, next ? '1' : '0')
      return next
    })
  }

  useEffect(() => {
    if (crumbs.length > 0) {
      const title = crumbs.map((c) => c.label).join(' / ')
      recordVisit(pathname, title)
    }
  }, [pathname, crumbs])

  return (
    <div className="flex h-dvh bg-neutral-50 text-neutral-900 dark:bg-zinc-950 dark:text-zinc-200">
      <Sidebar collapsed={collapsed} onToggle={toggleSidebar} />
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <TopBar breadcrumbs={crumbs} onOpenSearch={() => setSearchOpen(true)} collapsed={collapsed} onToggleSidebar={toggleSidebar} />
        <main className="flex-1 overflow-y-auto overflow-x-hidden">
          <Outlet />
        </main>
        <footer className="flex h-10 shrink-0 items-center border-t border-neutral-200/60 px-6 dark:border-zinc-800/50">
          <div className="flex items-center justify-between gap-4">
            <span className="text-xs font-medium text-neutral-400 dark:text-zinc-600">
              agencycli <span className="font-mono text-neutral-300 dark:text-zinc-700">v0.1</span>
            </span>
            <div className="flex items-center gap-3">
              <a
                href="https://github.com/chenhg5/agencycli/wiki"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 rounded-md px-2 py-0.5 text-xs text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-400"
              >
                <ExternalLink className="size-3" strokeWidth={1.8} />
                {t('footer.docs')}
              </a>
              <a
                href="https://github.com/chenhg5/agencycli"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 rounded-md px-2 py-0.5 text-xs text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-400"
              >
                <ExternalLink className="size-3" strokeWidth={1.8} />
                GitHub
              </a>
            </div>
          </div>
        </footer>
      </div>
      <CommandPalette open={searchOpen} onOpenChange={setSearchOpen} />
    </div>
  )
}
