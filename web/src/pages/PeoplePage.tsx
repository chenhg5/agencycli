import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { User, FolderKanban, X, Eye, EyeOff } from 'lucide-react'
import { apiFetch, apiPost } from '../lib/api'
import { cn } from '../lib/cn'

type PersonRow = {
  username: string; role: string; displayName?: string
  email?: string; avatar?: string; phone?: string; bio?: string
  projects?: { project: string; role: string }[]
  linkedAgents?: string[]; disabled?: boolean; createdAt?: string
}

const fieldCls =
  'w-full rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200 dark:[color-scheme:dark]'

export default function PeoplePage() {
  const { t } = useTranslation()
  const [people, setPeople] = useState<PersonRow[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState({ username: '', displayName: '', password: '', email: '', phone: '' })
  const [showPwd, setShowPwd] = useState(false)
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    try {
      const data = await apiFetch<PersonRow[]>('/api/v1/users')
      setPeople((data ?? []).filter(u => u.role === 'member'))
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { void refresh() }, [refresh])

  async function handleCreate() {
    if (!form.username.trim() || !form.password) return
    setSaving(true); setErr(null)
    try {
      await apiPost('/api/v1/users', {
        username: form.username.trim(),
        password: form.password,
        role: 'member',
        displayName: form.displayName.trim(),
        email: form.email.trim(),
        phone: form.phone.trim(),
      })
      setCreating(false)
      setForm({ username: '', displayName: '', password: '', email: '', phone: '' })
      await refresh()
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)) }
    finally { setSaving(false) }
  }

  return (
    <div className="animate-fade-in px-8 py-6">
      <div className="flex items-center justify-between pb-5">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('people.title')}</h1>
          <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('people.subtitle')}</p>
        </div>
        <button type="button" onClick={() => { setCreating(true); setErr(null); setShowPwd(false) }}
          className="rounded-lg border border-sky-600 bg-white px-3 py-2 text-sm font-medium text-sky-700 hover:bg-sky-50 dark:border-sky-500 dark:bg-zinc-900 dark:text-sky-400 dark:hover:bg-zinc-800">
          {t('people.add')}
        </button>
      </div>

      {loading ? (
        <p className="py-12 text-center text-sm text-neutral-400">{t('forms.loading')}</p>
      ) : people.length === 0 ? (
        <div className="rounded-xl border border-dashed border-neutral-300 bg-white p-12 text-center dark:border-zinc-700 dark:bg-zinc-900/40">
          <User className="mx-auto mb-3 size-10 text-neutral-300 dark:text-zinc-600" strokeWidth={1.2} />
          <p className="text-sm font-medium text-neutral-500 dark:text-zinc-400">{t('people.empty')}</p>
          <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('people.emptyHint')}</p>
        </div>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {people.map(p => (
            <Link key={p.username} to={`/people/${encodeURIComponent(p.username)}`}
              className={cn(
                'group rounded-xl border bg-white p-4 transition-all hover:shadow-md dark:bg-zinc-900/40',
                p.disabled
                  ? 'border-neutral-200/50 opacity-60 dark:border-zinc-700/40'
                  : 'border-neutral-200/80 hover:border-sky-300 dark:border-zinc-700/60 dark:hover:border-sky-600/40'
              )}>
              <div className="flex items-start gap-3">
                {p.avatar ? (
                  <img src={p.avatar} alt="" className="size-10 shrink-0 rounded-full object-cover" />
                ) : (
                  <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-indigo-100 dark:bg-indigo-900/30">
                    <User className="size-5 text-indigo-600 dark:text-indigo-400" strokeWidth={1.8} />
                  </div>
                )}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate text-sm font-semibold text-neutral-900 dark:text-zinc-100">{p.displayName || p.username}</span>
                    {p.disabled && <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] font-medium text-red-600 dark:bg-red-900/30 dark:text-red-400">{t('users.disabled')}</span>}
                  </div>
                  {p.displayName && (
                    <p className="truncate text-xs text-neutral-400 dark:text-zinc-500">{p.username}</p>
                  )}
                  {p.email && (
                    <p className="truncate text-xs text-neutral-400 dark:text-zinc-500">{p.email}</p>
                  )}
                </div>
              </div>
              <div className="mt-3 flex items-center gap-3 text-xs text-neutral-400 dark:text-zinc-500">
                <span className="flex items-center gap-1">
                  <FolderKanban className="size-3.5" strokeWidth={1.5} />
                  {(p.projects?.length ?? 0)} {t('people.projects')}
                </span>
                {p.linkedAgents && p.linkedAgents.length > 0 && (
                  <span className="truncate">{p.linkedAgents.join(', ')}</span>
                )}
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Create Person Dialog */}
      {creating && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" onClick={() => !saving && setCreating(false)}>
          <div className="w-full max-w-md rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
              <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{t('people.add')}</h2>
              <button type="button" onClick={() => setCreating(false)} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 dark:hover:bg-zinc-800"><X className="size-4" /></button>
            </div>
            <div className="space-y-3 px-5 py-4">
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('auth.username')} *</span>
                <input value={form.username} onChange={e => setForm({ ...form, username: e.target.value })} className={fieldCls} placeholder="alice" autoFocus />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('users.displayName')}</span>
                <input value={form.displayName} onChange={e => setForm({ ...form, displayName: e.target.value })} className={fieldCls} placeholder="Alice Wang" />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('auth.password')} *</span>
                <div className="flex items-center gap-2">
                  <input type={showPwd ? 'text' : 'password'} value={form.password}
                    onChange={e => setForm({ ...form, password: e.target.value })}
                    className={cn(fieldCls, 'flex-1')} placeholder={t('auth.pwdMinHint')} />
                  <button type="button" onClick={() => setShowPwd(!showPwd)} className="rounded p-1.5 text-neutral-400 hover:text-neutral-600 dark:hover:text-zinc-300">
                    {showPwd ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </button>
                </div>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('users.email')}</span>
                <input type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} className={fieldCls} placeholder="alice@example.com" />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('users.phone')}</span>
                <input value={form.phone} onChange={e => setForm({ ...form, phone: e.target.value })} className={fieldCls} placeholder="+86 138..." />
              </label>
              {err && <p className="text-sm text-red-600 dark:text-red-400">{err}</p>}
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setCreating(false)} disabled={saving} className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600">{t('forms.cancel')}</button>
                <button type="button" onClick={() => void handleCreate()} disabled={saving || !form.username.trim() || !form.password}
                  className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">
                  {saving ? t('forms.saving') : t('people.create')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
