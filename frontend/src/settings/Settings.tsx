import '@fontsource/young-serif/latin-400.css'
import '@fontsource-variable/figtree'
import { useEffect, useState } from 'react'
import './settings.css'

/*
  The phone page at /settings. Anyone on the home Wi-Fi can put a message
  on the mirror, fix a calendar link, pick the style, summon a visitor to the
  haunted page, or switch the screen.
  The server only answers on the home network and never sends a calendar's
  secret address back, just its host.
*/

interface Person {
  name: string
  links: { host: string; real: boolean }[]
  events: number
  failed: boolean
}

interface SettingsData {
  people: Person[]
  style: string
  sleep: { from: string; to: string }
  screenControl: boolean
  // The Google speaker messages can be said on; empty when none is found.
  speaker?: string
}

const MINUTES = [5, 10, 30, 60]

function remembered(key: string): string {
  try {
    return localStorage.getItem(key) ?? ''
  } catch {
    return ''
  }
}

function remember(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* private mode: fine */
  }
}

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms))

function status(p: Person): string {
  if (p.links.filter((l) => l.real).length === 0) return 'No calendar yet.'
  if (p.failed) return 'The mirror can’t read one of these right now.'
  const n = p.events
  return `Working: ${n === 0 ? 'nothing' : n === 1 ? '1 event' : `${n} events`} in the next week.`
}

