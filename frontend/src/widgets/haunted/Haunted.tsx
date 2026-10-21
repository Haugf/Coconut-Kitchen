import '@fontsource/im-fell-english/400.css'
import '@fontsource/im-fell-english/400-italic.css'
import '@fontsource/unifrakturmaguntia/400.css'
import { useEffect, useState } from 'react'
import { useWidgetData } from '../../lib/useWidgetData'
import { useNow } from '../../lib/useNow'
import { useMessages } from '../../lib/useMessages'
import { useEnabledWidgets } from '../../lib/widgets'
import { WidgetSlot } from '../loader'
import type { WeatherData } from '../terrain/types'
import { dateLine } from '../terrain/lists'
import { Playing, useNowPlaying } from '../kitchen/Playing'
import { Timers, useTimers } from '../kitchen/Timers'
import { rainAmount } from '../kitchen/rain'
import { moonOn, moonPath, minutesToMidnight, hauntedLine } from './night'
import { Grounds } from './Grounds'
import { Stars } from './Stars'
import { Sky, useVisitor } from './Visitors'
import '../terrain/terrain.css'
import './haunted.css'

/*
  The haunted house: a party page for Halloween. An ornate, candlelit
  room in the dark, with the house on its hill along the bottom and
  lightning now and then.

  It never fetches the calendar. Guests see the night, the moon, the
  music, timers from the kitchen and the last trains home, and nothing
  about anyone's week.
*/

// Lightning every 20 to 60 seconds: two soft pulses, never a strobe.
// ?calm in the address turns it off.
function useLightning(on: boolean): number {
  const [strikes, setStrikes] = useState(0)
  useEffect(() => {
    if (!on) return
    let timer = 0
    const next = (wait: number) => {
      timer = window.setTimeout(() => {
        setStrikes((n) => n + 1)
        next(20000 + Math.random() * 40000)
      }, wait)
    }
    next(6000)
    return () => window.clearTimeout(timer)
  }, [on])
  return strikes
}

function Countdown() {
  const now = useNow(1000)
  const left = minutesToMidnight(now)
  if (left === null) return null
  const secs = Math.max(0, Math.round((new Date(now).setHours(24, 0, 0, 0) - now.getTime()) / 1000))
  const h = Math.floor(secs / 3600)
  const m = String(Math.floor((secs % 3600) / 60)).padStart(2, '0')
  const s = String(secs % 60).padStart(2, '0')
  return (
    <div className="h-countdown">
      <p className="k-from">Until midnight</p>
      <p className="h-count">{h > 0 ? `${h}:${m}:${s}` : `${m}:${s}`}</p>
    </div>
  )
}

function Mark() {
  return (
    <p className="h-mark">
      <svg viewBox="0 0 26 26" aria-hidden="true">
        <circle cx="13" cy="13" r="11.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
        <circle cx="9.5" cy="10.5" r="1.6" fill="currentColor" />
        <circle cx="16.5" cy="10.5" r="1.6" fill="currentColor" />
        <circle cx="13" cy="16.5" r="1.6" fill="currentColor" />
      </svg>
      The Haunted Coconut
    </p>
  )
}

// Gilt scrollwork for the corners of the room.
function Flourish({ corner }: { corner: 'tl' | 'tr' | 'bl' | 'br' }) {
  return (
    <svg className={`h-flourish is-${corner}`} viewBox="0 0 120 120" aria-hidden="true">
      <path d="M4 116 C4 60 20 20 60 8 C84 2 104 6 116 4" />
      <path d="M12 108 C18 72 38 44 68 32 C88 24 98 38 86 46 C76 52 70 42 78 38" />
      <path d="M4 116 C20 104 30 90 28 76 C26 66 14 68 16 76" />
      <path d="M40 40 C46 30 56 26 62 30 C56 32 50 36 46 44 Z" className="is-fill" />
      <circle cx="4" cy="116" r="3" className="is-fill" />
    </svg>
  )
}

export function Haunted() {
  const now = useNow(60 * 1000)
  const wx = useWidgetData<WeatherData>('/api/weather', 10 * 60 * 1000)
  const widgets = useEnabledWidgets()
  const message = useMessages()
  const playing = useNowPlaying()
  const timers = useTimers()
  const calm = new URLSearchParams(window.location.search).has('calm')
  const strikes = useLightning(!calm)
  const visitor = useVisitor()
  const moon = moonOn(now)
  const raining = rainAmount(wx.data?.code) > 0

  const side = message ? 'message' : playing ? 'playing' : null
  const hasSide = side !== null || timers.length > 0
  const columns = [
    'minmax(0, 0.9fr)',
    ...(hasSide ? ['minmax(0, 1fr)'] : []),
    ...widgets.map((w) => (w.width === 'fit' ? 'auto' : 'minmax(calc(var(--u) * 42), 1fr)')),
  ].join(' ')

  return (
    <div className="terrain haunted">
      <Stars />
      <Sky visitor={visitor} />
      <Flourish corner="tl" />
      <Flourish corner="tr" />
      <Flourish corner="bl" />
      <Flourish corner="br" />

      <header className="h-top">
        <div className="h-meta">
          <p className="t-date">{dateLine(now, wx.data)}</p>
          <Mark />
        </div>
        <h1 className="h-headline">{hauntedLine(now, raining)}</h1>
      </header>

      <div className="h-middle" style={{ gridTemplateColumns: columns }}>
        <section className="t-list h-tonight">
          <h2 className="t-heading">Tonight</h2>
          <div className="h-moonrow">
            <svg className="h-moon" viewBox="0 0 100 100" aria-hidden="true">
              <defs>
                <radialGradient id="h-halo">
                  <stop offset="0.6" stopColor="#C4B4EB" stopOpacity="0.28" />
                  <stop offset="1" stopColor="#C4B4EB" stopOpacity="0" />
                </radialGradient>
              </defs>
              <circle cx="50" cy="50" r="62" className="h-moon-halo" />
              <circle cx="50" cy="50" r="40" className="h-moon-dark" />
              <path d={moonPath(moon.phase, 50, 50, 40)} className="h-moon-lit" />
            </svg>
            <div>
              <p className="h-moon-name">{moon.name}</p>
              <p className="t-soft">{Math.round(moon.lit * 100)}% lit</p>
            </div>
          </div>
          <Countdown />
        </section>
        {hasSide && (
          <div className="h-side">
            {timers.length > 0 && <Timers timers={timers} />}
            {side === 'message' && message && (
              <section className="t-list k-said">
                <h2 className="t-heading">A message from beyond</h2>
                {message.source && <p className="k-from">From {message.source}</p>}
                <p className="h-words">{message.text}</p>
              </section>
            )}
            {side === 'playing' && playing && <Playing np={playing} heading="On the phonograph" />}
          </div>
        )}
        {widgets.map((w) => (
          <WidgetSlot key={w.id} widget={w} />
        ))}
      </div>

      <Grounds flash={strikes} visitor={visitor} />
      {strikes > 0 && <div key={strikes} className="h-flash" aria-hidden="true" />}
    </div>
  )
}
