import '@fontsource/young-serif/latin-400.css'
import '@fontsource-variable/figtree'
import { useWidgetData } from '../../lib/useWidgetData'
import { useNow } from '../../lib/useNow'
import { useMessages } from '../../lib/useMessages'
import { useEnabledWidgets } from '../../lib/widgets'
import { WidgetSlot } from '../loader'
import type { CalendarData, WeatherData } from '../terrain/types'
import { Timers, useTimers } from '../kitchen/Timers'
import { Scene } from './Scene'
import { peaksFor, skyOf, timeOf, type Sky, type Time } from './scene'
import '../terrain/terrain.css'
import '../kitchen/kitchen.css'
import './park.css'

/*
  Central Park: the whole screen is the park. A big clock on the sky at
  the left, the map and trains on the sky at the right, and the month
  along the ground. ?day=14&time=dusk&sky=rain in the address tries
  another day, hour or sky on this screen only.
*/

const SKY_WORDS: Record<Sky, string> = {
  clear: 'and clear over the Park',
  cloudy: 'and cloudy over the Park',
  rain: 'and raining in the Park',
  snow: 'and snowing in the Park',
}

export function Park() {
  const clock = useNow(60 * 1000)
  const cal = useWidgetData<CalendarData>('/api/calendar', 5 * 60 * 1000)
  const wx = useWidgetData<WeatherData>('/api/weather', 10 * 60 * 1000)
  const widgets = useEnabledWidgets()
  const message = useMessages()
  const timers = useTimers()

  const q = new URLSearchParams(window.location.search)
  const tryDay = Number(q.get('day'))
  const now = tryDay ? new Date(new Date(clock).setDate(tryDay)) : clock
  const time = (q.get('time') as Time | null) ?? timeOf(now)
  const sky = (q.get('sky') as Sky | null) ?? skyOf(wx.data?.code)

  const allDay = (cal.data?.events ?? [])
    .filter((e) => e.allDay && new Date(e.start).getMonth() === now.getMonth())
    .map((e) => ({ day: new Date(e.start).getDate(), title: e.title }))
  const peaks = peaksFor(now, allDay)

  const hours = now.getHours() % 12 || 12
  const minutes = String(now.getMinutes()).padStart(2, '0')
  const date = now.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })

  return (
    <div className="terrain kitchen park" data-time={time}>
      <Scene now={now} time={time} sky={sky} peaks={peaks} />
      <div className="pk-left">
        <p className="pk-clock">
          {hours}:{minutes}
        </p>
        <p className="pk-date">{date}</p>
        {wx.data && (
          <p className="pk-weather">
            {Math.round(wx.data.temp)}° {SKY_WORDS[sky]}
          </p>
        )}
        {timers.length > 0 && <Timers timers={timers} />}
        {message && <p className="k-words">{message.text}</p>}
      </div>
      <div className="pk-right">
        {widgets.map((w) => (
          <WidgetSlot key={w.id} widget={w} />
        ))}
      </div>
    </div>
  )
}
