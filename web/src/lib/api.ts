import { getStoredToken } from './auth'
import { showToast } from '../components/ui/Toast'

export function apiBase(): string {
  return (import.meta.env.VITE_API_BASE ?? '').replace(/\/$/, '')
}

export function apiUrl(path: string): string {
  const p = path.startsWith('/') ? path : `/${path}`
  const b = apiBase()
  return b ? `${b}${p}` : p
}

function authHeaders(extra?: HeadersInit): Headers {
  const headers = new Headers(extra)
  const token = getStoredToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  return headers
}

function handle401() {
  localStorage.removeItem('agencycli-token')
  localStorage.removeItem('agencycli-user')
  window.dispatchEvent(new Event('auth-expired'))
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (res.status === 401) {
    handle401()
    throw new Error('unauthorized')
  }
  if (!res.ok) {
    const text = await res.text()
    let detail = text.trim().slice(0, 200) || res.statusText
    try {
      const j = JSON.parse(text) as { error?: string }
      if (j?.error) detail = j.error
    } catch { /* plain text */ }
    showToast(detail, 'error')
    throw new Error(`${res.status} ${detail}`)
  }
  if (res.status === 204) return undefined as T
  const ct = res.headers.get('content-type') ?? ''
  if (ct.includes('application/json')) return res.json() as Promise<T>
  return undefined as T
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = authHeaders(init?.headers)
  if (!headers.has('Accept')) headers.set('Accept', 'application/json')
  const res = await fetch(apiUrl(path), { ...init, headers })
  return handleResponse<T>(res)
}

export async function apiPost<T>(path: string, body: unknown): Promise<T> {
  const headers = authHeaders()
  headers.set('Content-Type', 'application/json')
  headers.set('Accept', 'application/json')
  const res = await fetch(apiUrl(path), { method: 'POST', headers, body: JSON.stringify(body) })
  return handleResponse<T>(res)
}

export async function apiPatch<T>(path: string, body: unknown): Promise<T> {
  const headers = authHeaders()
  headers.set('Content-Type', 'application/json')
  headers.set('Accept', 'application/json')
  const res = await fetch(apiUrl(path), { method: 'PATCH', headers, body: JSON.stringify(body) })
  return handleResponse<T>(res)
}

export async function apiPut<T>(path: string, body: unknown): Promise<T> {
  const headers = authHeaders()
  headers.set('Content-Type', 'application/json')
  headers.set('Accept', 'application/json')
  const res = await fetch(apiUrl(path), { method: 'PUT', headers, body: JSON.stringify(body) })
  return handleResponse<T>(res)
}

export async function apiDelete(path: string): Promise<void> {
  const headers = authHeaders()
  headers.set('Accept', 'application/json')
  const res = await fetch(apiUrl(path), { method: 'DELETE', headers })
  await handleResponse<void>(res)
}

/** No-auth POST for login */
export async function apiLoginPost<T>(path: string, body: unknown): Promise<T> {
  const headers = new Headers()
  headers.set('Content-Type', 'application/json')
  headers.set('Accept', 'application/json')
  const res = await fetch(apiUrl(path), { method: 'POST', headers, body: JSON.stringify(body) })
  if (!res.ok) {
    const text = await res.text()
    let detail = text.trim().slice(0, 200) || res.statusText
    try {
      const j = JSON.parse(text) as { error?: string }
      if (j?.error) detail = j.error
    } catch { /* plain text */ }
    throw new Error(`${res.status} ${detail}`)
  }
  return res.json() as Promise<T>
}

export function apiTeamPath(teamId: string): string {
  return teamId.split('/').filter(Boolean).map((s) => encodeURIComponent(s)).join('/')
}
