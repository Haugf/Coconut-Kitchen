import { smooth, xOf, BASE, DAY_START, DAY_END } from '../terrain/shape'

export interface TidePoint {
  t: string
  v: number
  type?: 'H' | 'L'
}

export interface TideData {
  place: string
  points: TidePoint[]
  turns: TidePoint[]
}

/** Hours since today's midnight, so tomorrow's 00:00 is 24. */
function hourOf(t: string, now: Date): number {
  const midnight = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  return (new Date(t).getTime() - midnight) / 3_600_000
}

// The water rises from just above the ridge's baseline to near the top of
// the strip, scaled to the day's own range.
const LOW_Y = BASE - 8
const HIGH_Y = 30

export interface TideShape {
  /** The water's surface. */
  line: string
  /** The same surface closed down to the baseline, for the fill. */
  fill: string
  turns: { x: number; y: number; high: boolean; label: string; past: boolean; next: boolean }[]
}

/** "3:11a", the way a printed tide table writes it. */
function short(d: Date): string {
  const h = d.getHours() % 12 || 12
  return `${h}:${String(d.getMinutes()).padStart(2, '0')}${d.getHours() < 12 ? 'a' : 'p'}`
}

export function tideShape(data: TideData | null, now: Date): TideShape | null {
  if (!data) return null
  const pts = data.points
    .map((p) => ({ h: hourOf(p.t, now), v: p.v }))
    .filter((p) => p.h >= DAY_START && p.h <= DAY_END)
  if (pts.length < 2) return null

  const lo = Math.min(...pts.map((p) => p.v))
  const hi = Math.max(...pts.map((p) => p.v))
  const y = (v: number) => LOW_Y - ((v - lo) / (hi - lo || 1)) * (LOW_Y - HIGH_Y)
  const xy: [number, number][] = pts.map((p) => [xOf(p.h), y(p.v)])
  const line = smooth(xy)
  const fill = `${line}L${xy[xy.length - 1][0].toFixed(1)} ${BASE}L${xy[0][0].toFixed(1)} ${BASE}Z`

  const nowH = hourOf(now.toISOString(), now)
  const today = data.turns
    .map((p) => ({ p, h: hourOf(p.t, now) }))
    .filter(({ h }) => h >= DAY_START && h < DAY_END)
  const nextIndex = today.findIndex(({ h }) => h > nowH)
  const turns = today.map(({ p, h }, i) => ({
    x: xOf(h),
    y: y(Math.min(Math.max(p.v, lo), hi)),
    high: p.type === 'H',
    label: `${p.type === 'H' ? 'high' : 'low'} ${short(new Date(p.t))}`,
    past: h <= nowH,
    next: i === nextIndex,
  }))
  return { line, fill, turns }
}
