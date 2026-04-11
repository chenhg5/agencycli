import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { KeyRound, Plus, Server, Trash2, Pencil, X, Eye, EyeOff, Users, Shield, ShieldCheck, UserPlus } from 'lucide-react'
import { i18n } from '../i18n'
import type { ThemeMode } from '../theme/ThemeProvider'
import { useTheme } from '../theme/ThemeProvider'
import { useAuth } from '../lib/auth'
import { apiFetch, apiPost, apiPut, apiDelete } from '../lib/api'
import { cn } from '../lib/cn'

const selectCls =
  'max-w-xs rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm text-neutral-800 outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800 dark:text-zinc-200 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-800 [&>option]:dark:text-zinc-200'
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

type UserRow = {
  username: string; role: string; displayName?: string
  email?: string; avatar?: string; phone?: string; bio?: string
  projects?: { project: string; role: string }[]
  linkedAgents?: string[]; disabled?: boolean; createdAt?: string
}

type ProjectItem = { name: string }

const PROJECT_ROLES = ['viewer', 'operator', 'manager'] as const
const USER_ROLES = ['admin', 'member'] as const

function UsersSection() {
  const { t } = useTranslation()
  const [users, setUsers] = useState<UserRow[]>([])
  const [projects, setProjects] = useState<ProjectItem[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<{
    isNew: boolean; username: string; displayName: string; role: string
    email: string; avatar: string; phone: string; bio: string
    password: string; disabled: boolean
    projects: { project: string; role: string }[]
    linkedAgents: string[]
  } | null>(null)
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [showPwd, setShowPwd] = useState(false)

  const refresh = useCallback(async () => {
    try {
      const [u, p] = await Promise.all([
        apiFetch<UserRow[]>('/api/v1/users'),
        apiFetch<ProjectItem[]>('/api/v1/projects'),
      ])
      setUsers(u ?? [])
      setProjects(p ?? [])
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { void refresh() }, [refresh])

  function openNew() {
    setEditing({ isNew: true, username: '', displayName: '', role: 'member', email: '', avatar: '', phone: '', bio: '', password: '', disabled: false, projects: [], linkedAgents: [] })
    setShowPwd(false); setErr(null)
  }

  function openEdit(u: UserRow) {
    setEditing({
      isNew: false, username: u.username, displayName: u.displayName ?? '',
      email: u.email ?? '', avatar: u.avatar ?? '', phone: u.phone ?? '', bio: u.bio ?? '',
      role: u.role, password: '', disabled: u.disabled ?? false,
      projects: u.projects ?? [], linkedAgents: u.linkedAgents ?? [],
    })
    setShowPwd(false); setErr(null)
  }

  async function handleSave() {
    if (!editing) return
    setSaving(true); setErr(null)
    try {
      if (editing.isNew) {
        if (!editing.username.trim() || !editing.password) {
          setErr(t('users.usernamePasswordRequired')); setSaving(false); return
        }
        await apiPost('/api/v1/users', {
          username: editing.username.trim(), password: editing.password,
          role: editing.role, displayName: editing.displayName,
          email: editing.email, avatar: editing.avatar, phone: editing.phone, bio: editing.bio,
        })
        if (editing.projects.length || editing.linkedAgents.length) {
          await apiPut(`/api/v1/users/${encodeURIComponent(editing.username.trim())}`, {
            projects: editing.projects, linkedAgents: editing.linkedAgents,
          })
        }
      } else {
        const body: Record<string, unknown> = {
          role: editing.role, displayName: editing.displayName,
          email: editing.email, avatar: editing.avatar, phone: editing.phone, bio: editing.bio,
          disabled: editing.disabled, projects: editing.projects,
          linkedAgents: editing.linkedAgents,
        }
        if (editing.password) body.password = editing.password
        await apiPut(`/api/v1/users/${encodeURIComponent(editing.username)}`, body)
      }
      setEditing(null)
      await refresh()
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)) }
    finally { setSaving(false) }
  }

  async function handleDelete(username: string) {
    if (!confirm(t('users.confirmDelete', { username }))) return
    try { await apiDelete(`/api/v1/users/${encodeURIComponent(username)}`); await refresh() }
    catch { /* ignore */ }
  }

  function addProjectAccess() {
    if (!editing) return
    const available = projects.filter(p => !editing.projects.some(ep => ep.project === p.name))
    if (!available.length) return
    setEditing({ ...editing, projects: [...editing.projects, { project: available[0].name, role: 'operator' }] })
  }

  function removeProjectAccess(idx: number) {
    if (!editing) return
    setEditing({ ...editing, projects: editing.projects.filter((_, i) => i !== idx) })
  }

  function addLinkedAgent() {
    if (!editing) return
    setEditing({ ...editing, linkedAgents: [...editing.linkedAgents, ''] })
  }

  function removeLinkedAgent(idx: number) {
    if (!editing) return
    setEditing({ ...editing, linkedAgents: editing.linkedAgents.filter((_, i) => i !== idx) })
  }

  const fieldCls = 'w-full rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200 dark:[color-scheme:dark]'
  const roleIcon = (role: string) => role === 'admin'
    ? <ShieldCheck className="size-3.5 text-amber-500" />
    : <Shield className="size-3.5 text-sky-500" />

  return (
    <section className="rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="flex items-center justify-between pb-3">
        <div className="flex items-center gap-2">
          <Users className="size-4 text-neutral-500 dark:text-zinc-500" strokeWidth={1.8} />
          <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{t('users.title')}</h3>
        </div>
        <button type="button" onClick={openNew}
          className="flex items-center gap-1 rounded-lg bg-sky-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-sky-700">
          <UserPlus className="size-3.5" /> {t('users.add')}
        </button>
      </div>
      <p className="mb-3 text-xs text-neutral-400 dark:text-zinc-500">{t('users.desc')}</p>

      {loading ? (
        <p className="py-4 text-center text-sm text-neutral-400">{t('forms.loading')}</p>
      ) : users.length === 0 ? (
        <p className="py-4 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('users.empty')}</p>
      ) : (
        <div className="space-y-2">
          {users.map(u => (
            <div key={u.username} className={cn(
              'flex items-center justify-between rounded-lg border px-4 py-2.5',
              u.disabled
                ? 'border-neutral-200/50 bg-neutral-100/30 opacity-60 dark:border-zinc-700/40 dark:bg-zinc-800/20'
                : 'border-neutral-200/80 bg-neutral-50/30 dark:border-zinc-700/60 dark:bg-zinc-800/30'
            )}>
              <div className="flex items-center gap-3">
                {roleIcon(u.role)}
                <div className="flex flex-col">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-neutral-800 dark:text-zinc-200">{u.username}</span>
                    {u.displayName && <span className="text-xs text-neutral-400 dark:text-zinc-500">({u.displayName})</span>}
                    {u.disabled && <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] font-medium text-red-600 dark:bg-red-900/30 dark:text-red-400">{t('users.disabled')}</span>}
                  </div>
                  <span className="text-xs text-neutral-400 dark:text-zinc-500">
                    {u.role}
                    {u.projects && u.projects.length > 0 && ` · ${u.projects.length} ${t('users.projectCount')}`}
                    {u.linkedAgents && u.linkedAgents.length > 0 && ` · ${u.linkedAgents.join(', ')}`}
                  </span>
                </div>
              </div>
              <div className="flex gap-1">
                <button type="button" onClick={() => openEdit(u)}
                  className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300">
                  <Pencil className="size-3.5" />
                </button>
                {u.username !== 'admin' && (
                  <button type="button" onClick={() => void handleDelete(u.username)}
                    className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">
                    <Trash2 className="size-3.5" />
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" onClick={() => !saving && setEditing(null)}>
          <div className="max-h-[85vh] w-full max-w-lg overflow-y-auto rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
              <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
                {editing.isNew ? t('users.add') : t('users.edit')}
              </h2>
              <button type="button" onClick={() => setEditing(null)} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800">
                <X className="size-4" />
              </button>
            </div>
            <div className="space-y-3 px-5 py-4">
              {/* Username */}
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('auth.username')}</span>
                <input value={editing.username} onChange={e => editing.isNew && setEditing({ ...editing, username: e.target.value })}
                  disabled={!editing.isNew} className={cn(fieldCls, !editing.isNew && 'opacity-60')} />
              </label>

              {/* Display Name */}
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('users.displayName')}</span>
                <input value={editing.displayName} onChange={e => setEditing({ ...editing, displayName: e.target.value })} className={fieldCls} />
              </label>

              {/* Password */}
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">
                  {t('auth.password')}{!editing.isNew && <span className="ml-1 text-xs text-neutral-400">({t('users.passwordOptional')})</span>}
                </span>
                <div className="flex items-center gap-2">
                  <input type={showPwd ? 'text' : 'password'} value={editing.password}
                    onChange={e => setEditing({ ...editing, password: e.target.value })}
                    className={cn(fieldCls, 'flex-1')} placeholder={editing.isNew ? t('auth.pwdMinHint') : t('users.passwordUnchanged')} />
                  <button type="button" onClick={() => setShowPwd(!showPwd)} className="rounded p-1.5 text-neutral-400 hover:text-neutral-600 dark:hover:text-zinc-300">
                    {showPwd ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </button>
                </div>
              </label>

              {/* Email */}
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('users.email')}</span>
                <input type="email" value={editing.email} onChange={e => setEditing({ ...editing, email: e.target.value })} className={fieldCls} placeholder="alice@example.com" />
              </label>

              {/* Phone */}
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('users.phone')}</span>
                <input value={editing.phone} onChange={e => setEditing({ ...editing, phone: e.target.value })} className={fieldCls} placeholder="+86 138..." />
              </label>

              {/* Avatar URL */}
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('users.avatar')}</span>
                <input value={editing.avatar} onChange={e => setEditing({ ...editing, avatar: e.target.value })} className={fieldCls} placeholder="https://..." />
              </label>

              {/* Bio */}
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('users.bio')}</span>
                <textarea value={editing.bio} onChange={e => setEditing({ ...editing, bio: e.target.value })} rows={2} className={cn(fieldCls, 'resize-none')} />
              </label>

              {/* Role */}
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('users.role')}</span>
                <select value={editing.role} onChange={e => setEditing({ ...editing, role: e.target.value })} className={fieldCls}>
                  {USER_ROLES.map(r => <option key={r} value={r}>{t(`users.role_${r}`)}</option>)}
                </select>
              </label>

              {/* Disabled */}
              {!editing.isNew && (
                <label className="flex items-center gap-2 py-1">
                  <input type="checkbox" checked={editing.disabled} onChange={e => setEditing({ ...editing, disabled: e.target.checked })}
                    className="size-4 rounded border-neutral-300 text-sky-600 dark:border-zinc-600" />
                  <span className="text-sm text-neutral-600 dark:text-zinc-400">{t('users.disableAccount')}</span>
                </label>
              )}

              {/* Project Access */}
              {editing.role === 'member' && (
                <div className="rounded-lg border border-neutral-200/80 p-3 dark:border-zinc-700/60">
                  <div className="flex items-center justify-between pb-2">
                    <span className="text-sm font-medium text-neutral-700 dark:text-zinc-300">{t('users.projectAccess')}</span>
                    <button type="button" onClick={addProjectAccess}
                      className="flex items-center gap-1 rounded bg-neutral-100 px-2 py-1 text-xs text-neutral-600 hover:bg-neutral-200 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700">
                      <Plus className="size-3" /> {t('users.addProject')}
                    </button>
                  </div>
                  {editing.projects.length === 0 ? (
                    <p className="py-2 text-center text-xs text-neutral-400 dark:text-zinc-500">{t('users.noProjectAccess')}</p>
                  ) : (
                    <div className="space-y-1.5">
                      {editing.projects.map((pa, idx) => (
                        <div key={idx} className="flex items-center gap-2">
                          <select value={pa.project} onChange={e => {
                            const np = [...editing.projects]; np[idx] = { ...np[idx], project: e.target.value }
                            setEditing({ ...editing, projects: np })
                          }} className={cn(fieldCls, 'flex-1 text-xs')}>
                            {projects.map(p => <option key={p.name} value={p.name}>{p.name}</option>)}
                          </select>
                          <select value={pa.role} onChange={e => {
                            const np = [...editing.projects]; np[idx] = { ...np[idx], role: e.target.value }
                            setEditing({ ...editing, projects: np })
                          }} className={cn(fieldCls, 'w-28 text-xs')}>
                            {PROJECT_ROLES.map(r => <option key={r} value={r}>{t(`users.prole_${r}`)}</option>)}
                          </select>
                          <button type="button" onClick={() => removeProjectAccess(idx)} className="rounded p-1 text-neutral-400 hover:text-red-500">
                            <X className="size-3.5" />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* Linked Agents */}
              <div className="rounded-lg border border-neutral-200/80 p-3 dark:border-zinc-700/60">
                <div className="flex items-center justify-between pb-2">
                  <span className="text-sm font-medium text-neutral-700 dark:text-zinc-300">{t('users.linkedAgents')}</span>
                  <button type="button" onClick={addLinkedAgent}
                    className="flex items-center gap-1 rounded bg-neutral-100 px-2 py-1 text-xs text-neutral-600 hover:bg-neutral-200 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700">
                    <Plus className="size-3" /> {t('users.addAgent')}
                  </button>
                </div>
                <p className="mb-2 text-[11px] text-neutral-400 dark:text-zinc-500">{t('users.linkedAgentsHint')}</p>
                {editing.linkedAgents.length === 0 ? (
                  <p className="py-1 text-center text-xs text-neutral-400 dark:text-zinc-500">{t('users.noLinkedAgents')}</p>
                ) : (
                  <div className="space-y-1.5">
                    {editing.linkedAgents.map((agent, idx) => (
                      <div key={idx} className="flex items-center gap-2">
                        <input value={agent} onChange={e => {
                          const na = [...editing.linkedAgents]; na[idx] = e.target.value
                          setEditing({ ...editing, linkedAgents: na })
                        }} className={cn(fieldCls, 'flex-1 font-mono text-xs')} placeholder="project/agent-name" />
                        <button type="button" onClick={() => removeLinkedAgent(idx)} className="rounded p-1 text-neutral-400 hover:text-red-500">
                          <X className="size-3.5" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {err && <p className="text-sm text-red-600 dark:text-red-400">{err}</p>}
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setEditing(null)} disabled={saving}
                  className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600">{t('forms.cancel')}</button>
                <button type="button" onClick={() => void handleSave()} disabled={saving || (editing.isNew && (!editing.username.trim() || !editing.password))}
                  className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">
                  {saving ? t('forms.saving') : t('forms.save')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
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

  const fieldCls = 'w-full rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200 dark:[color-scheme:dark]'

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

        {/* User Management (admin only) */}
        {user?.role === 'admin' && <UsersSection />}

        {/* API Providers (admin only) */}
        {user?.role === 'admin' && <ProvidersSection />}

        {/* Change Password */}
        <ChangePasswordSection />
      </div>
    </div>
  )
}
