import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Bot, User, Wrench, Terminal, AlertTriangle, CheckCircle2, Info, BrainCircuit } from 'lucide-react'
import { cn } from '../../lib/cn'

type ContentBlock =
  | { type: 'text'; text: string }
  | { type: 'tool_use'; id?: string; name: string; input?: unknown }
  | { type: 'tool_result'; tool_use_id?: string; content?: string; is_error?: boolean; output?: string }

/* eslint-disable @typescript-eslint/no-explicit-any */
type StreamEvent = {
  type: string
  subtype?: string
  session_id?: string
  text?: string
  message?: {
    role?: string
    content?: ContentBlock[] | string
    model?: string
    stop_reason?: string
    usage?: { input_tokens?: number; output_tokens?: number }
  }
  call_id?: string
  tool_call?: Record<string, any>
  result?: string
  total_cost_usd?: number
  cost_usd?: number
  is_error?: boolean
  duration_ms?: number
  num_turns?: number
  usage?: { input_tokens?: number; output_tokens?: number }
  content?: ContentBlock[] | string
  role?: string
}
/* eslint-enable @typescript-eslint/no-explicit-any */

type ConversationItem =
  | { kind: 'header'; text: string }
  | { kind: 'system'; text: string }
  | { kind: 'thinking'; text: string }
  | { kind: 'human'; text: string }
  | { kind: 'assistant'; blocks: ContentBlock[] }
  | { kind: 'tool_result'; name?: string; content: string; isError: boolean }
  | { kind: 'result'; text: string; cost?: number; turns?: number; isError: boolean }

function extractCursorToolInfo(tc: Record<string, unknown>): { name: string; desc: string; input: unknown } | null {
  const toolNames: Record<string, (inner: Record<string, unknown>) => { name: string; desc: string; input: unknown }> = {
    shellToolCall: (inner) => {
      const args = (inner.args || {}) as Record<string, unknown>
      return {
        name: 'Shell',
        desc: (inner.description as string) || (args.command as string) || '',
        input: { command: args.command, ...(args.workingDirectory ? { workingDirectory: args.workingDirectory } : {}) },
      }
    },
    readToolCall: (inner) => {
      const args = (inner.args || {}) as Record<string, unknown>
      return { name: 'Read', desc: (args.filePath as string) || '', input: args }
    },
    editToolCall: (inner) => {
      const args = (inner.args || {}) as Record<string, unknown>
      return { name: 'Edit', desc: (args.filePath as string) || '', input: args }
    },
    writeToolCall: (inner) => {
      const args = (inner.args || {}) as Record<string, unknown>
      return { name: 'Write', desc: (args.filePath as string) || '', input: args }
    },
    grepToolCall: (inner) => {
      const args = (inner.args || {}) as Record<string, unknown>
      return { name: 'Grep', desc: (args.pattern as string) || '', input: args }
    },
    globToolCall: (inner) => {
      const args = (inner.args || {}) as Record<string, unknown>
      return { name: 'Glob', desc: (args.pattern as string) || '', input: args }
    },
    taskToolCall: (inner) => {
      const args = (inner.args || {}) as Record<string, unknown>
      return { name: 'Task', desc: (args.description as string) || '', input: args }
    },
    updateTodosToolCall: (inner) => {
      const args = (inner.args || {}) as Record<string, unknown>
      const todos = args.todos as Array<{ content?: string }> | undefined
      const summary = todos?.slice(0, 3).map((t) => t.content).join(', ') || ''
      return { name: 'TodoList', desc: summary, input: args }
    },
  }
  for (const [key, extract] of Object.entries(toolNames)) {
    if (tc[key]) return extract(tc[key] as Record<string, unknown>)
  }
  const firstKey = Object.keys(tc)[0]
  if (firstKey) return { name: firstKey.replace(/ToolCall$/, ''), desc: '', input: (tc[firstKey] as Record<string, unknown>)?.args }
  return null
}

