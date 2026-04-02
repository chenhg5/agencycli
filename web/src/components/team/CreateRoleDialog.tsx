import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { apiPost } from '../../lib/api'

type Props = {
  teamPath: string
  allSkills: { name: string }[]
  onCreated: () => void
}

const fieldCls =
  'mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-sm text-neutral-900 outline-none transition-colors focus:border-sky-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100'

export function CreateRoleDialog({ teamPath, allSkills, onCreated }: Props) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [desc, setDesc] = useState('')
  const [selectedSkills, setSelectedSkills] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  function reset() {
    setName('')
    setDesc('')
    setSelectedSkills([])
    setErr(null)
  }

  function openDialog() {
    reset()
    setOpen(true)
  }

  function toggleSkill(skill: string) {
    setSelectedSkills((prev) =>
      prev.includes(skill) ? prev.filter((s) => s !== skill) : [...prev, skill],
    )
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setErr(null)
    if (!name.trim()) {
      setErr(t('forms.fillRequired'))
      return
    }
    setBusy(true)
    try {
      await apiPost('/api/v1/roles/create', {
        team: teamPath,
        name: name.trim(),
        description: desc.trim(),
        skills: selectedSkills,
      })
      setOpen(false)
      onCreated()
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
        {t('teams.createRole')}
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
            aria-labelledby="create-role-title"
          >
            <div className="border-b border-neutral-200 px-4 py-3 dark:border-zinc-700">
              <h2 id="create-role-title" className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
                {t('teams.createRole')}
              </h2>
              <p className="mt-0.5 text-xs text-neutral-400 dark:text-zinc-500">
                {t('teams.createRoleDesc', { team: teamPath })}
              </p>
            </div>
            <form onSubmit={onSubmit} className="space-y-3 px-4 py-3">
              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('teams.roleName')} *</span>
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className={fieldCls}
                  placeholder="e.g. content-writer"
                  autoFocus
                />
              </label>

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('teams.roleDescription')}</span>
                <input
                  value={desc}
                  onChange={(e) => setDesc(e.target.value)}
                  className={fieldCls}
                  placeholder={t('teams.roleDescPlaceholder')}
                />
              </label>

              {allSkills.length > 0 && (
                <div className="text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('teams.roleSkills')}</span>
                  <div className="mt-1.5 flex max-h-32 flex-wrap gap-1.5 overflow-y-auto">
                    {allSkills.map((sk) => {
                      const active = selectedSkills.includes(sk.name)
                      return (
                        <button
                          key={sk.name}
                          type="button"
                          onClick={() => toggleSkill(sk.name)}
                          className={`rounded-md px-2 py-1 text-xs font-medium transition-colors ${
                            active
                              ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400'
                              : 'bg-neutral-100 text-neutral-500 hover:bg-neutral-200 dark:bg-zinc-800 dark:text-zinc-500 dark:hover:bg-zinc-700'
                          }`}
                        >
                          {sk.name}
                        </button>
                      )
                    })}
                  </div>
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
                  {busy ? t('forms.saving') : t('forms.submit')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  )
}
