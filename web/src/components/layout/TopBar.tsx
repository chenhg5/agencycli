import { useState, useRef, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight, Globe, LogOut, Monitor, Moon, PanelLeft, Search, Settings, Sun } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { i18n } from '../../i18n'
import { useAuth } from '../../lib/auth'
import { useTheme } from '../../theme/ThemeProvider'
import type { BreadcrumbSegment } from './AppShell'

const iconBtn =
  'flex size-7 items-center justify-center rounded-md text-neutral-400 transition-all duration-150 hover:bg-neutral-500/[0.07] hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-white/[0.06] dark:hover:text-zinc-300'

const languages = [
  { code: 'en', label: 'English' },
  { code: 'zh-CN', label: '简体中文' },
  { code: 'zh-TW', label: '繁體中文' },
  { code: 'ja', label: '日本語' },
] as const

function currentLang(): string {
  const lng = i18n.language
  if (lng.startsWith('zh-TW') || lng === 'zh-Hant') return 'zh-TW'
  if (lng.startsWith('zh')) return 'zh-CN'
  if (lng.startsWith('ja')) return 'ja'
  return 'en'
}

function LanguageDropdown() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const lang = currentLang()

  useEffect(() => {
    if (!open) return
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [open])

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={iconBtn}
        aria-label={t('language.label')}
        title={t('language.label')}
      >
        <Globe className="size-3.5" strokeWidth={1.8} />
      </button>
      {open && (
        <div className="absolute right-0 top-full z-50 mt-1.5 w-36 rounded-lg border border-neutral-200 bg-white py-1 shadow-lg dark:border-zinc-700 dark:bg-zinc-800">
          {languages.map((l) => (
            <button
              key={l.code}
              type="button"
              onClick={() => { void i18n.changeLanguage(l.code); setOpen(false) }}
              className={`flex w-full items-center px-3 py-1.5 text-left text-sm transition-colors ${
                lang === l.code
                  ? 'bg-sky-50 font-medium text-sky-700 dark:bg-sky-900/20 dark:text-sky-400'
                  : 'text-neutral-700 hover:bg-neutral-50 dark:text-zinc-300 dark:hover:bg-zinc-700'
              }`}
            >
              {l.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function UserMenu() {
  const { t } = useTranslation()
  const { user, logout } = useAuth()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [open])

  const initial = (user?.username ?? 'U')[0].toUpperCase()

  return (
    <div ref={ref} className="relative ml-1">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex size-7 items-center justify-center rounded-full bg-gradient-to-br from-sky-400 to-sky-600 text-xs font-bold text-white ring-2 ring-sky-200/40 transition-shadow hover:ring-sky-300/60 dark:from-sky-500 dark:to-sky-700 dark:ring-sky-800/40"
        title={user?.username}
      >
        {initial}
      </button>
      {open && (
        <div className="absolute right-0 top-full z-50 mt-1.5 w-44 rounded-lg border border-neutral-200 bg-white py-1 shadow-lg dark:border-zinc-700 dark:bg-zinc-800">
          <div className="border-b border-neutral-100 px-3 py-2 dark:border-zinc-700">
            <p className="text-sm font-medium text-neutral-900 dark:text-zinc-100">{user?.username}</p>
            <p className="text-xs text-neutral-400 dark:text-zinc-600">{user?.role}</p>
          </div>
          <Link
            to="/settings"
            onClick={() => setOpen(false)}
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-neutral-700 transition-colors hover:bg-neutral-50 dark:text-zinc-300 dark:hover:bg-zinc-700"
          >
            <Settings className="size-3.5" strokeWidth={1.8} />
            {t('nav.settings')}
          </Link>
          <button
            type="button"
            onClick={() => { logout(); setOpen(false) }}
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
          >
            <LogOut className="size-3.5" strokeWidth={1.8} />
            {t('auth.logout')}
          </button>
        </div>
      )}
    </div>
  )
}

export function TopBar({
  breadcrumbs,
  onOpenSearch,
  collapsed,
  onToggleSidebar,
}: {
  breadcrumbs: BreadcrumbSegment[]
  onOpenSearch?: () => void
  collapsed?: boolean
  onToggleSidebar?: () => void
}) {
  const { t } = useTranslation()
  const { theme, cycleTheme } = useTheme()
  const ThemeIcon = theme === 'light' ? Sun : theme === 'dark' ? Moon : Monitor

  return (
    <header className="flex h-11 w-full shrink-0 items-center justify-between gap-4 border-b border-neutral-200/80 bg-white px-5 dark:border-zinc-800/60 dark:bg-zinc-950">
      {/* Left */}
      <nav className="flex min-w-0 items-center gap-0.5 overflow-hidden">
        {collapsed && onToggleSidebar && (
          <button
            type="button"
            onClick={onToggleSidebar}
            className="mr-2 flex size-7 items-center justify-center rounded-md text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-400"
            title={t('sidebar.expand')}
          >
            <PanelLeft className="size-4" strokeWidth={1.8} />
          </button>
        )}
        {breadcrumbs.map((seg, i) => {
          const isLast = i === breadcrumbs.length - 1
          return (
            <div key={`${seg.label}-${i}`} className="flex items-center gap-0.5">
              {i > 0 && (
                <ChevronRight
                  className="mx-0.5 size-3.5 shrink-0 text-neutral-300 dark:text-zinc-700"
                  strokeWidth={1.8}
                />
              )}
              {seg.to && !isLast ? (
                <Link
                  to={seg.to}
                  className="truncate rounded-sm px-1.5 py-0.5 text-[13px] font-medium text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-800 dark:text-zinc-500 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
                >
                  {seg.label}
                </Link>
              ) : (
                <span className="truncate px-1.5 py-0.5 text-[13px] font-medium text-neutral-900 dark:text-zinc-100">
                  {seg.label}
                </span>
              )}
            </div>
          )
        })}
      </nav>

      {/* Right controls */}
      <div className="flex items-center gap-1">
        <button
          type="button"
          onClick={onOpenSearch}
          className="flex h-7 w-40 max-w-[32vw] items-center gap-1.5 rounded-md border border-neutral-200/80 bg-neutral-50/60 px-2 text-left text-[11px] text-neutral-400 transition-all duration-150 hover:border-neutral-300 dark:border-zinc-800/50 dark:bg-zinc-900/30 dark:text-zinc-600 dark:hover:border-zinc-700"
        >
          <Search className="size-3 shrink-0 opacity-50" strokeWidth={2} />
          <span className="flex-1 truncate">{t('search.placeholder')}</span>
          <kbd className="ml-auto hidden rounded border border-neutral-200 bg-neutral-100 px-1 py-px font-mono text-[9px] text-neutral-400 sm:inline dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-600">
            ⌘K
          </kbd>
        </button>
        <LanguageDropdown />
        <button
          type="button"
          onClick={cycleTheme}
          className={iconBtn}
          aria-label={t('theme.cycle')}
          title={`${t('theme.cycle')}: ${t(`theme.${theme}`)}`}
        >
          <ThemeIcon className="size-3.5" strokeWidth={1.8} />
        </button>
        <UserMenu />
      </div>
    </header>
  )
}