function extractCursorToolResult(tc: Record<string, unknown>): { content: string; isError: boolean } | null {
  for (const key of Object.keys(tc)) {
    const inner = tc[key] as Record<string, unknown> | undefined
    if (!inner?.result) continue
    const result = inner.result as Record<string, unknown>
    if (result.success) {
      const s = result.success as Record<string, unknown>
      if (key === 'shellToolCall') {
        const parts: string[] = []
        if (s.stdout) parts.push(String(s.stdout))
        if (s.stderr) parts.push(String(s.stderr))
        return { content: parts.join('\n') || `exit ${s.exitCode ?? 0}`, isError: false }
      }
      if (key === 'readToolCall') {
        const text = (s.content as string) || (s.text as string) || ''
        return { content: text ? truncateStr(text, 3000) : '(read ok)', isError: false }
      }
      return { content: JSON.stringify(s, null, 2), isError: false }
    }
    if (result.error) {
      const e = result.error as Record<string, unknown>
      return { content: (e.message as string) || JSON.stringify(e), isError: true }
    }
  }
  return null
}

function isCodexLog(lines: string[]): boolean {
  return lines.some(l => l.includes('OpenAI Codex') || /^model:\s/.test(l.trim()))
}

function parseCodexLog(lines: string[]): ConversationItem[] {
  const items: ConversationItem[] = []
  type Section = 'none' | 'header' | 'user' | 'thinking' | 'exec' | 'exec_output' | 'response' | 'tokens'
  let section: Section = 'none'
  let buf: string[] = []
  let execCmd = ''
  let execExitCode = -1
  let tokensTotal = 0
  let seenResponse = false

  const isNoise = (l: string) =>
    /^\d{4}-\d{2}-\d{2}T.*\s(ERROR|WARN)\s/.test(l) ||
    l.startsWith('warning:') ||
    l.startsWith('WARNING:')

  const flush = () => {
    const text = buf.join('\n').trim()
    buf = []
    if (!text) return
    switch (section) {
      case 'user':
        items.push({ kind: 'human', text })
        break
      case 'thinking':
        items.push({ kind: 'thinking', text })
        break
      case 'exec_output':
        items.push({
          kind: 'tool_result',
          content: text,
          isError: execExitCode !== 0,
        })
        break
      case 'response':
        if (!seenResponse) {
          seenResponse = true
          items.push({ kind: 'assistant', blocks: [{ type: 'text', text }] })
        }
        break
    }
  }

  for (let i = 0; i < lines.length; i++) {
    const raw = lines[i]
    const line = raw.trim()

    if (!line || isNoise(line)) continue

    if (line.startsWith('===')) {
      flush()
      section = 'none'
      const headerText = line.replace(/^=+\s*/, '').replace(/\s*=+$/, '')
      if (headerText) items.push({ kind: 'header', text: headerText })
      continue
    }

    if (line.startsWith('Command:') || line.startsWith('Started:')) {
      flush()
      section = 'none'
      items.push({ kind: 'header', text: line })
      continue
    }

    if (line === '--------') {
      flush()
      section = section === 'header' ? 'none' : 'header'
      continue
    }

    if (section === 'header') {
      if (line.startsWith('model:') || line.startsWith('provider:') || line.startsWith('session id:')) {
        items.push({ kind: 'system', text: line })
      }
      continue
    }

    if (line === 'user') {
      flush()
      section = 'user'
      continue
    }

    if (line === 'exec') {
      flush()
      section = 'exec'
      continue
    }

    if (line === 'codex') {
      flush()
      section = 'response'
      continue
    }

    if (/^tokens used$/i.test(line)) {
      flush()
      section = 'tokens'
      continue
    }

    if (section === 'tokens') {
      const n = parseInt(line.replace(/,/g, ''), 10)
      if (!isNaN(n)) tokensTotal = n
      section = 'none'
      continue
    }

    if (/^\*\*.*\*\*$/.test(line)) {
      flush()
      section = 'thinking'
      buf.push(line)
      continue
    }

    if (section === 'exec') {
      const exitMatch = line.match(/^\s*exited\s+(\d+)\s+in\s+/)
      if (exitMatch) {
        execExitCode = parseInt(exitMatch[1], 10)
        items.push({
          kind: 'assistant',
          blocks: [{ type: 'tool_use', name: 'Shell', input: { command: execCmd } }],
        })
        section = 'exec_output'
        continue
      }
      execCmd = line
      continue
    }

    buf.push(raw)
  }

  flush()

  if (tokensTotal > 0) {
    const lastResult = items.findIndex(it => it.kind === 'result')
    if (lastResult === -1) {
      const lastAssistant = [...items].reverse().find(it => it.kind === 'assistant')
      const resultText = lastAssistant && lastAssistant.kind === 'assistant'
        ? lastAssistant.blocks.find(b => b.type === 'text')?.text || 'Completed'
        : 'Completed'
      items.push({
        kind: 'result',
        text: `${tokensTotal.toLocaleString()} tokens used`,
        isError: false,
      })
      void resultText
    }
  }

  return items
}

