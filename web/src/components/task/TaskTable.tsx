import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Archive, Pencil, Trash2, X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { useFormatDateTime } from '../../lib/format-datetime'
import { formatGoDuration, taskElapsedLabel } from '../../lib/task-duration'
import {
  type TaskRow,
  statusColor,
  priorityLabel,
  isTerminal,
} from './TaskModals'
import { TaskLabel } from './TaskLabel'

type Props = {
  tasks: TaskRow[]
  checked: Set<string>
  allChecked: boolean
  someChecked: boolean
  onToggleAll: () => void
  onToggleOne: (id: string) => void
  onRowClick: (task: TaskRow) => void
  onEdit: (task: TaskRow) => void
  onCancel: (task: TaskRow, e: React.MouseEvent) => void
  onArchive: (task: TaskRow, e: React.MouseEvent) => void
  onDelete: (task: TaskRow, e: React.MouseEvent) => void
  showProject?: boolean
}

export function TaskTable({
  tasks,
  checked,
  allChecked,
  someChecked,
  onToggleAll,
  onToggleOne,
  onRowClick,
  onEdit,
  onCancel,
  onArchive,
  onDelete,
  showProject = false,
}: Props) {
  const { t } = useTranslation()
  const fmt = useFormatDateTime()

  return (
    <div className="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-zinc-700/60">
      <table className={cn('w-full', showProject ? 'min-w-[1100px]' : 'min-w-[1000px]')}>
        <thead>
          <tr className="border-b border-neutral-200/80 bg-neutral-50/80 dark:border-zinc-700/60 dark:bg-zinc-900/40">
            <th className="w-10 px-3 py-2.5 text-center">
              <input
                type="checkbox"
                checked={allChecked}
                ref={(el) => { if (el) el.indeterminate = someChecked && !allChecked }}
                onChange={onToggleAll}
                className="size-3.5 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600"
              />
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
              {t('api.taskColTitle')}
            </th>
            {showProject && (
              <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
                {t('workbench.filterProject')}
              </th>
            )}
            <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
              {t('tasks.colAssignee')}
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
              {t('api.taskColStatus')}
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
              {t('forms.priority')}
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
              {t('api.taskColUpdated')}
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
              {t('tasks.colEstimate')}
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-zinc-500">
              {t('tasks.colElapsed')}
            </th>
            <th className="sticky right-0 bg-neutral-50/95 px-4 py-2.5 text-right text-xs font-semibold uppercase tracking-wider text-neutral-400 backdrop-blur-sm dark:bg-zinc-900/95 dark:text-zinc-500">
              {t('messages.actions')}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
          {tasks.map((row) => {
            const prio = priorityLabel[row.priority] ?? priorityLabel[2]
            const sCls = statusColor[row.status] ?? statusColor.pending
            const terminal = isTerminal(row.status)
            const isChecked = checked.has(row.id)
            return (
              <tr
                key={row.id}
                onClick={() => onRowClick(row)}
                className={cn(
                  'group cursor-pointer transition-colors duration-100',
                  isChecked
                    ? 'bg-sky-50/60 dark:bg-sky-900/[0.10]'
                    : 'bg-white hover:bg-neutral-50/80 dark:bg-zinc-900/20 dark:hover:bg-zinc-800/30',
                )}
              >
                <td className="w-10 px-3 py-3 text-center align-middle" onClick={(e) => e.stopPropagation()}>
                  <input
                    type="checkbox"
                    checked={isChecked}
                    onChange={() => onToggleOne(row.id)}
                    className="size-3.5 rounded border-neutral-300 accent-sky-600 dark:border-zinc-600"
                  />
                </td>
                <td className="px-4 py-3 align-middle">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className={cn('text-[11px] font-bold', prio.cls)}>{prio.text}</span>
                    <span className="text-[13px] font-medium text-neutral-900 dark:text-zinc-100">{row.title}</span>
                    {row.type && (
                      <span className="rounded border border-neutral-200 bg-neutral-50 px-1.5 py-0.5 text-[10px] font-medium text-neutral-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-500">
                        {t(`forms.taskType.${row.type}`, { defaultValue: row.type })}
                      </span>
                    )}
                    {row.labels?.map((l) => <TaskLabel key={l} label={l} />)}
                    {row.dueDate && (
                      <span className="text-[10px] text-neutral-400 dark:text-zinc-500">{row.dueDate}</span>
                    )}
                    {row.parentId && (
                      <span className="text-[10px] text-neutral-400 dark:text-zinc-500" title={t('tasks.parentTask')}>↳ {row.parentId}</span>
                    )}
                    {row.archived && (
                      <Archive className="size-3.5 text-neutral-400 dark:text-zinc-500" strokeWidth={1.5} />
                    )}
                  </div>
                  <span className="mt-0.5 block font-mono text-[11px] text-neutral-400 dark:text-zinc-500">{row.id}</span>
                </td>
                {showProject && (
                  <td className="whitespace-nowrap px-4 py-3 align-middle">
                    <Link
                      to={`/projects/${encodeURIComponent(row.project)}/tasks`}
                      onClick={(e) => e.stopPropagation()}
                      className="font-mono text-[13px] text-sky-700 underline-offset-2 hover:underline dark:text-sky-400"
                    >
                      {row.project}
                    </Link>
                    <span className="ml-2 font-mono text-[12px] text-neutral-400 dark:text-zinc-500">{row.agent}</span>
                  </td>
                )}
                <td className="whitespace-nowrap px-4 py-3 align-middle font-mono text-[13px] text-neutral-700 dark:text-zinc-400">
                  {row.assignee === 'human' ? (
                    <span className="rounded bg-violet-50 px-1.5 py-0.5 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">human</span>
                  ) : (
                    row.agent
                  )}
                </td>
                <td className="whitespace-nowrap px-4 py-3 align-middle">
                  <span className={cn('inline-block rounded-full px-2.5 py-0.5 text-[11px] font-semibold', sCls)}>
                    {t(`tasks.status.${row.status}`, { defaultValue: row.status })}
                  </span>
                </td>
                <td className="whitespace-nowrap px-4 py-3 align-middle">
                  <span className={cn('text-[12px] font-bold', prio.cls)}>{prio.text}</span>
                </td>
                <td className="whitespace-nowrap px-4 py-3 align-middle text-[13px] text-neutral-500 dark:text-zinc-500">
                  {fmt(row.updatedAt)}
                </td>
                <td className="whitespace-nowrap px-4 py-3 align-middle text-[12px] tabular-nums text-neutral-500 dark:text-zinc-500">
                  {row.estimateDuration ? formatGoDuration(row.estimateDuration) : '—'}
                </td>
                <td className="whitespace-nowrap px-4 py-3 align-middle text-[12px] tabular-nums text-neutral-500 dark:text-zinc-500">
                  {taskElapsedLabel(row) ?? '—'}
                </td>
                <td
                  className="sticky right-0 bg-white/95 px-4 py-3 align-middle backdrop-blur-sm group-hover:bg-neutral-50/95 dark:bg-zinc-900/95 dark:group-hover:bg-zinc-800/95"
                  onClick={(e) => e.stopPropagation()}
                >
                  <div className="flex items-center justify-end gap-1 whitespace-nowrap opacity-0 transition-opacity duration-100 group-hover:opacity-100">
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); onEdit(row) }}
                      className="rounded p-1 text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-700 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
                      title={t('tasks.edit')}
                    >
                      <Pencil className="size-3.5" strokeWidth={1.8} />
                    </button>
                    {!terminal && !row.archived && (
                      <button
                        type="button"
                        onClick={(e) => onCancel(row, e)}
                        className="rounded p-1 text-amber-600 transition-colors hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/30"
                        title={t('tasks.cancel')}
                      >
                        <X className="size-3.5" strokeWidth={1.8} />
                      </button>
                    )}
                    {!row.archived && (
                      <button
                        type="button"
                        onClick={(e) => onArchive(row, e)}
                        className="rounded p-1 text-neutral-500 transition-colors hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800"
                        title={t('tasks.archive')}
                      >
                        <Archive className="size-3.5" strokeWidth={1.8} />
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={(e) => onDelete(row, e)}
                      className="rounded p-1 text-red-500 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30"
                      title={t('tasks.delete')}
                    >
                      <Trash2 className="size-3.5" strokeWidth={1.8} />
                    </button>
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
