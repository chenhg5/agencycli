import { useEffect, useState } from 'react'
import { AlertCircle, CheckCircle2, Info, X } from 'lucide-react'
import { cn } from '../../lib/cn'

type ToastItem = { id: number; message: string; type: 'error' | 'success' | 'info' }

let nextId = 0
const listeners = new Set<(t: ToastItem) => void>()

export function showToast(message: string, type: 'error' | 'success' | 'info' = 'error') {
  const item: ToastItem = { id: ++nextId, message, type }
  listeners.forEach((fn) => fn(item))
}

export function ToastContainer() {
  const [items, setItems] = useState<ToastItem[]>([])

  useEffect(() => {
    const handler = (t: ToastItem) => {
      setItems((prev) => [...prev, t])
      setTimeout(() => setItems((prev) => prev.filter((x) => x.id !== t.id)), 4000)
    }
    listeners.add(handler)
    return () => {
      listeners.delete(handler)
    }
  }, [])

  if (items.length === 0) return null

  return (
    <div className="fixed top-4 right-4 z-[100] flex w-80 flex-col gap-2">
      {items.map((t) => (
        <div
          key={t.id}
          className={cn(
            'flex items-start gap-2.5 rounded-lg border px-4 py-3 shadow-lg backdrop-blur-sm animate-[slideInRight_0.3s_ease-out]',
            t.type === 'error' &&
              'border-red-200 bg-red-50/95 dark:border-red-800/60 dark:bg-red-950/90',
            t.type === 'success' &&
              'border-emerald-200 bg-emerald-50/95 dark:border-emerald-800/60 dark:bg-emerald-950/90',
            t.type === 'info' &&
              'border-sky-200 bg-sky-50/95 dark:border-sky-800/60 dark:bg-sky-950/90',
          )}
        >
          {t.type === 'error' && (
            <AlertCircle className="mt-0.5 size-4 shrink-0 text-red-500" strokeWidth={2} />
          )}
          {t.type === 'success' && (
            <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-emerald-500" strokeWidth={2} />
          )}
          {t.type === 'info' && (
            <Info className="mt-0.5 size-4 shrink-0 text-sky-500" strokeWidth={2} />
          )}
          <p
            className={cn(
              'flex-1 text-sm leading-snug',
              t.type === 'error' && 'text-red-700 dark:text-red-300',
              t.type === 'success' && 'text-emerald-700 dark:text-emerald-300',
              t.type === 'info' && 'text-sky-700 dark:text-sky-300',
            )}
          >
            {t.message}
          </p>
          <button
            type="button"
            onClick={() => setItems((prev) => prev.filter((x) => x.id !== t.id))}
            className="mt-0.5 shrink-0 rounded p-0.5 text-neutral-400 transition-colors hover:text-neutral-600 dark:text-zinc-500 dark:hover:text-zinc-300"
          >
            <X className="size-3.5" strokeWidth={2} />
          </button>
        </div>
      ))}
    </div>
  )
}
