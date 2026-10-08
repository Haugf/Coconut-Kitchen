import { useEffect, useState } from 'react'

export interface WidgetData<T> {
  data: T | null
  error: string | null
}

/**
 * Polls a backend endpoint. Keeps the last good data on screen if a
 * refresh fails, so a network blip never blanks the mirror.
 */
export function useWidgetData<T>(url: string, refreshMs: number): WidgetData<T> {
  const [state, setState] = useState<WidgetData<T>>({ data: null, error: null })

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const res = await fetch(url)
        if (!res.ok) throw new Error((await res.text()) || `HTTP ${res.status}`)
        const data = (await res.json()) as T
        if (!cancelled) setState({ data, error: null })
      } catch (err) {
        if (!cancelled)
          setState((prev) => ({ data: prev.data, error: String((err as Error).message ?? err) }))
      }
    }
    load()
    const timer = window.setInterval(load, refreshMs)
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [url, refreshMs])

  return state
}
