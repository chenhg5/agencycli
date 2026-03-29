import {
  BarChart3,
  Briefcase,
  CalendarClock,
  FolderKanban,
  LayoutDashboard,
  ListTodo,
  MessageSquare,
  Puzzle,
  Settings,
  Users,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

/** 一级导航 i18n：`nav.*` */
export type NavKey =
  | 'overview'
  | 'teams'
  | 'projects'
  | 'workbench'
  | 'skills'
  | 'settings'

/** 项目内执行面：`projectNav.*` */
export type ProjectNavKey = 'tasks' | 'messages' | 'members' | 'schedule' | 'runs' | 'settings'

export type NavItem = {
  to: string
  navKey: NavKey
  icon: LucideIcon
  /** 为 true 时，子路径也算激活（如 /projects/demo/tasks） */
  activePrefix?: string
}

export type ProjectNavItem = {
  segment: ProjectNavKey
  icon: LucideIcon
}

/** 工作区一级导航（团队 = 编制，项目 = 干活入口） */
export const workspaceNav: NavItem[] = [
  { to: '/', navKey: 'overview', icon: LayoutDashboard, activePrefix: undefined },
  { to: '/workbench', navKey: 'workbench', icon: Briefcase, activePrefix: '/workbench' },
  {
    to: '/projects',
    navKey: 'projects',
    icon: FolderKanban,
    activePrefix: '/projects',
  },
  { to: '/teams', navKey: 'teams', icon: Users, activePrefix: '/teams' },
  { to: '/skills', navKey: 'skills', icon: Puzzle, activePrefix: '/skills' },
  { to: '/settings', navKey: 'settings', icon: Settings, activePrefix: '/settings' },
]

export const projectSubNav: ProjectNavItem[] = [
  { segment: 'tasks', icon: ListTodo },
  { segment: 'messages', icon: MessageSquare },
  { segment: 'members', icon: Users },
  { segment: 'schedule', icon: CalendarClock },
  { segment: 'runs', icon: BarChart3 },
  { segment: 'settings', icon: Settings },
]

export function navKeyFromPath(pathname: string): NavKey {
  if (pathname === '/') return 'overview'
  const seg = pathname.split('/').filter(Boolean)[0]
  const map: Record<string, NavKey> = {
    teams: 'teams',
    projects: 'projects',
    workbench: 'workbench',
    skills: 'skills',
    settings: 'settings',
  }
  return map[seg ?? ''] ?? 'overview'
}

/** `/projects/foo/tasks` → `foo` */
export function projectIdFromPath(pathname: string): string | null {
  const m = /^\/projects\/([^/]+)/.exec(pathname)
  if (!m || m[1] === '') return null
  return decodeURIComponent(m[1])
}

export function projectNavKeyFromPath(pathname: string): ProjectNavKey | null {
  const m =
    /^\/projects\/[^/]+\/(tasks|messages|members|schedule|runs|settings)(?:\/|$)/.exec(pathname)
  return (m?.[1] as ProjectNavKey) ?? null
}

export function isNavActive(
  pathname: string,
  to: string,
  end: boolean,
  activePrefix?: string,
): boolean {
  if (activePrefix) {
    return pathname === activePrefix || pathname.startsWith(`${activePrefix}/`)
  }
  if (end) return pathname === to
  return pathname === to || pathname.startsWith(`${to}/`)
}
