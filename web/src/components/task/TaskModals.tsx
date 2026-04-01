import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { FileText, Pencil, X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { apiPut } from '../../lib/api'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'

export type TaskRow = {
  id: string
  project: string
  agent: string
  title: string
  type?: string
  priority: number
  status: string
  archived: boolean
  assignee?: string
  prompt?: string
  summary?: string
  createdBy?: string
  createdAt: string
  updatedAt: string
}

type RunRow = {
  project: string; agent: string; kind: string; status: string
  startedAt: string; finishedAt: string; model?: string
  taskId?: string; taskTitle?: string; logPath?: string
  inputTokens?: number; outputTokens?: number; cacheReadTokens?: number
  costUSD?: number; errorMsg?: string; command?: string
}

type LogData = { content: string; truncated: boolean }

export const STATUS_KEYS = ['pending', 'in_progress', 'awaiting_confirmation', 'blocked', 'done_success', 'done_failed', 'cancelled'] as const

export const statusColor: Record<string, string> = {
  pending: 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300',
  in_progress: 'bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-300',
  awaiting_confirmation: 'bg-violet-100 text-violet-800 dark:bg-violet-900/40 dark:text-violet-300',
  blocked: 'bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-300',
  done_success: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300',
  done_failed: 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300',
  cancelled: 'bg-neutral-100 text-neutral-600 dark:bg-zinc-800 dark:text-zinc-500',
}

export const priorityLabel: Record<number, { text: string; cls: string }> = {
  0: { text: 'P0', cls: 'text-red-600 dark:text-red-400' },
  1: { text: 'P1', cls: 'text-amber-600 dark:text-amber-400' },
  2: { text: 'P2', cls: 'text-sky-600 dark:text-sky-400' },
  3: { text: 'P3', cls: 'text-neutral-400 dark:text-zinc-600' },
}

export function isTerminal(s: string) {
  return s === 'done_success' || s === 'done_failed' || s === 'cancelled'
}

const fieldCls =
  'w-full rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-sm text-neutral-900 outline-none transition-colors focus:border-sky-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100'

/* ── Edit modal ─── */

export function EditTaskModal({ task, onClose, onSaved }: { task: TaskRow; onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()
  const [status, setStatus] = useState(task.status)
  const [priority, setPriority] = useState(task.priority)
  const [taskType, setTaskType] = useState(task.type ?? '')
  const [summary, setSummary] = useState(task.summary ?? '')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const showSummary = isTerminal(status)
  const changed = status !== task.status || priority !== task.priority || taskType !== (task.type ?? '') || summary !== (task.summary ?? '')

  async function onSave() {
    setErr(null)
    setBusy(true)
    try {
      const body: Record<string, unknown> = { project: task.project, agent: task.agent, id: task.id }
      if (status !== task.status) body.status = status
      if (priority !== task.priority) body.priority = priority
      if (taskType !== (task.type ?? '')) body.type = taskType
      if (summary !== (task.summary ?? '')) body.summary = summary
      await apiPut('/api/v1/tasks/update', body)
      onSaved()
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" onClick={() => !busy && onClose()}>
      <div className="w-full max-w-md rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
          <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{t('tasks.edit')}</h2>
          <button type="button" onClick={onClose} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 dark:text-zinc-600 dark:hover:bg-zinc-800"><X className="size-4" /></button>
        </div>
        <div className="space-y-3 px-5 py-4">
          <div className="text-sm font-medium text-neutral-700 dark:text-zinc-300">{task.title}</div>
          <div className="font-mono text-xs text-neutral-400 dark:text-zinc-600">{task.id}</div>
          <label className="block text-sm">
            <span className="text-neutral-600 dark:text-zinc-400">{t('tasks.filterStatus')}</span>
            <select value={status} onChange={(e) => setStatus(e.target.value)} className={cn(fieldCls, 'mt-1')}>
              {STATUS_KEYS.map((s) => <option key={s} value={s}>{t(`tasks.status.${s}`)}</option>)}
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-neutral-600 dark:text-zinc-400">{t('forms.priority')}</span>
            <select value={priority} onChange={(e) => setPriority(Number(e.target.value))} className={cn(fieldCls, 'mt-1')}>
              {[0, 1, 2, 3].map((p) => <option key={p} value={p}>P{p} — {t(`forms.priorityLabel.${p}`)}</option>)}
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-neutral-600 dark:text-zinc-400">{t('forms.type')}</span>
            <select value={taskType} onChange={(e) => setTaskType(e.target.value)} className={cn(fieldCls, 'mt-1')}>
              {['chore', 'feature', 'bug', 'review', 'triage', 'test', 'research'].map((ty) => <option key={ty} value={ty}>{t(`forms.taskType.${ty}`, { defaultValue: ty })}</option>)}
            </select>
          </label>
          {showSummary && (
            <label className="block text-sm">
              <span className="text-neutral-600 dark:text-zinc-400">{t('tasks.summary')}</span>
              <textarea
                value={summary}
                onChange={(e) => setSummary(e.target.value)}
                rows={6}
                placeholder={t('tasks.summaryPlaceholder')}
                className={cn(fieldCls, 'mt-1')}
              />
              {task.createdBy && (
                <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-600">
                  {t('tasks.willNotifyCreator', { creator: task.createdBy })}
                </p>
              )}
            </label>
          )}
          {err && <p className="text-sm text-red-600 dark:text-red-400">{err}</p>}
          <div className="flex justify-end gap-2 pt-1">
            <button type="button" onClick={onClose} disabled={busy} className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600">{t('forms.cancel')}</button>
            <button type="button" onClick={() => void onSave()} disabled={busy || !changed} className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">{busy ? t('forms.saving') : t('forms.save')}</button>
          </div>
        </div>
      </div>
    </div>
  )
}

/* ── Detail modal ─── */

export function TaskDetailModal({ task, onClose, onEdit }: { task: TaskRow; onClose: () => void; onEdit: (r: TaskRow) => void }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()

  const prio = priorityLabel[task.priority] ?? priorityLabel[2]
  const sCls = statusColor[task.status] ?? statusColor.pending

  const runsQuery = `/api/v1/telemetry/runs?allTime=1&project=${encodeURIComponent(task.project)}`
  const runsState = useApiJson<{ runs: RunRow[] }>(runsQuery, 0)
  const matchingRun = useMemo(() => {
    if (runsState.status !== 'ok' || !runsState.data?.runs) return null
    return runsState.data.runs.find((r) => r.taskId === task.id) ?? null
  }, [runsState, task.id])

  const hasLog = Boolean(matchingRun?.logPath)
  const logQuery = hasLog ? `/api/v1/telemetry/log?path=${encodeURIComponent(matchingRun!.logPath!)}` : null
  const logState = useApiJson<LogData>(logQuery, 0)

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[3vh]">
      <div className="absolute inset-0 bg-black/30 backdrop-blur-[2px] animate-fade-in dark:bg-black/50" onClick={onClose} />
      <div className="relative w-full max-w-4xl max-h-[94vh] flex flex-col overflow-hidden rounded-xl border border-neutral-200/80 bg-white shadow-2xl animate-scale-in dark:border-zinc-800/80 dark:bg-zinc-900">
        <div className="flex items-center justify-between border-b border-neutral-200/80 px-5 py-3 dark:border-zinc-800/60">
          <div className="flex items-center gap-3 min-w-0">
            <span className={cn('shrink-0 rounded-full px-2.5 py-0.5 text-[11px] font-semibold', sCls)}>{t(`tasks.status.${task.status}`, { defaultValue: task.status })}</span>
            <span className={cn('shrink-0 text-[11px] font-bold', prio.cls)}>{prio.text}</span>
            <span className="truncate text-sm font-medium text-neutral-900 dark:text-zinc-100">{task.title}</span>
          </div>
          <div className="flex items-center gap-1 shrink-0">
            <button type="button" onClick={() => onEdit(task)} className="rounded-md p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-600 dark:hover:bg-zinc-800" title={t('tasks.edit')}>
              <Pencil className="size-4" strokeWidth={1.8} />
            </button>
            <button type="button" onClick={onClose} className="rounded-md p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-600 dark:hover:bg-zinc-800">
              <X className="size-4" strokeWidth={2} />
            </button>
          </div>
        </div>

        <div className="shrink-0 grid grid-cols-2 gap-x-6 gap-y-2.5 border-b border-neutral-100 px-5 py-3 text-sm dark:border-zinc-800/40 sm:grid-cols-3">
          <InfoCell label="ID"><span className="font-mono text-xs">{task.id}</span></InfoCell>
          <InfoCell label={t('tasks.colProject')}><span className="font-mono">{task.project}</span></InfoCell>
          <InfoCell label={t('tasks.colAssignee')}>{task.assignee === 'human' ? <span className="rounded bg-violet-50 px-1.5 py-0.5 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">human</span> : <span className="font-mono">{task.agent}</span>}</InfoCell>
          <InfoCell label={t('forms.type')}>{task.type ? t(`forms.taskType.${task.type}`, { defaultValue: task.type }) : '—'}</InfoCell>
          <InfoCell label={t('api.taskColUpdated')}>{fmt(task.updatedAt)}</InfoCell>
          {task.createdBy && <InfoCell label={t('tasks.createdBy')}><span className="font-mono">{task.createdBy}</span></InfoCell>}
          {matchingRun && (
            <>
              <InfoCell label={t('runs.model')}><span className="font-mono">{matchingRun.model ?? '—'}</span></InfoCell>
              {matchingRun.costUSD != null && (
                <InfoCell label={t('runs.colTok')}>
                  <span className="tabular-nums">{fmtNum((matchingRun.inputTokens ?? 0) + (matchingRun.outputTokens ?? 0))} tok</span>
                  <span className="ml-1 text-neutral-400">(${matchingRun.costUSD.toFixed(4)})</span>
                </InfoCell>
              )}
            </>
          )}
        </div>

        {task.prompt && (
          <div className="shrink-0 border-b border-neutral-100 px-5 py-3 dark:border-zinc-800/40">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">{t('forms.prompt')}</span>
            <div className="mt-1.5 overflow-y-auto rounded-lg bg-neutral-50 p-3 text-sm text-neutral-700 dark:bg-zinc-800/50 dark:text-zinc-300">
              <div className="prose prose-sm max-w-none dark:prose-invert"><ReactMarkdown remarkPlugins={[remarkGfm]}>{task.prompt}</ReactMarkdown></div>
            </div>
          </div>
        )}

        {task.summary && (
          <div className="shrink-0 border-b border-neutral-100 px-5 py-3 dark:border-zinc-800/40">
            <span className="text-xs font-semibold uppercase tracking-wider text-emerald-500 dark:text-emerald-400">{t('tasks.summary')}</span>
            <div className="mt-1.5 overflow-y-auto rounded-lg bg-emerald-50 p-3 text-sm text-neutral-700 dark:bg-emerald-900/20 dark:text-zinc-300">
              <div className="prose prose-sm max-w-none dark:prose-invert"><ReactMarkdown remarkPlugins={[remarkGfm]}>{task.summary}</ReactMarkdown></div>
            </div>
          </div>
        )}

        <div className="flex-1 overflow-y-auto">
          {matchingRun ? (
            <>
              <div className="flex items-center gap-1.5 px-5 pt-3 pb-2">
                <FileText className="size-3.5 text-neutral-400 dark:text-zinc-600" strokeWidth={1.8} />
                <span className="text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">{t('runs.logTitle')}</span>
              </div>
              <div className="px-5 pb-4">
                {hasLog && logState.status === 'loading' && (
                  <div className="flex items-center gap-2 py-6 justify-center">
                    <div className="size-4 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
                    <span className="text-sm text-neutral-500">{t('api.loading')}</span>
                  </div>
                )}
                {hasLog && logState.status === 'error' && <p className="py-4 text-center text-sm text-neutral-400">{t('runs.logNotFound')}</p>}
                {hasLog && logState.status === 'ok' && <ConversationLog content={logState.data.content} />}
                {!hasLog && <p className="py-4 text-center text-sm text-neutral-400 dark:text-zinc-600">{t('runs.noLog')}</p>}
              </div>
            </>
          ) : runsState.status === 'loading' ? (
            <div className="flex items-center gap-2 py-8 justify-center">
              <div className="size-4 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
              <span className="text-sm text-neutral-500">{t('tasks.loadingRuns')}</span>
            </div>
          ) : (
            <p className="py-8 text-center text-sm text-neutral-400 dark:text-zinc-600">{t('tasks.noRunRecord')}</p>
          )}
        </div>
      </div>
    </div>
  )
}

/* ── Helpers ── */

function InfoCell({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <span className="text-xs font-medium text-neutral-400 dark:text-zinc-600">{label}</span>
      <p className="text-neutral-800 dark:text-zinc-200">{children}</p>
    </div>
  )
}

interface ConvMsg { role: 'user' | 'assistant' | 'system'; text: string }

function parseConversation(raw: string): ConvMsg[] {
  const msgs: ConvMsg[] = []
  for (const line of raw.split('\n')) {
    const trimmed = line.trim()
    if (!trimmed) continue
    try {
      const obj = JSON.parse(trimmed)
      if (obj.type === 'user' && typeof obj.message?.content === 'string') {
        msgs.push({ role: 'user', text: obj.message.content })
      } else if (obj.type === 'assistant' && typeof obj.message?.content === 'string') {
        msgs.push({ role: 'assistant', text: obj.message.content })
      } else if (obj.type === 'assistant' && Array.isArray(obj.message?.content)) {
        const texts = obj.message.content.filter((b: any) => b.type === 'text').map((b: any) => b.text)
        if (texts.length > 0) msgs.push({ role: 'assistant', text: texts.join('\n\n') })
      } else if (obj.type === 'system' && typeof obj.message?.content === 'string') {
        msgs.push({ role: 'system', text: obj.message.content })
      }
    } catch { /* skip non-JSON */ }
  }
  return msgs
}

function ConversationLog({ content }: { content: string }) {
  const msgs = useMemo(() => parseConversation(content), [content])
  if (msgs.length === 0) return <p className="py-4 text-center text-sm text-neutral-400 dark:text-zinc-600">No conversation data</p>
  return (
    <div className="space-y-3">
      {msgs.map((m, i) => (
        <div key={i} className={cn('rounded-lg px-4 py-3 text-sm', m.role === 'user' ? 'bg-sky-50 dark:bg-sky-900/20' : m.role === 'system' ? 'bg-amber-50 dark:bg-amber-900/20' : 'bg-neutral-50 dark:bg-zinc-800/40')}>
          <span className={cn('mb-1 block text-[11px] font-semibold uppercase tracking-wider', m.role === 'user' ? 'text-sky-600 dark:text-sky-400' : m.role === 'system' ? 'text-amber-600 dark:text-amber-400' : 'text-neutral-400 dark:text-zinc-600')}>{m.role}</span>
          <div className="prose prose-sm max-w-none dark:prose-invert prose-p:my-1 prose-pre:my-1.5">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{m.text}</ReactMarkdown>
          </div>
        </div>
      ))}
    </div>
  )
}

function fmtNum(n: number): string {
  return n.toLocaleString()
}
