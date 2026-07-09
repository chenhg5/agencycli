import { useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Archive, Calendar, GripVertical } from 'lucide-react'
import { cn } from '../../lib/cn'
import {
  type TaskRow,
  STATUS_KEYS,
  priorityLabel,
  isTerminal,
} from './TaskModals'
import { TaskLabel } from './TaskLabel'

type Props = {
  tasks: TaskRow[]
  onTaskClick: (task: TaskRow) => void
  onStatusChange?: (task: TaskRow, newStatus: string) => void
  showProject?: boolean
}

const STATUS_ORDER = STATUS_KEYS

const columnBg: Record<string, string> = {
  pending: 'bg-amber-50/40 dark:bg-amber-950/10',
  in_progress: 'bg-sky-50/40 dark:bg-sky-950/10',
  awaiting_confirmation: 'bg-violet-50/40 dark:bg-violet-950/10',
  blocked: 'bg-orange-50/40 dark:bg-orange-950/10',
  done_success: 'bg-emerald-50/40 dark:bg-emerald-950/10',
  done_failed: 'bg-red-50/40 dark:bg-red-950/10',
  cancelled: 'bg-neutral-50/40 dark:bg-zinc-950/10',
}

const dotColor: Record<string, string> = {
  pending: 'bg-amber-500',
  in_progress: 'bg-sky-500',
  awaiting_confirmation: 'bg-violet-500',
  blocked: 'bg-orange-500',
  done_success: 'bg-emerald-500',
  done_failed: 'bg-red-500',
  cancelled: 'bg-neutral-400 dark:bg-zinc-600',
}

