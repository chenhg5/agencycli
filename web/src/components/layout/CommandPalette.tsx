import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Command } from 'cmdk'
import {
  BarChart3,
  Briefcase,
  CalendarClock,
  MessageSquareText,
  FolderKanban,
  LayoutDashboard,
  ListTodo,
  MessageSquare,
  Puzzle,
  Search,
  Settings,
  Users,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { apiFetch } from '../../lib/api'
import { useApiJson } from '../../lib/use-api'

type ProjectRow = { name: string }
type ProjectAgent = { name: string; model?: string; team?: string; project?: string }
type ProjectAgents = { project: string; agents: ProjectAgent[] }

type SearchItem = {
  id: string
  label: string
  group: string
  icon: LucideIcon
  to: string
  keywords?: string
}

export function CommandPalette({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        onOpenChange(!open)
        return
      }
      if (e.key === 'Escape' && open) {
        e.preventDefault()
        onOpenChange(false)
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [open, onOpenChange])

  const projectsState = useApiJson<ProjectRow[]>('/api/v1/projects', 0)
  const projects = projectsState.status === 'ok' ? (projectsState.data ?? []) : []
  const [projectAgents, setProjectAgents] = useState<ProjectAgents[]>([])

  useEffect(() => {
    let cancelled = false
    if (projects.length === 0) {
      setProjectAgents([])
      return
    }
    ;(async () => {
      const rows: ProjectAgents[] = []
      for (const project of projects) {
        try {
          const agents = await apiFetch<ProjectAgent[]>(`/api/v1/projects/${encodeURIComponent(project.name)}/agents`)
          rows.push({ project: project.name, agents: agents ?? [] })
        } catch {
          rows.push({ project: project.name, agents: [] })
        }
      }
      if (!cancelled) setProjectAgents(rows)
    })()
    return () => { cancelled = true }
  }, [projects])

  const items = useMemo<SearchItem[]>(() => {
    const nav: SearchItem[] = [
      { id: 'nav-overview', label: t('nav.overview'), group: t('search.groupNav'), icon: LayoutDashboard, to: '/' },
      { id: 'nav-teams', label: t('nav.teams'), group: t('search.groupNav'), icon: Users, to: '/teams' },
      { id: 'nav-projects', label: t('nav.projects'), group: t('search.groupNav'), icon: FolderKanban, to: '/projects' },
      { id: 'nav-workbench', label: t('nav.workbench'), group: t('search.groupNav'), icon: Briefcase, to: '/workbench' },
      { id: 'nav-skills', label: t('nav.skills'), group: t('search.groupNav'), icon: Puzzle, to: '/skills' },
      { id: 'nav-settings', label: t('nav.settings'), group: t('search.groupNav'), icon: Settings, to: '/settings' },
    ]

    const proj: SearchItem[] = projects.flatMap((p) => {
      const base = `/projects/${encodeURIComponent(p.name)}`
      return [
        { id: `p-${p.name}`, label: p.name, group: t('search.groupProjects'), icon: FolderKanban, to: `${base}/tasks`, keywords: `project ${p.name}` },
        { id: `p-${p.name}-tasks`, label: `${p.name} / ${t('projectNav.tasks')}`, group: t('search.groupProjects'), icon: ListTodo, to: `${base}/tasks` },
        { id: `p-${p.name}-messages`, label: `${p.name} / ${t('projectNav.messages')}`, group: t('search.groupProjects'), icon: MessageSquare, to: `${base}/messages` },
        { id: `p-${p.name}-members`, label: `${p.name} / ${t('projectNav.members')}`, group: t('search.groupProjects'), icon: Users, to: `${base}/members` },
        { id: `p-${p.name}-schedule`, label: `${p.name} / ${t('projectNav.schedule')}`, group: t('search.groupProjects'), icon: CalendarClock, to: `${base}/schedule` },
        { id: `p-${p.name}-runs`, label: `${p.name} / ${t('projectNav.runs')}`, group: t('search.groupProjects'), icon: BarChart3, to: `${base}/runs` },
      ]
    })

    const memberGroup = t('search.groupMembers')
    const members: SearchItem[] = projectAgents.flatMap(({ project, agents }) => {
      const base = `/projects/${encodeURIComponent(project)}/members`
      return agents.flatMap((agent) => {
        const agentName = agent.name
        return [
          {
            id: `agent-${project}-${agentName}`,
            label: `${project} / ${agentName}`,
            group: memberGroup,
            icon: Users,
            to: `${base}/${encodeURIComponent(agentName)}`,
            keywords: `agent member ${project} ${agentName} ${agent.model ?? ''} ${agent.team ?? ''}`,
          },
          {
            id: `agent-chat-${project}-${agentName}`,
            label: `${project} / ${agentName} / ${t('agentChat.title')}`,
            group: memberGroup,
            icon: MessageSquareText,
            to: `${base}/${encodeURIComponent(agentName)}/chat`,
            keywords: `chat conversation session agent member ${project} ${agentName} ${agent.model ?? ''}`,
          },
        ]
      })
    })

    return [...nav, ...proj, ...members]
  }, [t, projects, projectAgents])

  const groups = useMemo(() => {
    const map = new Map<string, SearchItem[]>()
    for (const item of items) {
      const arr = map.get(item.group) ?? []
      arr.push(item)
      map.set(item.group, arr)
    }
    return map
  }, [items])

  const select = useCallback(
    (to: string) => {
      onOpenChange(false)
      navigate(to)
    },
    [navigate, onOpenChange],
  )

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/30 backdrop-blur-[2px] animate-fade-in dark:bg-black/50"
        onClick={() => onOpenChange(false)}
      />
      {/* Panel */}
      <div className="relative w-full max-w-lg animate-scale-in">
        <Command
          className="overflow-hidden rounded-xl border border-neutral-200/80 bg-white shadow-2xl dark:border-zinc-700/80 dark:bg-zinc-900"
          label={t('search.placeholder')}
        >
          <div className="flex items-center gap-2 border-b border-neutral-200/80 px-4 dark:border-zinc-700/60">
            <Search className="size-4 shrink-0 text-neutral-400 dark:text-zinc-500" strokeWidth={2} />
            <Command.Input
              autoFocus
              placeholder={t('search.placeholder')}
              className="h-12 w-full bg-transparent text-sm text-neutral-900 outline-none placeholder:text-neutral-400 dark:text-zinc-100 dark:placeholder:text-zinc-600"
            />
            <kbd className="shrink-0 rounded border border-neutral-200 bg-neutral-100 px-1.5 py-0.5 font-mono text-[10px] text-neutral-400 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-500">
              ESC
            </kbd>
          </div>
          <Command.List className="max-h-72 overflow-y-auto p-2">
            <Command.Empty className="py-8 text-center text-sm text-neutral-400 dark:text-zinc-500">
              {t('search.noResults')}
            </Command.Empty>
            {[...groups.entries()].map(([group, groupItems]) => (
              <Command.Group
                key={group}
                heading={group}
                className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-[11px] [&_[cmdk-group-heading]]:font-semibold [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-wider [&_[cmdk-group-heading]]:text-neutral-400 dark:[&_[cmdk-group-heading]]:text-zinc-600"
              >
                {groupItems.map((item) => (
                  <Command.Item
                    key={item.id}
                    value={`${item.label} ${item.keywords ?? ''}`}
                    onSelect={() => select(item.to)}
                    className="flex cursor-pointer items-center gap-2.5 rounded-md px-2 py-2 text-[13px] text-neutral-700 transition-colors select-none data-[selected=true]:bg-sky-500/[0.10] data-[selected=true]:text-sky-800 dark:text-zinc-300 dark:data-[selected=true]:bg-sky-400/[0.12] dark:data-[selected=true]:text-sky-300"
                  >
                    <item.icon className="size-4 shrink-0 opacity-60" strokeWidth={1.8} />
                    <span className="truncate">{item.label}</span>
                  </Command.Item>
                ))}
              </Command.Group>
            ))}
          </Command.List>
        </Command>
      </div>
    </div>
  )
}
