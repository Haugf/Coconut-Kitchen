import { useEffect, useState } from 'react'

/*
  Visitors to the haunted grounds: now and then one silhouette passes
  through, then the night is quiet again for a minute or two. Only one
  is ever out at a time, and everything moves with CSS transforms so the
  Pi stays cool.

    zombie   shambles along the hill, stops to look at you, moves on
    cat      trots in, sits a while with its tail going, leaves
    crow     lands on a grave's cross, hops, flies off
    witch    crosses low on her broom
    hand     rises from the soil between the graves, waves, sinks
    balloon  the one thing in colour: a red balloon drifting up out of
             the hill, rarer, and more likely late at night

  ?visitor=zombie (or any name) in the address sends only that one,
  again and again, for a look. The phone settings page can summon any
  of them on demand (POST /api/settings/summon).
*/

export type Visitor = 'zombie' | 'cat' | 'crow' | 'witch' | 'hand' | 'balloon'

// How long each stays, in ms. Matches the animations in haunted.css.
export const STAY: Record<Visitor, number> = {
  zombie: 48000,
  cat: 38000,
  crow: 24000,
  witch: 22000,
  hand: 14000,
  balloon: 60000,
}

function weights(now: Date): [Visitor, number][] {
  const late = now.getHours() >= 23 || now.getHours() < 5
  return [
    ['zombie', 3],
    ['cat', 3],
    ['crow', 3],
    ['witch', 2],
    ['hand', 2],
    ['balloon', late ? 3 : 1],
  ]
}

function pick(now: Date, last: Visitor | null): Visitor {
  const options = weights(now).filter(([v]) => v !== last)
  let r = Math.random() * options.reduce((s, [, w]) => s + w, 0)
  for (const [v, w] of options) {
    if ((r -= w) < 0) return v
  }
  return options[0][0]
}

/**
 * The visitor out on the grounds right now, if any. One comes every
 * minute or two on its own; a visitor summoned from the phone settings
 * page (a "summon" event on /api/events) arrives at once, cutting off
 * whoever was there.
 */
export function useVisitor(): Visitor | null {
  const [visitor, setVisitor] = useState<Visitor | null>(null)
  useEffect(() => {
    const forced = new URLSearchParams(window.location.search).get('visitor') as Visitor | null
    const only = forced && forced in STAY ? forced : null
    let timer = 0
    let frame = 0
    let last: Visitor | null = null
    const quiet = (wait: number) => {
      window.clearTimeout(timer)
      timer = window.setTimeout(() => show(only ?? pick(new Date(), last)), wait)
    }
    const show = (v: Visitor) => {
      window.clearTimeout(timer)
      last = v
      // clear first, so the same visitor twice in a row starts over
      setVisitor(null)
      frame = requestAnimationFrame(() => setVisitor(v))
      timer = window.setTimeout(() => {
        setVisitor(null)
        quiet(only ? 3000 : 60000 + Math.random() * 90000)
      }, STAY[v])
    }
    quiet(only ? 1000 : 25000)

    let source: EventSource | null = null
    const onSummon = (e: MessageEvent) => {
      try {
        const v = (JSON.parse(e.data) as { text: string }).text as Visitor
        if (v in STAY) show(v)
      } catch {
        /* ignore malformed events */
      }
    }
    const connect = () => {
      source?.close()
      source = new EventSource('/api/events')
      source.addEventListener('summon', onSummon)
    }
    const onVisible = () => {
      if (document.visibilityState === 'visible') connect()
    }
    connect()
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      window.clearTimeout(timer)
      cancelAnimationFrame(frame)
      document.removeEventListener('visibilitychange', onVisible)
      source?.close()
    }
  }, [])
  return visitor
}

