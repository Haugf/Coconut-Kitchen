import '@fontsource/young-serif/latin-400.css'
import '@fontsource-variable/figtree'
import { useWidgetData } from '../../lib/useWidgetData'
import { useNow } from '../../lib/useNow'
import { useMessages } from '../../lib/useMessages'
import { useEnabledWidgets } from '../../lib/widgets'
import { WidgetSlot } from '../loader'
import type { CalendarData, WeatherData } from '../terrain/types'
import { timedEventsOn } from '../terrain/day'
import { dateLine, todayItems, tomorrowItems, List } from '../terrain/lists'
import { headline, partOfDay } from '../terrain/voice'
import { DayStrip } from './DayStrip'
import { Playing, useNowPlaying } from './Playing'
import { Timers, useTimers } from './Timers'
import { Rain } from './Rain'
import { rainAmount } from './rain'
import { sceneFor } from '../../scenes'
import '../terrain/terrain.css'
import './kitchen.css'

/*
  Coconut Kitchen: the terrain page's day, moved to the kitchen. The
  calendar line runs along the bottom with the tide drawn behind it on the
  same 6 AM to midnight clock; the lists and widgets sit above. Messages
  sent with `mirror say` land in their own column instead of a bar.
*/

function CoconutMark() {
  return (
    <svg className="k-mark-icon" viewBox="0 0 26 26" aria-hidden="true">
      <circle cx="13" cy="13" r="11.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
      <circle cx="9.5" cy="10.5" r="1.6" fill="currentColor" />
      <circle cx="16.5" cy="10.5" r="1.6" fill="currentColor" />
      <circle cx="13" cy="16.5" r="1.6" fill="currentColor" />
    </svg>
  )
}

export function Kitchen() {
  const now = useNow(60 * 1000)
  const cal = useWidgetData<CalendarData>('/api/calendar', 5 * 60 * 1000)
  const wx = useWidgetData<WeatherData>('/api/weather', 10 * 60 * 1000)
  const widgets = useEnabledWidgets()
  const message = useMessages()
  const playing = useNowPlaying()
  const timers = useTimers()

  const all = cal.data?.events ?? []
  const people = cal.data?.people ?? []
  const today = timedEventsOn(all, now)
  const tomorrow = tomorrowItems(all, people, now)
  // The second column holds the latest message, else what the speaker is
  // playing, else tomorrow, with any running timers on top. With none of
  // these, it steps aside for the widgets.
  const side = message ? 'message' : playing ? 'playing' : tomorrow.length > 0 ? 'tomorrow' : null
  const hasSide = side !== null || timers.length > 0
  // Rain on the window, only while it's raining. ?rain=1 (or 0.4 to 1)
  // in the address tries it on a dry day.
  const tryRain = new URLSearchParams(window.location.search).get('rain')
  const rain = tryRain ? Math.min(1, Math.max(0.2, Number(tryRain) || 1)) : rainAmount(wx.data?.code)

  const columns = [
    'minmax(0, 1.1fr)',
    ...(hasSide ? ['minmax(0, 1fr)'] : []),
    // A filling widget (train times) keeps enough room for a row on one
    // line; the lists give way first.
    ...widgets.map((w) => (w.width === 'fit' ? 'auto' : 'minmax(calc(var(--u) * 42), 1fr)')),
  ].join(' ')

  return (
    <div className={rain > 0 ? 'terrain kitchen is-raining' : 'terrain kitchen'} data-calendar-missing={cal.data ? undefined : ''}>
      {rain > 0 && <Rain amount={rain} night={Boolean(sceneFor(now).dim)} />}
      <header className="k-top">
        <div className="k-meta">
          <p className="t-date">{dateLine(now, wx.data)}</p>
          <p className="k-mark">
            <CoconutMark />
            Coconut Kitchen
          </p>
        </div>
        <h1 className="t-headline k-headline">
          {cal.data ? headline(today) : cal.error ? 'The calendar isn’t connected yet.' : ' '}
        </h1>
      </header>

      <div className="k-middle" style={{ gridTemplateColumns: columns }}>
        <div className="t-col">
          <List
            heading="Today"
            items={todayItems(all, people, wx.data, now)}
            empty={cal.data ? `Nothing needs you ${partOfDay(now)}.` : 'Add calendar addresses to config.json on the Pi to fill this in.'}
          />
        </div>
        {hasSide && (
          <div className="k-side">
            {timers.length > 0 && <Timers timers={timers} />}
            {side === 'message' && message && (
              <section className="t-list k-said">
                <h2 className="t-heading">In the kitchen</h2>
                {message.source && <p className="k-from">From {message.source}</p>}
                <p className="k-words">{message.text}</p>
              </section>
            )}
            {side === 'playing' && playing && <Playing np={playing} />}
            {side === 'tomorrow' && <List heading="Tomorrow" items={tomorrow} empty="" className="k-tomorrow" />}
          </div>
        )}
        {widgets.map((w) => (
          <WidgetSlot key={w.id} widget={w} />
        ))}
      </div>

      <DayStrip now={now} events={all} people={people} />
    </div>
  )
}
