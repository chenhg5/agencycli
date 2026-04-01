import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { apiFetch, apiPost } from '../../lib/api'
import { cn } from '../../lib/cn'

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
  const [toList, setToList] = useState<string[]>([])
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

  const allOpts = useMemo(() => {
    const opts: string[] = ['human']
    if (allMailboxes.length > 0) {
      for (const mb of allMailboxes) opts.push(mb)
    } else {
      for (const a of agents) opts.push(`${projectId}/${a.name}`)
    }
    return opts
  }, [allMailboxes, agents, projectId])

  const toOptions = allOpts
  const fromOptions = allOpts

  const availableToOptions = useMemo(() => toOptions.filter((o) => !toList.includes(o)), [toOptions, toList])

  function openDialog() {
    setFrom('human')
    setToList([])
    setSubject('')
    setBody('')
    setErr(null)
    setOpen(true)
  }

  function addRecipient(value: string) {
    if (value && !toList.includes(value)) {
      setToList((prev) => [...prev, value])
    }
  }

  function removeRecipient(value: string) {
    setToList((prev) => prev.filter((v) => v !== value))
  }

  function selectAllAgents() {
    const all = allMailboxes.length > 0 ? allMailboxes : agents.map((a) => `${projectId}/${a.name}`)
    setToList((prev) => {
      const set = new Set(prev)
      for (const mb of all) set.add(mb)
      return [...set]
    })
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setErr(null)
    if (toList.length === 0) {
      setErr(t('forms.toRequired'))
      return
    }
    if (!body.trim()) {
      setErr(t('forms.bodyRequired'))
      return
    }
    setBusy(true)
    try {
      await apiPost<{ ids: string[] }>('/api/v1/messages', {
        from,
        to: toList.length === 1 ? toList[0] : toList,
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

  const fieldCls = 'w-full rounded-lg border border-neutral-300 bg-white px-2 py-1.5 text-sm dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-100'

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
            className="max-h-[min(90vh,640px)] w-full max-w-lg overflow-y-auto rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in"
            onClick={(e) => e.stopPropagation()}
            role="dialog"
            aria-labelledby="create-msg-title"
          >
            <div className="border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
              <h2 id="create-msg-title" className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
                {t('forms.createMessage')}
              </h2>
            </div>
            <form onSubmit={onSubmit} className="space-y-3 px-5 py-4">
              {/* From */}
              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.from')}</span>
                <select value={from} onChange={(e) => setFrom(e.target.value)} className={cn(fieldCls, 'mt-1 font-mono')}>
                  {fromOptions.map((o) => <option key={o} value={o}>{o}</option>)}
                </select>
              </label>

              {/* To (multi-select) */}
              <div className="text-sm">
                <div className="mb-1 flex items-center justify-between">
                  <span className="text-neutral-600 dark:text-zinc-400">{t('forms.to')}</span>
                  {(allMailboxes.length > 0 || agents.length > 0) && (
                    <button type="button" onClick={selectAllAgents} className="text-[11px] font-medium text-sky-600 hover:text-sky-700 dark:text-sky-400">
                      {t('forms.selectAllAgents')}
                    </button>
                  )}
                </div>
                {/* Selected tags */}
                {toList.length > 0 && (
                  <div className="mb-1.5 flex flex-wrap gap-1">
                    {toList.map((r) => (
                      <span key={r} className="inline-flex items-center gap-1 rounded-md bg-sky-50 px-2 py-0.5 font-mono text-xs text-sky-700 dark:bg-sky-900/30 dark:text-sky-400">
                        {r}
                        <button type="button" onClick={() => removeRecipient(r)} className="rounded-sm p-0.5 hover:bg-sky-100 dark:hover:bg-sky-800/50">
                          <X className="size-3" strokeWidth={2} />
                        </button>
                      </span>
                    ))}
                  </div>
                )}
                {/* Dropdown to add */}
                {availableToOptions.length > 0 && (
                  <select
                    value=""
                    onChange={(e) => { addRecipient(e.target.value); e.target.value = '' }}
                    className={cn(fieldCls, 'font-mono')}
                  >
                    <option value="" disabled>{t('forms.addRecipient')}</option>
                    {availableToOptions.map((o) => <option key={o} value={o}>{o}</option>)}
                  </select>
                )}
                {toList.length === 0 && availableToOptions.length === 0 && (
                  <p className="text-xs text-neutral-400">{t('forms.noRecipients')}</p>
                )}
              </div>

              {/* Subject */}
              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.subject')}</span>
                <input value={subject} onChange={(e) => setSubject(e.target.value)} className={cn(fieldCls, 'mt-1')} />
              </label>

              {/* Body */}
              <label className="block text-sm">
                <span className="text-neutral-600 dark:text-zinc-400">{t('forms.body')}</span>
                <textarea value={body} onChange={(e) => setBody(e.target.value)} rows={10} className={cn(fieldCls, 'mt-1 resize-y')} />
              </label>

              {err ? <p className="text-sm text-red-600 dark:text-red-400">{err}</p> : null}
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setOpen(false)} disabled={busy}
                  className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600">
                  {t('forms.cancel')}
                </button>
                <button type="submit" disabled={busy || toList.length === 0}
                  className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">
                  {busy ? t('forms.sending') : toList.length > 1 ? `${t('forms.send')} (${toList.length})` : t('forms.send')}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </>
  )
}
