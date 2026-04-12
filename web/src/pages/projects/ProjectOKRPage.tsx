import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Plus, ChevronDown, ChevronRight, Trash2, X, TrendingUp, Pencil } from 'lucide-react'
import { apiFetch, apiPost, apiPut, apiDelete } from '../../lib/api'
import { cn } from '../../lib/cn'

type KR = {
  id: string; description: string; metricType: string
  targetValue: number; currentValue: number; unit: string; progress: number; weight: number
}
type OKR = {
  id: string; scope?: string; scopeRef?: string; parentId?: string
  objective: string; description?: string; owner: string; quarter: string
  status: string; keyResults: KR[]; progress: number
  createdAt: string; updatedAt: string
}
type OKRData = { currentQuarter: string; okrs: OKR[] }
type PersonRow = { username: string; displayName: string }

const STATUS_CFG: Record<string, { bg: string; text: string; dot: string }> = {
  on_track:     { bg: 'bg-emerald-50 dark:bg-emerald-900/20', text: 'text-emerald-700 dark:text-emerald-400', dot: 'bg-emerald-500' },
  in_progress:  { bg: 'bg-sky-50 dark:bg-sky-900/20',         text: 'text-sky-700 dark:text-sky-400',         dot: 'bg-sky-500' },
  at_risk:      { bg: 'bg-amber-50 dark:bg-amber-900/20',     text: 'text-amber-700 dark:text-amber-400',     dot: 'bg-amber-500' },
  off_track:    { bg: 'bg-red-50 dark:bg-red-900/20',         text: 'text-red-700 dark:text-red-400',         dot: 'bg-red-500' },
  achieved:     { bg: 'bg-sky-50 dark:bg-sky-900/20',         text: 'text-sky-700 dark:text-sky-400',         dot: 'bg-sky-500' },
}
const STATUS_KEYS = ['on_track', 'in_progress', 'at_risk', 'off_track', 'achieved'] as const

function quarterOptions(): string[] {
  const now = new Date()
  const y = now.getFullYear()
  const curQ = Math.ceil((now.getMonth() + 1) / 3)
  const opts: string[] = []
  for (let offset = -1; offset <= 4; offset++) {
    const q = curQ + offset
    const yr = y + Math.floor((q - 1) / 4)
    const qn = ((q - 1) % 4 + 4) % 4 + 1
    opts.push(`${yr}-Q${qn}`)
  }
  return [...new Set(opts)]
}

function safeProgress(v: number | undefined | null): number {
  if (v == null || isNaN(v)) return 0
  return Math.round(Math.min(Math.max(v, 0), 100))
}

const selectCls = 'block w-full rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-800 [&>option]:dark:text-zinc-200'
const inputCls = 'block w-full rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200'

