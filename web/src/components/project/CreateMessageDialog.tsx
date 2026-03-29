import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { apiFetch, apiPost } from '../../lib/api'

type AgentOpt = { name: string }
type ProjectRow = { name: string }

type Props = {
  projectId: string
  agents: AgentOpt[]
  onSent: () => void
}

export function CreateMessageDialog({ projectId, agents, onSent }: Props) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [from, setFrom] = useState('human')
  const [to, setTo] = useState('human')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const [allMailboxes, setAllMailboxes] = useState<string[]>([])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    ;(async () => {
      try {
        const projects = await apiFetch<ProjectRow[]>('/api/v1/projects')
        const boxes: string[] = []
        for (const p of projects) {
          try {
            const ag = await apiFetch<AgentOpt[]>(`/api/v1/projects/${encodeURIComponent(p.name)}/agents`)
            for (const a of ag) boxes.push(`${p.name}/${a.name}`)
          } catch { /* skip */ }
        }
        if (!cancelled) setAllMailboxes(boxes.sort())
      } catch { /* ignore */ }
    })()
    return () => { cancelled = true }
  }, [open])

  const toOptions = useMemo(() => {
    const opts: { value: string; label: string }[] = [
      { value: 'human', label: 'human' },
    ]
    for (const a of agents) {
      const v = `${projectId}/${a.name}`
      opts.push({ value: v, label: v })
    }
    return opts
  }, [agents, projectId])

  const fromOptions = useMemo(() => {
    const opts: { value: string; label: string }[] = [
      { value: 'human', label: 'human' },
    ]
    for (const mb of allMailboxes) {
      opts.push({ value: mb, label: mb })
    }
    return opts
  }, [allMailboxes])

  function openDialog() {
    setFrom('human')
    setTo('human')
    setSubject('')
    setBody('')
    setErr(null)
    setOpen(true)
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setErr(null)
    if (!body.trim()) {
      setErr(t('forms.bodyRequired'))
      return
    }
    setBusy(true)
    try {
      await apiPost<{ ids: string[] }>('/api/v1/messages', {
        from,
        to,
        subject: subject.trim() || undefined,
        body: body.trim(),
      })
      setOpen(false)
      onSent()
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
        {t('forms.createMessage')}
      </button>
      {open ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4"
          role="presentation"
          onClick={() => !busy && setOpen(false)}
        >
          <div
            className="max-h-[min(90vh,600px)] w-full max-w-md overflow-y-auto rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900"
            onClick={(e) => e.stopPropagation()}
            role="dialog"
            aria-labelledby="create-msg-title"
          >
            <div className="border-b border-neutral-200 px-4 py-3 dark:border-zinc-700">
              <h2
                id="create-msg-title"
                className="text-base font-semibold text-neutral-900 dark:text-zinc-100"
              >
                {t('forms.createMessage')}
              </h2>
            </div>
            <form onSubmit={onSubmit} className="space-y-3 px-4 py-3">
              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.from')}</span>
                <select
                  value={from}
                  onChange={(e) => setFrom(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2 py-1.5 font-mono text-sm dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100"
                >
                  {fromOptions.map((o) => (
                    <option key={o.value} value={o.value}>{o.label}</option>
                  ))}
                </select>
              </label>
              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.to')}</span>
                <select
                  value={to}
                  onChange={(e) => setTo(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2 py-1.5 font-mono text-sm dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100"
                >
                  {toOptions.map((o) => (
                    <option key={o.value} value={o.value}>{o.label}</option>
                  ))}
                </select>
              </label>
              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.subject')}</span>
                <input
                  value={subject}
                  onChange={(e) => setSubject(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2 py-1.5 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100"
                />
              </label>
              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.body')}</span>
                <textarea
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  rows={5}
                  className="mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2 py-1.5 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100"
                />
              </label>
              {err ? <p className="text-sm text-red-600 dark:text-red-400">{err}</p> : null}
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
                  {busy ? t('forms.sending') : t('forms.send')}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </>
  )
}
