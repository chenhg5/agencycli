import { useCallback, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Archive, FileText, ListTodo, Pencil, Trash2, X } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { CreateTaskDialog } from '../../components/project/CreateTaskDialog'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { apiPost, apiPut } from '../../lib/api'
import { cn } from '../../lib/cn'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'

type TaskRow = {
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
  createdAt: string
  updatedAt: string
}

type AgentRow = { name: string }

type RunRow = {
  project: string; agent: string; kind: string; status: string
  startedAt: string; finishedAt: string; model?: string
  taskId?: string; taskTitle?: string; logPath?: string
  inputTokens?: number; outputTokens?: number; cacheReadTokens?: number
  costUSD?: number; errorMsg?: string; command?: string
}

type LogData = { content: string; truncated: boolean }

type Filters = { status: string; agent: string; priority: string; scope: string }
const defaultFilters: Filters = { status: '', agent: '', priority: '', scope: 'all' }

function buildQuery(f: Filters) {
  const p = new URLSearchParams()
  if (f.status) p.set('status', f.status)
  if (f.agent) p.set('agent', f.agent)
  if (f.priority) p.set('priority', f.priority)
  p.set('scope', f.scope || 'all')
  return `?${p.toString()}`
}

const STATUS_KEYS = ['pending', 'in_progress', 'awaiting_confirmation', 'blocked', 'done_success', 'done_failed', 'cancelled'] as const

const statusColor: Record<string, string> = {
  pending: 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300',
  in_progress: 'bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-300',
  awaiting_confirmation: 'bg-violet-100 text-violet-800 dark:bg-violet-900/40 dark:text-violet-300',
  blocked: 'bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-300',
  done_success: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300',
  done_failed: 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300',
  cancelled: 'bg-neutral-100 text-neutral-600 dark:bg-zinc-800 dark:text-zinc-500',
}

const priorityLabel: Record<number, { text: string; cls: string }> = {
  0: { text: 'P0', cls: 'text-red-600 dark:text-red-400' },
  1: { text: 'P1', cls: 'text-amber-600 dark:text-amber-400' },
  2: { text: 'P2', cls: 'text-sky-600 dark:text-sky-400' },
  3: { text: 'P3', cls: 'text-neutral-400 dark:text-zinc-600' },
}

const selectCls =
  'h-8 rounded-md border border-neutral-200/80 bg-white px-2.5 pr-7 text-[13px] text-neutral-700 outline-none transition-colors hover:border-neutral-300 focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-600'
const fieldCls =
  'w-full rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-sm text-neutral-900 outline-none transition-colors focus:border-sky-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100'

function isTerminal(s: string) {
  return s === 'done_success' || s === 'done_failed' || s === 'cancelled'
}

