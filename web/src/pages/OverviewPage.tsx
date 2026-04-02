import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Zap, ListTodo, Activity, Clock } from 'lucide-react'
import { useApiJson } from '../lib/use-api'
import { getRecentVisits } from '../lib/recent-visits'

type Stats = {
  pendingTasks: number
  inProgressTasks: number
  runsToday: number
}

type Agency = {
  name: string
  description?: string
  lang?: string
}

type TelemetrySummaryBlock = {
  runs: number
  inputTokens: number
  outputTokens: number
  costUSD: number
}

type TelemetrySummaryResponse = {
  available: boolean
  summary: TelemetrySummaryBlock | null
}

function fmtNum(n: number) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}

function timeAgo(ts: number, t: (k: string, o?: Record<string, string>) => string) {
  const diff = Math.floor((Date.now() - ts) / 1000)
  if (diff < 60) return t('overview.justNow')
  if (diff < 3600) return t('overview.minutesAgo', { n: String(Math.floor(diff / 60)) })
  if (diff < 86400) return t('overview.hoursAgo', { n: String(Math.floor(diff / 3600)) })
  return t('overview.daysAgo', { n: String(Math.floor(diff / 86400)) })
}

export default function OverviewPage() {
  const { t } = useTranslation()
  const statsState = useApiJson<Stats>('/api/v1/stats')
  const agencyState = useApiJson<Agency>('/api/v1/agency')
  const telemetryState = useApiJson<TelemetrySummaryResponse>('/api/v1/telemetry/summary?since=7d', 0)

  const pending =
    statsState.status === 'ok' ? String(statsState.data.pendingTasks) : statsState.status === 'loading' ? '…' : '—'
  const inProg =
    statsState.status === 'ok' ? String(statsState.data.inProgressTasks) : statsState.status === 'loading' ? '…' : '—'
  const runs =
    statsState.status === 'ok' ? String(statsState.data.runsToday) : statsState.status === 'loading' ? '…' : '—'

  const metricCards: { icon: typeof Zap; key: string; hint: string; value: string }[] = [
    { icon: ListTodo, key: 'metricPending', hint: 'metricHintPending', value: pending },
    { icon: Activity, key: 'metricInProgress', hint: 'metricHintProgress', value: inProg },
    { icon: Zap, key: 'metricRunsToday', hint: 'metricHintRuns', value: runs },
  ]

  const recentVisits = getRecentVisits().filter((v) => v.path !== '/')

  return (
    <div className="animate-fade-in px-8 py-6">
      {(statsState.status === 'error' || agencyState.status === 'error') && (
        <div className="mb-5">
          <p className="rounded-md bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-400">
            {t('api.loadError')} — {t('api.hintServe')}
          </p>
        </div>
      )}

      <div className="grid gap-3 sm:grid-cols-3">
        {metricCards.map(({ icon: Icon, key, hint, value }) => (
          <div
            key={key}
            className="rounded-lg border border-neutral-200/80 bg-white px-5 py-4 transition-colors duration-150 hover:border-neutral-300 dark:border-zinc-700/60 dark:bg-zinc-900/40 dark:hover:border-zinc-700"
          >
            <div className="flex items-center gap-2">
              <Icon className="size-4 text-neutral-400 dark:text-zinc-500" strokeWidth={1.8} />
              <span className="text-xs font-medium uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
                {t(`overview.${key}`)}
              </span>
            </div>
            <p className="mt-2 text-3xl font-bold tabular-nums text-neutral-900 dark:text-zinc-100">{value}</p>
            <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t(`overview.${hint}`)}</p>
          </div>
        ))}
      </div>

      <div className="mt-6 rounded-lg border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
        <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
          {t('overview.telemetryTitle')}
        </h2>
        <p className="mt-1 text-sm text-neutral-400 dark:text-zinc-500">{t('overview.telemetryHint')}</p>
        {telemetryState.status === 'loading' && (
          <div className="flex items-center gap-2 py-6">
            <div className="size-4 animate-spin rounded-full border-2 border-neutral-200 border-t-sky-600 dark:border-zinc-700 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-400">{t('api.loading')}</span>
          </div>
        )}
        {telemetryState.status === 'error' && (
          <p className="mt-3 text-sm text-amber-600 dark:text-amber-400">{telemetryState.error.message}</p>
        )}
        {telemetryState.status === 'ok' && !telemetryState.data.available && (
          <p className="mt-3 text-sm text-neutral-400 dark:text-zinc-500">{t('overview.telemetryNoDb')}</p>
        )}
        {telemetryState.status === 'ok' &&
          telemetryState.data.available &&
          telemetryState.data.summary != null && (
            <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {(
                [
                  ['telemetryRuns', telemetryState.data.summary.runs, false],
                  ['telemetryInTok', telemetryState.data.summary.inputTokens, true],
                  ['telemetryOutTok', telemetryState.data.summary.outputTokens, true],
                  ['telemetryCost', telemetryState.data.summary.costUSD, false],
                ] as const
              ).map(([labelKey, val, shouldFmt]) => (
                <div key={labelKey} className="rounded-md border border-neutral-100 bg-neutral-50/60 px-4 py-3 dark:border-zinc-700/40 dark:bg-zinc-800/30">
                  <p className="text-xs font-medium uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
                    {t(`overview.${labelKey}`)}
                  </p>
                  <p className="mt-1 text-xl font-bold tabular-nums text-neutral-900 dark:text-zinc-100">
                    {labelKey === 'telemetryCost' ? `$${(val as number).toFixed(4)}` : shouldFmt ? fmtNum(val as number) : val}
                  </p>
                </div>
              ))}
            </div>
          )}
      </div>

      {/* Recent visits */}
      {recentVisits.length > 0 && (
        <div className="mt-6 rounded-lg border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
          <div className="flex items-center gap-2 pb-3">
            <Clock className="size-4 text-neutral-400 dark:text-zinc-500" strokeWidth={1.8} />
            <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
              {t('overview.recentTitle')}
            </h2>
          </div>
          <div className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
            {recentVisits.map((v) => (
              <Link
                key={v.path}
                to={v.path}
                className="flex items-center justify-between gap-4 py-2.5 text-sm transition-colors hover:bg-neutral-50/60 dark:hover:bg-zinc-800/20 -mx-2 px-2 rounded-md"
              >
                <span className="truncate font-medium text-neutral-700 dark:text-zinc-300">{v.title}</span>
                <span className="shrink-0 text-xs text-neutral-400 dark:text-zinc-500">{timeAgo(v.visitedAt, t)}</span>
              </Link>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
