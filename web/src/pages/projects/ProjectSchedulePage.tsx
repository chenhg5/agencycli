import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  CalendarClock, ClipboardCopy, Heart, Pause, Pencil, Play, Plus, Power,
  MessageSquareText, Trash2, X, Zap,
} from 'lucide-react'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { ConversationLog } from '../../components/ui/ConversationLog'
import { cn } from '../../lib/cn'
import { apiFetch, apiDelete, apiPatch, apiPost, apiPut } from '../../lib/api'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'

/* ─── types ─── */

type SchedInstance = { key: string; running: boolean; pid?: number; startedAt?: string; error?: string }
type SchedStatusResp = { schedulers: SchedInstance[] }

type HeartbeatRow = {
  enabled: boolean; interval: string; paused: boolean
  activeHours?: string; activeDays?: string; jitter?: string
  wakeupPrompt?: string; wakeupCondition?: string; wakeupPreset?: string
  maxTasksPerCycle?: number; maxCycleDuration?: string; pid?: number
  lastWakeup?: string; lastWakeupStatus?: string; lastCycleDuration?: string
  wakeupCount?: number; wakeupCountToday?: number; nextWakeupAt?: string
  schedulerStartedAt?: string; lastConditionStatus?: string
  sessionScope?: string; sessionId?: string; sessionStartedAt?: string
  triggers?: string[]
}

type CronRow = {
  id: string; title: string; schedule: string; enabled: boolean; prompt: string
  lastRun?: string; lastRunStatus?: string; runCount?: number; nextRun?: string
  sessionScope?: string; sessionId?: string; sessionStartedAt?: string
  jitter?: string
}

type SessionUsage = {
  lastInputTokens: number; totalInputTokens: number; totalOutputTokens: number
  totalCacheRead: number; totalCostUsd: number; runCount: number; contextLimit: number
}

type AgentSchedule = { name: string; heartbeat: HeartbeatRow; crons: CronRow[]; model?: string; agentDir?: string; sessionUsage?: SessionUsage }
type ScheduleResp = { project: string; agents: AgentSchedule[] }

/* ─── shared styles ─── */

const tabCls = 'cursor-pointer px-4 py-3 text-sm font-medium transition-colors border-b-2'
const tabActive = 'border-sky-600 text-sky-700 dark:border-sky-400 dark:text-sky-300'
const tabInactive = 'border-transparent text-neutral-500 hover:text-neutral-700 dark:text-zinc-500 dark:hover:text-zinc-300'
const thCls = 'whitespace-nowrap px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500'
const tdCls = 'whitespace-nowrap px-4 py-3 align-middle text-center text-[13px] text-neutral-700 dark:text-zinc-300'
const thSticky = 'sticky right-0 z-[2] bg-neutral-50 dark:bg-zinc-900'
const tdSticky = 'sticky right-0 z-[1] bg-white group-hover:bg-neutral-50 dark:bg-zinc-900 dark:group-hover:bg-zinc-800'
const selectCls = 'h-8 rounded-md border border-neutral-200/80 bg-white px-2.5 pr-7 text-[13px] text-neutral-700 outline-none hover:border-neutral-300 focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-900 dark:text-zinc-300 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-900 [&>option]:dark:text-zinc-300'
const fieldCls = 'w-full rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-sm text-neutral-900 outline-none focus:border-sky-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-800 [&>option]:dark:text-zinc-100'
const smallBtn = 'rounded p-1 transition-colors'

type Tab = 'runtime' | 'heartbeat' | 'cron'

