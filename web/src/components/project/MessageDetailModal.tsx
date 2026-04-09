import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { CheckCircle2, Reply, Send, X } from 'lucide-react'
import { apiPost } from '../../lib/api'
import { useFormatDateTime } from '../../lib/format-datetime'

export type MessageDetailModel = {
  id: string
  from: string
  to: string
  subject?: string
  body: string
  sentAt: string
  readAt?: string
  archivedAt?: string
  mailbox: string
}

type Props = {
  open: boolean
  message: MessageDetailModel | null
  onClose: () => void
  onMutated?: () => void
}

export function MessageDetailModal({ open, message, onClose, onMutated }: Props) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()
  const [localReadAt, setLocalReadAt] = useState<string | null>(null)
  const [busy, setBusy] = useState<'read' | 'archive' | 'delete' | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const [replyOpen, setReplyOpen] = useState(false)
  const [replyBody, setReplyBody] = useState('')
  const [replySent, setReplySent] = useState(false)
  const [replyBusy, setReplyBusy] = useState(false)

  useEffect(() => {
    setLocalReadAt(null)
    setErr(null)
    setBusy(null)
    setReplyOpen(false)
    setReplyBody('')
    setReplySent(false)
  }, [message?.id, open])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  const sendReply = useCallback(async () => {
    if (!message || !replyBody.trim()) return
    setReplyBusy(true)
    try {
      await apiPost('/api/v1/messages', { from: 'human', to: message.from, body: replyBody.trim() })
      await apiPost('/api/v1/messages/mark-read', { mailbox: message.mailbox, id: message.id }).catch(() => {})
      setReplyBody('')
      setReplyOpen(false)
      setReplySent(true)
      setLocalReadAt(new Date().toISOString())
      onMutated?.()
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setReplyBusy(false)
    }
  }, [replyBody, message, onMutated])

  if (!open || message == null) return null

  const m = message
  const effectiveReadAt = localReadAt ?? m.readAt
  const isUnread = !effectiveReadAt
  const isArchived = Boolean(m.archivedAt)
  const isSent = m.from === 'human'

  async function markRead() {
    setErr(null)
    setBusy('read')
    try {
      await apiPost<{ ok?: boolean }>('/api/v1/messages/mark-read', { mailbox: m.mailbox, id: m.id })
      setLocalReadAt(new Date().toISOString())
      onMutated?.()
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(null)
    }
  }

  async function archive() {
    setErr(null)
    setBusy('archive')
    try {
      await apiPost<{ ok?: boolean }>('/api/v1/messages/archive', { mailbox: m.mailbox, id: m.id })
      onMutated?.()
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(null)
    }
  }

  async function remove() {
    if (!window.confirm(t('messages.confirmDelete'))) return
    setErr(null)
    setBusy('delete')
    try {
      await apiPost<{ ok?: boolean }>('/api/v1/messages/delete', { mailbox: m.mailbox, id: m.id })
      onMutated?.()
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(null)
    }
  }

  const actionBtn =
    'rounded-md border px-2.5 py-1 text-[11.5px] font-medium transition-all duration-150 disabled:opacity-40'

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm animate-fade-in"
      role="presentation"
      onClick={onClose}
    >
      <div
        className="max-h-[min(85vh,720px)] w-full max-w-lg flex flex-col overflow-hidden rounded-xl border border-neutral-200/80 bg-white shadow-xl animate-scale-in dark:border-zinc-700/60 dark:bg-zinc-900"
        role="dialog"
        aria-labelledby="msg-detail-title"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="shrink-0 flex items-start justify-between gap-3 border-b border-neutral-100 px-4 py-3 dark:border-zinc-700/60">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 id="msg-detail-title" className="text-[13px] font-semibold text-neutral-900 dark:text-zinc-100">
                {t('forms.messageDetail')}
              </h2>
              {isUnread && (
                <span className="rounded-full bg-amber-100 px-1.5 py-0.5 text-[9.5px] font-bold uppercase text-amber-800 dark:bg-amber-900/40 dark:text-amber-300">
                  {t('messages.badgeUnread')}
                </span>
              )}
              {isArchived && (
                <span className="rounded-full bg-neutral-100 px-1.5 py-0.5 text-[9.5px] font-bold uppercase text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500">
                  {t('messages.badgeArchived')}
                </span>
              )}
            </div>
            <p className="mt-0.5 font-mono text-[10.5px] text-neutral-400 dark:text-zinc-500">{m.id}</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="flex size-7 items-center justify-center rounded-md text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
          >
            <X className="size-4" strokeWidth={1.8} />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto space-y-3 px-4 py-3 text-[13px]">
          {err && <p className="rounded-md bg-red-50 px-2.5 py-1.5 text-[12px] text-red-700 dark:bg-red-900/20 dark:text-red-400">{err}</p>}

          <div className="flex flex-wrap gap-x-4 gap-y-1 text-[12px] text-neutral-600 dark:text-zinc-400">
            <span>
              <span className="text-neutral-400 dark:text-zinc-500">{t('forms.from')}:</span>{' '}
              <span className="font-mono text-neutral-800 dark:text-zinc-200">{m.from}</span>
            </span>
            <span>
              <span className="text-neutral-400 dark:text-zinc-500">{t('forms.to')}:</span>{' '}
              <span className="font-mono text-neutral-800 dark:text-zinc-200">{m.to}</span>
            </span>
            <span>
              <span className="text-neutral-400 dark:text-zinc-500">{t('forms.mailbox')}:</span>{' '}
              <span className="font-mono">{m.mailbox}</span>
            </span>
          </div>

          <p className="text-[11px] text-neutral-400 dark:text-zinc-500">
            {t('forms.sentAt')}: {fmt(m.sentAt)}
            {effectiveReadAt ? (
              <>{' · '}{t('forms.readAt')}: {fmt(effectiveReadAt)}</>
            ) : (
              <>{' · '}<span className="text-amber-600 dark:text-amber-400">{t('forms.unread')}</span></>
            )}
          </p>

          {m.subject && (
            <p className="text-[13px] font-medium text-neutral-900 dark:text-zinc-100">{m.subject}</p>
          )}

          <div className="prose prose-sm prose-neutral dark:prose-invert max-w-none rounded-lg border border-neutral-100 bg-neutral-50/50 p-3 text-[12.5px] leading-relaxed dark:border-zinc-700/40 dark:bg-zinc-800/30 [&_pre]:overflow-x-auto [&_pre]:rounded-md [&_pre]:bg-neutral-100 [&_pre]:p-2.5 dark:[&_pre]:bg-zinc-800 [&_code]:text-[12px] [&_table]:text-[12px] [&_a]:text-sky-600 dark:[&_a]:text-sky-400">
            <Markdown remarkPlugins={[remarkGfm]}>{m.body}</Markdown>
          </div>

          {/* Inline reply area */}
          {!isSent && replyOpen && (
            <div className="rounded-lg border border-sky-200/80 bg-sky-50/30 p-3 dark:border-sky-800/40 dark:bg-sky-950/20">
              <div className="mb-2 text-[12px] font-medium text-neutral-500 dark:text-zinc-500">
                {t('workbench.replyTo')} <span className="font-mono text-neutral-700 dark:text-zinc-300">{m.from}</span>
              </div>
              <textarea
                autoFocus
                value={replyBody}
                onChange={(e) => setReplyBody(e.target.value)}
                rows={5}
                placeholder={t('forms.body')}
                className="block w-full resize-y rounded-lg border border-neutral-200 bg-white p-2.5 text-sm leading-relaxed text-neutral-800 outline-none transition-colors placeholder:text-neutral-400 focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:placeholder:text-zinc-600"
              />
              <div className="mt-2.5 flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={() => { setReplyOpen(false); setReplyBody('') }}
                  className="rounded-md px-3 py-1.5 text-[12px] font-medium text-neutral-500 transition-colors hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800"
                >
                  {t('forms.cancel')}
                </button>
                <button
                  type="button"
                  disabled={replyBusy || !replyBody.trim()}
                  onClick={() => void sendReply()}
                  className="flex items-center gap-1.5 rounded-md bg-sky-600 px-4 py-1.5 text-[12px] font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50"
                >
                  <Send className="size-3.5" strokeWidth={2} />
                  {replyBusy ? t('forms.sending') : t('forms.send')}
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Footer actions */}
        <div className="shrink-0 flex flex-wrap items-center gap-2 border-t border-neutral-100 px-4 py-2.5 dark:border-zinc-700/60">
          {!isSent && !replyOpen && (
            <button
              type="button"
              onClick={() => setReplyOpen(true)}
              className={`${actionBtn} border-sky-200 bg-sky-50 text-sky-700 hover:bg-sky-100 dark:border-sky-800 dark:bg-sky-900/30 dark:text-sky-300`}
            >
              {replySent ? (
                <span className="flex items-center gap-1">
                  <CheckCircle2 className="size-3" strokeWidth={2} />
                  {t('workbench.replySent')}
                </span>
              ) : (
                <span className="flex items-center gap-1">
                  <Reply className="size-3" strokeWidth={2} />
                  {t('workbench.replyTo')}
                </span>
              )}
            </button>
          )}
          {isUnread && (
            <button type="button" disabled={busy != null} onClick={() => void markRead()} className={`${actionBtn} border-sky-200 bg-sky-50 text-sky-700 hover:bg-sky-100 dark:border-sky-800 dark:bg-sky-900/30 dark:text-sky-300`}>
              {busy === 'read' ? t('forms.working') : t('forms.markAsRead')}
            </button>
          )}
          {!isArchived && (
            <button type="button" disabled={busy != null} onClick={() => void archive()} className={`${actionBtn} border-neutral-200 text-neutral-600 hover:bg-neutral-50 dark:border-zinc-700 dark:text-zinc-400 dark:hover:bg-zinc-800`}>
              {busy === 'archive' ? t('forms.working') : t('forms.archiveMessage')}
            </button>
          )}
          <button type="button" disabled={busy != null} onClick={() => void remove()} className={`${actionBtn} border-red-200 text-red-600 hover:bg-red-50 dark:border-red-900 dark:text-red-400 dark:hover:bg-red-950/30`}>
            {busy === 'delete' ? t('forms.working') : t('messages.delete')}
          </button>
        </div>
      </div>
    </div>
  )
}
