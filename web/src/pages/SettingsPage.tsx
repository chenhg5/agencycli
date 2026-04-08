import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { KeyRound, Plus, Server, Trash2, Pencil, X, Eye, EyeOff } from 'lucide-react'
import { i18n } from '../i18n'
import type { ThemeMode } from '../theme/ThemeProvider'
import { useTheme } from '../theme/ThemeProvider'
import { useAuth } from '../lib/auth'
import { apiFetch, apiPost, apiPut, apiDelete } from '../lib/api'
import { cn } from '../lib/cn'

const selectCls =
  'max-w-xs rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm text-neutral-800 outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200'
const inputCls =
  'block w-full max-w-xs rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm text-neutral-800 outline-none transition-colors placeholder:text-neutral-400 focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200 dark:placeholder:text-zinc-600'

function ChangePasswordSection() {
  const { t } = useTranslation()
  const [oldPwd, setOldPwd] = useState('')
  const [newPwd, setNewPwd] = useState('')
  const [confirmPwd, setConfirmPwd] = useState('')
  const [saving, setSaving] = useState(false)
  const [msg, setMsg] = useState<{ type: 'ok' | 'err'; text: string } | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setMsg(null)
    if (newPwd.length < 6) {
      setMsg({ type: 'err', text: t('auth.pwdTooShort') })
      return
    }
    if (newPwd !== confirmPwd) {
      setMsg({ type: 'err', text: t('auth.pwdMismatch') })
      return
    }
    setSaving(true)
    try {
      await apiPut('/api/v1/auth/password', { oldPassword: oldPwd, newPassword: newPwd })
      setMsg({ type: 'ok', text: t('auth.pwdChanged') })
      setOldPwd('')
      setNewPwd('')
      setConfirmPwd('')
    } catch (err) {
      setMsg({ type: 'err', text: err instanceof Error ? err.message : String(err) })
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="flex items-center gap-2 pb-3">
        <KeyRound className="size-4 text-neutral-500 dark:text-zinc-500" strokeWidth={1.8} />
        <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
          {t('auth.changePassword')}
        </h3>
      </div>
      <form onSubmit={handleSubmit} className="space-y-3">
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('auth.oldPassword')}</span>
          <input type="password" autoComplete="current-password" value={oldPwd} onChange={(e) => setOldPwd(e.target.value)} className={inputCls} />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('auth.newPassword')}</span>
          <input type="password" autoComplete="new-password" value={newPwd} onChange={(e) => setNewPwd(e.target.value)} className={inputCls} placeholder={t('auth.pwdMinHint')} />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('auth.confirmPassword')}</span>
          <input type="password" autoComplete="new-password" value={confirmPwd} onChange={(e) => setConfirmPwd(e.target.value)} className={inputCls} />
        </label>
        {msg && (
          <p className={`rounded-md px-3 py-2 text-sm ${msg.type === 'ok' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400' : 'bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400'}`}>
            {msg.text}
          </p>
        )}
        <button
          type="submit"
          disabled={saving || !oldPwd || !newPwd || !confirmPwd}
          className="rounded-lg bg-sky-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50"
        >
          {saving ? t('prompt.saving') : t('auth.changePassword')}
        </button>
      </form>
    </section>
  )
}

type ProviderRow = {
  id: string; name: string; type: string; baseUrl?: string; model?: string
  hasKey: boolean; env?: Record<string, string>
}

const PROVIDER_TYPES = ['anthropic', 'openai', 'gemini', 'custom'] as const

