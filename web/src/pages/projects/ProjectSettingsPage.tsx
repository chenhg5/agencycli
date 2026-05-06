import { useCallback, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { FileText, Save } from 'lucide-react'
import { cn } from '../../lib/cn'
import { useApiJson } from '../../lib/use-api'
import { apiPut } from '../../lib/api'

type ProjectDetail = { name: string; description: string; repo: string }
type PromptData = { content: string }

export default function ProjectSettingsPage() {
  const { t } = useTranslation()
  const { projectId } = useParams<{ projectId: string }>()

  const detailPath = projectId ? `/api/v1/projects/${encodeURIComponent(projectId)}` : null
  const detailState = useApiJson<ProjectDetail>(detailPath)

  const promptPath = projectId ? `/api/v1/projects/${encodeURIComponent(projectId)}/prompt` : null
  const promptState = useApiJson<PromptData>(promptPath)

  const detail = detailState.status === 'ok' ? detailState.data : null

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="shrink-0 px-6 pt-5 pb-3">
        <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('projectNav.settings')}</h1>
        <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('projectSettings.subtitle')}</p>
      </div>

      <div className="flex-1 overflow-y-auto px-6 pb-6">
        <div className="space-y-6">
          {/* Basic info */}
          {detail && projectId && (
            <BasicInfoEditor
              projectId={projectId}
              name={detail.name}
              initialDescription={detail.description}
              initialRepo={detail.repo}
            />
          )}

          {/* Project prompt */}
          {promptState.status === 'ok' && projectId && (
            <PromptEditor
              label={t('prompt.projectPrompt')}
              apiPath={`/api/v1/projects/${encodeURIComponent(projectId)}/prompt`}
              initialContent={promptState.data.content}
            />
          )}
        </div>
      </div>
    </div>
  )
}

