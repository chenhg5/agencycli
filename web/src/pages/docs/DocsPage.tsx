import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import type { Components } from 'react-markdown'
import {
  BookOpen, ChevronRight, ChevronDown, FileText, FolderOpen, Folder,
  ListTree, Maximize2, Minimize2, Plus, Search, ArrowLeft, Pencil, Trash2, X, Save, Copy, Check,
  Calendar, User, Tag, FolderTree, Download, PanelLeftClose, PanelLeft,
} from 'lucide-react'
import { apiFetch, apiPost, apiUrl } from '../../lib/api'
import { getStoredToken } from '../../lib/auth'

function stripFrontmatter(md: string): string {
  const trimmed = md.trimStart()
  if (!trimmed.startsWith('---')) return md
  const end = trimmed.indexOf('---', 3)
  if (end === -1) return md
  return trimmed.slice(end + 3).trimStart()
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, '-')
    .replace(/^-+|-+$/g, '')
}

function useLocaleDate() {
  const { i18n } = useTranslation()
  const locale = i18n.language ?? 'en'
  return useCallback((dateStr: string | null | undefined) => {
    if (!dateStr) return '—'
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return '—'
    return new Intl.DateTimeFormat(locale, { year: 'numeric', month: '2-digit', day: '2-digit' }).format(d)
  }, [locale])
}

type DocEntry = {
  id: string; title: string; filePath: string; index: string
  createdBy: string; tags?: string[]; description?: string
  createdAt: string; updatedAt: string
}
type TreeNode = { name: string; children?: TreeNode[]; docs?: DocEntry[] }

const btn = 'inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors'
const btnPrimary = `${btn} bg-sky-600 text-white hover:bg-sky-700`
const btnGhost = `${btn} text-neutral-500 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800`

