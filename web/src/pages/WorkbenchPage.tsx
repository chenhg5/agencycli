import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  CheckCircle2,
  KanbanSquare,
  LayoutGrid,
  LayoutList,
  Link as LinkIcon,
  List,
  ListTodo,
  Mail,
  MessageSquare,
  Pencil,
  Play,
  Power,
  RefreshCw,
  Reply,
  Send,
  Square,
  Star,
  Trash2,
  X,
} from 'lucide-react'
import { cn } from '../lib/cn'
import { apiFetch, apiPost, apiPut } from '../lib/api'
import { useFormatDateTime } from '../lib/format-datetime'
import { useApiJson } from '../lib/use-api'
import {
  MessageDetailModal,
  type MessageDetailModel,
} from '../components/project/MessageDetailModal'
import { CreateMessageDialog } from '../components/project/CreateMessageDialog'
import { CreateTaskDialog } from '../components/project/CreateTaskDialog'
import { RunAgentDialog } from '../components/project/RunAgentDialog'
import { Pagination } from '../components/ui/Pagination'
import { TaskKanban } from '../components/task/TaskKanban'
import { TaskTable } from '../components/task/TaskTable'
import { getQuickLinks, removeQuickLink, type QuickLink } from '../lib/quick-links'
import {
  EditTaskModal,
  TaskDetailModal,
  type TaskRow as SharedTaskRow,
  statusColor,
  priorityLabel,
  isTerminal,
  STATUS_KEYS,
} from '../components/task/TaskModals'

type ProjectAgents = { projectId: string; agents: { name: string }[] }

function useProjectsAgents() {
  const [data, setData] = useState<ProjectAgents[]>([])
  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const projects = await apiFetch<{ name: string }[]>('/api/v1/projects')
        const result: ProjectAgents[] = []
        for (const p of projects) {
          try {
            const agents = await apiFetch<{ name: string }[]>(`/api/v1/projects/${encodeURIComponent(p.name)}/agents`)
            result.push({ projectId: p.name, agents })
          } catch { result.push({ projectId: p.name, agents: [] }) }
        }
        if (!cancelled) setData(result)
      } catch { /* ignore */ }
    })()
    return () => { cancelled = true }
  }, [])
  return data
}

type Tab = 'overview' | 'quickLinks' | 'messages' | 'tasks'

type MessageRow = MessageDetailModel

type TaskRow = SharedTaskRow

type Filters = {
  direction: 'all' | 'inbox' | 'sent'
  read: 'all' | 'read' | 'unread'
  archived: 'all' | 'no' | 'yes'
  from: string
}

const defaultFilters: Filters = { direction: 'inbox', read: 'unread', archived: 'no', from: '' }

function buildMsgQuery(f: Filters) {
  const p = new URLSearchParams()
  if (f.direction !== 'all') p.set('direction', f.direction)
  if (f.read !== 'all') p.set('read', f.read)
  if (f.archived !== 'no') p.set('archived', f.archived)
  if (f.from) p.set('from', f.from)
  const qs = p.toString()
  return qs ? `?${qs}` : ''
}

function preview(body: string, max = 160) {
  const one = body.replace(/\s+/g, ' ').trim()
  return one.length <= max ? one : `${one.slice(0, max)}…`
}

const selectCls =
  'h-9 rounded-lg border border-neutral-200/80 bg-white px-3 pr-8 text-sm text-neutral-700 outline-none transition-colors hover:border-neutral-300 focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-600 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-900 [&>option]:dark:text-zinc-300'


/* ── Inline reply ─────────────────────────────────────────────────────────── */

function InlineReply({ originalFrom, mailbox, messageId, onSent }: { originalFrom: string; mailbox?: string; messageId?: string; onSent: () => void }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [body, setBody] = useState('')
  const [busy, setBusy] = useState(false)
  const [sent, setSent] = useState(false)

  const send = useCallback(async () => {
    if (!body.trim()) return
    setBusy(true)
    try {
      await apiPost('/api/v1/messages', { from: 'human', to: originalFrom, body: body.trim() })
      if (mailbox && messageId) {
        await apiPost('/api/v1/messages/mark-read', { mailbox, id: messageId }).catch(() => {})
      }
      setBody('')
      setOpen(false)
      setSent(true)
      setTimeout(() => setSent(false), 3000)
      onSent()
    } catch (e) {
      alert(String(e))
    } finally {
      setBusy(false)
    }
  }, [body, originalFrom, mailbox, messageId, onSent])

  if (sent) {
    return (
      <span className="flex items-center gap-1.5 text-sm text-emerald-600 dark:text-emerald-400">
        <CheckCircle2 className="size-4" strokeWidth={2} />
        {t('workbench.replySent')}
      </span>
    )
  }

  if (!open) {
    return (
      <button
        type="button"
        onClick={(e) => { e.stopPropagation(); setOpen(true) }}
        className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium text-sky-700 transition-colors hover:bg-sky-50 dark:text-sky-400 dark:hover:bg-sky-900/20"
      >
        <Reply className="size-4" strokeWidth={2} />
        {t('workbench.replyTo')}
      </button>
    )
  }

  return (
    <div className="w-full" onClick={(e) => e.stopPropagation()}>
      <div className="mt-2 rounded-xl border border-neutral-200 bg-neutral-50/50 p-4 dark:border-zinc-700 dark:bg-zinc-800/40">
        <div className="mb-2.5 text-sm font-medium text-neutral-500 dark:text-zinc-500">
          {t('workbench.replyTo')} <span className="font-mono text-neutral-700 dark:text-zinc-300">{originalFrom}</span>
        </div>
        <textarea
          autoFocus
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={8}
          placeholder={t('forms.body')}
          className="block w-full resize-y rounded-lg border border-neutral-200 bg-white p-3 text-sm leading-relaxed text-neutral-800 outline-none transition-colors placeholder:text-neutral-400 focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:placeholder:text-zinc-600"
        />
        <div className="mt-3 flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={() => { setOpen(false); setBody('') }}
            className="rounded-lg px-4 py-2 text-sm font-medium text-neutral-500 transition-colors hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800"
          >
            {t('forms.cancel')}
          </button>
          <button
            type="button"
            disabled={busy || !body.trim()}
            onClick={() => void send()}
            className="flex items-center gap-2 rounded-lg bg-sky-600 px-5 py-2 text-sm font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50"
          >
            <Send className="size-4" strokeWidth={2} />
            {busy ? t('forms.sending') : t('forms.send')}
          </button>
        </div>
      </div>
    </div>
  )
}

/* ── Messages panel ───────────────────────────────────────────────────────── */

