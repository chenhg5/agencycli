import { useCallback, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Users, Save, ChevronDown, ChevronRight, FileText, Puzzle, Plus, X } from 'lucide-react'
import { CreateRoleDialog } from '../../components/team/CreateRoleDialog'
import { cn } from '../../lib/cn'
import { PlaceholderCard } from '../../components/ui/PlaceholderCard'
import { apiTeamPath, apiPut, apiPost } from '../../lib/api'
import { useApiJson } from '../../lib/use-api'

type SkillRow = { name: string; description?: string }

type RoleRow = {
  name: string
  description?: string
  skills?: string[]
}

type TeamDetail = {
  path: string
  name: string
  description?: string
  parent?: string
  skills?: string[]
  goals?: string[]
  roles: RoleRow[]
}

type PromptData = { content: string }

function InlinePromptEditor({
  label,
  apiPath,
  initialContent,
}: {
  label: string
  apiPath: string
  initialContent: string
}) {
  const { t } = useTranslation()
  const [value, setValue] = useState(initialContent)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [preview, setPreview] = useState(false)

  const save = useCallback(async () => {
    setSaving(true)
    setSaved(false)
    try {
      await apiPut(apiPath, { content: value })
      setDirty(false)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (e) {
      alert(String(e))
    } finally {
      setSaving(false)
    }
  }, [apiPath, value])

  const change = useCallback((v: string) => {
    setValue(v)
    setDirty(true)
    setSaved(false)
  }, [])

  return (
    <div className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-2.5 dark:border-zinc-700/40">
        <div className="flex items-center gap-2">
          <FileText className="size-4 text-neutral-400 dark:text-zinc-500" strokeWidth={1.8} />
          <span className="text-sm font-medium text-neutral-700 dark:text-zinc-300">{label}</span>
          {dirty && <span className="text-[10px] text-amber-500">●</span>}
          {saved && <span className="text-[10px] text-emerald-500">{t('prompt.saved')}</span>}
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setPreview((p) => !p)}
            className={cn(
              'rounded-md px-2 py-1 text-[11px] font-medium transition-colors',
              preview
                ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400'
                : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400',
            )}
          >
            {preview ? t('prompt.edit') : t('prompt.preview')}
          </button>
          <button
            type="button"
            onClick={save}
            disabled={saving}
            className="flex items-center gap-1 rounded-md bg-sky-600 px-2.5 py-1 text-[11px] font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50"
          >
            <Save className="size-3" strokeWidth={2} />
            {saving ? t('prompt.saving') : t('prompt.save')}
          </button>
        </div>
      </div>
      {preview ? (
        <div className="prose-none max-h-[40vh] overflow-auto p-4 text-sm leading-relaxed text-neutral-800 dark:text-zinc-200">
          <Markdown remarkPlugins={[remarkGfm]}>{value || '*（空）*'}</Markdown>
        </div>
      ) : (
        <textarea
          value={value}
          onChange={(e) => change(e.target.value)}
          className="block w-full resize-y bg-transparent p-4 font-mono text-[13px] leading-relaxed text-neutral-800 outline-none placeholder:text-neutral-300 dark:text-zinc-200 dark:placeholder:text-zinc-700"
          rows={Math.max(4, Math.min(20, value.split('\n').length + 1))}
          placeholder="Markdown prompt..."
        />
      )}
    </div>
  )
}

function SkillTags({
  skills,
  onRemove,
}: {
  skills: string[]
  onRemove?: (name: string) => void
}) {
  if (!skills.length) return null
  return (
    <div className="flex flex-wrap gap-1.5">
      {skills.map((sk) => (
        <span
          key={sk}
          className="inline-flex items-center gap-1 rounded-md bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-400"
        >
          <Puzzle className="size-3" strokeWidth={2} />
          {sk}
          {onRemove && (
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); onRemove(sk) }}
              className="ml-0.5 rounded p-0.5 hover:bg-amber-100 dark:hover:bg-amber-800/30"
            >
              <X className="size-3" strokeWidth={2} />
            </button>
          )}
        </span>
      ))}
    </div>
  )
}

