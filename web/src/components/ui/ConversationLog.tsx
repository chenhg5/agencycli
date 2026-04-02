import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Bot, User, Wrench, Terminal, AlertTriangle, CheckCircle2, Info } from 'lucide-react'
import { cn } from '../../lib/cn'

type ContentBlock =
  | { type: 'text'; text: string }
  | { type: 'tool_use'; id?: string; name: string; input?: unknown }
  | { type: 'tool_result'; tool_use_id?: string; content?: string; is_error?: boolean; output?: string }

type StreamEvent = {
  type: string
  subtype?: string
  session_id?: string
  message?: {
    role?: string
    content?: ContentBlock[] | string
    model?: string
    stop_reason?: string
    usage?: { input_tokens?: number; output_tokens?: number }
  }
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

type ConversationItem =
  | { kind: 'header'; text: string }
  | { kind: 'system'; text: string }
  | { kind: 'human'; text: string }
  | { kind: 'assistant'; blocks: ContentBlock[] }
  | { kind: 'tool_result'; name?: string; content: string; isError: boolean }
  | { kind: 'result'; text: string; cost?: number; turns?: number; isError: boolean }

function parseLog(content: string): ConversationItem[] {
  const items: ConversationItem[] = []
  const lines = content.split('\n')

  for (const raw of lines) {
    const line = raw.trim()
    if (!line) continue

    if (line.startsWith('===')) {
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

    if (ev.type === 'system') {
      const info = ev.subtype === 'init' && ev.session_id
        ? `Session: ${ev.session_id}`
        : ev.subtype || 'system'
      items.push({ kind: 'system', text: info })
      continue
    }

    if (ev.type === 'human' || ev.role === 'human') {
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
                <div className="flex size-6 shrink-0 items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/30">
                  <Bot className="size-3.5 text-violet-700 dark:text-violet-400" strokeWidth={2} />
                </div>
                <div className="min-w-0 flex-1 space-y-2">
                  <p className="text-xs font-medium text-violet-700 dark:text-violet-400">Assistant</p>
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
