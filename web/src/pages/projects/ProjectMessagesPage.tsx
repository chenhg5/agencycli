import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ListFilter, RefreshCw, X } from 'lucide-react'
import {
  MessageDetailModal,
  type MessageDetailModel,
} from '../../components/project/MessageDetailModal'
import { CreateMessageDialog } from '../../components/project/CreateMessageDialog'
import { Pagination } from '../../components/ui/Pagination'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { apiPost } from '../../lib/api'
import { cn } from '../../lib/cn'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'

type MessageRow = MessageDetailModel
type AgentRow = { name: string }

function preview(body: string, max = 120) {
  const one = body.replace(/\s+/g, ' ').trim()
  return one.length <= max ? one : `${one.slice(0, max)}…`
}

type Filters = {
  read: 'all' | 'read' | 'unread'
  archived: 'all' | 'no' | 'yes'
  from: string
  mailbox: string
}

const defaultFilters: Filters = { read: 'unread', archived: 'no', from: '', mailbox: '' }

function buildQuery(f: Filters) {
  const p = new URLSearchParams()
  if (f.read !== 'all') p.set('read', f.read)
  if (f.archived !== 'no') p.set('archived', f.archived)
  if (f.from) p.set('from', f.from)
  if (f.mailbox) p.set('mailbox', f.mailbox)
  const qs = p.toString()
  return qs ? `?${qs}` : ''
}

const selectCls =
  'h-8 rounded-md border border-neutral-200/80 bg-white px-2.5 pr-7 text-[13px] text-neutral-700 outline-none transition-colors hover:border-neutral-300 focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-600 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-900 [&>option]:dark:text-zinc-300'

