import '@fontsource/young-serif/latin-400.css'
import '@fontsource-variable/figtree'
import { useWidgetData } from '../../lib/useWidgetData'
import { useNow } from '../../lib/useNow'
import { useMessages } from '../../lib/useMessages'
import { useEnabledWidgets } from '../../lib/widgets'
import { WidgetSlot } from '../loader'
import type { CalendarData, WeatherData } from '../terrain/types'
import { timedEventsOn, belongsTo } from '../terrain/day'
import { dateLine, todayItems, tomorrowItems, List } from '../terrain/lists'
import { terrainPath, dots, xOf, yAt, BASE, DAY_START, DAY_END } from '../terrain/shape'
import { headline, partOfDay } from '../terrain/voice'
import { tideShape, type TideData } from './tide'
import { Playing, useNowPlaying } from './Playing'
import '../terrain/terrain.css'
import './kitchen.css'

/*
  Coconut Kitchen: the terrain page's day, moved to the kitchen. The
  calendar line runs along the bottom with the tide drawn behind it on the
  same 6 AM to midnight clock; the lists and widgets sit above. Messages
  sent with `mirror say` land in their own column instead of a bar.
*/

const TICKS: [number, string][] = [
  [6, '6a'],
  [9, '9a'],
  [12, '12p'],
  [15, '3p'],
  [18, '6p'],
  [21, '9p'],
  [24, '12a'],
]

function anchor(x: number): 'start' | 'middle' | 'end' {
  if (x < 30) return 'start'
  if (x > 1170) return 'end'
  return 'middle'
}

function CoconutMark() {
  return (
    <svg className="k-mark-icon" viewBox="0 0 26 26" aria-hidden="true">
      <circle cx="13" cy="13" r="11.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
      <circle cx="9.5" cy="10.5" r="1.6" fill="currentColor" />
      <circle cx="16.5" cy="10.5" r="1.6" fill="currentColor" />
      <circle cx="13" cy="16.5" r="1.6" fill="currentColor" />
    </svg>
  )
}

export function Kitchen() {
  const now = useNow(60 * 1000)
  const cal = useWidgetData<CalendarData>('/api/calendar', 5 * 60 * 1000)
  const wx = useWidgetData<WeatherData>('/api/weather', 10 * 60 * 1000)
  const tides = useWidgetData<TideData>('/api/tides', 30 * 60 * 1000)
  const widgets = useEnabledWidgets()
  const message = useMessages()
  const playing = useNowPlaying()

  const all = cal.data?.events ?? []
  const people = cal.data?.people ?? []
  const today = timedEventsOn(all, now)
  const first = belongsTo(today, people[0])
  const second = people.length > 1 ? belongsTo(today, people[1]) : []
  const secondOnly = second.filter((e) => !first.includes(e))
  const nowH = now.getHours() + now.getMinutes() / 60
  const showNow = nowH > DAY_START && nowH < DAY_END
  const tide = tideShape(tides.data, now)
  const tomorrow = tomorrowItems(all, people, now)
  // The second column holds the latest message, else what the speaker is
  // playing, else tomorrow. With none, it steps aside for the widgets.
  const side = message ? 'message' : playing ? 'playing' : tomorrow.length > 0 ? 'tomorrow' : null

  const columns = [
    'minmax(0, 1.1fr)',
    ...(side ? ['minmax(0, 1fr)'] : []),
    // A filling widget (train times) keeps enough room for a row on one
    // line; the lists give way first.
    ...widgets.map((w) => (w.width === 'fit' ? 'auto' : 'minmax(calc(var(--u) * 42), 1fr)')),
  ].join(' ')

  return (
    <div className="terrain kitchen" data-calendar-missing={cal.data ? undefined : ''}>
      <header className="k-top">
        <div className="k-meta">
          <p className="t-date">{dateLine(now, wx.data)}</p>
          <p className="k-mark">
            <CoconutMark />
            Coconut Kitchen
          </p>
        </div>
        <h1 className="t-headline k-headline">
          {cal.data ? headline(today) : cal.error ? 'The calendar isn’t connected yet.' : ' '}
        </h1>
      </header>

      <div className="k-middle" style={{ gridTemplateColumns: columns }}>
        <div className="t-col">
          <List
            heading="Today"
            items={todayItems(all, people, wx.data, now)}
            empty={cal.data ? `Nothing needs you ${partOfDay(now)}.` : 'Add calendar addresses to config.json on the Pi to fill this in.'}
          />
        </div>
        {side === 'message' && message && (
          <section className="t-list k-said">
            <h2 className="t-heading">In the kitchen</h2>
            {message.source && <p className="k-from">From {message.source}</p>}
            <p className="k-words">{message.text}</p>
          </section>
        )}
        {side === 'playing' && playing && <Playing np={playing} />}
        {side === 'tomorrow' && <List heading="Tomorrow" items={tomorrow} empty="" className="k-tomorrow" />}
        {widgets.map((w) => (
          <WidgetSlot key={w.id} widget={w} />
        ))}
      </div>

      <svg className="k-strip" viewBox="0 -24 1200 200" fill="none" role="img" aria-label="Today’s calendar drawn as a line, with the tide behind it">
        {tide && <path d={tide.fill} className="k-tide-fill" />}
        {tide && <path d={tide.line} className="k-tide-line" />}
        {showNow && <line x1={xOf(nowH)} x2={xOf(nowH)} y1={0} y2={BASE} className="k-now" />}
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
        {showNow && <circle cx={xOf(nowH)} cy={yAt(nowH, first)} r={7} className="k-now-dot" />}
        {tide?.turns.map((t) => (
          <text
            key={t.label}
            x={t.x}
            y={t.y - 12}
            textAnchor={anchor(t.x)}
            className={`k-turn${t.past ? ' is-past' : ''}${t.next ? ' is-next' : ''}`}
          >
            {t.label}
          </text>
        ))}
        {TICKS.map(([h, label]) => (
          <text key={label} x={xOf(h)} y={BASE + 30} textAnchor={anchor(xOf(h))} className="k-tick">
            {label}
          </text>
        ))}
      </svg>
    </div>
  )
}