export default function ProjectOKRPage() {
  const { t } = useTranslation()
  const { projectId } = useParams<{ projectId: string }>()
  const [data, setData] = useState<OKRData | null>(null)
  const [loading, setLoading] = useState(true)
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [showCreate, setShowCreate] = useState(false)
  const [editingKRValue, setEditingKRValue] = useState<{ okrId: string; krId: string; value: string } | null>(null)
  const [editingKRDesc, setEditingKRDesc] = useState<{ okrId: string; krId: string; value: string } | null>(null)
  const [editingOKR, setEditingOKR] = useState<OKR | null>(null)
  const [people, setPeople] = useState<PersonRow[]>([])

  const load = useCallback(async () => {
    if (!projectId) return
    try {
      const [d, p] = await Promise.all([
        apiFetch<OKRData>(`/api/v1/okrs?scope=project&scopeRef=${encodeURIComponent(projectId)}`),
        apiFetch<PersonRow[]>('/api/v1/users').catch(() => [] as PersonRow[]),
      ])
      setData(d)
      setPeople(p)
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [projectId])

  useEffect(() => { void load() }, [load])

  if (!projectId) return null

  function toggle(id: string) {
    setExpanded(prev => { const n = new Set(prev); n.has(id) ? n.delete(id) : n.add(id); return n })
  }

  async function deleteOKR(id: string) {
    if (!confirm(t('okr.confirmDelete'))) return
    await apiDelete(`/api/v1/okrs/${id}`); void load()
  }
  async function deleteKR(okrId: string, krId: string) {
    if (!confirm(t('okr.confirmDeleteKR'))) return
    await apiDelete(`/api/v1/okrs/${okrId}/key-results/${krId}`); void load()
  }
  async function saveKRValue(okrId: string, krId: string, val: string) {
    const num = parseFloat(val); if (isNaN(num)) return
    await apiPut(`/api/v1/okrs/${okrId}/key-results/${krId}`, { currentValue: num })
    setEditingKRValue(null); void load()
  }
  async function saveKRDesc(okrId: string, krId: string, desc: string) {
    if (!desc.trim()) return
    await apiPut(`/api/v1/okrs/${okrId}/key-results/${krId}`, { description: desc })
    setEditingKRDesc(null); void load()
  }

  if (loading) return <div className="flex justify-center py-20 text-neutral-400">{t('api.loading')}</div>

  const okrs = data?.okrs ?? []
  const avgProgress = okrs.length > 0 ? okrs.reduce((s, o) => s + safeProgress(o.progress), 0) / okrs.length : 0

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-5 pb-3">
        <div className="flex items-center justify-between gap-4">
          <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('okr.title')}</h1>
          <button onClick={() => setShowCreate(true)}
            className="rounded-lg border border-sky-600 bg-white px-3 py-2 text-sm font-medium text-sky-700 hover:bg-sky-50 dark:border-sky-500 dark:bg-zinc-900 dark:text-sky-400 dark:hover:bg-zinc-800">
            {t('okr.newOKR')}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-6 pb-6">
        <div className="space-y-4">
          {okrs.length > 0 && (
            <div className="rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-zinc-700/60 dark:bg-zinc-900/40">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium text-neutral-700 dark:text-zinc-300">{t('okr.overallProgress')}</span>
                <span className="font-bold text-neutral-900 dark:text-zinc-100">{safeProgress(avgProgress)}%</span>
              </div>
              <div className="mt-2 h-2.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-zinc-800">
                <div className="h-full rounded-full bg-gradient-to-r from-sky-500 to-emerald-500 transition-all" style={{ width: `${safeProgress(avgProgress)}%` }} />
              </div>
            </div>
          )}
          {okrs.length === 0 && !showCreate && (
            <div className="rounded-xl border border-dashed border-neutral-300 py-16 text-center dark:border-zinc-700">
              <p className="text-sm text-neutral-400 dark:text-zinc-500">{t('okr.empty')}</p>
            </div>
          )}
          {okrs.map(okr => {
            const isOpen = expanded.has(okr.id)
            const sc = STATUS_CFG[okr.status] ?? STATUS_CFG.on_track
            const prog = safeProgress(okr.progress)
            return (
              <div key={okr.id} className="rounded-xl border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
                <div className="flex cursor-pointer items-center gap-3 px-5 py-4" onClick={() => toggle(okr.id)}>
                  {isOpen ? <ChevronDown className="size-4 shrink-0 text-neutral-400" /> : <ChevronRight className="size-4 shrink-0 text-neutral-400" />}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <h3 className="truncate text-sm font-semibold text-neutral-900 dark:text-zinc-100">{okr.objective}</h3>
                      <span className={cn('inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium', sc.bg, sc.text)}>
                        <span className={cn('size-1.5 rounded-full', sc.dot)} />
                        {t(`okr.status_${okr.status}`)}
                      </span>
                    </div>
                    <div className="mt-1.5 flex items-center gap-3">
                      <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-zinc-800">
                        <div className="h-full rounded-full bg-gradient-to-r from-sky-500 to-emerald-500 transition-all" style={{ width: `${prog}%` }} />
                      </div>
                      <span className="shrink-0 text-xs font-medium text-neutral-500 dark:text-zinc-400">{prog}%</span>
                    </div>
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <button onClick={e => { e.stopPropagation(); setEditingOKR(okr) }} className="rounded-md p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"><Pencil className="size-3.5" /></button>
                    <button onClick={e => { e.stopPropagation(); deleteOKR(okr.id) }} className="rounded-md p-1.5 text-neutral-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20"><Trash2 className="size-3.5" /></button>
                  </div>
                </div>
                {isOpen && (
                  <div className="border-t border-neutral-100 px-5 py-3 dark:border-zinc-800/40">
                    {okr.description && <p className="mb-3 text-xs leading-relaxed text-neutral-500 dark:text-zinc-400">{okr.description}</p>}
                    <table className="w-full">
                      <thead>
                        <tr className="text-left text-[11px] font-medium uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
                          <th className="pb-2 pr-3">{t('okr.krDescription')}</th>
                          <th className="w-32 pb-2 pr-3">{t('okr.overallProgress')}</th>
                          <th className="w-44 pb-2 pr-3 text-right">{t('okr.targetValue')}</th>
                          <th className="w-8 pb-2" />
                        </tr>
                      </thead>
                      <tbody className="text-xs">
                        {okr.keyResults.map(kr => {
                          const krProg = safeProgress(kr.progress)
                          const isEditingDesc = editingKRDesc?.okrId === okr.id && editingKRDesc.krId === kr.id
                          return (
                            <tr key={kr.id} className="group border-t border-neutral-50 dark:border-zinc-800/30">
                              <td className="py-2.5 pr-3">
                                <div className="flex items-center gap-2">
                                  <TrendingUp className="size-3 shrink-0 text-neutral-300 dark:text-zinc-600" />
                                  {isEditingDesc ? (
                                    <input autoFocus value={editingKRDesc.value}
                                      onChange={e => setEditingKRDesc({ ...editingKRDesc, value: e.target.value })}
                                      onBlur={() => saveKRDesc(okr.id, kr.id, editingKRDesc.value)}
                                      onKeyDown={e => { if (e.key === 'Enter') saveKRDesc(okr.id, kr.id, editingKRDesc.value); if (e.key === 'Escape') setEditingKRDesc(null) }}
                                      className="w-full rounded border border-sky-400 bg-white px-2 py-0.5 text-xs dark:bg-zinc-800 dark:text-zinc-200" />
                                  ) : (
                                    <span className="cursor-pointer rounded px-1 py-0.5 font-medium text-neutral-700 hover:bg-neutral-100 dark:text-zinc-300 dark:hover:bg-zinc-800"
                                      onClick={() => setEditingKRDesc({ okrId: okr.id, krId: kr.id, value: kr.description })}>{kr.description}</span>
                                  )}
                                </div>
                              </td>
                              <td className="py-2.5 pr-3">
                                <div className="flex items-center gap-2">
                                  <div className="h-1 flex-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-zinc-800">
                                    <div className={cn('h-full rounded-full transition-all', krProg >= 70 ? 'bg-emerald-500' : krProg >= 40 ? 'bg-amber-500' : 'bg-red-400')} style={{ width: `${krProg}%` }} />
                                  </div>
                                  <span className="w-8 text-right text-neutral-500 dark:text-zinc-400">{krProg}%</span>
                                </div>
                              </td>
                              <td className="py-2.5 pr-3 text-right">
                                {editingKRValue?.okrId === okr.id && editingKRValue.krId === kr.id ? (
                                  <input autoFocus type="number" value={editingKRValue.value}
                                    onChange={e => setEditingKRValue({ ...editingKRValue, value: e.target.value })}
                                    onBlur={() => saveKRValue(okr.id, kr.id, editingKRValue.value)}
                                    onKeyDown={e => { if (e.key === 'Enter') saveKRValue(okr.id, kr.id, editingKRValue.value); if (e.key === 'Escape') setEditingKRValue(null) }}
                                    className="w-20 rounded border border-sky-400 bg-white px-2 py-0.5 text-right text-xs dark:bg-zinc-800 dark:text-zinc-200" />
                                ) : (
                                  <button onClick={() => setEditingKRValue({ okrId: okr.id, krId: kr.id, value: String(kr.currentValue ?? 0) })}
                                    className="whitespace-nowrap rounded px-1.5 py-0.5 font-mono text-neutral-600 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800">
                                    {kr.currentValue ?? 0}{kr.unit ? ` ${kr.unit}` : ''} / {kr.targetValue ?? 0}{kr.unit ? ` ${kr.unit}` : ''}
                                  </button>
                                )}
                              </td>
                              <td className="py-2.5">
                                <button onClick={() => deleteKR(okr.id, kr.id)} className="rounded p-1 text-neutral-300 opacity-0 transition-opacity hover:bg-red-50 hover:text-red-500 group-hover:opacity-100 dark:text-zinc-600 dark:hover:bg-red-900/20"><Trash2 className="size-3" /></button>
                              </td>
                            </tr>
                          )
                        })}
                      </tbody>
                    </table>
                    {okr.keyResults.length === 0 && <p className="py-3 text-center text-xs text-neutral-400 dark:text-zinc-500">{t('okr.noKR')}</p>}
                    <AddKRInline okrId={okr.id} onCreated={load} />
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {showCreate && <CreateProjectOKR people={people} projectId={projectId} allOKRs={okrs} onClose={() => setShowCreate(false)} onSaved={() => { setShowCreate(false); void load() }} quarter={data?.currentQuarter || ''} />}
      {editingOKR && <EditProjectOKR people={people} projectId={projectId} allOKRs={okrs} okr={editingOKR} onClose={() => setEditingOKR(null)} onSaved={() => { setEditingOKR(null); void load() }} quarter={data?.currentQuarter || ''} />}
    </div>
  )
}

function AddKRInline({ okrId, onCreated }: { okrId: string; onCreated: () => void }) {
  const { t } = useTranslation()
  const [show, setShow] = useState(false)
  const [desc, setDesc] = useState(''); const [target, setTarget] = useState('100')
  const [unit, setUnit] = useState(''); const [metric, setMetric] = useState('number')
  const [busy, setBusy] = useState(false)
  async function submit() {
    if (!desc.trim()) return; setBusy(true)
    try {
      await apiPost(`/api/v1/okrs/${okrId}/key-results`, { description: desc, targetValue: parseFloat(target) || 100, currentValue: 0, unit, metricType: metric })
      setDesc(''); setTarget('100'); setUnit(''); setMetric('number'); setShow(false); onCreated()
    } finally { setBusy(false) }
  }
  if (!show) return (
    <div className="pt-2">
      <button onClick={() => setShow(true)} className="flex items-center gap-1 text-xs font-medium text-sky-600 hover:text-sky-700 dark:text-sky-400">
        <Plus className="size-3" /> {t('okr.addKR')}
      </button>
    </div>
  )
  return (
    <div className="mt-3 rounded-lg border border-neutral-200/80 bg-neutral-50/50 p-4 dark:border-zinc-700/60 dark:bg-zinc-800/30">
      <div className="space-y-3">
        <div>
          <label className="mb-1 block text-[11px] font-medium text-neutral-500 dark:text-zinc-500">{t('okr.krDescription')}</label>
          <input value={desc} onChange={e => setDesc(e.target.value)} placeholder={t('okr.krDescription')} className={inputCls} />
        </div>
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="mb-1 block text-[11px] font-medium text-neutral-500 dark:text-zinc-500">{t('forms.type')}</label>
            <select value={metric} onChange={e => setMetric(e.target.value)} className={selectCls}>
              <option value="number">{t('okr.metricNumber')}</option><option value="percentage">{t('okr.metricPercentage')}</option>
              <option value="boolean">{t('okr.metricBoolean')}</option><option value="currency">{t('okr.metricCurrency')}</option>
            </select>
          </div>
          <div>
            <label className="mb-1 block text-[11px] font-medium text-neutral-500 dark:text-zinc-500">{t('okr.targetValue')}</label>
            <input value={target} onChange={e => setTarget(e.target.value)} type="number" className={inputCls} />
          </div>
          <div>
            <label className="mb-1 block text-[11px] font-medium text-neutral-500 dark:text-zinc-500">{t('okr.unit')}</label>
            <input value={unit} onChange={e => setUnit(e.target.value)} placeholder="ms, %, ..." className={inputCls} />
          </div>
        </div>
      </div>
      <div className="mt-4 flex items-center justify-end gap-2">
        <button onClick={() => setShow(false)} className="rounded-md px-3 py-1.5 text-xs font-medium text-neutral-500 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800">{t('forms.cancel')}</button>
        <button onClick={() => void submit()} disabled={busy || !desc.trim()} className="rounded-md bg-sky-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-sky-700 disabled:opacity-50">{t('forms.submit')}</button>
      </div>
    </div>
  )
}

function CreateProjectOKR({ people, projectId, allOKRs, onClose, onSaved, quarter }: { people: PersonRow[]; projectId: string; allOKRs: OKR[]; onClose: () => void; onSaved: () => void; quarter: string }) {
  return <OKRModal people={people} projectId={projectId} allOKRs={allOKRs} onClose={onClose} onSaved={onSaved} quarter={quarter} />
}
function EditProjectOKR({ people, projectId, allOKRs, okr, onClose, onSaved, quarter }: { people: PersonRow[]; projectId: string; allOKRs: OKR[]; okr: OKR; onClose: () => void; onSaved: () => void; quarter: string }) {
  return <OKRModal people={people} projectId={projectId} allOKRs={allOKRs} okr={okr} onClose={onClose} onSaved={onSaved} quarter={quarter} />
}

function OKRModal({ people, projectId, allOKRs, okr, onClose, onSaved, quarter }: {
  people: PersonRow[]; projectId: string; allOKRs: OKR[]; okr?: OKR; onClose: () => void; onSaved: () => void; quarter: string
}) {
  const { t } = useTranslation()
  const isEdit = !!okr
  const [objective, setObjective] = useState(okr?.objective ?? '')
  const [description, setDescription] = useState(okr?.description ?? '')
  const [owner, setOwner] = useState(okr?.owner ?? 'human')
  const [parentId, setParentId] = useState(okr?.parentId ?? '')
  const [status, setStatus] = useState(okr?.status ?? 'on_track')
  const defaultQ = quarter || `${new Date().getFullYear()}-Q${Math.ceil((new Date().getMonth() + 1) / 3)}`
  const [q, setQ] = useState(okr?.quarter ?? defaultQ)
  const [busy, setBusy] = useState(false)
  const quarters = quarterOptions()
  if (!quarters.includes(q)) quarters.unshift(q)
  const parentCandidates = allOKRs.filter(o => o.id !== okr?.id)

  async function submit() {
    if (!objective.trim()) return; setBusy(true)
    try {
      const body: Record<string, unknown> = { objective, description, owner, quarter: q, status, scope: 'project', scopeRef: projectId, parentId: parentId || undefined }
      if (isEdit) await apiPut(`/api/v1/okrs/${okr.id}`, body)
      else await apiPost('/api/v1/okrs', body)
      onSaved()
    } finally { setBusy(false) }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div className="w-full max-w-md rounded-xl border border-neutral-200 bg-white p-6 shadow-xl dark:border-zinc-700 dark:bg-zinc-900" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between pb-4">
          <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{isEdit ? t('okr.editOKR') : t('okr.newOKR')}</h3>
          <button onClick={onClose} className="text-neutral-400 hover:text-neutral-600 dark:hover:text-zinc-300"><X className="size-4" /></button>
        </div>
        <div className="space-y-3">
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('okr.objective')}</span>
            <input value={objective} onChange={e => setObjective(e.target.value)} className={inputCls} />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('okr.descriptionLabel')}</span>
            <textarea value={description} onChange={e => setDescription(e.target.value)} rows={2} className={inputCls} placeholder={t('okr.descriptionHint')} />
          </label>
          {parentCandidates.length > 0 && (
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('okr.parentOKR')}</span>
              <select value={parentId} onChange={e => setParentId(e.target.value)} className={selectCls}>
                <option value="">{t('okr.noParent')}</option>
                {parentCandidates.map(o => <option key={o.id} value={o.id}>{o.objective}</option>)}
              </select>
            </label>
          )}
          <div className="flex gap-3">
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('okr.quarter')}</span>
              <select value={q} onChange={e => setQ(e.target.value)} className={selectCls}>
                {quarters.map(qv => <option key={qv} value={qv}>{qv}</option>)}
              </select>
            </label>
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('okr.owner')}</span>
              <select value={owner} onChange={e => setOwner(e.target.value)} className={selectCls}>
                <option value="human">human</option>
                {people.map(p => <option key={p.username} value={p.username}>{p.displayName || p.username}</option>)}
              </select>
            </label>
          </div>
          {isEdit && (
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('forms.status')}</span>
              <select value={status} onChange={e => setStatus(e.target.value)} className={selectCls}>
                {STATUS_KEYS.map(k => <option key={k} value={k}>{t(`okr.status_${k}`)}</option>)}
              </select>
            </label>
          )}
        </div>
        <div className="flex justify-end gap-2 pt-5">
          <button onClick={onClose} className="rounded-md px-4 py-2 text-sm text-neutral-500 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800">{t('forms.cancel')}</button>
          <button onClick={() => void submit()} disabled={busy || !objective.trim()} className="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50">{t('forms.submit')}</button>
        </div>
      </div>
    </div>
  )
}