export default function DocsPage() {
  const { t } = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const fmtDate = useLocaleDate()
  const [tree, setTree] = useState<TreeNode | null>(null)
  const [allDocs, setAllDocs] = useState<DocEntry[]>([])
  const [selectedIndex, setSelectedIndex] = useState<string | null>(null)
  const [selectedDoc, setSelectedDoc] = useState<DocEntry | null>(null)
  const [docContent, setDocContent] = useState('')
  const [searchQ, setSearchQ] = useState('')
  const [showAdd, setShowAdd] = useState(false)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const initialRouteHandled = useRef(false)

  const load = useCallback(async () => {
    const [t, d] = await Promise.all([
      apiFetch<TreeNode>('/api/v1/docs/tree'),
      apiFetch<DocEntry[]>('/api/v1/docs'),
    ])
    setTree(t)
    setAllDocs(d ?? [])
    return d ?? []
  }, [])

  useEffect(() => { load() }, [load])

  // URL -> state: on first load or navigation, resolve /docs/<path> to a doc or index
  useEffect(() => {
    if (allDocs.length === 0) return
    if (initialRouteHandled.current) return
    initialRouteHandled.current = true
    const sub = location.pathname.replace(/^\/docs\/?/, '')
    if (!sub) return
    const decoded = decodeURIComponent(sub)
    // Try to find a doc whose id matches
    const byId = allDocs.find(d => d.id === decoded)
    if (byId) { openDoc(byId); return }
    // Try to find a doc whose index+title slug matches
    const byIndex = allDocs.find(d => d.index === decoded || `${d.index}/${slugify(d.title)}` === decoded)
    if (byIndex) { openDoc(byIndex); return }
    // Treat as index/directory
    const hasDocsUnder = allDocs.some(d => d.index === decoded || d.index.startsWith(decoded + '/'))
    if (hasDocsUnder) { setSelectedIndex(decoded); return }
    // Single doc under this index
    const underIndex = allDocs.filter(d => d.index.startsWith(decoded))
    if (underIndex.length === 1) { openDoc(underIndex[0]); return }
  }, [allDocs, location.pathname]) // eslint-disable-line react-hooks/exhaustive-deps

  function onSearch(q: string) {
    setSearchQ(q)
    if (q) {
      setSelectedDoc(null)
      setSelectedIndex(null)
    }
  }

  const visibleDocs = useMemo(() => {
    if (!allDocs) return []
    const q = searchQ.toLowerCase()
    let docs = allDocs
    if (!q && selectedIndex !== null) {
      docs = docs.filter(d => d.index === selectedIndex || d.index.startsWith(selectedIndex + '/'))
    }
    if (q) {
      docs = docs.filter(d =>
        d.title.toLowerCase().includes(q) ||
        (d.description ?? '').toLowerCase().includes(q) ||
        (d.tags ?? []).some(t => t.toLowerCase().includes(q)) ||
        d.index.toLowerCase().includes(q),
      )
    }
    return docs.slice().sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
  }, [allDocs, selectedIndex, searchQ])

  async function openDoc(doc: DocEntry) {
    setSelectedDoc(doc)
    const slug = doc.index ? `${doc.index}/${slugify(doc.title)}` : slugify(doc.title)
    navigate(`/docs/${slug}`, { replace: true })
    const res = await apiFetch<DocEntry & { content: string }>(`/api/v1/docs/${doc.id}?content=true`)
    setDocContent(res?.content ?? '')
  }

  function goBackToList() {
    setSelectedDoc(null)
    navigate(selectedIndex ? `/docs/${selectedIndex}` : '/docs', { replace: true })
  }

  async function removeDoc(id: string) {
    if (!confirm(t('docs.removeConfirm'))) return
    await apiFetch(`/api/v1/docs/${id}`, { method: 'DELETE' })
    setSelectedDoc(null)
    load()
  }

  return (
    <div className="flex h-full">
      {/* Sidebar */}
      {sidebarOpen && (
        <div className="w-72 shrink-0 border-r border-neutral-200 dark:border-zinc-700/60 overflow-y-auto bg-neutral-50/50 dark:bg-zinc-900/50 flex flex-col">
          <div className="p-3 border-b border-neutral-200 dark:border-zinc-700/60 flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-neutral-400" />
              <input
                value={searchQ} onChange={e => onSearch(e.target.value)}
                placeholder={t('docs.search')}
                className="w-full rounded-lg border border-neutral-200 bg-white py-2 pl-9 pr-3 text-sm outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-900 dark:focus:border-sky-600 dark:text-zinc-200"
              />
            </div>
            <button onClick={() => setSidebarOpen(false)}
              className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
              title={t('docs.collapseSidebar')}>
              <PanelLeftClose className="size-4" />
            </button>
          </div>
          <nav className="p-2 flex-1 overflow-y-auto">
            <button
              onClick={() => { setSelectedIndex(null); setSelectedDoc(null); setSearchQ(''); navigate('/docs', { replace: true }) }}
              className={`w-full flex items-center gap-2 rounded-lg px-2.5 py-2 text-sm font-medium transition-colors ${
                selectedIndex === null && !selectedDoc && !searchQ
                  ? 'bg-sky-500/10 text-sky-700 dark:text-sky-300'
                  : 'text-neutral-600 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800'
              }`}
            >
              <BookOpen className="size-4" />
              {t('docs.allDocuments')}
              <span className="ml-auto text-xs text-neutral-400 dark:text-zinc-500">{allDocs.length}</span>
            </button>
            {tree && tree.children?.map(node => (
              <TreeItem
                key={node.name} node={node} depth={0} parentPath=""
                selectedIndex={selectedIndex}
                onSelect={idx => { setSelectedIndex(idx); setSelectedDoc(null); setSearchQ(''); navigate(`/docs/${idx}`, { replace: true }) }}
              />
            ))}
          </nav>
        </div>
      )}

      {/* Main content */}
      <div className="flex-1 overflow-y-auto">
        {selectedDoc ? (
          <DocViewer
            doc={selectedDoc} content={docContent}
            onBack={goBackToList}
            onRemove={() => removeDoc(selectedDoc.id)}
            onUpdated={load}
            sidebarOpen={sidebarOpen}
            onToggleSidebar={() => setSidebarOpen(v => !v)}
          />
        ) : (
          <div className="p-6 max-w-6xl mx-auto">
            <div className="flex items-center justify-between mb-5">
              <div className="flex items-center gap-2">
                {!sidebarOpen && (
                  <button onClick={() => setSidebarOpen(true)}
                    className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
                    title={t('docs.expandSidebar')}>
                    <PanelLeft className="size-4" />
                  </button>
                )}
                <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">
                  {searchQ
                    ? `${t('docs.searchResults')} "${searchQ}"`
                    : selectedIndex ? selectedIndex.split('/').pop() : t('docs.title')}
                </h1>
                {searchQ && (
                  <button onClick={() => setSearchQ('')} className="rounded-full p-0.5 text-neutral-400 hover:text-neutral-600 dark:hover:text-zinc-300">
                    <X className="size-4" />
                  </button>
                )}
              </div>
              <button onClick={() => setShowAdd(true)} className={btnPrimary}>
                <Plus className="size-4" /> {t('docs.addDoc')}
              </button>
            </div>
            {selectedIndex && !searchQ && (
              <button onClick={() => { setSelectedIndex(null); navigate('/docs', { replace: true }) }} className={`${btnGhost} mb-4`}>
                <ArrowLeft className="size-3.5" /> {t('docs.allDocuments')}
              </button>
            )}
            {visibleDocs.length === 0 ? (
              <p className="text-sm text-neutral-400 dark:text-zinc-500 py-16 text-center">{t('docs.noDocuments')}</p>
            ) : (
              <div className="grid gap-2">
                {visibleDocs.map(d => (
                  <button
                    key={d.id}
                    onClick={() => openDoc(d)}
                    className="flex items-start gap-3 rounded-xl border border-neutral-200 bg-white p-4 text-left transition-all hover:border-sky-300 hover:shadow-sm dark:border-zinc-700/60 dark:bg-zinc-900 dark:hover:border-sky-700"
                  >
                    <FileText className="mt-0.5 size-5 shrink-0 text-sky-500" />
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-semibold text-neutral-900 dark:text-zinc-100">{d.title}</p>
                      <p className="mt-0.5 text-xs text-neutral-400 dark:text-zinc-500">{d.index}</p>
                      {d.description && (
                        <p className="mt-1 text-sm text-neutral-500 dark:text-zinc-400 line-clamp-2">{d.description}</p>
                      )}
                      <div className="mt-2 flex items-center gap-3 text-xs text-neutral-400 dark:text-zinc-500">
                        <span className="flex items-center gap-1"><User className="size-3" />{d.createdBy}</span>
                        <span className="flex items-center gap-1"><Calendar className="size-3" />{fmtDate(d.createdAt)}</span>
                        {d.tags?.map(tag => (
                          <span key={tag} className="rounded-full bg-neutral-100 px-2 py-0.5 dark:bg-zinc-800">{tag}</span>
                        ))}
                      </div>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {showAdd && <AddDocModal onClose={() => setShowAdd(false)} onAdded={() => { setShowAdd(false); load() }} />}
    </div>
  )
}

function TreeItem({ node, depth, parentPath, selectedIndex, onSelect }: {
  node: TreeNode; depth: number; parentPath: string; selectedIndex: string | null
  onSelect: (idx: string) => void
}) {
  const [open, setOpen] = useState(true)
  const hasChildren = (node.children?.length ?? 0) > 0
  const fullPath = parentPath ? `${parentPath}/${node.name}` : node.name
  const isActive = selectedIndex === fullPath

  return (
    <div style={{ paddingLeft: depth * 12 }}>
      <button
        onClick={() => { if (hasChildren) setOpen(!open); onSelect(fullPath) }}
        className={`w-full flex items-center gap-2 rounded-lg px-2.5 py-1.5 text-sm transition-colors ${
          isActive
            ? 'bg-sky-500/10 text-sky-700 dark:text-sky-300 font-medium'
            : 'text-neutral-600 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800'
        }`}
      >
        {hasChildren ? (
          open ? <ChevronDown className="size-3.5 shrink-0" /> : <ChevronRight className="size-3.5 shrink-0" />
        ) : <span className="w-3.5" />}
        {open ? <FolderOpen className="size-4 shrink-0 text-amber-500" /> : <Folder className="size-4 shrink-0 text-amber-500" />}
        <span className="truncate">{node.name}</span>
        <span className="ml-auto text-xs text-neutral-400 dark:text-zinc-500">{countDocs(node)}</span>
      </button>
      {open && node.children?.map(c => (
        <TreeItem
          key={c.name} node={c} depth={depth + 1}
          parentPath={fullPath}
          selectedIndex={selectedIndex}
          onSelect={onSelect}
        />
      ))}
    </div>
  )
}

function countDocs(node: TreeNode): number {
  let n = node.docs?.length ?? 0
  for (const c of node.children ?? []) n += countDocs(c)
  return n
}

/* ─── Code block with copy button ──────────────────────────────────────────── */

function extractText(node: React.ReactNode): string {
  if (node == null || typeof node === 'boolean') return ''
  if (typeof node === 'string' || typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(extractText).join('')
  if (typeof node === 'object' && 'props' in node) return extractText((node as React.ReactElement<{ children?: React.ReactNode }>).props.children)
  return ''
}

function CodeBlock({ className, children, ...props }: React.HTMLAttributes<HTMLElement>) {
  const [copied, setCopied] = useState(false)
  const text = extractText(children).replace(/\n$/, '')
  const isInline = !className && !text.includes('\n')

  if (isInline) {
    return (
      <code className="rounded-md bg-neutral-100 px-1.5 py-0.5 text-[0.9em] font-mono text-rose-600 dark:bg-zinc-800 dark:text-rose-400" {...props}>
        {children}
      </code>
    )
  }

  function copy() {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const lang = (className ?? '').replace('hljs language-', '').replace('language-', '')

  return (
    <div className="group relative">
      {lang && (
        <div className="flex items-center justify-between px-4 pt-2 pb-0 text-[11px] font-mono text-neutral-500 dark:text-zinc-500">
          <span>{lang}</span>
          <button
            onClick={copy}
            className="rounded-md p-1 text-neutral-400 transition hover:bg-neutral-300/50 hover:text-neutral-600 dark:hover:bg-zinc-600/50 dark:hover:text-zinc-300"
            title="Copy"
          >
            {copied ? <Check className="size-3.5 text-green-500" /> : <Copy className="size-3.5" />}
          </button>
        </div>
      )}
      {!lang && (
        <button
          onClick={copy}
          className="absolute right-2 top-2 rounded-md p-1.5 text-neutral-400 opacity-0 transition hover:bg-neutral-300/50 hover:text-neutral-600 group-hover:opacity-100 dark:hover:bg-zinc-600/50 dark:hover:text-zinc-300"
          title="Copy"
        >
          {copied ? <Check className="size-3.5 text-green-500" /> : <Copy className="size-3.5" />}
        </button>
      )}
      <code className={className} {...props}>{children}</code>
    </div>
  )
}

/* ─── Custom markdown components for Notion-like rendering ─────────────────── */

const mdComponents: Components = {
  h1: ({ children }) => {
    const id = slugify(extractText(children))
    return <h1 id={id} className="mt-10 mb-4 text-[2em] font-bold leading-tight tracking-tight text-neutral-900 dark:text-zinc-50 first:mt-0">{children}</h1>
  },
  h2: ({ children }) => {
    const id = slugify(extractText(children))
    return <h2 id={id} className="mt-8 mb-3 text-[1.5em] font-semibold leading-tight tracking-tight text-neutral-900 dark:text-zinc-50 border-b border-neutral-200 pb-2 dark:border-zinc-700/60">{children}</h2>
  },
  h3: ({ children }) => {
    const id = slugify(extractText(children))
    return <h3 id={id} className="mt-6 mb-2 text-[1.25em] font-semibold leading-snug text-neutral-900 dark:text-zinc-50">{children}</h3>
  },
  h4: ({ children }) => {
    const id = slugify(extractText(children))
    return <h4 id={id} className="mt-5 mb-2 text-[1.1em] font-semibold text-neutral-800 dark:text-zinc-100">{children}</h4>
  },
  p: ({ children }) => (
    <p className="my-3 text-base leading-7 text-neutral-700 dark:text-zinc-300">{children}</p>
  ),
  a: ({ href, children }) => (
    <a href={href} target="_blank" rel="noopener noreferrer"
      className="text-sky-600 underline decoration-sky-300/50 underline-offset-2 hover:decoration-sky-500 dark:text-sky-400 dark:decoration-sky-600/40 dark:hover:decoration-sky-400">
      {children}
    </a>
  ),
  blockquote: ({ children }) => (
    <blockquote className="my-4 border-l-[3px] border-neutral-300 pl-4 text-neutral-500 dark:border-zinc-600 dark:text-zinc-400 [&>p]:text-[0.95em]">
      {children}
    </blockquote>
  ),
  ul: ({ children }) => (
    <ul className="my-3 ml-1 list-disc pl-5 space-y-1.5 text-base leading-7 text-neutral-700 dark:text-zinc-300 marker:text-neutral-400 dark:marker:text-zinc-500">
      {children}
    </ul>
  ),
  ol: ({ children }) => (
    <ol className="my-3 ml-1 list-decimal pl-5 space-y-1.5 text-base leading-7 text-neutral-700 dark:text-zinc-300 marker:text-neutral-500 dark:marker:text-zinc-400">
      {children}
    </ol>
  ),
  li: ({ children }) => <li className="pl-1">{children}</li>,
  hr: () => <hr className="my-8 border-neutral-200 dark:border-zinc-700/60" />,
  img: ({ src, alt }) => (
    <span className="my-4 block">
      <img src={src} alt={alt ?? ''} className="max-w-full rounded-lg shadow-sm" loading="lazy" />
      {alt && <span className="mt-2 block text-center text-sm text-neutral-400 dark:text-zinc-500">{alt}</span>}
    </span>
  ),
  table: ({ children }) => (
    <div className="my-5 overflow-x-auto rounded-lg border border-neutral-200 dark:border-zinc-700/60">
      <table className="w-full text-sm">{children}</table>
    </div>
  ),
  thead: ({ children }) => (
    <thead className="bg-neutral-50 dark:bg-zinc-800/50">{children}</thead>
  ),
  th: ({ children }) => (
    <th className="border-b border-neutral-200 px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:border-zinc-700/60 dark:text-zinc-400">
      {children}
    </th>
  ),
  td: ({ children }) => (
    <td className="border-b border-neutral-100 px-4 py-2.5 text-neutral-700 dark:border-zinc-800 dark:text-zinc-300">
      {children}
    </td>
  ),
  pre: ({ children }) => (
    <pre className="my-5 overflow-x-auto rounded-xl bg-neutral-50 px-4 pb-4 pt-1 text-sm leading-6 ring-1 ring-neutral-200 dark:bg-zinc-950 dark:ring-zinc-800">
      {children}
    </pre>
  ),
  code: CodeBlock,
  input: ({ checked, ...props }) => (
    <input
      type="checkbox" checked={checked} readOnly
      className="mr-2 size-4 rounded border-neutral-300 accent-sky-500 dark:border-zinc-600"
      {...props}
    />
  ),
  strong: ({ children }) => <strong className="font-semibold text-neutral-900 dark:text-zinc-100">{children}</strong>,
  em: ({ children }) => <em className="italic text-neutral-600 dark:text-zinc-400">{children}</em>,
}

function CopyPathBtn({ path }: { path: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <button
      onClick={e => {
        e.stopPropagation()
        navigator.clipboard.writeText(path).then(() => { setCopied(true); setTimeout(() => setCopied(false), 1500) })
      }}
      title={path}
      className="rounded p-0.5 text-neutral-300 hover:text-neutral-500 dark:text-zinc-600 dark:hover:text-zinc-400 transition-colors"
    >
      {copied ? <Check className="size-3" /> : <Copy className="size-3" />}
    </button>
  )
}

/* ─── Table of contents ────────────────────────────────────────────────────── */

type TocItem = { level: number; text: string; id: string }

function parseHeadings(md: string): TocItem[] {
  const stripped = stripFrontmatter(md)
  const items: TocItem[] = []
  let inCode = false
  for (const line of stripped.split('\n')) {
    if (line.trimStart().startsWith('```')) { inCode = !inCode; continue }
    if (inCode) continue
    const m = line.match(/^(#{2,3})\s+(.+)$/)
    if (!m) continue
    const level = m[1].length
    const text = m[2].replace(/\[([^\]]*)\]\([^)]*\)/g, '$1').replace(/[*_`#]/g, '').trim()
    if (!text) continue
    items.push({ level, text, id: slugify(text) })
  }
  return items
}

function DocToc({ items, scrollRef }: {
  items: TocItem[]
  scrollRef: React.RefObject<HTMLDivElement | null>
}) {
  const [activeId, setActiveId] = useState('')

  useEffect(() => {
    const container = scrollRef.current
    if (!container || items.length === 0) return
    function onScroll() {
      const cRect = container!.getBoundingClientRect()
      let current = items[0]?.id ?? ''
      for (const item of items) {
        const el = document.getElementById(item.id)
        if (!el) continue
        if (el.getBoundingClientRect().top - cRect.top <= 80) current = item.id
      }
      setActiveId(current)
    }
    container.addEventListener('scroll', onScroll, { passive: true })
    requestAnimationFrame(onScroll)
    return () => container.removeEventListener('scroll', onScroll)
  }, [items, scrollRef])

  const minLevel = Math.min(...items.map(h => h.level))

  function scrollTo(id: string) {
    const el = document.getElementById(id)
    const container = scrollRef.current
    if (!el || !container) return
    const top = el.getBoundingClientRect().top - container.getBoundingClientRect().top + container.scrollTop - 80
    container.scrollTo({ top, behavior: 'smooth' })
  }

  return (
    <nav className="text-[13px] leading-relaxed">
      <ul className="space-y-0.5 border-l border-neutral-200 dark:border-zinc-700/60">
        {items.map((h, i) => (
          <li key={`${h.id}-${i}`} style={{ paddingLeft: (h.level - minLevel) * 12 }}>
            <button
              onClick={() => scrollTo(h.id)}
              className={`block w-full text-left py-1 pl-3 -ml-px border-l-2 transition-colors truncate ${
                activeId === h.id
                  ? 'border-sky-500 text-sky-600 dark:text-sky-400 font-medium'
                  : 'border-transparent text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-300'
              }`}
            >
              {h.text}
            </button>
          </li>
        ))}
      </ul>
    </nav>
  )
}

function FloatingToc({ items, scrollRef }: {
  items: TocItem[]
  scrollRef: React.RefObject<HTMLDivElement | null>
}) {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    if (!open) return
    function onClick(e: MouseEvent) {
      if ((e.target as HTMLElement).closest('[data-floating-toc]')) return
      setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [open])

  if (items.length <= 1) return null

  return (
    <div className="xl:hidden" data-floating-toc>
      <button
        onClick={() => setOpen(v => !v)}
        className={`fixed bottom-6 right-6 z-40 rounded-full p-3 shadow-lg transition-colors ${
          open
            ? 'bg-sky-600 text-white'
            : 'bg-white/80 text-neutral-500 hover:bg-white hover:text-neutral-700 dark:bg-zinc-800/80 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-200'
        } backdrop-blur-sm border border-neutral-200/60 dark:border-zinc-700/60`}
      >
        <ListTree className="size-5" />
      </button>
      {open && (
        <div className="fixed bottom-20 right-6 z-40 w-60 max-h-[60vh] overflow-y-auto rounded-xl border border-neutral-200/60 bg-white/85 p-4 shadow-xl backdrop-blur-md dark:border-zinc-700/60 dark:bg-zinc-900/85">
          <DocToc items={items} scrollRef={scrollRef} />
        </div>
      )}
    </div>
  )
}

/* ─── Document viewer ──────────────────────────────────────────────────────── */

function DocViewer({ doc, content, onBack, onRemove, onUpdated, sidebarOpen, onToggleSidebar }: {
  doc: DocEntry; content: string; onBack: () => void; onRemove: () => void; onUpdated: () => void
  sidebarOpen: boolean; onToggleSidebar: () => void
}) {
  const { t } = useTranslation()
  const fmtDate = useLocaleDate()
  const [editing, setEditing] = useState(false)
  const [fullscreen, setFullscreen] = useState(false)
  const [editTitle, setEditTitle] = useState(doc.title)
  const [editDesc, setEditDesc] = useState(doc.description ?? '')
  const [editIndex, setEditIndex] = useState(doc.index)
  const [editTags, setEditTags] = useState((doc.tags ?? []).join(', '))
  const scrollRef = useRef<HTMLDivElement>(null)
  const tocItems = useMemo(() => parseHeadings(content), [content])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && fullscreen) setFullscreen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [fullscreen])

  async function downloadDoc(d: DocEntry) {
    try {
      const headers: HeadersInit = {}
      const token = getStoredToken()
      if (token) headers['Authorization'] = `Bearer ${token}`
      const resp = await fetch(apiUrl(`/api/v1/docs/${d.id}/download`), { headers })
      if (!resp.ok) return
      const blob = await resp.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = d.filePath.split('/').pop() ?? `${d.title}.md`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
    } catch { /* ignore */ }
  }

  async function saveEdit() {
    await apiFetch(`/api/v1/docs/${doc.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: editTitle,
        description: editDesc,
        index: editIndex,
        tags: editTags.split(',').map(s => s.trim()).filter(Boolean),
      }),
    })
    setEditing(false)
    onUpdated()
  }

  if (fullscreen) {
    return (
      <div className="fixed inset-0 z-50 flex flex-col bg-white dark:bg-zinc-950">
        <div className="flex items-center justify-between px-6 py-3 border-b border-neutral-100 dark:border-zinc-800/60">
          <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100 truncate">{doc.title}</h2>
          <button onClick={() => setFullscreen(false)} className={btnGhost}>
            <Minimize2 className="size-3.5" /> {t('docs.exitFullscreen')}
          </button>
        </div>
        <div className="flex-1 overflow-y-auto" ref={scrollRef}>
          <div className="flex justify-center">
            <article className="w-full max-w-4xl px-10 py-10">
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeHighlight]}
                components={mdComponents}
              >
                {stripFrontmatter(content)}
              </ReactMarkdown>
            </article>
            {tocItems.length > 1 && (
              <aside className="hidden xl:block w-56 shrink-0 py-10 pr-6">
                <div className="sticky top-10">
                  <DocToc items={tocItems} scrollRef={scrollRef} />
                </div>
              </aside>
            )}
          </div>
        </div>
        <FloatingToc items={tocItems} scrollRef={scrollRef} />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center gap-2 border-b border-neutral-200 dark:border-zinc-700/60 px-5 py-3">
        {!sidebarOpen && (
          <button onClick={onToggleSidebar}
            className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300">
            <PanelLeft className="size-4" />
          </button>
        )}
        <button onClick={onBack} className={btnGhost}><ArrowLeft className="size-4" /></button>
        <div className="flex-1 min-w-0">
          {editing ? (
            <input value={editTitle} onChange={e => setEditTitle(e.target.value)}
              className="w-full rounded-lg border border-neutral-200 px-3 py-1.5 text-sm font-semibold dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100" />
          ) : (
            <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100 truncate">{doc.title}</h2>
          )}
        </div>
        <div className="flex items-center gap-1.5">
          {editing ? (
            <>
              <button onClick={saveEdit} className={btnPrimary}><Save className="size-3.5" /> {t('docs.save')}</button>
              <button onClick={() => setEditing(false)} className={btnGhost}><X className="size-3.5" /> {t('docs.cancel')}</button>
            </>
          ) : (
            <>
              <button onClick={() => setFullscreen(true)} className={btnGhost}><Maximize2 className="size-3.5" /> {t('docs.fullscreen')}</button>
              <button onClick={() => downloadDoc(doc)} className={btnGhost}><Download className="size-3.5" /> {t('docs.download')}</button>
              <button onClick={() => setEditing(true)} className={btnGhost}><Pencil className="size-3.5" /> {t('docs.edit')}</button>
              <button onClick={onRemove} className={`${btnGhost} text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950/20`}>
                <Trash2 className="size-3.5" /> {t('docs.remove')}
              </button>
            </>
          )}
        </div>
      </div>

      {/* Meta info bar */}
      <div className="flex items-center gap-5 border-b border-neutral-100 dark:border-zinc-800 px-5 py-2 text-xs text-neutral-400 dark:text-zinc-500">
        <span className="flex items-center gap-1">
          <FolderTree className="size-3.5" /> {doc.index}
          <CopyPathBtn path={doc.filePath} />
        </span>
        <span className="flex items-center gap-1"><User className="size-3.5" /> {doc.createdBy}</span>
        <span className="flex items-center gap-1"><Calendar className="size-3.5" /> {fmtDate(doc.createdAt)}</span>
        {doc.tags && doc.tags.length > 0 && (
          <span className="flex items-center gap-1">
            <Tag className="size-3.5" />
            {doc.tags.map(tag => (
              <span key={tag} className="rounded-full bg-neutral-100 px-2 py-0.5 dark:bg-zinc-800">{tag}</span>
            ))}
          </span>
        )}
      </div>

      {/* Edit meta panel */}
      {editing && (
        <div className="border-b border-neutral-200 dark:border-zinc-700/60 px-5 py-3 grid grid-cols-3 gap-3 text-sm">
          <label className="space-y-1">
            <span className="text-neutral-500 dark:text-zinc-400 text-xs font-medium">{t('docs.virtualDir')}</span>
            <input value={editIndex} onChange={e => setEditIndex(e.target.value)}
              className="w-full rounded-lg border border-neutral-200 px-3 py-1.5 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200" />
          </label>
          <label className="space-y-1">
            <span className="text-neutral-500 dark:text-zinc-400 text-xs font-medium">{t('docs.tags')}</span>
            <input value={editTags} onChange={e => setEditTags(e.target.value)} placeholder="tag1, tag2"
              className="w-full rounded-lg border border-neutral-200 px-3 py-1.5 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200" />
          </label>
          <label className="space-y-1">
            <span className="text-neutral-500 dark:text-zinc-400 text-xs font-medium">{t('docs.description')}</span>
            <input value={editDesc} onChange={e => setEditDesc(e.target.value)}
              className="w-full rounded-lg border border-neutral-200 px-3 py-1.5 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200" />
          </label>
        </div>
      )}

      {/* Document body */}
      <div className="flex-1 overflow-y-auto" ref={scrollRef}>
        <div className="flex justify-center">
          <article className="w-full max-w-5xl px-8 py-8">
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeHighlight]}
              components={mdComponents}
            >
              {stripFrontmatter(content)}
            </ReactMarkdown>
          </article>
          {tocItems.length > 1 && (
            <aside className="hidden xl:block w-56 shrink-0 py-8 pr-6">
              <div className="sticky top-8">
                <DocToc items={tocItems} scrollRef={scrollRef} />
              </div>
            </aside>
          )}
        </div>
      </div>
      <FloatingToc items={tocItems} scrollRef={scrollRef} />
    </div>
  )
}

/* ─── Add document modal ───────────────────────────────────────────────────── */

function AddDocModal({ onClose, onAdded }: { onClose: () => void; onAdded: () => void }) {
  const { t } = useTranslation()
  const [filePath, setFilePath] = useState('')
  const [title, setTitle] = useState('')
  const [index, setIndex] = useState('')
  const [createdBy, setCreatedBy] = useState('human')
  const [tags, setTags] = useState('')
  const [description, setDescription] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    try {
      await apiPost('/api/v1/docs', {
        filePath, title, index, createdBy,
        tags: tags.split(',').map(s => s.trim()).filter(Boolean),
        description,
      })
      onAdded()
    } finally {
      setBusy(false)
    }
  }

  const inputCls = 'w-full rounded-lg border border-neutral-200 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <form
        onClick={e => e.stopPropagation()}
        onSubmit={submit}
        className="w-full max-w-lg rounded-2xl border border-neutral-200 bg-white p-6 shadow-2xl dark:border-zinc-700 dark:bg-zinc-900"
      >
        <h3 className="text-base font-semibold mb-5 text-neutral-900 dark:text-zinc-100">{t('docs.addDoc')}</h3>
        <div className="space-y-3">
          <label className="block space-y-1">
            <span className="text-sm text-neutral-600 dark:text-zinc-400">{t('docs.filePath')} *</span>
            <input required value={filePath} onChange={e => setFilePath(e.target.value)} placeholder="/path/to/file.md"
              className={inputCls} />
          </label>
          <div className="grid grid-cols-2 gap-3">
            <label className="block space-y-1">
              <span className="text-sm text-neutral-600 dark:text-zinc-400">Title</span>
              <input value={title} onChange={e => setTitle(e.target.value)} placeholder="(auto from filename)"
                className={inputCls} />
            </label>
            <label className="block space-y-1">
              <span className="text-sm text-neutral-600 dark:text-zinc-400">{t('docs.virtualDir')}</span>
              <input value={index} onChange={e => setIndex(e.target.value)} placeholder="category/subcategory"
                className={inputCls} />
            </label>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <label className="block space-y-1">
              <span className="text-sm text-neutral-600 dark:text-zinc-400">{t('docs.createdBy')} *</span>
              <input required value={createdBy} onChange={e => setCreatedBy(e.target.value)}
                className={inputCls} />
            </label>
            <label className="block space-y-1">
              <span className="text-sm text-neutral-600 dark:text-zinc-400">{t('docs.tags')}</span>
              <input value={tags} onChange={e => setTags(e.target.value)} placeholder="tag1, tag2"
                className={inputCls} />
            </label>
          </div>
          <label className="block space-y-1">
            <span className="text-sm text-neutral-600 dark:text-zinc-400">{t('docs.description')}</span>
            <input value={description} onChange={e => setDescription(e.target.value)}
              className={inputCls} />
          </label>
        </div>
        <div className="mt-5 flex justify-end gap-2">
          <button type="button" onClick={onClose} className={btnGhost}>{t('docs.cancel')}</button>
          <button type="submit" disabled={busy} className={btnPrimary}>{busy ? '...' : t('docs.save')}</button>
        </div>
      </form>
    </div>
  )
}
