import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Users, Puzzle, ArrowRight } from 'lucide-react'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { useApiJson } from '../../lib/use-api'

type TeamRow = {
  path: string
  name: string
  description?: string
  parent?: string
  skills?: string[]
}

export default function TeamsPage() {
  const { t } = useTranslation()
  const state = useApiJson<TeamRow[]>('/api/v1/teams')

  return (
    <div className="animate-fade-in px-8 py-6">
      <div className="pb-5">
        <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('nav.teams')}</h1>
        <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('teams.subtitle')}</p>
      </div>

      {state.status === 'loading' && (
        <div className="flex items-center gap-2 py-16 justify-center">
          <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
          <span className="text-sm text-neutral-500">{t('api.loading')}</span>
        </div>
      )}
      {state.status === 'error' && (
        <PlaceholderCard title={t('api.loadError')}>
          <p className="text-[13px]">{state.error.message}</p>
          <p className="mt-1.5 text-[11px] text-neutral-400 dark:text-zinc-500">{t('api.hintServe')}</p>
        </PlaceholderCard>
      )}
      {state.status === 'ok' && state.data.length === 0 && (
        <div className="flex flex-col items-center justify-center py-24 text-center">
          <div className="mb-4 flex size-16 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
            <Users className="size-7 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-medium text-neutral-600 dark:text-zinc-400">{t('teams.placeholderTitle')}</p>
          <p className="mt-1.5 text-sm text-neutral-400 dark:text-zinc-500">{t('api.noTeams')}</p>
        </div>
      )}
      {state.status === 'ok' && state.data.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {state.data.map((row) => (
            <Link
              key={row.path}
              to={`/teams/${encodeURIComponent(row.path)}`}
              className="group flex flex-col justify-between rounded-xl border border-neutral-200/80 bg-white p-5 transition-all duration-150 hover:border-sky-300/60 hover:shadow-md dark:border-zinc-700/60 dark:bg-zinc-900/30 dark:hover:border-sky-800/40"
            >
              <div>
                <div className="flex items-center gap-2.5">
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-sky-100 dark:bg-sky-900/30">
                    <Users className="size-4.5 text-sky-600 dark:text-sky-400" strokeWidth={1.8} />
                  </div>
                  <div className="min-w-0">
                    <h3 className="text-sm font-semibold text-neutral-900 dark:text-zinc-100">{row.name}</h3>
                    <p className="font-mono text-xs text-neutral-400 dark:text-zinc-500">{row.path}</p>
                  </div>
                </div>
                {row.description && (
                  <p className="mt-2.5 text-sm leading-relaxed text-neutral-500 dark:text-zinc-500 line-clamp-2">{row.description}</p>
                )}
                {row.parent && (
                  <p className="mt-1.5 text-xs text-neutral-400 dark:text-zinc-500">
                    parent: <span className="font-mono">{row.parent}</span>
                  </p>
                )}
                {row.skills && row.skills.length > 0 && (
                  <div className="mt-2.5 flex flex-wrap gap-1.5">
                    {row.skills.map((sk) => (
                      <span key={sk} className="inline-flex items-center gap-1 rounded-md bg-amber-50 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-400">
                        <Puzzle className="size-3" strokeWidth={2} />
                        {sk}
                      </span>
                    ))}
                  </div>
                )}
              </div>
              <div className="mt-4 flex items-center justify-end">
                <span className="flex items-center gap-1 text-xs font-medium text-sky-600 opacity-0 transition-opacity group-hover:opacity-100 dark:text-sky-400">
                  <ArrowRight className="size-3.5" strokeWidth={2} />
                </span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
