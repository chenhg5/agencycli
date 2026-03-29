import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  CalendarClock, Heart, Pause, Pencil, Play, Plus, Power,
  Trash2, X, Zap,
} from 'lucide-react'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { cn } from '../../lib/cn'
import { apiFetch, apiDelete, apiPatch, apiPost } from '../../lib/api'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'

/* ─── types ─── */

type SchedInstance = { key: string; running: boolean; pid?: number; startedAt?: string; error?: string }
type SchedStatusResp = { schedulers: SchedInstance[] }

type HeartbeatRow = {
  enabled: boolean; interval: string; paused: boolean
  activeHours?: string; activeDays?: string; wakeupPrompt?: string; wakeupCondition?: string
  maxTasksPerCycle?: number; maxCycleDuration?: string; pid?: number
  lastWakeup?: string; lastWakeupStatus?: string; lastCycleDuration?: string
  wakeupCount?: number; wakeupCountToday?: number; nextWakeupAt?: string
  schedulerStartedAt?: string; lastConditionStatus?: string
  sessionScope?: string; sessionId?: string; sessionStartedAt?: string
}

type CronRow = {
  id: string; title: string; schedule: string; enabled: boolean; prompt: string
  lastRun?: string; lastRunStatus?: string; runCount?: number
}

type AgentSchedule = { name: string; heartbeat: HeartbeatRow; crons: CronRow[] }
type ScheduleResp = { project: string; agents: AgentSchedule[] }

/* ─── shared styles ─── */

const tabCls = 'cursor-pointer px-4 py-3 text-sm font-medium transition-colors border-b-2'
const tabActive = 'border-sky-600 text-sky-700 dark:border-sky-400 dark:text-sky-300'
const tabInactive = 'border-transparent text-neutral-500 hover:text-neutral-700 dark:text-zinc-500 dark:hover:text-zinc-300'
const thCls = 'px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600'
const tdCls = 'whitespace-nowrap px-4 py-3 align-middle text-[13px] text-neutral-700 dark:text-zinc-300'
const selectCls = 'h-8 rounded-md border border-neutral-200/80 bg-white px-2.5 pr-7 text-[13px] text-neutral-700 outline-none hover:border-neutral-300 focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-900 dark:text-zinc-300'
const fieldCls = 'w-full rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-sm text-neutral-900 outline-none focus:border-sky-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100'
const smallBtn = 'rounded p-1 transition-colors'

type Tab = 'heartbeat' | 'cron' | 'runtime'