function BasicInfoEditor({
  projectId,
  name,
  initialDescription,
  initialRepo,
}: {
  projectId: string
  name: string
  initialDescription: string
  initialRepo: string
}) {
  const { t } = useTranslation()
  const [description, setDescription] = useState(initialDescription ?? '')
  const [repo, setRepo] = useState(initialRepo ?? '')
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  const save = useCallback(async () => {
    setSaving(true); setSaved(false)
    try {
      await apiPut(`/api/v1/projects/${encodeURIComponent(projectId)}`, { description, repo })
      setDirty(false); setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (e) { alert(String(e)) }
    finally { setSaving(false) }
  }, [projectId, description, repo])

  const change = useCallback((setter: (v: string) => void) => (v: string) => {
    setter(v); setDirty(true); setSaved(false)
  }, [])

  return (
    <section className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="flex items-center justify-between border-b border-neutral-100 px-5 py-3 dark:border-zinc-700/40">
        <div className="flex items-center gap-2">
          <span className="text-sm font-semibold text-neutral-800 dark:text-zinc-200">{t('projectSettings.basicInfo')}</span>
          {dirty && <span className="text-[10px] text-amber-500">●</span>}
          {saved && <span className="text-[10px] text-emerald-500">{t('prompt.saved')}</span>}
        </div>
        <button
          type="button"
          onClick={save}
          disabled={saving || !dirty}
          className="flex items-center gap-1 rounded-md bg-sky-600 px-2.5 py-1 text-[11px] font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50"
        >
          <Save className="size-3" strokeWidth={2} />
          {saving ? t('prompt.saving') : t('prompt.save')}
        </button>
      </div>
      <dl className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
        {/* Name — read-only */}
        <div className="flex items-baseline gap-4 px-5 py-2.5">
          <dt className="w-28 shrink-0 text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('projectSettings.name')}</dt>
          <dd className="font-mono text-sm text-neutral-800 dark:text-zinc-200">{name}</dd>
        </div>
        {/* Description — editable */}
        <div className="flex items-start gap-4 px-5 py-2.5">
          <dt className="w-28 shrink-0 pt-1.5 text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('projectSettings.description')}</dt>
          <dd className="flex-1">
            <input
              type="text"
              value={description}
              onChange={(e) => change(setDescription)(e.target.value)}
              placeholder="—"
              className="w-full rounded-md border border-neutral-200 bg-transparent px-2.5 py-1 text-sm text-neutral-800 outline-none placeholder:text-neutral-400 focus:border-sky-400 focus:ring-1 focus:ring-sky-400/30 dark:border-zinc-700 dark:text-zinc-200 dark:placeholder:text-zinc-600 dark:focus:border-sky-500"
            />
          </dd>
        </div>
        {/* Repo — editable */}
        <div className="flex items-start gap-4 px-5 py-2.5">
          <dt className="w-28 shrink-0 pt-1.5 text-xs font-medium text-neutral-500 dark:text-zinc-500">{t('projectSettings.repo')}</dt>
          <dd className="flex-1">
            <input
              type="text"
              value={repo}
              onChange={(e) => change(setRepo)(e.target.value)}
              placeholder="/path/to/repo"
              className="w-full rounded-md border border-neutral-200 bg-transparent px-2.5 py-1 font-mono text-sm text-neutral-800 outline-none placeholder:text-neutral-400 focus:border-sky-400 focus:ring-1 focus:ring-sky-400/30 dark:border-zinc-700 dark:text-zinc-200 dark:placeholder:text-zinc-600 dark:focus:border-sky-500"
            />
          </dd>
        </div>
      </dl>
    </section>
  )
}

function PromptEditor({ label, apiPath, initialContent }: { label: string; apiPath: string; initialContent: string }) {
  const { t } = useTranslation()
  const [value, setValue] = useState(initialContent)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [preview, setPreview] = useState(false)
  const [saved, setSaved] = useState(false)

  const save = useCallback(async () => {
    setSaving(true); setSaved(false)
    try {
      await apiPut(apiPath, { content: value })
      setDirty(false); setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (e) { alert(String(e)) }
    finally { setSaving(false) }
  }, [apiPath, value])

  const change = useCallback((v: string) => { setValue(v); setDirty(true); setSaved(false) }, [])

  return (
    <section className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="flex items-center justify-between border-b border-neutral-100 px-5 py-3 dark:border-zinc-700/40">
        <div className="flex items-center gap-2">
          <FileText className="size-4 text-neutral-400 dark:text-zinc-500" strokeWidth={1.8} />
          <span className="text-sm font-semibold text-neutral-800 dark:text-zinc-200">{label}</span>
          {dirty && <span className="text-[10px] text-amber-500">●</span>}
          {saved && <span className="text-[10px] text-emerald-500">{t('prompt.saved')}</span>}
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => setPreview((p) => !p)} className={cn(
            'rounded-md px-2 py-1 text-[11px] font-medium transition-colors',
            preview ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400' : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400',
          )}>
            {preview ? t('prompt.edit') : t('prompt.preview')}
          </button>
          <button type="button" onClick={save} disabled={saving} className="flex items-center gap-1 rounded-md bg-sky-600 px-2.5 py-1 text-[11px] font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50">
            <Save className="size-3" strokeWidth={2} />
            {saving ? t('prompt.saving') : t('prompt.save')}
          </button>
        </div>
      </div>
      {preview ? (
        <div className="prose-none max-h-[50vh] overflow-auto p-5 text-sm leading-relaxed text-neutral-800 dark:text-zinc-200">
          <Markdown remarkPlugins={[remarkGfm]}>{value || '*（空）*'}</Markdown>
        </div>
      ) : (
        <textarea
          value={value}
          onChange={(e) => change(e.target.value)}
          className="block w-full resize-y bg-transparent p-5 font-mono text-[13px] leading-relaxed text-neutral-800 outline-none placeholder:text-neutral-300 dark:text-zinc-200 dark:placeholder:text-zinc-700"
          rows={Math.max(8, Math.min(24, value.split('\n').length + 1))}
          placeholder="Markdown prompt..."
        />
      )}
    </section>
  )
}
