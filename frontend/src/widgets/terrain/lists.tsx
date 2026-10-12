import { describe } from '../weather/codes'
import type { CalendarEvent, WeatherData } from './types'
import { eventsOn, clock, sourcePhrase } from './day'

/*
  The date line and the Today / Tomorrow lists, shared by the terrain and
  kitchen pages.
*/

export interface Item {
  title: string
  before: string
  source: string
  after: string
}

export function dateLine(now: Date, weather: WeatherData | null): string {
  const h = now.getHours() % 12 || 12
  const time = `${h}:${String(now.getMinutes()).padStart(2, '0')}`
  const weekday = now.toLocaleDateString('en-US', { weekday: 'long' })
  const month = now.toLocaleDateString('en-US', { month: 'long' })
  const parts = [time, weekday, `${month} ${now.getDate()} ${now.getFullYear()}`]
  if (weather) parts.push(`${Math.round(weather.temp)}°, ${describe(weather.code).toLowerCase()}`)
  return parts.join(' · ')
}

export function todayItems(events: CalendarEvent[], people: string[], weather: WeatherData | null, now: Date): Item[] {
  const items: Item[] = []
  for (const e of eventsOn(events, now)) {
    if (!e.allDay && e.end && new Date(e.end) <= now) continue
    if (e.allDay) {
      items.push({ title: e.title, before: 'All day, ', source: sourcePhrase(e.who, people), after: '.' })
    } else {
      const start = new Date(e.start)
      const end = e.end ? new Date(e.end) : null
      const happening = start <= now && end && now < end
      items.push({
        title: happening ? `${e.title}, now` : `${e.title} at ${clock(start)}`,
        before: end ? `Until ${clock(end)}, ` : '',
        source: sourcePhrase(e.who, people),
        after: '.',
      })
    }
  }
  const today = weather?.days[0]
  // Today's rain chance stops being news by evening.
  if (today && today.rainChance >= 40 && now.getHours() < 18) {
    items.push({ title: 'Rain likely later', before: `A ${today.rainChance}% chance `, source: 'in the forecast', after: '.' })
  }
  return items
}

export function tomorrowItems(events: CalendarEvent[], people: string[], now: Date): Item[] {
  const tomorrow = new Date(now)
  tomorrow.setDate(now.getDate() + 1)
  return eventsOn(events, tomorrow).map((e) => ({
    title: e.title,
    before: e.allDay ? 'All day, ' : `At ${clock(new Date(e.start))}, `,
    source: sourcePhrase(e.who, people),
    after: '.',
  }))
}

export function List({ heading, items, empty, className }: { heading: string; items: Item[]; empty: string; className?: string }) {
  return (
    <section className={`t-list ${className ?? ''}`}>
      <h2 className="t-heading">{heading}</h2>
      {items.length === 0 ? (
        <p className="t-soft">{empty}</p>
      ) : (
        <ol className="t-items">
          {items.map((it, i) => (
            <li key={it.title + i}>
              <span className="t-num">{i + 1}</span>
              <div>
                <p className="t-title">{it.title}</p>
                <p className="t-soft">
                  {it.before}
                  <span className="t-source">{it.source}</span>
                  {it.after}
                </p>
              </div>
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}
