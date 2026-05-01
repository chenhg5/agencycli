import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Edit3, Maximize2, Minimize2, RefreshCw, Send, Sparkles, Square } from 'lucide-react'
import { ConversationLog } from '../../components/ui/ConversationLog'
import { apiFetch, apiUrl } from '../../lib/api'
import { getStoredToken } from '../../lib/auth'
import { cn } from '../../lib/cn'

type HistoryResp = {
  sessionId?: string
  content?: string
  truncated?: boolean
  runs?: Array<{ startedAt: string; status: string; logPath: string }>
}

type AgentContext = {
  model: string
  provider?: string
  env?: Record<string, string>
  httpAgent?: {
    model?: string
  }
}

type ProviderOption = {
  id: string
  model?: string
}

type LiveLogResp = {
  content: string
  finished: boolean
}

function appendLog(prev: string, line: string): string {
  return prev ? `${prev}\n${line}` : line
}

function resetTextareaHeight(el: HTMLTextAreaElement | null) {
  if (el) el.style.height = 'auto'
}

function agentChatDraftKey(projectId?: string, agentName?: string) {
  if (!projectId || !agentName) return null
  return `agencycli.agentChatDraft:${projectId}:${agentName}`
}

function readAgentChatDraft(projectId?: string, agentName?: string) {
  const key = agentChatDraftKey(projectId, agentName)
  if (!key) return ''
  try {
    return window.sessionStorage.getItem(key) ?? ''
  } catch {
    return ''
  }
}

