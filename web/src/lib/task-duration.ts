/** Format Go-style duration strings (e.g. "1h30m0s") for compact UI display. */
export function formatGoDuration(raw: string): string {
  const s = raw.trim()
  if (!s) return '—'
  const m = s.match(/^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+(?:\.\d+)?)s)?$/)
  if (!m) return s
  const h = m[1] ? Number(m[1]) : 0
  const min = m[2] ? Number(m[2]) : 0
  const sec = m[3] ? Math.round(Number(m[3])) : 0
  const parts: string[] = []
  if (h) parts.push(`${h}h`)
  if (min) parts.push(`${min}m`)
  if (sec && !h && !min) parts.push(`${sec}s`)
  return parts.join(' ') || s
}

export function taskElapsedLabel(task: { startedAt?: string; finishedAt?: string }): string | null {
  if (!task.startedAt) return null
  const start = new Date(task.startedAt).getTime()
  const end = task.finishedAt ? new Date(task.finishedAt).getTime() : Date.now()
  const ms = end - start
  if (ms <= 0) return null
  const sec = Math.floor(ms / 1000)
  if (sec < 60) return `${sec}s`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m`
  const h = Math.floor(min / 60)
  const rem = min % 60
  return rem ? `${h}h ${rem}m` : `${h}h`
}
