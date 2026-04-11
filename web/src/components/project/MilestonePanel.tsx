import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Milestone as MsIcon, Trash2, X, Calendar, User, CheckCircle2, Tag, Pencil } from 'lucide-react'
import { apiFetch, apiPost, apiPut, apiDelete } from '../../lib/api'
import { cn } from '../../lib/cn'

type Milestone = {
  id: string; title: string; description: string; status: string
  dueDate: string | null; owner: string; progress: number
  criteria: string[]; linkedKR: string[]; taskLabels: string[]
  createdAt: string; updatedAt: string
}
type AgentEntry = { name: string }

const STATUS_CFG: Record<string, { bg: string; text: string }> = {
  planned:     { bg: 'bg-neutral-100 dark:bg-zinc-800', text: 'text-neutral-600 dark:text-zinc-400' },
  in_progress: { bg: 'bg-sky-50 dark:bg-sky-900/20',    text: 'text-sky-700 dark:text-sky-400' },
  completed:   { bg: 'bg-emerald-50 dark:bg-emerald-900/20', text: 'text-emerald-700 dark:text-emerald-400' },
  cancelled:   { bg: 'bg-red-50 dark:bg-red-900/20',    text: 'text-red-700 dark:text-red-400' },
}

const selectCls = 'block w-full rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-800 [&>option]:dark:text-zinc-200'
const inputCls = 'block w-full rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200'

function useLocaleDate() {
  const { i18n } = useTranslation()
  const locale = i18n.language ?? 'en'
  return useCallback((dateStr: string | null) => {
    if (!dateStr) return null
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return null
    return new Intl.DateTimeFormat(locale, { year: 'numeric', month: '2-digit', day: '2-digit' }).format(d)
  }, [locale])
}

