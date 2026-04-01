import { useCallback, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ArrowLeft, RefreshCw, Save, ChevronRight, Bot, BookOpen, Puzzle, Check, Plus, Trash2 } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '../../lib/cn'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'
import { apiPost, apiPut } from '../../lib/api'

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
}

type SessionInfo = { sessionId?: string; sessionStartedAt?: string; sessionScope?: string }

type HTTPAgentConfig = {
  url?: string
  model?: string
  api_key?: string
  timeout?: string
  stream?: boolean
}

type AgentContext = {
  contextFile: string
  context: string
  wakeup: string
  model: string
  team: string
  role: string
  syncedAt: string | null
  skills: string[]
  httpAgent?: HTTPAgentConfig
  env?: Record<string, string>
}

const WELL_KNOWN_ENV: Record<string, { keys: string[]; hint: string }> = {
  claudecode: {
    keys: ['ANTHROPIC_MODEL', 'ANTHROPIC_BASE_URL', 'ANTHROPIC_AUTH_TOKEN'],
    hint: 'e.g. claude-sonnet-4-20250514, claude-opus-4-20250514',
  },
  codex: {
    keys: ['OPENAI_API_KEY', 'OPENAI_MODEL', 'OPENAI_BASE_URL'],
    hint: 'e.g. o3, gpt-4.1',
  },
  gemini: {
    keys: ['GEMINI_API_KEY', 'GOOGLE_API_KEY', 'GOOGLE_CLOUD_PROJECT'],
    hint: 'e.g. gemini-2.5-pro',
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
    <div className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-800/60 dark:bg-zinc-900/40">
      <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-2.5 dark:border-zinc-800/40">
        <div className="flex items-center gap-2">
          <Icon className="size-4 text-neutral-400 dark:text-zinc-600" strokeWidth={1.8} />
          <span className="text-sm font-medium text-neutral-700 dark:text-zinc-300">{label}</span>
          {dirty && <span className="text-[10px] text-amber-500">●</span>}
          {saved && <span className="text-[10px] text-emerald-500">{t('prompt.saved')}</span>}
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => setPreview((p) => !p)} className={cn('rounded-md px-2 py-1 text-[11px] font-medium transition-colors', preview ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400' : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-600 dark:hover:text-zinc-400')}>
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
    <div className="rounded-lg border border-neutral-200/80 bg-neutral-50/50 px-4 py-3 dark:border-zinc-800/60 dark:bg-zinc-900/30">
      <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">{t('session.sessionLabel')}</h4>
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-xs">
        <span className="text-neutral-500 dark:text-zinc-500">
          Session ID: {hasSession ? <span className="font-mono text-emerald-700 dark:text-emerald-400" title={info.sessionId}>{info.sessionId!.slice(0, 16)}…</span> : <span className="text-neutral-400 dark:text-zinc-600">{t('session.noSession')}</span>}
        </span>
        {info.sessionStartedAt && (
          <span className="text-neutral-500 dark:text-zinc-500">{t('session.startedAt')}: {fmt(info.sessionStartedAt)}</span>
        )}
        <span className="text-neutral-500 dark:text-zinc-500">{t('session.scopeLabel')}: <span className="font-medium text-neutral-700 dark:text-zinc-300">{info.sessionScope === 'task' ? t('session.scopeTask') : t('session.scopeCycle')}</span></span>

        <div className="flex items-center gap-2">
          {hasSession && (
            <button type="button" onClick={() => void doReset()} disabled={resetting}
              className="cursor-pointer rounded-md border border-amber-200 bg-white px-2.5 py-1 text-xs font-medium text-amber-700 transition-colors hover:bg-amber-50 disabled:opacity-50 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-400">
              {resetting ? t('session.resettingSession') : t('session.resetSession')}
            </button>
          )}
          <button type="button" onClick={() => void doRun()} disabled={running}
            className="cursor-pointer rounded-md border border-sky-200 bg-white px-2.5 py-1 text-xs font-medium text-sky-700 transition-colors hover:bg-sky-50 disabled:opacity-50 dark:border-sky-800 dark:bg-sky-900/30 dark:text-sky-400">
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

  const inputCls = 'h-7 rounded-md border border-neutral-200 bg-white px-2 text-xs text-neutral-700 outline-none hover:border-neutral-300 focus:border-sky-400 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300'

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
  const { projectId, agentName } = useParams<{ projectId: string; agentName: string }>()

  const ctxPath = projectId && agentName
    ? `/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/context`
    : null
  const [ctxReload, setCtxReload] = useState(0)
  const ctxState = useApiJson<AgentContext>(ctxPath, ctxReload)

  const [syncing, setSyncing] = useState(false)
  const [syncOutput, setSyncOutput] = useState<string | null>(null)

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

  if (!projectId || !agentName) return null

  const modelCls = MODEL_COLORS[ctxState.status === 'ok' ? ctxState.data.model : ''] ?? ''

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Header */}
      <div className="shrink-0 px-6 pt-5 pb-4">
        <Link
          to={`/projects/${encodeURIComponent(projectId)}/members`}
          className="mb-3 inline-flex items-center gap-1.5 text-sm text-neutral-500 transition-colors hover:text-neutral-700 dark:text-zinc-500 dark:hover:text-zinc-300"
        >
          <ArrowLeft className="size-3.5" strokeWidth={2} />
          {t('projectNav.members')}
        </Link>
        <div className="flex items-center gap-4">
          <div className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-violet-100 dark:bg-violet-900/30">
            <Bot className="size-6 text-violet-600 dark:text-violet-400" strokeWidth={1.8} />
          </div>
          <div className="min-w-0 flex-1">
            <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{agentName}</h1>
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
        {ctxState.status === 'ok' && (() => {
          const ctx = ctxState.data
          return (
            <div className="space-y-6">
              {/* Info grid */}
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                {ctx.team && (
                  <InfoCard label={t('prompt.team')} value={ctx.team} />
                )}
                {ctx.role && (
                  <InfoCard label={t('prompt.role')} value={ctx.role} />
                )}
                <InfoCard label={t('prompt.contextFile')} value={ctx.contextFile} mono />
                {ctx.syncedAt && (
                  <InfoCard label={t('prompt.lastSync')} value={fmt(ctx.syncedAt)} />
                )}
              </div>

              {/* Agent type selector */}
              <div className="rounded-lg border border-neutral-200/80 bg-white p-4 dark:border-zinc-800/60 dark:bg-zinc-900/40">
                <ModelSelector
                  project={projectId}
                  agentName={agentName}
                  currentModel={ctx.model}
                  currentHttpAgent={ctx.httpAgent}
                  onChanged={() => setCtxReload((k) => k + 1)}
                />
              </div>

              {/* API Provider env vars */}
              <EnvEditor
                project={projectId}
                agentName={agentName}
                model={ctx.model}
                initialEnv={ctx.env ?? {}}
                onChanged={() => setCtxReload((k) => k + 1)}
              />

              {/* Skills */}
              {ctx.skills && ctx.skills.length > 0 && (
                <div>
                  <h3 className="mb-2 flex items-center gap-1.5 text-sm font-semibold text-neutral-700 dark:text-zinc-300">
                    <Puzzle className="size-4 text-amber-500" strokeWidth={2} />
                    {t('skill.agentSkills')}
                  </h3>
                  <div className="flex flex-wrap gap-2">
                    {ctx.skills.map((sk) => (
                      <span key={sk} className="inline-flex items-center gap-1.5 rounded-md bg-amber-50 px-2.5 py-1 text-sm font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-400">
                        <Puzzle className="size-3.5" strokeWidth={2} />
                        {sk}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Session */}
              <SessionPanel project={projectId} agentName={agentName} />

              {/* Sync */}
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={doSync}
                  disabled={syncing}
                  className="flex items-center gap-1.5 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm font-medium text-neutral-700 transition-colors hover:border-neutral-300 hover:bg-neutral-50 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:border-zinc-600"
                >
                  <RefreshCw className={cn('size-4', syncing && 'animate-spin')} strokeWidth={2} />
                  {syncing ? t('prompt.syncing') : t('prompt.sync')}
                </button>
                {syncOutput && (
                  <span className="text-sm text-neutral-500 dark:text-zinc-500">{syncOutput}</span>
                )}
              </div>

              {/* Merged context */}
              {ctx.context && (
                <details className="group">
                  <summary className="flex cursor-pointer items-center gap-1.5 text-sm font-medium text-neutral-600 dark:text-zinc-400">
                    <ChevronRight className="size-4 transition-transform group-open:rotate-90" strokeWidth={2} />
                    {t('prompt.mergedContext')} ({ctx.contextFile})
                  </summary>
                  <div className="mt-2 max-h-[50vh] overflow-auto rounded-lg border border-neutral-200/80 bg-neutral-50 p-4 font-mono text-sm leading-relaxed text-neutral-600 dark:border-zinc-800/60 dark:bg-zinc-950 dark:text-zinc-400">
                    <Markdown remarkPlugins={[remarkGfm]}>{ctx.context}</Markdown>
                  </div>
                </details>
              )}

              {/* Wakeup prompt */}
              <PromptEditor
                label={t('prompt.wakeup')}
                icon={BookOpen}
                apiPath={`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/wakeup`}
                initialContent={ctx.wakeup}
              />
            </div>
          )
        })()}
      </div>
    </div>
  )
}

function InfoCard({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="rounded-lg border border-neutral-200/80 bg-neutral-50/50 px-4 py-3 dark:border-zinc-800/60 dark:bg-zinc-900/30">
      <p className="text-xs font-medium text-neutral-400 dark:text-zinc-600">{label}</p>
      <p className={cn('mt-0.5 text-sm font-medium text-neutral-800 dark:text-zinc-200', mono && 'font-mono text-xs')}>{value}</p>
    </div>
  )
}

function EnvEditor({ project, agentName, model, initialEnv, onChanged }: {
  project: string; agentName: string; model: string; initialEnv: Record<string, string>; onChanged: () => void
}) {
  const { t } = useTranslation()
  const [entries, setEntries] = useState<{ key: string; value: string }[]>(() => {
    const items = Object.entries(initialEnv).map(([key, value]) => ({ key, value }))
    return items.length > 0 ? items : []
  })
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState(false)

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
      await apiPut(`/api/v1/projects/${encodeURIComponent(project)}/agents/${encodeURIComponent(agentName)}/env`, { env })
      setSaved(true)
      setTimeout(() => setSaved(false), 2500)
      onChanged()
    } catch (e) { alert(String(e)) }
    finally { setBusy(false) }
  }

  const inputCls = 'h-8 rounded-md border border-neutral-200 bg-white px-2.5 text-sm text-neutral-700 outline-none hover:border-neutral-300 focus:border-sky-400 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300'

  return (
    <div className="rounded-lg border border-neutral-200/80 bg-white p-4 dark:border-zinc-800/60 dark:bg-zinc-900/40">
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

      {wellKnown && (
        <p className="mb-3 text-xs text-neutral-400 dark:text-zinc-600">
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
            <span className="text-neutral-300 dark:text-zinc-700">=</span>
            <input
              value={entry.value}
              onChange={(e) => updateEntry(idx, 'value', e.target.value)}
              placeholder="value"
              type={entry.key.toLowerCase().includes('key') || entry.key.toLowerCase().includes('token') ? 'password' : 'text'}
              className={cn(inputCls, 'min-w-0 flex-1')}
              disabled={busy}
            />
            <button type="button" onClick={() => removeEntry(idx)} disabled={busy}
              className="rounded-md p-1.5 text-neutral-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:text-zinc-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">
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
