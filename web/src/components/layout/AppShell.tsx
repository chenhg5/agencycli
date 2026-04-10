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
import AssistantWidget from '../assistant/AssistantWidget'
import { ToastContainer } from '../ui/Toast'

export type BreadcrumbSegment = {
  label: string
  to?: string
}

function useBreadcrumbs(): BreadcrumbSegment[] {
  const { pathname } = useLocation()
  const { t } = useTranslation()
  const pid = projectIdFromPath(pathname)
  const pseg = projectNavKeyFromPath(pathname)

  // Check for agent detail page: /projects/:id/members/:agentName
  const agentMatch = /^\/projects\/[^/]+\/members\/([^/]+)$/.exec(pathname)
  if (pid && agentMatch) {
    const agentName = decodeURIComponent(agentMatch[1])
    return [
      { label: t('nav.projects'), to: '/projects' },
      { label: pid, to: `/projects/${encodeURIComponent(pid)}/tasks` },
      { label: t('projectNav.members'), to: `/projects/${encodeURIComponent(pid)}/members` },
      { label: agentName },
    ]
  }

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

  if (pathname.startsWith('/people/') && pathname !== '/people') {
    const id = decodeURIComponent(pathname.split('/')[2] ?? '')
    return [
      { label: t('nav.people'), to: '/people' },
      { label: id },
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
    <nav className="flex h-10 min-w-0 shrink-0 items-center gap-1 border-b border-neutral-200/60 px-5 dark:border-zinc-700/40">
      <span className="mr-1 flex size-5 shrink-0 items-center justify-center rounded bg-sky-500/10 text-[10px] font-bold text-sky-600 dark:bg-sky-400/10 dark:text-sky-400">
        A
      </span>
      {crumbs.map((seg, i) => {
        const isLast = i === crumbs.length - 1
        return (
          <div key={`${seg.label}-${i}`} className="flex items-center gap-1">
            <ChevronRight
              className="size-3 shrink-0 text-neutral-300 dark:text-zinc-500"
              strokeWidth={2}
            />
            {seg.to && !isLast ? (
              <Link
                to={seg.to}
                className="truncate rounded-md px-1.5 py-0.5 text-[13px] text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-300"
              >
                {seg.label}
              </Link>
            ) : (
              <span className="truncate px-1.5 py-0.5 text-[13px] font-medium text-neutral-800 dark:text-zinc-200">
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
  const [updateInfo, setUpdateInfo] = useState<{ hasUpdate: boolean; latestVersion?: string } | null>(null)

  useEffect(() => {
    apiFetch<{ version: string }>('/api/v1/health')
      .then((d) => setAppVersion(d.version || 'dev'))
      .catch(() => setAppVersion('dev'))

    apiFetch<{ hasUpdate: boolean; latestVersion?: string }>('/api/v1/check-update')
      .then((d) => setUpdateInfo(d))
      .catch(() => {})
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
      <>
          <div className="flex h-dvh bg-neutral-50 text-neutral-900 dark:bg-zinc-950 dark:text-zinc-200">
          <Sidebar collapsed={collapsed} onToggle={toggleSidebar} />
          <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
            <TopBar onOpenSearch={() => setSearchOpen(true)} collapsed={collapsed} onToggleSidebar={toggleSidebar} />
            <Breadcrumbs crumbs={crumbs} />
            <main className="flex-1 overflow-y-auto overflow-x-hidden">
              <Outlet />
            </main>
            <footer className="flex h-10 w-full shrink-0 items-center justify-between border-t border-neutral-200/60 px-6 dark:border-zinc-700/50">
              <div className="flex items-center gap-3">
                <span className="text-xs font-medium text-neutral-400 dark:text-zinc-500">
                  agencycli <span className="font-mono text-neutral-300 dark:text-zinc-500">{appVersion}</span>
                </span>
                {updateInfo?.hasUpdate && updateInfo.latestVersion && (
                  <a
                    href={`https://github.com/chenhg5/agencycli/releases/tag/${updateInfo.latestVersion}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-[11px] font-medium text-blue-600 transition-colors hover:bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 dark:hover:bg-blue-900/50"
                  >
                    {t('footer.updateAvailable', { version: updateInfo.latestVersion })}
                  </a>
                )}
              </div>
              <div className="flex items-center gap-3">
                <a href="https://github.com/chenhg5/agencycli/wiki" target="_blank" rel="noopener noreferrer"
                  className="rounded-md px-2 py-0.5 text-xs text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300">
                  {t('footer.docs')}
                </a>
                <a href="https://github.com/chenhg5/agencycli" target="_blank" rel="noopener noreferrer"
                  className="rounded-md px-2 py-0.5 text-xs text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300">
                  GitHub
                </a>
              </div>
            </footer>
          </div>
          <CommandPalette open={searchOpen} onOpenChange={setSearchOpen} />
        </div>
        <AssistantWidget />
        <ToastContainer />
      </>
    </PageTabsProvider>
  )
}
