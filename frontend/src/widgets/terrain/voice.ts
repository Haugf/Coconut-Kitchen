import type { DayEvent } from './types'
import { shortClock, clock } from './day'

/*
  Plain, observational sentences built from the day's data. Observe and
  hand over: no commands, no hype. A Claude-written version can replace
  these later; the page doesn't care where the words come from.
*/

function at(e: DayEvent): string {
  return `${e.title} at ${shortClock(e.start)}`
}

/** One line. Name the one distinct thing if there is one, otherwise the shape. */
export function headline(events: DayEvent[]): string {
  if (events.length === 0) return 'Open all day.'
  if (events.length === 1) return `One thing today: ${at(events[0])}.`

  const longest = [...events].sort((a, b) => b.endH - b.startH - (a.endH - a.startH))[0]
  const longestDur = longest.endH - longest.startH
  const others = events.filter((e) => e !== longest)
  const standsOut = others.every((e) => e.endH - e.startH <= longestDur * 0.6)
  if (standsOut) {
    const after = events.filter((e) => e.startH >= longest.endH)
    return after.length === 0
      ? `A climb to ${at(longest)}, then the day opens up.`
      : `The day peaks with ${at(longest)}.`
  }

  if (events[0].startH >= 16) return `Open until ${shortClock(events[0].start)}, then ${events[0].title}.`
  const last = events[events.length - 1]
  if (last.endH <= 13) return `A busy morning, then open from ${shortClock(last.end)}.`
  return events.length >= 5 ? 'A full day, steady from start to finish.' : 'A steady day with room between things.'
}

export interface Act {
  label: string
  sentence: string
}

const ACTS = [
  { name: 'Morning', from: 0, to: 12 },
  { name: 'Afternoon', from: 12, to: 17 },
  { name: 'Evening', from: 17, to: 24 },
]

export function acts(events: DayEvent[]): Act[] {
  return ACTS.map(({ name, from, to }) => {
    const inAct = events.filter((e) => e.startH >= from && e.startH < to)
    if (inAct.length === 0) return { label: name, sentence: 'Open.' }
    const first = inAct[0]
    const lastEnd = inAct.reduce((m, e) => (e.end > m ? e.end : m), inAct[0].end)
    const label = to === 24 && lastEnd.getHours() >= 22
      ? `${clock(first.start)} onward`
      : `${clock(first.start)} – ${clock(lastEnd)}`
    let sentence: string
    if (inAct.length === 1) sentence = `${at(first)}.`
    else if (inAct.length === 2) sentence = `${at(first)}, then ${at(inAct[1])}.`
    else sentence = `${at(first)}, then ${inAct.length - 1} more.`
    return { label, sentence }
  })
}

export function partOfDay(now: Date): string {
  const h = now.getHours()
  return h < 12 ? 'this morning' : h < 17 ? 'this afternoon' : 'tonight'
}
