import { useWidgetData } from '../../lib/useWidgetData'
import type { CalendarEvent } from '../terrain/types'
import { timedEventsOn, belongsTo } from '../terrain/day'
import { terrainPath, dots, xOf, yAt, BASE, DAY_START, DAY_END } from '../terrain/shape'
import { tideShape, type TideData } from './tide'

// The day strip: calendar line in front, the tide behind on the same
// 6 AM to midnight clock. Shared by the kitchen and park pages.

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

export function DayStrip({ now, events, people }: { now: Date; events: CalendarEvent[]; people: string[] }) {
  const tides = useWidgetData<TideData>('/api/tides', 30 * 60 * 1000)
  const today = timedEventsOn(events, now)
  const first = belongsTo(today, people[0])
  const second = people.length > 1 ? belongsTo(today, people[1]) : []
  const secondOnly = second.filter((e) => !first.includes(e))
  const nowH = now.getHours() + now.getMinutes() / 60
  const showNow = nowH > DAY_START && nowH < DAY_END
  const tide = tideShape(tides.data, now)

  return (
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
  )
}
