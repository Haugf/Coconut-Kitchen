/*
  Central Park in the fall. Everything is placed on a 1920 by 1080 stage
  that scales with the screen. One clump leaves the tree each day, one
  leaf falls, and past leaves land on their date along the ridge.
*/

export type Time = 'morning' | 'afternoon' | 'dusk' | 'night'
export type Sky = 'clear' | 'cloudy' | 'rain' | 'snow'

export const TIMES: Record<Time, { ground: string; sky: string; lake: string; filter: string; orb: [number, number] }> = {
  morning: { ground: '#2C3F2E', sky: '#D5E5E2', lake: '#6FA1A0', filter: 'none', orb: [640, 300] },
  afternoon: { ground: '#35502F', sky: '#BFDDEB', lake: '#4F93A8', filter: 'saturate(1.08)', orb: [1500, 110] },
  dusk: { ground: '#27301F', sky: '#E59A5B', lake: '#A8605A', filter: 'sepia(.3) saturate(1.2) brightness(.78)', orb: [1340, 430] },
  night: { ground: '#131C19', sky: '#141C33', lake: '#1E2F4A', filter: 'brightness(.42) saturate(.7)', orb: [640, 170] },
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
const CANOPY = { x: 1000, y: 300, rx: 310, ry: 270 }
const BASE = 905

export interface Peak {
  day: number
  height: number
  label: string
}

/** Election Day and Thanksgiving in November, plus all-day events from the calendar. */
export function peaksFor(date: Date, allDay: { day: number; title: string }[]): Peak[] {
  const peaks: Peak[] = []
  if (date.getMonth() === 10) {
    peaks.push({ day: electionDay(date.getFullYear()), height: 0.35, label: 'Election Day' })
    peaks.push({ day: thanksgiving(date.getFullYear()), height: 1, label: 'Thanksgiving' })
  }
  for (const e of allDay) {
    if (peaks.length >= 4) break
    if (peaks.every((p) => Math.abs(p.day - e.day) >= 4)) peaks.push({ day: e.day, height: 0.7, label: e.title })
  }
  return peaks
}

export const dayToClump = (day: number) => (day * 11) % CLUMPS

export function clumpAt(i: number) {
  const a = i * 2.39996
  const r = Math.sqrt((i + 0.5) / CLUMPS)
  return { x: CANOPY.x + r * CANOPY.rx * Math.cos(a), y: CANOPY.y + r * CANOPY.ry * Math.sin(a) }
}

export const dayX = (day: number, last: number) => 90 + ((day - 1) * 1740) / (last - 1)

export function groundY(day: number, peaks: Peak[]) {
  return peaks.reduce((y, p) => y - p.height * 80 * Math.exp(-(((day - p.day) / 1.6) ** 2)), BASE)
}

export function groundPath(last: number, peaks: Peak[]) {
  let d = `M0 1080 L0 ${groundY(-1, peaks).toFixed(1)}`
  for (let day = -1; day <= last + 2; day += 0.5) d += ` L${dayX(day, last).toFixed(1)} ${groundY(day, peaks).toFixed(1)}`
  return `${d} L1920 1080 Z`
}
