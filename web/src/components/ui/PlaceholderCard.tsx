import { cn } from '../../lib/cn'

export function PlaceholderCard({
  title,
  children,
  className,
}: {
  title: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <section
      className={cn(
        'rounded-lg border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40',
        className,
      )}
    >
      <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
        {title}
      </h2>
      <div className="mt-2 text-sm leading-relaxed text-neutral-500 dark:text-zinc-500">
        {children}
      </div>
    </section>
  )
}