function parseLog(content: string): ConversationItem[] {
  const lines = content.split('\n')

  if (isCodexLog(lines)) return parseCodexLog(lines)

  const items: ConversationItem[] = []
  let thinkingBuf = ''

  const flushThinking = () => {
    if (thinkingBuf.trim()) {
      items.push({ kind: 'thinking', text: thinkingBuf.trim() })
    }
    thinkingBuf = ''
  }

  for (const raw of lines) {
    const line = raw.trim()
    if (!line) continue

    if (line.startsWith('===')) {
      flushThinking()
      items.push({ kind: 'header', text: line.replace(/^=+\s*/, '').replace(/\s*=+$/, '') })
      continue
    }

    if (line.startsWith('Command:') || line.startsWith('Started:')) {
      items.push({ kind: 'header', text: line })
      continue
    }

    if (!line.startsWith('{')) continue

    let ev: StreamEvent
    try {
      ev = JSON.parse(line)
    } catch {
      continue
    }

    // --- Thinking deltas (Cursor) ---
    if (ev.type === 'thinking') {
      if (ev.subtype === 'delta' && ev.text) {
        thinkingBuf += ev.text
      } else if (ev.subtype === 'completed') {
        flushThinking()
      }
      continue
    }

    if (thinkingBuf) flushThinking()

    if (ev.type === 'system') {
      const info = ev.subtype === 'init' && ev.session_id
        ? `Session: ${ev.session_id}`
        : ev.subtype || 'system'
      items.push({ kind: 'system', text: info })
      continue
    }

    if (ev.type === 'human' || ev.type === 'user' || ev.role === 'human') {
      const text = typeof ev.content === 'string'
        ? ev.content
        : typeof ev.message?.content === 'string'
          ? ev.message.content
          : Array.isArray(ev.message?.content)
            ? ev.message!.content.filter((b): b is { type: 'text'; text: string } => b.type === 'text').map((b) => b.text).join('\n')
            : Array.isArray(ev.content)
              ? (ev.content as ContentBlock[]).filter((b): b is { type: 'text'; text: string } => b.type === 'text').map((b) => b.text).join('\n')
              : ''
      if (text) items.push({ kind: 'human', text })
      continue
    }

    // --- Tool calls (Cursor format) ---
    if (ev.type === 'tool_call' && ev.tool_call) {
      if (ev.subtype === 'started') {
        const info = extractCursorToolInfo(ev.tool_call)
        if (info) {
          items.push({
            kind: 'assistant',
            blocks: [{
              type: 'tool_use',
              id: ev.call_id,
              name: info.name + (info.desc ? `: ${truncateStr(info.desc, 80)}` : ''),
              input: info.input,
            }],
          })
        }
      } else if (ev.subtype === 'completed') {
        const res = extractCursorToolResult(ev.tool_call)
        if (res && res.content) {
          items.push({
            kind: 'tool_result',
            content: res.content,
            isError: res.isError,
          })
        }
      }
      continue
    }

    if (ev.type === 'assistant') {
      const c = ev.message?.content
      if (Array.isArray(c)) {
        const blocks = c as ContentBlock[]
        const textBlocks = blocks.filter((b) => b.type === 'text')
        const toolUseBlocks = blocks.filter((b) => b.type === 'tool_use')
        const toolResultBlocks = blocks.filter((b) => b.type === 'tool_result')

        if (textBlocks.length > 0 || toolUseBlocks.length > 0) {
          items.push({ kind: 'assistant', blocks: [...textBlocks, ...toolUseBlocks] })
        }
        for (const tr of toolResultBlocks) {
          if (tr.type === 'tool_result') {
            items.push({
              kind: 'tool_result',
              content: tr.content || tr.output || '',
              isError: tr.is_error ?? false,
            })
          }
        }
      } else if (typeof c === 'string' && c) {
        items.push({ kind: 'assistant', blocks: [{ type: 'text', text: c }] })
      }
      continue
    }

    if (ev.type === 'result') {
      flushThinking()
      items.push({
        kind: 'result',
        text: ev.result || (ev.is_error ? 'Error' : 'Completed'),
        cost: ev.total_cost_usd ?? ev.cost_usd,
        turns: ev.num_turns,
        isError: ev.is_error ?? false,
      })
      continue
    }
  }

  flushThinking()
  return items
}

