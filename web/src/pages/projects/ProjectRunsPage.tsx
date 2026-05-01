import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { BarChart3, FileText, MessageSquareText, RefreshCw, StopCircle, X } from 'lucide-react'
import { Pagination } from '../../components/ui/Pagination'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { ConversationLog } from '../../components/ui/ConversationLog'
import { cn } from '../../lib/cn'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'
import { apiPost } from '../../lib/api'

type Summary = {
  runs: number
  taskRuns: number
  execRuns: number
  inputTokens: number
  outputTokens: number
  cacheReadTokens: number
  costUSD: number
  success: number
  failed: number
  awaiting: number
  other: number
  wallDurationMs: number
}

type AgentSum = {
  project: string
  agent: string
  runs: number
  inputTokens: number
  outputTokens: number
  cacheReadTokens: number
  costUSD: number
  success: number
  failed: number
}

type TelemetrySummary = {
  available: boolean
  window: { from?: string; to?: string; allTime?: boolean }
  summary: Summary | null
  byAgent: AgentSum[]
}

type RunRow = {
  project: string
  agent: string
  kind: string
  status: string
  startedAt: string
  finishedAt: string
  inputTokens?: number
  outputTokens?: number
  cacheReadTokens?: number
  costUSD?: number
  model?: string
  apiModel?: string
  apiBaseUrl?: string
  command?: string
  logPath?: string
  sessionId?: string
  errorMsg?: string
  taskId?: string
  taskTitle?: string
}

type TelemetryRuns = {
  available: boolean
  window: { from?: string; to?: string; allTime?: boolean }
  runs: RunRow[]
}

type LogData = {
  content: string
  truncated: boolean
}

const windowKeys = ['today', '7d', '30d', 'all'] as const
type WindowKey = (typeof windowKeys)[number]

function fmtNum(n: number) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}

const statusCls: Record<string, string> = {
  success: 'text-emerald-700 dark:text-emerald-400',
  done_success: 'text-emerald-700 dark:text-emerald-400',
  failed: 'text-red-600 dark:text-red-400',
  done_failed: 'text-red-600 dark:text-red-400',
  error: 'text-red-600 dark:text-red-400',
  running: 'text-sky-700 dark:text-sky-400',
  aborted: 'text-orange-600 dark:text-orange-400',
  awaiting: 'text-amber-700 dark:text-amber-400',
  awaiting_confirmation: 'text-amber-700 dark:text-amber-400',
}

function isFailed(status: string) {
  return status === 'failed' || status === 'done_failed' || status === 'error'
}

