import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import {
  RefreshCw, Save, ChevronRight, Bot, BookOpen, Puzzle, Check, Plus, Trash2, X,
  Settings2, Users, UserCog, FileCode, Clock, Activity, User, Mail, ListTodo, Reply, Send, KeyRound, Pencil, Eye, EyeOff, Container, AlertTriangle, ShieldCheck, CircleX, MessageSquareText,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '../../lib/cn'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'
import { apiFetch, apiPost, apiPut, apiDelete, apiPatch } from '../../lib/api'
import { Pagination } from '../../components/ui/Pagination'
import { IMConnectionPanel } from '../../components/project/IMConnectionPanel'

const AGENT_MODELS = [
  'claudecode', 'codex', 'cursor', 'gemini',
  'qoder', 'opencode', 'iflow', 'generic-cli', 'http-agent',
] as const

const MODEL_COLORS: Record<string, string> = {
  claudecode:    'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300',
  codex:         'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
  cursor:        'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300',
  gemini:        'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
  qoder:         'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300',
  opencode:      'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-300',
  iflow:         'bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-300',
  'generic-cli': 'bg-neutral-200 text-neutral-700 dark:bg-zinc-700 dark:text-zinc-300',
  'http-agent':  'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
  human:         'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300',
}

const AVATAR_BACKGROUNDS = ['b6e3f4', 'c0aede', 'd1d4f9', 'ffd5dc', 'ffdfbf']

function avatarHash(s: string): number {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

function dicebearAvatar(seed: string): string {
  const bg = AVATAR_BACKGROUNDS[avatarHash(seed) % AVATAR_BACKGROUNDS.length]
  return `https://api.dicebear.com/9.x/notionists/svg?seed=${encodeURIComponent(seed)}&backgroundColor=${bg}`
}

function avatarChoices(project: string, agent: string, nonce: number): string[] {
  const base = `${project}-${agent || 'agent'}`
  return Array.from({ length: 8 }, (_, i) => dicebearAvatar(`${base}-${nonce}-${i + 1}`))
}

type SessionInfo = { sessionId?: string; sessionStartedAt?: string; sessionScope?: string }

type HTTPAgentConfig = {
  url?: string
  model?: string
  api_key?: string
  timeout?: string
  stream?: boolean
}

type SandboxConfig = {
  provider: string
  docker?: {
    image?: string
    network_mode?: string
    memory_mb?: number
  }
}

type SetupCheck = {
  key: string
  label: string
  status: 'ok' | 'warning' | 'error'
  detail?: string
}

type AgentContext = {
  contextFile: string
  context: string
  wakeup: string
  model: string
  team: string
  role: string
  avatar?: string
  syncedAt: string | null
  skills: string[]
  httpAgent?: HTTPAgentConfig
  env?: Record<string, string>
  provider?: string
  workDir?: string
  sandbox?: SandboxConfig
  setupChecks?: SetupCheck[]
  addDirs?: string[]
}

const WELL_KNOWN_ENV: Record<string, { keys: string[]; hint: string }> = {
  claudecode: {
    keys: [
      'ANTHROPIC_MODEL', 'ANTHROPIC_BASE_URL', 'ANTHROPIC_AUTH_TOKEN',
      'CLAUDE_AUTOCOMPACT_PCT_OVERRIDE', 'CLAUDE_CODE_AUTO_COMPACT_WINDOW',
    ],
    hint: 'AUTOCOMPACT_PCT: auto-compact threshold % (default 90, recommend 70-80); AUTO_COMPACT_WINDOW: effective token window size',
  },
  codex: {
    keys: ['OPENAI_API_KEY', 'OPENAI_MODEL', 'OPENAI_BASE_URL'],
    hint: 'Auto-compact: config model_auto_compact_token_limit in codex config',
  },
  gemini: {
    keys: ['GEMINI_API_KEY', 'GOOGLE_API_KEY', 'GOOGLE_CLOUD_PROJECT'],
    hint: 'Auto-compress: set chatCompression.contextPercentageThreshold (0-1) in .gemini/settings.json',
  },
  cursor: {
    keys: ['ANTHROPIC_AUTH_TOKEN', 'OPENAI_API_KEY'],
    hint: 'Cursor uses its own auth; env vars are optional overrides',
  },
  opencode: {
    keys: ['ANTHROPIC_AUTH_TOKEN', 'ANTHROPIC_BASE_URL', 'OPENAI_API_KEY', 'OPENAI_BASE_URL'],
    hint: 'OpenCode supports multiple providers',
  },
}

// ── Agent Env Panel ─────────────────────────────────────────────────────────

const envInputCls = 'block w-full rounded-md border border-neutral-200/80 bg-neutral-50/50 px-2.5 py-1.5 text-sm text-neutral-800 outline-none placeholder:text-neutral-400 focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200 dark:placeholder:text-zinc-600'

function AgentEnvPanel({ project, agent }: { project: string; agent: string }) {
  const { t } = useTranslation()
  type Entry = { key: string; value: string }
  const [entries, setEntries] = useState<Entry[]>([])
  const [revealed, setRevealed] = useState<Set<string>>(new Set())
  const [showAdd, setShowAdd] = useState(false)
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [editVal, setEditVal] = useState('')
  const [newKey, setNewKey] = useState('')
  const [newVal, setNewVal] = useState('')

  const load = useCallback(async () => {
    try {
      const data = await apiFetch(`/api/v1/projects/${encodeURIComponent(project)}/agents/${encodeURIComponent(agent)}/env`)
      setEntries(data as Entry[])
    } catch { /* ignore */ }
  }, [project, agent])

  useEffect(() => { load() }, [load])

  const envApi = `/api/v1/projects/${encodeURIComponent(project)}/agents/${encodeURIComponent(agent)}/env`

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    if (!newKey.trim()) return
    try {
      await apiPost(envApi, { key: newKey.trim(), value: newVal })
      setNewKey(''); setNewVal(''); setShowAdd(false); load()
    } catch { /* ignore */ }
  }

  async function handleUpdate(key: string) {
    try {
      await apiPost(envApi, { key, value: editVal })
      setEditingKey(null); setEditVal(''); load()
    } catch { /* ignore */ }
  }

  async function handleRemove(key: string) {
    try {
      await apiDelete(`${envApi}?key=${encodeURIComponent(key)}`)
      load()
    } catch { /* ignore */ }
  }

  function toggleReveal(key: string) {
    setRevealed(prev => { const n = new Set(prev); n.has(key) ? n.delete(key) : n.add(key); return n })
  }

  const isSensitive = (k: string) => /key|token|secret|password|credential/i.test(k)

  return (
    <section className="rounded-xl border border-neutral-200/70 bg-white p-5 dark:border-zinc-700/50 dark:bg-zinc-900">
      <div className="flex items-center justify-between">
        <h3 className="flex items-center gap-2 text-sm font-semibold text-neutral-800 dark:text-zinc-100">
          <KeyRound className="size-4 text-sky-600" />
          {t('agentEnv.title')}
        </h3>
        {!showAdd && (
          <button onClick={() => setShowAdd(true)} className="flex items-center gap-1 rounded-md border border-sky-600 px-2 py-1 text-xs font-medium text-sky-700 hover:bg-sky-50 dark:border-sky-500 dark:text-sky-400 dark:hover:bg-zinc-800">
            <Plus className="size-3" />
            {t('agentEnv.add')}
          </button>
        )}
      </div>
      <p className="mt-1 text-xs text-neutral-500 dark:text-zinc-500">{t('agentEnv.hint')}</p>

      {showAdd && (
        <form onSubmit={handleAdd} className="mt-3 flex items-end gap-2">
          <div className="flex-1">
            <label className="mb-1 block text-[11px] font-medium text-neutral-500 dark:text-zinc-400">KEY</label>
            <input value={newKey} onChange={e => setNewKey(e.target.value)} placeholder="MY_TOKEN" required className={envInputCls} />
          </div>
          <div className="flex-1">
            <label className="mb-1 block text-[11px] font-medium text-neutral-500 dark:text-zinc-400">VALUE</label>
            <input value={newVal} onChange={e => setNewVal(e.target.value)} placeholder="" required className={envInputCls} />
          </div>
          <button type="submit" className="rounded-md bg-sky-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-sky-700">{t('common.save')}</button>
          <button type="button" onClick={() => { setShowAdd(false); setNewKey(''); setNewVal('') }} className="rounded-md border border-neutral-300 px-3 py-1.5 text-xs text-neutral-600 hover:bg-neutral-50 dark:border-zinc-600 dark:text-zinc-400 dark:hover:bg-zinc-800">{t('common.cancel')}</button>
        </form>
      )}

      {entries.length > 0 ? (
        <div className="mt-3 space-y-1.5">
          {entries.map(e => {
            const sensitive = isSensitive(e.key)
            const show = !sensitive || revealed.has(e.key)
            const isEditing = editingKey === e.key
            return (
              <div key={e.key} className="flex items-center justify-between rounded-lg bg-neutral-50/70 px-3 py-2 dark:bg-zinc-800/30">
                <div className="flex min-w-0 flex-1 items-center gap-2">
                  <span className="shrink-0 font-mono text-xs font-semibold text-neutral-800 dark:text-zinc-200">{e.key}</span>
                  <span className="text-[10px] text-neutral-300 dark:text-zinc-600">=</span>
                  {isEditing ? (
                    <div className="flex items-center gap-1.5">
                      <input autoFocus value={editVal} onChange={ev => setEditVal(ev.target.value)} className="w-40 rounded border border-sky-300 bg-white px-1.5 py-0.5 font-mono text-xs text-neutral-800 outline-none dark:border-sky-700 dark:bg-zinc-800 dark:text-zinc-200" />
                      <button onClick={() => handleUpdate(e.key)} className="text-sky-600 hover:text-sky-700 dark:text-sky-400"><Check className="size-3.5" /></button>
                      <button onClick={() => setEditingKey(null)} className="text-neutral-400 hover:text-neutral-600 dark:text-zinc-500"><X className="size-3.5" /></button>
                    </div>
                  ) : (
                    <span className="truncate font-mono text-xs text-neutral-600 dark:text-zinc-400">{show ? (e.value || '—') : '••••••'}</span>
                  )}
                </div>
                <div className="flex shrink-0 items-center gap-1.5">
                  {sensitive && !isEditing && (
                    <button onClick={() => toggleReveal(e.key)} className="text-neutral-400 hover:text-sky-600 dark:text-zinc-500 dark:hover:text-sky-400">
                      {show ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                    </button>
                  )}
                  {!isEditing && (
                    <button onClick={() => { setEditingKey(e.key); setEditVal(e.value) }} className="text-neutral-400 hover:text-sky-600 dark:text-zinc-500 dark:hover:text-sky-400"><Pencil className="size-3.5" /></button>
                  )}
                  <button onClick={() => handleRemove(e.key)} className="text-neutral-400 hover:text-red-500 dark:text-zinc-500 dark:hover:text-red-400"><Trash2 className="size-3.5" /></button>
                </div>
              </div>
            )
          })}
        </div>
      ) : !showAdd ? (
        <p className="mt-3 text-xs text-neutral-400 dark:text-zinc-600">{t('agentEnv.empty')}</p>
      ) : null}
    </section>
  )
}

function PromptEditor({ label, icon: Icon, apiPath, initialContent }: { label: string; icon: LucideIcon; apiPath: string; initialContent: string }) {
  const { t } = useTranslation()
  const [value, setValue] = useState(initialContent)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [preview, setPreview] = useState(false)
  const [saved, setSaved] = useState(false)
  const save = useCallback(async () => {
    setSaving(true); setSaved(false)
    try { await apiPut(apiPath, { content: value }); setDirty(false); setSaved(true); setTimeout(() => setSaved(false), 2000) }
    catch (e) { alert(String(e)) } finally { setSaving(false) }
  }, [apiPath, value])
  return (
    <div className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-2.5 dark:border-zinc-700/40">
        <div className="flex items-center gap-2">
          <Icon className="size-4 text-neutral-400 dark:text-zinc-500" strokeWidth={1.8} />
          <span className="text-sm font-medium text-neutral-700 dark:text-zinc-300">{label}</span>
          {dirty && <span className="text-[10px] text-amber-500">●</span>}
          {saved && <span className="text-[10px] text-emerald-500">{t('prompt.saved')}</span>}
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => setPreview((p) => !p)} className={cn('rounded-md px-2 py-1 text-[11px] font-medium transition-colors', preview ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400' : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400')}>
            {preview ? t('prompt.edit') : t('prompt.preview')}
          </button>
          <button type="button" onClick={save} disabled={saving} className="flex items-center gap-1 rounded-md bg-sky-600 px-2.5 py-1 text-[11px] font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50">
            <Save className="size-3" strokeWidth={2} />{saving ? t('prompt.saving') : t('prompt.save')}
          </button>
        </div>
      </div>
      {preview ? (
        <div className="prose-none max-h-[40vh] overflow-auto p-4 text-sm leading-relaxed text-neutral-800 dark:text-zinc-200">
          <Markdown remarkPlugins={[remarkGfm]}>{value || '*（空）*'}</Markdown>
        </div>
      ) : (
        <textarea value={value} onChange={(e) => { setValue(e.target.value); setDirty(true); setSaved(false) }}
          className="block w-full resize-y bg-transparent p-4 font-mono text-[13px] leading-relaxed text-neutral-800 outline-none placeholder:text-neutral-300 dark:text-zinc-200 dark:placeholder:text-zinc-700"
          rows={Math.max(6, Math.min(20, value.split('\n').length + 1))} placeholder="Markdown prompt..." />
      )}
    </div>
  )
}

function SessionPanel({ project, agentName }: { project: string; agentName: string }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const hbPath = `/api/v1/projects/${encodeURIComponent(project)}/agents/${encodeURIComponent(agentName)}/heartbeat`
  const [reloadKey, setReloadKey] = useState(0)
  const state = useApiJson<SessionInfo & Record<string, unknown>>(hbPath, reloadKey)
  const [resetting, setResetting] = useState(false)
  const [runResult, setRunResult] = useState<string | null>(null)
  const [running, setRunning] = useState(false)

  if (state.status !== 'ok') return null
  const info = state.data
  const hasSession = !!info.sessionId

  async function doReset() {
    setResetting(true)
    try {
      await apiPost('/api/v1/session/reset', { project, agent: agentName })
      setReloadKey((k) => k + 1)
    } catch (e) { alert(String(e)) }
    finally { setResetting(false) }
  }

  async function doRun() {
    setRunning(true); setRunResult(null)
    try {
      const res = await apiPost<{ ok: boolean; output: string }>('/api/v1/run', { project, agent: agentName })
      setRunResult(res.output || t('session.runDone'))
    } catch (e) { setRunResult(String(e)) }
    finally { setRunning(false); setReloadKey((k) => k + 1) }
  }

  return (
    <div className="rounded-lg border border-neutral-200/80 bg-neutral-50/50 px-4 py-3 dark:border-zinc-700/60 dark:bg-zinc-900/30">
      <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('session.sessionLabel')}</h4>
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-xs">
        <span className="text-neutral-500 dark:text-zinc-500">
          Session ID: {hasSession ? <span className="font-mono text-emerald-700 dark:text-emerald-400" title={info.sessionId}>{info.sessionId!.slice(0, 16)}…</span> : <span className="text-neutral-400 dark:text-zinc-500">{t('session.noSession')}</span>}
        </span>
        {info.sessionStartedAt && (
          <span className="text-neutral-500 dark:text-zinc-500">{t('session.startedAt')}: {fmt(info.sessionStartedAt)}</span>
        )}
        <span className="text-neutral-500 dark:text-zinc-500">{t('session.scopeLabel')}: <span className="font-medium text-neutral-700 dark:text-zinc-300">{info.sessionScope === 'task' ? t('session.scopeTask') : t('session.scopeCycle')}</span></span>

        <div className="flex items-center gap-2">
          <Link
            to={`/projects/${encodeURIComponent(project)}/members/${encodeURIComponent(agentName)}/chat${hasSession ? `?sessionId=${encodeURIComponent(info.sessionId!)}` : ''}`}
            className="flex items-center gap-1 rounded-md border border-sky-200 bg-white px-2.5 py-1 text-xs font-medium text-sky-700 transition-colors hover:bg-sky-50 dark:border-sky-800 dark:bg-sky-900/30 dark:text-sky-400 dark:hover:bg-sky-900/50"
          >
            <MessageSquareText className="size-3.5" strokeWidth={2} />
            {t('agentChat.openChat')}
          </Link>
          {hasSession && (
            <button type="button" onClick={() => void doReset()} disabled={resetting}
              className="cursor-pointer rounded-md border border-amber-200 bg-white px-2.5 py-1 text-xs font-medium text-amber-700 transition-colors hover:bg-amber-50 disabled:opacity-50 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-400 dark:hover:bg-amber-900/50">
              {resetting ? t('session.resettingSession') : t('session.resetSession')}
            </button>
          )}
          <button type="button" onClick={() => void doRun()} disabled={running}
            className="cursor-pointer rounded-md border border-sky-200 bg-white px-2.5 py-1 text-xs font-medium text-sky-700 transition-colors hover:bg-sky-50 disabled:opacity-50 dark:border-sky-800 dark:bg-sky-900/30 dark:text-sky-400 dark:hover:bg-sky-900/50">
            {running ? t('session.running') : t('session.run')}
          </button>
        </div>
      </div>
      {runResult && (
        <pre className="mt-2 max-h-36 overflow-auto rounded-md bg-white p-3 font-mono text-xs leading-relaxed text-neutral-600 dark:bg-zinc-800 dark:text-zinc-400">{runResult}</pre>
      )}
    </div>
  )
}

