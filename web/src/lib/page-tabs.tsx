import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

export type PageTab = {
  path: string
  title: string
  originalTitle?: string
  renamed?: boolean
}

type PageTabsContextValue = {
  tabs: PageTab[]
  activePath: string
  close: (path: string) => void
  closeOthers: (path: string) => void
  closeAll: () => void
  reorder: (fromIndex: number, toIndex: number) => void
  rename: (path: string, title: string) => void
  resetTitle: (path: string) => void
}

const Ctx = createContext<PageTabsContextValue>({
  tabs: [],
  activePath: '/',
  close: () => {},
  closeOthers: () => {},
  closeAll: () => {},
  reorder: () => {},
  rename: () => {},
  resetTitle: () => {},
})

const STORAGE_KEY = 'page-tabs'
const MAX_TABS = 12

function loadTabs(): PageTab[] {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (raw) return JSON.parse(raw) as PageTab[]
  } catch { /* ignore */ }
  return []
}

function saveTabs(tabs: PageTab[]) {
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(tabs))
}

export function PageTabsProvider({ children, pageTitle }: { children: ReactNode; pageTitle?: string }) {
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const [tabs, setTabs] = useState<PageTab[]>(loadTabs)

  useEffect(() => {
    saveTabs(tabs)
  }, [tabs])

  const addOrActivate = useCallback(
    (path: string, title: string) => {
      setTabs((prev) => {
        const idx = prev.findIndex((t) => t.path === path)
        if (idx >= 0) {
          if (prev[idx].renamed) {
            const next = [...prev]
            next[idx] = { ...next[idx], originalTitle: title }
            return next
          }
          if (prev[idx].title !== title) {
            const next = [...prev]
            next[idx] = { ...next[idx], title, originalTitle: title }
            return next
          }
          return prev
        }
        const next = [...prev, { path, title, originalTitle: title }]
        if (next.length > MAX_TABS) next.shift()
        return next
      })
    },
    [],
  )

  useEffect(() => {
    if (pageTitle) addOrActivate(pathname, pageTitle)
  }, [pathname, pageTitle, addOrActivate])

  const close = useCallback(
    (path: string) => {
      setTabs((prev) => {
        const next = prev.filter((t) => t.path !== path)
        if (path === pathname && next.length > 0) {
          const closedIdx = prev.findIndex((t) => t.path === path)
          const target = next[Math.min(closedIdx, next.length - 1)]
          setTimeout(() => navigate(target.path), 0)
        } else if (next.length === 0) {
          setTimeout(() => navigate('/'), 0)
        }
        return next
      })
    },
    [pathname, navigate],
  )

  const closeOthers = useCallback(
    (path: string) => {
      setTabs((prev) => prev.filter((t) => t.path === path))
    },
    [],
  )

  const closeAll = useCallback(() => {
    setTabs([])
    navigate('/')
  }, [navigate])

  const reorder = useCallback((fromIndex: number, toIndex: number) => {
    setTabs((prev) => {
      if (fromIndex === toIndex || fromIndex < 0 || toIndex < 0 || fromIndex >= prev.length || toIndex >= prev.length) return prev
      const next = [...prev]
      const [moved] = next.splice(fromIndex, 1)
      next.splice(toIndex, 0, moved)
      return next
    })
  }, [])

  const rename = useCallback((path: string, title: string) => {
    const nextTitle = title.trim()
    if (!nextTitle) return
    setTabs((prev) => prev.map((tab) => (
      tab.path === path ? { ...tab, title: nextTitle, renamed: true } : tab
    )))
  }, [])

  const resetTitle = useCallback((path: string) => {
    setTabs((prev) => prev.map((tab) => (
      tab.path === path ? { ...tab, title: tab.originalTitle || tab.title, renamed: false } : tab
    )))
  }, [])

  const value = useMemo<PageTabsContextValue>(
    () => ({ tabs, activePath: pathname, close, closeOthers, closeAll, reorder, rename, resetTitle }),
    [tabs, pathname, close, closeOthers, closeAll, reorder, rename, resetTitle],
  )

  return (
    <Ctx.Provider value={value}>
      {children}
    </Ctx.Provider>
  )
}

export function usePageTabs() {
  return useContext(Ctx)
}
