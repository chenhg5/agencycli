import { useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { apiPost, apiFetch } from '../../lib/api'
import { apiTeamPath } from '../../lib/api'

const MODELS = [
  'claudecode',
  'codex',
  'cursor',
  'gemini',
  'qoder',
  'opencode',
  'iflow',
  'generic-cli',
  'http-agent',
  'human',
] as const

type TeamInfo = { path: string; name: string }
type TeamDetail = { roles: { name: string; description?: string }[] }
type PersonRow = { username: string; displayName?: string; linkedAgents?: string[] }

type Props = {
  projectId: string
  onHired: () => void
}

const fieldCls =
  'mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-sm text-neutral-900 outline-none transition-colors focus:border-sky-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100'

export function HireAgentDialog({ projectId, onHired }: Props) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [team, setTeam] = useState('')
  const [role, setRole] = useState('')
  const [model, setModel] = useState<string>('claudecode')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [output, setOutput] = useState<string | null>(null)

  const [teams, setTeams] = useState<TeamInfo[]>([])
  const [roles, setRoles] = useState<{ name: string; description?: string }[]>([])
  const [people, setPeople] = useState<PersonRow[]>([])

  useEffect(() => {
    if (!open) return
    apiFetch<TeamInfo[]>('/api/v1/teams').then(setTeams).catch(() => {})
    apiFetch<PersonRow[]>('/api/v1/users')
      .then(users => setPeople((users ?? []).filter(u => (u as any).role === 'member')))
      .catch(() => {})
  }, [open])

  useEffect(() => {
    if (!team) {
      setRoles([])
      setRole('')
      return
    }
    apiFetch<TeamDetail>(`/api/v1/teams/${apiTeamPath(team)}`)
      .then((d) => {
        setRoles(d.roles ?? [])
        setRole('')
      })
      .catch(() => setRoles([]))
  }, [team])

  function reset() {
    setName('')
    setTeam('')
    setRole('')
    setModel('claudecode')
    setErr(null)
    setOutput(null)
  }

  function openDialog() {
    reset()
    setOpen(true)
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setErr(null)
    setOutput(null)
    if (!name.trim() || !team.trim() || !model.trim()) {
      setErr(t('forms.fillRequired'))
      return
    }
    setBusy(true)
    try {
      const res = await apiPost<{ ok: boolean; output: string }>(
        `/api/v1/projects/${encodeURIComponent(projectId)}/hire`,
        {
          name: name.trim(),
          team: team.trim(),
          role: role.trim() || undefined,
          model: model.trim(),
        },
      )
      setOutput(res.output)
      setTimeout(() => {
        setOpen(false)
        onHired()
      }, 1500)
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <button
        type="button"
        onClick={openDialog}
        className="rounded-lg border border-sky-600 bg-white px-3 py-2 text-sm font-medium text-sky-700 hover:bg-sky-50 dark:border-sky-500 dark:bg-zinc-900 dark:text-sky-400 dark:hover:bg-zinc-800"
      >
        {t('members.hireAgent')}
      </button>
      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4"
          role="presentation"
          onClick={() => !busy && setOpen(false)}
        >
          <div
            className="max-h-[min(90vh,640px)] w-full max-w-md overflow-y-auto rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in"
            onClick={(e) => e.stopPropagation()}
            role="dialog"
            aria-labelledby="hire-agent-title"
          >
            <div className="border-b border-neutral-200 px-4 py-3 dark:border-zinc-700">
              <h2 id="hire-agent-title" className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
                {t('members.hireAgent')}
              </h2>
              <p className="mt-0.5 text-xs text-neutral-400 dark:text-zinc-500">
                {t('members.hireAgentDesc', { project: projectId })}
              </p>
            </div>
            <form onSubmit={onSubmit} className="space-y-3 px-4 py-3">
              {model === 'human' ? (
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('people.selectPerson')} *</span>
                  <select value={name} onChange={(e) => setName(e.target.value)} className={fieldCls} autoFocus>
                    <option value="">{t('people.selectPerson')}</option>
                    {people
                      .filter(p => !p.linkedAgents?.some(la => la.startsWith(projectId + '/')))
                      .map(p => (
                        <option key={p.username} value={p.username}>
                          {p.displayName ? `${p.displayName} (@${p.username})` : p.username}
                        </option>
                      ))}
                  </select>
                  {people.length === 0 && (
                    <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">{t('people.createFirst')}</p>
                  )}
                </label>
              ) : (
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('members.agentName')} *</span>
                  <input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className={fieldCls}
                    placeholder="e.g. dev-claude"
                    autoFocus
                  />
                </label>
              )}

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('members.team')} *</span>
                <select value={team} onChange={(e) => setTeam(e.target.value)} className={fieldCls}>
                  <option value="">{t('members.selectTeam')}</option>
                  {teams.map((te) => (
                    <option key={te.path} value={te.path}>
                      {te.name} ({te.path})
                    </option>
                  ))}
                </select>
              </label>

              {roles.length > 0 && (
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('members.role')}</span>
                  <select value={role} onChange={(e) => setRole(e.target.value)} className={fieldCls}>
                    <option value="">{t('members.noRole')}</option>
                    {roles.map((r) => (
                      <option key={r.name} value={r.name}>
                        {r.name}{r.description ? ` — ${r.description}` : ''}
                      </option>
                    ))}
                  </select>
                </label>
              )}

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('members.model')} *</span>
                <select value={model} onChange={(e) => { setModel(e.target.value); setName('') }} className={fieldCls}>
                  {MODELS.map((m) => (
                    <option key={m} value={m}>{m}</option>
                  ))}
                </select>
              </label>

              {output && (
                <div className="rounded-md bg-emerald-50 p-3 text-xs text-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-300">
                  <pre className="whitespace-pre-wrap">{output}</pre>
                </div>
              )}
              {err && <p className="text-sm text-red-600 dark:text-red-400">{err}</p>}
              <div className="flex justify-end gap-2 pt-1">
                <button
                  type="button"
                  onClick={() => setOpen(false)}
                  disabled={busy}
                  className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600"
                >
                  {t('forms.cancel')}
                </button>
                <button
                  type="submit"
                  disabled={busy}
                  className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
                >
                  {busy ? t('members.hiring') : t('members.hire')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  )
}
