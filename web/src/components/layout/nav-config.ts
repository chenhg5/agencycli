import {
  BarChart3,
  BookOpen,
  Briefcase,
  CalendarClock,
  FolderKanban,
  FolderOpen,
  LayoutDashboard,
  ListTodo,
  Milestone,
  MessageSquare,
  Puzzle,
  Settings,
  Target,
  User,
  Users,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

/** 一级导航 i18n：`nav.*` */
export type NavKey =
  | 'overview'
  | 'teams'
  | 'projects'
  | 'people'
  | 'workbench'
  | 'goals'
  | 'skills'
  | 'docs'
  | 'files'
  | 'settings'

/** 项目内执行面：`projectNav.*` */
export type ProjectNavKey = 'tasks' | 'goals' | 'milestones' | 'messages' | 'members' | 'schedule' | 'runs' | 'settings'

export type NavItem = {
  to: string
  navKey: NavKey
  icon: LucideIcon
  /** 为 true 时，子路径也算激活（如 /projects/demo/tasks） */
  activePrefix?: string
  /** admin-only nav items are hidden for member users */
  adminOnly?: boolean
}

export type ProjectNavItem = {
  segment: ProjectNavKey
  icon: LucideIcon
  adminOnly?: boolean
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
  { to: '/goals', navKey: 'goals', icon: Target, activePrefix: '/goals' },
  { to: '/people', navKey: 'people', icon: User, activePrefix: '/people' },
  { to: '/teams', navKey: 'teams', icon: Users, activePrefix: '/teams', adminOnly: true },
  { to: '/skills', navKey: 'skills', icon: Puzzle, activePrefix: '/skills', adminOnly: true },
  { to: '/docs', navKey: 'docs', icon: BookOpen, activePrefix: '/docs' },
  { to: '/files', navKey: 'files', icon: FolderOpen, activePrefix: '/files' },
  { to: '/settings', navKey: 'settings', icon: Settings, activePrefix: '/settings' },
]

export const projectSubNav: ProjectNavItem[] = [
  { segment: 'tasks', icon: ListTodo },
  { segment: 'goals', icon: Target },
  { segment: 'milestones', icon: Milestone },
  { segment: 'messages', icon: MessageSquare },
  { segment: 'members', icon: Users },
  { segment: 'schedule', icon: CalendarClock, adminOnly: true },
  { segment: 'runs', icon: BarChart3 },
  { segment: 'settings', icon: Settings, adminOnly: true },
]

export function navKeyFromPath(pathname: string): NavKey {
  if (pathname === '/') return 'overview'
  const seg = pathname.split('/').filter(Boolean)[0]
  const map: Record<string, NavKey> = {
    teams: 'teams',
    projects: 'projects',
    people: 'people',
    goals: 'goals',
    workbench: 'workbench',
    skills: 'skills',
    docs: 'docs',
    files: 'files',
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
    /^\/projects\/[^/]+\/(tasks|goals|milestones|messages|members|schedule|runs|settings)(?:\/|$)/.exec(pathname)
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
