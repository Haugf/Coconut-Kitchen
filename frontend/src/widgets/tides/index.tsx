import { useSource, type WidgetProps } from '../../lib/widgets'
import { useNow } from '../../lib/useNow'
import './style.css'

// NOAA CO-OPS predictions. Times are the station's local time, as
// "YYYY-MM-DD HH:MM"; heights are feet above mean lower low water.
interface Prediction {
  t: string
  v: string
  type?: 'H' | 'L'
}
interface Predictions {
  predictions?: Prediction[]
}

interface Settings {
  station: string
  place: string
}

const W = 240
const H = 60

function when(t: string): Date {
  const [d, hm] = t.split(' ')
  const [y, mo, day] = d.split('-').map(Number)
  const [h, m] = hm.split(':').map(Number)
  return new Date(y, mo - 1, day, h, m)
}

function clock(d: Date): string {
  return d.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' }).replace(/\s?([AP])M/i, (_, x: string) => ` ${x.toLowerCase()}m`)
}

/** Minutes since midnight, as a 0..1 share of the day. */
/** "3:11a", the way a printed tide table writes it. */
function short(d: Date): string {
  const h = d.getHours() % 12 || 12
  return `${h}:${String(d.getMinutes()).padStart(2, '0')}${d.getHours() < 12 ? 'a' : 'p'}`
}

function dayShare(d: Date): number {
  return (d.getHours() * 60 + d.getMinutes()) / (24 * 60)
}

/** "Rising toward high tide at 9:30 pm." */
function sentence(turns: Prediction[], now: Date): string {
  const next = turns.find((p) => when(p.t) > now)
  if (next) {
    const high = next.type === 'H'
    return `${high ? 'Rising' : 'Falling'} toward ${high ? 'high' : 'low'} tide at ${clock(when(next.t))}.`
  }
  const last = turns[turns.length - 1]
  if (!last) return ''
  return `${last.type === 'H' ? 'Falling' : 'Rising'} since the ${clock(when(last.t))} ${last.type === 'H' ? 'high' : 'low'}.`
}

export default function Tides(props: WidgetProps<Settings>) {
  const now = useNow(60 * 1000)
  const curve = useSource<Predictions>(props, 'curve', 60 * 60 * 1000).data?.predictions ?? []
  const turns = (useSource<Predictions>(props, 'turns', 60 * 60 * 1000).data?.predictions ?? []).filter((p) => p.type)
  if (curve.length < 2) return null

  const pts = curve.map((p) => ({ x: dayShare(when(p.t)) * W, v: Number(p.v) }))
  const lo = Math.min(0, ...pts.map((p) => p.v))
  const hi = Math.max(...pts.map((p) => p.v))
  const y = (v: number) => H - ((v - lo) / (hi - lo || 1)) * H
  const last = pts[pts.length - 1]
  const line = pts.map((p, i) => `${i ? 'L' : 'M'}${p.x.toFixed(1)} ${y(p.v).toFixed(1)}`).join(' ')
  const water = `${line} L${last.x.toFixed(1)} ${H} L${pts[0].x.toFixed(1)} ${H} Z`

  // Where the water is now, between the two nearest half-hour points.
  const nx = dayShare(now) * W
  const i = Math.max(1, pts.findIndex((p) => p.x >= nx))
  const a = pts[i - 1]
  const b = pts[Math.min(i, pts.length - 1)]
  const f = b.x === a.x ? 0 : Math.min(1, Math.max(0, (nx - a.x) / (b.x - a.x)))
  const ny = y(a.v + (b.v - a.v) * f)

  return (
    <section className="t-list w-tides">
      <h2 className="t-heading">Tides</h2>
      <p>{sentence(turns, now)}</p>
      <svg className="w-tides-curve" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" aria-hidden="true">
        <path className="w-tides-water" d={water} />
        <line className="w-tides-base" x1="0" x2={W} y1={H} y2={H} />
        <line className="w-tides-now" x1={nx} x2={nx} y1="0" y2={H} vectorEffect="non-scaling-stroke" />
        <circle className="w-tides-dot" cx={nx} cy={ny} r="3" vectorEffect="non-scaling-stroke" />
      </svg>
      <p className="t-soft w-tides-turns">
        {turns.map((p) => (
          <span key={p.t} className={when(p.t) < now ? 'is-past' : undefined}>
            <b>{p.type === 'H' ? 'High' : 'Low'}</b> {short(when(p.t))}
          </span>
        ))}
      </p>
      <p className="t-soft">At {props.settings.place}.</p>
    </section>
  )
}