function MessagesPanel({ projectsAgents, onMutated }: { projectsAgents: ProjectAgents[]; onMutated?: () => void }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const [filters, setFilters] = useState<Filters>({ ...defaultFilters })
  const [msgSort, setMsgSort] = useState<'newest' | 'oldest'>('newest')
  const [reloadKey, setReloadKey] = useState(0)
  const [selected, setSelected] = useState<MessageRow | null>(null)
  const [checked, setChecked] = useState<Set<string>>(new Set())
  const [batchBusy, setBatchBusy] = useState(false)
  const firstProject = projectsAgents[0]

  const [wbMsgPage, setWbMsgPage] = useState(1)
  const wbMsgsPerPage = 20

  const queryString = useMemo(() => buildMsgQuery(filters), [filters])
  const state = useApiJson<MessageRow[]>(`/api/v1/workbench/messages${queryString}`, reloadKey)
  const rawMessages = state.status === 'ok' ? (state.data ?? []) : []
  const messages = useMemo(() => {
    const sorted = [...rawMessages]
    if (msgSort === 'oldest') sorted.reverse()
    return sorted
  }, [rawMessages, msgSort])

  const totalWbMsgPages = Math.ceil(messages.length / wbMsgsPerPage)
  const pagedWbMessages = useMemo(() => {
    const start = (wbMsgPage - 1) * wbMsgsPerPage
    return messages.slice(start, start + wbMsgsPerPage)
  }, [messages, wbMsgPage])

  useEffect(() => { setWbMsgPage(1) }, [filters])

  function setFilter<K extends keyof Filters>(key: K, val: Filters[K]) {
    setFilters((prev) => {
      const next = { ...prev, [key]: val }
      if (key === 'direction' && val === 'sent') next.read = 'all'
      return next
    })
    setChecked(new Set())
  }
  function resetFilters() { setFilters({ ...defaultFilters }); setChecked(new Set()) }
  const hasFilters = filters.direction !== defaultFilters.direction || filters.read !== defaultFilters.read || filters.archived !== defaultFilters.archived || filters.from !== defaultFilters.from

  const allChecked = messages.length > 0 && checked.size === messages.length
  const someChecked = checked.size > 0
  function toggleAll() { setChecked(allChecked ? new Set() : new Set(messages.map((m) => m.id))) }
  function toggleOne(id: string) {
    setChecked((prev) => { const n = new Set(prev); if (n.has(id)) n.delete(id); else n.add(id); return n })
  }
  function getCheckedRows() { return messages.filter((m) => checked.has(m.id)) }

  function reloadAndNotify() { setReloadKey((k) => k + 1); onMutated?.() }

  async function batchMarkRead() {
    setBatchBusy(true)
    try {
      for (const row of getCheckedRows().filter((r) => !r.readAt))
        await apiPost('/api/v1/messages/mark-read', { mailbox: row.mailbox, id: row.id })
      setChecked(new Set()); reloadAndNotify()
    } finally { setBatchBusy(false) }
  }
  async function batchArchive() {
    setBatchBusy(true)
    try {
      for (const row of getCheckedRows().filter((r) => !r.archivedAt))
        await apiPost('/api/v1/messages/archive', { mailbox: row.mailbox, id: row.id })
      setChecked(new Set()); reloadAndNotify()
    } finally { setBatchBusy(false) }
  }
  async function batchDelete() {
    const count = checked.size
    if (!window.confirm(t('messages.confirmBatchDelete', { count: String(count) }))) return
    setBatchBusy(true)
    try {
      for (const row of getCheckedRows())
        await apiPost('/api/v1/messages/delete', { mailbox: row.mailbox, id: row.id })
      setChecked(new Set()); reloadAndNotify()
    } finally { setBatchBusy(false) }
  }

  async function quickMarkRead(row: MessageRow, e: React.MouseEvent) {
    e.stopPropagation()
    await apiPost('/api/v1/messages/mark-read', { mailbox: row.mailbox, id: row.id })
    reloadAndNotify()
  }
  async function quickArchive(row: MessageRow, e: React.MouseEvent) {
    e.stopPropagation()
    await apiPost('/api/v1/messages/archive', { mailbox: row.mailbox, id: row.id })
    reloadAndNotify()
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Filters */}
      <div className="shrink-0 px-8 py-4">
        <div className="flex flex-wrap items-center gap-3">
          <select value={filters.direction} onChange={(e) => setFilter('direction', e.target.value as Filters['direction'])} className={selectCls}>
            <option value="all">{t('messages.directionAll')}</option>
            <option value="inbox">{t('messages.directionInbox')}</option>
            <option value="sent">{t('messages.directionSent')}</option>
          </select>
          <select value={filters.read} onChange={(e) => setFilter('read', e.target.value as Filters['read'])} className={selectCls}>
            <option value="all">{t('messages.filterRead')}: {t('messages.readAll')}</option>
            <option value="unread">{t('messages.readUnread')}</option>
            <option value="read">{t('messages.readRead')}</option>
          </select>
          <select value={filters.archived} onChange={(e) => setFilter('archived', e.target.value as Filters['archived'])} className={selectCls}>
            <option value="no">{t('messages.filterArchived')}: {t('messages.archivedActive')}</option>
            <option value="yes">{t('messages.archivedOnly')}</option>
            <option value="all">{t('messages.archivedAll')}</option>
          </select>
          <select value={msgSort} onChange={(e) => { setMsgSort(e.target.value as 'newest' | 'oldest'); setWbMsgPage(1) }} className={selectCls}>
            <option value="newest">{t('workbench.sortNewest')}</option>
            <option value="oldest">{t('workbench.sortOldest')}</option>
          </select>
          {hasFilters && (
            <button type="button" onClick={resetFilters} className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
              <X className="size-3.5" strokeWidth={2} /> {t('messages.resetFilters')}
            </button>
          )}
          <button type="button" onClick={() => reloadAndNotify()} className="flex items-center gap-1 rounded-md px-2 py-1 text-[13px] text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
            <RefreshCw className="size-3" strokeWidth={2} />
            {t('api.refresh')}
          </button>
          {firstProject && (
            <div className="ml-auto">
              <CreateMessageDialog projectId={firstProject.projectId} agents={firstProject.agents} onSent={() => { reloadAndNotify(); setChecked(new Set()) }} />
            </div>
          )}
        </div>
      </div>

      {/* Batch bar */}
      {someChecked && (
        <div className="shrink-0 flex items-center gap-4 border-y border-sky-200/80 bg-sky-50/50 px-8 py-2.5 dark:border-sky-900/30 dark:bg-sky-950/20">
          <input type="checkbox" checked={allChecked} onChange={toggleAll} className="size-3.5 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
          <span className="text-sm font-medium text-sky-800 dark:text-sky-300">{t('messages.selected', { count: String(checked.size) })}</span>
          {!allChecked && <button type="button" onClick={toggleAll} className="text-xs font-medium text-sky-600 hover:text-sky-800 dark:text-sky-400">{t('messages.selectAll')}</button>}
          <div className="flex items-center gap-2">
            <button type="button" disabled={batchBusy} onClick={() => void batchMarkRead()} className="rounded-lg border border-sky-200 bg-white px-3 py-1.5 text-sm font-medium text-sky-700 transition-colors hover:bg-sky-50 disabled:opacity-40 dark:border-sky-800 dark:bg-sky-900/40 dark:text-sky-300 dark:hover:bg-sky-900/60">{t('messages.batchMarkRead')}</button>
            <button type="button" disabled={batchBusy} onClick={() => void batchArchive()} className="rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-50 disabled:opacity-40 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800">{t('messages.batchArchive')}</button>
            <button type="button" disabled={batchBusy} onClick={() => void batchDelete()} className="rounded-lg border border-red-200 bg-white px-3 py-1.5 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-40 dark:border-red-900 dark:bg-red-950/30 dark:text-red-400 dark:hover:bg-red-900/50">{t('messages.batchDelete')}</button>
          </div>
          <button type="button" onClick={() => setChecked(new Set())} className="ml-auto text-sm text-sky-600 hover:text-sky-800 dark:text-sky-400">{t('forms.cancel')}</button>
        </div>
      )}

      <MessageDetailModal open={selected != null} message={selected} onClose={() => setSelected(null)} onMutated={reloadAndNotify} />

      {/* List */}
      <div className="flex-1 overflow-y-auto px-8 pb-8 pt-2">
        {state.status === 'loading' && (
          <div className="flex items-center gap-2 py-20 justify-center">
            <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {state.status === 'ok' && messages.length === 0 && (
          <div className="flex flex-col items-center justify-center py-24 text-center">
            <div className="mb-4 flex size-16 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <Mail className="size-7 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-lg font-medium text-neutral-600 dark:text-zinc-400">{t('workbench.emptyMessages')}</p>
            <p className="mt-1.5 text-sm text-neutral-400 dark:text-zinc-500">{t('workbench.emptyMessagesHint')}</p>
          </div>
        )}
        {state.status === 'ok' && messages.length > 0 && (
          <div className="space-y-3">
            {/* List toolbar */}
            <div className="flex items-center gap-3 px-1">
              <input type="checkbox" checked={allChecked} onChange={toggleAll} className="size-3.5 cursor-pointer rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
              <span className="text-xs text-neutral-400 dark:text-zinc-500">{messages.length} {t('workbench.tabMessages').toLowerCase()}</span>
            </div>
            {pagedWbMessages.map((row) => {
              const isSent = row.from === 'human'
              const unread = !isSent && !row.readAt
              const archived = Boolean(row.archivedAt)
              const isChecked = checked.has(row.id)
              return (
                <div
                  key={row.id}
                  onClick={() => setSelected(row)}
                  className={cn(
                    'group rounded-xl border transition-all duration-150 cursor-pointer',
                    isChecked
                      ? 'border-sky-300 bg-sky-50/50 shadow-sm dark:border-sky-800 dark:bg-sky-900/10'
                      : unread
                        ? 'border-sky-200/60 bg-sky-50/20 hover:border-sky-300/60 hover:bg-sky-50/40 hover:shadow-sm dark:border-sky-900/30 dark:bg-sky-900/[0.04] dark:hover:bg-sky-900/10'
                        : 'border-neutral-200/80 bg-white hover:border-neutral-300/60 hover:bg-neutral-50/60 hover:shadow-sm dark:border-zinc-700/60 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30',
                  )}
                >
                  <div className="flex items-start gap-4 px-5 py-4">
                    <div className="pt-1" onClick={(e) => e.stopPropagation()}>
                      <input type="checkbox" checked={isChecked} onChange={() => toggleOne(row.id)} className="size-4 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-3">
                        {unread && <span className="size-2.5 shrink-0 rounded-full bg-sky-500" />}
                        {isSent && <span className="size-2.5 shrink-0 rounded-full bg-emerald-400 dark:bg-emerald-600" />}
                        <span className="font-mono text-sm font-semibold text-neutral-800 dark:text-zinc-200">{row.from}</span>
                        <span className="text-xs text-neutral-400 dark:text-zinc-500">→ {row.to}</span>
                        <span className="ml-auto shrink-0 text-xs text-neutral-400 dark:text-zinc-500">{fmt(row.sentAt)}</span>
                      </div>
                      {row.subject && (
                        <p className="mt-1.5 text-sm font-medium text-neutral-900 dark:text-zinc-100">{row.subject}</p>
                      )}
                      <p className="mt-1 text-sm leading-relaxed text-neutral-500 dark:text-zinc-500">{preview(row.body)}</p>
                      <div className="mt-2.5 flex items-center gap-2">
                        {isSent && (
                          <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-100 px-2.5 py-0.5 text-[11px] font-semibold text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                            {t('messages.directionSent')}
                          </span>
                        )}
                        {unread && (
                          <span className="inline-flex items-center gap-1.5 rounded-full bg-sky-100 px-2.5 py-0.5 text-[11px] font-semibold text-sky-800 dark:bg-sky-900/30 dark:text-sky-300">
                            <span className="size-1.5 rounded-full bg-sky-500" />
                            {t('messages.badgeUnread')}
                          </span>
                        )}
                        {archived && (
                          <span className="rounded-full bg-neutral-100 px-2.5 py-0.5 text-[11px] font-semibold text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500">
                            {t('messages.badgeArchived')}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Quick actions */}
                  <div
                    className="border-t border-neutral-100/80 px-5 py-2.5 opacity-0 transition-opacity duration-150 group-hover:opacity-100 has-[textarea]:opacity-100 focus-within:opacity-100 dark:border-zinc-700/40"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <div className="flex items-center gap-3">
                      {!isSent && <InlineReply originalFrom={row.from} mailbox={row.mailbox} messageId={row.id} onSent={reloadAndNotify} />}
                      <div className="ml-auto flex items-center gap-2">
                        {unread && (
                          <button type="button" onClick={(e) => void quickMarkRead(row, e)} className="rounded-lg px-2.5 py-1 text-xs font-medium text-sky-700 transition-colors hover:bg-sky-50 dark:text-sky-400 dark:hover:bg-sky-900/20">{t('forms.markAsRead')}</button>
                        )}
                        {!archived && (
                          <button type="button" onClick={(e) => void quickArchive(row, e)} className="rounded-lg px-2.5 py-1 text-xs font-medium text-neutral-500 transition-colors hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800">{t('forms.archiveMessage')}</button>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              )
            })}
            <Pagination page={wbMsgPage} totalPages={totalWbMsgPages} onPageChange={setWbMsgPage} />
          </div>
        )}
      </div>
    </div>
  )
}

/* ── Tasks panel ──────────────────────────────────────────────────────────── */

type TaskView = 'list' | 'table' | 'kanban'

function TasksPanel({ projectsAgents, onMutated }: { projectsAgents: ProjectAgents[]; onMutated?: () => void }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const [view, setView] = useState<TaskView>('list')
  const [statusFilter, setStatusFilter] = useState('pending')
  const [projectFilter, setProjectFilter] = useState('')
  const [priorityFilter, setPriorityFilter] = useState<string>('')
  const [taskSort, setTaskSort] = useState<'newest' | 'oldest' | 'priority'>('newest')
  const [reloadKey, setReloadKey] = useState(0)
  const [detailRow, setDetailRow] = useState<TaskRow | null>(null)
  const [editRow, setEditRow] = useState<TaskRow | null>(null)
  const [checked, setChecked] = useState<Set<string>>(new Set())
  const [batchBusy, setBatchBusy] = useState(false)
  const firstProject = projectsAgents[0]

  const qp = new URLSearchParams()
  if (view === 'list' && statusFilter) qp.set('status', statusFilter)
  if (projectFilter) qp.set('project', projectFilter)
  const qs = qp.toString() ? `?${qp.toString()}` : ''
  const [wbTaskPage, setWbTaskPage] = useState(1)
  const wbTasksPerPage = 20

  const state = useApiJson<TaskRow[]>(`/api/v1/workbench/tasks${qs}`, reloadKey)
  const rawTasks = state.status === 'ok' ? (state.data ?? []) : []

  const tasks = useMemo(() => {
    let filtered = rawTasks.filter((t) => !t.archived)
    if (priorityFilter !== '') filtered = filtered.filter((t) => t.priority === Number(priorityFilter))
    const sorted = [...filtered]
    if (taskSort === 'oldest') sorted.reverse()
    else if (taskSort === 'priority') sorted.sort((a, b) => a.priority - b.priority)
    return sorted
  }, [rawTasks, priorityFilter, taskSort])

  const taskOptions = useMemo(
    () => tasks.map((r) => ({ id: r.id, title: r.title, project: r.project })),
    [tasks],
  )

  const totalWbTaskPages = Math.ceil(tasks.length / wbTasksPerPage)
  const pagedWbTasks = useMemo(() => {
    const start = (wbTaskPage - 1) * wbTasksPerPage
    return tasks.slice(start, start + wbTasksPerPage)
  }, [tasks, wbTaskPage])

  useEffect(() => { setWbTaskPage(1) }, [statusFilter, projectFilter, priorityFilter, taskSort, view])

  const projects = useMemo(() => {
    const s = new Set(tasks.map((t) => t.project))
    return Array.from(s).sort()
  }, [tasks])

  const reload = useCallback(() => { setReloadKey((k) => k + 1); setChecked(new Set()); onMutated?.() }, [onMutated])

  async function handleKanbanStatusChange(task: TaskRow, newStatus: string) {
    await apiPut('/api/v1/tasks/update', { project: task.project, agent: task.agent, id: task.id, status: newStatus })
    reload()
  }

  const allChecked = tasks.length > 0 && checked.size === tasks.length
  const someChecked = checked.size > 0
  function toggleAll() { setChecked(allChecked ? new Set() : new Set(tasks.map((t) => t.id))) }
  function toggleOne(id: string) {
    setChecked((prev) => { const n = new Set(prev); if (n.has(id)) n.delete(id); else n.add(id); return n })
  }
  function getCheckedRows() { return tasks.filter((t) => checked.has(t.id)) }

  async function batchCancel() {
    const count = checked.size
    if (!window.confirm(t('tasks.confirmBatchCancel', { count: String(count) }))) return
    setBatchBusy(true)
    try {
      for (const row of getCheckedRows().filter((r) => !isTerminal(r.status)))
        await apiPost('/api/v1/tasks/cancel', { project: row.project, agent: row.agent, id: row.id })
      reload()
    } finally { setBatchBusy(false) }
  }
  async function batchArchive() {
    setBatchBusy(true)
    try {
      for (const row of getCheckedRows())
        await apiPost('/api/v1/tasks/archive', { project: row.project, agent: row.agent, id: row.id })
      reload()
    } finally { setBatchBusy(false) }
  }
  async function batchDelete() {
    const count = checked.size
    if (!window.confirm(t('tasks.confirmBatchDelete', { count: String(count) }))) return
    setBatchBusy(true)
    try {
      for (const row of getCheckedRows())
        await apiPost('/api/v1/tasks/delete', { project: row.project, agent: row.agent, id: row.id })
      reload()
    } finally { setBatchBusy(false) }
  }

  async function quickCancel(row: TaskRow, e: React.MouseEvent) {
    e.stopPropagation()
    if (!window.confirm(t('tasks.confirmCancel'))) return
    await apiPost('/api/v1/tasks/cancel', { project: row.project, agent: row.agent, id: row.id })
    reload()
  }
  async function quickArchive(row: TaskRow, e: React.MouseEvent) {
    e.stopPropagation()
    await apiPost('/api/v1/tasks/archive', { project: row.project, agent: row.agent, id: row.id })
    reload()
  }
  async function quickDelete(row: TaskRow, e: React.MouseEvent) {
    e.stopPropagation()
    if (!window.confirm(t('tasks.confirmDelete'))) return
    await apiPost('/api/v1/tasks/delete', { project: row.project, agent: row.agent, id: row.id })
    reload()
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Filters */}
      <div className="shrink-0 px-8 py-4">
        <div className="flex flex-wrap items-center gap-3">
          {/* View toggle */}
          <div className="flex rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
            <button type="button" onClick={() => setView('list')} title={t('workbench.viewList')} className={cn(
              'flex items-center gap-1.5 rounded-l-lg px-3 py-1.5 text-sm font-medium transition-colors',
              view === 'list' ? 'bg-neutral-100 text-neutral-800 dark:bg-zinc-800 dark:text-zinc-200' : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400'
            )}>
              <List className="size-3.5" strokeWidth={2} />
            </button>
            <button type="button" onClick={() => setView('table')} title={t('tasks.view_table')} className={cn(
              'flex items-center gap-1.5 border-x border-neutral-200/80 px-3 py-1.5 text-sm font-medium transition-colors dark:border-zinc-700/60',
              view === 'table' ? 'bg-neutral-100 text-neutral-800 dark:bg-zinc-800 dark:text-zinc-200' : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400'
            )}>
              <LayoutList className="size-3.5" strokeWidth={2} />
            </button>
            <button type="button" onClick={() => setView('kanban')} title={t('workbench.viewKanban')} className={cn(
              'flex items-center gap-1.5 rounded-r-lg px-3 py-1.5 text-sm font-medium transition-colors',
              view === 'kanban' ? 'bg-neutral-100 text-neutral-800 dark:bg-zinc-800 dark:text-zinc-200' : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400'
            )}>
              <KanbanSquare className="size-3.5" strokeWidth={2} />
            </button>
          </div>
          {view === 'list' && (
            <select value={statusFilter} onChange={(e) => { setStatusFilter(e.target.value); setChecked(new Set()) }} className={selectCls}>
              <option value="">{t('tasks.filterStatus')}: {t('messages.readAll')}</option>
              {STATUS_KEYS.map((s) => <option key={s} value={s}>{t(`tasks.status.${s}`)}</option>)}
            </select>
          )}
          <select value={priorityFilter} onChange={(e) => { setPriorityFilter(e.target.value); setChecked(new Set()) }} className={selectCls}>
            <option value="">{t('workbench.priorityAll')}</option>
            {[0, 1, 2, 3].map((p) => <option key={p} value={p}>P{p}</option>)}
          </select>
          <select value={taskSort} onChange={(e) => { setTaskSort(e.target.value as 'newest' | 'oldest' | 'priority'); setChecked(new Set()) }} className={selectCls}>
            <option value="newest">{t('workbench.sortNewest')}</option>
            <option value="oldest">{t('workbench.sortOldest')}</option>
            <option value="priority">{t('workbench.sortPriority')}</option>
          </select>
          {projects.length > 1 && (
            <select value={projectFilter} onChange={(e) => { setProjectFilter(e.target.value); setChecked(new Set()) }} className={cn(selectCls, 'font-mono')}>
              <option value="">{t('workbench.filterProject')}: {t('workbench.allProjects')}</option>
              {projects.map((p) => <option key={p} value={p}>{p}</option>)}
            </select>
          )}
          <button type="button" onClick={reload} className="flex items-center gap-1 rounded-md px-2 py-1 text-[13px] text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
            <RefreshCw className="size-3" strokeWidth={2} />
            {t('api.refresh')}
          </button>
          {firstProject && (
            <div className="ml-auto flex items-center gap-2">
              <RunAgentDialog projects={projectsAgents} onDone={reload} />
              <CreateTaskDialog projectId={firstProject.projectId} agents={firstProject.agents} allProjectsAgents={projectsAgents} taskOptions={taskOptions} onCreated={reload} />
            </div>
          )}
        </div>
      </div>

      {/* Batch bar */}
      {someChecked && (
        <div className="shrink-0 flex items-center gap-4 border-y border-sky-200/80 bg-sky-50/50 px-8 py-2.5 dark:border-sky-900/30 dark:bg-sky-950/20">
          <input type="checkbox" checked={allChecked} onChange={toggleAll} className="size-3.5 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
          <span className="text-sm font-medium text-sky-800 dark:text-sky-300">{t('messages.selected', { count: String(checked.size) })}</span>
          {!allChecked && <button type="button" onClick={toggleAll} className="text-xs font-medium text-sky-600 hover:text-sky-800 dark:text-sky-400">{t('messages.selectAll')}</button>}
          <div className="flex items-center gap-2">
            <button type="button" disabled={batchBusy} onClick={() => void batchCancel()} className="rounded-lg border border-amber-200 bg-white px-3 py-1.5 text-sm font-medium text-amber-700 transition-colors hover:bg-amber-50 disabled:opacity-40 dark:border-amber-800 dark:bg-amber-900/40 dark:text-amber-300 dark:hover:bg-amber-900/50">{t('tasks.batchCancel')}</button>
            <button type="button" disabled={batchBusy} onClick={() => void batchArchive()} className="rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-50 disabled:opacity-40 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800">{t('tasks.batchArchive')}</button>
            <button type="button" disabled={batchBusy} onClick={() => void batchDelete()} className="rounded-lg border border-red-200 bg-white px-3 py-1.5 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-40 dark:border-red-900 dark:bg-red-950/30 dark:text-red-400 dark:hover:bg-red-900/50">{t('tasks.batchDelete')}</button>
          </div>
          <button type="button" onClick={() => setChecked(new Set())} className="ml-auto text-sm text-sky-600 hover:text-sky-800 dark:text-sky-400">{t('forms.cancel')}</button>
        </div>
      )}

      {/* Detail / Edit modals */}
      {editRow && <EditTaskModal task={editRow} taskOptions={taskOptions} onClose={() => setEditRow(null)} onSaved={reload} />}
      {detailRow && <TaskDetailModal task={detailRow} onClose={() => setDetailRow(null)} onEdit={(r) => { setDetailRow(null); setEditRow(r) }} />}

      {/* Kanban view */}
      {view === 'kanban' && (
        <div className="flex-1 overflow-y-auto overflow-x-auto px-8 pb-8 pt-2">
          {state.status === 'loading' && (
            <div className="flex items-center gap-2 py-20 justify-center">
              <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
              <span className="text-sm text-neutral-500">{t('api.loading')}</span>
            </div>
          )}
          {state.status === 'ok' && (
            <TaskKanban
              tasks={tasks}
              onTaskClick={setDetailRow}
              onStatusChange={(task, status) => void handleKanbanStatusChange(task, status)}
              showProject
            />
          )}
        </div>
      )}

      {/* Table view */}
      {view === 'table' && (
        <div className="flex-1 overflow-y-auto px-8 pb-8 pt-2">
          {state.status === 'loading' && (
            <div className="flex items-center gap-2 py-20 justify-center">
              <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
              <span className="text-sm text-neutral-500">{t('api.loading')}</span>
            </div>
          )}
          {state.status === 'ok' && tasks.length === 0 && (
            <div className="flex flex-col items-center justify-center py-24 text-center">
              <div className="mb-4 flex size-16 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
                <ListTodo className="size-7 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
              </div>
              <p className="text-lg font-medium text-neutral-600 dark:text-zinc-400">{t('workbench.emptyTasks')}</p>
              <p className="mt-1.5 text-sm text-neutral-400 dark:text-zinc-500">{t('workbench.emptyTasksHint')}</p>
            </div>
          )}
          {state.status === 'ok' && tasks.length > 0 && (
            <div className="space-y-3">
              <div className="flex items-center gap-3 px-1">
                <input type="checkbox" checked={allChecked} onChange={toggleAll} className="size-3.5 cursor-pointer rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
                <span className="text-xs text-neutral-400 dark:text-zinc-500">{tasks.length} {t('workbench.tabTasks').toLowerCase()}</span>
              </div>
              <TaskTable
                tasks={pagedWbTasks}
                checked={checked}
                allChecked={allChecked}
                someChecked={someChecked}
                onToggleAll={toggleAll}
                onToggleOne={toggleOne}
                onRowClick={setDetailRow}
                onEdit={(row) => { setDetailRow(null); setEditRow(row) }}
                onCancel={(row, e) => void quickCancel(row, e)}
                onArchive={(row, e) => void quickArchive(row, e)}
                onDelete={(row, e) => void quickDelete(row, e)}
                showProject
              />
              <Pagination page={wbTaskPage} totalPages={totalWbTaskPages} onPageChange={setWbTaskPage} />
            </div>
          )}
        </div>
      )}

      {/* List view */}
      {view === 'list' && (
      <div className="flex-1 overflow-y-auto px-8 pb-8 pt-2">
        {state.status === 'loading' && (
          <div className="flex items-center gap-2 py-20 justify-center">
            <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {state.status === 'ok' && tasks.length === 0 && (
          <div className="flex flex-col items-center justify-center py-24 text-center">
            <div className="mb-4 flex size-16 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <ListTodo className="size-7 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-lg font-medium text-neutral-600 dark:text-zinc-400">{t('workbench.emptyTasks')}</p>
            <p className="mt-1.5 text-sm text-neutral-400 dark:text-zinc-500">{t('workbench.emptyTasksHint')}</p>
          </div>
        )}
        {state.status === 'ok' && tasks.length > 0 && (
          <div className="space-y-3">
            {/* List toolbar */}
            <div className="flex items-center gap-3 px-1">
              <input type="checkbox" checked={allChecked} onChange={toggleAll} className="size-3.5 cursor-pointer rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
              <span className="text-xs text-neutral-400 dark:text-zinc-500">{tasks.length} {t('workbench.tabTasks').toLowerCase()}</span>
            </div>
            {pagedWbTasks.map((row) => {
              const prio = priorityLabel[row.priority] ?? priorityLabel[2]
              const sCls = statusColor[row.status] ?? statusColor.pending
              const terminal = isTerminal(row.status)
              const isChecked = checked.has(row.id)
              return (
                <div
                  key={row.id}
                  onClick={() => setDetailRow(row)}
                  className={cn(
                    'group rounded-xl border transition-all duration-150 cursor-pointer',
                    isChecked
                      ? 'border-sky-300 bg-sky-50/50 shadow-sm dark:border-sky-800 dark:bg-sky-900/10'
                      : 'border-neutral-200/80 bg-white hover:border-neutral-300/60 hover:shadow-sm dark:border-zinc-700/60 dark:bg-zinc-900/20 dark:hover:border-zinc-700/60',
                  )}
                >
                  <div className="flex items-start gap-4 px-5 py-4">
                    <div className="pt-0.5" onClick={(e) => e.stopPropagation()}>
                      <input type="checkbox" checked={isChecked} onChange={() => toggleOne(row.id)} className="size-4 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2.5">
                        <span className={cn('text-xs font-bold', prio.cls)}>{prio.text}</span>
                        <span className="text-sm font-medium text-neutral-900 dark:text-zinc-100">{row.title}</span>
                        {row.type && (
                          <span className="rounded-md border border-neutral-200 bg-neutral-50 px-2 py-0.5 text-[11px] font-medium text-neutral-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-500">{t(`forms.taskType.${row.type}`, { defaultValue: row.type })}</span>
                        )}
                        <span className={cn('ml-auto inline-block shrink-0 rounded-full px-3 py-0.5 text-[11px] font-semibold', sCls)}>
                          {t(`tasks.status.${row.status}`, { defaultValue: row.status })}
                        </span>
                      </div>
                      {row.prompt && (
                        <p className="mt-1.5 line-clamp-2 text-sm leading-relaxed text-neutral-500 dark:text-zinc-500">
                          {row.prompt.replace(/\s+/g, ' ').slice(0, 200)}
                        </p>
                      )}
                      <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-neutral-500 dark:text-zinc-500">
                        <Link
                          to={`/projects/${encodeURIComponent(row.project)}/tasks`}
                          onClick={(e) => e.stopPropagation()}
                          className="font-mono text-sky-700 underline-offset-2 hover:underline dark:text-sky-400"
                        >
                          {row.project}
                        </Link>
                        <span className="font-mono text-neutral-400 dark:text-zinc-500">{row.agent}</span>
                        {row.assignee && row.assignee !== row.agent && (
                          <span className="rounded bg-violet-50 px-1.5 py-0.5 text-xs text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">
                            → {row.assignee}
                          </span>
                        )}
                        <span className="text-neutral-400 dark:text-zinc-500">{fmt(row.updatedAt)}</span>
                      </div>
                    </div>
                  </div>

                  {/* Quick actions */}
                  <div
                    className="flex items-center gap-3 border-t border-neutral-100/80 px-5 py-2.5 opacity-0 transition-opacity duration-150 group-hover:opacity-100 dark:border-zinc-700/40"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={(e) => { e.stopPropagation(); setDetailRow(null); setEditRow(row) }}
                        className="flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs font-medium text-sky-700 transition-colors hover:bg-sky-50 dark:text-sky-400 dark:hover:bg-sky-900/20"
                      >
                        <Pencil className="size-3" strokeWidth={2} />
                        {t('tasks.edit')}
                      </button>
                    </div>
                    <div className="ml-auto flex items-center gap-2">
                      {!terminal && (
                        <button type="button" onClick={(e) => void quickCancel(row, e)} className="rounded-lg px-2.5 py-1 text-xs font-medium text-amber-700 transition-colors hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/20">{t('tasks.cancel')}</button>
                      )}
                      <button type="button" onClick={(e) => void quickArchive(row, e)} className="rounded-lg px-2.5 py-1 text-xs font-medium text-neutral-500 transition-colors hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800">{t('tasks.archive')}</button>
                      <button type="button" onClick={(e) => void quickDelete(row, e)} className="rounded-lg px-2.5 py-1 text-xs font-medium text-red-500 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20">
                        <Trash2 className="size-3.5" strokeWidth={1.8} />
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
            <Pagination page={wbTaskPage} totalPages={totalWbTaskPages} onPageChange={setWbTaskPage} />
          </div>
        )}
      </div>
      )}
    </div>
  )
}

/* ── Overview panel ───────────────────────────────────────────────────────── */

type ProjectOverview = {
  project: string
  agentCount: number
  heartbeatEnabled: number
  runningAgents: number
  schedulerRunning: boolean
  pendingTasks: number
  runningTasks: number
  completedTasks: number
  totalTasks: number
  unreadMessages: number
  totalMessages: number
}

function OverviewPanel() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [reloadKey, setReloadKey] = useState(0)
  const [busy, setBusy] = useState<string | null>(null)
  const state = useApiJson<ProjectOverview[]>('/api/v1/workbench/overview', reloadKey)
  const projects = state.status === 'ok' ? (state.data ?? []) : []

  const reload = useCallback(() => setReloadKey(k => k + 1), [])

  const allRunning = projects.length > 0 && projects.every(p => p.schedulerRunning)
  const someRunning = projects.some(p => p.schedulerRunning)

  async function toggleProject(project: string, running: boolean) {
    setBusy(project)
    try {
      const url = running ? '/api/v1/scheduler/stop' : '/api/v1/scheduler/start'
      await apiPost(url, { project })
      reload()
    } catch { /* toast handled by apiPost */ }
    finally { setBusy(null) }
  }

  async function toggleAll(start: boolean) {
    setBusy('__all')
    try {
      if (start) {
        for (const p of projects) {
          if (!p.schedulerRunning) await apiPost('/api/v1/scheduler/start', { project: p.project })
        }
      } else {
        for (const p of projects) {
          if (p.schedulerRunning) await apiPost('/api/v1/scheduler/stop', { project: p.project })
        }
      }
      reload()
    } catch { /* ignore */ }
    finally { setBusy(null) }
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <div className="shrink-0 px-8 py-4">
        <div className="flex items-center gap-3">
          <button type="button" onClick={reload} className="flex items-center gap-1 rounded-md px-2 py-1 text-[13px] text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
            <RefreshCw className="size-3" strokeWidth={2} />
            {t('api.refresh')}
          </button>
          {projects.length > 0 && (
            <div className="ml-auto flex items-center gap-2">
              {!allRunning && (
                <button type="button" disabled={busy === '__all'}
                  onClick={() => void toggleAll(true)}
                  className="flex items-center gap-1.5 rounded-lg border border-sky-200 bg-white px-3 py-1.5 text-sm font-medium text-sky-700 transition-colors hover:bg-sky-50 disabled:opacity-50 dark:border-sky-800 dark:bg-zinc-900 dark:text-sky-400 dark:hover:bg-sky-900/20">
                  <Play className="size-3.5" strokeWidth={2} />
                  {t('workbench.startAll')}
                </button>
              )}
              {someRunning && (
                <button type="button" disabled={busy === '__all'}
                  onClick={() => void toggleAll(false)}
                  className="flex items-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-50 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800">
                  <Square className="size-3" strokeWidth={2} />
                  {t('workbench.stopAll')}
                </button>
              )}
            </div>
          )}
        </div>
      </div>
      <div className="flex-1 overflow-y-auto px-8 pb-8">
        {state.status === 'loading' && (
          <div className="flex items-center gap-2 py-20 justify-center">
            <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {state.status === 'ok' && projects.length === 0 && (
          <div className="flex flex-col items-center justify-center py-24 text-center">
            <p className="text-lg font-medium text-neutral-600 dark:text-zinc-400">{t('workbench.noProjects')}</p>
          </div>
        )}
        {state.status === 'ok' && projects.length > 0 && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {projects.map(p => {
              const isBusy = busy === p.project
              return (
                <div
                  key={p.project}
                  onClick={() => navigate(`/projects/${encodeURIComponent(p.project)}/schedule`)}
                  className="group cursor-pointer rounded-xl border border-neutral-200/80 bg-white p-5 transition-all hover:border-neutral-300 hover:shadow-md dark:border-zinc-700/60 dark:bg-zinc-900/40 dark:hover:border-zinc-600"
                >
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-neutral-900 dark:text-zinc-100">{p.project}</h3>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={e => { e.stopPropagation(); void toggleProject(p.project, p.schedulerRunning) }}
                      className={cn(
                        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold transition-colors disabled:opacity-50',
                        p.schedulerRunning
                          ? 'bg-sky-50 text-sky-700 hover:bg-sky-100 dark:bg-sky-900/20 dark:text-sky-400 dark:hover:bg-sky-900/40'
                          : 'bg-neutral-100 text-neutral-500 hover:bg-neutral-200 dark:bg-zinc-800 dark:text-zinc-500 dark:hover:bg-zinc-700'
                      )}
                    >
                      {isBusy ? (
                        <div className="size-3 animate-spin rounded-full border-2 border-current border-t-transparent" />
                      ) : p.schedulerRunning ? (
                        <Power className="size-3" strokeWidth={2.5} />
                      ) : (
                        <Play className="size-3" strokeWidth={2.5} />
                      )}
                      {p.schedulerRunning ? t('workbench.schedulerOn') : t('workbench.schedulerOff')}
                    </button>
                  </div>

                  <div className="mt-4 grid grid-cols-2 gap-3">
                    <div className="rounded-lg bg-neutral-50 px-3 py-2 dark:bg-zinc-800/40">
                      <div className="text-[11px] font-medium text-neutral-400 dark:text-zinc-500">{t('workbench.ovAgents')}</div>
                      <div className="mt-0.5 text-lg font-bold text-neutral-800 dark:text-zinc-200">
                        {p.agentCount}
                        <span className="ml-1.5 text-xs font-normal text-neutral-500 dark:text-zinc-500">
                          {p.heartbeatEnabled > 0 && <span className="text-sky-600 dark:text-sky-400">{p.heartbeatEnabled} {t('workbench.ovHeartbeat')}</span>}
                          {p.heartbeatEnabled > 0 && p.runningAgents > 0 && ' · '}
                          {p.runningAgents > 0 && <span className="text-emerald-600 dark:text-emerald-400">{p.runningAgents} {t('workbench.ovRunningAgents')}</span>}
                        </span>
                      </div>
                    </div>
                    <div className="rounded-lg bg-neutral-50 px-3 py-2 dark:bg-zinc-800/40">
                      <div className="text-[11px] font-medium text-neutral-400 dark:text-zinc-500">{t('workbench.ovTasks')}</div>
                      <div className="mt-0.5 flex items-baseline gap-2">
                        <span className="text-lg font-bold text-neutral-800 dark:text-zinc-200">{p.totalTasks}</span>
                        <span className="text-xs text-neutral-500 dark:text-zinc-500">
                          {p.pendingTasks > 0 && <span className="text-amber-600 dark:text-amber-400">{p.pendingTasks} {t('workbench.ovPending')}</span>}
                          {p.pendingTasks > 0 && p.runningTasks > 0 && ' · '}
                          {p.runningTasks > 0 && <span className="text-sky-600 dark:text-sky-400">{p.runningTasks} {t('workbench.ovRunning')}</span>}
                        </span>
                      </div>
                    </div>
                    <div className="rounded-lg bg-neutral-50 px-3 py-2 dark:bg-zinc-800/40">
                      <div className="text-[11px] font-medium text-neutral-400 dark:text-zinc-500">{t('workbench.ovMessages')}</div>
                      <div className="mt-0.5 flex items-baseline gap-2">
                        <span className="text-lg font-bold text-neutral-800 dark:text-zinc-200">{p.totalMessages}</span>
                        {p.unreadMessages > 0 && (
                          <span className="text-xs font-medium text-sky-600 dark:text-sky-400">{p.unreadMessages} {t('workbench.ovUnread')}</span>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="mt-3 flex items-center justify-end text-xs font-medium text-sky-600 opacity-0 transition-opacity group-hover:opacity-100 dark:text-sky-400">
                    {t('workbench.ovViewSchedule')} →
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

/* ── Quick links panel ─────────────────────────────────────────────────────── */

function QuickLinksPanel({ onChanged }: { onChanged: () => void }) {
  const { t } = useTranslation()
  const [links, setLinks] = useState<QuickLink[]>(() => getQuickLinks())

  const reload = useCallback(() => {
    setLinks(getQuickLinks())
    onChanged()
  }, [onChanged])

  useEffect(() => {
    window.addEventListener('quick-links-changed', reload)
    window.addEventListener('storage', reload)
    return () => {
      window.removeEventListener('quick-links-changed', reload)
      window.removeEventListener('storage', reload)
    }
  }, [reload])

  function remove(path: string) {
    removeQuickLink(path)
    reload()
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <div className="flex-1 overflow-y-auto px-8 py-6">
        {links.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 text-center">
            <div className="mb-4 flex size-16 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <Star className="size-7 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-lg font-medium text-neutral-600 dark:text-zinc-400">{t('workbench.quickLinksEmpty')}</p>
            <p className="mt-1.5 text-sm text-neutral-400 dark:text-zinc-500">{t('workbench.quickLinksHint')}</p>
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {links.map((link) => (
              <Link
                key={link.path}
                to={link.path}
                className="group rounded-xl border border-neutral-200/80 bg-white p-4 transition-all hover:border-sky-200 hover:bg-sky-50/30 hover:shadow-sm dark:border-zinc-700/60 dark:bg-zinc-900/40 dark:hover:border-sky-900/50 dark:hover:bg-sky-900/10"
              >
                <div className="flex items-start gap-3">
                  <div className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-lg bg-sky-100/70 dark:bg-sky-900/30">
                    <LinkIcon className="size-4 text-sky-600 dark:text-sky-400" strokeWidth={1.8} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-semibold text-neutral-900 dark:text-zinc-100">{link.title}</p>
                    <p className="mt-1 truncate font-mono text-xs text-neutral-400 dark:text-zinc-500">{link.path}</p>
                  </div>
                  <button
                    type="button"
                    onClick={(e) => { e.preventDefault(); e.stopPropagation(); remove(link.path) }}
                    className="rounded-md p-1 text-neutral-300 opacity-0 transition-all hover:bg-red-50 hover:text-red-500 group-hover:opacity-100 dark:text-zinc-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                    title={t('workbench.quickLinkRemove')}
                  >
                    <Trash2 className="size-3.5" strokeWidth={1.8} />
                  </button>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

/* ── Tab component (Plane-style) ──────────────────────────────────────────── */

function TabButton({
  active,
  onClick,
  children,
  badge,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
  badge?: number
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'relative flex items-center gap-2 border-b-2 px-4 py-3.5 text-sm font-medium whitespace-nowrap transition-colors',
        active
          ? 'border-sky-600 text-sky-700 dark:border-sky-400 dark:text-sky-400'
          : 'border-transparent text-neutral-500 hover:text-neutral-800 dark:text-zinc-500 dark:hover:text-zinc-300',
      )}
    >
      {children}
      {badge != null && badge > 0 && (
        <span className="min-w-[1.25rem] rounded-full bg-sky-600 px-1.5 py-0.5 text-center text-[10px] font-bold leading-none text-white">
          {badge}
        </span>
      )}
    </button>
  )
}

/* ── Main page ────────────────────────────────────────────────────────────── */

export default function WorkbenchPage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<Tab>('overview')
  const projectsAgents = useProjectsAgents()
  const [badgeKey, setBadgeKey] = useState(0)
  const [, setQuickLinksVersion] = useState(0)

  const msgCount = useApiJson<MessageRow[]>('/api/v1/workbench/messages?direction=inbox&read=unread', badgeKey)
  const unreadMsgs = msgCount.status === 'ok' ? msgCount.data.length : 0
  const taskCount = useApiJson<TaskRow[]>('/api/v1/workbench/tasks?status=pending', badgeKey)
  const pendingTasks = taskCount.status === 'ok' ? taskCount.data.filter((t) => !t.archived).length : 0
  const refreshBadge = useCallback(() => setBadgeKey((k) => k + 1), [])
  const refreshQuickLinks = useCallback(() => setQuickLinksVersion((v) => v + 1), [])

  useEffect(() => {
    window.addEventListener('quick-links-changed', refreshQuickLinks)
    window.addEventListener('storage', refreshQuickLinks)
    return () => {
      window.removeEventListener('quick-links-changed', refreshQuickLinks)
      window.removeEventListener('storage', refreshQuickLinks)
    }
  }, [refreshQuickLinks])

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Header */}
      <div className="shrink-0 px-8 pt-6 pb-0">
        <div className="flex items-center gap-3.5">
          <div>
            <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('workbench.title')}</h1>
            <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('workbench.subtitle')}</p>
          </div>
        </div>
      </div>

      {/* Tabs — Plane-style border-b-2 underline */}
      <div className="shrink-0 border-b border-neutral-200/80 px-8 dark:border-zinc-700/50">
        <div className="-mb-px flex gap-1">
          <TabButton active={tab === 'overview'} onClick={() => setTab('overview')}>
            <LayoutGrid className="size-4" strokeWidth={1.8} />
            {t('workbench.tabOverview')}
          </TabButton>
          <TabButton active={tab === 'quickLinks'} onClick={() => setTab('quickLinks')}>
            <Star className="size-4" strokeWidth={1.8} />
            {t('workbench.tabQuickLinks')}
          </TabButton>
          <TabButton active={tab === 'messages'} onClick={() => setTab('messages')} badge={unreadMsgs}>
            <MessageSquare className="size-4" strokeWidth={1.8} />
            {t('workbench.tabMessages')}
          </TabButton>
          <TabButton active={tab === 'tasks'} onClick={() => setTab('tasks')} badge={pendingTasks}>
            <ListTodo className="size-4" strokeWidth={1.8} />
            {t('workbench.tabTasks')}
          </TabButton>
        </div>
      </div>

      {/* Panel */}
      {tab === 'overview' && <OverviewPanel />}
      {tab === 'quickLinks' && <QuickLinksPanel onChanged={refreshQuickLinks} />}
      {tab === 'messages' && <MessagesPanel projectsAgents={projectsAgents} onMutated={refreshBadge} />}
      {tab === 'tasks' && <TasksPanel projectsAgents={projectsAgents} onMutated={refreshBadge} />}
    </div>
  )
}