export default function ProjectTasksPage() {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const { projectId } = useParams<{ projectId: string }>()
  const base =
    projectId != null && projectId !== ''
      ? `/api/v1/projects/${encodeURIComponent(projectId)}`
      : null

  const [filters, setFilters] = useState<Filters>({ ...defaultFilters })
  const [reloadKey, setReloadKey] = useState(0)
  const [checked, setChecked] = useState<Set<string>>(new Set())
  const [batchBusy, setBatchBusy] = useState(false)
  const [editRow, setEditRow] = useState<TaskRow | null>(null)
  const [detailRow, setDetailRow] = useState<TaskRow | null>(null)

  const queryString = useMemo(() => buildQuery(filters), [filters])
  const tasksPath = base != null ? `${base}/tasks${queryString}` : null
  const agentsPath = base != null ? `${base}/agents` : null

  const state = useApiJson<TaskRow[]>(tasksPath, reloadKey)
  const agentsState = useApiJson<AgentRow[]>(agentsPath, reloadKey)
  const tasks = state.status === 'ok' ? (state.data ?? []) : []
  const agents = agentsState.status === 'ok' ? (agentsState.data ?? []) : []

  function setFilter<K extends keyof Filters>(key: K, val: Filters[K]) {
    setFilters((prev) => ({ ...prev, [key]: val }))
    setChecked(new Set())
  }
  function resetFilters() {
    setFilters({ ...defaultFilters })
    setChecked(new Set())
  }
  const hasFilters = filters.status !== '' || filters.agent !== '' || filters.priority !== '' || filters.scope !== 'all'

  const allChecked = tasks.length > 0 && checked.size === tasks.length
  const someChecked = checked.size > 0
  function toggleAll() { setChecked(allChecked ? new Set() : new Set(tasks.map((r) => r.id))) }
  function toggleOne(id: string) {
    setChecked((prev) => { const n = new Set(prev); if (n.has(id)) n.delete(id); else n.add(id); return n })
  }
  function getCheckedRows() { return tasks.filter((r) => checked.has(r.id)) }

  const reload = useCallback(() => { setReloadKey((k) => k + 1); setChecked(new Set()) }, [])

  async function batchCancel() {
    const rows = getCheckedRows().filter((r) => !isTerminal(r.status))
    if (rows.length === 0) return
    if (!window.confirm(t('tasks.confirmBatchCancel', { count: String(rows.length) }))) return
    setBatchBusy(true)
    try { for (const r of rows) await apiPost('/api/v1/tasks/cancel', { project: r.project, agent: r.agent, id: r.id }); reload() }
    finally { setBatchBusy(false) }
  }
  async function batchArchive() {
    setBatchBusy(true)
    try { for (const r of getCheckedRows()) await apiPost('/api/v1/tasks/archive', { project: r.project, agent: r.agent, id: r.id }); reload() }
    finally { setBatchBusy(false) }
  }
  async function batchDelete() {
    const rows = getCheckedRows()
    if (!window.confirm(t('tasks.confirmBatchDelete', { count: String(rows.length) }))) return
    setBatchBusy(true)
    try { for (const r of rows) await apiPost('/api/v1/tasks/delete', { project: r.project, agent: r.agent, id: r.id }); reload() }
    finally { setBatchBusy(false) }
  }
  async function quickCancel(row: TaskRow, e: React.MouseEvent) {
    e.stopPropagation()
    if (!window.confirm(t('tasks.confirmCancel'))) return
    await apiPost('/api/v1/tasks/cancel', { project: row.project, agent: row.agent, id: row.id }); reload()
  }
  async function quickArchive(row: TaskRow, e: React.MouseEvent) {
    e.stopPropagation()
    await apiPost('/api/v1/tasks/archive', { project: row.project, agent: row.agent, id: row.id }); reload()
  }
  async function quickDelete(row: TaskRow, e: React.MouseEvent) {
    e.stopPropagation()
    if (!window.confirm(t('tasks.confirmDelete'))) return
    await apiPost('/api/v1/tasks/delete', { project: row.project, agent: row.agent, id: row.id }); reload()
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-5 pb-3">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('projectNav.tasks')}</h1>
            <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('tasks.subtitle')}</p>
          </div>
          {projectId != null && projectId !== '' && (
            <CreateTaskDialog projectId={projectId} agents={agents} onCreated={reload} />
          )}
        </div>
      </div>

      {/* Filters */}
      <div className="shrink-0 border-b border-neutral-200/80 px-6 pb-3 dark:border-zinc-800/50">
        <div className="flex flex-wrap items-center gap-2">
          <select value={filters.scope} onChange={(e) => setFilter('scope', e.target.value)} className={selectCls}>
            <option value="all">{t('tasks.scopeAll')}</option>
            <option value="active">{t('tasks.scopeActive')}</option>
            <option value="archived">{t('tasks.scopeArchived')}</option>
          </select>
          <select value={filters.status} onChange={(e) => setFilter('status', e.target.value)} className={selectCls}>
            <option value="">{t('tasks.filterStatus')}: {t('messages.readAll')}</option>
            {STATUS_KEYS.map((s) => <option key={s} value={s}>{t(`tasks.status.${s}`)}</option>)}
          </select>
          <select value={filters.agent} onChange={(e) => setFilter('agent', e.target.value)} className={cn(selectCls, 'font-mono')}>
            <option value="">{t('tasks.filterAgent')}: {t('messages.readAll')}</option>
            <option value="human">human</option>
            {agents.map((a) => <option key={a.name} value={a.name}>{a.name}</option>)}
          </select>
          <select value={filters.priority} onChange={(e) => setFilter('priority', e.target.value)} className={selectCls}>
            <option value="">{t('tasks.filterPriority')}: {t('messages.readAll')}</option>
            {[0, 1, 2, 3].map((p) => <option key={p} value={String(p)}>P{p} — {t(`forms.priorityLabel.${p}`)}</option>)}
          </select>
          {hasFilters && (
            <button type="button" onClick={resetFilters} className="flex items-center gap-1 rounded-md px-2 py-1 text-[13px] text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
              <X className="size-3" strokeWidth={2} />
              {t('messages.resetFilters')}
            </button>
          )}
        </div>
      </div>

      {/* Batch bar */}
      {someChecked && (
        <div className="shrink-0 flex items-center gap-3 border-b border-sky-200 bg-sky-50/60 px-6 py-2 animate-slide-down dark:border-sky-900/40 dark:bg-sky-950/30">
          <span className="text-[13px] font-medium text-sky-800 dark:text-sky-300">{t('messages.selected', { count: String(checked.size) })}</span>
          <div className="flex items-center gap-1.5">
            <button type="button" disabled={batchBusy} onClick={() => void batchCancel()} className="whitespace-nowrap rounded-md border border-amber-200 bg-white px-2.5 py-1 text-[12px] font-medium text-amber-700 transition-colors hover:bg-amber-50 disabled:opacity-40 dark:border-amber-800 dark:bg-amber-900/40 dark:text-amber-300">{t('tasks.batchCancel')}</button>
            <button type="button" disabled={batchBusy} onClick={() => void batchArchive()} className="whitespace-nowrap rounded-md border border-neutral-200 bg-white px-2.5 py-1 text-[12px] font-medium text-neutral-600 transition-colors hover:bg-neutral-50 disabled:opacity-40 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400">{t('tasks.batchArchive')}</button>
            <button type="button" disabled={batchBusy} onClick={() => void batchDelete()} className="whitespace-nowrap rounded-md border border-red-200 bg-white px-2.5 py-1 text-[12px] font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-40 dark:border-red-800 dark:bg-red-900/40 dark:text-red-400">{t('tasks.batchDelete')}</button>
          </div>
          <button type="button" onClick={() => setChecked(new Set())} className="ml-auto text-[12px] text-sky-600 hover:text-sky-800 dark:text-sky-400">{t('forms.cancel')}</button>
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-6 py-3">
        {state.status === 'loading' && (
          <div className="flex items-center gap-2 py-16 justify-center">
            <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {state.status === 'error' && (
          <PlaceholderCard title={t('api.loadError')}>
            <p>{state.error.message}</p>
            <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-600">{t('api.hintServe')}</p>
          </PlaceholderCard>
        )}
        {state.status === 'ok' && tasks.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <ListTodo className="size-6 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-base font-medium text-neutral-700 dark:text-zinc-300">{t('tasks.emptyTitle')}</p>
            <p className="mt-1 text-sm text-neutral-400 dark:text-zinc-600">{t('api.noTasks')}</p>
          </div>
        )}

        {state.status === 'ok' && tasks.length > 0 && (
          <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-800/60">
            <table className="min-w-[900px] w-full">
              <thead>
                <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-800/60 dark:bg-zinc-900/40">
                  <th className="w-10 px-3 py-2.5 text-center">
                    <input type="checkbox" checked={allChecked} ref={(el) => { if (el) el.indeterminate = someChecked && !allChecked }} onChange={toggleAll} className="size-3.5 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">{t('api.taskColTitle')}</th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">{t('tasks.colAssignee')}</th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">{t('api.taskColStatus')}</th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">{t('forms.priority')}</th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">{t('api.taskColUpdated')}</th>
                  <th className="sticky right-0 bg-neutral-50/95 px-4 py-2.5 text-right text-xs font-semibold uppercase tracking-wider text-neutral-400 backdrop-blur-sm dark:bg-zinc-900/95 dark:text-zinc-600">{t('messages.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
                {tasks.map((row) => {
                  const prio = priorityLabel[row.priority] ?? priorityLabel[2]
                  const sCls = statusColor[row.status] ?? statusColor.pending
                  const terminal = isTerminal(row.status)
                  const isChecked = checked.has(row.id)
                  return (
                    <tr
                      key={row.id}
                      onClick={() => setDetailRow(row)}
                      className={cn(
                        'group cursor-pointer transition-colors duration-100',
                        isChecked ? 'bg-sky-50/60 dark:bg-sky-900/[0.10]' : 'bg-white hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30',
                      )}
                    >
                      <td className="w-10 px-3 py-3 text-center align-middle" onClick={(e) => e.stopPropagation()}>
                        <input type="checkbox" checked={isChecked} onChange={() => toggleOne(row.id)} className="size-3.5 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
                      </td>
                      <td className="px-4 py-3 align-middle">
                        <div className="flex items-center gap-2">
                          <span className={cn('text-[11px] font-bold', prio.cls)}>{prio.text}</span>
                          <span className="text-[13px] font-medium text-neutral-900 dark:text-zinc-100">{row.title}</span>
                          {row.type && <span className="rounded border border-neutral-200 bg-neutral-50 px-1.5 py-0.5 text-[10px] font-medium text-neutral-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-500">{t(`forms.taskType.${row.type}`, { defaultValue: row.type })}</span>}
                          {row.archived && <Archive className="size-3.5 text-neutral-400 dark:text-zinc-600" strokeWidth={1.5} />}
                        </div>
                        <span className="mt-0.5 block font-mono text-[11px] text-neutral-400 dark:text-zinc-600">{row.id}</span>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 align-middle font-mono text-[13px] text-neutral-700 dark:text-zinc-400">
                        {row.assignee === 'human' ? <span className="rounded bg-violet-50 px-1.5 py-0.5 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">human</span> : row.agent}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 align-middle">
                        <span className={cn('inline-block rounded-full px-2.5 py-0.5 text-[11px] font-semibold', sCls)}>{t(`tasks.status.${row.status}`, { defaultValue: row.status })}</span>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 align-middle">
                        <span className={cn('text-[12px] font-bold', prio.cls)}>{prio.text}</span>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 align-middle text-[13px] text-neutral-500 dark:text-zinc-500">{fmt(row.updatedAt)}</td>
                      <td className="sticky right-0 bg-white/95 px-4 py-3 align-middle backdrop-blur-sm group-hover:bg-neutral-50/95 dark:bg-zinc-900/95 dark:group-hover:bg-zinc-800/95" onClick={(e) => e.stopPropagation()}>
                        <div className="flex items-center justify-end gap-1 whitespace-nowrap opacity-0 transition-opacity duration-100 group-hover:opacity-100">
                          <button type="button" onClick={(e) => { e.stopPropagation(); setEditRow(row) }} className="rounded p-1 text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300" title={t('tasks.edit')}>
                            <Pencil className="size-3.5" strokeWidth={1.8} />
                          </button>
                          {!terminal && !row.archived && (
                            <button type="button" onClick={(e) => void quickCancel(row, e)} className="rounded p-1 text-amber-600 transition-colors hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/30" title={t('tasks.cancel')}>
                              <X className="size-3.5" strokeWidth={1.8} />
                            </button>
                          )}
                          {!row.archived && (
                            <button type="button" onClick={(e) => void quickArchive(row, e)} className="rounded p-1 text-neutral-500 transition-colors hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800" title={t('tasks.archive')}>
                              <Archive className="size-3.5" strokeWidth={1.8} />
                            </button>
                          )}
                          <button type="button" onClick={(e) => void quickDelete(row, e)} className="rounded p-1 text-red-500 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30" title={t('tasks.delete')}>
                            <Trash2 className="size-3.5" strokeWidth={1.8} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {editRow && <EditTaskModal task={editRow} onClose={() => setEditRow(null)} onSaved={reload} />}
      {detailRow && <TaskDetailModal task={detailRow} projectId={projectId!} onClose={() => setDetailRow(null)} onEdit={(r) => { setDetailRow(null); setEditRow(r) }} />}
    </div>
  )
}

/* ─── Edit modal ─── */

function EditTaskModal({ task, onClose, onSaved }: { task: TaskRow; onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()
  const [status, setStatus] = useState(task.status)
  const [priority, setPriority] = useState(task.priority)
  const [taskType, setTaskType] = useState(task.type ?? '')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const changed = status !== task.status || priority !== task.priority || taskType !== (task.type ?? '')

  async function onSave() {
    setErr(null)
    setBusy(true)
    try {
      const body: Record<string, unknown> = { project: task.project, agent: task.agent, id: task.id }
      if (status !== task.status) body.status = status
      if (priority !== task.priority) body.priority = priority
      if (taskType !== (task.type ?? '')) body.type = taskType
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

/* ─── Detail modal ─── */

function TaskDetailModal({ task, projectId, onClose, onEdit }: { task: TaskRow; projectId: string; onClose: () => void; onEdit: (r: TaskRow) => void }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()

  const prio = priorityLabel[task.priority] ?? priorityLabel[2]
  const sCls = statusColor[task.status] ?? statusColor.pending

  const runsQuery = `/api/v1/telemetry/runs?allTime=1&project=${encodeURIComponent(projectId)}`
  const runsState = useApiJson<{ runs: RunRow[] }>(runsQuery, 0)
  const matchingRun = useMemo(() => {
    if (runsState.status !== 'ok' || !runsState.data?.runs) return null
    return runsState.data.runs.find((r) => r.taskId === task.id) ?? null
  }, [runsState, task.id])

  const hasLog = Boolean(matchingRun?.logPath)
  const logQuery = hasLog ? `/api/v1/telemetry/log?path=${encodeURIComponent(matchingRun!.logPath!)}` : null
  const logState = useApiJson<LogData>(logQuery, 0)

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[6vh]">
      <div className="absolute inset-0 bg-black/30 backdrop-blur-[2px] animate-fade-in dark:bg-black/50" onClick={onClose} />
      <div className="relative w-full max-w-3xl max-h-[85vh] flex flex-col overflow-hidden rounded-xl border border-neutral-200/80 bg-white shadow-2xl animate-scale-in dark:border-zinc-800/80 dark:bg-zinc-900">
        {/* Header */}
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

        {/* Info */}
        <div className="shrink-0 grid grid-cols-2 gap-x-6 gap-y-2.5 border-b border-neutral-100 px-5 py-3 text-sm dark:border-zinc-800/40 sm:grid-cols-3">
          <InfoCell label="ID"><span className="font-mono text-xs">{task.id}</span></InfoCell>
          <InfoCell label={t('tasks.colAssignee')}>{task.assignee === 'human' ? <span className="rounded bg-violet-50 px-1.5 py-0.5 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">human</span> : <span className="font-mono">{task.agent}</span>}</InfoCell>
          <InfoCell label={t('forms.type')}>{task.type ? t(`forms.taskType.${task.type}`, { defaultValue: task.type }) : '—'}</InfoCell>
          <InfoCell label={t('api.taskColUpdated')}>{fmt(task.updatedAt)}</InfoCell>
          <InfoCell label={t('tasks.colArchived')}>{task.archived ? t('tasks.yes') : t('tasks.no')}</InfoCell>
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

        {/* Prompt */}
        {task.prompt && (
          <div className="shrink-0 border-b border-neutral-100 px-5 py-3 dark:border-zinc-800/40">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">{t('forms.prompt')}</span>
            <div className="mt-1.5 max-h-32 overflow-y-auto rounded-lg bg-neutral-50 p-3 text-sm text-neutral-700 dark:bg-zinc-800/50 dark:text-zinc-300">
              <ReactMarkdown remarkPlugins={[remarkGfm]} className="prose prose-sm max-w-none dark:prose-invert">{task.prompt}</ReactMarkdown>
            </div>
          </div>
        )}

        {/* Execution log */}
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

function InfoCell({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <span className="text-xs font-medium text-neutral-400 dark:text-zinc-600">{label}</span>
      <p className="text-neutral-800 dark:text-zinc-200">{children}</p>
    </div>
  )
}

/* ─── Conversation log (reused from runs page) ─── */

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