export default function Settings() {
  const [data, setData] = useState<SettingsData | null>(null)
  const [locked, setLocked] = useState<{ pinSet: boolean } | null>(null)
  const [loadError, setLoadError] = useState('')
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)

  const [text, setText] = useState('')
  const [from, setFrom] = useState(remembered('mirror-from'))
  const [minutes, setMinutes] = useState(10)
  const [speak, setSpeak] = useState(remembered('mirror-speak') !== 'no')
  const [links, setLinks] = useState<Record<string, string>>({})
  const [sleepFrom, setSleepFrom] = useState('')
  const [sleepTo, setSleepTo] = useState('')

  async function load() {
    try {
      const res = await fetch('/api/settings', { cache: 'no-store' })
      if (res.status === 401) {
        const why = (await res.json()) as { pinSet: boolean }
        setLocked({ pinSet: why.pinSet })
        setData(null)
        return
      }
      if (!res.ok) throw new Error((await res.text()).trim() || res.statusText)
      const d = (await res.json()) as SettingsData
      setLocked(null)
      setData(d)
      setLoadError('')
      setSleepFrom(d.sleep.from)
      setSleepTo(d.sleep.to)
      if (!from && d.people[0]) setFrom(d.people[0].name)
    } catch (e) {
      setLoadError(String((e as Error).message ?? e))
    }
  }

  useEffect(() => {
    void load()
  }, [])

  /** After a saved change the server restarts; wait for it to answer again. */
  async function backAgain() {
    await wait(1500)
    for (let i = 0; i < 30; i++) {
      try {
        const res = await fetch('/api/settings', { cache: 'no-store' })
        if (res.ok || res.status === 401) return true
      } catch {
        /* still restarting */
      }
      await wait(1000)
    }
    return false
  }

  async function post(path: string, body: unknown, done: string): Promise<boolean> {
    setBusy(true)
    setNote('')
    try {
      const res = await fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
      if (res.status === 401) {
        await load()
        return false
      }
      if (!res.ok) {
        setNote((await res.text()).trim() || 'That didn’t work.')
        return false
      }
      const answer = (await res.json()) as { restarting?: boolean }
      if (answer.restarting) {
        setNote('Saved. The mirror is restarting…')
        const back = await backAgain()
        await load()
        setNote(back ? done : 'Saved, but the mirror is taking a while to come back. Check it in a minute.')
      } else {
        setNote(done)
      }
      return true
    } catch {
      setNote('The mirror didn’t answer. Are you on the home Wi-Fi?')
      return false
    } finally {
      setBusy(false)
    }
  }

  if (locked) {
    return (
      <PinScreen
        pinSet={locked.pinSet}
        onUnlocked={() => {
          setLocked(null)
          void load()
        }}
      />
    )
  }

  if (!data) {
    return (
      <main className="settings">
        <Header />
        <p className="s-soft">{loadError ? `Can’t reach the mirror: ${loadError}` : 'Loading…'}</p>
      </main>
    )
  }

  return (
    <main className="settings">
      <Header />

      <section className="s-section" aria-labelledby="say">
        <h2 id="say">Say something</h2>
        <form
          className="s-stack"
          onSubmit={async (e) => {
            e.preventDefault()
            if (!text.trim()) {
              setNote('Write something first.')
              return
            }
            remember('mirror-from', from)
            const aloud = speak && !!data.speaker
            if (data.speaker) remember('mirror-speak', speak ? 'yes' : 'no')
            if (await post('/api/settings/say', { text, from, minutes, speak: aloud }, aloud ? `On the mirror and ${data.speaker}.` : 'On the mirror.')) setText('')
          }}
        >
          <label className="s-label" htmlFor="say-text">Message</label>
          <input id="say-text" className="s-input" value={text} maxLength={280} placeholder="Rice is done." onChange={(e) => setText(e.target.value)} />
          <div className="s-row">
            <label className="s-label" htmlFor="say-from">From</label>
            <select id="say-from" className="s-input s-select" value={from} onChange={(e) => setFrom(e.target.value)}>
              {data.people.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.name}
                </option>
              ))}
              <option value="">No name</option>
            </select>
          </div>
          <fieldset className="s-chips">
            <legend className="s-label">Stays up for</legend>
            {MINUTES.map((m) => (
              <button type="button" key={m} className="s-chip s-chip-wide" aria-pressed={minutes === m} onClick={() => setMinutes(m)}>
                {m < 60 ? `${m} min` : '1 hour'}
              </button>
            ))}
          </fieldset>
          {data.speaker && (
            <label className="s-check">
              <input type="checkbox" checked={speak} onChange={(e) => setSpeak(e.target.checked)} />
              <span>
                Say it out loud on {data.speaker}
                <small className="s-soft">Stops whatever it’s playing.</small>
              </span>
            </label>
          )}
          <button className="s-button s-primary" disabled={busy}>
            {speak && data.speaker ? 'Show and say it' : 'Show on the mirror'}
          </button>
        </form>
      </section>

      <section className="s-section" aria-labelledby="calendars">
        <h2 id="calendars">Calendars</h2>
        {data.people.length === 0 && <p className="s-soft">No one is set up yet. Add people in config.json on the Pi.</p>}
        {data.people.map((p) => (
          <div className="s-person" key={p.name}>
            <h3>{p.name}</h3>
            <p className={p.failed ? 's-warn' : 's-soft'}>{status(p)}</p>
            <ul className="s-links">
              {p.links.map((l, i) => (
                <li key={i}>
                  <span>
                    {p.links.length > 1 && <span className="s-num">Calendar {i + 1} · </span>}
                    {l.real ? l.host : 'Not a link yet'}
                  </span>
                  <button
                    type="button"
                    className="s-button s-quiet"
                    disabled={busy}
                    onClick={() => {
                      if (window.confirm(`Remove this calendar from ${p.name}?`)) {
                        void post('/api/settings/calendar', { person: p.name, remove: i }, 'Removed.')
                      }
                    }}
                  >
                    Remove
                  </button>
                </li>
              ))}
            </ul>
            <form
              className="s-stack"
              onSubmit={async (e) => {
                e.preventDefault()
                const link = (links[p.name] ?? '').trim()
                if (!link) {
                  setNote('Paste a calendar link first.')
                  return
                }
                if (await post('/api/settings/calendar', { person: p.name, add: link }, `Added to ${p.name}.`)) {
                  setLinks({ ...links, [p.name]: '' })
                }
              }}
            >
              <label className="s-label" htmlFor={`link-${p.name}`}>
                Add a calendar link
              </label>
              <input
                id={`link-${p.name}`}
                className="s-input"
                inputMode="url"
                autoComplete="off"
                placeholder="webcal://… or https://…"
                value={links[p.name] ?? ''}
                onChange={(e) => setLinks({ ...links, [p.name]: e.target.value })}
              />
              <button className="s-button" disabled={busy}>
                Add
              </button>
            </form>
          </div>
        ))}
        <details className="s-help">
          <summary>Where do I find the link?</summary>
          <p>
            <strong>iPhone (iCloud):</strong> Calendar › Calendars › the (i) next to the calendar your events go in › turn on Public
            Calendar › Share Link › Copy.
          </p>
          <p>
            <strong>Google:</strong> on a computer, Google Calendar › Settings › the calendar › Secret address in iCal format.
          </p>
          <p>Calendars stored “On My iPhone” can’t be shared. Each calendar has its own link, so pick the one your events go in.</p>
        </details>
      </section>

      <section className="s-section" aria-labelledby="look">
        <h2 id="look">Look</h2>
        <div className="s-chips">
          {[
            ['kitchen', 'Coconut Kitchen'],
            ['terrain', 'Terrain'],
            ['haunted', 'Haunted'],
          ].map(([id, label]) => (
            <button
              type="button"
              key={id}
              className="s-chip s-chip-wide"
              aria-pressed={data.style === id}
              disabled={busy || data.style === id}
              onClick={() => void post('/api/settings/style', { style: id }, `Switched to ${label}.`)}
            >
              {label}
            </button>
          ))}
        </div>
      </section>

      {data.style === 'haunted' && (
        <section className="s-section" aria-labelledby="visitors">
          <h2 id="visitors">Visitors</h2>
          <div className="s-chips">
            {[
              ['zombie', 'Zombie'],
              ['balloon', 'Red balloon'],
              ['cat', 'Black cat'],
              ['crow', 'Crow'],
              ['witch', 'Witch'],
              ['hand', 'Hand'],
              ['bats', 'Bats'],
            ].map(([id, label]) => (
              <button
                type="button"
                key={id}
                className="s-chip"
                disabled={busy}
                onClick={() => void post('/api/settings/summon', { visitor: id }, `${label} on the way.`)}
              >
                {label}
              </button>
            ))}
          </div>
          <p className="s-soft">Sends one across the grounds right now.</p>
        </section>
      )}

      <section className="s-section" aria-labelledby="screen">
        <h2 id="screen">Screen</h2>
        {data.screenControl && (
          <div className="s-chips">
            <button type="button" className="s-chip s-chip-wide" disabled={busy} onClick={() => void post('/api/settings/screen', { on: false }, 'Screen off.')}>
              Turn off now
            </button>
            <button type="button" className="s-chip s-chip-wide" disabled={busy} onClick={() => void post('/api/settings/screen', { on: true }, 'Screen on.')}>
              Turn on
            </button>
          </div>
        )}
        <p className="s-soft">
          {data.sleep.from && data.sleep.to ? `Sleeps from ${data.sleep.from} to ${data.sleep.to}.` : 'Stays on all night.'}
        </p>
        <form
          className="s-stack"
          onSubmit={(e) => {
            e.preventDefault()
            if (!sleepFrom || !sleepTo) {
              setNote('Pick both times, or choose “Keep it on all night”.')
              return
            }
            void post('/api/settings/sleep', { from: sleepFrom, to: sleepTo }, `Sleeps from ${sleepFrom} to ${sleepTo}.`)
          }}
        >
          <div className="s-pair">
            <div className="s-stack">
              <label className="s-label" htmlFor="sleep-from">Sleep from</label>
              <input id="sleep-from" type="time" className="s-input" value={sleepFrom} onChange={(e) => setSleepFrom(e.target.value)} />
            </div>
            <div className="s-stack">
              <label className="s-label" htmlFor="sleep-to">Wake at</label>
              <input id="sleep-to" type="time" className="s-input" value={sleepTo} onChange={(e) => setSleepTo(e.target.value)} />
            </div>
          </div>
          <button className="s-button" disabled={busy}>
            Save sleep hours
          </button>
          <button
            type="button"
            className="s-button s-quiet"
            disabled={busy}
            onClick={() => void post('/api/settings/sleep', { from: '', to: '' }, 'Stays on all night.')}
          >
            Keep it on all night
          </button>
        </form>
      </section>

      <section className="s-section">
        <button
          type="button"
          className="s-button s-quiet"
          onClick={async () => {
            await fetch('/api/settings/lock', { method: 'POST' }).catch(() => {})
            setNote('')
            await load()
          }}
        >
          Lock settings on this phone
        </button>
      </section>

      <p className="s-note" role="status" aria-live="polite">
        {note}
      </p>
    </main>
  )
}