export default function ProjectSchedulePage() {
  const { t } = useTranslation()
  const { projectId } = useParams<{ projectId: string }>()
  const path = projectId ? `/api/v1/projects/${encodeURIComponent(projectId)}/schedule` : null
  const [reloadKey, setReloadKey] = useState(0)
  const [tab, setTab] = useState<Tab>('runtime')
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
      <div className="shrink-0 border-b border-neutral-200/80 px-6 dark:border-zinc-700/50">
        <div className="flex gap-0">
          {(['runtime', 'heartbeat', 'cron'] as Tab[]).map((key) => (
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
    setBusy(true)
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
    } catch { /* toast handled by apiFetch */ }
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
      <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
        <table className="min-w-[1040px] w-full">
          <thead>
            <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-700/60 dark:bg-zinc-900/40">
              <th className={thCls}>Agent</th>
              <th className={thCls}>{t('schedule.statusLabel')}</th>
              <th className={thCls}>{t('schedule.interval')}</th>
              <th className={thCls}>{t('schedule.activeHours')}</th>
              <th className={thCls}>{t('schedule.wakeupCondition')}</th>
              <th className={thCls}>{t('schedule.triggers')}</th>
              <th className={thCls}>{t('schedule.lastWakeup')}</th>
              <th className={thCls}>{t('schedule.wakeupCountLabel')}</th>
              <th className={cn(thCls, thSticky)}>{t('messages.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
            {agents.map((ag) => {
              const hb = ag.heartbeat
              return (
                <tr key={ag.name} className="group bg-white transition-colors hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30">
                  <td className={cn(tdCls, 'font-mono font-medium')}><Link to={`/projects/${projectId}/members/${ag.name}`} className="text-sky-700 hover:underline dark:text-sky-400">{ag.name}</Link></td>
                  <td className={tdCls}>
                    {heartbeatStatusBadge(hb, t)}
                  </td>
                  <td className={cn(tdCls, 'font-mono')}>{hb.interval || '—'}</td>
                  <td className={tdCls}>{hb.activeHours || '—'}</td>
                  <td className={tdCls}>
                    <WakeupConditionSummary hb={hb} />
                  </td>
                  <td className={tdCls}>
                    <TriggerSummary triggers={hb.triggers} />
                  </td>
                  <td className={tdCls}>
                    {hb.lastWakeup ? (
                      <Link to={`/projects/${projectId}/runs?agent=${encodeURIComponent(projectId + '/' + ag.name)}`} className="text-neutral-700 transition-colors hover:text-sky-600 hover:underline dark:text-zinc-300 dark:hover:text-sky-400">
                        {fmt(hb.lastWakeup)}
                      </Link>
                    ) : '—'}
                  </td>
                  <td className={cn(tdCls, 'tabular-nums')}>
                    {hb.wakeupCount ?? 0}
                    {(hb.wakeupCountToday ?? 0) > 0 && <span className="ml-1 text-neutral-400 dark:text-zinc-500">({hb.wakeupCountToday} {t('schedule.today')})</span>}
                  </td>
                  <td className={cn(tdCls, tdSticky)}>
                    <div className="flex items-center justify-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
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
      <p className="mt-3 text-xs text-neutral-400 dark:text-zinc-500">{t('schedule.configEffectHint')}</p>

      {editing && (
        <EditHeartbeatModal projectId={projectId} agentName={editing.name} hb={editing.hb} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); onChanged() }} />
      )}
    </>
  )
}

function EditHeartbeatModal({ projectId, agentName, hb, onClose, onSaved }: { projectId: string; agentName: string; hb: HeartbeatRow; onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()

  // Interval: parse "30m" → {num:30, unit:"m"}, "2h" → {num:2, unit:"h"}
  const parseInterval = (v: string) => {
    const m = v.match(/^(\d+(?:\.\d+)?)\s*(m|h|s)/)
    if (m) return { num: parseFloat(m[1]), unit: m[2] }
    return { num: 30, unit: 'm' }
  }
  const iv = parseInterval(hb.interval ?? '30m')
  const [ivNum, setIvNum] = useState(iv.num)
  const [ivUnit, setIvUnit] = useState<'m' | 'h'>(iv.unit === 'h' ? 'h' : 'm')

  // Jitter
  const jt = parseInterval(hb.jitter ?? '0m')
  const [jtNum, setJtNum] = useState(jt.num)
  const [jtUnit, setJtUnit] = useState<'m' | 'h'>(jt.unit === 'h' ? 'h' : 'm')

  // Active Hours: parse "09:00-18:00" (split on '-' would break HH:MM-HH:MM)
  const parseAH = (v: string) => {
    const m = v.trim().match(/^(\d{1,2}:\d{2})\s*-\s*(\d{1,2}:\d{2})$/)
    if (!m) return { start: '', end: '' }
    const pad = (x: string) => {
      const p = x.match(/^(\d{1,2}):(\d{2})$/)
      return p ? `${p[1].padStart(2, '0')}:${p[2]}` : x
    }
    return { start: pad(m[1]), end: pad(m[2]) }
  }
  const ah = parseAH(hb.activeHours ?? '')
  const [ahStart, setAhStart] = useState(ah.start)
  const [ahEnd, setAhEnd] = useState(ah.end)

  // Active Days: parse "Mon,Wed,Fri" or "weekdays"
  const ALL_DAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] as const
  const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri']
  const WEEKENDS = ['Sat', 'Sun']
  const parseDays = (v: string): string[] => {
    if (!v) return []
    const lower = v.toLowerCase().trim()
    if (lower === 'weekdays') return [...WEEKDAYS]
    if (lower === 'weekends') return [...WEEKENDS]
    return v.split(',').map((d) => d.trim()).filter((d) => (ALL_DAYS as readonly string[]).includes(d))
  }
  const [days, setDays] = useState<string[]>(parseDays(hb.activeDays ?? ''))

  const toggleDay = (d: string) => setDays((prev) => (prev.includes(d) ? prev.filter((x) => x !== d) : [...prev, d]))
  const setWeekdays = () => setDays([...WEEKDAYS])
  const setWeekends = () => setDays([...WEEKENDS])
  const setAllDays = () => setDays([...ALL_DAYS])
  const clearDays = () => setDays([])

  // Max Tasks
  const [maxTasks, setMaxTasks] = useState(hb.maxTasksPerCycle ?? 0)

  // Max Duration
  const parseDur = (v: string) => {
    const m = v.match(/^(\d+(?:\.\d+)?)\s*(m|h|s)/)
    if (m) return { num: parseFloat(m[1]), unit: m[2] }
    return { num: 0, unit: 'm' }
  }
  const md = parseDur(hb.maxCycleDuration ?? '')
  const [mdNum, setMdNum] = useState(md.num)
  const [mdUnit, setMdUnit] = useState<'m' | 'h'>(md.unit === 'h' ? 'h' : 'm')

  // Session scope & ID
  const [sessionScope, setSessionScope] = useState(hb.sessionScope || 'cycle')
  const [sessionId, setSessionId] = useState(hb.sessionId ?? '')

  // Wakeup preset
  const [wakeupPreset, setWakeupPreset] = useState(hb.wakeupPreset ?? '')
  const [wakeupCondition, setWakeupCondition] = useState(hb.wakeupCondition ?? '')
  const [useScriptCondition, setUseScriptCondition] = useState(Boolean(hb.wakeupCondition))

  // Triggers
  const TRIGGER_TYPES = ['message', 'task'] as const
  const [triggers, setTriggers] = useState<string[]>(hb.triggers ?? [])
  const toggleTrigger = (t: string) => setTriggers(prev => prev.includes(t) ? prev.filter(x => x !== t) : [...prev, t])

  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const buildDaysString = () => {
    const sorted = ALL_DAYS.filter((d) => days.includes(d))
    if (sorted.length === 0) return ''
    if (sorted.length === 5 && WEEKDAYS.every((d) => sorted.includes(d as (typeof ALL_DAYS)[number])) && !sorted.includes('Sat') && !sorted.includes('Sun')) return 'weekdays'
    if (sorted.length === 2 && sorted.includes('Sat') && sorted.includes('Sun')) return 'weekends'
    return sorted.join(',')
  }

  async function save() {
    setErr(null); setBusy(true)
    try {
      if (useScriptCondition && !wakeupCondition.trim()) {
        setErr(t('schedule.scriptConditionRequired'))
        return
      }
      const intervalStr = ivNum > 0 ? `${ivNum}${ivUnit}` : ''
      const jitterStr = jtNum > 0 ? `${jtNum}${jtUnit}` : ''
      const activeHoursStr = ahStart && ahEnd ? `${ahStart}-${ahEnd}` : ''
      const maxDurStr = mdNum > 0 ? `${mdNum}${mdUnit}` : ''
      await apiPatch(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/heartbeat`, {
        interval: intervalStr || undefined,
        jitter: jitterStr,
        activeHours: activeHoursStr,
        activeDays: buildDaysString(),
        maxTasksPerCycle: maxTasks,
        maxCycleDuration: maxDurStr,
        wakeupPreset: wakeupPreset,
        wakeupCondition: useScriptCondition ? wakeupCondition.trim() : '',
        triggers: triggers,
        sessionScope: sessionScope,
        sessionId: sessionId !== (hb.sessionId ?? '') ? sessionId : undefined,
      })
      onSaved()
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)) }
    finally { setBusy(false) }
  }

  const chipCls = 'cursor-pointer rounded-md border px-2 py-1 text-xs font-medium transition-colors'
  const chipActive = 'border-sky-500 bg-sky-50 text-sky-700 dark:border-sky-600 dark:bg-sky-900/30 dark:text-sky-300'
  const chipInactive = 'border-neutral-200 bg-white text-neutral-500 hover:border-neutral-300 hover:text-neutral-700 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-500 dark:hover:border-zinc-600'
  const shortcutCls = 'cursor-pointer text-[11px] text-sky-600 hover:text-sky-800 dark:text-sky-400 dark:hover:text-sky-300'

  const presetHasTasks = wakeupPreset === 'require_tasks' || wakeupPreset === 'require_any'
  const presetHasMessages = wakeupPreset === 'require_messages' || wakeupPreset === 'require_any'
  const setPresetConditions = (next: { tasks: boolean; messages: boolean }) => {
    if (next.tasks && next.messages) setWakeupPreset('require_any')
    else if (next.tasks) setWakeupPreset('require_tasks')
    else if (next.messages) setWakeupPreset('require_messages')
    else setWakeupPreset('')
  }
  const toggleScriptCondition = () => {
    setUseScriptCondition((prev) => {
      if (prev) setWakeupCondition('')
      return !prev
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" onClick={() => !busy && onClose()}>
      <div className="flex max-h-[90vh] w-full max-w-lg flex-col rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in" onClick={(e) => e.stopPropagation()}>
        <div className="flex shrink-0 items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
          <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{t('schedule.editHb')} — <span className="font-mono">{agentName}</span></h2>
          <button type="button" onClick={onClose} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800"><X className="size-4" /></button>
        </div>
        <div className="flex-1 space-y-4 overflow-y-auto px-5 py-4">
          {/* Interval */}
          <Field label={t('schedule.interval')}>
            <div className="flex items-center gap-2">
              <input type="number" min={1} value={ivNum} onChange={(e) => setIvNum(Number(e.target.value))} className={cn(fieldCls, 'w-24 tabular-nums')} />
              <select value={ivUnit} onChange={(e) => setIvUnit(e.target.value as 'm' | 'h')} className={cn(fieldCls, 'w-28')}>
                <option value="m">{t('schedule.unitMinutes')}</option>
                <option value="h">{t('schedule.unitHours')}</option>
              </select>
            </div>
          </Field>

          {/* Jitter */}
          <Field label={t('schedule.jitter')}>
            <div className="flex items-center gap-2">
              <input type="number" min={0} value={jtNum} onChange={(e) => setJtNum(Number(e.target.value))} className={cn(fieldCls, 'w-24 tabular-nums')} />
              <select value={jtUnit} onChange={(e) => setJtUnit(e.target.value as 'm' | 'h')} className={cn(fieldCls, 'w-28')}>
                <option value="m">{t('schedule.unitMinutes')}</option>
                <option value="h">{t('schedule.unitHours')}</option>
              </select>
            </div>
            <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('schedule.jitterHint')}</p>
          </Field>

          {/* Active Hours */}
          <Field label={t('schedule.activeHours')}>
            <div className="flex items-center gap-2">
              <input type="time" value={ahStart} onChange={(e) => setAhStart(e.target.value)} className={cn(fieldCls, 'w-32')} />
              <span className="text-neutral-400">—</span>
              <input type="time" value={ahEnd} onChange={(e) => setAhEnd(e.target.value)} className={cn(fieldCls, 'w-32')} />
            </div>
          </Field>

          {/* Active Days */}
          <Field label={t('schedule.activeDaysLabel')}>
            <div className="flex flex-wrap gap-1.5">
              {ALL_DAYS.map((d) => (
                <button key={d} type="button" onClick={() => toggleDay(d)} className={cn(chipCls, days.includes(d) ? chipActive : chipInactive)}>{d}</button>
              ))}
            </div>
            <div className="mt-1.5 flex gap-3">
              <button type="button" onClick={setWeekdays} className={shortcutCls}>{t('schedule.shortcutWeekdays')}</button>
              <button type="button" onClick={setWeekends} className={shortcutCls}>{t('schedule.shortcutWeekends')}</button>
              <button type="button" onClick={setAllDays} className={shortcutCls}>{t('schedule.shortcutAll')}</button>
              <button type="button" onClick={clearDays} className={shortcutCls}>{t('schedule.shortcutClear')}</button>
            </div>
          </Field>

          {/* Wakeup condition */}
          <Field label={t('schedule.wakeupCondition')}>
            <div className="space-y-2 rounded-lg border border-neutral-200 bg-neutral-50/50 p-3 dark:border-zinc-700/60 dark:bg-zinc-800/30">
              <div className="flex flex-wrap gap-2">
                <button type="button" onClick={() => setPresetConditions({ tasks: !presetHasTasks, messages: presetHasMessages })} className={cn(chipCls, presetHasTasks ? chipActive : chipInactive)}>
                  {t('schedule.conditionTaskPass')}
                </button>
                <button type="button" onClick={() => setPresetConditions({ tasks: presetHasTasks, messages: !presetHasMessages })} className={cn(chipCls, presetHasMessages ? chipActive : chipInactive)}>
                  {t('schedule.conditionMessagePass')}
                </button>
                <button type="button" onClick={toggleScriptCondition} className={cn(chipCls, useScriptCondition ? chipActive : chipInactive)}>
                  {t('schedule.conditionScriptPass')}
                </button>
              </div>
              {useScriptCondition && (
                <textarea
                  value={wakeupCondition}
                  onChange={(e) => setWakeupCondition(e.target.value)}
                  rows={3}
                  placeholder={t('schedule.wakeupConditionPlaceholder')}
                  className={cn(fieldCls, 'min-h-20 font-mono text-xs')}
                />
              )}
              <p className="text-xs text-neutral-400 dark:text-zinc-500">{t('schedule.wakeupConditionHint')}</p>
            </div>
          </Field>

          {/* Triggers */}
          <Field label={t('schedule.triggers')}>
            <div className="flex flex-wrap gap-2">
              {TRIGGER_TYPES.map(trig => (
                <button key={trig} type="button" onClick={() => toggleTrigger(trig)}
                  className={cn(chipCls, triggers.includes(trig) ? chipActive : chipInactive)}>
                  {t(`schedule.trigger_${trig}`)}
                </button>
              ))}
            </div>
            <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('schedule.triggersHint')}</p>
          </Field>

          {/* Session */}
          <div className="rounded-lg border border-neutral-200 bg-neutral-50/50 px-4 py-3 dark:border-zinc-700/60 dark:bg-zinc-800/30">
            <p className="mb-2.5 text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:text-zinc-400">{t('session.sessionLabel')}</p>
            <div className="space-y-3">
              <Field label={t('session.scopeLabel')}>
                <select value={sessionScope} onChange={(e) => setSessionScope(e.target.value)} className={fieldCls}>
                  <option value="cycle">{t('session.scopeCycle')}</option>
                  <option value="task">{t('session.scopeTask')}</option>
                  <option value="persistent">{t('session.scopePersistent')}</option>
                </select>
                <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('session.scopeHint')}</p>
              </Field>
              <Field label={t('session.sessionIdLabel')}>
                <div className="flex items-center gap-2">
                  <input
                    value={sessionId}
                    onChange={(e) => setSessionId(e.target.value)}
                    placeholder={t('session.sessionIdPlaceholder')}
                    className={cn(fieldCls, 'flex-1 font-mono text-xs')}
                  />
                  {sessionId && (
                    <button type="button" onClick={() => setSessionId('')}
                      className="shrink-0 rounded-md border border-amber-300 bg-amber-50 px-2 py-1.5 text-xs font-medium text-amber-700 hover:bg-amber-100 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-400 dark:hover:bg-amber-900/50">
                      {t('session.clearSession')}
                    </button>
                  )}
                </div>
                <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('session.sessionIdHint')}</p>
              </Field>
            </div>
          </div>

          {/* Max tasks + duration */}
          <div className="grid grid-cols-2 gap-3">
            <Field label={t('schedule.maxTasks')}>
              <input type="number" min={0} value={maxTasks} onChange={(e) => setMaxTasks(Number(e.target.value))} className={fieldCls} />
            </Field>
            <Field label={t('schedule.maxDuration')}>
              <div className="flex items-center gap-2">
                <input type="number" min={0} value={mdNum} onChange={(e) => setMdNum(Number(e.target.value))} className={cn(fieldCls, 'w-20 tabular-nums')} />
                <select value={mdUnit} onChange={(e) => setMdUnit(e.target.value as 'm' | 'h')} className={cn(fieldCls, 'w-24')}>
                  <option value="m">{t('schedule.unitMinutes')}</option>
                  <option value="h">{t('schedule.unitHours')}</option>
                </select>
              </div>
            </Field>
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
  const [editing, setEditing] = useState<{ agent: string; cron: CronRow } | null>(null)

  const allCrons = useMemo(() => {
    const rows: { agent: string; cron: CronRow; model?: string; agentDir?: string }[] = []
    for (const ag of agents) for (const c of ag.crons) rows.push({ agent: ag.name, cron: c, model: ag.model, agentDir: ag.agentDir })
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
          <CalendarClock className="mb-3 size-8 text-neutral-300 dark:text-zinc-500" strokeWidth={1.5} />
          <p className="text-sm text-neutral-500 dark:text-zinc-500">{t('schedule.noCrons')}</p>
        </div>
      )}

      {allCrons.length > 0 && (
        <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
          <table className="min-w-[900px] w-full">
            <thead>
              <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-700/60 dark:bg-zinc-900/40">
                <th className={thCls}>Agent</th>
                <th className={thCls}>{t('forms.title')}</th>
                <th className={thCls}>{t('schedule.cronSchedule')}</th>
                <th className={thCls}>{t('schedule.statusLabel')}</th>
                <th className={thCls}>{t('session.scopeLabel')}</th>
                <th className={thCls}>{t('session.sessionLabel')}</th>
                <th className={thCls}>{t('schedule.nextRun')}</th>
                <th className={thCls}>{t('schedule.lastRun')}</th>
                <th className={thCls}>{t('schedule.runCountLabel')}</th>
                <th className={cn(thCls, thSticky)}>{t('messages.actions')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
              {allCrons.map(({ agent, cron: c, model, agentDir }) => (
                <tr key={`${agent}-${c.id}`} className="group bg-white transition-colors hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30">
                  <td className={cn(tdCls, 'font-mono font-medium')}><Link to={`/projects/${projectId}/members/${agent}`} className="text-sky-700 hover:underline dark:text-sky-400">{agent}</Link></td>
                  <td className={tdCls}>{c.title}</td>
                  <td className={cn(tdCls, 'font-mono')}>{c.schedule}</td>
                  <td className={tdCls}>
                    {c.enabled ? <StatusBadge color="emerald">{t('schedule.cronOn')}</StatusBadge> : <StatusBadge color="neutral">{t('schedule.cronOff')}</StatusBadge>}
                  </td>
                  <td className={tdCls}>
                    {c.sessionScope === 'persistent'
                      ? <StatusBadge color="violet">{t('session.scopePersistent')}</StatusBadge>
                      : <StatusBadge color="neutral">{t('schedule.cronSessionNew')}</StatusBadge>}
                  </td>
                  <td className={tdCls}>
                    {c.sessionScope === 'persistent' && c.sessionId ? (
                      <div className="flex items-center gap-1">
                        <span className="font-mono text-xs text-emerald-700 dark:text-emerald-400" title={c.sessionId}>{c.sessionId.slice(0, 12)}…</span>
                        <CopySessionCmd model={model} sessionId={c.sessionId} agentDir={agentDir} />
                      </div>
                    ) : <span className="text-neutral-400 dark:text-zinc-500">—</span>}
                  </td>
                  <td className={tdCls}>
                    {c.nextRun ? <span className="font-mono text-sky-700 dark:text-sky-400">{fmt(c.nextRun)}</span> : '—'}
                  </td>
                  <td className={tdCls}>{c.lastRun ? fmt(c.lastRun) : '—'}</td>
                  <td className={cn(tdCls, 'tabular-nums')}>{c.runCount ?? 0}</td>
                  <td className={cn(tdCls, tdSticky)}>
                    <div className="flex items-center justify-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                      <button type="button" onClick={() => setEditing({ agent, cron: c })} className={cn(smallBtn, 'text-sky-600 hover:bg-sky-50 dark:text-sky-400 dark:hover:bg-sky-900/30')} title={t('schedule.editCron')}>
                        <Pencil className="size-3.5" strokeWidth={1.8} />
                      </button>
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
      <p className="mt-3 text-xs text-neutral-400 dark:text-zinc-500">{t('schedule.restartHint')}</p>

      {editing && (
        <EditCronModal projectId={projectId} agent={editing.agent} cron={editing.cron} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); onChanged() }} />
      )}
    </>
  )
}

/* ── Cron schedule presets & builder ── */

const CRON_PRESETS = [
  { label: 'cron.presetEveryHour', value: '0 * * * *' },
  { label: 'cron.presetEvery2Hours', value: '0 */2 * * *' },
  { label: 'cron.presetEvery6Hours', value: '0 */6 * * *' },
  { label: 'cron.presetDaily9am', value: '0 9 * * *' },
  { label: 'cron.presetDaily9amWeekdays', value: '0 9 * * 1-5' },
  { label: 'cron.presetTwiceDaily', value: '0 9,18 * * *' },
  { label: 'cron.presetWeeklyMonday', value: '0 9 * * 1' },
  { label: 'cron.presetMonthly1st', value: '0 9 1 * *' },
] as const

function CronScheduleInput({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<'preset' | 'custom'>(() => {
    return CRON_PRESETS.some((p) => p.value === value) ? 'preset' : 'custom'
  })

  const chipCls = 'cursor-pointer rounded-md border px-2.5 py-1 text-xs font-medium transition-colors'
  const chipActive = 'border-sky-500 bg-sky-50 text-sky-700 dark:border-sky-600 dark:bg-sky-900/30 dark:text-sky-300'
  const chipInactive = 'border-neutral-200 bg-white text-neutral-500 hover:border-neutral-300 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-500'

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <button type="button" onClick={() => setMode('preset')} className={cn('text-xs font-medium', mode === 'preset' ? 'text-sky-700 dark:text-sky-400' : 'text-neutral-400 dark:text-zinc-500')}>{t('cron.modePreset')}</button>
        <span className="text-neutral-300 dark:text-zinc-600">|</span>
        <button type="button" onClick={() => setMode('custom')} className={cn('text-xs font-medium', mode === 'custom' ? 'text-sky-700 dark:text-sky-400' : 'text-neutral-400 dark:text-zinc-500')}>{t('cron.modeCustom')}</button>
      </div>
      {mode === 'preset' ? (
        <div className="flex flex-wrap gap-1.5">
          {CRON_PRESETS.map((p) => (
            <button key={p.value} type="button" onClick={() => onChange(p.value)} className={cn(chipCls, value === p.value ? chipActive : chipInactive)}>
              {t(p.label)}
            </button>
          ))}
        </div>
      ) : (
        <input value={value} onChange={(e) => onChange(e.target.value)} className={cn(fieldCls, 'font-mono')} placeholder="0 9 * * 1-5" />
      )}
      {value && <p className="text-[11px] font-mono text-neutral-400 dark:text-zinc-500">{value}</p>}
    </div>
  )
}

function EditCronModal({ projectId, agent, cron, onClose, onSaved }: { projectId: string; agent: string; cron: CronRow; onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()
  const [title, setTitle] = useState(cron.title)
  const [schedule, setSchedule] = useState(cron.schedule)
  const [prompt, setPrompt] = useState(cron.prompt)
  const [sessionScope, setSessionScope] = useState(cron.sessionScope || 'new')
  const [sessionId, setSessionId] = useState(cron.sessionId ?? '')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const parseInterval = (v: string) => {
    const m = v.match(/^(\d+(?:\.\d+)?)\s*(m|h|s)/)
    if (m) return { num: parseFloat(m[1]), unit: m[2] }
    return { num: 0, unit: 'm' }
  }
  const jt = parseInterval(cron.jitter ?? '0m')
  const [jtNum, setJtNum] = useState(jt.num)
  const [jtUnit, setJtUnit] = useState<'m' | 'h'>(jt.unit === 'h' ? 'h' : 'm')

  async function save() {
    setErr(null); setBusy(true)
    try {
      const jitterStr = jtNum > 0 ? `${jtNum}${jtUnit}` : ''
      const body: Record<string, unknown> = {
        title: title.trim(), schedule: schedule.trim(), prompt: prompt.trim(),
        sessionScope, jitter: jitterStr,
      }
      if (sessionId !== (cron.sessionId ?? '')) body.sessionId = sessionId
      await apiPut(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agent)}/crons/${encodeURIComponent(cron.id)}`, body)
      onSaved()
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)) }
    finally { setBusy(false) }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" onClick={() => !busy && onClose()}>
      <div className="flex max-h-[90vh] w-full max-w-lg flex-col rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in" onClick={(e) => e.stopPropagation()}>
        <div className="flex shrink-0 items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
          <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{t('schedule.editCron')} — <span className="font-mono">{agent}</span></h2>
          <button type="button" onClick={onClose} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800"><X className="size-4" /></button>
        </div>
        <div className="flex-1 space-y-4 overflow-y-auto px-5 py-4">
          <Field label={t('forms.title')}>
            <input value={title} onChange={(e) => setTitle(e.target.value)} className={fieldCls} />
          </Field>
          <Field label={t('schedule.cronSchedule')}>
            <CronScheduleInput value={schedule} onChange={setSchedule} />
          </Field>
          <Field label={t('forms.prompt')}>
            <textarea value={prompt} onChange={(e) => setPrompt(e.target.value)} rows={4} className={fieldCls} />
          </Field>

          {/* Jitter */}
          <Field label={t('schedule.jitter')}>
            <div className="flex items-center gap-2">
              <input type="number" min={0} value={jtNum} onChange={(e) => setJtNum(Number(e.target.value))} className={cn(fieldCls, 'w-24 tabular-nums')} />
              <select value={jtUnit} onChange={(e) => setJtUnit(e.target.value as 'm' | 'h')} className={cn(fieldCls, 'w-28')}>
                <option value="m">{t('schedule.unitMinutes')}</option>
                <option value="h">{t('schedule.unitHours')}</option>
              </select>
            </div>
            <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('schedule.jitterHint')}</p>
          </Field>

          {/* Session */}
          <div className="rounded-lg border border-neutral-200 bg-neutral-50/50 px-4 py-3 dark:border-zinc-700/60 dark:bg-zinc-800/30">
            <p className="mb-2.5 text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:text-zinc-400">{t('session.sessionLabel')}</p>
            <div className="space-y-3">
              <Field label={t('session.scopeLabel')}>
                <select value={sessionScope} onChange={(e) => setSessionScope(e.target.value)} className={fieldCls}>
                  <option value="new">{t('schedule.cronSessionNew')}</option>
                  <option value="persistent">{t('session.scopePersistent')}</option>
                </select>
                <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('schedule.cronSessionHint')}</p>
              </Field>
              <Field label={t('session.sessionIdLabel')}>
                <div className="flex items-center gap-2">
                  <input
                    value={sessionId}
                    onChange={(e) => setSessionId(e.target.value)}
                    placeholder={t('session.sessionIdPlaceholder')}
                    className={cn(fieldCls, 'flex-1 font-mono text-xs')}
                  />
                  {sessionId && (
                    <button type="button" onClick={() => setSessionId('')}
                      className="shrink-0 rounded-md border border-amber-300 bg-amber-50 px-2 py-1.5 text-xs font-medium text-amber-700 hover:bg-amber-100 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-400 dark:hover:bg-amber-900/50">
                      {t('session.clearSession')}
                    </button>
                  )}
                </div>
                <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('session.sessionIdHint')}</p>
              </Field>
            </div>
          </div>
        </div>
        {err && <p className="px-5 pb-2 text-sm text-red-600 dark:text-red-400">{err}</p>}
        <div className="flex shrink-0 items-center justify-end gap-2 border-t border-neutral-200 px-5 py-3 dark:border-zinc-700">
          <button type="button" onClick={onClose} className="rounded-lg px-4 py-2 text-sm font-medium text-neutral-500 hover:text-neutral-700 dark:text-zinc-500">{t('forms.cancel')}</button>
          <button type="button" disabled={busy || !title.trim() || !schedule.trim() || !prompt.trim()} onClick={() => void save()} className="rounded-lg bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-40">{busy ? t('forms.saving') : t('common.save')}</button>
        </div>
      </div>
    </div>
  )
}

