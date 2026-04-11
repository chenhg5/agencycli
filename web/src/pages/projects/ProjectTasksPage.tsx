import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Archive, Kanban, LayoutList, ListTodo, Pencil, RefreshCw, Trash2, X } from 'lucide-react'
import { CreateTaskDialog } from '../../components/project/CreateTaskDialog'
import { TaskKanban } from '../../components/task/TaskKanban'
import { Pagination } from '../../components/ui/Pagination'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { apiPost, apiPut } from '../../lib/api'
import { cn } from '../../lib/cn'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'
import {
  EditTaskModal,
  TaskDetailModal,
  type TaskRow,
  STATUS_KEYS,
  statusColor,
  priorityLabel,
  isTerminal,
} from '../../components/task/TaskModals'

type AgentRow = { name: string }

type ViewMode = 'table' | 'board'
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

const selectCls =
  'h-8 rounded-md border border-neutral-200/80 bg-white px-2.5 pr-7 text-[13px] text-neutral-700 outline-none transition-colors hover:border-neutral-300 focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-600 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-900 [&>option]:dark:text-zinc-300'

export default function ProjectTasksPage() {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const { projectId } = useParams<{ projectId: string }>()
  const base =
    projectId != null && projectId !== ''
      ? `/api/v1/projects/${encodeURIComponent(projectId)}`
      : null

  const [viewMode, setViewMode] = useState<ViewMode>(() => (localStorage.getItem('agencycli_tasks_view') as ViewMode) || 'table')
  const [filters, setFilters] = useState<Filters>({ ...defaultFilters })
  const [reloadKey, setReloadKey] = useState(0)
  const [checked, setChecked] = useState<Set<string>>(new Set())
  const [batchBusy, setBatchBusy] = useState(false)
  const [editRow, setEditRow] = useState<TaskRow | null>(null)
  const [detailRow, setDetailRow] = useState<TaskRow | null>(null)
  const [taskPage, setTaskPage] = useState(1)
  const tasksPerPage = 20

  const queryString = useMemo(() => buildQuery(filters), [filters])
  const tasksPath = base != null ? `${base}/tasks${queryString}` : null
  const agentsPath = base != null ? `${base}/agents` : null

  const state = useApiJson<TaskRow[]>(tasksPath, reloadKey)
  const agentsState = useApiJson<AgentRow[]>(agentsPath, reloadKey)
  const tasks = state.status === 'ok' ? (state.data ?? []) : []
  const agents = agentsState.status === 'ok' ? (agentsState.data ?? []) : []

  const totalTaskPages = Math.ceil(tasks.length / tasksPerPage)
  const pagedTasks = useMemo(() => {
    const start = (taskPage - 1) * tasksPerPage
    return tasks.slice(start, start + tasksPerPage)
  }, [tasks, taskPage])

  useEffect(() => {
    setTaskPage(1)
  }, [filters])

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
          <div className="flex items-center gap-2">
            <div className="flex items-center rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
              {([['table', LayoutList], ['board', Kanban]] as const).map(([mode, Icon]) => (
                <button
                  key={mode}
                  type="button"
                  onClick={() => { setViewMode(mode); localStorage.setItem('agencycli_tasks_view', mode) }}
                  className={cn(
                    'p-1.5 transition-colors',
                    viewMode === mode
                      ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-400'
                      : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400',
                    mode === 'table' ? 'rounded-l-md' : 'rounded-r-md',
                  )}
                  title={t(`tasks.view_${mode}`)}
                >
                  <Icon className="size-4" strokeWidth={1.8} />
                </button>
              ))}
            </div>
            {projectId != null && projectId !== '' && (
              <CreateTaskDialog projectId={projectId} agents={agents} onCreated={reload} />
            )}
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="shrink-0 border-b border-neutral-200/80 px-6 pb-3 dark:border-zinc-700/50">
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
            <button type="button" onClick={resetFilters} className="flex items-center gap-1 rounded-md px-2 py-1 text-[13px] text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
              <X className="size-3" strokeWidth={2} />
              {t('messages.resetFilters')}
            </button>
          )}
          <button type="button" onClick={reload} className="flex items-center gap-1 rounded-md px-2 py-1 text-[13px] text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
            <RefreshCw className="size-3" strokeWidth={2} />
            {t('api.refresh')}
          </button>
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
            <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('api.hintServe')}</p>
          </PlaceholderCard>
        )}
        {state.status === 'ok' && tasks.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <ListTodo className="size-6 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-base font-medium text-neutral-700 dark:text-zinc-300">{t('tasks.emptyTitle')}</p>
            <p className="mt-1 text-sm text-neutral-400 dark:text-zinc-500">{t('api.noTasks')}</p>
          </div>
        )}

        {state.status === 'ok' && tasks.length > 0 && viewMode === 'board' && (
          <TaskKanban tasks={tasks} onTaskClick={setDetailRow} onStatusChange={async (task, status) => {
            await apiPut('/api/v1/tasks/update', { project: task.project, agent: task.agent, id: task.id, status })
            reload()
          }} />
        )}

        {state.status === 'ok' && tasks.length > 0 && viewMode === 'table' && (
          <>
            <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
            <table className="min-w-[900px] w-full">
              <thead>
                <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-700/60 dark:bg-zinc-900/40">
                  <th className="w-10 px-3 py-2.5 text-center">
                    <input type="checkbox" checked={allChecked} ref={(el) => { if (el) el.indeterminate = someChecked && !allChecked }} onChange={toggleAll} className="size-3.5 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
                  </th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('api.taskColTitle')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('tasks.colAssignee')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('api.taskColStatus')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('forms.priority')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('api.taskColUpdated')}</th>
                  <th className="sticky right-0 bg-neutral-50/95 px-4 py-2.5 text-right text-xs font-semibold uppercase tracking-wider text-neutral-400 backdrop-blur-sm dark:bg-zinc-900/95 dark:text-zinc-500">{t('messages.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
                {pagedTasks.map((row) => {
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
                          {row.labels?.map(l => (
                            <span key={l} className="rounded-full bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400">{l}</span>
                          ))}
                          {row.dueDate && <span className="text-[10px] text-neutral-400 dark:text-zinc-500">{row.dueDate}</span>}
                          {row.archived && <Archive className="size-3.5 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />}
                        </div>
                        <span className="mt-0.5 block font-mono text-[11px] text-neutral-400 dark:text-zinc-500">{row.id}</span>
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
            <Pagination page={taskPage} totalPages={totalTaskPages} onPageChange={setTaskPage} />
          </>
        )}
      </div>

      {editRow && <EditTaskModal task={editRow} onClose={() => setEditRow(null)} onSaved={reload} />}
      {detailRow && <TaskDetailModal task={detailRow} onClose={() => setDetailRow(null)} onEdit={(r) => { setDetailRow(null); setEditRow(r) }} />}
    </div>
  )
}