// Swaps a pose at set moments (ms after arriving), in step with the CSS path.
function usePose<T extends string>(steps: [number, T][], first: T): T {
  const [pose, setPose] = useState<T>(first)
  useEffect(() => {
    const timers = steps.map(([at, p]) => window.setTimeout(() => setPose(p), at))
    return () => timers.forEach((t) => window.clearTimeout(t))
    // the steps are constant for each visitor
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  return pose
}

/* Each is drawn in the grounds' 1200x200 space, feet at 0,0. */

function Zombie() {
  // walks 29.4 s, stops 8 s to look at you, then walks on
  const pose = usePose([[29400, 'looking'], [37400, 'walking']], 'walking')
  return (
    <g className="hv-path hv-zombie-path">
      <g className={`hv-zombie is-${pose}`} transform="scale(1.5)">
        <g className="hv-limp">
          <g className="hv-leg is-back"><path d="M-1.5 -13 L1.5 -13 L1.6 -1 L4 -1 L4 1 L-1.4 1 Z" /></g>
          <g className="hv-leg is-front"><path d="M-1.5 -13 L1.5 -13 L1.6 -1 L4 -1 L4 1 L-1.4 1 Z" /></g>
          {/* a hunched, ragged torso */}
          <path d="M-4 -13 L-4.5 -22 Q-2 -26 3 -25 L6 -23 L5 -16 L4 -13 L2 -11.5 L0 -13 L-2 -11.8 Z" />
          {/* arms out in front, one lower, hands hanging */}
          <g className="hv-arms">
            <path d="M2 -23 L15 -21.5 L16.5 -19 L15.5 -18 L14 -19.6 L2 -20.5 Z" />
            <path d="M3 -20 L14 -16.5 L15 -14 L14 -13.4 L12.6 -14.8 L2.6 -17.4 Z" />
          </g>
          <g className="hv-head">
            <path d="M3 -24.5 Q1 -27 2.4 -30 Q4.6 -33.6 8 -31.6 Q9.6 -30 9 -27.4 Q8 -25 5.6 -24.4 Z" />
            <circle className="hv-eye" cx="6.4" cy="-28.8" r="1" />
            <circle className="hv-eye" cx="8.4" cy="-28.6" r="0.9" />
          </g>
        </g>
      </g>
    </g>
  )
}

function Cat() {
  // trots in from the right, sits by the graves, trots off
  const pose = usePose([[20000, 'sitting'], [34000, 'walking']], 'walking')
  return (
    <g className="hv-path hv-cat-path">
      <g className={`hv-cat is-${pose}`}>
        {pose === 'sitting' ? (
          <g>
            <path d="M-5 0 Q-7 -6 -4 -11 Q-3 -14 -1 -15 L-1.4 -18 L0.6 -16 L2.4 -16 L3.8 -18.4 L4 -15 Q5.6 -13 4.4 -10.6 Q6.4 -6 5 0 Z" />
            <g className="hv-tail is-swish"><path d="M4 -1 Q10 -1 11 -5 Q12 -9 9.6 -11 Q11 -7 9 -3.6 Q7.6 -1.8 4 0.6 Z" /></g>
          </g>
        ) : (
          <g className="hv-trot">
            <path d="M-10 -8 Q-10 -11 -6 -11 L4 -11 L6 -13.6 L6.4 -16 L8 -14 L9.6 -14 L11 -16.2 L11.2 -13 Q12.4 -11 11 -9 Q9 -7.4 6 -7.6 L-8 -7 Z" />
            <g className="hv-leg is-front"><path d="M6 -8 L7.6 -8 L7.4 0 L6 0 Z" /></g>
            <g className="hv-leg is-back"><path d="M3.6 -8 L5 -8 L4.8 0 L3.4 0 Z" /></g>
            <g className="hv-leg is-back"><path d="M-8.4 -8 L-7 -8 L-7.2 0 L-8.6 0 Z" /></g>
            <g className="hv-leg is-front"><path d="M-6 -8 L-4.6 -8 L-4.8 0 L-6.2 0 Z" /></g>
            <path d="M-9.6 -10 Q-14 -11 -15 -16 Q-15.4 -19 -13.6 -20 Q-14 -16 -12 -13.4 Q-10.4 -11.6 -9 -11.4 Z" />
          </g>
        )}
      </g>
    </g>
  )
}

function Crow() {
  // flies in, perches on a cross, hops twice, flies off
  const pose = usePose([[4800, 'perched'], [19200, 'flying']], 'flying')
  return (
    <g className="hv-path hv-crow-path">
      <g className={`hv-crow is-${pose}`} transform="scale(1.3)">
        {pose === 'perched' ? (
          <g className="hv-hop">
            <path d="M-6 -1 L-9 2 L-4 -1.4 Q-3 -6 1 -7.4 Q2.6 -9.6 4.6 -8.6 L7.4 -8 L4.8 -7 Q4.4 -3 1 -1 Z" />
            <path className="hv-thin" d="M0 -1 L-0.4 1.4 M1.6 -1.4 L1.6 1.4" />
          </g>
        ) : (
          <g className="hv-flap">
            <path d="M-8 0 L-12 -1 L-8 -2 Q-3 -3 2 -2 L5 -2.4 L7 -1.4 L4 -0.6 Q0 1 -8 0 Z" />
            <path className="hv-wing" d="M-3 -2 Q-6 -10 -12 -13 Q-4 -12 1 -2.4 Z" />
          </g>
        )}
      </g>
    </g>
  )
}

function Witch() {
  // she flies right to left, so she's drawn facing left
  return (
    <g className="hv-path hv-witch-path">
      <g transform="scale(1.3)">
      <g className="hv-witch">
        <path className="hv-thin" d="M-16 2 L22 -2" />
        <path d="M20 -2.4 L32 -8 L33 -4 L34 0 L33 4 L21 -0.6 Z" />
        {/* cloak streaming back, body, hat */}
        <g className="hv-cloak"><path d="M-1 -2 Q6 -6 12 -6 Q18 -6 24 -11 Q20 -2 12 0 L2 1 Z" /></g>
        <path d="M-4 1 L-3 -10 Q-1 -14 3 -12 L4 -2 Z" />
        <path d="M-2 -2 L-9 1.6 L-9 2.6 L-2 0.6 Z" />
        <circle cx="0" cy="-15" r="3" />
        <path d="M-6 -16 L7 -17.4 L6.6 -16.2 L2.4 -16 L7 -27 L-1 -16.4 Z" />
      </g>
      </g>
    </g>
  )
}

function Hand() {
  // between two graves; the soil hides it until it rises
  return (
    <g className="hv-hand-spot" clipPath="url(#hv-soil)">
      <g className="hv-hand">
        <g className="hv-wave">
          <path d="M-2.4 10 L-2 -6 L2.6 -6 L3 10 Z" />
          <path d="M-3.4 -5 L-3.8 -11 L-5.6 -18 L-4.2 -18.4 L-2.4 -12 L-2.6 -19.6 L-1 -19.8 L-0.6 -12.6 L0 -20.4 L1.6 -20.2 L1.2 -12.4 L2.6 -19.2 L4.2 -18.8 L3 -11.6 L4.2 -9 L6.6 -11.8 L7.6 -10.8 L4.4 -5.4 Z" />
        </g>
      </g>
    </g>
  )
}

/** The visitor's figure inside the grounds' SVG. The balloon draws itself in Sky. */
export function OnTheGrounds({ visitor }: { visitor: Visitor | null }) {
  return (
    <>
      <defs>
        <clipPath id="hv-soil">
          <rect x="-40" y="-60" width="80" height="60" />
        </clipPath>
      </defs>
      {visitor === 'zombie' && <Zombie key="zombie" />}
      {visitor === 'cat' && <Cat key="cat" />}
      {visitor === 'crow' && <Crow key="crow" />}
      {visitor === 'witch' && <Witch key="witch" />}
      {visitor === 'hand' && <Hand key="hand" />}
    </>
  )
}

/** The whole-page layer behind the words, where the balloon rises. */
export function Sky({ visitor }: { visitor: Visitor | null }) {
  return visitor === 'balloon' ? <Balloon /> : null
}

function Balloon() {
  // somewhere behind the middle of the page, chosen once as it appears
  const [x] = useState(() => 380 + Math.round(Math.random() * 520))
  return (
    <svg className="hv-sky" viewBox="0 0 1280 720" preserveAspectRatio="xMidYMid slice" aria-hidden="true">
      <defs>
        <radialGradient id="hv-red" cx="0.36" cy="0.32" r="0.75">
          <stop offset="0" stopColor="#E0453F" />
          <stop offset="0.55" stopColor="#B81D25" />
          <stop offset="1" stopColor="#6E0D14" />
        </radialGradient>
      </defs>
      <g transform={`translate(${x} 780)`}>
        <g className="hv-rise">
          <g className="hv-sway">
            <path className="hv-string" d="M0 24 C-5 40 6 54 -1 70 C-5 80 3 90 0 100" />
            <path d="M-2.6 26 L2.6 26 L0 21.6 Z" fill="#6E0D14" />
            <ellipse cx="0" cy="0" rx="17" ry="21" fill="url(#hv-red)" />
            <ellipse cx="-6" cy="-8" rx="3.4" ry="6" fill="#FFFFFF" opacity="0.22" transform="rotate(20 -6 -8)" />
          </g>
        </g>
      </g>
    </svg>
  )
}
