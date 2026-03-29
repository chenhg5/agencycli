import { useEffect, useState } from 'react'
import { Link, Outlet, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ChevronRight } from 'lucide-react'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { CommandPalette } from './CommandPalette'
import { PageTabsProvider } from '../../lib/page-tabs'
import { recordVisit } from '../../lib/recent-visits'
import { apiFetch } from '../../lib/api'
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

function Breadcrumbs({ crumbs }: { crumbs: BreadcrumbSegment[] }) {
  return (
    <nav className="flex min-w-0 items-center gap-0.5 overflow-hidden px-5 py-1.5">
      {crumbs.map((seg, i) => {
        const isLast = i === crumbs.length - 1
        return (
          <div key={`${seg.label}-${i}`} className="flex items-center gap-0.5">
            {i > 0 && (
              <ChevronRight
                className="mx-0.5 size-3 shrink-0 text-neutral-300 dark:text-zinc-700"
                strokeWidth={1.8}
              />
            )}
            {seg.to && !isLast ? (
              <Link
                to={seg.to}
                className="truncate rounded-sm px-1 py-0.5 text-[12px] font-medium text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-800 dark:text-zinc-500 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
              >
                {seg.label}
              </Link>
            ) : (
              <span className="truncate px-1 py-0.5 text-[12px] font-medium text-neutral-700 dark:text-zinc-300">
                {seg.label}
              </span>
            )}
          </div>
        )
      })}
    </nav>
  )
}

const SIDEBAR_KEY = 'sidebar-collapsed'

export function AppShell() {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const crumbs = useBreadcrumbs()
  const [searchOpen, setSearchOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(SIDEBAR_KEY) === '1')
  const [appVersion, setAppVersion] = useState('…')

  useEffect(() => {
    apiFetch<{ version: string }>('/api/v1/health')
      .then((d) => setAppVersion(d.version || 'dev'))
      .catch(() => setAppVersion('dev'))
  }, [])

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

  const pageTitle = crumbs.length > 0 ? crumbs[crumbs.length - 1].label : ''

  return (
    <PageTabsProvider pageTitle={pageTitle}>
      <div className="flex h-dvh bg-neutral-50 text-neutral-900 dark:bg-zinc-950 dark:text-zinc-200">
        <Sidebar collapsed={collapsed} onToggle={toggleSidebar} />
        <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
          <TopBar onOpenSearch={() => setSearchOpen(true)} collapsed={collapsed} onToggleSidebar={toggleSidebar} />
          <Breadcrumbs crumbs={crumbs} />
          <main className="flex-1 overflow-y-auto overflow-x-hidden">
            <Outlet />
          </main>
          <footer className="flex h-10 w-full shrink-0 items-center justify-between border-t border-neutral-200/60 px-6 dark:border-zinc-800/50">
            <span className="text-xs font-medium text-neutral-400 dark:text-zinc-600">
              agencycli <span className="font-mono text-neutral-300 dark:text-zinc-700">{appVersion}</span>
            </span>
            <div className="flex items-center gap-3">
              <a href="https://github.com/chenhg5/agencycli/wiki" target="_blank" rel="noopener noreferrer"
                className="rounded-md px-2 py-0.5 text-xs text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
                {t('footer.docs')}
              </a>
              <a href="https://github.com/chenhg5/agencycli" target="_blank" rel="noopener noreferrer"
                className="rounded-md px-2 py-0.5 text-xs text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
                GitHub
              </a>
            </div>
          </footer>
        </div>
        <CommandPalette open={searchOpen} onOpenChange={setSearchOpen} />
      </div>
    </PageTabsProvider>
  )
}
