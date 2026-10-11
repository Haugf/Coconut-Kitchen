import { useWidgetData } from '../../lib/useWidgetData'
import { useNow } from '../../lib/useNow'
import { describe } from '../weather/codes'
import type { CalendarData, CalendarEvent, WeatherData } from './types'
import { timedEventsOn, eventsOn, clock, belongsTo, sourcePhrase } from './day'
import { terrainPath, dots, clayMotif, eveningIsFree, xOf, yAt, W, H, BASE, DAY_START, DAY_END } from './shape'
import { headline, acts, partOfDay } from './voice'
import { useEnabledWidgets } from '../../lib/widgets'
import { WidgetSlot, widgetColumns } from '../loader'
import './terrain.css'

interface Item {
  title: string
  before: string
  source: string
  after: string
}

function dateLine(now: Date, weather: WeatherData | null): string {
  const h = now.getHours() % 12 || 12
  const time = `${h}:${String(now.getMinutes()).padStart(2, '0')}`
  const weekday = now.toLocaleDateString('en-US', { weekday: 'long' })
  const month = now.toLocaleDateString('en-US', { month: 'long' })
  const parts = [time, weekday, `${month} ${now.getDate()} ${now.getFullYear()}`]
  if (weather) parts.push(`${Math.round(weather.temp)}°, ${describe(weather.code).toLowerCase()}`)
  return parts.join(' · ')
}

function todayItems(events: CalendarEvent[], people: string[], weather: WeatherData | null, now: Date): Item[] {
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

function tomorrowItems(events: CalendarEvent[], people: string[], now: Date): Item[] {
  const tomorrow = new Date(now)
  tomorrow.setDate(now.getDate() + 1)
  return eventsOn(events, tomorrow).map((e) => ({
    title: e.title,
    before: e.allDay ? 'All day, ' : `At ${clock(new Date(e.start))}, `,
    source: sourcePhrase(e.who, people),
    after: '.',
  }))
}

function List({ heading, items, empty, className }: { heading: string; items: Item[]; empty: string; className?: string }) {
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

export function Terrain() {
  const now = useNow(60 * 1000)
  const cal = useWidgetData<CalendarData>('/api/calendar', 5 * 60 * 1000)
  const wx = useWidgetData<WeatherData>('/api/weather', 10 * 60 * 1000)
  const widgets = useEnabledWidgets()

  const all = cal.data?.events ?? []
  const people = cal.data?.people ?? []
  // Everything today drives the words; each person gets their own line.
  const today = timedEventsOn(all, now)
  const first = belongsTo(today, people[0])
  const second = people.length > 1 ? belongsTo(today, people[1]) : []
  const secondOnly = second.filter((e) => !first.includes(e))
  const motif = clayMotif(today)
  const nowH = now.getHours() + now.getMinutes() / 60
  const showNow = nowH > DAY_START && nowH < DAY_END

  const todayList = todayItems(all, people, wx.data, now)
  const tomorrowList = tomorrowItems(all, people, now)

  return (
    <div className="terrain">
      <header className="t-top">
        <p className="t-date">{dateLine(now, wx.data)}</p>
        <h1 className="t-headline">
          {cal.data ? headline(today) : cal.error ? 'The calendar isn’t connected yet.' : ' '}
        </h1>
        <svg className="t-land" viewBox={`0 0 ${W} ${H}`} fill="none" role="img" aria-label="Today’s load drawn as terrain">
          {motif.kind === 'dawn' && <path d={`M${motif.x - 16} ${BASE} A16 16 0 0 1 ${motif.x + 16} ${BASE}Z`} className="t-clay" />}
          {motif.kind === 'sun' && <circle cx={motif.x} cy={46} r={16} className="t-clay" />}
          {eveningIsFree(today) && (
            <path d={`M${xOf(19.5)} 70 q9 -10 18 0 q9 -10 18 0 M${xOf(20.4)} 52 q7 -8 14 0 q7 -8 14 0`} className="t-birds" />
          )}
          {showNow && <line x1={xOf(nowH)} x2={xOf(nowH)} y1={14} y2={yAt(nowH, first) - 10} className="t-now" />}
          {people.length > 1 && <path d={terrainPath(second)} className="t-line t-line-second" />}
          <path d={terrainPath(first)} className="t-line" />
          {secondOnly.map((e, i) => (
            <circle key={`s${i}`} cx={xOf(e.startH)} cy={yAt(e.startH, second)} r={7} className="t-dot t-dot-second" />
          ))}
          {dots(first).map((d, i) =>
            d.collision ? (
              <g key={i}>
                <circle cx={d.x - d.r * 0.45} cy={d.y} r={d.r} className="t-hollow" />
                <circle cx={d.x + d.r * 0.45} cy={d.y} r={d.r} className="t-hollow" />
              </g>
            ) : (
              <circle key={i} cx={d.x} cy={d.y} r={d.r} className="t-dot" />
            ),
          )}
        </svg>
        <div className="t-acts" hidden={!cal.data}>
          {acts(today).map((a) => (
            <div className="t-act" key={a.label}>
              <p className="t-act-label">{a.label}</p>
              <p className="t-soft">{a.sentence}</p>
            </div>
          ))}
        </div>
      </header>
      <div className="t-bottom" style={{ gridTemplateColumns: widgetColumns(widgets) }}>
        <div className="t-col">
        <List
          heading="Today"
          items={todayList}
          empty={cal.data ? `Nothing needs you ${partOfDay(now)}.` : 'Add calendar addresses to config.json on the Pi to fill this in.'}
        />
        <List heading="Tomorrow" items={tomorrowList} empty="Nothing on the calendar yet." className="t-list-tomorrow" />
        </div>
        {widgets.map((w) => (
          <WidgetSlot key={w.id} widget={w} />
        ))}
      </div>
    </div>
  )
}
