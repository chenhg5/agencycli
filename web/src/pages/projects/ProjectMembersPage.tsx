import { useCallback, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Users, RefreshCw, Save, ChevronDown, ChevronRight, Bot, BookOpen, Puzzle } from 'lucide-react'
import { HireAgentDialog } from '../../components/project/HireAgentDialog'
import type { LucideIcon } from 'lucide-react'
import { cn } from '../../lib/cn'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'
import { apiPost, apiPut } from '../../lib/api'

type SessionInfo = { sessionId?: string; sessionStartedAt?: string; sessionScope?: string }

type AgentRow = {
  name: string
  model: string
  team: string
  project: string
  hiredAt: string
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

function AgentDetail({ project, agentName }: { project: string; agentName: string }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()

  const ctxPath = `/api/v1/projects/${encodeURIComponent(project)}/agents/${encodeURIComponent(agentName)}/context`
  const [ctxReload, setCtxReload] = useState(0)
  const ctxState = useApiJson<AgentContext>(ctxPath, ctxReload)

  const [syncing, setSyncing] = useState(false)
  const [syncOutput, setSyncOutput] = useState<string | null>(null)

  const doSync = useCallback(async () => {
    setSyncing(true)
    setSyncOutput(null)
    try {
      const res = await apiPost<{ ok: boolean; output: string }>(`/api/v1/projects/${encodeURIComponent(project)}/sync`, { agent: agentName })
      setSyncOutput(res.output || 'Sync completed.')
      setCtxReload((k) => k + 1)
    } catch (e) {
      setSyncOutput(String(e))
    } finally {
      setSyncing(false)
    }
  }, [project, agentName, ctxState])

  if (ctxState.status === 'loading') {
    return (
      <div className="flex items-center gap-2 py-4 pl-4">
        <div className="size-4 animate-spin rounded-full border-2 border-neutral-200 border-t-sky-600 dark:border-zinc-700 dark:border-t-sky-400" />
        <span className="text-xs text-neutral-400">{t('api.loading')}</span>
      </div>
    )
  }
  if (ctxState.status === 'error') {
    return <p className="py-3 pl-4 text-xs text-red-500">{ctxState.error.message}</p>
  }

  const ctx = ctxState.data

  return (
    <div className="space-y-4 pb-2 pt-1">
      {/* Meta */}
      <div className="flex flex-wrap items-center gap-x-5 gap-y-1 text-xs text-neutral-500 dark:text-zinc-500">
        {ctx.team && <span>{t('prompt.team')}: <span className="font-medium text-neutral-700 dark:text-zinc-300">{ctx.team}</span></span>}
        {ctx.role && <span>{t('prompt.role')}: <span className="font-medium text-neutral-700 dark:text-zinc-300">{ctx.role}</span></span>}
        <span>{t('prompt.contextFile')}: <span className="font-mono text-neutral-700 dark:text-zinc-300">{ctx.contextFile}</span></span>
        {ctx.syncedAt && <span>{t('prompt.lastSync')}: {fmt(ctx.syncedAt)}</span>}
      </div>

      {/* Skills */}
      {ctx.skills && ctx.skills.length > 0 && (
        <div>
          <h4 className="mb-1.5 flex items-center gap-1.5 text-xs font-semibold text-neutral-500 dark:text-zinc-500">
            <Puzzle className="size-3.5 text-amber-500" strokeWidth={2} />
            {t('skill.agentSkills')}
          </h4>
          <div className="flex flex-wrap gap-1.5">
            {ctx.skills.map((sk) => (
              <span
                key={sk}
                className="inline-flex items-center gap-1 rounded-md bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-400"
              >
                <Puzzle className="size-3" strokeWidth={2} />
                {sk}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Session info */}
      <SessionPanel project={project} agentName={agentName} />

      {/* Sync button */}
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={doSync}
          disabled={syncing}
          className="flex items-center gap-1.5 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-xs font-medium text-neutral-700 transition-colors hover:border-neutral-300 hover:bg-neutral-50 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:border-zinc-600"
        >
          <RefreshCw className={cn('size-3.5', syncing && 'animate-spin')} strokeWidth={2} />
          {syncing ? t('prompt.syncing') : t('prompt.sync')}
        </button>
        {syncOutput && (
          <span className="text-xs text-neutral-500 dark:text-zinc-500">{syncOutput}</span>
        )}
      </div>

      {/* Merged context (read-only) */}
      {ctx.context && (
        <details className="group">
          <summary className="flex items-center gap-1.5 text-xs font-medium text-neutral-500 dark:text-zinc-500">
            <ChevronRight className="size-3.5 transition-transform group-open:rotate-90" strokeWidth={2} />
            {t('prompt.mergedContext')} ({ctx.contextFile})
          </summary>
          <div className="mt-2 max-h-[40vh] overflow-auto rounded-md border border-neutral-200/80 bg-neutral-50 p-4 font-mono text-xs leading-relaxed text-neutral-600 dark:border-zinc-800/60 dark:bg-zinc-950 dark:text-zinc-400">
            <Markdown remarkPlugins={[remarkGfm]}>{ctx.context}</Markdown>
          </div>
        </details>
      )}

      {/* Wakeup prompt (editable) */}
      <PromptEditor
        label={t('prompt.wakeup')}
        icon={BookOpen}
        apiPath={`/api/v1/projects/${encodeURIComponent(project)}/agents/${encodeURIComponent(agentName)}/wakeup`}
        initialContent={ctx.wakeup}
      />
    </div>
  )
}

export default function ProjectMembersPage() {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const { projectId } = useParams<{ projectId: string }>()

  const [reloadKey, setReloadKey] = useState(0)
  const agentsPath =
    projectId != null && projectId !== ''
      ? `/api/v1/projects/${encodeURIComponent(projectId)}/agents`
      : null
  const agentsState = useApiJson<AgentRow[]>(agentsPath, reloadKey)
  const members = agentsState.status === 'ok' ? (agentsState.data ?? []) : []

  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  const toggle = (name: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-5 pb-3">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('projectNav.members')}</h1>
            <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('members.subtitle')}</p>
          </div>
          {projectId && (
            <HireAgentDialog
              projectId={projectId}
              onHired={() => setReloadKey((k) => k + 1)}
            />
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-6 pb-6">
        {agentsState.status === 'loading' && (
          <div className="flex items-center gap-2 py-12 justify-center">
            <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {agentsState.status === 'error' && (
          <PlaceholderCard title={t('api.loadError')}>
            <p className="text-[13px]">{agentsState.error.message}</p>
          </PlaceholderCard>
        )}
        {agentsState.status === 'ok' && members.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="mb-3 flex size-14 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <Users className="size-6 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-base font-medium text-neutral-600 dark:text-zinc-400">{t('members.emptyTitle')}</p>
          </div>
        )}
        {agentsState.status === 'ok' && members.length > 0 && (
          <div className="space-y-3">
            {members.map((row) => {
              const isOpen = expanded.has(row.name)
              return (
                <div
                  key={row.name}
                  className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-800/60 dark:bg-zinc-900/40"
                >
                  <button
                    type="button"
                    onClick={() => toggle(row.name)}
                    className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-neutral-50/80 dark:hover:bg-zinc-800/30"
                  >
                    <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/30">
                      <Bot className="size-4 text-violet-700 dark:text-violet-400" strokeWidth={2} />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="font-mono text-sm font-medium text-neutral-900 dark:text-zinc-100">{row.name}</p>
                      <p className="mt-0.5 text-xs text-neutral-400 dark:text-zinc-600">
                        {row.model} · {row.team} · {t('prompt.hired')} {fmt(row.hiredAt)}
                      </p>
                    </div>
                    {isOpen
                      ? <ChevronDown className="size-4 shrink-0 text-neutral-400 dark:text-zinc-600" strokeWidth={2} />
                      : <ChevronRight className="size-4 shrink-0 text-neutral-400 dark:text-zinc-600" strokeWidth={2} />
                    }
                  </button>
                  {isOpen && projectId && (
                    <div className="border-t border-neutral-100 px-4 py-3 dark:border-zinc-800/40">
                      <AgentDetail project={projectId} agentName={row.name} />
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
