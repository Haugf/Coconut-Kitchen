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
    let source: EventSource | null = null
    const connect = () => {
      source?.close()
      source = new EventSource('/api/events')
      source.onmessage = (e) => {
        try {
          setMessage(JSON.parse(e.data) as MirrorMessage)
        } catch {
          /* ignore malformed events */
        }
      }
    }
    // Phones pause background tabs and can leave the stream silently dead.
    // Reconnect on return; the server replays the current message.
    const onVisible = () => {
      if (document.visibilityState === 'visible') connect()
    }
    connect()
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      document.removeEventListener('visibilitychange', onVisible)
      source?.close()
    }
  }, [])

  useEffect(() => {
    if (!message) return
    const timer = window.setTimeout(() => setMessage(null), (message.ttl ?? 20) * 1000)
    return () => window.clearTimeout(timer)
  }, [message])

  return message
}
