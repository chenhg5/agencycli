import { useCallback, useEffect, useRef, useState, type KeyboardEvent, type PointerEvent as RPointerEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { MessageSquareText, Send, X, Sparkles, Square } from 'lucide-react'
import { ConversationLog } from '../ui/ConversationLog'
import { apiBase } from '../../lib/api'
import { getStoredToken } from '../../lib/auth'
import { cn } from '../../lib/cn'

type ChatMsg =
  | { role: 'user'; content: string }
  | { role: 'assistant'; rawLog: string; summary: string }

const STORAGE_KEY = 'assistant-btn-pos'
function loadPos(): { x: number; y: number } | null {
  try { const v = localStorage.getItem(STORAGE_KEY); return v ? JSON.parse(v) : null } catch { return null }
}

function extractSummary(rawLog: string): string {
  const lines = rawLog.split('\n')
  const texts: string[] = []
  for (const raw of lines) {
    const line = raw.trim()
    if (!line.startsWith('{')) continue
    try {
      const ev = JSON.parse(line)
      if (ev.type === 'assistant' && ev.message?.content) {
        const content = ev.message.content
        if (Array.isArray(content)) {
          for (const b of content) {
            if (b.type === 'text' && b.text) texts.push(b.text)
          }
        } else if (typeof content === 'string') {
          texts.push(content)
        }
      }
      if (ev.type === 'result' && ev.result) texts.push(ev.result)
    } catch { /* skip */ }
  }
  return texts.join('\n').trim() || rawLog.slice(0, 300)
}

export default function AssistantWidget() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [msgs, setMsgs] = useState<ChatMsg[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [streamLog, setStreamLog] = useState('')
  const abortRef = useRef<AbortController | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  const [pos, setPos] = useState<{ x: number; y: number }>(() => loadPos() ?? { x: -1, y: -1 })
  const dragging = useRef(false)
  const dragOffset = useRef({ dx: 0, dy: 0 })
  const didDrag = useRef(false)

  useEffect(() => {
    if (pos.x < 0) setPos({ x: window.innerWidth - 68, y: window.innerHeight - 120 })
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

  useEffect(() => { scrollToBottom() }, [msgs, streamLog, scrollToBottom])
  useEffect(() => { if (open) inputRef.current?.focus() }, [open])

  function stopStream() {
    abortRef.current?.abort()
  }

  async function send() {
    const text = input.trim()
    if (!text || loading) return
    setInput('')

    const userMsg: ChatMsg = { role: 'user', content: text }
    const history = [...msgs, userMsg].slice(-10).map((m) =>
      m.role === 'user' ? { role: 'user', content: m.content } : { role: 'assistant', content: m.summary }
    )
    setMsgs((prev) => [...prev, userMsg])
    setLoading(true)
    setStreamLog('')

    const controller = new AbortController()
    abortRef.current = controller

    let accumulated = ''
    try {
      const token = getStoredToken()
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers['Authorization'] = `Bearer ${token}`

      const res = await fetch(`${apiBase()}/api/v1/assistant/chat`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ message: text, history }),
        signal: controller.signal,
      })

      if (!res.ok) {
        const errText = await res.text()
        throw new Error(errText || `HTTP ${res.status}`)
      }

      const reader = res.body?.getReader()
      if (!reader) throw new Error('no response body')

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        const parts = buffer.split('\n')
        buffer = parts.pop() ?? ''

        for (const part of parts) {
          if (!part.startsWith('data: ')) continue
          const data = part.slice(6)
          if (data === '{"type":"done"}') continue
          accumulated += data + '\n'
          setStreamLog(accumulated)
        }
      }
    } catch (e) {
      if ((e as Error).name === 'AbortError') {
        accumulated += '\n=== Stopped by user ===\n'
      } else {
        const errMsg = e instanceof Error ? e.message : String(e)
        accumulated += `\n=== Error: ${errMsg} ===\n`
      }
    } finally {
      abortRef.current = null
      setStreamLog('')
      setLoading(false)
      if (accumulated.trim()) {
        const summary = extractSummary(accumulated)
        setMsgs((prev) => [...prev, { role: 'assistant', rawLog: accumulated, summary }])
      }
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

      {open && (
        <div
          className="fixed z-[60] flex h-[600px] w-[460px] flex-col overflow-hidden rounded-2xl border border-neutral-200/60 bg-white/95 shadow-2xl backdrop-blur-md animate-scale-in dark:border-zinc-700/60 dark:bg-zinc-900/95"
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
            {msgs.length > 0 && !loading && (
              <button type="button" onClick={() => setMsgs([])} className="rounded-md px-2 py-1 text-[11px] text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 dark:text-zinc-600 dark:hover:bg-zinc-800">
                {t('assistant.clear')}
              </button>
            )}
          </div>

          {/* Messages */}
          <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
            {msgs.length === 0 && !loading && (
              <div className="flex h-full flex-col items-center justify-center text-center">
                <Sparkles className="mb-2 size-8 text-neutral-200 dark:text-zinc-700" strokeWidth={1.2} />
                <p className="text-sm text-neutral-400 dark:text-zinc-600">{t('assistant.welcome')}</p>
                <p className="mt-1 text-xs text-neutral-300 dark:text-zinc-700">{t('assistant.examples')}</p>
              </div>
            )}
            {msgs.map((m, i) => (
              m.role === 'user' ? (
                <div key={i} className="flex justify-end">
                  <div className="max-w-[85%] rounded-2xl rounded-br-md bg-sky-600 px-3.5 py-2.5 text-[13px] leading-relaxed text-white">
                    {m.content}
                  </div>
                </div>
              ) : (
                <div key={i} className="w-full">
                  <ConversationLog content={m.rawLog} />
                </div>
              )
            ))}
            {loading && streamLog && (
              <div className="w-full">
                <ConversationLog content={streamLog} />
              </div>
            )}
            {loading && !streamLog && (
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
              {loading ? (
                <button
                  type="button"
                  onClick={stopStream}
                  className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-red-500 text-white transition-colors hover:bg-red-600"
                  title="Stop"
                >
                  <Square className="size-3" fill="currentColor" />
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => void send()}
                  disabled={!input.trim()}
                  className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-sky-600 text-white transition-colors hover:bg-sky-700 disabled:opacity-30 dark:bg-sky-500"
                >
                  <Send className="size-3.5" />
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  )
}
