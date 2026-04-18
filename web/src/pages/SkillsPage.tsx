import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { BookOpen, ChevronDown, ChevronRight, Puzzle, Save } from 'lucide-react'
import { cn } from '../lib/cn'
import { useApiJson } from '../lib/use-api'
import { apiPut } from '../lib/api'

type SkillRow = { name: string; description?: string }
type SkillDetail = { name: string; description?: string; prompt: string }

function SkillItem({ skill, defaultOpen }: { skill: SkillRow; defaultOpen?: boolean }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(!!defaultOpen)
  const elRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (defaultOpen && elRef.current) {
      elRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }, [defaultOpen])
  const detailState = useApiJson<SkillDetail>(open ? `/api/v1/skills/${encodeURIComponent(skill.name)}` : null, 0)

  const [value, setValue] = useState<string | null>(null)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [preview, setPreview] = useState(false)

  const content = value ?? (detailState.status === 'ok' ? detailState.data.prompt : '')

  const save = useCallback(async () => {
    setSaving(true)
    setSaved(false)
    try {
      await apiPut(`/api/v1/skills/${encodeURIComponent(skill.name)}`, { content })
      setDirty(false)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (e) {
      alert(String(e))
    } finally {
      setSaving(false)
    }
  }, [skill.name, content])

  return (
    <div ref={elRef} className="border-b border-neutral-100 last:border-b-0 dark:border-zinc-700/40">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-3 px-5 py-3.5 text-left transition-colors hover:bg-neutral-50/80 dark:hover:bg-zinc-800/30"
      >
        <Puzzle className="size-4 shrink-0 text-amber-500 dark:text-amber-400" strokeWidth={1.8} />
        <div className="min-w-0 flex-1">
          <p className="font-mono text-sm font-medium text-neutral-900 dark:text-zinc-100">{skill.name}</p>
          {skill.description && (
            <p className="mt-0.5 truncate text-xs text-neutral-400 dark:text-zinc-500">{skill.description}</p>
          )}
        </div>
        {open
          ? <ChevronDown className="size-4 shrink-0 text-neutral-400" strokeWidth={2} />
          : <ChevronRight className="size-4 shrink-0 text-neutral-400" strokeWidth={2} />
        }
      </button>
      {open && (
        <div className="px-5 pb-4">
          {detailState.status === 'loading' && (
            <div className="flex items-center gap-2 py-3">
              <div className="size-3.5 animate-spin rounded-full border-2 border-neutral-200 border-t-sky-600" />
              <span className="text-xs text-neutral-400">{t('api.loading')}</span>
            </div>
          )}
          {detailState.status === 'error' && (
            <p className="py-2 text-xs text-red-500">{detailState.error.message}</p>
          )}
          {detailState.status === 'ok' && (
            <div className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
              <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-2.5 dark:border-zinc-700/40">
                <div className="flex items-center gap-2">
                  <BookOpen className="size-4 text-neutral-400 dark:text-zinc-500" strokeWidth={1.8} />
                  <span className="text-sm font-medium text-neutral-700 dark:text-zinc-300">SKILL.md</span>
                  {dirty && <span className="text-[10px] text-amber-500">●</span>}
                  {saved && <span className="text-[10px] text-emerald-500">{t('prompt.saved')}</span>}
                </div>
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setPreview((p) => !p)}
                    className={cn(
                      'rounded-md px-2 py-1 text-[11px] font-medium transition-colors',
                      preview
                        ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400'
                        : 'text-neutral-400 hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-400',
                    )}
                  >
                    {preview ? t('prompt.edit') : t('prompt.preview')}
                  </button>
                  <button
                    type="button"
                    onClick={save}
                    disabled={saving}
                    className="flex items-center gap-1 rounded-md bg-sky-600 px-2.5 py-1 text-[11px] font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50"
                  >
                    <Save className="size-3" strokeWidth={2} />
                    {saving ? t('prompt.saving') : t('prompt.save')}
                  </button>
                </div>
              </div>
              {preview ? (
                <div className="prose-none max-h-[40vh] overflow-auto p-4 text-sm leading-relaxed text-neutral-800 dark:text-zinc-200">
                  <Markdown remarkPlugins={[remarkGfm]}>{content || '*(empty)*'}</Markdown>
                </div>
              ) : (
                <textarea
                  value={content}
                  onChange={(e) => { setValue(e.target.value); setDirty(true); setSaved(false) }}
                  className="block w-full resize-y bg-transparent p-4 font-mono text-[13px] leading-relaxed text-neutral-800 outline-none placeholder:text-neutral-300 dark:text-zinc-200 dark:placeholder:text-zinc-700"
                  rows={Math.max(6, Math.min(25, content.split('\n').length + 1))}
                  placeholder="Skill prompt..."
                />
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default function SkillsPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const openSkill = searchParams.get('open') ?? ''
  const skillsState = useApiJson<SkillRow[]>('/api/v1/skills', 0)
  const skills = skillsState.status === 'ok' ? (skillsState.data ?? []) : []

  return (
    <div className="animate-fade-in px-8 py-6">
      <div className="pb-5">
        <div className="flex items-center gap-2.5">
          <Puzzle className="size-5 text-amber-500" strokeWidth={1.8} />
          <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">
            {t('skill.catalogTitle')}
          </h1>
          <span className="rounded-full bg-neutral-100 px-2.5 py-0.5 text-xs font-medium text-neutral-500 dark:bg-zinc-800 dark:text-zinc-500">
            {skills.length}
          </span>
        </div>
        <p className="mt-1 text-sm text-neutral-500 dark:text-zinc-500">{t('skill.catalogHint')}</p>
      </div>

      {skillsState.status === 'loading' && (
        <div className="flex items-center gap-2 py-16 justify-center">
          <div className="size-5 animate-spin rounded-full border-2 border-neutral-300 border-t-sky-600 dark:border-zinc-600 dark:border-t-sky-400" />
          <span className="text-sm text-neutral-500">{t('api.loading')}</span>
        </div>
      )}
      {skillsState.status === 'ok' && skills.length === 0 && (
        <div className="flex flex-col items-center justify-center py-24 text-center">
          <div className="mb-4 flex size-16 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-zinc-800/50">
            <Puzzle className="size-7 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-medium text-neutral-600 dark:text-zinc-400">{t('skill.empty')}</p>
        </div>
      )}
      {skills.length > 0 && (
        <div className="overflow-hidden rounded-xl border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
          {skills.map((sk) => (
            <SkillItem key={sk.name} skill={sk} defaultOpen={sk.name === openSkill} />
          ))}
        </div>
      )}
    </div>
  )
}
