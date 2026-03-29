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
}

type PageTabsContextValue = {
  tabs: PageTab[]
  activePath: string
  close: (path: string) => void
  closeOthers: (path: string) => void
  closeAll: () => void
}

const Ctx = createContext<PageTabsContextValue>({
  tabs: [],
  activePath: '/',
  close: () => {},
  closeOthers: () => {},
  closeAll: () => {},
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
          if (prev[idx].title !== title) {
            const next = [...prev]
            next[idx] = { ...next[idx], title }
            return next
          }
          return prev
        }
        const next = [...prev, { path, title }]
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

  const value = useMemo<PageTabsContextValue>(
    () => ({ tabs, activePath: pathname, close, closeOthers, closeAll }),
    [tabs, pathname, close, closeOthers, closeAll],
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