function PinScreen({ pinSet, onUnlocked }: { pinSet: boolean; onUnlocked: () => void }) {
  const [pin, setPin] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  if (!pinSet) {
    return (
      <main className="settings">
        <Header />
        <section className="s-section">
          <h2>Set a PIN first</h2>
          <p className="s-soft">
            Settings stay locked until the mirror has a PIN. On the Pi, run <code>mirror pin</code> followed by the PIN you want, then
            reload this page.
          </p>
        </section>
      </main>
    )
  }

  return (
    <main className="settings">
      <Header />
      <form
        className="s-section s-stack"
        onSubmit={async (e) => {
          e.preventDefault()
          if (!pin.trim()) {
            setError('Enter the PIN first.')
            return
          }
          setBusy(true)
          setError('')
          try {
            const res = await fetch('/api/settings/unlock', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ pin }),
            })
            if (res.ok) {
              onUnlocked()
              return
            }
            setError((await res.text()).trim() || 'That didn’t work.')
            setPin('')
          } catch {
            setError('The mirror didn’t answer. Are you on the home Wi-Fi?')
          } finally {
            setBusy(false)
          }
        }}
      >
        <h2>Enter the PIN</h2>
        <p className="s-soft">This phone will remember it for a month.</p>
        <label className="s-label" htmlFor="pin">
          PIN
        </label>
        <input
          id="pin"
          className="s-input s-pin"
          type="password"
          autoCapitalize="characters"
          autoComplete="off"
          autoCorrect="off"
          spellCheck={false}
          value={pin}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? 'pin-error' : undefined}
          onChange={(e) => {
            setPin(e.target.value)
            setError('')
          }}
        />
        {error && (
          <p id="pin-error" className="s-warn" role="alert">
            {error}
          </p>
        )}
        <button className="s-button s-primary" disabled={busy}>
          Unlock
        </button>
      </form>
    </main>
  )
}

function Header() {
  return (
    <header className="s-header">
      <svg className="s-mark" viewBox="0 0 26 26" aria-hidden="true">
        <circle cx="13" cy="13" r="11.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
        <circle cx="9.5" cy="10.5" r="1.6" fill="currentColor" />
        <circle cx="16.5" cy="10.5" r="1.6" fill="currentColor" />
        <circle cx="13" cy="16.5" r="1.6" fill="currentColor" />
      </svg>
      <div>
        <p className="s-brand">Coconut Kitchen</p>
        <h1>Mirror settings</h1>
      </div>
    </header>
  )
}
