import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '../../lib/cn'

type Props = {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

const btnBase =
  'flex size-8 items-center justify-center rounded-md text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-30'
const btnIdle =
  'text-neutral-600 hover:bg-neutral-100 dark:text-zinc-400 dark:hover:bg-zinc-800'
const btnActive = 'bg-sky-600 font-medium text-white'

function pageRange(current: number, total: number): (number | '...')[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
  const pages: (number | '...')[] = [1]
  const left = Math.max(2, current - 1)
  const right = Math.min(total - 1, current + 1)
  if (left > 2) pages.push('...')
  for (let i = left; i <= right; i++) pages.push(i)
  if (right < total - 1) pages.push('...')
  pages.push(total)
  return pages
}

export function Pagination({ page, totalPages, onPageChange }: Props) {
  if (totalPages <= 1) return null

  return (
    <div className="flex items-center justify-center gap-1 py-3">
      <button
        type="button"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
        className={cn(btnBase, btnIdle)}
      >
        <ChevronLeft className="size-4" strokeWidth={2} />
      </button>
      {pageRange(page, totalPages).map((p, i) =>
        p === '...' ? (
          <span
            key={`e${i}`}
            className="flex size-8 items-center justify-center text-sm text-neutral-400 dark:text-zinc-500"
          >
            …
          </span>
        ) : (
          <button
            key={p}
            type="button"
            onClick={() => onPageChange(p)}
            className={cn(btnBase, p === page ? btnActive : btnIdle)}
          >
            {p}
          </button>
        ),
      )}
      <button
        type="button"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
        className={cn(btnBase, btnIdle)}
      >
        <ChevronRight className="size-4" strokeWidth={2} />
      </button>
    </div>
  )
}
