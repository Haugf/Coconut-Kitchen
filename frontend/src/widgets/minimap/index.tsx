import { useSource, type WidgetProps } from '../../lib/widgets'
import './style.css'

interface LatLon {
  lat: number
  lon: number
}

interface Vehicle extends LatLon {
  id: string
  kind: 'subway' | 'bus'
  route: string
  from?: LatLon
  minutes: number
  state: 'moving' | 'stopped' | 'waiting' | 'far'
  note?: string
}

interface MapData {
  center: LatLon
  home?: LatLon
  radiusMeters: number
  streets: LatLon[][]
  lines: { route: string; points: LatLon[] }[]
  stations: ({ name: string } & LatLon)[]
  vehicles: Vehicle[]
}

// The MTA's own colors: M orange, L grey, buses blue.
const ROUTE_COLOR: Record<string, string> = { M: '#FF6319', L: '#A7A9AC' }
const BUS_COLOR = '#2F6BFF'
const colorOf = (v: { kind?: string; route: string }) =>
  v.kind === 'bus' ? BUS_COLOR : ROUTE_COLOR[v.route] ?? '#8C8A80'

// Drawn in a 200-unit square; the map's reach fills a circle of R.
const R = 92
const REFRESH_MS = 15 * 1000

export default function Minimap(props: WidgetProps) {
  const { data, error } = useSource<MapData>(props, 'map', REFRESH_MS)
  if (!data) return null

  const c = data.center
  const k = R / data.radiusMeters
  const cos = Math.cos((c.lat * Math.PI) / 180)
  const xy = (p: LatLon): [number, number] => [
    (p.lon - c.lon) * cos * 111320 * k,
    -(p.lat - c.lat) * 110540 * k,
  ]
  const path = (pts: LatLon[]) =>
    pts.map((p, i) => `${i ? 'L' : 'M'}${xy(p)[0].toFixed(1)} ${xy(p)[1].toFixed(1)}`).join(' ')

  const home = data.home ? xy(data.home) : null

  return (
    <figure className="t-minimap" aria-label="Map of trains and buses near home">
      <svg viewBox="-100 -100 200 200" role="img">
        <defs>
          <clipPath id="t-mm-clip">
            <circle r={R} />
          </clipPath>
        </defs>
        <g clipPath="url(#t-mm-clip)">
          <circle r={R} className="t-mm-ground" />
          <g className="t-mm-streets">
            {data.streets.map((s, i) => (
              <path key={i} d={path(s)} />
            ))}
          </g>
          {data.lines.map((l) => (
            <path key={l.route} d={path(l.points)} className="t-mm-line" stroke={colorOf(l)} />
          ))}
          {data.stations.map((s) => {
            const [x, y] = xy(s)
            return <circle key={s.name} cx={x} cy={y} r={2.6} className="t-mm-station" />
          })}
          {home && (
            <g transform={`translate(${home[0]} ${home[1]})`}>
              <circle r={8} className="t-mm-home-glow" />
              <circle r={3} className="t-mm-home" />
            </g>
          )}
        </g>
        <circle r={R} className="t-mm-ring" />
        {/* Names sit outside the clip so they never get cut at the edge,
            on the side of the station that faces the middle. */}
        {data.stations.map((s) => {
          const [x, y] = xy(s)
          if (Math.hypot(x, y) > R - 12) return null // too close to the edge to label
          const left = x > 0
          return (
            <text key={s.name} x={x + (left ? -5 : 5)} y={y - 5} textAnchor={left ? 'end' : 'start'} className="t-mm-name">
              {s.name}
            </text>
          )
        })}
        <text y={-R - 2} textAnchor="middle" className="t-mm-north">
          N
        </text>
        <g className={error ? 't-mm-stale' : undefined}>
          {data.vehicles.map((v) => (
            <VehicleMark key={v.id} v={v} xy={xy} />
          ))}
        </g>
      </svg>
    </figure>
  )
}

function VehicleMark({ v, xy }: { v: Vehicle; xy: (p: LatLon) => [number, number] }) {
  const [x, y] = xy(v)
  const dist = Math.hypot(x, y)
  const color = colorOf(v)
  const label = v.minutes <= 0 ? 'now' : String(v.minutes)
  const hollow = v.state === 'waiting' || v.note === "hasn't left yet"
  // Glide between refreshes instead of jumping.
  const glide = { transition: `transform ${REFRESH_MS}ms linear` }

  if (dist > R - 4 || v.state === 'far') {
    // Off the map: an arrow on the ring pointing to where it is.
    const a = Math.atan2(y, x)
    const ax = Math.cos(a) * (R - 5)
    const ay = Math.sin(a) * (R - 5)
    const lx = Math.cos(a) * (R - 16)
    const ly = Math.sin(a) * (R - 16) + 3
    return (
      <g style={{ ...glide, transform: `translate(${ax}px, ${ay}px)` }}>
        <path
          d="M4 0 L-3 -3.6 L-3 3.6 Z"
          transform={`rotate(${(a * 180) / Math.PI})`}
          fill={hollow ? 'none' : color}
          stroke={color}
          strokeWidth={1.4}
          strokeLinejoin="round"
        />
        <text x={lx - ax} y={ly - ay} textAnchor="middle" className="t-mm-min">
          {label}
        </text>
      </g>
    )
  }

  // A short trail behind the dot, pointing back where it came from.
  let trail = null
  if (v.from && v.state === 'moving') {
    const [fx, fy] = xy(v.from)
    const len = Math.hypot(fx - x, fy - y)
    if (len > 0.5) {
      const t = Math.min(12, len) / len
      trail = <path d={`M0 0 L${((fx - x) * t).toFixed(1)} ${((fy - y) * t).toFixed(1)}`} className="t-mm-trail" stroke={color} />
    }
  }
  return (
    <g style={{ ...glide, transform: `translate(${x}px, ${y}px)` }}>
      {trail}
      {v.state === 'stopped' && <circle r={7} fill={color} opacity={0.18} />}
      <circle r={4.2} fill={hollow ? 'var(--bg)' : color} stroke={hollow ? color : 'var(--bg)'} strokeWidth={1.6} />
      <text x={0} y={13} textAnchor="middle" className="t-mm-min">
        {label}
      </text>
    </g>
  )
}
