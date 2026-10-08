import type { CalendarEvent, DayEvent } from './types'

export function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
}

function hours(d: Date): number {
  return d.getHours() + d.getMinutes() / 60
}

/** Timed events that start on `day`, sorted. All-day events are left out of the terrain. */
export function timedEventsOn(events: CalendarEvent[], day: Date): DayEvent[] {
  return events
    .filter((e) => !e.allDay && sameDay(new Date(e.start), day))
    .map((e) => {
      const start = new Date(e.start)
      const end = e.end ? new Date(e.end) : new Date(start.getTime() + 30 * 60 * 1000)
      const endH = sameDay(end, day) ? hours(end) : 24
      return { title: e.title, start, end, startH: hours(start), endH: Math.max(endH, hours(start) + 0.25), who: e.who ?? [] }
    })
    .sort((a, b) => a.startH - b.startH)
}

export function eventsOn(events: CalendarEvent[], day: Date): CalendarEvent[] {
  return events.filter((e) => {
    const start = new Date(e.start)
    if (!e.allDay) return sameDay(start, day)
    // All-day events run midnight to midnight; count the day they start.
    return sameDay(start, day)
  })
}

/** "2 PM", "2:30 PM" */
export function clock(d: Date): string {
  const h = d.getHours() % 12 || 12
  const m = d.getMinutes()
  const ap = d.getHours() < 12 ? 'AM' : 'PM'
  return m === 0 ? `${h} ${ap}` : `${h}:${String(m).padStart(2, '0')} ${ap}`
}

/** "2", "2:30" for use inside a sentence where AM/PM is obvious. */
export function shortClock(d: Date): string {
  const h = d.getHours() % 12 || 12
  const m = d.getMinutes()
  return m === 0 ? `${h}` : `${h}:${String(m).padStart(2, '0')}`
}

/** Events on one person's calendar. With a single unnamed person, everything. */
export function belongsTo<T extends { who?: string[] }>(items: T[], name: string | undefined): T[] {
  if (name === undefined) return []
  return items.filter((e) => !e.who || e.who.length === 0 || e.who.includes(name))
}

/** "on your calendar", "on Ally's calendar", "on both calendars" */
export function sourcePhrase(who: string[] | undefined, people: string[]): string {
  if (people.length < 2 || !who || who.length === 0) return 'on your calendar'
  if (who.length >= 2) return 'on both calendars'
  return `on ${who[0]}’s calendar`
}
