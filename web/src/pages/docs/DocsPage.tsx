import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import {
  BookOpen, ChevronRight, ChevronDown, FileText, FolderOpen, Folder,
  Plus, Search, ArrowLeft, Pencil, Trash2, FolderInput, X, Save,
} from 'lucide-react'
import { apiFetch, apiPost } from '../../lib/api'

type DocEntry = {
  id: string; title: string; filePath: string; index: string
  createdBy: string; tags?: string[]; description?: string
  createdAt: string; updatedAt: string
}
type TreeNode = { name: string; children?: TreeNode[]; docs?: DocEntry[] }

const btn = 'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors'
const btnPrimary = `${btn} bg-sky-600 text-white hover:bg-sky-700`
const btnGhost = `${btn} text-neutral-500 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800`

export default function DocsPage() {
  const { t } = useTranslation()
  const [tree, setTree] = useState<TreeNode | null>(null)
  const [allDocs, setAllDocs] = useState<DocEntry[]>([])
  const [selectedIndex, setSelectedIndex] = useState<string | null>(null)
  const [selectedDoc, setSelectedDoc] = useState<DocEntry | null>(null)
  const [docContent, setDocContent] = useState('')
  const [searchQ, setSearchQ] = useState('')
  const [showAdd, setShowAdd] = useState(false)

  const load = useCallback(async () => {
    const [t, d] = await Promise.all([
      apiFetch<TreeNode>('/api/v1/docs/tree'),
      apiFetch<DocEntry[]>('/api/v1/docs'),
    ])
    setTree(t)
    setAllDocs(d ?? [])
  }, [])

  useEffect(() => { load() }, [load])

  const visibleDocs = useMemo(() => {
    if (!allDocs) return []
    const q = searchQ.toLowerCase()
    let docs = allDocs
    if (selectedIndex !== null) {
      docs = docs.filter(d => d.index === selectedIndex || d.index.startsWith(selectedIndex + '/'))
    }
    if (q) {
      docs = docs.filter(d =>
        d.title.toLowerCase().includes(q) ||
        (d.description ?? '').toLowerCase().includes(q) ||
        (d.tags ?? []).some(t => t.toLowerCase().includes(q)),
      )
    }
    return docs
  }, [allDocs, selectedIndex, searchQ])

  async function openDoc(doc: DocEntry) {
    setSelectedDoc(doc)
    const res = await apiFetch<DocEntry & { content: string }>(`/api/v1/docs/${doc.id}?content=true`)
    setDocContent(res?.content ?? '')
  }

  async function removeDoc(id: string) {
    if (!confirm(t('docs.removeConfirm'))) return
    await apiFetch(`/api/v1/docs/${id}`, { method: 'DELETE' })
    setSelectedDoc(null)
    load()
  }

  return (
    <div className="flex h-full">
      {/* Sidebar - tree */}
      <div className="w-64 shrink-0 border-r border-neutral-200 dark:border-zinc-800 overflow-y-auto bg-neutral-50/50 dark:bg-zinc-950/50">
        <div className="p-3 border-b border-neutral-200 dark:border-zinc-800">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-neutral-400" />
            <input
              value={searchQ} onChange={e => setSearchQ(e.target.value)}
              placeholder={t('docs.search')}
              className="w-full rounded-md border border-neutral-200 bg-white py-1.5 pl-8 pr-3 text-xs outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-900 dark:focus:border-sky-600"
            />
          </div>
        </div>
        <div className="p-2">
          <button
            onClick={() => { setSelectedIndex(null); setSelectedDoc(null) }}
            className={`w-full flex items-center gap-2 rounded-md px-2 py-1.5 text-xs font-medium transition-colors ${
              selectedIndex === null && !selectedDoc
                ? 'bg-sky-500/10 text-sky-700 dark:text-sky-300'
                : 'text-neutral-600 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800'
            }`}
          >
            <BookOpen className="size-3.5" />
            {t('docs.allDocuments')}
            <span className="ml-auto text-[10px] text-neutral-400">{allDocs.length}</span>
          </button>
          {tree && tree.children?.map(node => (
            <TreeItem
              key={node.name} node={node} depth={0}
              selectedIndex={selectedIndex}
              onSelect={idx => { setSelectedIndex(idx); setSelectedDoc(null) }}
            />
          ))}
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 overflow-y-auto">
        {selectedDoc ? (
          <DocViewer
            doc={selectedDoc} content={docContent}
            onBack={() => setSelectedDoc(null)}
            onRemove={() => removeDoc(selectedDoc.id)}
            onUpdated={load}
          />
        ) : (
          <div className="p-6">
            <div className="flex items-center justify-between mb-4">
              <h1 className="text-lg font-semibold text-neutral-900 dark:text-zinc-100">
                {selectedIndex ? selectedIndex.split('/').pop() : t('docs.title')}
              </h1>
              <button onClick={() => setShowAdd(true)} className={btnPrimary}>
                <Plus className="size-3.5" /> {t('docs.addDoc')}
              </button>
            </div>
            {selectedIndex && (
              <button onClick={() => setSelectedIndex(null)} className={`${btnGhost} mb-3`}>
                <ArrowLeft className="size-3" /> {t('docs.allDocuments')}
              </button>
            )}
            {visibleDocs.length === 0 ? (
              <p className="text-sm text-neutral-400 dark:text-zinc-600 py-12 text-center">{t('docs.noDocuments')}</p>
            ) : (
              <div className="grid gap-2">
                {visibleDocs.map(d => (
                  <button
                    key={d.id}
                    onClick={() => openDoc(d)}
                    className="flex items-start gap-3 rounded-lg border border-neutral-200 bg-white p-3 text-left transition-colors hover:border-sky-300 hover:bg-sky-50/30 dark:border-zinc-800 dark:bg-zinc-900 dark:hover:border-sky-700 dark:hover:bg-sky-950/20"
                  >
                    <FileText className="mt-0.5 size-4 shrink-0 text-sky-500" />
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium text-neutral-900 dark:text-zinc-100 truncate">{d.title}</p>
                      <p className="mt-0.5 text-xs text-neutral-400 dark:text-zinc-500 truncate">{d.index}</p>
                      {d.description && (
                        <p className="mt-1 text-xs text-neutral-500 dark:text-zinc-500 line-clamp-2">{d.description}</p>
                      )}
                      <div className="mt-1.5 flex items-center gap-3 text-[10px] text-neutral-400 dark:text-zinc-600">
                        <span>{d.createdBy}</span>
                        <span>{new Date(d.createdAt).toLocaleDateString()}</span>
                        {d.tags?.map(tag => (
                          <span key={tag} className="rounded bg-neutral-100 px-1.5 py-0.5 dark:bg-zinc-800">{tag}</span>
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

function TreeItem({ node, depth, selectedIndex, onSelect }: {
  node: TreeNode; depth: number; selectedIndex: string | null
  onSelect: (idx: string) => void
}) {
  const [open, setOpen] = useState(true)
  const hasChildren = (node.children?.length ?? 0) > 0
  const fullPath = useMemo(() => {
    return node.name
  }, [node.name])

  const buildPath = useCallback((n: TreeNode, parentPath: string): string => {
    return parentPath ? `${parentPath}/${n.name}` : n.name
  }, [])

  const isActive = selectedIndex === fullPath

  return (
    <div style={{ paddingLeft: depth * 8 }}>
      <button
        onClick={() => { hasChildren ? setOpen(!open) : onSelect(fullPath); onSelect(fullPath) }}
        className={`w-full flex items-center gap-1.5 rounded-md px-2 py-1 text-xs transition-colors ${
          isActive
            ? 'bg-sky-500/10 text-sky-700 dark:text-sky-300'
            : 'text-neutral-600 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800'
        }`}
      >
        {hasChildren ? (
          open ? <ChevronDown className="size-3 shrink-0" /> : <ChevronRight className="size-3 shrink-0" />
        ) : <span className="w-3" />}
        {open ? <FolderOpen className="size-3.5 shrink-0 text-amber-500" /> : <Folder className="size-3.5 shrink-0 text-amber-500" />}
        <span className="truncate">{node.name}</span>
        <span className="ml-auto text-[10px] text-neutral-400">{countDocs(node)}</span>
      </button>
      {open && node.children?.map(c => (
        <TreeItem
          key={c.name} node={c} depth={depth + 1}
          selectedIndex={selectedIndex}
          onSelect={idx => onSelect(buildPath(c, fullPath) === idx ? idx : `${fullPath}/${c.name}`)}
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

function DocViewer({ doc, content, onBack, onRemove, onUpdated }: {
  doc: DocEntry; content: string; onBack: () => void; onRemove: () => void; onUpdated: () => void
}) {
  const { t } = useTranslation()
  const [editing, setEditing] = useState(false)
  const [editTitle, setEditTitle] = useState(doc.title)
  const [editDesc, setEditDesc] = useState(doc.description ?? '')
  const [editIndex, setEditIndex] = useState(doc.index)
  const [editTags, setEditTags] = useState((doc.tags ?? []).join(', '))

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

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-2 border-b border-neutral-200 dark:border-zinc-800 px-4 py-2.5">
        <button onClick={onBack} className={btnGhost}><ArrowLeft className="size-3.5" /></button>
        <div className="flex-1 min-w-0">
          {editing ? (
            <input value={editTitle} onChange={e => setEditTitle(e.target.value)}
              className="w-full rounded border border-neutral-200 px-2 py-1 text-sm dark:border-zinc-700 dark:bg-zinc-900" />
          ) : (
            <h2 className="text-sm font-semibold text-neutral-900 dark:text-zinc-100 truncate">{doc.title}</h2>
          )}
          <p className="text-[10px] text-neutral-400 dark:text-zinc-600 truncate mt-0.5">
            {doc.index} · {doc.createdBy} · {new Date(doc.createdAt).toLocaleDateString()}
          </p>
        </div>
        <div className="flex items-center gap-1">
          {editing ? (
            <>
              <button onClick={saveEdit} className={btnPrimary}><Save className="size-3" /> {t('docs.save')}</button>
              <button onClick={() => setEditing(false)} className={btnGhost}><X className="size-3" /> {t('docs.cancel')}</button>
            </>
          ) : (
            <>
              <button onClick={() => setEditing(true)} className={btnGhost}><Pencil className="size-3" /> {t('docs.edit')}</button>
              <button onClick={onRemove} className={`${btnGhost} text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950/20`}>
                <Trash2 className="size-3" /> {t('docs.remove')}
              </button>
            </>
          )}
        </div>
      </div>
      {editing && (
        <div className="border-b border-neutral-200 dark:border-zinc-800 px-4 py-2 grid grid-cols-3 gap-2 text-xs">
          <label className="space-y-1">
            <span className="text-neutral-500">{t('docs.virtualDir')}</span>
            <input value={editIndex} onChange={e => setEditIndex(e.target.value)}
              className="w-full rounded border border-neutral-200 px-2 py-1 dark:border-zinc-700 dark:bg-zinc-900" />
          </label>
          <label className="space-y-1">
            <span className="text-neutral-500">{t('docs.tags')}</span>
            <input value={editTags} onChange={e => setEditTags(e.target.value)} placeholder="tag1, tag2"
              className="w-full rounded border border-neutral-200 px-2 py-1 dark:border-zinc-700 dark:bg-zinc-900" />
          </label>
          <label className="space-y-1">
            <span className="text-neutral-500">{t('docs.description')}</span>
            <input value={editDesc} onChange={e => setEditDesc(e.target.value)}
              className="w-full rounded border border-neutral-200 px-2 py-1 dark:border-zinc-700 dark:bg-zinc-900" />
          </label>
        </div>
      )}
      <div className="flex-1 overflow-y-auto p-6">
        <article className="prose prose-neutral dark:prose-invert prose-sm max-w-none">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
        </article>
      </div>
    </div>
  )
}

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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <form
        onClick={e => e.stopPropagation()}
        onSubmit={submit}
        className="w-full max-w-md rounded-xl border border-neutral-200 bg-white p-5 shadow-xl dark:border-zinc-700 dark:bg-zinc-900"
      >
        <h3 className="text-sm font-semibold mb-4 text-neutral-900 dark:text-zinc-100">{t('docs.addDoc')}</h3>
        <div className="space-y-3 text-xs">
          <label className="block space-y-1">
            <span className="text-neutral-500">{t('docs.filePath')} *</span>
            <input required value={filePath} onChange={e => setFilePath(e.target.value)} placeholder="/path/to/file.md"
              className="w-full rounded-md border border-neutral-200 px-2.5 py-1.5 dark:border-zinc-700 dark:bg-zinc-800" />
          </label>
          <div className="grid grid-cols-2 gap-2">
            <label className="block space-y-1">
              <span className="text-neutral-500">Title</span>
              <input value={title} onChange={e => setTitle(e.target.value)} placeholder="(auto from filename)"
                className="w-full rounded-md border border-neutral-200 px-2.5 py-1.5 dark:border-zinc-700 dark:bg-zinc-800" />
            </label>
            <label className="block space-y-1">
              <span className="text-neutral-500">{t('docs.virtualDir')}</span>
              <input value={index} onChange={e => setIndex(e.target.value)} placeholder="category/subcategory"
                className="w-full rounded-md border border-neutral-200 px-2.5 py-1.5 dark:border-zinc-700 dark:bg-zinc-800" />
            </label>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <label className="block space-y-1">
              <span className="text-neutral-500">{t('docs.createdBy')} *</span>
              <input required value={createdBy} onChange={e => setCreatedBy(e.target.value)}
                className="w-full rounded-md border border-neutral-200 px-2.5 py-1.5 dark:border-zinc-700 dark:bg-zinc-800" />
            </label>
            <label className="block space-y-1">
              <span className="text-neutral-500">{t('docs.tags')}</span>
              <input value={tags} onChange={e => setTags(e.target.value)} placeholder="tag1, tag2"
                className="w-full rounded-md border border-neutral-200 px-2.5 py-1.5 dark:border-zinc-700 dark:bg-zinc-800" />
            </label>
          </div>
          <label className="block space-y-1">
            <span className="text-neutral-500">{t('docs.description')}</span>
            <input value={description} onChange={e => setDescription(e.target.value)}
              className="w-full rounded-md border border-neutral-200 px-2.5 py-1.5 dark:border-zinc-700 dark:bg-zinc-800" />
          </label>
        </div>
        <div className="mt-4 flex justify-end gap-2">
          <button type="button" onClick={onClose} className={btnGhost}>{t('docs.cancel')}</button>
          <button type="submit" disabled={busy} className={btnPrimary}>{busy ? '...' : t('docs.save')}</button>
        </div>
      </form>
    </div>
  )
}
