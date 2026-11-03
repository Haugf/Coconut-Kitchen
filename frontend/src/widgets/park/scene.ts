/*
  Central Park in the fall. Everything is placed on a 1920 by 1080 stage
  that scales with the screen. One clump leaves the tree each day, one
  leaf falls, and past leaves land on their date along the ridge.
*/

export type Time = 'morning' | 'afternoon' | 'dusk' | 'night'
export type Sky = 'clear' | 'cloudy' | 'rain' | 'snow'

export const TIMES: Record<Time, { sky: string; lake: string; filter: string; orb: [number, number] }> = {
  morning: { sky: '#D5E5E2', lake: '#6FA1A0', filter: 'none', orb: [800, 170] },
  afternoon: { sky: '#BFDDEB', lake: '#4F93A8', filter: 'saturate(1.08)', orb: [1270, 120] },
  dusk: { sky: '#E59A5B', lake: '#A8605A', filter: 'sepia(.3) saturate(1.2) brightness(.78)', orb: [1280, 420] },
  night: { sky: '#141C33', lake: '#1E2F4A', filter: 'brightness(.42) saturate(.7)', orb: [760, 130] },
}

const GREY: Record<Exclude<Sky, 'clear'>, Record<Time, string>> = {
  cloudy: { morning: '#C6D0CE', afternoon: '#C3CACB', dusk: '#B78A7B', night: '#1E2433' },
  rain: { morning: '#A3AFB1', afternoon: '#9AA6AB', dusk: '#8E7270', night: '#171C29' },
  snow: { morning: '#DDE3E4', afternoon: '#DCE2E4', dusk: '#B9A6A6', night: '#222A3B' },
}

export const skyColor = (time: Time, sky: Sky) => (sky === 'clear' ? TIMES[time].sky : GREY[sky][time])

export function timeOf(date: Date): Time {
  const h = date.getHours()
  if (h >= 5 && h < 11) return 'morning'
  if (h >= 11 && h < 16) return 'afternoon'
  if (h >= 16 && h < 18) return 'dusk'
  return 'night'
}

/** What the sky is doing, from the WMO weather code. */
export function skyOf(code: number | undefined): Sky {
  if (code === undefined) return 'clear'
  if ((code >= 71 && code <= 77) || code === 85 || code === 86) return 'snow'
  if (code >= 51) return 'rain'
  if (code >= 3) return 'cloudy'
  return 'clear'
}

export function thanksgiving(year: number): number {
  const first = new Date(year, 10, 1).getDay()
  return 1 + ((4 - first + 7) % 7) + 21
}

export function electionDay(year: number): number {
  const first = new Date(year, 10, 1).getDay()
  return 1 + ((1 - first + 7) % 7) + 1
}

export const CLUMPS = 30
const CANOPY = { x: 960, y: 270, rx: 300, ry: 250 }
const BASE = 745

export const dayToClump = (day: number) => (day * 11) % CLUMPS

export function clumpAt(i: number) {
  const a = i * 2.39996
  const r = Math.sqrt((i + 0.5) / CLUMPS)
  return { x: CANOPY.x + r * CANOPY.rx * Math.cos(a), y: CANOPY.y + r * CANOPY.ry * Math.sin(a) }
}

export const dayX = (day: number, last: number) => 60 + ((day - 1) * 1800) / (last - 1)

export function groundY(day: number, peak: number | null) {
  return peak === null ? BASE : BASE - 62 * Math.exp(-(((day - peak) / 1.5) ** 2))
}

export function groundPath(last: number, peak: number | null) {
  let d = `M0 1080 L0 ${groundY(-1, peak)}`
  for (let day = -1; day <= last + 2; day += 0.5) d += ` L${dayX(day, last).toFixed(1)} ${groundY(day, peak).toFixed(1)}`
  return `${d} L1920 1080 Z`
}
