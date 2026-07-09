import { useMemo, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { apiPost } from '../../lib/api'
import { cn } from '../../lib/cn'
import type { TaskOption } from '../task/TaskModals'

const TASK_TYPES = ['chore', 'feature', 'bug', 'review', 'triage', 'test', 'research'] as const

type AgentOpt = { name: string }

type ProjectAgentsOpt = { projectId: string; agents: AgentOpt[] }

type Props = {
  projectId: string
  agents: AgentOpt[]
  allProjectsAgents?: ProjectAgentsOpt[]
  taskOptions?: TaskOption[]
  onCreated: () => void
}

const fieldCls =
  'mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-sm text-neutral-900 outline-none transition-colors focus:border-sky-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100'

export function CreateTaskDialog({ projectId: defaultProjectId, agents: defaultAgents, allProjectsAgents, taskOptions = [], onCreated }: Props) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [selectedProject, setSelectedProject] = useState(defaultProjectId)
  const [agent, setAgent] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [prompt, setPrompt] = useState('')
  const [taskType, setTaskType] = useState<string>('chore')
  const [priority, setPriority] = useState(2)
  const [assignee, setAssignee] = useState('')
  const [labelsStr, setLabelsStr] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [parentId, setParentId] = useState('')
  const [estimateDuration, setEstimateDuration] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const multiProject = Boolean(allProjectsAgents && allProjectsAgents.length > 1)

  const currentAgents = useMemo(() => {
    if (!allProjectsAgents) return defaultAgents
    return allProjectsAgents.find((p) => p.projectId === selectedProject)?.agents ?? []
  }, [allProjectsAgents, selectedProject, defaultAgents])

  const allAgentsFlat = useMemo(() => {
    if (!allProjectsAgents) return defaultAgents.map((a) => ({ projectId: defaultProjectId, name: a.name }))
    return allProjectsAgents.flatMap((p) => p.agents.map((a) => ({ projectId: p.projectId, name: a.name })))
  }, [allProjectsAgents, defaultAgents, defaultProjectId])

  function reset() {
    setSelectedProject(defaultProjectId)
    setAgent('')
    setTitle('')
    setDescription('')
    setPrompt('')
    setTaskType('chore')
    setPriority(2)
    setAssignee('')
    setLabelsStr('')
    setDueDate('')
    setParentId('')
    setEstimateDuration('')
    setErr(null)
  }

  function openDialog() {
    reset()
    setOpen(true)
    setTimeout(() => {
      const first = allProjectsAgents
        ? (allProjectsAgents.find((p) => p.projectId === defaultProjectId)?.agents[0]?.name ?? '')
        : (defaultAgents[0]?.name ?? '')
      setAgent(first)
    }, 0)
  }

  function onProjectChange(proj: string) {
    setSelectedProject(proj)
    const projAgents = allProjectsAgents?.find((p) => p.projectId === proj)?.agents ?? []
    setAgent(projAgents[0]?.name ?? '')
  }

  const parentChoices = useMemo(() => {
    return taskOptions.filter((o) => !o.project || o.project === selectedProject)
  }, [taskOptions, selectedProject])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setErr(null)
    if (!agent.trim() || !title.trim() || !prompt.trim()) {
      setErr(t('forms.fillRequired'))
      return
    }
    setBusy(true)
    try {
      const labels = labelsStr.split(',').map(l => l.trim()).filter(Boolean)
      await apiPost<{ id: string }>(
        `/api/v1/projects/${encodeURIComponent(selectedProject)}/tasks`,
        {
          agent: agent.trim(),
          title: title.trim(),
          description: description.trim(),
          prompt: prompt.trim(),
          type: taskType,
          priority,
          ...(assignee ? { assignee } : {}),
          ...(labels.length > 0 ? { labels } : {}),
          ...(dueDate ? { dueDate } : {}),
          ...(parentId ? { parentId } : {}),
          ...(estimateDuration.trim() ? { estimateDuration: estimateDuration.trim() } : {}),
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
        className="rounded-lg border border-sky-600 bg-white px-3 py-2 text-sm font-medium text-sky-700 hover:bg-sky-50 dark:border-sky-500 dark:bg-zinc-900 dark:text-sky-400 dark:hover:bg-zinc-800"
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
              {multiProject && (
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('workbench.filterProject')}</span>
                  <select value={selectedProject} onChange={(e) => onProjectChange(e.target.value)} className={fieldCls}>
                    {allProjectsAgents!.map((p) => <option key={p.projectId} value={p.projectId}>{p.projectId}</option>)}
                  </select>
                </label>
              )}

              {currentAgents.length === 0 && (
                <p className="text-sm text-amber-800 dark:text-amber-400">{t('forms.needAgentsForTask')}</p>
              )}

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.agent')}</span>
                <select value={agent} onChange={(e) => setAgent(e.target.value)} className={fieldCls} disabled={currentAgents.length === 0}>
                  {currentAgents.map((a) => <option key={a.name} value={a.name}>{a.name}</option>)}
                </select>
              </label>

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('tasks.colAssignee')}</span>
                <select value={assignee} onChange={(e) => setAssignee(e.target.value)} className={fieldCls}>
                  <option value="">{t('tasks.assignDefault')}</option>
                  <option value="human">human</option>
                  {allAgentsFlat.map((a) => (
                    <option key={`${a.projectId}/${a.name}`} value={`${a.projectId}/${a.name}`}>
                      {a.projectId}/{a.name}
                    </option>
                  ))}
                </select>
                <p className="mt-0.5 text-xs text-neutral-400 dark:text-zinc-500">{t('tasks.assignHint')}</p>
              </label>

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.title')}</span>
                <input value={title} onChange={(e) => setTitle(e.target.value)} className={fieldCls} />
              </label>

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('tasks.description')}</span>
                <textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={3} className={cn(fieldCls, 'resize-y')} placeholder={t('tasks.descriptionHint')} />
              </label>

              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.prompt')}</span>
                <textarea value={prompt} onChange={(e) => setPrompt(e.target.value)} rows={8} className={cn(fieldCls, 'resize-y')} />
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

              <div className="grid grid-cols-2 gap-3">
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('tasks.dueDate')}</span>
                  <input type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} className={fieldCls} />
                </label>
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('tasks.estimateDuration')}</span>
                  <input value={estimateDuration} onChange={(e) => setEstimateDuration(e.target.value)} placeholder="30m" className={fieldCls} />
                </label>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('tasks.parentTask')}</span>
                  {parentChoices.length > 0 ? (
                    <select value={parentId} onChange={(e) => setParentId(e.target.value)} className={fieldCls}>
                      <option value="">{t('tasks.parentTaskNone')}</option>
                      {parentChoices.map((o) => (
                        <option key={o.id} value={o.id}>{o.title}</option>
                      ))}
                    </select>
                  ) : (
                    <input value={parentId} onChange={(e) => setParentId(e.target.value)} placeholder="t-..." className={cn(fieldCls, 'font-mono text-xs')} />
                  )}
                </label>
                <label className="block text-sm">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('tasks.labels')}</span>
                  <input value={labelsStr} onChange={(e) => setLabelsStr(e.target.value)} placeholder={t('tasks.labelsHint')} className={fieldCls} />
                </label>
              </div>

              {err && <p className="text-sm text-red-600 dark:text-red-400">{err}</p>}
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setOpen(false)} disabled={busy} className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600">{t('forms.cancel')}</button>
                <button type="submit" disabled={busy || currentAgents.length === 0} className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">{busy ? t('forms.saving') : t('forms.submit')}</button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </>
  )
}
