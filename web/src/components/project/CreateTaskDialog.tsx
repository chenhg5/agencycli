import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { apiPost } from '../../lib/api'

const TASK_TYPES = ['chore', 'feature', 'bug', 'review', 'triage', 'test', 'research'] as const

type AgentOpt = { name: string }

type Props = {
  projectId: string
  agents: AgentOpt[]
  onCreated: () => void
}

const fieldCls =
  'mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-sm text-neutral-900 outline-none transition-colors focus:border-sky-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100'

export function CreateTaskDialog({ projectId, agents, onCreated }: Props) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [agent, setAgent] = useState('')
  const [title, setTitle] = useState('')
  const [prompt, setPrompt] = useState('')
  const [taskType, setTaskType] = useState<string>('chore')
  const [priority, setPriority] = useState(2)
  const [assignee, setAssignee] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  function reset() {
    setAgent(agents[0]?.name ?? '')
    setTitle('')
    setPrompt('')
    setTaskType('chore')
    setPriority(2)
    setAssignee('')
    setErr(null)
  }

  function openDialog() {
    reset()
    setOpen(true)
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setErr(null)
    if (!agent.trim() || !title.trim() || !prompt.trim()) {
      setErr(t('forms.fillRequired'))
      return
    }
    setBusy(true)
    try {
      await apiPost<{ id: string }>(
        `/api/v1/projects/${encodeURIComponent(projectId)}/tasks`,
        {
          agent: agent.trim(),
          title: title.trim(),
          prompt: prompt.trim(),
          type: taskType,
          priority,
          ...(assignee ? { assignee } : {}),
        },
      )
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
        className="rounded-lg bg-sky-600 px-3 py-2 text-sm font-medium text-white shadow-sm hover:bg-sky-700 dark:bg-sky-600 dark:hover:bg-sky-500"
      >
        {t('forms.createTask')}
      </button>
      {open ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4"
          role="presentation"
          onClick={() => !busy && setOpen(false)}
        >
          <div
            className="max-h-[min(90vh,640px)] w-full max-w-md overflow-y-auto rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in"
            onClick={(e) => e.stopPropagation()}
            role="dialog"
            aria-labelledby="create-task-title"
          >
            <div className="border-b border-neutral-200 px-4 py-3 dark:border-zinc-700">
              <h2 id="create-task-title" className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
                {t('forms.createTask')}
              </h2>
            </div>
            <form onSubmit={onSubmit} className="space-y-3 px-4 py-3">
              {agents.length === 0 && (
                <p className="text-sm text-amber-800 dark:text-amber-400">{t('forms.needAgentsForTask')}</p>
              )}

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.agent')}</span>
                <select value={agent} onChange={(e) => setAgent(e.target.value)} className={fieldCls} disabled={agents.length === 0}>
                  {agents.map((a) => <option key={a.name} value={a.name}>{a.name}</option>)}
                </select>
              </label>

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('tasks.colAssignee')}</span>
                <select value={assignee} onChange={(e) => setAssignee(e.target.value)} className={fieldCls}>
                  <option value="">{t('tasks.assignDefault')}</option>
                  <option value="human">human</option>
                  {agents.map((a) => <option key={a.name} value={`${projectId}/${a.name}`}>{projectId}/{a.name}</option>)}
                </select>
                <p className="mt-0.5 text-xs text-neutral-400 dark:text-zinc-600">{t('tasks.assignHint')}</p>
              </label>

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.title')}</span>
                <input value={title} onChange={(e) => setTitle(e.target.value)} className={fieldCls} />
              </label>

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.prompt')}</span>
                <textarea value={prompt} onChange={(e) => setPrompt(e.target.value)} rows={5} className={fieldCls} />
              </label>

              <div className="grid grid-cols-2 gap-3">
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('forms.type')}</span>
                  <select value={taskType} onChange={(e) => setTaskType(e.target.value)} className={fieldCls}>
                    {TASK_TYPES.map((ty) => <option key={ty} value={ty}>{t(`forms.taskType.${ty}`, { defaultValue: ty })}</option>)}
                  </select>
                </label>
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('forms.priority')}</span>
                  <select value={priority} onChange={(e) => setPriority(Number(e.target.value))} className={fieldCls}>
                    {[0, 1, 2, 3].map((p) => <option key={p} value={p}>P{p} — {t(`forms.priorityLabel.${p}`)}</option>)}
                  </select>
                </label>
              </div>

              {err && <p className="text-sm text-red-600 dark:text-red-400">{err}</p>}
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setOpen(false)} disabled={busy} className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600">{t('forms.cancel')}</button>
                <button type="submit" disabled={busy || agents.length === 0} className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">{busy ? t('forms.saving') : t('forms.submit')}</button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </>
  )
}