function SkillAdder({
  allSkills,
  currentSkills,
  onAdd,
}: {
  allSkills: SkillRow[]
  currentSkills: string[]
  onAdd: (name: string) => void
}) {
  const { t } = useTranslation()
  const available = allSkills.filter((s) => !currentSkills.includes(s.name))
  const [open, setOpen] = useState(false)

  if (!available.length) return null

  return (
    <div className="relative inline-block">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-1 rounded-md border border-dashed border-neutral-300 px-2 py-1 text-xs text-neutral-500 transition-colors hover:border-neutral-400 hover:text-neutral-700 dark:border-zinc-700 dark:text-zinc-500 dark:hover:border-zinc-600 dark:hover:text-zinc-300"
      >
        <Plus className="size-3" strokeWidth={2} />
        {t('skill.addSkill')}
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-30" onClick={() => setOpen(false)} />
          <div className="absolute left-0 top-full z-40 mt-1 max-h-64 w-48 overflow-y-auto rounded-lg border border-neutral-200 bg-white py-1 shadow-lg dark:border-zinc-700 dark:bg-zinc-800">
            {available.map((sk) => (
              <button
                key={sk.name}
                type="button"
                onClick={() => { onAdd(sk.name); setOpen(false) }}
                className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs transition-colors hover:bg-neutral-50 dark:hover:bg-zinc-700"
              >
                <Puzzle className="size-3 text-amber-500" strokeWidth={2} />
                <span className="font-medium text-neutral-700 dark:text-zinc-300">{sk.name}</span>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

function RolePromptRow({
  teamPath,
  role,
  allSkills,
  onSkillsChanged,
}: {
  teamPath: string
  role: RoleRow
  allSkills: SkillRow[]
  onSkillsChanged: () => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const promptPath = `/api/v1/prompts/roles?team=${encodeURIComponent(teamPath)}&role=${encodeURIComponent(role.name)}`
  const promptState = useApiJson<PromptData>(open ? promptPath : null, 0)
  const [localSkills, setLocalSkills] = useState<string[]>(role.skills ?? [])

  const bindSkill = useCallback(async (skillName: string, action: 'add' | 'remove') => {
    try {
      const res = await apiPost<{ ok: boolean; skills: string[] }>('/api/v1/roles/skills', {
        team: teamPath,
        role: role.name,
        skill: skillName,
        action,
      })
      setLocalSkills(res.skills)
      onSkillsChanged()
    } catch (e) {
      alert(String(e))
    }
  }, [teamPath, role.name, onSkillsChanged])

  return (
    <div className="border-b border-neutral-100 last:border-b-0 dark:border-zinc-700/40">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-neutral-50/80 dark:hover:bg-zinc-800/30"
      >
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="font-mono text-sm text-neutral-900 dark:text-zinc-100">{role.name}</p>
            {localSkills.length > 0 && (
              <span className="rounded-full bg-amber-50 px-1.5 py-0.5 text-[10px] font-medium text-amber-600 dark:bg-amber-900/20 dark:text-amber-400">
                {localSkills.length} {t('skill.skills')}
              </span>
            )}
          </div>
          {role.description && (
            <p className="mt-0.5 text-xs text-neutral-400 dark:text-zinc-500">{role.description}</p>
          )}
        </div>
        {open
          ? <ChevronDown className="size-4 shrink-0 text-neutral-400" strokeWidth={2} />
          : <ChevronRight className="size-4 shrink-0 text-neutral-400" strokeWidth={2} />
        }
      </button>
      {open && (
        <div className="space-y-4 px-4 pb-4">
          {/* Role skills */}
          <div>
            <h4 className="mb-2 flex items-center gap-1.5 text-xs font-semibold text-neutral-500 dark:text-zinc-500">
              <Puzzle className="size-3.5" strokeWidth={2} />
              {t('skill.roleSkills')}
            </h4>
            <div className="flex flex-wrap items-center gap-2">
              <SkillTags skills={localSkills} onRemove={(sk) => bindSkill(sk, 'remove')} />
              <SkillAdder
                allSkills={allSkills}
                currentSkills={localSkills}
                onAdd={(sk) => bindSkill(sk, 'add')}
              />
            </div>
          </div>

          {/* Role prompt */}
          {promptState.status === 'loading' && (
            <div className="flex items-center gap-2 py-3">
              <div className="size-3.5 animate-spin rounded-full border-2 border-neutral-200 border-t-sky-600" />
              <span className="text-xs text-neutral-400">{t('api.loading')}</span>
            </div>
          )}
          {promptState.status === 'ok' && (
            <InlinePromptEditor
              label={t('prompt.rolePrompt', { name: role.name })}
              apiPath={promptPath}
              initialContent={promptState.data.content}
            />
          )}
        </div>
      )}
    </div>
  )
}

export default function TeamDetailPage() {
  const { t } = useTranslation()
  const { teamId } = useParams<{ teamId: string }>()
  const apiPath =
    teamId != null && teamId !== '' ? `/api/v1/teams/${apiTeamPath(teamId)}` : null
  const [reloadKey, setReloadKey] = useState(0)
  const state = useApiJson<TeamDetail>(apiPath, reloadKey)

  const teamPromptPath = teamId ? `/api/v1/prompts/teams/${apiTeamPath(teamId)}` : null
  const teamPromptState = useApiJson<PromptData>(teamPromptPath, 0)

  const allSkillsState = useApiJson<SkillRow[]>('/api/v1/skills', 0)
  const allSkills = allSkillsState.status === 'ok' ? (allSkillsState.data ?? []) : []

  const teamSkills = state.status === 'ok' ? (state.data.skills ?? []) : []

  const bindTeamSkill = useCallback(async (skillName: string, action: 'add' | 'remove') => {
    if (!teamId) return
    try {
      await apiPost('/api/v1/teams/skills', { team: teamId, skill: skillName, action })
      setReloadKey((k) => k + 1)
    } catch (e) {
      alert(String(e))
    }
  }, [teamId])

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-5 pb-3">
        <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('teams.detailSubtitle')}</h1>
        <span className="mt-0.5 block font-mono text-sm text-neutral-800 dark:text-zinc-300">{teamId}</span>
      </div>

      <div className="flex-1 overflow-y-auto px-6 pb-6">
        {state.status === 'loading' && (
          <div className="flex items-center gap-2 py-12 justify-center">
            <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {state.status === 'error' && (
          <PlaceholderCard title={t('api.loadError')}>
            <p className="text-[13px]">{state.error.message}</p>
          </PlaceholderCard>
        )}
        {state.status === 'ok' && (
          <div className="space-y-5">
            {/* Team info */}
            <div className="rounded-lg border border-neutral-200/80 bg-white p-4 dark:border-zinc-700/60 dark:bg-zinc-900/40">
              <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{state.data.name}</h2>
              {state.data.description && (
                <p className="mt-1.5 text-sm text-neutral-500 dark:text-zinc-500">{state.data.description}</p>
              )}
              {state.data.parent && (
                <p className="mt-1.5 text-[11px] text-neutral-400 dark:text-zinc-500">
                  parent: <span className="font-mono">{state.data.parent}</span>
                </p>
              )}
            </div>

            {/* Team skills */}
            <div className="rounded-lg border border-neutral-200/80 bg-white p-4 dark:border-zinc-700/60 dark:bg-zinc-900/40">
              <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-zinc-100">
                <Puzzle className="size-4 text-amber-500" strokeWidth={1.8} />
                {t('skill.teamSkills')}
                {teamSkills.length > 0 && (
                  <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[11px] font-medium text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500">
                    {teamSkills.length}
                  </span>
                )}
              </h3>
              <div className="flex flex-wrap items-center gap-2">
                <SkillTags skills={teamSkills} onRemove={(sk) => bindTeamSkill(sk, 'remove')} />
                <SkillAdder allSkills={allSkills} currentSkills={teamSkills} onAdd={(sk) => bindTeamSkill(sk, 'add')} />
              </div>
            </div>

            {/* Team prompt */}
            {teamPromptState.status === 'ok' && teamId && (
              <InlinePromptEditor
                label={t('prompt.teamPrompt')}
                apiPath={`/api/v1/prompts/teams/${apiTeamPath(teamId)}`}
                initialContent={teamPromptState.data.content}
              />
            )}

            {/* Roles with prompts & skills */}
            <div>
              <div className="mb-2 flex items-center justify-between">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
                  {t('prompt.roles')}
                </h3>
                {teamId && (
                  <CreateRoleDialog
                    teamPath={teamId}
                    allSkills={allSkills}
                    onCreated={() => setReloadKey((k) => k + 1)}
                  />
                )}
              </div>
              {state.data.roles.length > 0 ? (
                <div className="overflow-hidden rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
                  {state.data.roles.map((r) => (
                    <RolePromptRow
                      key={r.name}
                      teamPath={teamId!}
                      role={r}
                      allSkills={allSkills}
                      onSkillsChanged={() => setReloadKey((k) => k + 1)}
                    />
                  ))}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                  <Users className="mb-2 size-8 text-neutral-300 dark:text-zinc-500" strokeWidth={1.5} />
                  <p className="text-sm text-neutral-400 dark:text-zinc-500">{t('teams.detailPlaceholderBody')}</p>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
