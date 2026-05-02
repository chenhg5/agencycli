import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Users, Bot, User, UserMinus } from 'lucide-react'
import { HireAgentDialog } from '../../components/project/HireAgentDialog'
import { cn } from '../../lib/cn'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { useFormatDateTime } from '../../lib/format-datetime'
import { useApiJson } from '../../lib/use-api'
import { apiDelete } from '../../lib/api'

const MODEL_COLORS: Record<string, string> = {
  claudecode:    'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300',
  codex:         'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
  cursor:        'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300',
  gemini:        'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
  qoder:         'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300',
  opencode:      'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-300',
  iflow:         'bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-300',
  'generic-cli': 'bg-neutral-200 text-neutral-700 dark:bg-zinc-700 dark:text-zinc-300',
  'http-agent':  'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
  human:         'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300',
}

type AgentRow = {
  name: string
  model: string
  team: string
  project: string
  hiredAt: string
  avatar?: string
}

function MemberAvatar({ row }: { row: AgentRow }) {
  const isHuman = row.model === 'human'
  const IconCmp = isHuman ? User : Bot
  const iconBg = isHuman
    ? 'bg-indigo-100 dark:bg-indigo-900/30'
    : 'bg-violet-100 dark:bg-violet-900/30'
  const iconColor = isHuman
    ? 'text-indigo-600 dark:text-indigo-400'
    : 'text-violet-600 dark:text-violet-400'

  if (row.avatar) {
    return (
      <img
        src={row.avatar}
        alt=""
        className="size-10 shrink-0 rounded-lg bg-neutral-100 object-cover dark:bg-zinc-800"
        loading="lazy"
      />
    )
  }

  return (
    <div className={cn('flex size-10 shrink-0 items-center justify-center rounded-lg', iconBg)}>
      <IconCmp className={cn('size-5', iconColor)} strokeWidth={1.8} />
    </div>
  )
}

export default function ProjectMembersPage() {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const { projectId } = useParams<{ projectId: string }>()

  const [reloadKey, setReloadKey] = useState(0)
  const agentsPath =
    projectId != null && projectId !== ''
      ? `/api/v1/projects/${encodeURIComponent(projectId)}/agents`
      : null
  const agentsState = useApiJson<AgentRow[]>(agentsPath, reloadKey)
  const members = agentsState.status === 'ok' ? (agentsState.data ?? []) : []

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-5 pb-3">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('projectNav.members')}</h1>
            <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('members.subtitle')}</p>
          </div>
          {projectId && (
            <HireAgentDialog
              projectId={projectId}
              onHired={() => setReloadKey((k) => k + 1)}
            />
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-6 pb-6">
        {agentsState.status === 'loading' && (
          <div className="flex items-center gap-2 py-12 justify-center">
            <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {agentsState.status === 'error' && (
          <PlaceholderCard title={t('api.loadError')}>
            <p className="text-[13px]">{agentsState.error.message}</p>
          </PlaceholderCard>
        )}
        {agentsState.status === 'ok' && members.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="mb-3 flex size-14 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
              <Users className="size-6 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
            </div>
            <p className="text-base font-medium text-neutral-600 dark:text-zinc-400">{t('members.emptyTitle')}</p>
          </div>
        )}
        {agentsState.status === 'ok' && members.length > 0 && (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {members.map((row) => {
              const modelCls = MODEL_COLORS[row.model] ?? 'bg-neutral-100 text-neutral-600 dark:bg-zinc-800 dark:text-zinc-400'
              return (
                <div
                  key={row.name}
                  className="group relative flex flex-col rounded-xl border border-neutral-200/80 bg-white p-4 transition-all duration-150 hover:border-neutral-300 hover:shadow-sm dark:border-zinc-700/60 dark:bg-zinc-900/40 dark:hover:border-zinc-700"
                >
                  <Link
                    to={`/projects/${encodeURIComponent(projectId!)}/members/${encodeURIComponent(row.name)}`}
                    className="flex items-center gap-3"
                  >
                    <MemberAvatar row={row} />
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-mono text-sm font-semibold text-neutral-900 dark:text-zinc-100">{row.name}</p>
                      <p className="mt-0.5 truncate text-xs text-neutral-500 dark:text-zinc-500">{row.team}</p>
                    </div>
                  </Link>
                  <div className="mt-3 flex items-center gap-2">
                    <span className={cn('inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-bold tracking-wide', modelCls)}>
                      {row.model}
                    </span>
                    <span className="ml-auto text-[11px] text-neutral-400 dark:text-zinc-500">{fmt(row.hiredAt)}</span>
                    <button
                      type="button"
                      title={t('members.fire')}
                      onClick={async (e) => {
                        e.preventDefault()
                        if (!confirm(t('members.confirmFire', { name: row.name }))) return
                        await apiDelete(`/api/v1/projects/${encodeURIComponent(projectId!)}/agents/${encodeURIComponent(row.name)}`)
                        setReloadKey((k) => k + 1)
                      }}
                      className="rounded p-1 text-neutral-400 opacity-0 transition-all hover:bg-red-50 hover:text-red-600 group-hover:opacity-100 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                    >
                      <UserMinus className="size-3.5" strokeWidth={1.8} />
                    </button>
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