export function TaskKanban({ tasks, onTaskClick, onStatusChange, showProject }: Props) {
  const { t } = useTranslation()
  const draggable = !!onStatusChange
  const [dragId, setDragId] = useState<string | null>(null)
  const [dropTarget, setDropTarget] = useState<string | null>(null)
  const counters = useRef<Record<string, number>>({})

  const columns = useMemo(() => {
    const map = new Map<string, TaskRow[]>()
    for (const s of STATUS_ORDER) map.set(s, [])
    for (const task of tasks) {
      const list = map.get(task.status)
      if (list) list.push(task)
    }
    return STATUS_ORDER.map(s => ({ status: s, tasks: map.get(s) ?? [] }))
  }, [tasks])

  function handleDragStart(e: React.DragEvent, task: TaskRow) {
    setDragId(task.id)
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', task.id)
  }

  function handleDragEnd() {
    setDragId(null)
    setDropTarget(null)
    counters.current = {}
  }

  function handleDragEnter(colKey: string) {
    counters.current[colKey] = (counters.current[colKey] ?? 0) + 1
    setDropTarget(colKey)
  }

  function handleDragLeave(colKey: string) {
    counters.current[colKey] = (counters.current[colKey] ?? 0) - 1
    if ((counters.current[colKey] ?? 0) <= 0) {
      counters.current[colKey] = 0
      if (dropTarget === colKey) setDropTarget(null)
    }
  }

  function handleDrop(e: React.DragEvent, colKey: string) {
    e.preventDefault()
    setDropTarget(null)
    counters.current = {}
    const id = e.dataTransfer.getData('text/plain')
    const task = tasks.find(t => t.id === id)
    if (task && task.status !== colKey && onStatusChange) {
      onStatusChange(task, colKey)
    }
  }

  return (
    <div className="flex min-h-[480px] gap-3 overflow-x-auto pb-4">
      {columns.map(col => (
        <div
          key={col.status}
          onDragOver={draggable ? e => e.preventDefault() : undefined}
          onDragEnter={draggable ? () => handleDragEnter(col.status) : undefined}
          onDragLeave={draggable ? () => handleDragLeave(col.status) : undefined}
          onDrop={draggable ? e => handleDrop(e, col.status) : undefined}
          className={cn(
            'flex w-72 shrink-0 flex-col rounded-xl border border-neutral-200/60 dark:border-zinc-700/40',
            columnBg[col.status],
            draggable && dropTarget === col.status && dragId ? 'ring-2 ring-sky-400/50' : '',
          )}
        >
          <div className="flex items-center gap-2 px-3 py-2.5">
            <span className={cn('size-2 rounded-full', dotColor[col.status])} />
            <span className="text-[13px] font-semibold text-neutral-700 dark:text-zinc-300">
              {t(`tasks.status.${col.status}`, { defaultValue: col.status })}
            </span>
            <span className="ml-auto rounded-full bg-neutral-200/60 px-1.5 py-0.5 text-[11px] font-medium text-neutral-500 dark:bg-zinc-700/40 dark:text-zinc-500">
              {col.tasks.length}
            </span>
          </div>
          <div className="flex-1 space-y-2 px-2 pb-2 overflow-y-auto max-h-[calc(100vh-260px)]">
            {col.tasks.length === 0 ? (
              <p className="py-6 text-center text-xs text-neutral-400 dark:text-zinc-600">{t('tasks.emptyColumn')}</p>
            ) : col.tasks.map(task => {
              const prio = priorityLabel[task.priority] ?? priorityLabel[2]
              const isDragging = dragId === task.id
              return (
                <div
                  key={task.id}
                  draggable={draggable}
                  onDragStart={draggable ? e => handleDragStart(e, task) : undefined}
                  onDragEnd={draggable ? handleDragEnd : undefined}
                  onClick={() => onTaskClick(task)}
                  className={cn(
                    'cursor-pointer rounded-lg border border-neutral-200/80 bg-white p-3 shadow-sm transition-all dark:border-zinc-700/60 dark:bg-zinc-900/60',
                    isDragging ? 'opacity-40 scale-[0.97]' : 'hover:shadow-md hover:border-neutral-300 dark:hover:border-zinc-600',
                    draggable && 'cursor-grab active:cursor-grabbing',
                  )}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-start gap-1.5 min-w-0">
                      {draggable && (
                        <GripVertical className="mt-0.5 size-3.5 shrink-0 text-neutral-300 dark:text-zinc-600" strokeWidth={1.5} />
                      )}
                      <span className="text-[13px] font-medium leading-snug text-neutral-800 dark:text-zinc-200 line-clamp-2">{task.title}</span>
                    </div>
                    <span className={cn('shrink-0 text-[11px] font-bold', prio.cls)}>{prio.text}</span>
                  </div>
                  {task.description && (
                    <p className="mt-1 text-[12px] leading-relaxed text-neutral-500 dark:text-zinc-500 line-clamp-2">{task.description}</p>
                  )}
                  <div className="mt-2 flex flex-wrap items-center gap-1.5">
                    {task.type && (
                      <span className="rounded border border-neutral-200 bg-neutral-50 px-1.5 py-0.5 text-[10px] font-medium text-neutral-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-500">
                        {t(`forms.taskType.${task.type}`, { defaultValue: task.type })}
                      </span>
                    )}
                    {task.labels?.map(l => (
                      <TaskLabel key={l} label={l} />
                    ))}
                    {task.archived && <Archive className="size-3 text-neutral-400 dark:text-zinc-600" strokeWidth={1.5} />}
                  </div>
                  <div className="mt-2 flex items-center justify-between text-[11px] text-neutral-400 dark:text-zinc-600">
                    <span className="font-mono">{showProject ? `${task.project}/${task.agent}` : task.agent}</span>
                    <div className="flex items-center gap-2">
                      {task.dueDate && (
                        <span className={cn('flex items-center gap-0.5', isTerminal(task.status) ? '' : isOverdue(task.dueDate) ? 'text-red-500 dark:text-red-400' : '')}>
                          <Calendar className="size-3" strokeWidth={1.5} />
                          {task.dueDate}
                        </span>
                      )}
                      <span className="font-mono">{task.id.slice(-6)}</span>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      ))}
    </div>
  )
}

function isOverdue(dateStr: string): boolean {
  return new Date(dateStr) < new Date(new Date().toISOString().slice(0, 10))
}