export default function ProjectSchedulePage() {
  const { t } = useTranslation()
  const { projectId } = useParams<{ projectId: string }>()
  const path = projectId ? `/api/v1/projects/${encodeURIComponent(projectId)}/schedule` : null
  const [reloadKey, setReloadKey] = useState(0)
  const [tab, setTab] = useState<Tab>('heartbeat')
  const state = useApiJson<ScheduleResp>(path, reloadKey)
  const reload = useCallback(() => setReloadKey((k) => k + 1), [])
  const agents = state.status === 'ok' ? state.data.agents : []

  useEffect(() => {
    if (tab !== 'runtime') return
    const iv = setInterval(() => setReloadKey((k) => k + 1), 5000)
    return () => clearInterval(iv)
  }, [tab])

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Header + scheduler control */}
      <div className="shrink-0 px-6 pt-5 pb-3">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('projectNav.schedule')}</h1>
            <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('schedule.subtitle')}</p>
          </div>
          {projectId && <SchedulerControl projectId={projectId} onAction={reload} />}
        </div>
      </div>

      {/* Tabs */}
      <div className="shrink-0 border-b border-neutral-200/80 px-6 dark:border-zinc-800/50">
        <div className="flex gap-0">
          {(['heartbeat', 'cron', 'runtime'] as Tab[]).map((key) => (
            <button key={key} type="button" onClick={() => setTab(key)} className={cn(tabCls, tab === key ? tabActive : tabInactive)}>
              {key === 'heartbeat' && <Heart className="mr-1.5 inline size-3.5" strokeWidth={1.8} />}
              {key === 'cron' && <CalendarClock className="mr-1.5 inline size-3.5" strokeWidth={1.8} />}
              {key === 'runtime' && <Zap className="mr-1.5 inline size-3.5" strokeWidth={1.8} />}
              {t(`schedule.tab${key.charAt(0).toUpperCase() + key.slice(1)}`)}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-6 py-4">
        {state.status === 'loading' && <Spinner label={t('api.loading')} />}
        {state.status === 'error' && <PlaceholderCard title={t('api.loadError')}><p>{state.error.message}</p></PlaceholderCard>}
        {state.status === 'ok' && tab === 'heartbeat' && <HeartbeatTab agents={agents} projectId={projectId!} onChanged={reload} />}
        {state.status === 'ok' && tab === 'cron' && <CronTab agents={agents} projectId={projectId!} onChanged={reload} />}
        {state.status === 'ok' && tab === 'runtime' && <RuntimeTab agents={agents} projectId={projectId!} />}
      </div>
    </div>
  )
}

function Spinner({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 py-16 justify-center">
      <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
      <span className="text-sm text-neutral-500">{label}</span>
    </div>
  )
}

/* ═══════════════════════════════════════════════════════════════
   Scheduler control (top-right start/stop)
   ═══════════════════════════════════════════════════════════════ */

function SchedulerControl({ projectId, onAction }: { projectId: string; onAction?: () => void }) {
  const { t } = useTranslation()
  const [status, setStatus] = useState<SchedStatusResp | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const mountedRef = useRef(true)

  const fetchStatus = useCallback(async () => {
    try {
      const data = await apiFetch<SchedStatusResp>('/api/v1/scheduler/status')
      if (mountedRef.current) setStatus(data)
    } catch { /* swallow poll errors */ }
  }, [])

  useEffect(() => {
    mountedRef.current = true
    void fetchStatus()
    const iv = setInterval(fetchStatus, 5000)
    return () => { mountedRef.current = false; clearInterval(iv) }
  }, [fetchStatus])

  const instances = status?.schedulers ?? []
  const inst = instances.find((s) => s.running && (s.key === projectId || s.key === 'all'))
  const isRunning = Boolean(inst)

  async function toggle() {
    setError(null); setBusy(true)
    try {
      if (isRunning) {
        const body: Record<string, string> = {}
        const key = inst?.key ?? projectId
        if (key !== 'all') { const p = key.split('/'); body.project = p[0]; if (p[1]) body.agent = p[1] }
        await apiPost('/api/v1/scheduler/stop', body)
      } else {
        await apiPost('/api/v1/scheduler/start', { project: projectId })
      }
      await fetchStatus()
      onAction?.()
    } catch (e) { setError(e instanceof Error ? e.message : String(e)) }
    finally { setBusy(false) }
  }

  return (
    <div className="flex items-center gap-2.5">
      {isRunning && (
        <span className="flex items-center gap-1.5 rounded-full bg-emerald-100 px-2.5 py-0.5 text-[11px] font-semibold text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
          <span className="size-1.5 rounded-full bg-emerald-500 animate-pulse" />
          {t('schedule.schedulerRunning')}
        </span>
      )}
      <button type="button" disabled={busy} onClick={() => void toggle()} className={cn(
        'cursor-pointer rounded-lg border px-3 py-2 text-sm font-medium transition-colors disabled:opacity-40',
        isRunning
          ? 'border-red-300 bg-white text-red-600 hover:bg-red-50 dark:border-red-700 dark:bg-zinc-900 dark:text-red-400 dark:hover:bg-zinc-800'
          : 'border-sky-600 bg-white text-sky-700 hover:bg-sky-50 dark:border-sky-500 dark:bg-zinc-900 dark:text-sky-400 dark:hover:bg-zinc-800',
      )}>
        {isRunning ? t('schedule.schedulerStop') : t('schedule.schedulerStart')}
      </button>
      {error && <p className="text-[11px] text-red-500">{error}</p>}
    </div>
  )
}

/* ═══════════════════════════════════════════════════════════════
   Heartbeat tab
   ═══════════════════════════════════════════════════════════════ */

function HeartbeatTab({ agents, projectId, onChanged }: { agents: AgentSchedule[]; projectId: string; onChanged: () => void }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const [editing, setEditing] = useState<{ name: string; hb: HeartbeatRow } | null>(null)

  async function toggle(agent: string, enabled: boolean) {
    await apiPatch(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agent)}/heartbeat`, { enabled })
    onChanged()
  }
  async function togglePause(agent: string, paused: boolean) {
    await apiPost(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agent)}/heartbeat/${paused ? 'pause' : 'resume'}`, {})
    onChanged()
  }

  return (
    <>
      <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-800/60">
        <table className="min-w-[800px] w-full">
          <thead>
            <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-800/60 dark:bg-zinc-900/40">
              <th className={thCls}>Agent</th>
              <th className={thCls}>{t('schedule.statusLabel')}</th>
              <th className={thCls}>{t('schedule.interval')}</th>
              <th className={thCls}>{t('schedule.activeHours')}</th>
              <th className={thCls}>{t('schedule.lastWakeup')}</th>
              <th className={thCls}>{t('schedule.wakeupCountLabel')}</th>
              <th className={cn(thCls, 'text-right')}>{t('messages.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
            {agents.map((ag) => {
              const hb = ag.heartbeat
              return (
                <tr key={ag.name} className="group bg-white transition-colors hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30">
                  <td className={cn(tdCls, 'font-mono font-medium')}>{ag.name}</td>
                  <td className={tdCls}>
                    {hb.enabled && !hb.paused && <StatusBadge color="emerald">{t('schedule.hbActive')}</StatusBadge>}
                    {hb.enabled && hb.paused && <StatusBadge color="amber">{t('schedule.paused')}</StatusBadge>}
                    {!hb.enabled && <StatusBadge color="neutral">{t('schedule.off')}</StatusBadge>}
                  </td>
                  <td className={cn(tdCls, 'font-mono')}>{hb.interval || '—'}</td>
                  <td className={tdCls}>{hb.activeHours || '—'}</td>
                  <td className={tdCls}>{hb.lastWakeup ? fmt(hb.lastWakeup) : '—'}</td>
                  <td className={cn(tdCls, 'tabular-nums')}>
                    {hb.wakeupCount ?? 0}
                    {(hb.wakeupCountToday ?? 0) > 0 && <span className="ml-1 text-neutral-400 dark:text-zinc-600">({hb.wakeupCountToday} {t('schedule.today')})</span>}
                  </td>
                  <td className={cn(tdCls, 'text-right')}>
                    <div className="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                      <button type="button" onClick={() => setEditing({ name: ag.name, hb })} className={cn(smallBtn, 'text-neutral-500 hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-800')} title={t('tasks.edit')}>
                        <Pencil className="size-3.5" strokeWidth={1.8} />
                      </button>
                      {hb.enabled && (
                        <button type="button" onClick={() => void togglePause(ag.name, !hb.paused)} className={cn(smallBtn, 'text-neutral-500 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800')} title={hb.paused ? t('schedule.resumeHb') : t('schedule.pauseHb')}>
                          {hb.paused ? <Play className="size-3.5" strokeWidth={1.8} /> : <Pause className="size-3.5" strokeWidth={1.8} />}
                        </button>
                      )}
                      <button type="button" onClick={() => void toggle(ag.name, !hb.enabled)} className={cn(smallBtn, hb.enabled ? 'text-amber-600 hover:bg-amber-50 dark:text-amber-400' : 'text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400')} title={hb.enabled ? t('schedule.disableHb') : t('schedule.enableHb')}>
                        <Power className="size-3.5" strokeWidth={1.8} />
                      </button>
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <p className="mt-3 text-xs text-neutral-400 dark:text-zinc-600">{t('schedule.restartHint')}</p>

      {editing && (
        <EditHeartbeatModal projectId={projectId} agentName={editing.name} hb={editing.hb} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); onChanged() }} />
      )}
    </>
  )
}

function EditHeartbeatModal({ projectId, agentName, hb, onClose, onSaved }: { projectId: string; agentName: string; hb: HeartbeatRow; onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()
  const [interval, setInterval] = useState(hb.interval ?? '')
  const [activeHours, setActiveHours] = useState(hb.activeHours ?? '')
  const [activeDays, setActiveDays] = useState(hb.activeDays ?? '')
  const [maxTasks, setMaxTasks] = useState(hb.maxTasksPerCycle ?? 0)
  const [maxDur, setMaxDur] = useState(hb.maxCycleDuration ?? '')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  async function save() {
    setErr(null); setBusy(true)
    try {
      await apiPatch(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/heartbeat`, {
        interval: interval.trim() || undefined,
        activeHours: activeHours.trim(),
        activeDays: activeDays.trim(),
        maxTasksPerCycle: maxTasks,
        maxCycleDuration: maxDur.trim(),
      })
      onSaved()
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)) }
    finally { setBusy(false) }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" onClick={() => !busy && onClose()}>
      <div className="w-full max-w-md rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
          <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{t('schedule.editHb')} — <span className="font-mono">{agentName}</span></h2>
          <button type="button" onClick={onClose} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 dark:text-zinc-600 dark:hover:bg-zinc-800"><X className="size-4" /></button>
        </div>
        <div className="space-y-3 px-5 py-4">
          <Field label={t('schedule.interval')}><input value={interval} onChange={(e) => setInterval(e.target.value)} placeholder="30m, 1h" className={fieldCls} /></Field>
          <Field label={t('schedule.activeHours')}><input value={activeHours} onChange={(e) => setActiveHours(e.target.value)} placeholder="09:00-18:00" className={fieldCls} /></Field>
          <Field label={t('schedule.activeDaysLabel')}><input value={activeDays} onChange={(e) => setActiveDays(e.target.value)} placeholder="Mon,Tue,Wed,Thu,Fri" className={fieldCls} /></Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label={t('schedule.maxTasks')}><input type="number" min={0} value={maxTasks} onChange={(e) => setMaxTasks(Number(e.target.value))} className={fieldCls} /></Field>
            <Field label={t('schedule.maxDuration')}><input value={maxDur} onChange={(e) => setMaxDur(e.target.value)} placeholder="15m, 1h" className={fieldCls} /></Field>
          </div>
          {err && <p className="text-sm text-red-600 dark:text-red-400">{err}</p>}
          <div className="flex justify-end gap-2 pt-1">
            <button type="button" onClick={onClose} disabled={busy} className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600">{t('forms.cancel')}</button>
            <button type="button" onClick={() => void save()} disabled={busy} className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">{busy ? t('forms.saving') : t('forms.save')}</button>
          </div>
        </div>
      </div>
    </div>
  )
}

/* ═══════════════════════════════════════════════════════════════
   Cron tab
   ═══════════════════════════════════════════════════════════════ */

function CronTab({ agents, projectId, onChanged }: { agents: AgentSchedule[]; projectId: string; onChanged: () => void }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const [adding, setAdding] = useState<string | null>(null)

  const allCrons = useMemo(() => {
    const rows: { agent: string; cron: CronRow }[] = []
    for (const ag of agents) for (const c of ag.crons) rows.push({ agent: ag.name, cron: c })
    return rows
  }, [agents])

  const agentOptions = agents.map((a) => a.name)

  async function toggleCron(agent: string, cronId: string, enabled: boolean) {
    const base = `/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agent)}/crons/${encodeURIComponent(cronId)}`
    await apiPost(`${base}/${enabled ? 'resume' : 'pause'}`, {})
    onChanged()
  }
  async function deleteCron(agent: string, cronId: string) {
    if (!window.confirm(t('schedule.confirmDeleteCron'))) return
    await apiDelete(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agent)}/crons/${encodeURIComponent(cronId)}`)
    onChanged()
  }

  return (
    <>
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm font-medium text-neutral-700 dark:text-zinc-300">
          {allCrons.length} {t('schedule.cronJobs')}
        </span>
        {!adding && agentOptions.length > 0 && (
          <button type="button" onClick={() => setAdding(agentOptions[0])} className="inline-flex items-center gap-1 rounded-lg border border-sky-600 bg-white px-3 py-2 text-sm font-medium text-sky-700 hover:bg-sky-50 dark:border-sky-500 dark:bg-zinc-900 dark:text-sky-400 dark:hover:bg-zinc-800">
            <Plus className="size-3.5" strokeWidth={2} />{t('schedule.addCron')}
          </button>
        )}
      </div>

      {adding && (
        <AddCronForm projectId={projectId} agents={agentOptions} defaultAgent={adding} onClose={() => setAdding(null)} onCreated={() => { setAdding(null); onChanged() }} />
      )}

      {allCrons.length === 0 && !adding && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <CalendarClock className="mb-3 size-8 text-neutral-300 dark:text-zinc-700" strokeWidth={1.5} />
          <p className="text-sm text-neutral-500 dark:text-zinc-500">{t('schedule.noCrons')}</p>
        </div>
      )}

      {allCrons.length > 0 && (
        <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-800/60">
          <table className="min-w-[750px] w-full">
            <thead>
              <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-800/60 dark:bg-zinc-900/40">
                <th className={thCls}>Agent</th>
                <th className={thCls}>{t('forms.title')}</th>
                <th className={thCls}>{t('schedule.cronSchedule')}</th>
                <th className={thCls}>{t('schedule.statusLabel')}</th>
                <th className={thCls}>{t('schedule.lastRun')}</th>
                <th className={thCls}>{t('schedule.runCountLabel')}</th>
                <th className={cn(thCls, 'text-right')}>{t('messages.actions')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
              {allCrons.map(({ agent, cron: c }) => (
                <tr key={`${agent}-${c.id}`} className="group bg-white transition-colors hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30">
                  <td className={cn(tdCls, 'font-mono font-medium')}>{agent}</td>
                  <td className={tdCls}>{c.title}</td>
                  <td className={cn(tdCls, 'font-mono')}>{c.schedule}</td>
                  <td className={tdCls}>
                    {c.enabled ? <StatusBadge color="emerald">{t('schedule.cronOn')}</StatusBadge> : <StatusBadge color="neutral">{t('schedule.cronOff')}</StatusBadge>}
                  </td>
                  <td className={tdCls}>{c.lastRun ? fmt(c.lastRun) : '—'}</td>
                  <td className={cn(tdCls, 'tabular-nums')}>{c.runCount ?? 0}</td>
                  <td className={cn(tdCls, 'text-right')}>
                    <div className="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                      <button type="button" onClick={() => void toggleCron(agent, c.id, !c.enabled)} className={cn(smallBtn, 'text-neutral-500 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800')} title={c.enabled ? t('schedule.pauseCron') : t('schedule.resumeCron')}>
                        {c.enabled ? <Pause className="size-3.5" strokeWidth={1.8} /> : <Play className="size-3.5" strokeWidth={1.8} />}
                      </button>
                      <button type="button" onClick={() => void deleteCron(agent, c.id)} className={cn(smallBtn, 'text-red-500 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30')} title={t('schedule.deleteCron')}>
                        <Trash2 className="size-3.5" strokeWidth={1.8} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="mt-3 text-xs text-neutral-400 dark:text-zinc-600">{t('schedule.restartHint')}</p>
    </>
  )
}

function AddCronForm({ projectId, agents, defaultAgent, onClose, onCreated }: { projectId: string; agents: string[]; defaultAgent: string; onClose: () => void; onCreated: () => void }) {
  const { t } = useTranslation()
  const [agent, setAgent] = useState(defaultAgent)
  const [title, setTitle] = useState('')
  const [schedule, setSchedule] = useState('')
  const [prompt, setPrompt] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  async function create() {
    setErr(null); setBusy(true)
    try {
      await apiPost(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agent)}/crons`, { title: title.trim(), schedule: schedule.trim(), prompt: prompt.trim() })
      onCreated()
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)) }
    finally { setBusy(false) }
  }

  return (
    <div className="mb-4 rounded-lg border border-dashed border-sky-300 bg-sky-50/30 p-4 animate-fade-in dark:border-sky-800 dark:bg-sky-900/10">
      <div className="grid gap-3 sm:grid-cols-3">
        <div>
          <span className="mb-1 block text-xs font-medium text-neutral-500 dark:text-zinc-500">Agent</span>
          <select value={agent} onChange={(e) => setAgent(e.target.value)} className={selectCls + ' w-full'}>
            {agents.map((a) => <option key={a} value={a}>{a}</option>)}
          </select>
        </div>
        <div>
          <span className="mb-1 block text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('forms.title')}</span>
          <input value={title} onChange={(e) => setTitle(e.target.value)} className={fieldCls} placeholder="Daily standup" />
        </div>
        <div>
          <span className="mb-1 block text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('schedule.cronSchedule')}</span>
          <input value={schedule} onChange={(e) => setSchedule(e.target.value)} className={cn(fieldCls, 'font-mono')} placeholder="0 9 * * 1-5" />
        </div>
      </div>
      <div className="mt-3">
        <span className="mb-1 block text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('forms.prompt')}</span>
        <textarea value={prompt} onChange={(e) => setPrompt(e.target.value)} rows={2} className={fieldCls} />
      </div>
      {err && <p className="mt-2 text-sm text-red-600 dark:text-red-400">{err}</p>}
      <div className="mt-3 flex gap-2">
        <button type="button" disabled={busy || !title.trim() || !schedule.trim() || !prompt.trim()} onClick={() => void create()} className="rounded-lg bg-sky-600 px-4 py-1.5 text-xs font-medium text-white hover:bg-sky-700 disabled:opacity-40">{busy ? t('forms.saving') : t('schedule.createCron')}</button>
        <button type="button" onClick={onClose} className="rounded-lg px-4 py-1.5 text-xs font-medium text-neutral-500 hover:text-neutral-700 dark:text-zinc-500">{t('forms.cancel')}</button>
      </div>
    </div>
  )
}

/* ═══════════════════════════════════════════════════════════════
   Runtime tab
   ═══════════════════════════════════════════════════════════════ */

function RuntimeTab({ agents, projectId }: { agents: AgentSchedule[]; projectId: string }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const [waking, setWaking] = useState<string | null>(null)
  const [resetting, setResetting] = useState<string | null>(null)
  const [wakeErr, setWakeErr] = useState<string | null>(null)
  const [scopeUpdating, setScopeUpdating] = useState<string | null>(null)

  const activeAgents = agents.filter((ag) => ag.heartbeat.enabled)

  async function doWakeup(agentName: string) {
    setWaking(agentName); setWakeErr(null)
    try {
      await apiPost('/api/v1/scheduler/wakeup', { project: projectId, agent: agentName })
    } catch (e) { setWakeErr(e instanceof Error ? e.message : String(e)) }
    finally { setWaking(null) }
  }

  async function doSessionReset(agentName: string) {
    setResetting(agentName)
    try {
      await apiPost('/api/v1/session/reset', { project: projectId, agent: agentName })
    } catch (e) { setWakeErr(e instanceof Error ? e.message : String(e)) }
    finally { setResetting(null) }
  }

  async function doScopeChange(agentName: string, scope: string) {
    setScopeUpdating(agentName)
    try {
      await apiPatch(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/heartbeat`, { sessionScope: scope })
    } catch (e) { setWakeErr(e instanceof Error ? e.message : String(e)) }
    finally { setScopeUpdating(null) }
  }

  if (activeAgents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <Zap className="mb-3 size-8 text-neutral-300 dark:text-zinc-700" strokeWidth={1.5} />
        <p className="text-sm text-neutral-500 dark:text-zinc-500">{t('schedule.noActiveAgents')}</p>
      </div>
    )
  }

  return (
    <>
      {wakeErr && <p className="mb-3 text-sm text-red-600 dark:text-red-400">{wakeErr}</p>}
      <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-800/60">
        <table className="min-w-[1100px] w-full">
          <thead>
            <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-800/60 dark:bg-zinc-900/40">
              <th className={thCls}>Agent</th>
              <th className={thCls}>{t('schedule.statusLabel')}</th>
              <th className={thCls}>{t('schedule.nextWakeup')}</th>
              <th className={thCls}>{t('schedule.lastWakeup')}</th>
              <th className={thCls}>{t('schedule.lastDuration')}</th>
              <th className={thCls}>{t('schedule.wakeupCountLabel')}</th>
              <th className={thCls}>{t('schedule.today')}</th>
              <th className={thCls}>{t('session.sessionLabel')}</th>
              <th className={thCls}>{t('session.scopeLabel')}</th>
              <th className={thCls}>{t('schedule.conditionLabel')}</th>
              <th className={cn(thCls, 'text-center')}>{t('messages.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
            {activeAgents.map((ag) => {
              const hb = ag.heartbeat
              const isRunningNow = hb.lastWakeupStatus === 'running'
              const hasSession = !!hb.sessionId
              return (
                <tr key={ag.name} className="group bg-white transition-colors hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30">
                  <td className={cn(tdCls, 'font-mono font-medium')}>{ag.name}</td>
                  <td className={tdCls}>
                    {isRunningNow && <StatusBadge color="sky">{t('schedule.running')}</StatusBadge>}
                    {!isRunningNow && hb.paused && <StatusBadge color="amber">{t('schedule.paused')}</StatusBadge>}
                    {!isRunningNow && !hb.paused && hb.lastWakeupStatus === 'done' && <StatusBadge color="emerald">{t('schedule.idle')}</StatusBadge>}
                    {!isRunningNow && !hb.paused && hb.lastWakeupStatus === 'failed' && <StatusBadge color="red">{t('schedule.failed')}</StatusBadge>}
                    {!isRunningNow && !hb.paused && !hb.lastWakeupStatus && <StatusBadge color="neutral">{t('schedule.waiting')}</StatusBadge>}
                  </td>
                  <td className={tdCls}>
                    {hb.nextWakeupAt ? (
                      <span className="font-mono text-sky-700 dark:text-sky-400">{fmt(hb.nextWakeupAt)}</span>
                    ) : isRunningNow ? (
                      <span className="text-neutral-400 dark:text-zinc-600">—</span>
                    ) : '—'}
                  </td>
                  <td className={tdCls}>{hb.lastWakeup ? fmt(hb.lastWakeup) : '—'}</td>
                  <td className={cn(tdCls, 'font-mono tabular-nums')}>{hb.lastCycleDuration || '—'}</td>
                  <td className={cn(tdCls, 'tabular-nums font-semibold')}>{hb.wakeupCount ?? 0}</td>
                  <td className={cn(tdCls, 'tabular-nums')}>{hb.wakeupCountToday ?? 0}</td>
                  <td className={tdCls}>
                    {hasSession ? (
                      <div className="flex flex-col gap-0.5">
                        <span className="font-mono text-xs text-emerald-700 dark:text-emerald-400" title={hb.sessionId}>{hb.sessionId!.slice(0, 12)}…</span>
                        {hb.sessionStartedAt && <span className="text-[11px] text-neutral-400 dark:text-zinc-600">{fmt(hb.sessionStartedAt)}</span>}
                      </div>
                    ) : (
                      <span className="text-xs text-neutral-400 dark:text-zinc-600">{t('session.noSession')}</span>
                    )}
                  </td>
                  <td className={tdCls}>
                    <select
                      value={hb.sessionScope || 'cycle'}
                      onChange={(e) => void doScopeChange(ag.name, e.target.value)}
                      disabled={scopeUpdating === ag.name}
                      className="h-7 cursor-pointer rounded border border-neutral-200 bg-white px-1.5 text-xs outline-none hover:border-neutral-300 focus:border-sky-400 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300"
                    >
                      <option value="cycle">{t('session.scopeCycle')}</option>
                      <option value="task">{t('session.scopeTask')}</option>
                    </select>
                  </td>
                  <td className={tdCls}>
                    {hb.wakeupCondition ? (
                      hb.lastConditionStatus === 'met' ? <StatusBadge color="emerald">Met</StatusBadge>
                      : hb.lastConditionStatus === 'not_met' ? <StatusBadge color="amber">Not met</StatusBadge>
                      : <StatusBadge color="neutral">—</StatusBadge>
                    ) : <span className="text-neutral-400 dark:text-zinc-600">—</span>}
                  </td>
                  <td className={cn(tdCls, 'text-center')}>
                    <div className="flex items-center justify-center gap-1">
                      <button type="button" disabled={isRunningNow || waking === ag.name} onClick={() => void doWakeup(ag.name)}
                        className="cursor-pointer rounded-md px-2 py-1 text-xs font-medium text-sky-700 opacity-0 transition-all hover:bg-sky-50 disabled:opacity-40 group-hover:opacity-100 dark:text-sky-400 dark:hover:bg-sky-900/20">
                        {waking === ag.name ? t('schedule.wakingUp') : t('schedule.wakeupNow')}
                      </button>
                      {hasSession && (
                        <button type="button" disabled={resetting === ag.name} onClick={() => void doSessionReset(ag.name)}
                          className="cursor-pointer rounded-md px-2 py-1 text-xs font-medium text-amber-600 opacity-0 transition-all hover:bg-amber-50 disabled:opacity-40 group-hover:opacity-100 dark:text-amber-400 dark:hover:bg-amber-900/20">
                          {resetting === ag.name ? t('session.resettingSession') : t('session.resetSession')}
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </>
  )
}

/* ─── shared UI ─── */

function StatusBadge({ color, children }: { color: 'emerald' | 'amber' | 'neutral' | 'sky' | 'red'; children: React.ReactNode }) {
  const cls: Record<string, string> = {
    emerald: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
    amber: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
    neutral: 'bg-neutral-100 text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500',
    sky: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400',
    red: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  }
  return <span className={cn('inline-block rounded-full px-2 py-0.5 text-[11px] font-semibold', cls[color])}>{children}</span>
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <label className="block text-sm"><span className="mb-1 block text-neutral-600 dark:text-zinc-400">{label}</span>{children}</label>
}
