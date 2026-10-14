/*
  Rain on the glass: when it's raining outside, the kitchen page sits
  behind a rainy window. The lake behind the glass is painted once, out of
  focus, into the husk colours; only a few drops running down the glass
  move, at about 20 frames a second, so the Pi stays cool.
*/

/** How hard it's raining, 0 to 1, from the WMO weather code. 0 is dry. */
export function rainAmount(code: number | undefined): number {
  if (code === undefined) return 0
  if (code >= 51 && code <= 57) return 0.4 // drizzle
  if (code === 61 || code === 80) return 0.6 // light rain, showers
  if (code === 63 || code === 66 || code === 81) return 0.8
  if (code === 65 || code === 67 || code === 82) return 1 // heavy
  if (code >= 95) return 1 // thunderstorm
  return 0
}

// A small seeded random, so the lake and the drops look the same on
// every reload.
function seeded(seed: number) {
  let s = seed
  return () => (s = (s * 16807) % 2147483647) / 2147483647
}

/**
 * Paints a misty river between wooded banks. It's only ever seen blurred
 * through the glass, so soft blobs of green are enough; no photo needed.
 */
export function paintLake(c: HTMLCanvasElement, night: boolean) {
  const W = c.width
  const H = c.height
  const base = c.getContext('2d')!
  const rnd = seeded(7)
  const layer = () => {
    const t = document.createElement('canvas')
    t.width = W
    t.height = H
    return t
  }
  const blob = (g: CanvasRenderingContext2D, x: number, y: number, r: number, col: string, a: number) => {
    g.globalAlpha = a
    g.fillStyle = col
    g.beginPath()
    g.arc(x, y, r, 0, Math.PI * 2)
    g.fill()
  }

  const sky = base.createLinearGradient(0, 0, 0, H * 0.55)
  sky.addColorStop(0, night ? '#1a2320' : '#c9d3c9')
  sky.addColorStop(1, night ? '#24302a' : '#9fb0a0')
  base.fillStyle = sky
  base.fillRect(0, 0, W, H)

  const greens = night ? ['#13201a', '#183026', '#1d382b', '#0f1a14'] : ['#5f7f5c', '#4a6d48', '#6f8f62', '#3e5f3f', '#7d9b6c']
  // Trees on both banks, closing in towards the bend.
  const trees = layer()
  const tg = trees.getContext('2d')!
  for (let i = 0; i < 160; i++) blob(tg, rnd() * W, H * (0.18 + rnd() * 0.3), W * (0.02 + rnd() * 0.04), greens[i % greens.length], 0.35)
  for (const side of [0, 1]) {
    for (let i = 0; i < 260; i++) {
      const t = rnd()
      const x = side ? W * (0.6 + t * 0.45) : W * (0.42 - t * 0.47)
      const y = H * (0.3 + t * 0.42) + (rnd() - 0.5) * H * 0.35 * (0.4 + t)
      blob(tg, x, y, W * (0.012 + rnd() * 0.035) * (0.5 + t), greens[(i * 7) % greens.length], 0.55)
    }
  }
  base.globalAlpha = 1
  base.filter = `blur(${W / 220}px)`
  base.drawImage(trees, 0, 0)
  base.filter = 'none'

  // The river, widening from the bend to the bottom of the window.
  const river = layer()
  const rg = river.getContext('2d')!
  rg.beginPath()
  rg.moveTo(W * 0.47, H * 0.5)
  rg.lineTo(W * 0.55, H * 0.5)
  rg.bezierCurveTo(W * 0.62, H * 0.66, W * 0.86, H * 0.8, W * 0.96, H)
  rg.lineTo(W * 0.06, H)
  rg.bezierCurveTo(W * 0.24, H * 0.8, W * 0.42, H * 0.66, W * 0.47, H * 0.5)
  const water = rg.createLinearGradient(0, H * 0.5, 0, H)
  water.addColorStop(0, night ? 'rgba(43,58,53,0)' : 'rgba(196,207,196,0)')
  water.addColorStop(0.12, night ? '#2b3a35' : '#c4cfc4')
  water.addColorStop(0.5, night ? '#1d2a25' : '#8fa294')
  water.addColorStop(1, night ? '#16201c' : '#6f8676')
  rg.fillStyle = water
  rg.fill()
  base.filter = `blur(${W / 260}px)`
  base.drawImage(river, 0, 0)
  base.filter = 'none'
}

interface Drop {
  x: number
  y: number
  r: number
  v: number
  trail: [number, number][]
}

/**
 * Starts the rain on two canvases: `still` gets the drops that sit on the
 * glass (drawn once), `moving` the few that run down. Returns a stop
 * function.
 */
export function startRain(still: HTMLCanvasElement, moving: HTMLCanvasElement, lake: HTMLCanvasElement, amount: number): () => void {
  const W = still.width
  const H = still.height
  const rnd = seeded(99)
  const sx = lake.width / W
  const sy = lake.height / H

  // Each drop is a tiny lens: the lake behind it, flipped, with a glint.
  const drop = (g: CanvasRenderingContext2D, x: number, y: number, r: number) => {
    g.save()
    g.beginPath()
    g.ellipse(x, y, r, r * 1.12, 0, 0, Math.PI * 2)
    g.clip()
    g.translate(x, y)
    g.scale(-1, -1)
    const R = r * 4
    g.globalAlpha = 0.9
    g.drawImage(lake, Math.max(0, (x - R) * sx), Math.max(0, (y - R) * sy), R * 2 * sx, R * 2 * sy, -r, -r * 1.12, r * 2, r * 2.24)
    g.restore()
    g.fillStyle = 'rgba(255,240,220,.3)'
    g.beginPath()
    g.ellipse(x - r * 0.35, y - r * 0.45, r * 0.22, r * 0.14, -0.6, 0, Math.PI * 2)
    g.fill()
  }

  const sg = still.getContext('2d')!
  sg.clearRect(0, 0, W, H)
  for (let i = 0; i < Math.round(200 * amount); i++) drop(sg, rnd() * W, rnd() * H, 1 + Math.pow(rnd(), 4) * 7)

  const mg = moving.getContext('2d')!
  const runners: Drop[] = Array.from({ length: Math.max(2, Math.round(9 * amount)) }, () => ({
    x: rnd() * W,
    y: rnd() * H,
    r: 3.5 + rnd() * 3,
    v: 25 + rnd() * 55,
    trail: [],
  }))

  let last = performance.now()
  const step = () => {
    const now = performance.now()
    const dt = Math.min(0.1, (now - last) / 1000)
    last = now
    mg.clearRect(0, 0, W, H)
    mg.lineCap = 'round'
    for (const d of runners) {
      d.y += d.v * dt
      d.x += Math.sin(now / 400 + d.v) * 0.25
      d.trail.push([d.x, d.y])
      if (d.trail.length > 90) d.trail.shift()
      mg.strokeStyle = 'rgba(235,223,203,.09)'
      mg.lineWidth = d.r * 0.9
      mg.beginPath()
      d.trail.forEach(([x, y], i) => (i ? mg.lineTo(x, y) : mg.moveTo(x, y)))
      mg.stroke()
      drop(mg, d.x, d.y, d.r)
      if (d.y > H + 20) {
        d.y = -20 - rnd() * 200
        d.x = rnd() * W
        d.trail = []
      }
    }
  }
  // About 20 frames a second is smooth enough for drops and easy on the Pi.
  const timer = window.setInterval(step, 50)
  step()
  return () => window.clearInterval(timer)
}
