import { cn } from '../../lib/cn'

type Props = {
  label: string
  className?: string
}

/** Subtle task label chip — neutral, low-contrast styling. */
export function TaskLabel({ label, className }: Props) {
  return (
    <span
      className={cn(
        'rounded border border-neutral-200/80 bg-neutral-50 px-1.5 py-0.5 text-[10px] font-normal text-neutral-500 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-500',
        className,
      )}
    >
      {label}
    </span>
  )
}
