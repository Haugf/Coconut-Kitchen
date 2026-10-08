import { useEffect, useState } from 'react'

export interface MirrorMessage {
  text: string
  source?: string
  ttl?: number
  at: string
}

/**
 * Listens on /api/events (Server-Sent Events). Anything that can POST to
 * the backend, such as a script, Home Assistant, or a future voice
 * assistant, can put a message on the mirror.
 */
export function useMessages(): MirrorMessage | null {
  const [message, setMessage] = useState<MirrorMessage | null>(null)

  useEffect(() => {
    const source = new EventSource('/api/events')
    source.onmessage = (e) => {
      try {
        setMessage(JSON.parse(e.data) as MirrorMessage)
      } catch {
        /* ignore malformed events */
      }
    }
    return () => source.close()
  }, [])

  useEffect(() => {
    if (!message) return
    const timer = window.setTimeout(() => setMessage(null), (message.ttl ?? 20) * 1000)
    return () => window.clearTimeout(timer)
  }, [message])

  return message
}
