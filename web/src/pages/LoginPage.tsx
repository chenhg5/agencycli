import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Lock, LogIn } from 'lucide-react'
import { apiLoginPost } from '../lib/api'
import { useAuth, type AuthUser } from '../lib/auth'

type LoginResponse = {
  token: string
  username: string
  role: string
  displayName?: string
  projects?: { project: string; role: string }[]
  linkedAgents?: string[]
}

export default function LoginPage() {
  const { t } = useTranslation()
  const { login } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await apiLoginPost<LoginResponse>('/api/v1/auth/login', { username, password })
      const user: AuthUser = {
        username: res.username,
        role: res.role,
        displayName: res.displayName,
        projects: res.projects,
        linkedAgents: res.linkedAgents,
      }
      login(res.token, user)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-dvh items-center justify-center bg-neutral-50 px-4 dark:bg-zinc-950">
      <div className="w-full max-w-sm">
        {/* Brand */}
        <div className="mb-8 text-center">
          <h1
            className="text-3xl font-bold tracking-tight text-neutral-900 dark:text-zinc-100"
            style={{ fontFamily: "'Space Grotesk', sans-serif" }}
          >
            Agency<span className="text-sky-600 dark:text-sky-400">Cli</span>
          </h1>
          <p className="mt-2 text-sm text-neutral-500 dark:text-zinc-500">{t('auth.loginSubtitle')}</p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="rounded-2xl border border-neutral-200/80 bg-white p-6 shadow-sm dark:border-zinc-700/60 dark:bg-zinc-900">
          {error && (
            <div className="mb-4 rounded-lg bg-red-50 px-3.5 py-2.5 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-400">
              {error}
            </div>
          )}

          <label className="block">
            <span className="mb-1.5 block text-sm font-medium text-neutral-700 dark:text-zinc-300">{t('auth.username')}</span>
            <input
              type="text"
              autoComplete="username"
              autoFocus
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="block w-full rounded-lg border border-neutral-200 bg-neutral-50/50 px-3.5 py-2.5 text-sm text-neutral-900 outline-none transition-colors placeholder:text-neutral-400 focus:border-sky-400 focus:ring-1 focus:ring-sky-400/30 dark:border-zinc-700 dark:bg-zinc-800/50 dark:text-zinc-100 dark:placeholder:text-zinc-600"
              placeholder="admin"
            />
          </label>

          <label className="mt-4 block">
            <span className="mb-1.5 block text-sm font-medium text-neutral-700 dark:text-zinc-300">{t('auth.password')}</span>
            <input
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="block w-full rounded-lg border border-neutral-200 bg-neutral-50/50 px-3.5 py-2.5 text-sm text-neutral-900 outline-none transition-colors placeholder:text-neutral-400 focus:border-sky-400 focus:ring-1 focus:ring-sky-400/30 dark:border-zinc-700 dark:bg-zinc-800/50 dark:text-zinc-100 dark:placeholder:text-zinc-600"
              placeholder="••••••"
            />
          </label>

          <button
            type="submit"
            disabled={loading || !username.trim() || !password.trim()}
            className="mt-6 flex w-full items-center justify-center gap-2 rounded-lg bg-sky-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-sky-700 disabled:opacity-50"
          >
            {loading ? (
              <div className="size-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
            ) : (
              <LogIn className="size-4" strokeWidth={2} />
            )}
            {t('auth.loginButton')}
          </button>

          <div className="mt-4 flex items-start gap-2 rounded-lg bg-amber-50/80 px-3 py-2.5 dark:bg-amber-900/10">
            <Lock className="mt-0.5 size-3.5 shrink-0 text-amber-500" strokeWidth={2} />
            <p className="text-xs leading-relaxed text-amber-700 dark:text-amber-400">
              {t('auth.defaultHint')}
            </p>
          </div>
        </form>
      </div>
    </div>
  )
}
