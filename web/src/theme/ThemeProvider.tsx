import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react'

export type ThemeMode = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'agency-console-theme'

function getSystemDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function readStoredTheme(): ThemeMode {
  try {
    const v = localStorage.getItem(STORAGE_KEY) as ThemeMode | null
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {
    /* ignore */
  }
  return 'system'
}

function applyDomDark(isDark: boolean) {
  document.documentElement.classList.toggle('dark', isDark)
}

type ThemeContextValue = {
  theme: ThemeMode
  setTheme: (mode: ThemeMode) => void
  /** light → dark → system → light */
  cycleTheme: () => void
  effectiveDark: boolean
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<ThemeMode>(() => readStoredTheme())
  const [systemDark, setSystemDark] = useState(() => getSystemDark())

  const effectiveDark =
    theme === 'dark' || (theme === 'system' && systemDark)

  useEffect(() => {
    applyDomDark(effectiveDark)
  }, [effectiveDark])

  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => setSystemDark(mq.matches)
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [])

  const setTheme = useCallback((mode: ThemeMode) => {
    setThemeState(mode)
    try {
      localStorage.setItem(STORAGE_KEY, mode)
    } catch {
      /* ignore */
    }
  }, [])

  const cycleTheme = useCallback(() => {
    const order: ThemeMode[] = ['light', 'dark', 'system']
    const idx = order.indexOf(theme)
    const next = order[(idx + 1) % order.length]
    setTheme(next)
  }, [theme, setTheme])

  const value = useMemo(
    () => ({
      theme,
      setTheme,
      cycleTheme,
      effectiveDark,
    }),
    [theme, setTheme, cycleTheme, effectiveDark],
  )

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  )
}

export function useTheme() {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
