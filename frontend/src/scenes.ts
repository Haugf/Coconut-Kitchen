/**
 * Scenes swap the visible widgets by time of day, so the mirror shows
 * what matters now instead of everything at once.
 * `from` is the local hour (0-23) a scene starts. The last scene wraps
 * past midnight until the first scene begins.
 */
export interface Scene {
  name: string
  from: number
  widgets: string[]
  /** Lower brightness, e.g. overnight. */
  dim?: boolean
}

// 'page' is whichever style is chosen on the Pi with `mirror style`:
// 'kitchen' (Coconut Kitchen) or 'terrain' (the first design).
export const scenes: Scene[] = [
  { name: 'day', from: 5, widgets: ['page'] },
  { name: 'night', from: 22, widgets: ['page'], dim: true },
]

// The earlier dark layout, kept for when the mirror glass goes on:
//   { name: 'morning', from: 5, widgets: ['clock', 'weather', 'calendar'] },

export function sceneFor(date: Date): Scene {
  const hour = date.getHours()
  const sorted = [...scenes].sort((a, b) => a.from - b.from)
  let active = sorted[sorted.length - 1]
  for (const s of sorted) if (s.from <= hour) active = s
  return active
}
