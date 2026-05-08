import { useState, useRef, useEffect, useCallback } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Globe, LogOut, Monitor, Moon, PanelLeft, Pencil, RotateCcw, Search, Settings, SidebarClose, Star, Sun, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { i18n } from '../../i18n'
import { useAuth } from '../../lib/auth'
import { useTheme } from '../../theme/ThemeProvider'
import { usePageTabs } from '../../lib/page-tabs'
import { addQuickLink } from '../../lib/quick-links'
import { cn } from '../../lib/cn'

const iconBtn =
  'flex size-7 items-center justify-center rounded-md text-neutral-400 transition-all duration-150 hover:bg-neutral-500/[0.07] hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-700/50 dark:hover:text-zinc-300'

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
            <p className="text-xs text-neutral-400 dark:text-zinc-500">{user?.role}</p>
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

function PageTabBar() {
  const { tabs, activePath, close, closeOthers, reorder, rename, resetTitle } = usePageTabs()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const scrollRef = useRef<HTMLDivElement>(null)
  const dragIdx = useRef<number | null>(null)
  const [dropIdx, setDropIdx] = useState<number | null>(null)
  const [menu, setMenu] = useState<{ x: number; y: number; path: string; title: string; renamed?: boolean } | null>(null)

  const handleWheel = useCallback((e: React.WheelEvent) => {
    if (scrollRef.current) scrollRef.current.scrollLeft += e.deltaY
  }, [])

  useEffect(() => {
    if (!menu) return
    const closeMenu = () => setMenu(null)
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') closeMenu() }
    document.addEventListener('mousedown', closeMenu)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', closeMenu)
      document.removeEventListener('keydown', onKey)
    }
  }, [menu])

  function handleRename(path: string, currentTitle: string) {
    const next = window.prompt(t('pageTabs.renamePrompt'), currentTitle)
    if (next == null) return
    rename(path, next)
    setMenu(null)
  }

  if (tabs.length === 0) return null

  return (
    <div
      ref={scrollRef}
      onWheel={handleWheel}
      className="flex min-w-0 flex-1 items-end gap-0 overflow-x-auto scrollbar-none"
    >
      {tabs.map((tab, i) => {
        const isActive = tab.path === activePath
        const isDragOver = dropIdx === i && dragIdx.current !== null && dragIdx.current !== i
        return (
          <div
            key={tab.path}
            draggable
            onDragStart={(e) => {
              dragIdx.current = i
              e.dataTransfer.effectAllowed = 'move'
              e.dataTransfer.setData('text/plain', String(i))
            }}
            onDragOver={(e) => {
              e.preventDefault()
              e.dataTransfer.dropEffect = 'move'
              setDropIdx(i)
            }}
            onDragLeave={() => { if (dropIdx === i) setDropIdx(null) }}
            onDrop={(e) => {
              e.preventDefault()
              if (dragIdx.current !== null && dragIdx.current !== i) {
                reorder(dragIdx.current, i)
              }
              dragIdx.current = null
              setDropIdx(null)
            }}
            onDragEnd={() => { dragIdx.current = null; setDropIdx(null) }}
            onContextMenu={(e) => {
              e.preventDefault()
              e.stopPropagation()
              setMenu({ x: e.clientX, y: e.clientY, path: tab.path, title: tab.title, renamed: tab.renamed })
            }}
            className={cn(
              'group relative flex max-w-[180px] shrink-0 items-center gap-1.5 border-b-2 px-3 py-2 text-[12px] font-medium transition-colors select-none',
              isActive
                ? 'border-neutral-900 text-neutral-900 dark:border-zinc-200 dark:text-zinc-200'
                : 'border-transparent text-neutral-500 hover:text-neutral-700 dark:text-zinc-500 dark:hover:text-zinc-300',
              isDragOver && 'border-sky-400 dark:border-sky-500',
            )}
            onClick={() => navigate(tab.path)}
            title={tab.title}
          >
            <span className="truncate">{tab.title}</span>
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); close(tab.path) }}
              className={cn(
                'flex size-4 shrink-0 items-center justify-center rounded transition-colors',
                isActive
                  ? 'text-neutral-400 hover:bg-neutral-200 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-700'
                  : 'text-neutral-400 opacity-0 hover:bg-neutral-200 hover:text-neutral-600 group-hover:opacity-100 dark:text-zinc-500 dark:hover:bg-zinc-700',
              )}
            >
              <X className="size-3" strokeWidth={2} />
            </button>
          </div>
        )
      })}
      {menu && (
        <div
          className="fixed z-[90] w-48 overflow-hidden rounded-lg border border-neutral-200 bg-white py-1 shadow-xl dark:border-zinc-700 dark:bg-zinc-800"
          style={{ left: menu.x, top: menu.y }}
          onMouseDown={(e) => e.stopPropagation()}
        >
          <button
            type="button"
            onClick={() => { addQuickLink({ path: menu.path, title: menu.title }); setMenu(null) }}
            className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-neutral-700 transition-colors hover:bg-neutral-50 dark:text-zinc-300 dark:hover:bg-zinc-700"
          >
            <Star className="size-3.5" strokeWidth={1.8} />
            {t('pageTabs.addQuickLink')}
          </button>
          <button
            type="button"
            onClick={() => handleRename(menu.path, menu.title)}
            className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-neutral-700 transition-colors hover:bg-neutral-50 dark:text-zinc-300 dark:hover:bg-zinc-700"
          >
            <Pencil className="size-3.5" strokeWidth={1.8} />
            {t('pageTabs.rename')}
          </button>
          {menu.renamed && (
            <button
              type="button"
              onClick={() => { resetTitle(menu.path); setMenu(null) }}
              className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-neutral-700 transition-colors hover:bg-neutral-50 dark:text-zinc-300 dark:hover:bg-zinc-700"
            >
              <RotateCcw className="size-3.5" strokeWidth={1.8} />
              {t('pageTabs.resetName')}
            </button>
          )}
          <div className="my-1 border-t border-neutral-100 dark:border-zinc-700" />
          <button
            type="button"
            onClick={() => { closeOthers(menu.path); setMenu(null) }}
            className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-neutral-700 transition-colors hover:bg-neutral-50 dark:text-zinc-300 dark:hover:bg-zinc-700"
          >
            <SidebarClose className="size-3.5" strokeWidth={1.8} />
            {t('pageTabs.closeOthers')}
          </button>
        </div>
      )}
    </div>
  )
}

