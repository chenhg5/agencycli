const STORAGE_KEY = 'agency-console-quick-links'
const MAX_LINKS = 24

export type QuickLink = {
  path: string
  title: string
  createdAt: number
}

function notifyQuickLinksChanged() {
  window.dispatchEvent(new Event('quick-links-changed'))
}

export function getQuickLinks(): QuickLink[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as QuickLink[]
    return parsed.filter((item) => item?.path && item?.title)
  } catch {
    return []
  }
}

function saveQuickLinks(links: QuickLink[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(links))
  notifyQuickLinksChanged()
}

export function addQuickLink(link: { path: string; title: string }) {
  const path = link.path.trim()
  const title = link.title.trim()
  if (!path || !title) return
  const links = getQuickLinks().filter((item) => item.path !== path)
  links.unshift({ path, title, createdAt: Date.now() })
  if (links.length > MAX_LINKS) links.length = MAX_LINKS
  saveQuickLinks(links)
}

export function removeQuickLink(path: string) {
  saveQuickLinks(getQuickLinks().filter((item) => item.path !== path))
}

export function renameQuickLink(path: string, title: string) {
  const nextTitle = title.trim()
  if (!nextTitle) return
  saveQuickLinks(getQuickLinks().map((item) => (
    item.path === path ? { ...item, title: nextTitle } : item
  )))
}