export function MilestonePanel({ project }: { project: string }) {
  const { t } = useTranslation()
  const fmtDate = useLocaleDate()
  const [milestones, setMilestones] = useState<Milestone[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [editingMs, setEditingMs] = useState<Milestone | null>(null)
  const [editProgress, setEditProgress] = useState<{ id: string; value: string } | null>(null)
  const [agents, setAgents] = useState<AgentEntry[]>([])

  const load = useCallback(async () => {
    try {
      const [res, ag] = await Promise.all([
        apiFetch<{ milestones: Milestone[] }>(`/api/v1/projects/${encodeURIComponent(project)}/milestones`),
        apiFetch<AgentEntry[]>(`/api/v1/projects/${encodeURIComponent(project)}/agents`).catch(() => [] as AgentEntry[]),
      ])
      setMilestones(res.milestones ?? [])
      setAgents(ag)
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [project])

  useEffect(() => { void load() }, [load])

  async function deleteMilestone(id: string) {
    if (!confirm(t('milestone.confirmDelete'))) return
    await apiDelete(`/api/v1/projects/${encodeURIComponent(project)}/milestones/${id}`)
    void load()
  }

  async function saveProgress(id: string, val: string) {
    const num = parseInt(val, 10)
    if (isNaN(num) || num < 0 || num > 100) return
    await apiPut(`/api/v1/projects/${encodeURIComponent(project)}/milestones/${id}`, { progress: num })
    setEditProgress(null)
    void load()
  }

  async function updateStatus(id: string, status: string) {
    await apiPut(`/api/v1/projects/${encodeURIComponent(project)}/milestones/${id}`, { status })
    void load()
  }

  function dueDateLabel(dateStr: string | null) {
    if (!dateStr) return null
    const due = new Date(dateStr)
    const now = new Date()
    const diff = Math.ceil((due.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
    if (diff < 0) return { text: t('milestone.overdue', { days: Math.abs(diff) }), cls: 'text-red-500' }
    if (diff <= 7) return { text: t('milestone.dueSoon', { days: diff }), cls: 'text-amber-500' }
    return { text: fmtDate(dateStr) ?? '', cls: 'text-neutral-400 dark:text-zinc-500' }
  }

  if (loading) return <div className="flex justify-center py-10 text-neutral-400">{t('api.loading')}</div>

  return (
    <>
      <div className="shrink-0 px-6 pt-5 pb-3">
        <div className="flex items-center justify-between gap-4">
          <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-200">{t('milestone.title')}</h1>
          <button onClick={() => setShowCreate(true)}
            className="rounded-lg border border-sky-600 bg-white px-3 py-2 text-sm font-medium text-sky-700 hover:bg-sky-50 dark:border-sky-500 dark:bg-zinc-900 dark:text-sky-400 dark:hover:bg-zinc-800">
            {t('milestone.create')}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-6 pb-6">
        {milestones.length === 0 && !showCreate && (
          <div className="rounded-xl border border-dashed border-neutral-300 py-16 text-center dark:border-zinc-700">
            <MsIcon className="mx-auto mb-2 size-6 text-neutral-300 dark:text-zinc-600" strokeWidth={1.5} />
            <p className="text-sm text-neutral-400 dark:text-zinc-500">{t('milestone.empty')}</p>
          </div>
        )}

        <div className="space-y-3">
          {milestones.map(ms => {
            const sc = STATUS_CFG[ms.status] ?? STATUS_CFG.planned
            const due = dueDateLabel(ms.dueDate)
            return (
              <div key={ms.id} className="rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-zinc-700/60 dark:bg-zinc-900/40">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <h4 className="truncate text-sm font-semibold text-neutral-900 dark:text-zinc-100">{ms.title}</h4>
                      <select value={ms.status} onChange={e => updateStatus(ms.id, e.target.value)}
                        className={cn('rounded-full border-0 px-2 py-0.5 text-xs font-medium', sc.bg, sc.text)}>
                        <option value="planned">{t('milestone.statusPlanned')}</option>
                        <option value="in_progress">{t('milestone.statusInProgress')}</option>
                        <option value="completed">{t('milestone.statusCompleted')}</option>
                        <option value="cancelled">{t('milestone.statusCancelled')}</option>
                      </select>
                    </div>
                    {ms.description && <p className="mt-1 text-xs text-neutral-500 dark:text-zinc-500">{ms.description}</p>}
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <button onClick={() => setEditingMs(ms)}
                      className="rounded-md p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300">
                      <Pencil className="size-3.5" />
                    </button>
                    <button onClick={() => deleteMilestone(ms.id)}
                      className="rounded-md p-1.5 text-neutral-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20">
                      <Trash2 className="size-3.5" />
                    </button>
                  </div>
                </div>

                <div className="mt-3 flex items-center gap-2">
                  <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-zinc-800">
                    <div className={cn('h-full rounded-full transition-all',
                      ms.progress >= 70 ? 'bg-emerald-500' : ms.progress >= 40 ? 'bg-amber-500' : 'bg-sky-500')}
                      style={{ width: `${ms.progress}%` }} />
                  </div>
                  {editProgress?.id === ms.id ? (
                    <input autoFocus type="number" min={0} max={100} value={editProgress.value}
                      onChange={e => setEditProgress({ id: ms.id, value: e.target.value })}
                      onBlur={() => saveProgress(ms.id, editProgress.value)}
                      onKeyDown={e => { if (e.key === 'Enter') saveProgress(ms.id, editProgress.value); if (e.key === 'Escape') setEditProgress(null) }}
                      className="w-14 rounded border border-sky-400 bg-white px-1 py-0.5 text-center text-xs dark:bg-zinc-800 dark:text-zinc-200" />
                  ) : (
                    <button onClick={() => setEditProgress({ id: ms.id, value: String(ms.progress) })}
                      className="shrink-0 rounded px-1 py-0.5 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800">
                      {ms.progress}%
                    </button>
                  )}
                </div>

                <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-neutral-400 dark:text-zinc-500">
                  {ms.owner && <span className="flex items-center gap-1"><User className="size-3" />{ms.owner}</span>}
                  {due && <span className={cn('flex items-center gap-1', due.cls)}><Calendar className="size-3" />{due.text}</span>}
                </div>

                {ms.criteria && ms.criteria.length > 0 && (
                  <div className="mt-3 space-y-1">
                    {ms.criteria.map((c, i) => (
                      <div key={i} className="flex items-start gap-1.5 text-xs text-neutral-600 dark:text-zinc-400">
                        <CheckCircle2 className="mt-0.5 size-3 shrink-0 text-neutral-300 dark:text-zinc-600" />
                        {c}
                      </div>
                    ))}
                  </div>
                )}

                {ms.taskLabels && ms.taskLabels.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {ms.taskLabels.map(l => (
                      <span key={l} className="inline-flex items-center gap-0.5 rounded-full bg-neutral-100 px-2 py-0.5 text-xs text-neutral-600 dark:bg-zinc-800 dark:text-zinc-400">
                        <Tag className="size-2.5" />{l}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {showCreate && <MilestoneFormModal project={project} agents={agents}
        onClose={() => setShowCreate(false)} onSaved={() => { setShowCreate(false); void load() }} />}
      {editingMs && <MilestoneFormModal project={project} agents={agents} ms={editingMs}
        onClose={() => setEditingMs(null)} onSaved={() => { setEditingMs(null); void load() }} />}
    </>
  )
}

function toDateInputValue(dateStr: string | null | undefined): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return ''
  return d.toISOString().slice(0, 10)
}

function MilestoneFormModal({ project, agents, ms, onClose, onSaved }: {
  project: string; agents: AgentEntry[]; ms?: Milestone; onClose: () => void; onSaved: () => void
}) {
  const { t } = useTranslation()
  const isEdit = !!ms
  const [title, setTitle] = useState(ms?.title ?? '')
  const [description, setDescription] = useState(ms?.description ?? '')
  const [dueDate, setDueDate] = useState(toDateInputValue(ms?.dueDate))
  const [owner, setOwner] = useState(ms?.owner ?? 'human')
  const [status, setStatus] = useState(ms?.status ?? 'planned')
  const [criteria, setCriteria] = useState(ms?.criteria?.join('\n') ?? '')
  const [labels, setLabels] = useState(ms?.taskLabels?.join(', ') ?? '')
  const [busy, setBusy] = useState(false)

  async function submit() {
    if (!title.trim()) return
    setBusy(true)
    try {
      const body: Record<string, unknown> = {
        title, description,
        dueDate: dueDate ? new Date(dueDate).toISOString() : undefined,
        owner: owner || undefined,
        status,
        criteria: criteria.split('\n').map(s => s.trim()).filter(Boolean),
        taskLabels: labels.split(',').map(s => s.trim()).filter(Boolean),
      }
      if (isEdit) {
        await apiPut(`/api/v1/projects/${encodeURIComponent(project)}/milestones/${ms.id}`, body)
      } else {
        await apiPost(`/api/v1/projects/${encodeURIComponent(project)}/milestones`, body)
      }
      onSaved()
    } finally { setBusy(false) }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div className="w-full max-w-md rounded-xl border border-neutral-200 bg-white p-6 shadow-xl dark:border-zinc-700 dark:bg-zinc-900" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between pb-4">
          <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
            {isEdit ? t('milestone.edit') : t('milestone.create')}
          </h3>
          <button onClick={onClose} className="text-neutral-400 hover:text-neutral-600 dark:hover:text-zinc-300"><X className="size-4" /></button>
        </div>
        <div className="space-y-3">
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('milestone.titleLabel')}</span>
            <input value={title} onChange={e => setTitle(e.target.value)} className={inputCls} />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('milestone.descriptionLabel')}</span>
            <textarea value={description} onChange={e => setDescription(e.target.value)} rows={2} className={inputCls} />
          </label>
          <div className="flex gap-3">
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('milestone.dueDate')}</span>
              <input type="date" value={dueDate} onChange={e => setDueDate(e.target.value)} className={inputCls + ' dark:[color-scheme:dark]'} />
            </label>
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('milestone.ownerLabel')}</span>
              <select value={owner} onChange={e => setOwner(e.target.value)} className={selectCls}>
                <option value="human">human</option>
                {agents.map(a => <option key={a.name} value={`${project}/${a.name}`}>{a.name}</option>)}
              </select>
            </label>
          </div>
          {isEdit && (
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('forms.status')}</span>
              <select value={status} onChange={e => setStatus(e.target.value)} className={selectCls}>
                <option value="planned">{t('milestone.statusPlanned')}</option>
                <option value="in_progress">{t('milestone.statusInProgress')}</option>
                <option value="completed">{t('milestone.statusCompleted')}</option>
                <option value="cancelled">{t('milestone.statusCancelled')}</option>
              </select>
            </label>
          )}
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('milestone.criteriaLabel')}</span>
            <textarea value={criteria} onChange={e => setCriteria(e.target.value)} rows={3} placeholder={t('milestone.criteriaHint')} className={inputCls} />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('milestone.labelsLabel')}</span>
            <input value={labels} onChange={e => setLabels(e.target.value)} placeholder="v1.0, release-blocker" className={inputCls} />
          </label>
        </div>
        <div className="flex justify-end gap-2 pt-5">
          <button onClick={onClose} className="rounded-md px-4 py-2 text-sm text-neutral-500 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800">{t('forms.cancel')}</button>
          <button onClick={() => void submit()} disabled={busy || !title.trim()}
            className="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50">{t('forms.submit')}</button>
        </div>
      </div>
    </div>
  )
}
