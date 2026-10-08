import type { WidgetDef } from './types'
import { Clock } from './clock/Clock'
import { Weather } from './weather/Weather'
import { Calendar } from './calendar/Calendar'
import { Terrain } from './terrain/Terrain'

/**
 * Every feature on the mirror is a widget registered here.
 * To add one: create src/widgets/<name>/<Name>.tsx, register it below,
 * and add its id to a scene in src/scenes.ts.
 */
export const registry: Record<string, WidgetDef> = {
  clock: { id: 'clock', component: Clock, size: 'hero' },
  weather: { id: 'weather', component: Weather, size: 'block' },
  calendar: { id: 'calendar', component: Calendar, size: 'block' },
  terrain: { id: 'terrain', component: Terrain, size: 'page' },
}