function AddCronForm({ projectId, agents, defaultAgent, onClose, onCreated }: { projectId: string; agents: string[]; defaultAgent: string; onClose: () => void; onCreated: () => void }) {
  const { t } = useTranslation()
  const [agent, setAgent] = useState(defaultAgent)
  const [title, setTitle] = useState('')
  const [schedule, setSchedule] = useState('')
  const [prompt, setPrompt] = useState('')
  const [sessionScope, setSessionScope] = useState('new')
  const [jtNum, setJtNum] = useState(0)
  const [jtUnit, setJtUnit] = useState<'m' | 'h'>('m')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  async function create() {
    setErr(null); setBusy(true)
    try {
      const jitterStr = jtNum > 0 ? `${jtNum}${jtUnit}` : ''
      await apiPost(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agent)}/crons`, {
        title: title.trim(), schedule: schedule.trim(), prompt: prompt.trim(),
        sessionScope, jitter: jitterStr || undefined,
      })
      onCreated()
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)) }
    finally { setBusy(false) }
  }

  return (
    <div className="mb-4 rounded-lg border border-dashed border-sky-300 bg-sky-50/30 p-4 animate-fade-in dark:border-sky-800 dark:bg-sky-900/10">
      <div className="grid gap-3 sm:grid-cols-2">
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
      </div>
      <div className="mt-3">
        <span className="mb-1 block text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('schedule.cronSchedule')}</span>
        <CronScheduleInput value={schedule} onChange={setSchedule} />
      </div>
      <div className="mt-3">
        <span className="mb-1 block text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('forms.prompt')}</span>
        <textarea value={prompt} onChange={(e) => setPrompt(e.target.value)} rows={2} className={fieldCls} />
      </div>
      <div className="mt-3 grid grid-cols-2 gap-3">
        <div>
          <span className="mb-1 block text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('session.scopeLabel')}</span>
          <select value={sessionScope} onChange={(e) => setSessionScope(e.target.value)} className={selectCls + ' w-full'}>
            <option value="new">{t('schedule.cronSessionNew')}</option>
            <option value="persistent">{t('session.scopePersistent')}</option>
          </select>
        </div>
        <div>
          <span className="mb-1 block text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('schedule.jitter')}</span>
          <div className="flex items-center gap-1.5">
            <input type="number" min={0} value={jtNum} onChange={(e) => setJtNum(Number(e.target.value))} className={cn(fieldCls, 'w-20 tabular-nums')} />
            <select value={jtUnit} onChange={(e) => setJtUnit(e.target.value as 'm' | 'h')} className={selectCls}>
              <option value="m">{t('schedule.unitMinutes')}</option>
              <option value="h">{t('schedule.unitHours')}</option>
            </select>
          </div>
        </div>
      </div>
      <p className="mt-1 text-[11px] text-neutral-400 dark:text-zinc-500">{t('schedule.cronSessionHint')}</p>
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

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`
  return String(n)
}

function ContextUsageBar({ usage }: { usage: SessionUsage }) {
  const { t } = useTranslation()
  const pct = usage.contextLimit > 0 ? Math.min(100, (usage.lastInputTokens / usage.contextLimit) * 100) : 0
  const color = pct >= 80 ? 'bg-red-500' : pct >= 60 ? 'bg-amber-500' : 'bg-sky-500'
  const label = `${formatTokens(usage.lastInputTokens)} / ${formatTokens(usage.contextLimit)}`
  const detail = [
    `${t('context.runs')}: ${usage.runCount}`,
    `In: ${formatTokens(usage.totalInputTokens)}`,
    `Out: ${formatTokens(usage.totalOutputTokens)}`,
    usage.totalCostUsd > 0 ? `$${usage.totalCostUsd.toFixed(4)}` : '',
  ].filter(Boolean).join('  ·  ')

  return (
    <div className="min-w-[120px]" title={detail}>
      <div className="flex items-center justify-between text-[10px] tabular-nums text-neutral-500 dark:text-zinc-500">
        <span>{label}</span>
        <span>{pct.toFixed(0)}%</span>
      </div>
      <div className="mt-0.5 h-1.5 w-full overflow-hidden rounded-full bg-neutral-200 dark:bg-zinc-700">
        <div className={cn('h-full rounded-full transition-all', color)} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}

function RuntimeTab({ agents, projectId }: { agents: AgentSchedule[]; projectId: string }) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const [waking, setWaking] = useState<string | null>(null)
  const [aborting, setAborting] = useState<string | null>(null)
  const [resetting, setResetting] = useState<string | null>(null)
  const [scopeUpdating, setScopeUpdating] = useState<string | null>(null)
  const [viewingLog, setViewingLog] = useState<string | null>(null)

  const runtimeAgents = agents.filter((ag) => {
    const hb = ag.heartbeat
    return hb.enabled || hasTriggers(hb) || hb.lastWakeupStatus === 'running'
  })

  async function doWakeup(agentName: string) {
    setWaking(agentName)
    try {
      await apiPost('/api/v1/scheduler/wakeup', { project: projectId, agent: agentName })
    } catch { /* toast handled by apiFetch */ }
    finally { setWaking(null) }
  }

  async function doAbort(agentName: string) {
    if (!window.confirm(t('schedule.confirmAbort'))) return
    setAborting(agentName)
    try {
      await apiPost('/api/v1/scheduler/abort', { project: projectId, agent: agentName })
    } catch { /* toast handled by apiFetch */ }
    finally { setAborting(null) }
  }

  async function doSessionReset(agentName: string) {
    setResetting(agentName)
    try {
      await apiPost('/api/v1/session/reset', { project: projectId, agent: agentName })
    } catch { /* toast handled by apiFetch */ }
    finally { setResetting(null) }
  }

  async function doScopeChange(agentName: string, scope: string) {
    setScopeUpdating(agentName)
    try {
      await apiPatch(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/heartbeat`, { sessionScope: scope })
    } catch { /* toast handled by apiFetch */ }
    finally { setScopeUpdating(null) }
  }

  if (runtimeAgents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <Zap className="mb-3 size-8 text-neutral-300 dark:text-zinc-500" strokeWidth={1.5} />
        <p className="text-sm text-neutral-500 dark:text-zinc-500">{t('schedule.noRuntimeAgents')}</p>
      </div>
    )
  }

  return (
    <>
      <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
        <table className="min-w-[1100px] w-full">
          <thead>
            <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-700/60 dark:bg-zinc-900/40">
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
              <th className={thCls}>{t('schedule.triggers')}</th>
              <th className={cn(thCls, thSticky)}>{t('messages.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
            {runtimeAgents.map((ag) => {
              const hb = ag.heartbeat
              const isRunningNow = hb.lastWakeupStatus === 'running'
              const hasSession = !!hb.sessionId
              return (
                <tr key={ag.name} className="group bg-white transition-colors hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30">
                  <td className={cn(tdCls, 'font-mono font-medium')}><Link to={`/projects/${projectId}/members/${ag.name}`} className="text-sky-700 hover:underline dark:text-sky-400">{ag.name}</Link></td>
                  <td className={tdCls}>
                    {(() => {
                      if (isRunningNow) return <StatusBadge color="sky">{t('schedule.running')}</StatusBadge>
                      if (hb.paused) return <StatusBadge color="amber">{t('schedule.paused')}</StatusBadge>
                      if (!hb.enabled && hasTriggers(hb)) return <StatusBadge color="violet">{t('schedule.triggerOnly')}</StatusBadge>
                      if (hb.lastWakeupStatus === 'failed') return <StatusBadge color="red">{t('schedule.failed')}</StatusBadge>
                      if (hb.lastWakeupStatus === 'aborted') return <StatusBadge color="orange">{t('schedule.aborted')}</StatusBadge>
                      if (hb.lastWakeupStatus === 'done' || hb.nextWakeupAt) {
                        if (hb.nextWakeupAt && !hb.lastWakeup && (hb.activeHours || hb.activeDays)) {
                          return <StatusBadge color="amber">{t('schedule.outsideWindow')}</StatusBadge>
                        }
                        return <StatusBadge color="emerald">{t('schedule.idle')}</StatusBadge>
                      }
                      return <StatusBadge color="neutral">{t('schedule.waiting')}</StatusBadge>
                    })()}
                  </td>
                  <td className={tdCls}>
                    {hb.nextWakeupAt && new Date(hb.nextWakeupAt) > new Date() ? (
                      <span className="font-mono text-sky-700 dark:text-sky-400">{fmt(hb.nextWakeupAt)}</span>
                    ) : '—'}
                  </td>
                  <td className={tdCls}>
                    {hb.lastWakeup ? (
                      <Link to={`/projects/${projectId}/runs?agent=${encodeURIComponent(projectId + '/' + ag.name)}`} className="text-neutral-700 transition-colors hover:text-sky-600 hover:underline dark:text-zinc-300 dark:hover:text-sky-400">
                        {fmt(hb.lastWakeup)}
                      </Link>
                    ) : '—'}
                  </td>
                  <td className={cn(tdCls, 'font-mono tabular-nums')}>{hb.lastCycleDuration || '—'}</td>
                  <td className={cn(tdCls, 'tabular-nums font-semibold')}>{hb.wakeupCount ?? 0}</td>
                  <td className={cn(tdCls, 'tabular-nums')}>{hb.wakeupCountToday ?? 0}</td>
                  <td className={tdCls}>
                    {hasSession ? (
                      <div className="flex flex-col gap-1">
                        <div className="flex items-center gap-1.5">
                          <div className="flex flex-col gap-0.5">
                            <span className="font-mono text-xs text-emerald-700 dark:text-emerald-400" title={hb.sessionId}>{hb.sessionId!.slice(0, 12)}…</span>
                            {hb.sessionStartedAt && <span className="text-[11px] text-neutral-400 dark:text-zinc-500">{fmt(hb.sessionStartedAt)}</span>}
                          </div>
                          <CopySessionCmd model={ag.model} sessionId={hb.sessionId!} agentDir={ag.agentDir} />
                          <Link
                            to={`/projects/${encodeURIComponent(projectId)}/members/${encodeURIComponent(ag.name)}/chat?sessionId=${encodeURIComponent(hb.sessionId!)}`}
                            title={t('agentChat.openChat')}
                            className="shrink-0 rounded-md p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-sky-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-sky-400"
                          >
                            <MessageSquareText className="size-3.5" strokeWidth={2} />
                          </Link>
                        </div>
                        {ag.sessionUsage && ag.sessionUsage.runCount > 0 && (
                          <ContextUsageBar usage={ag.sessionUsage} />
                        )}
                      </div>
                    ) : (
                      <span className="text-xs text-neutral-400 dark:text-zinc-500">{t('session.noSession')}</span>
                    )}
                  </td>
                  <td className={tdCls}>
                    <select
                      value={hb.sessionScope || 'cycle'}
                      onChange={(e) => void doScopeChange(ag.name, e.target.value)}
                      disabled={scopeUpdating === ag.name}
                      className="h-7 cursor-pointer rounded border border-neutral-200 bg-white px-1.5 text-xs outline-none hover:border-neutral-300 focus:border-sky-400 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 dark:[color-scheme:dark] [&>option]:dark:bg-zinc-800 [&>option]:dark:text-zinc-300"
                    >
                      <option value="cycle">{t('session.scopeCycle')}</option>
                      <option value="task">{t('session.scopeTask')}</option>
                    </select>
                  </td>
                  <td className={tdCls}>
                    <WakeupRuntimeCondition hb={hb} />
                  </td>
                  <td className={tdCls}>
                    <TriggerSummary triggers={hb.triggers} />
                  </td>
                  <td className={cn(tdCls, tdSticky)}>
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
                      {isRunningNow && (
                        <>
                          <button type="button" disabled={aborting === ag.name} onClick={() => void doAbort(ag.name)}
                            className="cursor-pointer rounded-md px-2 py-1 text-xs font-medium text-red-600 opacity-0 transition-all hover:bg-red-50 disabled:opacity-40 group-hover:opacity-100 dark:text-red-400 dark:hover:bg-red-900/20">
                            {aborting === ag.name ? t('schedule.aborting') : t('schedule.abort')}
                          </button>
                          <button type="button" onClick={() => setViewingLog(ag.name)}
                            className="cursor-pointer rounded-md px-2 py-1 text-xs font-medium text-emerald-700 opacity-0 transition-all hover:bg-emerald-50 group-hover:opacity-100 dark:text-emerald-400 dark:hover:bg-emerald-900/20">
                            {t('schedule.viewLog')}
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {viewingLog && <LiveLogModal projectId={projectId} agentName={viewingLog} onClose={() => setViewingLog(null)} />}
    </>
  )
}

function LiveLogModal({ projectId, agentName, onClose }: { projectId: string; agentName: string; onClose: () => void }) {
  const { t } = useTranslation()
  const [content, setContent] = useState('')
  const [finished, setFinished] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let active = true
    const poll = async () => {
      try {
        const data = await apiFetch<{ content: string; path: string; finished: boolean }>(
          `/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/live-log`,
        )
        if (!active) return
        setContent(data.content)
        setFinished(data.finished)
        if (scrollRef.current) {
          scrollRef.current.scrollTop = scrollRef.current.scrollHeight
        }
      } catch { /* ignore poll errors */ }
    }
    void poll()
    const iv = setInterval(poll, 2000)
    return () => { active = false; clearInterval(iv) }
  }, [projectId, agentName])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" onClick={onClose}>
      <div className="flex h-[90vh] w-full max-w-4xl flex-col rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in" onClick={(e) => e.stopPropagation()}>
        <div className="flex shrink-0 items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
          <div className="flex items-center gap-3">
            <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
              <span className="font-mono">{agentName}</span>
            </h2>
            {!finished && (
              <span className="flex items-center gap-1.5 rounded-full bg-sky-100 px-2 py-0.5 text-[11px] font-semibold text-sky-700 dark:bg-sky-900/30 dark:text-sky-400">
                <span className="size-1.5 rounded-full bg-sky-500 animate-pulse" />
                {t('schedule.running')}
              </span>
            )}
            {finished && <StatusBadge color="emerald">{t('schedule.finished')}</StatusBadge>}
          </div>
          <button type="button" onClick={onClose} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800"><X className="size-4" /></button>
        </div>
        <div ref={scrollRef} className="flex-1 overflow-auto px-5 py-4">
          {content ? <ConversationLog content={content} /> : <p className="py-8 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('schedule.noLogYet')}</p>}
        </div>
      </div>
    </div>
  )
}

/* ─── copy session command ─── */

function buildResumeCmd(model: string | undefined, sessionId: string, agentDir: string | undefined): string {
  const cd = agentDir ? `cd ${agentDir} && ` : ''
  const m = (model ?? '').toLowerCase()
  if (m.includes('claude'))  return `${cd}claude --resume ${sessionId}`
  if (m.includes('codex'))   return `${cd}codex exec resume ${sessionId}`
  if (m.includes('gemini'))  return `${cd}gemini --resume ${sessionId}`
  if (m.includes('cursor'))  return `${cd}agent --resume ${sessionId}`
  return `${cd}# session: ${sessionId}`
}

function CopySessionCmd({ model, sessionId, agentDir }: { model?: string; sessionId: string; agentDir?: string }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)

  function doCopy() {
    const cmd = buildResumeCmd(model, sessionId, agentDir)
    void navigator.clipboard.writeText(cmd).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }

  return (
    <button
      type="button"
      onClick={doCopy}
      title={t('schedule.copyResumeCmd')}
      className="shrink-0 rounded-md p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
    >
      {copied
        ? <span className="text-[10px] font-medium text-emerald-600 dark:text-emerald-400">✓</span>
        : <ClipboardCopy className="size-3.5" strokeWidth={2} />}
    </button>
  )
}

/* ─── shared UI ─── */

function hasTriggers(hb: HeartbeatRow): boolean {
  return (hb.triggers?.length ?? 0) > 0
}

function wakeupPresetLabelKeys(preset?: string): string[] {
  if (preset === 'require_tasks') return ['schedule.conditionTaskPass']
  if (preset === 'require_messages') return ['schedule.conditionMessagePass']
  if (preset === 'require_any') return ['schedule.conditionTaskPass', 'schedule.conditionMessagePass']
  return []
}

function triggerSummaryKey(triggers?: string[]): string | null {
  if (!triggers?.length) return null
  const hasTask = triggers.includes('task')
  const hasMessage = triggers.includes('message')
  if (hasTask && hasMessage) return 'schedule.triggerTaskAndMessage'
  if (hasTask) return 'schedule.triggerTaskOnly'
  if (hasMessage) return 'schedule.triggerMessageOnly'
  return 'schedule.eventTrigger'
}

function heartbeatStatusBadge(hb: HeartbeatRow, t: (key: string) => string) {
  if (hb.enabled && !hb.paused) return <StatusBadge color="emerald">{t('schedule.hbActive')}</StatusBadge>
  if (hb.enabled && hb.paused) return <StatusBadge color="amber">{t('schedule.paused')}</StatusBadge>
  if (!hb.enabled && hasTriggers(hb)) return <StatusBadge color="violet">{t('schedule.triggerOnly')}</StatusBadge>
  return <StatusBadge color="neutral">{t('schedule.off')}</StatusBadge>
}

function WakeupConditionSummary({ hb }: { hb: HeartbeatRow }) {
  const { t } = useTranslation()
  const presetKeys = wakeupPresetLabelKeys(hb.wakeupPreset)
  const conditionParts = [
    ...presetKeys.map((key) => ({ key, color: 'sky' as const })),
    ...(hb.wakeupCondition ? [{ key: 'schedule.conditionScriptPass', color: 'sky' as const }] : []),
  ]
  const parts = conditionParts

  if (parts.length > 0) {
    return (
      <div className="flex flex-wrap justify-center gap-1">
        {parts.map((part) => <StatusBadge key={part.key} color={part.color}>{t(part.key)}</StatusBadge>)}
      </div>
    )
  }
  return <span className="text-neutral-400 dark:text-zinc-500">—</span>
}

function TriggerSummary({ triggers }: { triggers?: string[] }) {
  const { t } = useTranslation()
  const triggerKey = triggerSummaryKey(triggers)
  if (!triggerKey) return <span className="text-neutral-400 dark:text-zinc-500">—</span>
  return <StatusBadge color="violet">{t(triggerKey)}</StatusBadge>
}

function WakeupRuntimeCondition({ hb }: { hb: HeartbeatRow }) {
  return <WakeupConditionSummary hb={hb} />
}

function StatusBadge({ color, children }: { color: 'emerald' | 'amber' | 'neutral' | 'sky' | 'red' | 'violet' | 'orange'; children: React.ReactNode }) {
  const cls: Record<string, string> = {
    emerald: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
    amber: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
    neutral: 'bg-neutral-100 text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500',
    sky: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400',
    red: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
    violet: 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400',
    orange: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  }
  return <span className={cn('inline-block rounded-full px-2 py-0.5 text-[11px] font-semibold', cls[color])}>{children}</span>
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <label className="block text-sm"><span className="mb-1 block text-neutral-600 dark:text-zinc-400">{label}</span>{children}</label>
}
