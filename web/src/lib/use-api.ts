import { startTransition, useEffect, useState } from 'react'
import { apiFetch } from './api'

export type ApiState<T> =
  | { status: 'loading' }
  | { status: 'error'; error: Error }
  | { status: 'ok'; data: T }

export function useApiJson<T>(path: string | null, reloadKey = 0): ApiState<T> {
  const [state, setState] = useState<ApiState<T>>({ status: 'loading' })

  useEffect(() => {
    if (path == null) {
      return
    }
    let cancelled = false
    startTransition(() => {
      setState({ status: 'loading' })
    })
    const url = reloadKey ? `${path}${path.includes('?') ? '&' : '?'}_=${reloadKey}` : path
    apiFetch<T>(url)
      .then((data) => {
        if (!cancelled) {
          setState({ status: 'ok', data })
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setState({
            status: 'error',
            error: e instanceof Error ? e : new Error(String(e)),
          })
        }
      })
    return () => {
      cancelled = true
    }
  }, [path, reloadKey])

  if (path == null) {
    return { status: 'error', error: new Error('no path') }
  }
  return state
}