export default function ProjectMessagesPage() {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const { projectId } = useParams<{ projectId: string }>()
  const base =
    projectId != null && projectId !== ''
      ? `/api/v1/projects/${encodeURIComponent(projectId)}`
      : null

  const [filters, setFilters] = useState<Filters>({ ...defaultFilters })
  const [reloadKey, setReloadKey] = useState(0)
  const [selected, setSelected] = useState<MessageRow | null>(null)
  const [checked, setChecked] = useState<Set<string>>(new Set())
  const [batchBusy, setBatchBusy] = useState(false)
  const [msgPage, setMsgPage] = useState(1)
  const msgsPerPage = 20

  const queryString = useMemo(() => buildQuery(filters), [filters])
  const messagesPath = base != null ? `${base}/messages${queryString}` : null
  const agentsPath = base != null ? `${base}/agents` : null

  const state = useApiJson<MessageRow[]>(messagesPath, reloadKey)
  const agentsState = useApiJson<AgentRow[]>(agentsPath, reloadKey)
  const messages = state.status === 'ok' ? (state.data ?? []) : []
  const agents = agentsState.status === 'ok' ? (agentsState.data ?? []) : []

  const totalMsgPages = Math.ceil(messages.length / msgsPerPage)
  const pagedMessages = useMemo(() => {
    const start = (msgPage - 1) * msgsPerPage
    return messages.slice(start, start + msgsPerPage)
  }, [messages, msgPage])

  useEffect(() => {
    setMsgPage(1)
  }, [filters])

  function setFilter<K extends keyof Filters>(key: K, val: Filters[K]) {
    setFilters((prev) => ({ ...prev, [key]: val }))
    setChecked(new Set())
  }
  function resetFilters() {
    setFilters({ ...defaultFilters })
    setChecked(new Set())
  }
  const hasFilters = filters.read !== 'all' || filters.archived !== 'no' || filters.from !== '' || filters.mailbox !== ''

  const allChecked = messages.length > 0 && checked.size === messages.length
  const someChecked = checked.size > 0
  function toggleAll() {
    setChecked(allChecked ? new Set() : new Set(messages.map((m) => m.id)))
  }
  function toggleOne(id: string) {
    setChecked((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }
  function getCheckedRows() {
    return messages.filter((m) => checked.has(m.id))
  }

  async function batchMarkRead() {
    setBatchBusy(true)
    try {
      for (const row of getCheckedRows().filter((r) => !r.readAt))
        await apiPost('/api/v1/messages/mark-read', { mailbox: row.mailbox, id: row.id })
      setChecked(new Set()); setReloadKey((k) => k + 1)
    } finally { setBatchBusy(false) }
  }
  async function batchArchive() {
    setBatchBusy(true)
    try {
      for (const row of getCheckedRows().filter((r) => !r.archivedAt))
        await apiPost('/api/v1/messages/archive', { mailbox: row.mailbox, id: row.id })
      setChecked(new Set()); setReloadKey((k) => k + 1)
    } finally { setBatchBusy(false) }
  }
  async function batchDelete() {
    const count = checked.size
    if (!window.confirm(t('messages.confirmBatchDelete', { count: String(count) }))) return
    setBatchBusy(true)
    try {
      for (const row of getCheckedRows())
        await apiPost('/api/v1/messages/delete', { mailbox: row.mailbox, id: row.id })
      setChecked(new Set()); setReloadKey((k) => k + 1)
    } finally { setBatchBusy(false) }
  }

  async function quickMarkRead(row: MessageRow, e: React.MouseEvent) {
    e.stopPropagation()
    await apiPost('/api/v1/messages/mark-read', { mailbox: row.mailbox, id: row.id })
    setReloadKey((k) => k + 1)
  }
  async function quickArchive(row: MessageRow, e: React.MouseEvent) {
    e.stopPropagation()
    await apiPost('/api/v1/messages/archive', { mailbox: row.mailbox, id: row.id })
    setReloadKey((k) => k + 1)
  }
  async function quickDelete(row: MessageRow, e: React.MouseEvent) {
    e.stopPropagation()
    if (!window.confirm(t('messages.confirmDelete'))) return
    await apiPost('/api/v1/messages/delete', { mailbox: row.mailbox, id: row.id })
    setReloadKey((k) => k + 1)
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Page header */}
      <div className="shrink-0 px-6 pt-5 pb-3">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('projectNav.messages')}</h1>
            <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('inbox.subtitle')}</p>
          </div>
          {projectId != null && projectId !== '' && (
            <CreateMessageDialog projectId={projectId} agents={agents} onSent={() => { setReloadKey((k) => k + 1); setChecked(new Set()) }} />
          )}
        </div>
      </div>

      {/* Filter bar */}
      <div className="shrink-0 border-b border-neutral-200/80 px-6 pb-3 dark:border-zinc-700/50">
        <div className="flex flex-wrap items-center gap-2">
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
          <select value={filters.mailbox} onChange={(e) => setFilter('mailbox', e.target.value)} className={cn(selectCls, 'font-mono')}>
            <option value="">{t('messages.filterMailbox')}: {t('messages.mailboxAll')}</option>
            <option value="human">human</option>
            {agents.map((a) => { const mb = `${projectId}/${a.name}`; return <option key={a.name} value={mb}>{mb}</option> })}
          </select>
          <select value={filters.from} onChange={(e) => setFilter('from', e.target.value)} className={selectCls}>
            <option value="">{t('messages.filterFromPlaceholder')}: {t('messages.readAll')}</option>
            <option value="human">human</option>
            {agents.map((a) => {
              const mb = `${projectId}/${a.name}`
              return <option key={a.name} value={mb}>{mb}</option>
            })}
          </select>
          {hasFilters && (
            <button type="button" onClick={resetFilters} className="flex items-center gap-1 rounded-md px-2 py-1 text-[13px] text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
              <X className="size-3" strokeWidth={2} />
              {t('messages.resetFilters')}
            </button>
          )}
          <button type="button" onClick={() => { setReloadKey((k) => k + 1); setChecked(new Set()) }} className="flex items-center gap-1 rounded-md px-2 py-1 text-[13px] text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-400">
            <RefreshCw className="size-3" strokeWidth={2} />
            {t('api.refresh')}
          </button>
        </div>
      </div>

      {/* Batch action bar */}
      {someChecked && (
        <div className="shrink-0 flex items-center gap-3 border-b border-sky-200 bg-sky-50/60 px-6 py-2 animate-slide-down dark:border-sky-900/40 dark:bg-sky-950/30">
          <span className="text-[13px] font-medium text-sky-800 dark:text-sky-300">
            {t('messages.selected', { count: String(checked.size) })}
          </span>
          <div className="flex items-center gap-1.5">
            <button type="button" disabled={batchBusy} onClick={() => void batchMarkRead()} className="whitespace-nowrap rounded-md border border-sky-200 bg-white px-2.5 py-1 text-[12px] font-medium text-sky-700 transition-colors hover:bg-sky-100 disabled:opacity-40 dark:border-sky-800 dark:bg-sky-900/40 dark:text-sky-300 dark:hover:bg-sky-900/60">
              {t('messages.batchMarkRead')}
            </button>
            <button type="button" disabled={batchBusy} onClick={() => void batchArchive()} className="whitespace-nowrap rounded-md border border-neutral-200 bg-white px-2.5 py-1 text-[12px] font-medium text-neutral-600 transition-colors hover:bg-neutral-50 disabled:opacity-40 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800">
              {t('messages.batchArchive')}
            </button>
            <button type="button" disabled={batchBusy} onClick={() => void batchDelete()} className="whitespace-nowrap rounded-md border border-red-200 bg-white px-2.5 py-1 text-[12px] font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-40 dark:border-red-900 dark:bg-red-950/30 dark:text-red-400 dark:hover:bg-red-900/50">
              {t('messages.batchDelete')}
            </button>
          </div>
          <button type="button" onClick={() => setChecked(new Set())} className="ml-auto text-[12px] text-sky-600 hover:text-sky-800 dark:text-sky-400">
            {t('forms.cancel')}
          </button>
        </div>
      )}

      <MessageDetailModal open={selected != null} message={selected} onClose={() => setSelected(null)} onMutated={() => setReloadKey((k) => k + 1)} />

      {/* Content area */}
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
        {state.status === 'ok' && messages.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <ListFilter className="size-6 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-base font-medium text-neutral-700 dark:text-zinc-300">{t('inbox.emptyTitle')}</p>
            <p className="mt-1 text-sm text-neutral-400 dark:text-zinc-500">{t('api.noMessages')}</p>
          </div>
        )}

        {state.status === 'ok' && messages.length > 0 && (
          <>
            <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
            <table className="min-w-[900px] w-full">
              <thead>
                <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-700/60 dark:bg-zinc-900/40">
                  <th className="w-10 px-3 py-2.5 text-center">
                    <input type="checkbox" checked={allChecked} ref={(el) => { if (el) el.indeterminate = someChecked && !allChecked }} onChange={toggleAll} className="size-3.5 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
                  </th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('api.msgColTime')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('api.msgColFrom')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('api.msgColTo')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('api.msgColPreview')}</th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">{t('messages.colStatus')}</th>
                  <th className="sticky right-0 bg-neutral-50/95 px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 backdrop-blur-sm dark:bg-zinc-900/95 dark:text-zinc-500">
                    {t('messages.actions')}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
                {pagedMessages.map((row) => {
                  const unread = !row.readAt
                  const archived = Boolean(row.archivedAt)
                  const isChecked = checked.has(row.id)
                  return (
                    <tr
                      key={row.id}
                      onClick={() => setSelected(row)}
                      className={cn(
                        'group cursor-pointer transition-colors duration-100',
                        isChecked
                          ? 'bg-sky-50/60 dark:bg-sky-900/[0.10]'
                          : unread
                            ? 'bg-sky-50/30 hover:bg-sky-50/60 dark:bg-sky-900/[0.04] dark:hover:bg-sky-900/[0.10]'
                            : 'bg-white hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30',
                      )}
                    >
                      <td className="w-10 px-3 py-3 text-center" onClick={(e) => e.stopPropagation()}>
                        <input type="checkbox" checked={isChecked} onChange={() => toggleOne(row.id)} className="size-3.5 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600" />
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 align-middle text-[13px] text-neutral-600 dark:text-zinc-400">{fmt(row.sentAt)}</td>
                      <td className="whitespace-nowrap px-4 py-3 align-middle font-mono text-[13px] text-neutral-700 dark:text-zinc-400">{row.from}</td>
                      <td className="whitespace-nowrap px-4 py-3 align-middle font-mono text-[13px] text-neutral-700 dark:text-zinc-400">{row.to}</td>
                      <td className="max-w-md px-4 py-3 align-middle">
                        {row.subject && <span className="text-[13px] font-medium text-neutral-900 dark:text-zinc-200">{row.subject} </span>}
                        <span className="text-[13px] text-neutral-500 dark:text-zinc-500">{preview(row.body)}</span>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 align-middle">
                        <div className="flex items-center gap-1">
                          {unread ? (
                            <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[10.5px] font-semibold text-amber-800 dark:bg-amber-900/30 dark:text-amber-300">
                              <span className="size-1.5 rounded-full bg-amber-500" />
                              {t('messages.badgeUnread')}
                            </span>
                          ) : (
                            <span className="rounded-full bg-emerald-100 px-2 py-0.5 text-[10.5px] font-semibold text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
                              {t('messages.badgeRead')}
                            </span>
                          )}
                          {archived && (
                            <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10.5px] font-semibold text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500">
                              {t('messages.badgeArchived')}
                            </span>
                          )}
                        </div>
                      </td>
                      <td
                        className="sticky right-0 bg-white/95 px-4 py-3 align-middle backdrop-blur-sm group-hover:bg-neutral-50/95 dark:bg-zinc-900/95 dark:group-hover:bg-zinc-800/95"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <div className="flex items-center justify-end gap-2 whitespace-nowrap opacity-0 transition-opacity duration-100 group-hover:opacity-100">
                          {unread && (
                            <button type="button" onClick={(e) => void quickMarkRead(row, e)} className="rounded px-1.5 py-0.5 text-[12px] font-medium text-sky-700 transition-colors hover:bg-sky-100 dark:text-sky-400 dark:hover:bg-sky-900/30">{t('forms.markAsRead')}</button>
                          )}
                          {!archived && (
                            <button type="button" onClick={(e) => void quickArchive(row, e)} className="rounded px-1.5 py-0.5 text-[12px] font-medium text-neutral-500 transition-colors hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800">{t('forms.archiveMessage')}</button>
                          )}
                          <button type="button" onClick={(e) => void quickDelete(row, e)} className="rounded px-1.5 py-0.5 text-[12px] font-medium text-red-600/80 transition-colors hover:bg-red-50 dark:text-red-400/80 dark:hover:bg-red-900/20">{t('messages.delete')}</button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
            </div>
            <Pagination page={msgPage} totalPages={totalMsgPages} onPageChange={setMsgPage} />
          </>
        )}
      </div>
    </div>
  )
}
