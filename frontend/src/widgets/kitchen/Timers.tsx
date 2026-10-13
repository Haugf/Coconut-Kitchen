import { useWidgetData } from '../../lib/useWidgetData'
import { useNow } from '../../lib/useNow'

/*
  Timers set by voice on the Google speaker, from /api/timers. The Pi
  checks the speaker every few seconds; the countdown ticks here.
*/

export interface Timer {
  label?: string
  state: 'running' | 'paused' | 'ringing'
  fireAt?: string
  duration: number
  left?: number
}

export function useTimers(): Timer[] {
  const { data } = useWidgetData<{ enabled: boolean; timers: Timer[] }>('/api/timers', 3000)
  return data?.timers ?? []
}

function countdown(s: number): string {
  s = Math.max(0, Math.ceil(s))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = String(s % 60).padStart(2, '0')
  return h > 0 ? `${h}:${String(m).padStart(2, '0')}:${sec}` : `${m}:${sec}`
}

// "Timer · Pasta", "Paused · Bread", or just "Timer" when it has no name.
function heading(t: Timer): string {
  const l = t.label?.trim()
  const kind = t.state === 'paused' ? 'Paused' : 'Timer'
  return l ? `${kind} · ${l.charAt(0).toUpperCase()}${l.slice(1)}` : kind
}

export function Timers({ timers }: { timers: Timer[] }) {
  const now = useNow(1000)
  return (
    <section className="k-timers" aria-label="Timers">
      {timers.map((t, i) => {
        const left = t.state === 'paused' ? (t.left ?? 0) : t.fireAt ? (Date.parse(t.fireAt) - now.getTime()) / 1000 : 0
        const done = t.state === 'ringing' || (t.state === 'running' && left <= 0)
        const share = t.duration > 0 ? Math.min(1, Math.max(0, left / t.duration)) : 0
        return (
          <div key={i} className={`k-timer${i > 0 ? ' is-small' : ''}${done ? ' is-done' : ''}`}>
            <p className="k-from">{heading(t)}</p>
            <p className="k-count">{done ? 'Done' : countdown(left)}</p>
            {!done && (
              <span className="k-bar">
                <span style={{ width: `${share * 100}%` }} />
              </span>
            )}
          </div>
        )
      })}
    </section>
  )
}
