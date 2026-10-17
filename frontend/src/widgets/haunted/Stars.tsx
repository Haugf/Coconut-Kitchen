/*
  The night sky behind the haunted page: a few hundred stars, a faint
  band of the Milky Way, and a handful that twinkle. Placed with a seeded
  random so the sky is the same every time the page loads.
*/

function seeded(seed: number) {
  let s = seed
  return () => (s = (s * 16807) % 2147483647) / 2147483647
}

const rnd = seeded(31)
const STARS = Array.from({ length: 260 }, (_, i) => {
  const big = rnd() < 0.08
  return {
    x: rnd() * 1280,
    // more of them high up, thinning towards the horizon
    y: Math.pow(rnd(), 1.35) * 640,
    r: big ? 1.3 + rnd() * 0.7 : 0.45 + rnd() * 0.6,
    o: big ? 0.85 : 0.3 + rnd() * 0.5,
    twinkle: i % 13 === 0,
    delay: -rnd() * 6,
    warm: rnd() < 0.15,
  }
})

export function Stars() {
  return (
    <svg className="h-stars" viewBox="0 0 1280 720" preserveAspectRatio="xMidYMid slice" aria-hidden="true">
      <defs>
        <linearGradient id="h-night" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#05040B" />
          <stop offset="0.6" stopColor="#0C0A18" />
          <stop offset="1" stopColor="#1B1430" />
        </linearGradient>
        <radialGradient id="h-milky" cx="0.5" cy="0.5" r="0.5">
          <stop offset="0" stopColor="#B9B2E0" stopOpacity="0.09" />
          <stop offset="1" stopColor="#B9B2E0" stopOpacity="0" />
        </radialGradient>
      </defs>
      <rect width="1280" height="720" fill="url(#h-night)" />
      <ellipse cx="700" cy="230" rx="760" ry="120" fill="url(#h-milky)" transform="rotate(-18 700 230)" />
      {STARS.map((s, i) => (
        <circle
          key={i}
          cx={s.x}
          cy={s.y}
          r={s.r}
          fill={s.warm ? '#F3DDB8' : '#E6E8F8'}
          opacity={s.o}
          className={s.twinkle ? 'h-twinkle' : undefined}
          style={s.twinkle ? { animationDelay: `${s.delay}s` } : undefined}
        />
      ))}
    </svg>
  )
}
