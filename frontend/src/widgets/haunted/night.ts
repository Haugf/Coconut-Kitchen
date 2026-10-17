/*
  The haunted page's arithmetic: the moon, and how far off midnight is.
  Nothing here reads the calendar; at a party the screen shows nothing
  personal.
*/

const SYNODIC = 29.530588853 // days from new moon to new moon
const KNOWN_NEW = Date.UTC(2000, 0, 6, 18, 14) // a new moon

export interface Moon {
  /** 0 new, 0.25 first quarter, 0.5 full, 0.75 last quarter. */
  phase: number
  /** Share of the face that's lit, 0 to 1. */
  lit: number
  name: string
}

export function moonOn(d: Date): Moon {
  const days = (d.getTime() - KNOWN_NEW) / 86400000
  const phase = (((days % SYNODIC) + SYNODIC) % SYNODIC) / SYNODIC
  const lit = (1 - Math.cos(2 * Math.PI * phase)) / 2
  const name =
    phase < 0.03 || phase > 0.97
      ? 'New moon'
      : phase < 0.22
        ? 'Waxing crescent'
        : phase < 0.28
          ? 'First quarter'
          : phase < 0.47
            ? 'Waxing gibbous'
            : phase < 0.53
              ? 'Full moon'
              : phase < 0.72
                ? 'Waning gibbous'
                : phase < 0.78
                  ? 'Last quarter'
                  : 'Waning crescent'
  return { phase, lit, name }
}

/** The lit part of the moon as an SVG path, for a disc of radius r. */
export function moonPath(phase: number, cx: number, cy: number, r: number): string {
  const k = Math.cos(2 * Math.PI * phase) // 1 at new, -1 at full
  const rx = Math.abs(k) * r
  const top = `${cx} ${cy - r}`
  const bottom = `${cx} ${cy + r}`
  if (phase <= 0.5) {
    // Waxing: the right side is lit; the terminator bulges right while
    // it's a crescent and left once it's gibbous.
    return `M ${top} A ${r} ${r} 0 0 1 ${bottom} A ${rx} ${r} 0 0 ${k > 0 ? 0 : 1} ${top} Z`
  }
  return `M ${top} A ${r} ${r} 0 0 0 ${bottom} A ${rx} ${r} 0 0 ${k > 0 ? 1 : 0} ${top} Z`
}

/** Minutes until midnight, or null outside the evening. */
export function minutesToMidnight(now: Date): number | null {
  const h = now.getHours()
  if (h < 17) return null
  return 24 * 60 - (h * 60 + now.getMinutes())
}

/** The headline, in the page's voice: about the night, never the calendar. */
export function hauntedLine(now: Date, raining: boolean): string {
  const h = now.getHours()
  const halloween = now.getMonth() === 9 && now.getDate() === 31
  let line: string
  const left = minutesToMidnight(now)
  if (h === 0) line = 'It is the witching hour.'
  else if (h < 5) line = 'The ghosts are still dancing.'
  else if (left === null) line = 'The house is waiting for dark.'
  else if (left <= 60) line = `${left} ${left === 1 ? 'minute' : 'minutes'} until the witching hour.`
  else {
    const hours = Math.floor(left / 60)
    line = `The witching hour is ${hours === 1 ? 'an hour' : `${hours} hours`} away.`
  }
  const lead = [halloween ? 'Happy Halloween.' : '', raining ? 'Rain at the windows.' : ''].filter(Boolean).join(' ')
  return lead ? `${lead} ${line.charAt(0).toUpperCase()}${line.slice(1)}` : line
}
