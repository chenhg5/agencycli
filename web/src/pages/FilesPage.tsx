import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Check, ChevronRight, Copy, File, Film, Folder, FolderPlus, Grid3x3, Image,
  List, Music, Trash2, Upload, X, ZoomIn, ZoomOut, Maximize2, Download, Play,
} from 'lucide-react'
import { apiFetch, apiPost, apiUrl } from '../lib/api'
import { getStoredToken } from '../lib/auth'

type FileEntry = {
  name: string; path: string; isDir: boolean
  size: number; modTime: string; mime?: string
}

type ViewMode = 'grid' | 'list'
type FileCategory = 'image' | 'video' | 'audio' | 'other'

const DRAG_MIME = 'application/x-agencycli-file'

function startInternalDrag(e: React.DragEvent, entry: FileEntry) {
  e.dataTransfer.setData(DRAG_MIME, JSON.stringify(entry))
  e.dataTransfer.effectAllowed = 'move'
}

function isInternalDrag(e: React.DragEvent): boolean {
  return e.dataTransfer.types.includes(DRAG_MIME)
}

function getInternalEntry(e: React.DragEvent): FileEntry | null {
  try { return JSON.parse(e.dataTransfer.getData(DRAG_MIME)) } catch { return null }
}

function fileCat(mime: string | undefined): FileCategory {
  if (!mime) return 'other'
  if (mime.startsWith('image/')) return 'image'
  if (mime.startsWith('video/')) return 'video'
  if (mime.startsWith('audio/')) return 'audio'
  return 'other'
}

function fmtSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), units.length - 1)
  return `${(bytes / Math.pow(k, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

function fileUrl(path: string): string {
  const encoded = path.split('/').map(encodeURIComponent).join('/')
  const token = getStoredToken()
  const url = apiUrl(`/api/v1/files/content/${encoded}`)
  return token ? `${url}?_token=${encodeURIComponent(token)}` : url
}

const btn = 'inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors'
const btnPrimary = `${btn} bg-sky-600 text-white hover:bg-sky-700`
const btnGhost = `${btn} text-neutral-500 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800`

function FileIcon({ entry, className = 'size-5' }: { entry: FileEntry; className?: string }) {
  if (entry.isDir) return <Folder className={`${className} text-amber-500`} />
  const cat = fileCat(entry.mime)
  if (cat === 'image') return <Image className={`${className} text-sky-500`} />
  if (cat === 'video') return <Film className={`${className} text-violet-500`} />
  if (cat === 'audio') return <Music className={`${className} text-emerald-500`} />
  return <File className={`${className} text-neutral-400 dark:text-zinc-500`} />
}

function CopyPathBtn({ path, className }: { path: string; className?: string }) {
  const [copied, setCopied] = useState(false)
  function copy(e: React.MouseEvent) {
    e.stopPropagation()
    navigator.clipboard.writeText(path).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }
  return (
    <button onClick={copy} title={path} className={className}>
      {copied ? <Check className="size-3.5 text-emerald-400" /> : <Copy className="size-3.5" />}
    </button>
  )
}

function BreadcrumbDropTarget({ targetDir, onClick, className, onMove, children }: {
  targetDir: string; onClick: () => void; className?: string
  onMove: (entry: FileEntry, dir: string) => void; children: React.ReactNode
}) {
  const [over, setOver] = useState(false)
  return (
    <button onClick={onClick}
      className={`${className} rounded px-1 -mx-1 ${over ? 'ring-2 ring-sky-400 bg-sky-50 dark:bg-sky-900/30' : ''}`}
      onDragOver={e => { if (isInternalDrag(e)) { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; setOver(true) } }}
      onDragEnter={e => { if (isInternalDrag(e)) setOver(true) }}
      onDragLeave={() => setOver(false)}
      onDrop={e => {
        setOver(false)
        const entry = getInternalEntry(e)
        if (entry) { e.preventDefault(); e.stopPropagation(); onMove(entry, targetDir) }
      }}
    >{children}</button>
  )
}

export default function FilesPage() {
  const { t } = useTranslation()
  const [currentPath, setCurrentPath] = useState('')
  const [files, setFiles] = useState<FileEntry[]>([])
  const [viewMode, setViewMode] = useState<ViewMode>(() =>
    (localStorage.getItem('agencycli-files-view') as ViewMode) || 'grid',
  )
  const [preview, setPreview] = useState<FileEntry | null>(null)
  const [showMkdir, setShowMkdir] = useState(false)
  const [dragging, setDragging] = useState(false)
  const dragCounter = useRef(0)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const load = useCallback(async () => {
    const params = currentPath ? `?path=${encodeURIComponent(currentPath)}` : ''
    const data = await apiFetch<FileEntry[]>(`/api/v1/files${params}`)
    setFiles(data ?? [])
  }, [currentPath])

  useEffect(() => { load() }, [load])

  function toggleView(mode: ViewMode) {
    setViewMode(mode)
    localStorage.setItem('agencycli-files-view', mode)
  }

  const breadcrumbs = currentPath ? currentPath.split('/').filter(Boolean) : []

  async function uploadFiles(fileList: FileList) {
    const form = new FormData()
    for (let i = 0; i < fileList.length; i++) form.append('file', fileList[i])
    const params = currentPath ? `?path=${encodeURIComponent(currentPath)}` : ''
    const headers: HeadersInit = {}
    const token = getStoredToken()
    if (token) headers['Authorization'] = `Bearer ${token}`
    await fetch(apiUrl(`/api/v1/files/upload${params}`), { method: 'POST', headers, body: form })
    load()
  }

  async function moveEntry(entry: FileEntry, targetDir: string) {
    if (entry.path === targetDir) return
    const parent = entry.path.includes('/') ? entry.path.substring(0, entry.path.lastIndexOf('/')) : ''
    if (parent === targetDir) return
    await apiPost('/api/v1/files/move', { from: entry.path, to: targetDir })
    load()
  }

  function onDragEnter(e: React.DragEvent) {
    e.preventDefault()
    if (!isInternalDrag(e)) { dragCounter.current++; setDragging(true) }
  }
  function onDragLeave(e: React.DragEvent) {
    if (!isInternalDrag(e)) { dragCounter.current--; if (dragCounter.current <= 0) { dragCounter.current = 0; setDragging(false) } }
  }
  function onDrop(e: React.DragEvent) {
    e.preventDefault(); dragCounter.current = 0; setDragging(false)
    if (isInternalDrag(e)) return
    if (e.dataTransfer.files.length > 0) uploadFiles(e.dataTransfer.files)
  }

  async function deleteEntry(entry: FileEntry) {
    if (!confirm(entry.isDir ? t('files.deleteFolderConfirm', { name: entry.name }) : t('files.deleteConfirm', { name: entry.name }))) return
    const encoded = entry.path.split('/').map(encodeURIComponent).join('/')
    await apiFetch(`/api/v1/files/${encoded}`, { method: 'DELETE' })
    load()
  }

  function openEntry(entry: FileEntry) {
    if (entry.isDir) setCurrentPath(entry.path)
    else setPreview(entry)
  }

  const fmtDate = useCallback((s: string) => {
    const d = new Date(s)
    if (isNaN(d.getTime())) return '—'
    return d.toLocaleDateString(undefined, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  }, [])

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 border-b border-neutral-200 dark:border-zinc-700/60 px-5 py-3">
        <nav className="flex items-center gap-1 text-sm flex-1 min-w-0 overflow-x-auto">
          <BreadcrumbDropTarget targetDir="" onClick={() => setCurrentPath('')}
            className={`shrink-0 font-medium transition-colors ${!currentPath ? 'text-neutral-900 dark:text-zinc-100' : 'text-neutral-500 hover:text-neutral-700 dark:text-zinc-400 dark:hover:text-zinc-200'}`}
            onMove={moveEntry}>
            {t('files.title')}
          </BreadcrumbDropTarget>
          {breadcrumbs.map((seg, i) => {
            const path = breadcrumbs.slice(0, i + 1).join('/')
            const isLast = i === breadcrumbs.length - 1
            return (
              <span key={path} className="flex items-center gap-1 shrink-0">
                <ChevronRight className="size-3.5 text-neutral-300 dark:text-zinc-600" />
                <BreadcrumbDropTarget targetDir={path} onClick={() => setCurrentPath(path)}
                  className={`transition-colors ${isLast ? 'font-medium text-neutral-900 dark:text-zinc-100' : 'text-neutral-500 hover:text-neutral-700 dark:text-zinc-400 dark:hover:text-zinc-200'}`}
                  onMove={moveEntry}>
                  {seg}
                </BreadcrumbDropTarget>
              </span>
            )
          })}
        </nav>
        <div className="flex items-center gap-1.5 shrink-0">
          <input ref={fileInputRef} type="file" multiple className="hidden"
            onChange={e => { if (e.target.files?.length) { uploadFiles(e.target.files); e.target.value = '' } }} />
          <button onClick={() => fileInputRef.current?.click()} className={btnPrimary}>
            <Upload className="size-4" /> {t('files.upload')}
          </button>
          <button onClick={() => setShowMkdir(true)} className={btnGhost}>
            <FolderPlus className="size-4" /> {t('files.newFolder')}
          </button>
          <div className="ml-2 flex rounded-lg border border-neutral-200 dark:border-zinc-700 overflow-hidden">
            <button onClick={() => toggleView('grid')}
              className={`p-1.5 transition-colors ${viewMode === 'grid' ? 'bg-neutral-100 text-neutral-900 dark:bg-zinc-800 dark:text-zinc-100' : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-300'}`}>
              <Grid3x3 className="size-4" />
            </button>
            <button onClick={() => toggleView('list')}
              className={`p-1.5 transition-colors ${viewMode === 'list' ? 'bg-neutral-100 text-neutral-900 dark:bg-zinc-800 dark:text-zinc-100' : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-300'}`}>
              <List className="size-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-5 relative"
        onDragOver={e => e.preventDefault()}
        onDragEnter={onDragEnter}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
      >
        {dragging && (
          <div className="absolute inset-4 z-40 flex items-center justify-center rounded-2xl border-2 border-dashed border-sky-400 bg-sky-500/5 pointer-events-none">
            <p className="text-base font-medium text-sky-600 dark:text-sky-400">{t('files.dropHere')}</p>
          </div>
        )}

        {files.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-neutral-400 dark:text-zinc-500">
            <Folder className="size-12 mb-3 opacity-40" />
            <p className="text-sm">{t('files.empty')}</p>
          </div>
        ) : viewMode === 'grid' ? (
          <GridView files={files} onOpen={openEntry} onDelete={deleteEntry} onMove={moveEntry} />
        ) : (
          <ListView files={files} onOpen={openEntry} onDelete={deleteEntry} onMove={moveEntry} fmtDate={fmtDate} />
        )}
      </div>

      {preview && <PreviewModal entry={preview} onClose={() => setPreview(null)} />}
      {showMkdir && <MkdirModal currentPath={currentPath} onClose={() => setShowMkdir(false)} onCreated={() => { setShowMkdir(false); load() }} />}
    </div>
  )
}

