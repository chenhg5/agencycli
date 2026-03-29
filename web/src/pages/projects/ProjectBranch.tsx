import { Outlet } from 'react-router-dom'

/** 占位父路由，仅渲染子页面（任务 / 消息 / 成员） */
export function ProjectBranch() {
  return <Outlet />
}
