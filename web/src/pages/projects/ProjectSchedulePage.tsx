import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { CalendarClock, Heart, Plus, Pause, Play, Power, Trash2 } from 'lucide-react'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { cn } from '../../lib/cn'
import { apiDelete, apiPatch, apiPost } from '../../lib/api'
import { useApiJson } from '../../lib/use-api'

type CronRow = {
  id: string
  title: string
  schedule: string
  enabled: boolean
  prompt: string
  lastRun?: string
  lastRunStatus?: string
}

type HeartbeatRow = {
  enabled: boolean
  interval: string
  paused: boolean
  activeHours?: string
  activeDays?: string
  wakeupPrompt?: string
  lastWakeup?: string
  lastWakeupStatus?: string
  pid?: number
}

type AgentSchedule = {
  name: string
  heartbeat: HeartbeatRow
  crons: CronRow[]
}

type ScheduleResponse = {
  project: string
  agents: AgentSchedule[]
}

const tinyBtn =
  'inline-flex items-center gap-1 rounded-md border px-2.5 py-1 text-xs font-medium transition-all duration-150 disabled:opacity-40'

export default function ProjectSchedulePage() {
  const { t } = useTranslation()
  const { projectId } = useParams<{ projectId: string }>()
  const path =
    projectId != null && projectId !== ''
      ? `/api/v1/projects/${encodeURIComponent(projectId)}/schedule`
      : null
  const [reloadKey, setReloadKey] = useState(0)
  const state = useApiJson<ScheduleResponse>(path, reloadKey)

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Header */}
      <div className="shrink-0 px-6 pt-5 pb-3">
        <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('projectNav.schedule')}</h1>
        <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('schedule.subtitle')}</p>
      </div>

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
          </PlaceholderCard>
        )}
        {state.status === 'ok' && state.data.agents.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <CalendarClock className="size-6 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-base font-medium text-neutral-700 dark:text-zinc-300">{t('api.noMembers')}</p>
          </div>
        )}

        {state.status === 'ok' && state.data.agents.length > 0 && (
          <div className="space-y-4">
            {state.data.agents.map((ag) => (
              <AgentScheduleCard
                key={ag.name}
                projectId={projectId!}
                agent={ag}
                onChanged={() => setReloadKey((k) => k + 1)}
                t={t}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function AgentScheduleCard({
  projectId,
  agent,
  onChanged,
  t,
}: {
  projectId: string
  agent: AgentSchedule
  onChanged: () => void
  t: (k: string, o?: Record<string, string>) => string
}) {
  const base = `/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agent.name)}`
  const [interval, setInterval] = useState(agent.heartbeat.interval ?? '')
  const [busy, setBusy] = useState<string | null>(null)
  const [cronTitle, setCronTitle] = useState('')
  const [cronSched, setCronSched] = useState('')
  const [cronPrompt, setCronPrompt] = useState('')
  const [showCronForm, setShowCronForm] = useState(false)

  useEffect(() => {
    setInterval(agent.heartbeat.interval ?? '')
  }, [agent.heartbeat.interval])

  async function patchHb(body: Record<string, unknown>) {
    setBusy('hb')
    try {
      await apiPatch(`${base}/heartbeat`, body)
      onChanged()
    } finally { setBusy(null) }
  }

  const hb = agent.heartbeat

  return (
    <section className="overflow-hidden rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-800/60 dark:bg-zinc-900/40">
      {/* Agent header */}
      <div className="flex items-center justify-between border-b border-neutral-100 px-5 py-3 dark:border-zinc-800/50">
        <span className="font-mono text-sm font-semibold text-neutral-900 dark:text-zinc-100">
          {projectId}/{agent.name}
        </span>
        <div className="flex items-center gap-1.5">
          {hb.enabled && !hb.paused && (
            <span className="flex items-center gap-1 rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
              <span className="size-1.5 rounded-full bg-emerald-500 animate-pulse" />
              Active
            </span>
          )}
          {hb.paused && (
            <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-semibold text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
              Paused
            </span>
          )}
          {!hb.enabled && (
            <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[11px] font-semibold text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500">
              Off
            </span>
          )}
        </div>
      </div>

      <div className="grid gap-6 p-5 lg:grid-cols-2">
        {/* Heartbeat */}
        <div>
          <div className="flex items-center gap-1.5 pb-3">
            <Heart className="size-4 text-sky-600 dark:text-sky-400" strokeWidth={1.8} />
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">
              {t('schedule.heartbeat')}
            </span>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              disabled={busy != null}
              onClick={() => void patchHb({ enabled: !hb.enabled })}
              className={cn(tinyBtn, hb.enabled
                ? 'border-emerald-200 text-emerald-700 hover:bg-emerald-50 dark:border-emerald-800 dark:text-emerald-400'
                : 'border-neutral-200 text-neutral-500 hover:bg-neutral-50 dark:border-zinc-700 dark:text-zinc-500')}
            >
              <Power className="size-3.5" strokeWidth={2} />
              {hb.enabled ? t('schedule.disableHb') : t('schedule.enableHb')}
            </button>
            {hb.enabled && (
              <button
                type="button"
                disabled={busy != null}
                onClick={() => void apiPost(`${base}/heartbeat/${hb.paused ? 'resume' : 'pause'}`, {}).then(onChanged)}
                className={cn(tinyBtn, 'border-neutral-200 text-neutral-600 hover:bg-neutral-50 dark:border-zinc-700 dark:text-zinc-400')}
              >
                {hb.paused ? <Play className="size-3.5" strokeWidth={2} /> : <Pause className="size-3.5" strokeWidth={2} />}
                {hb.paused ? t('schedule.resumeHb') : t('schedule.pauseHb')}
              </button>
            )}
          </div>
          <div className="mt-4 flex items-end gap-2">
            <label className="flex-1">
              <span className="mb-1 block text-xs font-medium text-neutral-500 dark:text-zinc-500">
                {t('schedule.interval')}
              </span>
              <input
                value={interval}
                onChange={(e) => setInterval(e.target.value)}
                placeholder="30m, 1h"
                className="w-full rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-1.5 font-mono text-sm text-neutral-800 outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200"
              />
            </label>
            <button
              type="button"
              disabled={busy != null || !interval.trim()}
              onClick={() => void patchHb({ interval: interval.trim() })}
              className="rounded-md bg-sky-600 px-3 py-1.5 text-xs font-medium text-white transition-all duration-150 hover:bg-sky-700 disabled:opacity-40"
            >
              {busy === 'hb' ? t('forms.working') : t('schedule.saveInterval')}
            </button>
          </div>
          {hb.lastWakeup && (
            <p className="mt-3 text-xs text-neutral-400 dark:text-zinc-600">
              {t('schedule.lastWakeup')}: {hb.lastWakeup} ({hb.lastWakeupStatus ?? '—'})
              {hb.pid ? ` · pid ${hb.pid}` : ''}
            </p>
          )}
        </div>

        {/* Crons */}
        <div>
          <div className="flex items-center justify-between pb-3">
            <div className="flex items-center gap-1.5">
              <CalendarClock className="size-4 text-sky-600 dark:text-sky-400" strokeWidth={1.8} />
              <span className="text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-600">
                {t('schedule.crons')}
              </span>
              <span className="rounded-full bg-neutral-100 px-1.5 py-px text-[11px] font-bold text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500">
                {agent.crons.length}
              </span>
            </div>
            <button
              type="button"
              onClick={() => setShowCronForm(!showCronForm)}
              className={cn(tinyBtn, 'border-neutral-200 text-neutral-600 hover:bg-neutral-50 dark:border-zinc-700 dark:text-zinc-400')}
            >
              <Plus className="size-3.5" strokeWidth={2} />
              {t('schedule.addCron')}
            </button>
          </div>

          {agent.crons.length === 0 && !showCronForm && (
            <p className="py-4 text-center text-sm text-neutral-400 dark:text-zinc-600">{t('schedule.noCrons')}</p>
          )}

          {agent.crons.length > 0 && (
            <div className="divide-y divide-neutral-100 rounded-md border border-neutral-200/70 dark:divide-zinc-800/40 dark:border-zinc-800/50">
              {agent.crons.map((c) => (
                <div key={c.id} className="group flex items-center justify-between px-3 py-2.5">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-neutral-800 dark:text-zinc-200">{c.title}</span>
                      <span className={cn(
                        'rounded-full px-1.5 py-px text-[10px] font-semibold',
                        c.enabled
                          ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                          : 'bg-neutral-100 text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500',
                      )}>
                        {c.enabled ? t('schedule.cronOn') : t('schedule.cronOff')}
                      </span>
                    </div>
                    <span className="font-mono text-xs text-neutral-400 dark:text-zinc-600">{c.schedule}</span>
                  </div>
                  <div className="flex items-center gap-1 opacity-0 transition-opacity duration-100 group-hover:opacity-100">
                    <button
                      type="button"
                      onClick={() => void apiPost(`${base}/crons/${encodeURIComponent(c.id)}/${c.enabled ? 'pause' : 'resume'}`, {}).then(onChanged)}
                      className="rounded p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
                      title={c.enabled ? t('schedule.pauseCron') : t('schedule.resumeCron')}
                    >
                      {c.enabled ? <Pause className="size-3.5" strokeWidth={1.8} /> : <Play className="size-3.5" strokeWidth={1.8} />}
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        if (window.confirm(t('schedule.confirmDeleteCron')))
                          void apiDelete(`${base}/crons/${encodeURIComponent(c.id)}`).then(onChanged)
                      }}
                      className="rounded p-1 text-neutral-400 transition-colors hover:bg-red-50 hover:text-red-600 dark:text-zinc-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                      title={t('schedule.deleteCron')}
                    >
                      <Trash2 className="size-3.5" strokeWidth={1.8} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {showCronForm && (
            <div className="mt-3 rounded-md border border-dashed border-neutral-200 p-4 animate-fade-in dark:border-zinc-700/60">
              <div className="grid gap-2 sm:grid-cols-2">
                <input
                  value={cronTitle}
                  onChange={(e) => setCronTitle(e.target.value)}
                  placeholder={t('forms.title')}
                  className="rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-1.5 text-sm outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200"
                />
                <input
                  value={cronSched}
                  onChange={(e) => setCronSched(e.target.value)}
                  placeholder="0 9 * * 1-5"
                  className="rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-1.5 font-mono text-sm outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200"
                />
              </div>
              <textarea
                value={cronPrompt}
                onChange={(e) => setCronPrompt(e.target.value)}
                placeholder={t('forms.prompt')}
                rows={2}
                className="mt-2 w-full rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-1.5 text-sm outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200"
              />
              <div className="mt-3 flex gap-2">
                <button
                  type="button"
                  disabled={!cronTitle.trim() || !cronSched.trim() || !cronPrompt.trim()}
                  onClick={() =>
                    void apiPost(`${base}/crons`, {
                      title: cronTitle.trim(),
                      schedule: cronSched.trim(),
                      prompt: cronPrompt.trim(),
                    }).then(() => {
                      setCronTitle(''); setCronSched(''); setCronPrompt('')
                      setShowCronForm(false)
                      onChanged()
                    })
                  }
                  className="rounded-md bg-sky-600 px-4 py-1.5 text-xs font-medium text-white transition-all hover:bg-sky-700 disabled:opacity-40"
                >
                  {t('schedule.createCron')}
                </button>
                <button
                  type="button"
                  onClick={() => setShowCronForm(false)}
                  className="rounded-md px-4 py-1.5 text-xs font-medium text-neutral-500 transition-colors hover:text-neutral-700 dark:text-zinc-500"
                >
                  {t('forms.cancel')}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  )
}