const kindStyles: Record<string, { text: string; cls: string }> = {
  wakeup: { text: 'Wakeup', cls: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400' },
  chat: { text: 'Chat', cls: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' },
  task: { text: 'Task', cls: 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400' },
}

function resolveKind(r: RunRow): string {
  if (r.taskTitle?.includes('[wakeup]')) return 'wakeup'
  if (r.kind === 'exec') return 'chat'
  return 'task'
}

const filterSelect =
  'rounded-lg border border-neutral-200/80 bg-white px-3 py-1.5 text-sm text-neutral-700 outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-900 dark:text-zinc-300 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-900 [&>option]:dark:text-zinc-300'

export default function ProjectRunsPage() {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const { projectId } = useParams<{ projectId: string }>()
  const [searchParams] = useSearchParams()
  const [windowKey, setWindowKey] = useState<WindowKey>('7d')
  const [selectedRun, setSelectedRun] = useState<RunRow | null>(null)
  const [filterKind, setFilterKind] = useState<string>('all')
  const [filterStatus, setFilterStatus] = useState<string>('all')
  const [filterAgent, setFilterAgent] = useState<string>(() => searchParams.get('agent') ?? 'all')
  const [runsPage, setRunsPage] = useState(1)
  const runsPerPage = 20

  const summaryQuery = useMemo(() => {
    const p = new URLSearchParams()
    if (projectId) p.set('project', projectId)
    if (windowKey === 'all') p.set('allTime', '1')
    else if (windowKey === 'today') p.set('since', 'today')
    else if (windowKey === '7d') p.set('since', '7d')
    else p.set('since', '30d')
    return `/api/v1/telemetry/summary?${p}`
  }, [projectId, windowKey])

  const runsQuery = useMemo(() => {
    const p = new URLSearchParams()
    if (projectId) p.set('project', projectId)
    p.set('limit', '200')
    if (windowKey === 'all') p.set('allTime', '1')
    else if (windowKey === 'today') p.set('since', 'today')
    else if (windowKey === '7d') p.set('since', '7d')
    else p.set('since', '30d')
    return `/api/v1/telemetry/runs?${p}`
  }, [projectId, windowKey])

  const [reloadKey, setReloadKey] = useState(0)
  const refresh = useCallback(() => setReloadKey((k) => k + 1), [])
  const [abortingAgent, setAbortingAgent] = useState<string | null>(null)

  async function doAbort(project: string, agent: string) {
    if (!window.confirm(t('schedule.confirmAbort'))) return
    setAbortingAgent(`${project}/${agent}`)
    try {
      await apiPost('/api/v1/scheduler/abort', { project, agent })
      refresh()
    } catch { /* toast */ }
    finally { setAbortingAgent(null) }
  }
  const sumState = useApiJson<TelemetrySummary>(summaryQuery, reloadKey)
  const runsState = useApiJson<TelemetryRuns>(runsQuery, reloadKey)

  const allRuns = runsState.status === 'ok' && runsState.data.available ? runsState.data.runs : []

  const agentOptions = useMemo(() => {
    const set = new Set<string>()
    for (const r of allRuns) set.add(`${r.project}/${r.agent}`)
    return Array.from(set).sort()
  }, [allRuns])

  const statusOptions = useMemo(() => {
    const set = new Set<string>()
    for (const r of allRuns) set.add(r.status)
    return Array.from(set).sort()
  }, [allRuns])

  const filteredRuns = useMemo(() => {
    return allRuns.filter((r) => {
      if (filterKind !== 'all' && resolveKind(r) !== filterKind) return false
      if (filterStatus !== 'all' && r.status !== filterStatus) return false
      if (filterAgent !== 'all' && `${r.project}/${r.agent}` !== filterAgent) return false
      return true
    })
  }, [allRuns, filterKind, filterStatus, filterAgent])

  const totalRunPages = Math.ceil(filteredRuns.length / runsPerPage)
  const pagedRuns = useMemo(() => {
    const start = (runsPage - 1) * runsPerPage
    return filteredRuns.slice(start, start + runsPerPage)
  }, [filteredRuns, runsPage])

  useEffect(() => {
    setRunsPage(1)
  }, [filterKind, filterStatus, filterAgent, windowKey])

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Header */}
      <div className="shrink-0 px-6 pt-5 pb-3">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('projectNav.runs')}</h1>
            <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('runs.subtitle')}</p>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1">
              {windowKeys.map((k) => (
                <button
                  key={k}
                  type="button"
                  onClick={() => setWindowKey(k)}
                  className={cn(
                    'rounded-md px-3 py-1.5 text-xs font-medium transition-all duration-150',
                    windowKey === k
                      ? 'bg-sky-600 text-white shadow-sm'
                      : 'text-neutral-500 hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-800',
                  )}
                >
                  {t(`runs.window.${k}`)}
                </button>
              ))}
            </div>
            <button type="button" onClick={refresh} className="flex items-center gap-1 rounded-md px-2 py-1 text-[13px] text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
              <RefreshCw className="size-3" strokeWidth={2} />
              {t('api.refresh')}
            </button>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-6 py-3">
        {sumState.status === 'loading' && (
          <div className="flex items-center gap-2 py-16 justify-center">
            <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {sumState.status === 'error' && (
          <PlaceholderCard title={t('api.loadError')}>{sumState.error.message}</PlaceholderCard>
        )}
        {sumState.status === 'ok' && !sumState.data.available && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <BarChart3 className="size-6 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-base font-medium text-neutral-700 dark:text-zinc-300">{t('runs.noDbTitle')}</p>
            <p className="mt-1 text-sm text-neutral-400 dark:text-zinc-500">{t('runs.noDbBody')}</p>
          </div>
        )}

        {/* Summary metrics */}
        {sumState.status === 'ok' && sumState.data.available && sumState.data.summary && (
          <div className="grid gap-3 pb-4 sm:grid-cols-2 lg:grid-cols-4">
            {(
              [
                ['runs', sumState.data.summary.runs, false],
                ['inputTokens', sumState.data.summary.inputTokens, true],
                ['outputTokens', sumState.data.summary.outputTokens, true],
                ['costUSD', sumState.data.summary.costUSD, false],
              ] as const
            ).map(([key, val, shouldFmt]) => (
              <div
                key={key}
                className="rounded-lg border border-neutral-200/80 bg-white px-4 py-3 dark:border-zinc-700/60 dark:bg-zinc-900/40"
              >
                <p className="text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
                  {t(`runs.metric.${key}`)}
                </p>
                <p className="mt-1 text-2xl font-bold tabular-nums text-neutral-900 dark:text-zinc-100">
                  {key === 'costUSD' ? `$${(val as number).toFixed(4)}` : shouldFmt ? fmtNum(val as number) : val}
                </p>
              </div>
            ))}
          </div>
        )}

        {/* By agent breakdown */}
        {sumState.status === 'ok' && sumState.data.available && sumState.data.byAgent.length > 0 && (
          <div className="mb-5 overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
            <table className="min-w-full w-full">
              <thead>
                <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-700/60 dark:bg-zinc-900/40">
                  {[t('runs.colAgent'), t('runs.colRuns'), t('runs.colInTok'), t('runs.colOutTok'), t('runs.colCost')].map((h) => (
                    <th key={h} className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
                {sumState.data.byAgent.map((a) => (
                  <tr key={`${a.project}/${a.agent}`} className="bg-white transition-colors duration-100 hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30">
                    <td className="whitespace-nowrap px-4 py-3 text-center font-mono text-sm text-neutral-800 dark:text-zinc-300">{a.project}/{a.agent}</td>
                    <td className="px-4 py-3 text-center text-sm tabular-nums text-neutral-700 dark:text-zinc-400">{a.runs}</td>
                    <td className="px-4 py-3 text-center text-sm tabular-nums text-neutral-700 dark:text-zinc-400">{fmtNum(a.inputTokens)}</td>
                    <td className="px-4 py-3 text-center text-sm tabular-nums text-neutral-700 dark:text-zinc-400">{fmtNum(a.outputTokens)}</td>
                    <td className="px-4 py-3 text-center text-sm tabular-nums text-neutral-700 dark:text-zinc-400">${a.costUSD.toFixed(4)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Recent runs */}
        <div className="flex flex-col gap-3 pb-3 pt-1 sm:flex-row sm:items-center sm:justify-between">
          <span className="text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
            {t('runs.recentTitle')}
            {filteredRuns.length !== allRuns.length && (
              <span className="ml-2 text-neutral-300 dark:text-zinc-500">
                {filteredRuns.length}/{allRuns.length}
              </span>
            )}
          </span>
          <div className="flex flex-wrap items-center gap-2">
            <select value={filterKind} onChange={(e) => setFilterKind(e.target.value)} className={filterSelect}>
              <option value="all">{t('runs.filterKind')}</option>
              <option value="wakeup">Wakeup</option>
              <option value="chat">Chat</option>
              <option value="task">Task</option>
            </select>
            <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)} className={filterSelect}>
              <option value="all">{t('runs.filterStatus')}</option>
              {statusOptions.map((s) => (
                <option key={s} value={s}>{t(`runs.status.${s}`, { defaultValue: s })}</option>
              ))}
            </select>
            {agentOptions.length > 1 && (
              <select value={filterAgent} onChange={(e) => setFilterAgent(e.target.value)} className={filterSelect}>
                <option value="all">{t('runs.filterAgent')}</option>
                {agentOptions.map((a) => (
                  <option key={a} value={a}>{a}</option>
                ))}
              </select>
            )}
          </div>
        </div>
        {runsState.status === 'ok' && runsState.data.available && filteredRuns.length === 0 && (
          <p className="py-8 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('runs.noRuns')}</p>
        )}
        {runsState.status === 'ok' && runsState.data.available && filteredRuns.length > 0 && (
          <>
          <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
            <table className="min-w-[700px] w-full">
              <thead>
                <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-700/60 dark:bg-zinc-900/40">
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('runs.colStarted')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('runs.colAgent')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('runs.colKind')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('runs.colStatus')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('runs.colTok')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
                {pagedRuns.map((r, i) => {
                  const rk = resolveKind(r)
                  const kl = kindStyles[rk] ?? { text: rk, cls: 'bg-neutral-100 text-neutral-600 dark:bg-zinc-800 dark:text-zinc-500' }
                  return (
                    <tr
                      key={`${r.startedAt}-${i}`}
                      className="cursor-pointer bg-white transition-colors duration-100 hover:bg-sky-50/40 dark:bg-zinc-900/20 dark:hover:bg-sky-900/[0.06]"
                      onClick={() => setSelectedRun(r)}
                    >
                      <td className="whitespace-nowrap px-4 py-3 text-center text-[13px] text-neutral-500 dark:text-zinc-500">{fmt(r.startedAt)}</td>
                      <td className="whitespace-nowrap px-4 py-3 text-center font-mono text-[13px] text-neutral-800 dark:text-zinc-300">{r.project}/{r.agent}</td>
                      <td className="whitespace-nowrap px-4 py-3 text-center">
                        <span className={cn('rounded-full px-2 py-0.5 text-[11px] font-semibold', kl.cls)}>
                          {kl.text}
                        </span>
                        {r.taskTitle && (
                          <span className="ml-1.5 text-[13px] text-neutral-500 dark:text-zinc-500">{r.taskTitle}</span>
                        )}
                      </td>
                      <td className={cn('px-4 py-3 text-center text-[13px] font-medium', statusCls[r.status] ?? 'text-neutral-600 dark:text-zinc-400')}>
                        <span>{t(`runs.status.${r.status}`, { defaultValue: r.status })}</span>
                        {r.status === 'running' && (
                          <button type="button"
                            disabled={abortingAgent === `${r.project}/${r.agent}`}
                            onClick={(e) => { e.stopPropagation(); void doAbort(r.project, r.agent) }}
                            className="ml-2 inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[11px] font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-40 dark:text-red-400 dark:hover:bg-red-900/20">
                            <StopCircle className="size-3" strokeWidth={2} />
                            {t('schedule.abort')}
                          </button>
                        )}
                        {isFailed(r.status) && r.errorMsg && (
                          <p className="mt-0.5 max-w-[240px] truncate text-[11px] font-normal text-red-500/80 dark:text-red-400/70" title={r.errorMsg}>{r.errorMsg}</p>
                        )}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-center text-[13px] tabular-nums text-neutral-500 dark:text-zinc-500">
                        {fmtNum((r.inputTokens ?? 0) + (r.outputTokens ?? 0) + (r.cacheReadTokens ?? 0))}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
          <Pagination page={runsPage} totalPages={totalRunPages} onPageChange={setRunsPage} />
          </>
        )}
      </div>

      {/* Run detail modal */}
      {selectedRun && (
        <RunDetailModal run={selectedRun} onClose={() => setSelectedRun(null)} />
      )}
    </div>
  )
}

function RunDetailModal({ run, onClose }: { run: RunRow; onClose: () => void }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const rk = resolveKind(run)
  const kl = kindStyles[rk] ?? { text: rk, cls: 'bg-neutral-100 text-neutral-600' }

  const hasLog = Boolean(run.logPath)
  const logQuery = hasLog ? `/api/v1/telemetry/log?path=${encodeURIComponent(run.logPath!)}` : null
  const logState = useApiJson<LogData>(logQuery, 0)

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[8vh]">
      <div className="absolute inset-0 bg-black/30 backdrop-blur-[2px] animate-fade-in dark:bg-black/50" onClick={onClose} />
      <div className="relative w-full max-w-3xl max-h-[80vh] flex flex-col overflow-hidden rounded-xl border border-neutral-200/80 bg-white shadow-2xl animate-scale-in dark:border-zinc-700/80 dark:bg-zinc-900">
        {/* Modal header */}
        <div className="flex items-center justify-between border-b border-neutral-200/80 px-5 py-3 dark:border-zinc-700/60">
          <div className="flex items-center gap-3">
            <span className={cn('rounded-full px-2 py-0.5 text-[11px] font-semibold', kl.cls)}>{kl.text}</span>
            <span className="font-mono text-sm font-medium text-neutral-900 dark:text-zinc-100">{run.project}/{run.agent}</span>
            <span className={cn('text-sm font-medium', statusCls[run.status] ?? 'text-neutral-600')}>{t(`runs.status.${run.status}`, { defaultValue: run.status })}</span>
          </div>
          <button type="button" onClick={onClose} className="rounded-md p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300">
            <X className="size-4" strokeWidth={2} />
          </button>
        </div>

        {/* Info grid */}
        <div className="shrink-0 grid grid-cols-2 gap-x-6 gap-y-2 border-b border-neutral-100 px-5 py-3 text-sm dark:border-zinc-700/40 sm:grid-cols-3">
          <div>
            <span className="text-xs font-medium text-neutral-400 dark:text-zinc-500">{t('runs.colStarted')}</span>
            <p className="text-neutral-800 dark:text-zinc-200">{fmt(run.startedAt)}</p>
          </div>
          <div>
            <span className="text-xs font-medium text-neutral-400 dark:text-zinc-500">{t('runs.finished')}</span>
            <p className="text-neutral-800 dark:text-zinc-200">{fmt(run.finishedAt)}</p>
          </div>
          {run.model && (
            <div>
              <span className="text-xs font-medium text-neutral-400 dark:text-zinc-500">{t('runs.model')}</span>
              <p className="font-mono text-neutral-800 dark:text-zinc-200">{run.model}</p>
              {run.apiModel && (
                <p className="mt-0.5 font-mono text-xs text-neutral-500 dark:text-zinc-400">{run.apiModel}</p>
              )}
              {run.apiBaseUrl && (
                <p className="mt-0.5 truncate font-mono text-[11px] text-neutral-400 dark:text-zinc-500">{run.apiBaseUrl}</p>
              )}
            </div>
          )}
          {run.sessionId && (
            <div>
              <span className="text-xs font-medium text-neutral-400 dark:text-zinc-500">{t('runs.sessionLabel')}</span>
              <div className="mt-0.5 flex items-center gap-2">
                <p className="font-mono text-xs text-neutral-700 dark:text-zinc-300" title={run.sessionId}>{run.sessionId.length > 20 ? run.sessionId.slice(0, 20) + '…' : run.sessionId}</p>
                <Link
                  to={`/projects/${encodeURIComponent(run.project)}/members/${encodeURIComponent(run.agent)}/chat?sessionId=${encodeURIComponent(run.sessionId)}`}
                  className="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[11px] font-medium text-sky-700 hover:bg-sky-50 dark:text-sky-400 dark:hover:bg-sky-900/20"
                >
                  <MessageSquareText className="size-3" strokeWidth={2} />
                  {t('agentChat.openChat')}
                </Link>
              </div>
            </div>
          )}
          <div>
            <span className="text-xs font-medium text-neutral-400 dark:text-zinc-500">{t('runs.colTok')}</span>
            <p className="tabular-nums text-neutral-800 dark:text-zinc-200">
              {fmtNum((run.inputTokens ?? 0) + (run.outputTokens ?? 0) + (run.cacheReadTokens ?? 0))}
              {run.costUSD != null && <span className="ml-1.5 text-neutral-400 dark:text-zinc-500">(${run.costUSD.toFixed(4)})</span>}
            </p>
          </div>
          {run.taskTitle && (
            <div className="col-span-2">
              <span className="text-xs font-medium text-neutral-400 dark:text-zinc-500">{t('runs.taskLabel')}</span>
              <p className="text-neutral-800 dark:text-zinc-200">{run.taskTitle}</p>
            </div>
          )}
          {run.command && (
            <div className="col-span-full">
              <span className="text-xs font-medium text-neutral-400 dark:text-zinc-500">{t('runs.command')}</span>
              <p className="truncate font-mono text-xs text-neutral-600 dark:text-zinc-400">{run.command}</p>
            </div>
          )}
          {run.errorMsg && (() => {
            const hintIdx = run.errorMsg.indexOf('[hint]')
            const rawError = hintIdx >= 0 ? run.errorMsg.slice(0, hintIdx).trimEnd() : run.errorMsg
            const hintText = hintIdx >= 0 ? run.errorMsg.slice(hintIdx + 6).trim() : ''
            return (
              <div className="col-span-full space-y-2">
                <div>
                  <span className="text-xs font-medium text-red-500">{t('runs.errorLabel')}</span>
                  <pre className="mt-1 max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md border border-red-200/60 bg-red-50/50 p-2.5 font-mono text-xs text-red-700 dark:border-red-800/40 dark:bg-red-950/20 dark:text-red-400">{rawError}</pre>
                </div>
                {hintText && (
                  <div className="rounded-md border border-blue-200/60 bg-blue-50/50 p-3 dark:border-blue-800/40 dark:bg-blue-950/20">
                    <span className="text-xs font-semibold text-blue-600 dark:text-blue-400">💡 {t('runs.setupHint')}</span>
                    <pre className="mt-1.5 whitespace-pre-wrap break-words text-xs leading-relaxed text-blue-700 dark:text-blue-300">{hintText}</pre>
                  </div>
                )}
              </div>
            )
          })()}
        </div>

        {/* Log content */}
        <div className="flex-1 overflow-y-auto">
          <div className="flex items-center gap-1.5 px-5 pt-3 pb-2">
            <FileText className="size-3.5 text-neutral-400 dark:text-zinc-500" strokeWidth={1.8} />
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
              {t('runs.logTitle')}
            </span>
          </div>
          <div className="px-5 pb-4">
            {hasLog && logState.status === 'loading' && (
              <div className="flex items-center gap-2 py-6 justify-center">
                <div className="size-4 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
                <span className="text-sm text-neutral-500">{t('api.loading')}</span>
              </div>
            )}
            {hasLog && logState.status === 'error' && (
              <p className="py-4 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('runs.logNotFound')}</p>
            )}
            {hasLog && logState.status === 'ok' && (
              <>
                <ConversationLog content={logState.data.content} />
                {logState.data.truncated && (
                  <p className="mt-2 text-xs text-amber-600 dark:text-amber-400">{t('runs.logTruncated')}</p>
                )}
              </>
            )}
            {!hasLog && (
              <p className="py-4 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('runs.noLog')}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
