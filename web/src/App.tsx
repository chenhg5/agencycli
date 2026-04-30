import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { useAuth } from './lib/auth'
import LoginPage from './pages/LoginPage'
import WorkbenchPage from './pages/WorkbenchPage'
import OverviewPage from './pages/OverviewPage'
import SettingsPage from './pages/SettingsPage'
import SkillsPage from './pages/SkillsPage'
import { ProjectBranch } from './pages/projects/ProjectBranch'
import ProjectAgentDetailPage from './pages/projects/ProjectAgentDetailPage'
import ProjectAgentChatPage from './pages/projects/ProjectAgentChatPage'
import ProjectMembersPage from './pages/projects/ProjectMembersPage'
import ProjectMessagesPage from './pages/projects/ProjectMessagesPage'
import ProjectRunsPage from './pages/projects/ProjectRunsPage'
import ProjectSchedulePage from './pages/projects/ProjectSchedulePage'
import ProjectsListPage from './pages/projects/ProjectsListPage'
import ProjectSettingsPage from './pages/projects/ProjectSettingsPage'
import ProjectTasksPage from './pages/projects/ProjectTasksPage'
import TeamDetailPage from './pages/teams/TeamDetailPage'
import TeamsPage from './pages/teams/TeamsPage'
import DocsPage from './pages/docs/DocsPage'
import FilesPage from './pages/FilesPage'
import PeoplePage from './pages/PeoplePage'
import PersonDetailPage from './pages/PersonDetailPage'
import OKRPage from './pages/OKRPage'
import ProjectMilestonePage from './pages/projects/ProjectMilestonePage'
import ProjectOKRPage from './pages/projects/ProjectOKRPage'

export default function App() {
  const { token, user, logout } = useAuth()
  const isAdmin = !user || user.role === 'admin'

  useEffect(() => {
    const onExpired = () => logout()
    window.addEventListener('auth-expired', onExpired)
    return () => window.removeEventListener('auth-expired', onExpired)
  }, [logout])

  if (!token) {
    return <LoginPage />
  }

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<OverviewPage />} />
        {isAdmin && <Route path="teams/:teamId" element={<TeamDetailPage />} />}
        {isAdmin && <Route path="teams" element={<TeamsPage />} />}
        <Route path="projects" element={<ProjectsListPage />} />
        <Route path="projects/:projectId" element={<ProjectBranch />}>
          <Route index element={<Navigate to="tasks" replace />} />
          <Route path="tasks" element={<ProjectTasksPage />} />
          <Route path="goals" element={<ProjectOKRPage />} />
          <Route path="milestones" element={<ProjectMilestonePage />} />
          <Route path="messages" element={<ProjectMessagesPage />} />
          <Route path="members" element={<ProjectMembersPage />} />
          <Route path="members/:agentName/chat" element={<ProjectAgentChatPage />} />
          <Route path="members/:agentName" element={<ProjectAgentDetailPage />} />
          {isAdmin && <Route path="schedule" element={<ProjectSchedulePage />} />}
          <Route path="runs" element={<ProjectRunsPage />} />
          {isAdmin && <Route path="settings" element={<ProjectSettingsPage />} />}
        </Route>
        <Route path="goals" element={<OKRPage />} />
        <Route path="people" element={<PeoplePage />} />
        <Route path="people/:username" element={<PersonDetailPage />} />
        <Route path="workbench" element={<WorkbenchPage />} />
        {isAdmin && <Route path="skills" element={<SkillsPage />} />}
        {isAdmin && <Route path="docs/*" element={<DocsPage />} />}
        <Route path="files" element={<FilesPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}
