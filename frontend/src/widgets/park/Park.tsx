import '@fontsource/young-serif/latin-400.css'
import '@fontsource-variable/figtree'
import { useWidgetData } from '../../lib/useWidgetData'
import { useNow } from '../../lib/useNow'
import { useMessages } from '../../lib/useMessages'
import { useEnabledWidgets } from '../../lib/widgets'
import { WidgetSlot } from '../loader'
import type { CalendarData, WeatherData } from '../terrain/types'
import { timedEventsOn } from '../terrain/day'
import { dateLine, todayItems, List } from '../terrain/lists'
import { headline, partOfDay } from '../terrain/voice'
import { DayStrip } from '../kitchen/DayStrip'
import { Playing, useNowPlaying } from '../kitchen/Playing'
import { Timers, useTimers } from '../kitchen/Timers'
import { Scene } from './Scene'
import { skyOf, timeOf, type Sky, type Time } from './scene'
import '../terrain/terrain.css'
import '../kitchen/kitchen.css'
import './park.css'

/*
  Central Park: the kitchen page's words in dark side panels, with the
  park through a window between them. ?day=14&time=dusk&sky=rain in the
  address tries another day, hour or sky on this screen only.
*/

export function Park() {
  const clock = useNow(60 * 1000)
  const cal = useWidgetData<CalendarData>('/api/calendar', 5 * 60 * 1000)
  const wx = useWidgetData<WeatherData>('/api/weather', 10 * 60 * 1000)
  const widgets = useEnabledWidgets()
  const message = useMessages()
  const playing = useNowPlaying()
  const timers = useTimers()

  const q = new URLSearchParams(window.location.search)
  const tryDay = Number(q.get('day'))
  const now = tryDay ? new Date(new Date(clock).setDate(tryDay)) : clock
  const time = (q.get('time') as Time | null) ?? timeOf(now)
  const sky = (q.get('sky') as Sky | null) ?? skyOf(wx.data?.code)

  const all = cal.data?.events ?? []
  const people = cal.data?.people ?? []

  return (
    <div className="terrain kitchen park" data-calendar-missing={cal.data ? undefined : ''}>
      <Scene now={now} time={time} sky={sky} />
      <div className="pk-left">
        <header className="k-top">
          <div className="k-meta">
            <p className="t-date">{dateLine(now, wx.data)}</p>
          </div>
          <h1 className="t-headline k-headline">
            {cal.data ? headline(timedEventsOn(all, now)) : cal.error ? 'The calendar isn’t connected yet.' : ' '}
          </h1>
        </header>
        {timers.length > 0 && <Timers timers={timers} />}
        {message ? (
          <section className="t-list k-said">
            <h2 className="t-heading">In the kitchen</h2>
            {message.source && <p className="k-from">From {message.source}</p>}
            <p className="k-words">{message.text}</p>
          </section>
        ) : playing ? (
          <Playing np={playing} />
        ) : (
          <List
            heading="Today"
            items={todayItems(all, people, wx.data, now)}
            empty={cal.data ? `Nothing needs you ${partOfDay(now)}.` : 'Add calendar addresses to config.json on the Pi to fill this in.'}
          />
        )}
      </div>
      <div className="pk-right">
        {widgets.map((w) => (
          <WidgetSlot key={w.id} widget={w} />
        ))}
      </div>
      <DayStrip now={now} events={all} people={people} />
    </div>
  )
}