function truncateStr(s: string, max: number): string {
  return s.length > max ? s.slice(0, max) + '…' : s
}

const mdComponents = {
  pre({ children, ...props }: React.ComponentProps<'pre'>) {
    return (
      <pre className="my-2 overflow-auto rounded-md border border-neutral-200/60 bg-neutral-100/80 p-3 text-xs dark:border-zinc-700/40 dark:bg-zinc-900/60" {...props}>
        {children}
      </pre>
    )
  },
  code({ children, className, ...props }: React.ComponentProps<'code'>) {
    const isInline = !className
    if (isInline) {
      return (
        <code className="rounded bg-neutral-100 px-1 py-0.5 text-[0.85em] dark:bg-zinc-800" {...props}>
          {children}
        </code>
      )
    }
    return <code className={className} {...props}>{children}</code>
  },
  p({ children, ...props }: React.ComponentProps<'p'>) {
    return <p className="my-1.5 leading-relaxed" {...props}>{children}</p>
  },
  ul({ children, ...props }: React.ComponentProps<'ul'>) {
    return <ul className="my-1.5 ml-4 list-disc space-y-0.5" {...props}>{children}</ul>
  },
  ol({ children, ...props }: React.ComponentProps<'ol'>) {
    return <ol className="my-1.5 ml-4 list-decimal space-y-0.5" {...props}>{children}</ol>
  },
  li({ children, ...props }: React.ComponentProps<'li'>) {
    return <li className="leading-relaxed" {...props}>{children}</li>
  },
  h1({ children, ...props }: React.ComponentProps<'h1'>) {
    return <h1 className="mt-3 mb-1 text-base font-bold" {...props}>{children}</h1>
  },
  h2({ children, ...props }: React.ComponentProps<'h2'>) {
    return <h2 className="mt-2.5 mb-1 text-sm font-bold" {...props}>{children}</h2>
  },
  h3({ children, ...props }: React.ComponentProps<'h3'>) {
    return <h3 className="mt-2 mb-1 text-sm font-semibold" {...props}>{children}</h3>
  },
  a({ children, ...props }: React.ComponentProps<'a'>) {
    return <a className="text-sky-600 underline decoration-sky-300 hover:decoration-sky-500 dark:text-sky-400 dark:decoration-sky-700 dark:hover:decoration-sky-500" target="_blank" rel="noopener noreferrer" {...props}>{children}</a>
  },
  blockquote({ children, ...props }: React.ComponentProps<'blockquote'>) {
    return <blockquote className="my-1.5 border-l-2 border-neutral-300 pl-3 text-neutral-500 dark:border-zinc-600 dark:text-zinc-400" {...props}>{children}</blockquote>
  },
  table({ children, ...props }: React.ComponentProps<'table'>) {
    return <table className="my-2 w-full text-xs" {...props}>{children}</table>
  },
  th({ children, ...props }: React.ComponentProps<'th'>) {
    return <th className="border border-neutral-200 bg-neutral-50 px-2 py-1 text-left font-semibold dark:border-zinc-700 dark:bg-zinc-800" {...props}>{children}</th>
  },
  td({ children, ...props }: React.ComponentProps<'td'>) {
    return <td className="border border-neutral-200 px-2 py-1 dark:border-zinc-700" {...props}>{children}</td>
  },
} as import('react-markdown').Components