function ProvidersSection() {
  const { t } = useTranslation()
  const [providers, setProviders] = useState<ProviderRow[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<Partial<ProviderRow> & { apiKey?: string } | null>(null)
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [showKey, setShowKey] = useState(false)

  const refresh = useCallback(async () => {
    try {
      const data = await apiFetch<ProviderRow[]>('/api/v1/providers')
      setProviders(data ?? [])
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { void refresh() }, [refresh])

  function openNew() {
    setEditing({ name: '', type: 'anthropic', baseUrl: '', model: '', apiKey: '' })
    setShowKey(false)
    setErr(null)
  }

  function openEdit(p: ProviderRow) {
    setEditing({ ...p, apiKey: '' })
    setShowKey(false)
    setErr(null)
  }

  async function handleSave() {
    if (!editing || !editing.name?.trim()) return
    setSaving(true); setErr(null)
    try {
      const body: any = {
        name: editing.name,
        type: editing.type || 'anthropic',
        baseUrl: editing.baseUrl || '',
        model: editing.model || '',
      }
      if (editing.apiKey) body.apiKey = editing.apiKey
      if (editing.id) {
        await apiPut(`/api/v1/providers/${editing.id}`, body)
      } else {
        await apiPost('/api/v1/providers', body)
      }
      setEditing(null)
      await refresh()
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)) }
    finally { setSaving(false) }
  }

  async function handleDelete(id: string) {
    try {
      await apiDelete(`/api/v1/providers/${id}`)
      await refresh()
    } catch { /* ignore */ }
  }

  const fieldCls = 'w-full rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200'

  return (
    <section className="rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="flex items-center justify-between pb-3">
        <div className="flex items-center gap-2">
          <Server className="size-4 text-neutral-500 dark:text-zinc-500" strokeWidth={1.8} />
          <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{t('provider.title')}</h3>
        </div>
        <button type="button" onClick={openNew}
          className="flex items-center gap-1 rounded-lg bg-sky-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-sky-700">
          <Plus className="size-3.5" /> {t('provider.add')}
        </button>
      </div>
      <p className="mb-3 text-xs text-neutral-400 dark:text-zinc-500">{t('provider.desc')}</p>

      {loading ? (
        <p className="py-4 text-center text-sm text-neutral-400">{t('forms.loading')}</p>
      ) : providers.length === 0 && !editing ? (
        <p className="py-4 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('provider.empty')}</p>
      ) : (
        <div className="space-y-2">
          {providers.map(p => (
            <div key={p.id} className="flex items-center justify-between rounded-lg border border-neutral-200/80 bg-neutral-50/30 px-4 py-2.5 dark:border-zinc-700/60 dark:bg-zinc-800/30">
              <div className="flex flex-col">
                <span className="text-sm font-medium text-neutral-800 dark:text-zinc-200">{p.name}</span>
                <span className="text-xs text-neutral-400 dark:text-zinc-500">
                  {p.type}{p.model ? ` · ${p.model}` : ''}{p.baseUrl ? ` · ${p.baseUrl}` : ''}{p.hasKey ? ' · 🔑' : ''}
                </span>
              </div>
              <div className="flex gap-1">
                <button type="button" onClick={() => openEdit(p)}
                  className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300">
                  <Pencil className="size-3.5" />
                </button>
                <button type="button" onClick={() => void handleDelete(p.id)}
                  className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">
                  <Trash2 className="size-3.5" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" onClick={() => !saving && setEditing(null)}>
          <div className="w-full max-w-md rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
              <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
                {editing.id ? t('provider.edit') : t('provider.add')}
              </h2>
              <button type="button" onClick={() => setEditing(null)} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800"><X className="size-4" /></button>
            </div>
            <div className="space-y-3 px-5 py-4">
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('provider.nameLabel')}</span>
                <input value={editing.name ?? ''} onChange={e => setEditing({ ...editing, name: e.target.value })} className={fieldCls} placeholder="My Anthropic" />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('provider.typeLabel')}</span>
                <select value={editing.type ?? 'anthropic'} onChange={e => setEditing({ ...editing, type: e.target.value })} className={fieldCls}>
                  {PROVIDER_TYPES.map(t => <option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>)}
                </select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">Base URL</span>
                <input value={editing.baseUrl ?? ''} onChange={e => setEditing({ ...editing, baseUrl: e.target.value })} className={cn(fieldCls, 'font-mono text-xs')} placeholder="https://api.anthropic.com" />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('provider.modelLabel')}</span>
                <input value={editing.model ?? ''} onChange={e => setEditing({ ...editing, model: e.target.value })} className={cn(fieldCls, 'font-mono text-xs')} placeholder="claude-sonnet-4-20250514" />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">API Key</span>
                <div className="flex items-center gap-2">
                  <input
                    type={showKey ? 'text' : 'password'}
                    value={editing.apiKey ?? ''}
                    onChange={e => setEditing({ ...editing, apiKey: e.target.value })}
                    className={cn(fieldCls, 'flex-1 font-mono text-xs')}
                    placeholder={editing.id && editing.hasKey ? t('provider.keyUnchangedHint') : 'sk-...'}
                  />
                  <button type="button" onClick={() => setShowKey(!showKey)}
                    className="rounded p-1.5 text-neutral-400 hover:text-neutral-600 dark:hover:text-zinc-300">
                    {showKey ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </button>
                </div>
              </label>
              {err && <p className="text-sm text-red-600 dark:text-red-400">{err}</p>}
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setEditing(null)} disabled={saving} className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600">{t('forms.cancel')}</button>
                <button type="button" onClick={() => void handleSave()} disabled={saving || !editing.name?.trim()} className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">{saving ? t('forms.saving') : t('forms.save')}</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </section>
  )
}

export default function SettingsPage() {
  const { t } = useTranslation()
  const { theme, setTheme } = useTheme()
  const { user } = useAuth()
  const lang = (() => {
    const l = i18n.language
    if (l.startsWith('zh-TW') || l === 'zh-Hant') return 'zh-TW'
    if (l.startsWith('zh')) return 'zh-CN'
    if (l.startsWith('ja')) return 'ja'
    return 'en'
  })()

  return (
    <div className="animate-fade-in px-8 py-6">
      <div className="pb-5">
        <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('settings.title')}</h1>
        <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('settings.intro')}</p>
        {user && (
          <p className="mt-1.5 text-sm text-neutral-400 dark:text-zinc-500">
            {t('auth.loggedInAs')} <span className="font-medium text-neutral-700 dark:text-zinc-300">{user.username}</span>
            <span className="ml-2 rounded-full bg-sky-100 px-2 py-0.5 text-xs font-medium text-sky-700 dark:bg-sky-900/30 dark:text-sky-400">{user.role}</span>
          </p>
        )}
      </div>

      <div className="space-y-5">
        {/* Language */}
        <section className="rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
          <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
            {t('settings.languageSection')}
          </h3>
          <label className="mt-3 flex flex-col gap-1.5">
            <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('language.label')}</span>
            <select
              className={selectCls}
              value={lang}
              onChange={(e) => void i18n.changeLanguage(e.target.value)}
            >
              <option value="en">{t('language.en')}</option>
              <option value="zh-CN">{t('language.zhCN')}</option>
              <option value="zh-TW">{t('language.zhTW')}</option>
              <option value="ja">{t('language.ja')}</option>
            </select>
          </label>
        </section>

        {/* Appearance */}
        <section className="rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
          <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
            {t('settings.appearanceSection')}
          </h3>
          <label className="mt-3 flex flex-col gap-1.5">
            <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('theme.appearance')}</span>
            <select
              className={selectCls}
              value={theme}
              onChange={(e) => setTheme(e.target.value as ThemeMode)}
            >
              <option value="light">{t('theme.light')}</option>
              <option value="dark">{t('theme.dark')}</option>
              <option value="system">{t('theme.system')}</option>
            </select>
          </label>
          <p className="mt-3 text-sm text-neutral-400 dark:text-zinc-500">{t('settings.themeHint')}</p>
        </section>

        {/* API Providers */}
        <ProvidersSection />

        {/* Change Password */}
        <ChangePasswordSection />
      </div>
    </div>
  )
}
