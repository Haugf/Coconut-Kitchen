import { useWidgetData } from '../../lib/useWidgetData'
import { useNow } from '../../lib/useNow'

interface CalendarEvent {
  title: string
  start: string
  end: string
  allDay: boolean
  location?: string
}

function dayKey(d: Date) {
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
}

function dayLabel(d: Date, now: Date): string {
  const tomorrow = new Date(now)
  tomorrow.setDate(now.getDate() + 1)
  if (dayKey(d) === dayKey(now)) return 'Today'
  if (dayKey(d) === dayKey(tomorrow)) return 'Tomorrow'
  return d.toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric' })
}

function timeLabel(e: CalendarEvent): string {
  if (e.allDay) return 'All day'
  return new Date(e.start).toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
}

export function Calendar() {
  const { data, error } = useWidgetData<CalendarEvent[]>('/api/calendar', 5 * 60 * 1000)
  const now = useNow(60 * 1000)

  if (!data) {
    return (
      <section className="widget calendar">
        <p className="muted">{error ? 'Calendar unavailable. Add your calendar address to config.json.' : 'Loading calendar'}</p>
      </section>
    )
  }

  if (data.length === 0) {
    return (
      <section className="widget calendar">
        <p className="muted">Nothing on the calendar this week.</p>
      </section>
    )
  }

  const groups = new Map<string, { label: string; events: CalendarEvent[] }>()
  for (const e of data) {
    const start = new Date(e.start)
    const key = dayKey(start)
    if (!groups.has(key)) groups.set(key, { label: dayLabel(start, now), events: [] })
    groups.get(key)!.events.push(e)
  }

  return (
    <section className="widget calendar" aria-label="Calendar">
      {[...groups.entries()].map(([key, group]) => (
        <div className="calendar-day" key={key}>
          <h2 className="calendar-day-label">{group.label}</h2>
          <ul>
            {group.events.map((e) => {
              const happening = !e.allDay && new Date(e.start) <= now && now < new Date(e.end)
              return (
                <li key={e.start + e.title} className={happening ? 'calendar-event is-now' : 'calendar-event'}>
                  <span className="calendar-time">{happening ? 'Now' : timeLabel(e)}</span>
                  <span className="calendar-title">{e.title}</span>
                </li>
              )
            })}
          </ul>
        </div>
      ))}
    </section>
  )
}
