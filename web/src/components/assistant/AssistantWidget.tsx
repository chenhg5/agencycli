import { useCallback, useEffect, useRef, useState, type KeyboardEvent, type PointerEvent as RPointerEvent } from 'react'
import { useTranslation } from 'react-i18next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { MessageSquareText, Send, X, Sparkles } from 'lucide-react'
import { apiPost } from '../../lib/api'
import { cn } from '../../lib/cn'

type ChatMsg = { role: 'user' | 'assistant'; content: string }

const STORAGE_KEY = 'assistant-btn-pos'
function loadPos(): { x: number; y: number } | null {
  try { const v = localStorage.getItem(STORAGE_KEY); return v ? JSON.parse(v) : null } catch { return null }
}

export default function AssistantWidget() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [msgs, setMsgs] = useState<ChatMsg[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  const [pos, setPos] = useState<{ x: number; y: number }>(() => loadPos() ?? { x: -1, y: -1 })
  const dragging = useRef(false)
  const dragOffset = useRef({ dx: 0, dy: 0 })
  const didDrag = useRef(false)

  useEffect(() => {
    if (pos.x < 0) {
      setPos({ x: window.innerWidth - 68, y: window.innerHeight - 120 })
    }
  }, [pos.x])

  function onPointerDown(e: RPointerEvent<HTMLButtonElement>) {
    dragging.current = true
    didDrag.current = false
    dragOffset.current = { dx: e.clientX - pos.x, dy: e.clientY - pos.y }
    e.currentTarget.setPointerCapture(e.pointerId)
  }
  function onPointerMove(e: RPointerEvent<HTMLButtonElement>) {
    if (!dragging.current) return
    const nx = Math.max(0, Math.min(window.innerWidth - 48, e.clientX - dragOffset.current.dx))
    const ny = Math.max(0, Math.min(window.innerHeight - 48, e.clientY - dragOffset.current.dy))
    if (Math.abs(nx - pos.x) > 4 || Math.abs(ny - pos.y) > 4) didDrag.current = true
    setPos({ x: nx, y: ny })
  }
  function onPointerUp(e: RPointerEvent<HTMLButtonElement>) {
    dragging.current = false
    e.currentTarget.releasePointerCapture(e.pointerId)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(pos))
    if (!didDrag.current) setOpen((v) => !v)
  }

  const scrollToBottom = useCallback(() => {
    requestAnimationFrame(() => {
      if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    })
  }, [])

  useEffect(() => { scrollToBottom() }, [msgs, scrollToBottom])
  useEffect(() => { if (open) inputRef.current?.focus() }, [open])

  async function send() {
    const text = input.trim()
    if (!text || loading) return
    setInput('')

    const userMsg: ChatMsg = { role: 'user', content: text }
    const newMsgs = [...msgs, userMsg]
    setMsgs(newMsgs)
    setLoading(true)

    try {
      const data = await apiPost<{ response: string }>('/api/v1/assistant/chat', {
        message: text,
        history: newMsgs.slice(-10),
      })
      setMsgs((prev) => [...prev, { role: 'assistant', content: data.response }])
    } catch (e) {
      const errMsg = e instanceof Error ? e.message : String(e)
      setMsgs((prev) => [...prev, { role: 'assistant', content: `Error: ${errMsg}` }])
    } finally {
      setLoading(false)
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void send()
    }
  }

  const panelRight = Math.max(8, window.innerWidth - pos.x - 48)
  const panelBottom = Math.max(8, window.innerHeight - pos.y + 8)

  return (
    <>
      {/* Floating draggable button */}
      <button
        type="button"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        className={cn(
          'fixed z-[60] flex size-11 cursor-grab items-center justify-center rounded-full shadow-md backdrop-blur-sm transition-all active:cursor-grabbing',
          open
            ? 'bg-neutral-700/70 text-white hover:bg-neutral-800/80 dark:bg-zinc-600/70'
            : 'bg-sky-600/50 text-white hover:bg-sky-600/75 dark:bg-sky-500/50 dark:hover:bg-sky-500/70',
        )}
        style={{ left: pos.x, top: pos.y, touchAction: 'none' }}
        title={t('assistant.title')}
      >
        {open ? <X className="size-4" /> : <Sparkles className="size-4" />}
      </button>

      {/* Chat panel */}
      {open && (
        <div
          className="fixed z-[60] flex h-[540px] w-[400px] flex-col overflow-hidden rounded-2xl border border-neutral-200/60 bg-white/95 shadow-2xl backdrop-blur-md animate-scale-in dark:border-zinc-700/60 dark:bg-zinc-900/95"
          style={{ bottom: panelBottom, right: panelRight }}
        >
          {/* Header */}
          <div className="flex shrink-0 items-center gap-2.5 border-b border-neutral-200/60 px-4 py-3 dark:border-zinc-700/40">
            <div className="flex size-8 items-center justify-center rounded-lg bg-sky-100/80 dark:bg-sky-900/30">
              <MessageSquareText className="size-4 text-sky-600 dark:text-sky-400" />
            </div>
            <div className="flex-1">
              <h3 className="text-sm font-semibold text-neutral-900 dark:text-zinc-100">{t('assistant.title')}</h3>
              <p className="text-[11px] text-neutral-400 dark:text-zinc-600">{t('assistant.subtitle')}</p>
            </div>
            {msgs.length > 0 && (
              <button type="button" onClick={() => setMsgs([])} className="rounded-md px-2 py-1 text-[11px] text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-600 dark:hover:bg-zinc-800">
                {t('assistant.clear')}
              </button>
            )}
          </div>

          {/* Messages */}
          <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
            {msgs.length === 0 && (
              <div className="flex h-full flex-col items-center justify-center text-center">
                <Sparkles className="mb-2 size-8 text-neutral-200 dark:text-zinc-700" strokeWidth={1.2} />
                <p className="text-sm text-neutral-400 dark:text-zinc-600">{t('assistant.welcome')}</p>
                <p className="mt-1 text-xs text-neutral-300 dark:text-zinc-700">{t('assistant.examples')}</p>
              </div>
            )}
            {msgs.map((m, i) => (
              <div key={i} className={cn('flex', m.role === 'user' ? 'justify-end' : 'justify-start')}>
                <div className={cn(
                  'max-w-[85%] rounded-2xl px-3.5 py-2.5 text-[13px] leading-relaxed',
                  m.role === 'user'
                    ? 'rounded-br-md bg-sky-600 text-white'
                    : 'rounded-bl-md bg-neutral-100 text-neutral-800 dark:bg-zinc-800 dark:text-zinc-200',
                )}>
                  {m.role === 'assistant' ? (
                    <div className="prose prose-sm prose-neutral max-w-none dark:prose-invert [&_pre]:bg-neutral-900 [&_pre]:text-neutral-100 [&_code]:text-sky-600 dark:[&_code]:text-sky-400 [&_pre_code]:text-neutral-100">
                      <ReactMarkdown remarkPlugins={[remarkGfm]}>{m.content}</ReactMarkdown>
                    </div>
                  ) : m.content}
                </div>
              </div>
            ))}
            {loading && (
              <div className="flex justify-start">
                <div className="flex items-center gap-1.5 rounded-2xl rounded-bl-md bg-neutral-100 px-4 py-3 dark:bg-zinc-800">
                  <span className="size-1.5 animate-bounce rounded-full bg-neutral-400 dark:bg-zinc-500" style={{ animationDelay: '0ms' }} />
                  <span className="size-1.5 animate-bounce rounded-full bg-neutral-400 dark:bg-zinc-500" style={{ animationDelay: '150ms' }} />
                  <span className="size-1.5 animate-bounce rounded-full bg-neutral-400 dark:bg-zinc-500" style={{ animationDelay: '300ms' }} />
                </div>
              </div>
            )}
          </div>

          {/* Input */}
          <div className="shrink-0 border-t border-neutral-200/60 p-3 dark:border-zinc-700/40">
            <div className="flex items-end gap-2 rounded-xl border border-neutral-200/60 bg-neutral-50/50 px-3 py-2 focus-within:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50">
              <textarea
                ref={inputRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder={t('assistant.placeholder')}
                rows={1}
                className="flex-1 resize-none bg-transparent text-sm text-neutral-800 outline-none placeholder:text-neutral-400 dark:text-zinc-200 dark:placeholder:text-zinc-600"
                style={{ maxHeight: '120px' }}
                onInput={(e) => {
                  const el = e.currentTarget
                  el.style.height = 'auto'
                  el.style.height = Math.min(el.scrollHeight, 120) + 'px'
                }}
              />
              <button
                type="button"
                onClick={() => void send()}
                disabled={loading || !input.trim()}
                className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-sky-600 text-white transition-colors hover:bg-sky-700 disabled:opacity-30 dark:bg-sky-500"
              >
                <Send className="size-3.5" />
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