function MdBlock({ text, className }: { text: string; className?: string }) {
  return (
    <div className={cn('prose-none text-sm leading-relaxed text-neutral-800 dark:text-zinc-200', className)}>
      <Markdown remarkPlugins={[remarkGfm]} components={mdComponents}>
        {text}
      </Markdown>
    </div>
  )
}

function ToolInputDisplay({ input }: { input: unknown }) {
  if (input == null) return null
  const str = typeof input === 'string' ? input : JSON.stringify(input, null, 2)
  if (str.length <= 200) {
    return <pre className="mt-1 whitespace-pre-wrap break-all text-[11px] leading-relaxed text-neutral-500 dark:text-zinc-500">{str}</pre>
  }
  return (
    <details className="mt-1">
      <summary className="text-[11px] text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400">
        展开参数
      </summary>
      <pre className="mt-1 max-h-40 overflow-auto whitespace-pre-wrap break-all text-[11px] leading-relaxed text-neutral-500 dark:text-zinc-500">{str}</pre>
    </details>
  )
}

export function ConversationLog({ content }: { content: string }) {
  const { t } = useTranslation()
  const items = useMemo(() => parseLog(content), [content])

  if (items.length === 0) {
    return <p className="py-4 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('runs.logEmpty')}</p>
  }

  return (
    <div className="space-y-3">
      {items.map((item, i) => {
        switch (item.kind) {
          case 'header':
            return (
              <div key={i} className="flex items-center gap-2 text-[11px] text-neutral-400 dark:text-zinc-500">
                <Terminal className="size-3 shrink-0" strokeWidth={1.5} />
                <span className="font-mono">{item.text}</span>
              </div>
            )

          case 'system':
            return (
              <div key={i} className="flex items-center gap-2 rounded-md bg-neutral-50 px-3 py-1.5 dark:bg-zinc-800/40">
                <Info className="size-3.5 shrink-0 text-neutral-400 dark:text-zinc-500" strokeWidth={1.8} />
                <span className="text-xs text-neutral-500 dark:text-zinc-500">{item.text}</span>
              </div>
            )

          case 'thinking':
            return (
              <details key={i} className="group">
                <summary className="flex cursor-pointer items-center gap-2 text-xs text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400">
                  <BrainCircuit className="size-3.5 shrink-0" strokeWidth={1.5} />
                  <span>Thinking</span>
                  <span className="text-[10px] opacity-60">({item.text.length} chars)</span>
                </summary>
                <div className="ml-5 mt-1 max-h-48 overflow-auto rounded-md border border-neutral-200/60 bg-neutral-50/50 px-3 py-2 text-xs leading-relaxed whitespace-pre-wrap text-neutral-500 dark:border-zinc-700/40 dark:bg-zinc-800/20 dark:text-zinc-500">
                  {truncateStr(item.text, 4000)}
                </div>
              </details>
            )

          case 'human':
            return (
              <div key={i} className="flex gap-2.5">
                <div className="flex size-6 shrink-0 items-center justify-center rounded-full bg-sky-100 dark:bg-sky-900/30">
                  <User className="size-3.5 text-sky-700 dark:text-sky-400" strokeWidth={2} />
                </div>
                <div className="min-w-0 flex-1 rounded-lg bg-sky-50 px-3.5 py-2.5 dark:bg-sky-900/15">
                  <p className="mb-1 text-xs font-medium text-sky-800 dark:text-sky-300">User</p>
                  <MdBlock text={item.text} />
                </div>
              </div>
            )

          case 'assistant':
            return (
              <div key={i} className="flex gap-2.5">
                <div className="flex size-6 shrink-0 items-center justify-center rounded-full bg-sky-100 dark:bg-sky-900/30">
                  <Bot className="size-3.5 text-sky-700 dark:text-sky-400" strokeWidth={2} />
                </div>
                <div className="min-w-0 flex-1 space-y-2">
                  <p className="text-xs font-medium text-sky-700 dark:text-sky-400">Assistant</p>
                  {item.blocks.map((block, bi) => {
                    if (block.type === 'text') {
                      return <MdBlock key={bi} text={block.text} />
                    }
                    if (block.type === 'tool_use') {
                      return (
                        <div
                          key={bi}
                          className="rounded-md border border-amber-200/60 bg-amber-50/50 px-3 py-2 dark:border-amber-800/30 dark:bg-amber-900/10"
                        >
                          <div className="flex items-center gap-1.5">
                            <Wrench className="size-3.5 text-amber-600 dark:text-amber-500" strokeWidth={1.8} />
                            <span className="font-mono text-xs font-semibold text-amber-700 dark:text-amber-400">
                              {block.name}
                            </span>
                          </div>
                          <ToolInputDisplay input={block.input} />
                        </div>
                      )
                    }
                    return null
                  })}
                </div>
              </div>
            )

          case 'tool_result':
            return (
              <div key={i} className="ml-8 flex gap-2">
                <div className={cn(
                  'size-1.5 mt-2 shrink-0 rounded-full',
                  item.isError ? 'bg-red-400' : 'bg-emerald-400',
                )} />
                <div className={cn(
                  'min-w-0 flex-1 rounded-md border px-3 py-2',
                  item.isError
                    ? 'border-red-200/60 bg-red-50/50 dark:border-red-800/30 dark:bg-red-900/10'
                    : 'border-neutral-200/60 bg-neutral-50/50 dark:border-zinc-700/40 dark:bg-zinc-800/20',
                )}>
                  <pre className="max-h-32 overflow-auto whitespace-pre-wrap break-all text-xs leading-relaxed text-neutral-600 dark:text-zinc-400">
                    {truncateStr(item.content, 2000)}
                  </pre>
                </div>
              </div>
            )

          case 'result':
            return (
              <div key={i} className={cn(
                'flex items-start gap-2 rounded-lg border px-3.5 py-2.5',
                item.isError
                  ? 'border-red-200/80 bg-red-50 dark:border-red-800/40 dark:bg-red-900/20'
                  : 'border-emerald-200/80 bg-emerald-50 dark:border-emerald-800/40 dark:bg-emerald-900/20',
              )}>
                {item.isError
                  ? <AlertTriangle className="mt-0.5 size-4 shrink-0 text-red-500" strokeWidth={1.8} />
                  : <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-emerald-600 dark:text-emerald-400" strokeWidth={1.8} />
                }
                <div className="min-w-0 flex-1">
                  <p className="text-xs font-semibold text-neutral-700 dark:text-zinc-300">
                    {item.isError ? 'Error' : 'Result'}
                    {item.turns != null && (
                      <span className="ml-2 font-normal text-neutral-400 dark:text-zinc-500">{item.turns} turns</span>
                    )}
                    {item.cost != null && (
                      <span className="ml-2 font-normal text-neutral-400 dark:text-zinc-500">${item.cost.toFixed(4)}</span>
                    )}
                  </p>
                  <MdBlock text={item.text} className="mt-1" />
                </div>
              </div>
            )

          default:
            return null
        }
      })}
    </div>
  )
}
