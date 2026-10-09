import type { DayEvent } from './types'

/** The drawing spans 6 AM to midnight. */
export const DAY_START = 6
export const DAY_END = 24
export const W = 1200
export const H = 150
export const BASE = 134
const MAX_RISE = 104

export function xOf(hour: number): number {
  const t = (Math.min(Math.max(hour, DAY_START), DAY_END) - DAY_START) / (DAY_END - DAY_START)
  return t * W
}

/** Elevation is load: each event raises the land around it, longer events higher and wider. */
function rise(hour: number, events: DayEvent[]): number {
  let total = 0
  for (const e of events) {
    const dur = Math.min(Math.max(e.endH - e.startH, 0.25), 4)
    const mid = (e.startH + e.endH) / 2
    const spread = Math.max(dur / 2, 0.5) + 0.6
    const weight = 40 + dur * 40
    total += weight * Math.exp(-(((hour - mid) / spread) ** 2))
  }
  // Ease off so a packed day reads as a ridge, not a spike.
  return MAX_RISE * (1 - Math.exp(-total / MAX_RISE))
}

export function yAt(hour: number, events: DayEvent[]): number {
  return BASE - rise(hour, events)
}

/** One smooth stroke from edge to edge (Catmull-Rom through sampled points). */
export function terrainPath(events: DayEvent[]): string {
  const n = 96
  const pts: [number, number][] = []
  for (let i = 0; i <= n; i++) {
    const h = DAY_START + ((DAY_END - DAY_START) * i) / n
    pts.push([xOf(h), yAt(h, events)])
  }
  let d = `M${pts[0][0].toFixed(1)} ${pts[0][1].toFixed(1)}`
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[Math.max(i - 1, 0)]
    const p1 = pts[i]
    const p2 = pts[i + 1]
    const p3 = pts[Math.min(i + 2, pts.length - 1)]
    const c1x = p1[0] + (p2[0] - p0[0]) / 6
    const c1y = p1[1] + (p2[1] - p0[1]) / 6
    const c2x = p2[0] - (p3[0] - p1[0]) / 6
    const c2y = p2[1] - (p3[1] - p1[1]) / 6
    d += `C${c1x.toFixed(1)} ${c1y.toFixed(1)} ${c2x.toFixed(1)} ${c2y.toFixed(1)} ${p2[0].toFixed(1)} ${p2[1].toFixed(1)}`
  }
  return d
}

export interface Dot {
  x: number
  y: number
  r: number
  /** Overlapping commitments are drawn as two hollow circles. */
  collision: boolean
}

export function dots(events: DayEvent[]): Dot[] {
  return events.map((e, i) => {
    const dur = e.endH - e.startH
    const collision = events.some((o, j) => j !== i && o.startH < e.endH && e.startH < o.endH)
    return {
      x: xOf(e.startH),
      y: yAt(e.startH, events),
      r: Math.round(6 + Math.min(dur, 2.5) * 2.8),
      collision,
    }
  })
}

export type Motif =
  | { kind: 'dawn'; x: number }
  | { kind: 'sun'; x: number }
  | { kind: 'none' }

/**
 * The one clay mark. A half-risen sun for an early start, otherwise a
 * full sun over the longest stretch of open time (if there is one).
 */
export function clayMotif(events: DayEvent[]): Motif {
  if (events.length && events[0].startH < 9) return { kind: 'dawn', x: 34 }
  let best = { len: 0, mid: 0 }
  let cursor = DAY_START
  for (const e of events) {
    if (e.startH - cursor > best.len) best = { len: e.startH - cursor, mid: (cursor + e.startH) / 2 }
    cursor = Math.max(cursor, e.endH)
  }
  if (DAY_END - cursor > best.len) best = { len: DAY_END - cursor, mid: (cursor + DAY_END) / 2 }
  return best.len >= 3 ? { kind: 'sun', x: xOf(best.mid) } : { kind: 'none' }
}

/** Birds over a free evening: room to breathe. */
export function eveningIsFree(events: DayEvent[]): boolean {
  return !events.some((e) => e.endH > 17 && e.startH < DAY_END)
}
