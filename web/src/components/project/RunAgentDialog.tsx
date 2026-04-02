import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { apiPost } from '../../lib/api'

type Props = {
  projects: { projectId: string; agents: { name: string }[] }[]
  onDone?: () => void
}

export function RunAgentDialog({ projects, onDone }: Props) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [project, setProject] = useState('')
  const [agent, setAgent] = useState('')
  const [prompt, setPrompt] = useState('')
  const [running, setRunning] = useState(false)
  const [output, setOutput] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const agents = projects.find((p) => p.projectId === project)?.agents ?? []

  const reset = () => {
    setProject(projects[0]?.projectId ?? '')
    setAgent('')
    setPrompt('')
    setOutput(null)
    setError(null)
  }

  const doOpen = () => { reset(); setOpen(true) }

  const doRun = useCallback(async () => {
    if (!project || !agent) return
    setRunning(true); setOutput(null); setError(null)
    try {
      const body: Record<string, string> = { project, agent }
      if (prompt.trim()) body.prompt = prompt.trim()
      const res = await apiPost<{ ok: boolean; output: string }>('/api/v1/run', body)
      setOutput(res.output || t('session.runDone'))
      onDone?.()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally { setRunning(false) }
  }, [project, agent, prompt, onDone, t])

  const fieldCls = 'w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100'

  return (
    <>
      <button type="button" onClick={doOpen}
        className="rounded-lg border border-sky-600 bg-white px-3 py-2 text-sm font-medium text-sky-700 hover:bg-sky-50 dark:border-sky-500 dark:bg-zinc-900 dark:text-sky-400 dark:hover:bg-zinc-800">
        {t('session.runAgent')}
      </button>

      {open && (
        <div className="fixed inset-0 z-50 flex items-start justify-center pt-[10vh]">
          <div className="fixed inset-0 bg-black/30" onClick={() => setOpen(false)} />
          <div className="relative w-full max-w-lg rounded-xl border border-neutral-200 bg-white p-6 shadow-xl dark:border-zinc-700 dark:bg-zinc-900">
            <button type="button" onClick={() => setOpen(false)} className="absolute right-4 top-4 text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-300"><X className="size-5" /></button>
            <h2 className="text-lg font-semibold text-neutral-900 dark:text-zinc-100">{t('session.runAgent')}</h2>
            <p className="mt-1 text-sm text-neutral-500 dark:text-zinc-500">{t('session.runAgentDesc')}</p>

            <div className="mt-5 space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-neutral-700 dark:text-zinc-300">{t('tasks.colProject')}</label>
                <select value={project} onChange={(e) => { setProject(e.target.value); setAgent('') }} className={fieldCls}>
                  <option value="">{t('session.selectProject')}</option>
                  {projects.map((p) => <option key={p.projectId} value={p.projectId}>{p.projectId}</option>)}
                </select>
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium text-neutral-700 dark:text-zinc-300">Agent</label>
                <select value={agent} onChange={(e) => setAgent(e.target.value)} disabled={!project} className={`${fieldCls} disabled:opacity-50`}>
                  <option value="">{t('session.selectAgent')}</option>
                  {agents.map((a) => <option key={a.name} value={a.name}>{a.name}</option>)}
                </select>
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium text-neutral-700 dark:text-zinc-300">
                  Prompt <span className="text-xs font-normal text-neutral-400 dark:text-zinc-500">{t('session.promptOptional')}</span>
                </label>
                <textarea
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  rows={10}
                  placeholder={t('session.promptPlaceholder')}
                  className={`${fieldCls} resize-y font-mono text-[13px] leading-relaxed`}
                />
                <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('session.promptHint')}</p>
              </div>

              {error && <p className="rounded-lg bg-red-50 p-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-400">{error}</p>}
              {output && (
                <pre className="max-h-48 overflow-auto rounded-lg bg-neutral-50 p-3 font-mono text-xs leading-relaxed text-neutral-700 dark:bg-zinc-800 dark:text-zinc-300">{output}</pre>
              )}
            </div>

            <div className="mt-6 flex justify-end gap-3">
              <button type="button" onClick={() => setOpen(false)}
                className="rounded-lg px-4 py-2 text-sm font-medium text-neutral-500 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800">
                {t('forms.cancel')}
              </button>
              <button type="button" onClick={() => void doRun()} disabled={running || !project || !agent}
                className="rounded-lg bg-sky-600 px-5 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50">
                {running ? t('session.running') : t('session.run')}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