export function TopBar({
  onOpenSearch,
  collapsed,
  onToggleSidebar,
}: {
  onOpenSearch?: () => void
  collapsed?: boolean
  onToggleSidebar?: () => void
}) {
  const { t } = useTranslation()
  const { theme, cycleTheme } = useTheme()
  const ThemeIcon = theme === 'light' ? Sun : theme === 'dark' ? Moon : Monitor

  return (
    <header className="flex h-11 w-full shrink-0 items-center gap-2 border-b border-neutral-200/80 bg-white px-3 dark:border-zinc-700/60 dark:bg-zinc-900">
      {/* Sidebar toggle (when collapsed) */}
      {collapsed && onToggleSidebar && (
        <button
          type="button"
          onClick={onToggleSidebar}
          className="flex size-7 shrink-0 items-center justify-center rounded-md text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400"
          title={t('sidebar.expand')}
        >
          <PanelLeft className="size-4" strokeWidth={1.8} />
        </button>
      )}

      {/* Page tabs */}
      <PageTabBar />

      {/* Right controls */}
      <div className="flex shrink-0 items-center gap-1">
          <button
          type="button"
          onClick={onOpenSearch}
          className="flex h-7 w-40 max-w-[32vw] items-center gap-1.5 rounded-lg border border-neutral-200/80 bg-neutral-50/60 px-2 text-left text-[11px] text-neutral-400 transition-all duration-150 hover:border-neutral-300 dark:border-zinc-700/50 dark:bg-zinc-800/40 dark:text-zinc-500 dark:hover:border-zinc-600"
        >
          <Search className="size-3 shrink-0 opacity-50" strokeWidth={2} />
          <span className="flex-1 truncate">{t('search.placeholder')}</span>
          <kbd className="ml-auto hidden rounded border border-neutral-200 bg-neutral-100 px-1 py-px font-mono text-[9px] text-neutral-400 sm:inline dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-500">
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
