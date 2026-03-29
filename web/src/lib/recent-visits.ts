const STORAGE_KEY = 'agency-console-recent-visits'
const MAX_ITEMS = 10

export type RecentVisit = {
  path: string
  title: string
  visitedAt: number
}

export function getRecentVisits(): RecentVisit[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    return JSON.parse(raw) as RecentVisit[]
  } catch {
    return []
  }
}

export function recordVisit(path: string, title: string) {
  const list = getRecentVisits().filter((v) => v.path !== path)
  list.unshift({ path, title, visitedAt: Date.now() })
  if (list.length > MAX_ITEMS) list.length = MAX_ITEMS
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(list))
  } catch { /* ignore */ }
}