function ModelSelector({ project, agentName, currentModel, currentHttpAgent, onChanged }: {
  project: string; agentName: string; currentModel: string; currentHttpAgent?: HTTPAgentConfig; onChanged: () => void
}) {
  const { t } = useTranslation()
  const [model, setModel] = useState(currentModel)
  const [httpUrl, setHttpUrl] = useState(currentHttpAgent?.url ?? '')
  const [httpModel, setHttpModel] = useState(currentHttpAgent?.model ?? '')
  const [httpApiKey, setHttpApiKey] = useState(currentHttpAgent?.api_key ?? '')
  const [httpTimeout, setHttpTimeout] = useState(currentHttpAgent?.timeout ?? '10m')
  const [httpStream, setHttpStream] = useState(currentHttpAgent?.stream ?? true)
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<{ ok: boolean; msg: string } | null>(null)

  const isHttp = model === 'http-agent'
  const modelDirty = model !== currentModel
  const httpDirty = isHttp && (
    httpUrl !== (currentHttpAgent?.url ?? '') ||
    httpModel !== (currentHttpAgent?.model ?? '') ||
    httpApiKey !== (currentHttpAgent?.api_key ?? '') ||
    httpTimeout !== (currentHttpAgent?.timeout ?? '10m') ||
    httpStream !== (currentHttpAgent?.stream ?? true)
  )
  const dirty = modelDirty || httpDirty

  async function apply() {
    setBusy(true); setResult(null)
    try {
      const body: Record<string, unknown> = { model }
      if (isHttp) {
        body.httpUrl = httpUrl
        body.httpModel = httpModel
        body.httpApiKey = httpApiKey
        body.httpTimeout = httpTimeout
        body.httpStream = httpStream
      }
      const res = await apiPost<{ ok: boolean; output: string }>(
        `/api/v1/projects/${encodeURIComponent(project)}/agents/${encodeURIComponent(agentName)}/set-model`,
        body,
      )
      setResult({ ok: true, msg: res.output || t('forms.saved') })
      onChanged()
    } catch (e) {
      setResult({ ok: false, msg: e instanceof Error ? e.message : String(e) })
    } finally { setBusy(false) }
  }

  const inputCls = 'h-7 rounded-md border border-neutral-200 bg-white px-2 text-xs text-neutral-700 outline-none hover:border-neutral-300 focus:border-sky-400 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 dark:[color-scheme:dark]'

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <span className="text-sm text-neutral-500 dark:text-zinc-500">{t('members.agentType')}:</span>
        <select
          value={model}
          onChange={(e) => { setModel(e.target.value); setResult(null) }}
          disabled={busy}
          className={cn(inputCls, 'font-medium')}
        >
          {AGENT_MODELS.map((m) => (
            <option key={m} value={m}>{m}</option>
          ))}
        </select>
        {dirty && (
          <button
            type="button"
            onClick={() => void apply()}
            disabled={busy}
            className="flex items-center gap-1 rounded-md bg-sky-600 px-2 py-1 text-[11px] font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50"
          >
            {busy ? t('forms.saving') : t('forms.apply')}
          </button>
        )}
        {result && (
          <span className={cn('text-[11px]', result.ok ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-500')}>
            {result.ok && <Check className="mr-0.5 inline size-3" strokeWidth={2} />}
            {result.msg.split('\n')[0]}
          </span>
        )}
      </div>
      {isHttp && (
        <div className="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-1.5 pl-0.5 text-xs">
          <span className="text-neutral-500 dark:text-zinc-500">URL *</span>
          <input value={httpUrl} onChange={(e) => setHttpUrl(e.target.value)} disabled={busy}
            placeholder="http://localhost:11434/v1/chat/completions" className={cn(inputCls, 'w-full')} />
          <span className="text-neutral-500 dark:text-zinc-500">Model</span>
          <input value={httpModel} onChange={(e) => setHttpModel(e.target.value)} disabled={busy}
            placeholder="llama3.2, gpt-4o, ..." className={cn(inputCls, 'w-full')} />
          <span className="text-neutral-500 dark:text-zinc-500">API Key</span>
          <input value={httpApiKey} onChange={(e) => setHttpApiKey(e.target.value)} disabled={busy}
            type="password" placeholder="Bearer token" className={cn(inputCls, 'w-full')} />
          <span className="text-neutral-500 dark:text-zinc-500">Timeout</span>
          <input value={httpTimeout} onChange={(e) => setHttpTimeout(e.target.value)} disabled={busy}
            placeholder="10m" className={cn(inputCls, 'w-24')} />
          <span className="text-neutral-500 dark:text-zinc-500">Stream</span>
          <label className="flex cursor-pointer items-center gap-1.5">
            <input type="checkbox" checked={httpStream} onChange={(e) => setHttpStream(e.target.checked)} disabled={busy}
              className="size-3.5 rounded border-neutral-300 text-sky-600 focus:ring-sky-400" />
            <span className="text-neutral-500 dark:text-zinc-400">SSE</span>
          </label>
        </div>
      )}
    </div>
  )
}

export default function ProjectAgentDetailPage() {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const navigate = useNavigate()
  const { projectId, agentName } = useParams<{ projectId: string; agentName: string }>()

  const ctxPath = projectId && agentName
    ? `/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/context`
    : null
  const [ctxReload, setCtxReload] = useState(0)
  const ctxState = useApiJson<AgentContext>(ctxPath, ctxReload)

  const [syncing, setSyncing] = useState(false)
  const [syncOutput, setSyncOutput] = useState<string | null>(null)
  const [editingIdentity, setEditingIdentity] = useState(false)
  const [identityName, setIdentityName] = useState(agentName ?? '')
  const [identityAvatar, setIdentityAvatar] = useState('')
  const [avatarNonce, setAvatarNonce] = useState(() => Date.now())
  const [savingIdentity, setSavingIdentity] = useState(false)
  const [identityError, setIdentityError] = useState<string | null>(null)

  useEffect(() => {
    if (ctxState.status !== 'ok') return
    setIdentityName(agentName ?? '')
    setIdentityAvatar(ctxState.data.avatar ?? '')
  }, [agentName, ctxState.status, ctxState.status === 'ok' ? ctxState.data.avatar : undefined])

  const doSync = useCallback(async () => {
    if (!projectId || !agentName) return
    setSyncing(true)
    setSyncOutput(null)
    try {
      const res = await apiPost<{ ok: boolean; output: string }>(`/api/v1/projects/${encodeURIComponent(projectId)}/sync`, { agent: agentName })
      setSyncOutput(res.output || 'Sync completed.')
      setCtxReload((k) => k + 1)
    } catch (e) {
      setSyncOutput(String(e))
    } finally {
      setSyncing(false)
    }
  }, [projectId, agentName])

  const saveIdentity = useCallback(async () => {
    if (!projectId || !agentName) return
    setSavingIdentity(true)
    setIdentityError(null)
    try {
      const res = await apiPatch<{ ok: boolean; name: string; avatar: string }>(
        `/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}`,
        { name: identityName.trim(), avatar: identityAvatar.trim() },
      )
      setEditingIdentity(false)
      if (res.name && res.name !== agentName) {
        navigate(`/projects/${encodeURIComponent(projectId)}/members/${encodeURIComponent(res.name)}`, { replace: true })
        return
      }
      setIdentityAvatar(res.avatar ?? '')
      setCtxReload((k) => k + 1)
    } catch (e) {
      setIdentityError(e instanceof Error ? e.message : String(e))
    } finally {
      setSavingIdentity(false)
    }
  }, [projectId, agentName, identityName, identityAvatar, navigate])

  const identityAvatarChoices = useMemo(
    () => avatarChoices(projectId ?? '', identityName || agentName || 'agent', avatarNonce),
    [projectId, identityName, agentName, avatarNonce],
  )

  if (!projectId || !agentName) return null

  const isHuman = ctxState.status === 'ok' && ctxState.data.model === 'human'
  const modelCls = MODEL_COLORS[ctxState.status === 'ok' ? ctxState.data.model : ''] ?? ''
  const HeaderIcon = isHuman ? User : Bot
  const headerIconBg = isHuman ? 'bg-indigo-100 dark:bg-indigo-900/30' : 'bg-violet-100 dark:bg-violet-900/30'
  const headerIconColor = isHuman ? 'text-indigo-600 dark:text-indigo-400' : 'text-violet-600 dark:text-violet-400'
  const avatar = ctxState.status === 'ok' ? ctxState.data.avatar : ''

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Header */}
      <div className="shrink-0 px-6 pt-5 pb-4">
        <div className="flex items-center gap-4">
          {avatar ? (
            <img src={avatar} alt="" className="size-12 shrink-0 rounded-xl bg-neutral-100 object-cover dark:bg-zinc-800" />
          ) : (
            <div className={cn('flex size-12 shrink-0 items-center justify-center rounded-xl', headerIconBg)}>
              <HeaderIcon className={cn('size-6', headerIconColor)} strokeWidth={1.8} />
            </div>
          )}
          <div className="min-w-0 flex-1">
            {editingIdentity ? (
              <div className="max-w-2xl space-y-2">
                <div className="flex flex-wrap items-center gap-2">
                  <input
                    value={identityName}
                    onChange={(e) => setIdentityName(e.target.value)}
                    className="h-8 w-56 rounded-md border border-neutral-200 bg-white px-2.5 font-mono text-sm text-neutral-900 outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
                    placeholder="agent-name"
                  />
                  <input
                    value={identityAvatar}
                    onChange={(e) => setIdentityAvatar(e.target.value)}
                    className="h-8 min-w-0 flex-1 rounded-md border border-neutral-200 bg-white px-2.5 text-sm text-neutral-900 outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
                    placeholder={t('agentIdentity.avatarPlaceholder')}
                  />
                  <button type="button" onClick={() => void saveIdentity()} disabled={savingIdentity}
                    className="rounded-md bg-sky-600 px-2.5 py-1.5 text-xs font-medium text-white hover:bg-sky-700 disabled:opacity-50">
                    {savingIdentity ? t('forms.saving') : t('forms.save')}
                  </button>
                  <button type="button" onClick={() => { setEditingIdentity(false); setIdentityError(null); setIdentityName(agentName); setIdentityAvatar(avatar ?? '') }} disabled={savingIdentity}
                    className="rounded-md border border-neutral-300 px-2.5 py-1.5 text-xs text-neutral-600 hover:bg-neutral-50 dark:border-zinc-600 dark:text-zinc-300 dark:hover:bg-zinc-800">
                    {t('forms.cancel')}
                  </button>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {identityAvatarChoices.map((url) => (
                    <button
                      key={url}
                      type="button"
                      onClick={() => setIdentityAvatar(url)}
                      className={cn(
                        'rounded-lg border p-0.5 transition-colors',
                        identityAvatar === url
                          ? 'border-sky-500 bg-sky-50 dark:border-sky-400 dark:bg-sky-900/20'
                          : 'border-neutral-200 hover:border-neutral-300 dark:border-zinc-700 dark:hover:border-zinc-600',
                      )}
                      title={t('agentIdentity.chooseAvatar')}
                    >
                      <img src={url} alt="" className="size-8 rounded-md bg-neutral-100 object-cover dark:bg-zinc-800" />
                    </button>
                  ))}
                  <button
                    type="button"
                    onClick={() => setAvatarNonce(Date.now())}
                    className="rounded-md border border-neutral-300 px-2.5 py-1.5 text-xs text-neutral-600 hover:bg-neutral-50 dark:border-zinc-600 dark:text-zinc-300 dark:hover:bg-zinc-800"
                  >
                    {t('agentIdentity.randomAvatars')}
                  </button>
                  {identityAvatar && (
                    <button
                      type="button"
                      onClick={() => setIdentityAvatar('')}
                      className="rounded-md border border-neutral-300 px-2.5 py-1.5 text-xs text-neutral-600 hover:bg-neutral-50 dark:border-zinc-600 dark:text-zinc-300 dark:hover:bg-zinc-800"
                    >
                      {t('agentIdentity.clearAvatar')}
                    </button>
                  )}
                </div>
                {identityError && <p className="text-xs text-red-600 dark:text-red-400">{identityError}</p>}
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <h1 className="truncate text-xl font-semibold text-neutral-900 dark:text-zinc-100">{agentName}</h1>
                {ctxState.status === 'ok' && (
                  <button type="button" onClick={() => setEditingIdentity(true)} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300" title="Edit name/avatar">
                    <Pencil className="size-3.5" strokeWidth={1.8} />
                  </button>
                )}
              </div>
            )}
            <div className="mt-1 flex items-center gap-3">
              {ctxState.status === 'ok' && (
                <>
                  <span className={cn('inline-flex items-center rounded-md px-2.5 py-0.5 text-xs font-bold tracking-wide', modelCls)}>
                    {ctxState.data.model}
                  </span>
                  {ctxState.data.team && <span className="text-sm text-neutral-500 dark:text-zinc-500">{ctxState.data.team}</span>}
                  {ctxState.data.role && <span className="text-sm text-neutral-500 dark:text-zinc-500">/ {ctxState.data.role}</span>}
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-6 pb-8">
        {ctxState.status === 'loading' && (
          <div className="flex items-center gap-2 py-12 justify-center">
            <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {ctxState.status === 'error' && (
          <p className="py-3 text-sm text-red-500">{ctxState.error.message}</p>
        )}

        {/* ── Human member view ── */}
        {ctxState.status === 'ok' && isHuman && (
          <div className="space-y-8">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {ctxState.data.team && <InfoCard icon={Users} label={t('prompt.team')} value={ctxState.data.team} />}
              {ctxState.data.role && <InfoCard icon={UserCog} label={t('prompt.role')} value={ctxState.data.role} />}
            </div>
            <HumanMessagesPanel project={projectId} member={agentName} />
            <HumanTasksPanel project={projectId} member={agentName} />
          </div>
        )}

        {/* ── Agent member view ── */}
        {ctxState.status === 'ok' && !isHuman && (() => {
          const ctx = ctxState.data
          return (
            <div className="space-y-8">
              {/* Overview info grid */}
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
                {ctx.team && <InfoCard icon={Users} label={t('prompt.team')} value={ctx.team} />}
                {ctx.role && <InfoCard icon={UserCog} label={t('prompt.role')} value={ctx.role} />}
                <InfoCard icon={FileCode} label={t('prompt.contextFile')} value={ctx.contextFile} mono />
                {ctx.syncedAt && <InfoCard icon={Clock} label={t('prompt.lastSync')} value={fmt(ctx.syncedAt)} />}
              </div>

              {/* Setup Checklist */}
              {ctx.setupChecks && ctx.setupChecks.length > 0 && (
                <section>
                  <SectionHeader icon={ShieldCheck} title={t('members.setupChecklist')} />
                  <div className="mt-3 rounded-lg border border-neutral-200/80 bg-white p-4 dark:border-zinc-700/60 dark:bg-zinc-900/40">
                    <div className="space-y-2">
                      {ctx.setupChecks.map((c) => (
                        <div key={c.key} className="flex items-start gap-2.5 text-sm">
                          {c.status === 'ok' && <Check className="mt-0.5 size-4 shrink-0 text-green-500" />}
                          {c.status === 'warning' && <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-500" />}
                          {c.status === 'error' && <CircleX className="mt-0.5 size-4 shrink-0 text-red-500" />}
                          <div className="min-w-0">
                            <span className={cn(
                              'font-medium',
                              c.status === 'ok' && 'text-neutral-700 dark:text-zinc-300',
                              c.status === 'warning' && 'text-amber-700 dark:text-amber-400',
                              c.status === 'error' && 'text-red-700 dark:text-red-400',
                            )}>{c.label}</span>
                            {c.detail && (
                              <p className="mt-0.5 text-xs text-neutral-500 dark:text-zinc-500">{c.detail}</p>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </section>
              )}

              {/* Configuration: model + env */}
              <section>
                <SectionHeader icon={Settings2} title={t('members.configuration')} />
                <div className="mt-3 space-y-4">
                  <div className="rounded-lg border border-neutral-200/80 bg-white p-4 dark:border-zinc-700/60 dark:bg-zinc-900/40">
                    <ModelSelector
                      project={projectId}
                      agentName={agentName}
                      currentModel={ctx.model}
                      currentHttpAgent={ctx.httpAgent}
                      onChanged={() => setCtxReload((k) => k + 1)}
                    />
                  </div>
                  <EnvEditor
                    project={projectId}
                    agentName={agentName}
                    model={ctx.model}
                    initialEnv={ctx.env ?? {}}
                    initialProvider={ctx.provider}
                    onChanged={() => setCtxReload((k) => k + 1)}
                  />
                  <SandboxEditor
                    project={projectId}
                    agentName={agentName}
                    initial={ctx.sandbox}
                    initialAddDirs={ctx.addDirs ?? []}
                    onChanged={() => setCtxReload((k) => k + 1)}
                  />
                </div>
              </section>

              {/* Skills */}
              {ctx.skills && ctx.skills.length > 0 && (
                <section>
                  <SectionHeader icon={Puzzle} title={t('skill.agentSkills')} />
                  <div className="mt-3 flex flex-wrap gap-2">
                    {ctx.skills.map((sk) => (
                      <Link key={sk} to={`/skills?open=${encodeURIComponent(sk)}`}
                        className="inline-flex items-center gap-1.5 rounded-md bg-amber-50 px-2.5 py-1 text-sm font-medium text-amber-700 transition-colors hover:bg-amber-100 dark:bg-amber-900/20 dark:text-amber-400 dark:hover:bg-amber-900/40">
                        <Puzzle className="size-3.5" strokeWidth={2} />
                        {sk}
                      </Link>
                    ))}
                  </div>
                </section>
              )}

              {/* Session & Sync */}
              <section>
                <SectionHeader icon={Activity} title={t('session.sessionLabel')} />
                <div className="mt-3 space-y-3">
                  <SessionPanel project={projectId} agentName={agentName} />
                  <div className="flex items-center gap-3">
                    <button
                      type="button"
                      onClick={doSync}
                      disabled={syncing}
                      className="flex items-center gap-1.5 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm font-medium text-neutral-700 transition-colors hover:border-neutral-300 hover:bg-neutral-50 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:border-zinc-600 dark:hover:bg-zinc-700"
                    >
                      <RefreshCw className={cn('size-4', syncing && 'animate-spin')} strokeWidth={2} />
                      {syncing ? t('prompt.syncing') : t('prompt.sync')}
                    </button>
                    {syncOutput && (
                      <span className="text-sm text-neutral-500 dark:text-zinc-500">{syncOutput}</span>
                    )}
                  </div>
                </div>
              </section>

              {/* IM Connections */}
              <IMConnectionPanel
                project={projectId}
                agentName={agentName}
                model={ctx.model}
                workDir={ctx.workDir}
              />

              {/* Environment Variables */}
              <AgentEnvPanel project={projectId} agent={agentName} />

              {/* Wakeup prompt */}
              <PromptEditor
                label={t('prompt.wakeup')}
                icon={BookOpen}
                apiPath={`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/wakeup`}
                initialContent={ctx.wakeup}
              />

              {/* Merged context */}
              {ctx.context && (
                <details className="group">
                  <summary className="flex cursor-pointer items-center gap-1.5 text-sm font-medium text-neutral-600 dark:text-zinc-400">
                    <ChevronRight className="size-4 transition-transform group-open:rotate-90" strokeWidth={2} />
                    {t('prompt.mergedContext')} ({ctx.contextFile})
                  </summary>
                  <div className="mt-2 max-h-[50vh] overflow-auto rounded-lg border border-neutral-200/80 bg-neutral-50 p-4 font-mono text-sm leading-relaxed text-neutral-600 dark:border-zinc-700/60 dark:bg-zinc-950 dark:text-zinc-400">
                    <Markdown remarkPlugins={[remarkGfm]}>{ctx.context}</Markdown>
                  </div>
                </details>
              )}
            </div>
          )
        })()}
      </div>
    </div>
  )
}

function InfoCard({ icon: Icon, label, value, mono }: { icon?: LucideIcon; label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-neutral-200/80 bg-neutral-50/50 px-4 py-3 dark:border-zinc-700/60 dark:bg-zinc-800/30">
      {Icon && (
        <div className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md bg-neutral-100 dark:bg-zinc-700/60">
          <Icon className="size-3.5 text-neutral-500 dark:text-zinc-400" strokeWidth={1.8} />
        </div>
      )}
      <div className="min-w-0">
        <p className="text-xs font-medium text-neutral-400 dark:text-zinc-500">{label}</p>
        <p className={cn('mt-0.5 text-sm font-medium text-neutral-800 dark:text-zinc-200', mono && 'font-mono text-xs')} title={value}>{value}</p>
      </div>
    </div>
  )
}

function SandboxEditor({ project, agentName, initial, initialAddDirs, onChanged }: {
  project: string; agentName: string; initial?: SandboxConfig; initialAddDirs: string[]; onChanged: () => void
}) {
  const { t } = useTranslation()
  const [provider, setProvider] = useState(initial?.provider ?? '')
  const [image, setImage] = useState(initial?.docker?.image ?? '')
  const [network, setNetwork] = useState(initial?.docker?.network_mode ?? '')
  const [memoryMb, setMemoryMb] = useState(initial?.docker?.memory_mb ?? 0)
  const [addDirs, setAddDirs] = useState<string[]>(initialAddDirs)
  const [newDir, setNewDir] = useState('')
  const [saving, setSaving] = useState(false)
  const [dirty, setDirty] = useState(false)

  const inputCls = 'w-full rounded-md border border-neutral-200/80 bg-white px-3 py-1.5 text-sm text-neutral-800 outline-none transition-colors focus:border-sky-400 focus:ring-1 focus:ring-sky-400/30 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200 dark:focus:border-sky-500'
  const labelCls = 'block text-xs font-medium text-neutral-500 dark:text-zinc-400 mb-1'

  async function save() {
    setSaving(true)
    try {
      await apiPut(`/api/v1/projects/${encodeURIComponent(project)}/agents/${encodeURIComponent(agentName)}/sandbox`, {
        provider: provider || 'none',
        image,
        network,
        memoryMb,
        addDirs,
      })
      setDirty(false)
      onChanged()
    } finally { setSaving(false) }
  }

  function addDir() {
    const d = newDir.trim()
    if (!d || addDirs.includes(d)) return
    setAddDirs(prev => [...prev, d])
    setNewDir('')
    setDirty(true)
  }

  function removeDir(dir: string) {
    setAddDirs(prev => prev.filter(d => d !== dir))
    setDirty(true)
  }

  return (
    <div className="rounded-lg border border-neutral-200/80 bg-white p-4 dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Container className="size-4 text-neutral-400 dark:text-zinc-500" strokeWidth={1.8} />
          <span className="text-sm font-medium text-neutral-700 dark:text-zinc-300">{t('sandbox.title')}</span>
        </div>
        {dirty && (
          <button type="button" onClick={() => void save()} disabled={saving}
            className="flex items-center gap-1.5 rounded-md border border-sky-200 bg-sky-50 px-3 py-1 text-xs font-medium text-sky-700 transition-colors hover:bg-sky-100 disabled:opacity-40 dark:border-sky-800 dark:bg-sky-900/30 dark:text-sky-400 dark:hover:bg-sky-900/50">
            <Save className="size-3" strokeWidth={2} />
            {saving ? t('common.save') + '...' : t('common.save')}
          </button>
        )}
      </div>
      <div className="space-y-3">
        <div>
          <label className={labelCls}>{t('sandbox.provider')}</label>
          <select value={provider} onChange={(e) => { setProvider(e.target.value); setDirty(true) }} className={inputCls}>
            <option value="">{t('sandbox.providerNone')}</option>
            <option value="docker">Docker</option>
          </select>
        </div>
        {provider === 'docker' && (
          <>
            <div>
              <label className={labelCls}>{t('sandbox.image')}</label>
              <input type="text" value={image} onChange={(e) => { setImage(e.target.value); setDirty(true) }} placeholder={t('sandbox.imagePlaceholder')} className={inputCls} />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className={labelCls}>{t('sandbox.network')}</label>
                <select value={network} onChange={(e) => { setNetwork(e.target.value); setDirty(true) }} className={inputCls}>
                  <option value="">bridge ({t('sandbox.networkDefault')})</option>
                  <option value="host">host</option>
                  <option value="none">none ({t('sandbox.networkOffline')})</option>
                </select>
              </div>
              <div>
                <label className={labelCls}>{t('sandbox.memory')}</label>
                <input type="number" value={memoryMb || ''} onChange={(e) => { setMemoryMb(Number(e.target.value)); setDirty(true) }} placeholder={t('sandbox.memoryPlaceholder')} className={inputCls} />
              </div>
            </div>
          </>
        )}

        {/* Add Dirs — always visible, works with both docker and host mode */}
        <div>
          <label className={labelCls}>{t('sandbox.addDirs')}</label>
          <p className="mb-1.5 text-[11px] text-neutral-400 dark:text-zinc-500">{t('sandbox.addDirsHint')}</p>
          {addDirs.length > 0 && (
            <div className="mb-2 space-y-1">
              {addDirs.map(dir => (
                <div key={dir} className="flex items-center gap-2 rounded-md bg-neutral-50 px-3 py-1.5 dark:bg-zinc-800/40">
                  <span className="min-w-0 flex-1 truncate font-mono text-xs text-neutral-700 dark:text-zinc-300">{dir}</span>
                  <button type="button" onClick={() => removeDir(dir)}
                    className="shrink-0 text-neutral-400 transition-colors hover:text-red-500 dark:text-zinc-500 dark:hover:text-red-400">
                    <X className="size-3.5" strokeWidth={2} />
                  </button>
                </div>
              ))}
            </div>
          )}
          <div className="flex gap-2">
            <input
              type="text"
              value={newDir}
              onChange={(e) => setNewDir(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addDir() } }}
              placeholder="/path/to/repo"
              className="min-w-0 flex-1 rounded-md border border-neutral-200/80 bg-white px-3 py-1.5 font-mono text-xs text-neutral-800 outline-none focus:border-sky-400 focus:ring-1 focus:ring-sky-400/30 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200 dark:focus:border-sky-500"
            />
            <button type="button" onClick={addDir}
              className="flex shrink-0 items-center gap-1 rounded-md border border-neutral-200 bg-white px-2.5 py-1.5 text-xs font-medium text-neutral-600 transition-colors hover:border-sky-400 hover:text-sky-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:border-sky-600 dark:hover:text-sky-400">
              <Plus className="size-3.5" strokeWidth={2} />
              {t('sandbox.addDirsAdd')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function SectionHeader({ icon: Icon, title }: { icon: LucideIcon; title: string }) {
  return (
    <div className="flex items-center gap-2.5">
      <div className="flex size-7 items-center justify-center rounded-lg bg-neutral-100 dark:bg-zinc-800">
        <Icon className="size-3.5 text-neutral-500 dark:text-zinc-400" strokeWidth={1.8} />
      </div>
      <h3 className="text-sm font-semibold text-neutral-800 dark:text-zinc-200">{title}</h3>
    </div>
  )
}

type ProviderOption = { id: string; name: string; type: string; model?: string }

function EnvEditor({ project, agentName, model, initialEnv, initialProvider, onChanged }: {
  project: string; agentName: string; model: string; initialEnv: Record<string, string>; initialProvider?: string; onChanged: () => void
}) {
  const { t } = useTranslation()
  const [entries, setEntries] = useState<{ key: string; value: string }[]>(() => {
    const items = Object.entries(initialEnv).map(([key, value]) => ({ key, value }))
    return items.length > 0 ? items : []
  })
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState(false)
  const [providerOptions, setProviderOptions] = useState<ProviderOption[]>([])
  const [selectedProvider, setSelectedProvider] = useState(initialProvider ?? '')

  useEffect(() => {
    void apiFetch<ProviderOption[]>('/api/v1/providers').then(data => setProviderOptions(data ?? [])).catch(() => {})
  }, [])

  const wellKnown = WELL_KNOWN_ENV[model]
  const usedKeys = new Set(entries.map((e) => e.key))

  function addEntry(key = '') {
    setEntries((prev) => [...prev, { key, value: '' }])
    setSaved(false)
  }

  function removeEntry(idx: number) {
    setEntries((prev) => prev.filter((_, i) => i !== idx))
    setSaved(false)
  }

  function updateEntry(idx: number, field: 'key' | 'value', val: string) {
    setEntries((prev) => prev.map((e, i) => i === idx ? { ...e, [field]: val } : e))
    setSaved(false)
  }

  async function save() {
    setBusy(true); setSaved(false)
    try {
      const env: Record<string, string> = {}
      for (const e of entries) {
        const k = e.key.trim()
        if (k) env[k] = e.value
      }
      await apiPut(`/api/v1/projects/${encodeURIComponent(project)}/agents/${encodeURIComponent(agentName)}/env`, { env, provider: selectedProvider })
      setSaved(true)
      setTimeout(() => setSaved(false), 2500)
      onChanged()
    } catch (e) { alert(String(e)) }
    finally { setBusy(false) }
  }

  const inputCls = 'h-8 rounded-md border border-neutral-200 bg-white px-2.5 text-sm text-neutral-700 outline-none hover:border-neutral-300 focus:border-sky-400 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 dark:[color-scheme:dark]'

  return (
    <div className="rounded-lg border border-neutral-200/80 bg-white p-4 dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold text-neutral-700 dark:text-zinc-300">
          {t('members.apiProvider')}
        </h3>
        <div className="flex items-center gap-2">
          {saved && <span className="text-xs text-emerald-500">{t('forms.saved')}</span>}
          <button type="button" onClick={save} disabled={busy}
            className="flex items-center gap-1 rounded-md bg-sky-600 px-2.5 py-1 text-xs font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50">
            <Save className="size-3" strokeWidth={2} />
            {busy ? t('forms.saving') : t('forms.save')}
          </button>
        </div>
      </div>

      {/* Provider selector */}
      {providerOptions.length > 0 && (
        <div className="mb-3">
          <label className="flex items-center gap-2">
            <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('provider.selectLabel')}</span>
            <select value={selectedProvider} onChange={e => { setSelectedProvider(e.target.value); setSaved(false) }}
              className={cn(inputCls, 'w-56 text-xs')}>
              <option value="">{t('provider.none')}</option>
              {providerOptions.map(p => (
                <option key={p.id} value={p.id}>{p.name} ({p.type}{p.model ? ` · ${p.model}` : ''})</option>
              ))}
            </select>
          </label>
          {selectedProvider && (
            <p className="mt-1 text-[11px] text-neutral-400 dark:text-zinc-500">{t('provider.overrideHint')}</p>
          )}
        </div>
      )}

      {wellKnown && (
        <p className="mb-3 text-xs text-neutral-400 dark:text-zinc-500">
          {wellKnown.hint}
        </p>
      )}

      {/* Quick-add buttons for well-known keys */}
      {wellKnown && (
        <div className="mb-3 flex flex-wrap gap-1.5">
          {wellKnown.keys.filter((k) => !usedKeys.has(k)).map((k) => (
            <button key={k} type="button" onClick={() => addEntry(k)}
              className="inline-flex items-center gap-1 rounded-md border border-dashed border-neutral-300 px-2 py-0.5 text-xs text-neutral-500 transition-colors hover:border-sky-400 hover:text-sky-600 dark:border-zinc-600 dark:text-zinc-500 dark:hover:border-sky-600 dark:hover:text-sky-400">
              <Plus className="size-3" strokeWidth={2} />
              {k}
            </button>
          ))}
        </div>
      )}

      {/* Entries */}
      <div className="space-y-2">
        {entries.map((entry, idx) => (
          <div key={idx} className="flex items-center gap-2">
            <input
              value={entry.key}
              onChange={(e) => updateEntry(idx, 'key', e.target.value)}
              placeholder="ENV_KEY"
              className={cn(inputCls, 'w-48 font-mono text-xs')}
              disabled={busy}
            />
            <span className="text-neutral-300 dark:text-zinc-500">=</span>
            <input
              value={entry.value}
              onChange={(e) => updateEntry(idx, 'value', e.target.value)}
              placeholder="value"
              type={entry.key.toLowerCase().includes('key') || entry.key.toLowerCase().includes('token') ? 'password' : 'text'}
              className={cn(inputCls, 'min-w-0 flex-1')}
              disabled={busy}
            />
            <button type="button" onClick={() => removeEntry(idx)} disabled={busy}
              className="rounded-md p-1.5 text-neutral-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:text-zinc-500 dark:hover:bg-red-900/20 dark:hover:text-red-400">
              <Trash2 className="size-3.5" strokeWidth={2} />
            </button>
          </div>
        ))}
      </div>

      <button type="button" onClick={() => addEntry()} disabled={busy}
        className="mt-2 inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300">
        <Plus className="size-3" strokeWidth={2} />
        {t('members.addEnvVar')}
      </button>
    </div>
  )
}

/* ── Human member panels ─────────────────────────────────────────────────── */

type HumanMsg = {
  id: string; from: string; to: string; subject?: string; body: string
  sentAt: string; readAt?: string; archivedAt?: string; mailbox: string
}

function HumanMessagesPanel({ project, member }: { project: string; member: string }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const [page, setPage] = useState(1)
  const perPage = 10
  const [reloadKey, setReloadKey] = useState(0)
  const [expanded, setExpanded] = useState<string | null>(null)
  const [replyTo, setReplyTo] = useState<string | null>(null)
  const [replyBody, setReplyBody] = useState('')
  const [replyBusy, setReplyBusy] = useState(false)

  const mailbox = `${project}/${member}`
  const state = useApiJson<HumanMsg[]>(`/api/v1/projects/${encodeURIComponent(project)}/messages?mailbox=${encodeURIComponent(mailbox)}&archived=all`, reloadKey)
  const messages = state.status === 'ok' ? (state.data ?? []) : []
  const totalPages = Math.ceil(messages.length / perPage)
  const paged = useMemo(() => messages.slice((page - 1) * perPage, page * perPage), [messages, page])

  async function sendReply(msg: HumanMsg) {
    if (!replyBody.trim()) return
    setReplyBusy(true)
    try {
      await apiPost('/api/v1/messages', { from: 'human', to: msg.from, body: replyBody.trim() })
      await apiPost('/api/v1/messages/mark-read', { mailbox: msg.mailbox, id: msg.id }).catch(() => {})
      setReplyBody('')
      setReplyTo(null)
      setReloadKey((k) => k + 1)
    } catch (e) { alert(String(e)) }
    finally { setReplyBusy(false) }
  }

  async function markRead(msg: HumanMsg) {
    await apiPost('/api/v1/messages/mark-read', { mailbox: msg.mailbox, id: msg.id })
    setReloadKey((k) => k + 1)
  }

  return (
    <section>
      <SectionHeader icon={Mail} title={t('workbench.tabMessages')} />
      <div className="mt-3 rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
        {state.status === 'loading' && (
          <div className="flex items-center gap-2 py-8 justify-center">
            <div className="size-4 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
          </div>
        )}
        {state.status === 'ok' && messages.length === 0 && (
          <div className="flex flex-col items-center py-10 text-center">
            <Mail className="mb-2 size-6 text-neutral-300 dark:text-zinc-600" strokeWidth={1.5} />
            <p className="text-sm text-neutral-400 dark:text-zinc-500">{t('workbench.emptyMessages')}</p>
          </div>
        )}
        {state.status === 'ok' && messages.length > 0 && (
          <div className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
            {paged.map((msg) => {
              const unread = !msg.readAt
              const isExpanded = expanded === msg.id
              const isReplying = replyTo === msg.id
              return (
                <div key={msg.id} className="px-4 py-3">
                  <div
                    className="flex cursor-pointer items-center gap-3"
                    onClick={() => { setExpanded(isExpanded ? null : msg.id); setReplyTo(null) }}
                  >
                    {unread && <span className="size-2 shrink-0 rounded-full bg-sky-500" />}
                    {!unread && <span className="size-2 shrink-0" />}
                    <span className="font-mono text-xs font-semibold text-neutral-700 dark:text-zinc-300">{msg.from}</span>
                    <span className="text-xs text-neutral-400 dark:text-zinc-500">→ {msg.to}</span>
                    <span className="ml-auto shrink-0 text-[11px] text-neutral-400 dark:text-zinc-500">{fmt(msg.sentAt)}</span>
                  </div>
                  {!isExpanded && (
                    <p className="mt-1 truncate pl-5 text-xs text-neutral-500 dark:text-zinc-500">
                      {msg.body.replace(/\s+/g, ' ').slice(0, 120)}
                    </p>
                  )}
                  {isExpanded && (
                    <div className="mt-2 pl-5">
                      <div className="prose prose-sm prose-neutral dark:prose-invert max-w-none rounded-md border border-neutral-100 bg-neutral-50/50 p-3 text-xs leading-relaxed dark:border-zinc-700/40 dark:bg-zinc-800/30 [&_pre]:overflow-x-auto">
                        <Markdown remarkPlugins={[remarkGfm]}>{msg.body}</Markdown>
                      </div>
                      <div className="mt-2 flex items-center gap-2">
                        {unread && (
                          <button type="button" onClick={() => void markRead(msg)}
                            className="rounded-md px-2 py-1 text-[11px] font-medium text-sky-700 hover:bg-sky-50 dark:text-sky-400 dark:hover:bg-sky-900/20">
                            {t('forms.markAsRead')}
                          </button>
                        )}
                        <button type="button" onClick={() => { setReplyTo(isReplying ? null : msg.id); setReplyBody('') }}
                          className="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium text-sky-700 hover:bg-sky-50 dark:text-sky-400 dark:hover:bg-sky-900/20">
                          <Reply className="size-3" strokeWidth={2} />
                          {t('workbench.replyTo')}
                        </button>
                      </div>
                      {isReplying && (
                        <div className="mt-2 rounded-md border border-sky-200/60 bg-sky-50/30 p-3 dark:border-sky-800/40 dark:bg-sky-950/20">
                          <textarea
                            autoFocus value={replyBody} onChange={(e) => setReplyBody(e.target.value)}
                            rows={3} placeholder={t('forms.body')}
                            className="block w-full resize-y rounded-md border border-neutral-200 bg-white p-2 text-xs outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200"
                          />
                          <div className="mt-2 flex justify-end gap-2">
                            <button type="button" onClick={() => setReplyTo(null)}
                              className="rounded-md px-2.5 py-1 text-[11px] text-neutral-500 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800">
                              {t('forms.cancel')}
                            </button>
                            <button type="button" disabled={replyBusy || !replyBody.trim()} onClick={() => void sendReply(msg)}
                              className="flex items-center gap-1 rounded-md bg-sky-600 px-3 py-1 text-[11px] font-medium text-white hover:bg-sky-700 disabled:opacity-50">
                              <Send className="size-3" strokeWidth={2} />
                              {replyBusy ? t('forms.sending') : t('forms.send')}
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
        <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
      </div>
    </section>
  )
}

type HumanTask = {
  id: string; project: string; agent: string; assignee?: string
  title: string; prompt?: string; status: string; priority: number
  type?: string; createdAt: string; updatedAt: string
}

const taskStatusCls: Record<string, string> = {
  pending: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
  running: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400',
  done_success: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
  done_failed: 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400',
  cancelled: 'bg-neutral-100 text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500',
  awaiting_confirmation: 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400',
}

function HumanTasksPanel({ project, member }: { project: string; member: string }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const [page, setPage] = useState(1)
  const perPage = 10
  const [expanded, setExpanded] = useState<string | null>(null)

  const state = useApiJson<HumanTask[]>(`/api/v1/projects/${encodeURIComponent(project)}/tasks?agent=${encodeURIComponent(member)}`)
  const tasks = state.status === 'ok' ? (state.data ?? []) : []
  const totalPages = Math.ceil(tasks.length / perPage)
  const paged = useMemo(() => tasks.slice((page - 1) * perPage, page * perPage), [tasks, page])

  return (
    <section>
      <SectionHeader icon={ListTodo} title={t('workbench.tabTasks')} />
      <div className="mt-3 rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
        {state.status === 'loading' && (
          <div className="flex items-center gap-2 py-8 justify-center">
            <div className="size-4 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
          </div>
        )}
        {state.status === 'ok' && tasks.length === 0 && (
          <div className="flex flex-col items-center py-10 text-center">
            <ListTodo className="mb-2 size-6 text-neutral-300 dark:text-zinc-600" strokeWidth={1.5} />
            <p className="text-sm text-neutral-400 dark:text-zinc-500">{t('workbench.emptyTasks')}</p>
          </div>
        )}
        {state.status === 'ok' && tasks.length > 0 && (
          <div className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
            {paged.map((task) => {
              const sCls = taskStatusCls[task.status] ?? taskStatusCls.pending
              const isExpanded = expanded === task.id
              return (
                <div key={task.id} className="px-4 py-3">
                  <div
                    className="flex cursor-pointer items-center gap-3"
                    onClick={() => setExpanded(isExpanded ? null : task.id)}
                  >
                    <span className={cn('shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold', sCls)}>
                      {t(`tasks.status.${task.status}`, { defaultValue: task.status })}
                    </span>
                    <span className="min-w-0 flex-1 truncate text-sm font-medium text-neutral-800 dark:text-zinc-200">{task.title}</span>
                    <span className="shrink-0 text-[11px] text-neutral-400 dark:text-zinc-500">{fmt(task.updatedAt)}</span>
                  </div>
                  {isExpanded && task.prompt && (
                    <div className="mt-2 pl-5">
                      <div className="prose prose-sm prose-neutral dark:prose-invert max-w-none rounded-md border border-neutral-100 bg-neutral-50/50 p-3 text-xs leading-relaxed dark:border-zinc-700/40 dark:bg-zinc-800/30 [&_pre]:overflow-x-auto">
                        <Markdown remarkPlugins={[remarkGfm]}>{task.prompt}</Markdown>
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
        <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
      </div>
    </section>
  )
}