/* ─── Grid view ────────────────────────────────────────────────────────────── */

function GridView({ files, onOpen, onDelete, onMove }: {
  files: FileEntry[]; onOpen: (f: FileEntry) => void; onDelete: (f: FileEntry) => void
  onMove: (entry: FileEntry, dir: string) => void
}) {
  const [dropTarget, setDropTarget] = useState<string | null>(null)
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
      {files.map(f => (
        <div key={f.path}
          draggable
          onDragStart={e => startInternalDrag(e, f)}
          onDragOver={e => {
            if (f.isDir && isInternalDrag(e)) { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; setDropTarget(f.path) }
          }}
          onDragEnter={e => { if (f.isDir && isInternalDrag(e)) setDropTarget(f.path) }}
          onDragLeave={e => {
            const rect = e.currentTarget.getBoundingClientRect()
            if (e.clientX < rect.left || e.clientX > rect.right || e.clientY < rect.top || e.clientY > rect.bottom) {
              if (dropTarget === f.path) setDropTarget(null)
            }
          }}
          onDrop={e => {
            if (!f.isDir) return
            const entry = getInternalEntry(e)
            if (entry) { e.preventDefault(); e.stopPropagation(); onMove(entry, f.path) }
            setDropTarget(null)
          }}
          className={`group relative rounded-xl border bg-white overflow-hidden transition-all cursor-pointer
            dark:bg-zinc-900
            ${dropTarget === f.path && f.isDir
              ? 'border-sky-400 ring-2 ring-sky-300/50 shadow-md dark:border-sky-500 dark:ring-sky-500/30'
              : 'border-neutral-200 hover:border-sky-300 hover:shadow-sm dark:border-zinc-700/60 dark:hover:border-sky-700'}`}
          onClick={() => onOpen(f)}
        >
          <div className="aspect-[4/3] bg-neutral-50 dark:bg-zinc-800/50 flex items-center justify-center overflow-hidden">
            {f.isDir ? (
              <Folder className="size-12 text-amber-400/60" />
            ) : fileCat(f.mime) === 'image' ? (
              <img src={fileUrl(f.path)} alt={f.name} className="w-full h-full object-cover" loading="lazy" />
            ) : fileCat(f.mime) === 'video' ? (
              <div className="relative w-full h-full">
                <video src={fileUrl(f.path)} className="w-full h-full object-cover" preload="metadata" />
                <div className="absolute inset-0 flex items-center justify-center">
                  <div className="rounded-full bg-black/40 p-2"><Play className="size-5 text-white" fill="white" /></div>
                </div>
              </div>
            ) : (
              <FileIcon entry={f} className="size-10 opacity-40" />
            )}
          </div>
          <div className="px-3 py-2">
            <p className="text-sm font-medium text-neutral-900 dark:text-zinc-100 truncate">{f.name}</p>
            <p className="text-xs text-neutral-400 dark:text-zinc-500">{f.isDir ? '' : fmtSize(f.size)}</p>
          </div>
          <div className="absolute top-2 right-2 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <CopyPathBtn path={f.path} className="rounded-lg p-1 bg-black/40 text-white hover:bg-sky-600" />
            <button
              onClick={e => { e.stopPropagation(); onDelete(f) }}
              className="rounded-lg p-1 bg-black/40 text-white hover:bg-red-600"
            >
              <Trash2 className="size-3.5" />
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}

/* ─── List view ────────────────────────────────────────────────────────────── */

function ListView({ files, onOpen, onDelete, onMove, fmtDate }: {
  files: FileEntry[]; onOpen: (f: FileEntry) => void; onDelete: (f: FileEntry) => void
  onMove: (entry: FileEntry, dir: string) => void; fmtDate: (s: string) => string
}) {
  const { t } = useTranslation()
  const [dropTarget, setDropTarget] = useState<string | null>(null)
  return (
    <div className="rounded-xl border border-neutral-200 dark:border-zinc-700/60 overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="bg-neutral-50 dark:bg-zinc-800/50 text-left text-xs font-medium uppercase tracking-wider text-neutral-500 dark:text-zinc-400">
            <th className="px-4 py-2.5">{t('files.name')}</th>
            <th className="px-4 py-2.5 hidden sm:table-cell">{t('files.type')}</th>
            <th className="px-4 py-2.5">{t('files.size')}</th>
            <th className="px-4 py-2.5 hidden md:table-cell">{t('files.modified')}</th>
            <th className="px-4 py-2.5 w-10"></th>
          </tr>
        </thead>
        <tbody>
          {files.map(f => (
            <tr key={f.path} onClick={() => onOpen(f)}
              draggable
              onDragStart={e => startInternalDrag(e, f)}
              onDragOver={e => {
                if (f.isDir && isInternalDrag(e)) { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; setDropTarget(f.path) }
              }}
              onDragEnter={e => { if (f.isDir && isInternalDrag(e)) setDropTarget(f.path) }}
              onDragLeave={e => {
                const rect = e.currentTarget.getBoundingClientRect()
                if (e.clientX < rect.left || e.clientX > rect.right || e.clientY < rect.top || e.clientY > rect.bottom) {
                  if (dropTarget === f.path) setDropTarget(null)
                }
              }}
              onDrop={e => {
                if (!f.isDir) return
                const entry = getInternalEntry(e)
                if (entry) { e.preventDefault(); e.stopPropagation(); onMove(entry, f.path) }
                setDropTarget(null)
              }}
              className={`border-t cursor-pointer transition-colors
                ${dropTarget === f.path && f.isDir
                  ? 'bg-sky-50 border-sky-300 dark:bg-sky-900/20 dark:border-sky-700'
                  : 'border-neutral-100 dark:border-zinc-800 hover:bg-neutral-50 dark:hover:bg-zinc-800/40'}`}
            >
              <td className="px-4 py-2.5">
                <span className="flex items-center gap-2.5">
                  <FileIcon entry={f} />
                  <span className="text-neutral-900 dark:text-zinc-100 truncate max-w-xs">{f.name}</span>
                </span>
              </td>
              <td className="px-4 py-2.5 text-neutral-400 dark:text-zinc-500 hidden sm:table-cell">
                {f.isDir ? t('files.folder') : (f.mime || '—')}
              </td>
              <td className="px-4 py-2.5 text-neutral-400 dark:text-zinc-500">
                {f.isDir ? '—' : fmtSize(f.size)}
              </td>
              <td className="px-4 py-2.5 text-neutral-400 dark:text-zinc-500 hidden md:table-cell">
                {fmtDate(f.modTime)}
              </td>
              <td className="px-4 py-2.5">
                <span className="flex items-center gap-0.5">
                  <CopyPathBtn path={f.path}
                    className="rounded-lg p-1 text-neutral-300 hover:text-sky-500 hover:bg-sky-50 dark:text-zinc-600 dark:hover:text-sky-400 dark:hover:bg-sky-950/20 transition-colors" />
                  <button onClick={e => { e.stopPropagation(); onDelete(f) }}
                    className="rounded-lg p-1 text-neutral-300 hover:text-red-500 hover:bg-red-50 dark:text-zinc-600 dark:hover:text-red-400 dark:hover:bg-red-950/20 transition-colors">
                    <Trash2 className="size-3.5" />
                  </button>
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/* ─── Preview modal ────────────────────────────────────────────────────────── */

function PreviewModal({ entry, onClose }: { entry: FileEntry; onClose: () => void }) {
  const { t } = useTranslation()
  const cat = fileCat(entry.mime)
  const [zoom, setZoom] = useState(1)
  const [pos, setPos] = useState({ x: 0, y: 0 })
  const isDragging = useRef(false)
  const lastPos = useRef({ x: 0, y: 0 })
  const url = fileUrl(entry.path)

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  function onWheel(e: React.WheelEvent) {
    e.stopPropagation()
    setZoom(z => Math.max(0.1, Math.min(10, z + (e.deltaY > 0 ? -0.15 : 0.15))))
  }

  function onMouseDown(e: React.MouseEvent) {
    if (zoom <= 1) return
    isDragging.current = true
    lastPos.current = { x: e.clientX, y: e.clientY }
  }
  function onMouseMove(e: React.MouseEvent) {
    if (!isDragging.current) return
    setPos(p => ({ x: p.x + e.clientX - lastPos.current.x, y: p.y + e.clientY - lastPos.current.y }))
    lastPos.current = { x: e.clientX, y: e.clientY }
  }
  function onMouseUp() { isDragging.current = false }
  function resetZoom() { setZoom(1); setPos({ x: 0, y: 0 }) }

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-black/90" onClick={onClose}>
      {/* Top bar */}
      <div className="flex items-center justify-between px-5 py-3 shrink-0" onClick={e => e.stopPropagation()}>
        <div className="flex items-center gap-3 text-white min-w-0">
          <span className="text-sm font-medium truncate">{entry.name}</span>
          <span className="text-xs text-white/50">{fmtSize(entry.size)}</span>
        </div>
        <div className="flex items-center gap-1">
          {cat === 'image' && (
            <>
              <button onClick={() => setZoom(z => Math.min(10, z + 0.25))}
                className="rounded-lg p-2 text-white/70 hover:text-white hover:bg-white/10 transition-colors">
                <ZoomIn className="size-4" />
              </button>
              <button onClick={() => setZoom(z => Math.max(0.1, z - 0.25))}
                className="rounded-lg p-2 text-white/70 hover:text-white hover:bg-white/10 transition-colors">
                <ZoomOut className="size-4" />
              </button>
              <button onClick={resetZoom}
                className="rounded-lg p-2 text-white/70 hover:text-white hover:bg-white/10 transition-colors">
                <Maximize2 className="size-4" />
              </button>
              <span className="text-xs text-white/50 w-12 text-center">{Math.round(zoom * 100)}%</span>
            </>
          )}
          <CopyPathBtn path={entry.path}
            className="rounded-lg p-2 text-white/70 hover:text-white hover:bg-white/10 transition-colors" />
          <a href={url} download={entry.name} onClick={e => e.stopPropagation()}
            className="rounded-lg p-2 text-white/70 hover:text-white hover:bg-white/10 transition-colors">
            <Download className="size-4" />
          </a>
          <button onClick={onClose}
            className="rounded-lg p-2 text-white/70 hover:text-white hover:bg-white/10 transition-colors">
            <X className="size-5" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 flex items-center justify-center overflow-hidden select-none"
        onClick={e => e.stopPropagation()}
        onWheel={cat === 'image' ? onWheel : undefined}
        onMouseDown={cat === 'image' ? onMouseDown : undefined}
        onMouseMove={cat === 'image' ? onMouseMove : undefined}
        onMouseUp={cat === 'image' ? onMouseUp : undefined}
        onMouseLeave={cat === 'image' ? onMouseUp : undefined}
        style={{ cursor: cat === 'image' ? (zoom > 1 ? (isDragging.current ? 'grabbing' : 'grab') : 'default') : 'default' }}
      >
        {cat === 'image' ? (
          <img src={url} alt={entry.name}
            className="max-h-full max-w-full"
            style={{ transform: `translate(${pos.x}px, ${pos.y}px) scale(${zoom})`, transition: isDragging.current ? 'none' : 'transform 0.15s ease' }}
            draggable={false}
          />
        ) : cat === 'video' ? (
          <video src={url} controls autoPlay className="max-h-[85vh] max-w-[90vw] rounded-lg" />
        ) : cat === 'audio' ? (
          <div className="flex flex-col items-center gap-6">
            <Music className="size-20 text-white/20" />
            <audio src={url} controls autoPlay className="w-80" />
          </div>
        ) : (
          <div className="flex flex-col items-center gap-4 text-white/50">
            <File className="size-16" />
            <p className="text-sm">{t('files.noPreview')}</p>
            <a href={url} download={entry.name}
              className="inline-flex items-center gap-1.5 rounded-lg bg-white/10 px-4 py-2 text-sm text-white hover:bg-white/20 transition-colors">
              <Download className="size-4" /> {t('files.download')}
            </a>
          </div>
        )}
      </div>
    </div>
  )
}

/* ─── Mkdir modal ──────────────────────────────────────────────────────────── */

function MkdirModal({ currentPath, onClose, onCreated }: {
  currentPath: string; onClose: () => void; onCreated: () => void
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    setBusy(true)
    try {
      const path = currentPath ? `${currentPath}/${name.trim()}` : name.trim()
      await apiPost('/api/v1/files/mkdir', { path })
      onCreated()
    } finally { setBusy(false) }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <form onClick={e => e.stopPropagation()} onSubmit={submit}
        className="w-full max-w-sm rounded-2xl border border-neutral-200 bg-white p-6 shadow-2xl dark:border-zinc-700 dark:bg-zinc-900">
        <h3 className="text-base font-semibold mb-4 text-neutral-900 dark:text-zinc-100">{t('files.newFolder')}</h3>
        <input autoFocus value={name} onChange={e => setName(e.target.value)}
          placeholder={t('files.folderName')}
          className="w-full rounded-lg border border-neutral-200 px-3 py-2 text-sm outline-none focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200 dark:focus:border-sky-600" />
        <div className="mt-4 flex justify-end gap-2">
          <button type="button" onClick={onClose} className={btnGhost}>{t('files.cancel')}</button>
          <button type="submit" disabled={busy || !name.trim()} className={btnPrimary}>
            {busy ? '...' : t('files.create')}
          </button>
        </div>
      </form>
    </div>
  )
}