export default function ProjectAgentChatPage() {
  const { t } = useTranslation()
  const { projectId, agentName } = useParams<{ projectId: string; agentName: string }>()
  const [searchParams] = useSearchParams()
  const initialSessionId = searchParams.get('sessionId') ?? ''
  const [sessionId, setSessionId] = useState(initialSessionId)
  const [content, setContent] = useState('')
  const [input, setInput] = useState(() => readAgentChatDraft(projectId, agentName))
  const [loading, setLoading] = useState(false)
  const [historyLoading, setHistoryLoading] = useState(false)
  const [historyTruncated, setHistoryTruncated] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [freshNext, setFreshNext] = useState(false)
  const [focusMode, setFocusMode] = useState(false)
  const [agentContext, setAgentContext] = useState<AgentContext | null>(null)
  const [providers, setProviders] = useState<ProviderOption[]>([])
  const [sessionEditorOpen, setSessionEditorOpen] = useState(false)
  const [sessionDraft, setSessionDraft] = useState(initialSessionId)
  const [followingLiveLog, setFollowingLiveLog] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const abortRef = useRef<AbortController | null>(null)
  const draftKey = agentChatDraftKey(projectId, agentName)

  const historyPath = useCallback((sid: string) => {
    if (!projectId || !agentName) return null
    const base = `/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/chat/history`
    return sid ? `${base}?sessionId=${encodeURIComponent(sid)}` : base
  }, [projectId, agentName])

  const loadHistory = useCallback(async (sid: string) => {
    const path = historyPath(sid)
    if (!path) return
    setHistoryLoading(true)
    setError(null)
    try {
      const data = await apiFetch<HistoryResp>(path)
      setSessionId(data.sessionId ?? sid)
      setSessionDraft(data.sessionId ?? sid)
      setContent(data.content ?? '')
      setHistoryTruncated(Boolean(data.truncated))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setHistoryLoading(false)
    }
  }, [historyPath])

  useEffect(() => {
    void loadHistory(initialSessionId)
  }, [initialSessionId, loadHistory])

  useEffect(() => {
    if (!projectId || !agentName) return
    let cancelled = false
    const contextPath = `/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/context`
    void apiFetch<AgentContext>(contextPath)
      .then((data) => { if (!cancelled) setAgentContext(data) })
      .catch(() => {})
    void apiFetch<ProviderOption[]>('/api/v1/providers')
      .then((data) => { if (!cancelled) setProviders(data ?? []) })
      .catch(() => {})
    return () => { cancelled = true }
  }, [projectId, agentName])

  useEffect(() => {
    requestAnimationFrame(() => {
      if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    })
  }, [content, loading, followingLiveLog])

  useEffect(() => {
    if (!draftKey) return
    try {
      if (input) window.sessionStorage.setItem(draftKey, input)
      else window.sessionStorage.removeItem(draftKey)
    } catch {
      // Ignore storage failures; losing a draft is better than breaking chat.
    }
  }, [draftKey, input])

  useEffect(() => {
    const el = inputRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 128) + 'px'
  }, [input])

  useEffect(() => {
    if (!projectId || !agentName || loading) return
    let cancelled = false
    let timer: number | null = null

    const poll = async () => {
      try {
        const data = await apiFetch<LiveLogResp>(
          `/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/live-log`,
        )
        if (cancelled) return
        if (data.content && !data.finished) {
          if (!sessionId || data.content.includes(sessionId)) {
            setContent(data.content)
            setFollowingLiveLog(true)
          }
        } else if (data.finished) {
          setFollowingLiveLog(false)
        }
      } catch {
        if (!cancelled) setFollowingLiveLog(false)
      }
      if (!cancelled) timer = window.setTimeout(poll, 2000)
    }

    void poll()
    return () => {
      cancelled = true
      if (timer != null) window.clearTimeout(timer)
    }
  }, [projectId, agentName, sessionId, loading])

  useEffect(() => {
    if (!focusMode) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [focusMode])

  async function send() {
    const text = input.trim()
    if (!text || loading || !projectId || !agentName) return

    setInput('')
    resetTextareaHeight(inputRef.current)
    setError(null)
    setLoading(true)
    setContent((prev) => appendLog(prev, JSON.stringify({ type: 'human', content: text })))

    const controller = new AbortController()
    abortRef.current = controller
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        'Accept': 'text/event-stream',
      }
      const token = getStoredToken()
      if (token) headers.Authorization = `Bearer ${token}`

      const res = await fetch(apiUrl(`/api/v1/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentName)}/chat`), {
        method: 'POST',
        headers,
        body: JSON.stringify({
          message: text,
          sessionId,
          noSession: freshNext && !sessionId,
        }),
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
          if (!data) continue
          try {
            const evt = JSON.parse(data)
            if (evt.type === 'chat_done') {
              if (evt.session_id) setSessionId(evt.session_id)
              continue
            }
            if (evt.type === 'chat_error') {
              const msg = evt.error ? String(evt.error) : t('agentChat.error')
              setError(msg)
              setContent((prev) => appendLog(prev, `=== Error: ${msg} ===`))
              continue
            }
            if (evt.session_id) setSessionId(evt.session_id)
          } catch {
            // Non-JSON status lines are still useful in the raw log.
          }
          setContent((prev) => appendLog(prev, data))
        }
      }
    } catch (e) {
      const stopped = (e as Error).name === 'AbortError'
      const msg = stopped ? t('agentChat.stopped') : (e instanceof Error ? e.message : String(e))
      setError(stopped ? null : msg)
      setContent((prev) => appendLog(prev, `=== ${msg} ===`))
    } finally {
      abortRef.current = null
      setFreshNext(false)
      setLoading(false)
      inputRef.current?.focus()
    }
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void send()
    }
  }

  function startFresh() {
    abortRef.current?.abort()
    setSessionId('')
    setSessionDraft('')
    setContent('')
    setError(null)
    setFreshNext(true)
    resetTextareaHeight(inputRef.current)
    inputRef.current?.focus()
  }

  function switchSession() {
    const next = sessionDraft.trim()
    setSessionEditorOpen(false)
    setFreshNext(false)
    setSessionId(next)
    void loadHistory(next)
  }

  if (!projectId || !agentName) return null

  const agentType = agentContext?.model || ''
  const providerModel = providers.find((p) => p.id === agentContext?.provider)?.model
  const env = agentContext?.env ?? {}
  const concreteModel =
    agentContext?.httpAgent?.model ||
    env.ANTHROPIC_MODEL ||
    env.CLAUDE_MODEL ||
    env.OPENAI_MODEL ||
    env.CODEX_MODEL ||
    env.GEMINI_MODEL ||
    env.GOOGLE_MODEL ||
    env.CURSOR_MODEL ||
    providerModel ||
    ''

  const chatPanel = (
    <div className={cn(
      'flex h-full flex-col overflow-hidden bg-white dark:bg-zinc-950',
      focusMode && 'mx-auto w-full max-w-5xl rounded-2xl border border-neutral-200/80 shadow-2xl dark:border-zinc-700/70',
    )}>
      <div className="shrink-0 border-b border-neutral-200/70 px-6 py-4 dark:border-zinc-700/50">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0">
            <h1 className="truncate text-xl font-semibold text-neutral-900 dark:text-zinc-100">{agentName}</h1>
            <div className="mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-sm text-neutral-500 dark:text-zinc-500">
              {agentType && (
                <span className="rounded-md bg-neutral-100 px-2 py-0.5 font-mono text-xs font-semibold text-neutral-700 dark:bg-zinc-800 dark:text-zinc-300">
                  {agentType}
                </span>
              )}
              <span className="font-mono text-xs">
                {t('agentChat.modelLabel')}: {concreteModel || t('agentChat.modelUnknown')}
              </span>
              {sessionId && (
                <span className="break-all font-mono text-xs text-emerald-700 dark:text-emerald-400">
                  {sessionId}
                </span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => { setSessionDraft(sessionId); setSessionEditorOpen(true) }}
              disabled={loading}
              className="flex items-center gap-1.5 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-50 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800"
            >
              <Edit3 className="size-3.5" strokeWidth={1.8} />
              {t('agentChat.switchSession')}
            </button>
            <button
              type="button"
              onClick={() => void loadHistory(sessionId)}
              disabled={historyLoading || loading}
              className="flex items-center gap-1.5 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-50 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800"
            >
              <RefreshCw className={cn('size-3.5', historyLoading && 'animate-spin')} />
              {t('agentChat.reloadHistory')}
            </button>
            <button
              type="button"
              onClick={startFresh}
              disabled={loading}
              className="rounded-md border border-sky-200 bg-sky-50 px-3 py-1.5 text-sm font-medium text-sky-700 transition-colors hover:bg-sky-100 disabled:opacity-50 dark:border-sky-800 dark:bg-sky-900/30 dark:text-sky-400"
            >
              {t('agentChat.newSession')}
            </button>
            <button
              type="button"
              onClick={() => setFocusMode((v) => !v)}
              className="flex items-center gap-1.5 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800"
              title={focusMode ? t('agentChat.exitFocus') : t('agentChat.focusMode')}
            >
              {focusMode ? <Minimize2 className="size-3.5" /> : <Maximize2 className="size-3.5" />}
              {focusMode ? t('agentChat.exitFocus') : t('agentChat.focusMode')}
            </button>
          </div>
        </div>
        {historyTruncated && (
          <p className="mt-2 text-xs text-amber-600 dark:text-amber-400">{t('agentChat.historyTruncated')}</p>
        )}
        {followingLiveLog && !loading && (
          <p className="mt-2 text-xs text-sky-600 dark:text-sky-400">{t('agentChat.followingLiveLog')}</p>
        )}
        {sessionEditorOpen && (
          <div className="mt-3 rounded-xl border border-sky-200/70 bg-sky-50/40 p-3 dark:border-sky-800/50 dark:bg-sky-950/20">
            <label className="block text-xs font-medium text-sky-800 dark:text-sky-300">
              {t('agentChat.sessionIdLabel')}
            </label>
            <div className="mt-2 flex flex-col gap-2 sm:flex-row">
              <input
                autoFocus
                value={sessionDraft}
                onChange={(e) => setSessionDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') switchSession()
                  if (e.key === 'Escape') setSessionEditorOpen(false)
                }}
                placeholder={t('agentChat.sessionIdPlaceholder')}
                className="min-w-0 flex-1 rounded-lg border border-neutral-200 bg-white px-3 py-2 font-mono text-sm text-neutral-800 outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200"
              />
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={switchSession}
                  className="rounded-lg bg-sky-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-sky-700"
                >
                  {t('agentChat.switchSessionConfirm')}
                </button>
                <button
                  type="button"
                  onClick={() => setSessionEditorOpen(false)}
                  className="rounded-lg px-3 py-2 text-sm font-medium text-neutral-500 transition-colors hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
                >
                  {t('forms.cancel')}
                </button>
              </div>
            </div>
            <p className="mt-2 text-xs text-sky-700/80 dark:text-sky-300/80">{t('agentChat.sessionSwitchHint')}</p>
          </div>
        )}
        {error && (
          <p className="mt-2 text-xs text-red-500 dark:text-red-400">{error}</p>
        )}
      </div>

      <div ref={scrollRef} className="flex-1 overflow-y-auto px-6 py-5">
        {!content && !historyLoading ? (
          <div className="flex h-full flex-col items-center justify-center text-center">
            <Sparkles className="mb-3 size-10 text-neutral-200 dark:text-zinc-700" strokeWidth={1.3} />
            <p className="text-sm text-neutral-500 dark:text-zinc-400">{t('agentChat.empty')}</p>
            <p className="mt-1 text-xs text-neutral-400 dark:text-zinc-500">{t('agentChat.emptyHint')}</p>
          </div>
        ) : (
          <ConversationLog content={content} />
        )}
      </div>

      <div className="shrink-0 border-t border-neutral-200/70 bg-white px-6 py-4 dark:border-zinc-700/50 dark:bg-zinc-950">
        <div className="flex items-end gap-3 rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 focus-within:border-sky-400 dark:border-zinc-700 dark:bg-zinc-900">
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t('agentChat.placeholder')}
            rows={1}
            className="max-h-32 min-h-8 flex-1 resize-none bg-transparent py-1.5 text-sm leading-relaxed text-neutral-800 outline-none placeholder:text-neutral-400 dark:text-zinc-200 dark:placeholder:text-zinc-600"
            onInput={(e) => {
              const el = e.currentTarget
              el.style.height = 'auto'
              el.style.height = Math.min(el.scrollHeight, 128) + 'px'
            }}
          />
          {loading ? (
            <button
              type="button"
              onClick={() => abortRef.current?.abort()}
              className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-red-500 text-white transition-colors hover:bg-red-600"
            >
              <Square className="size-3" fill="currentColor" />
            </button>
          ) : (
            <button
              type="button"
              onClick={() => void send()}
              disabled={!input.trim()}
              className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-sky-600 text-white transition-colors hover:bg-sky-700 disabled:opacity-30"
            >
              <Send className="size-4" />
            </button>
          )}
        </div>
        <p className="mt-2 text-xs text-neutral-400 dark:text-zinc-500">{t('agentChat.sendHint')}</p>
      </div>
    </div>
  )

  if (focusMode) {
    return (
      <div className="fixed inset-0 z-[80] bg-neutral-100/95 p-3 backdrop-blur-sm dark:bg-zinc-950/95 sm:p-6">
        {chatPanel}
      </div>
    )
  }

  return chatPanel
}
