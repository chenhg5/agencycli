import { useParams } from 'react-router-dom'
import { MilestonePanel } from '../../components/project/MilestonePanel'

export default function ProjectMilestonePage() {
  const { projectId } = useParams<{ projectId: string }>()
  if (!projectId) return null
  return (
    <div className="flex h-full flex-col overflow-hidden">
      <MilestonePanel project={projectId} />
    </div>
  )
}
