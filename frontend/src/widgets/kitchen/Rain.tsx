import { useEffect, useRef } from 'react'
import { paintLake, startRain } from './rain'

/*
  The rainy window behind the kitchen page. Kitchen only draws it while
  it's raining (or with ?rain=1 in the address, to try it on a dry day).
*/
export function Rain({ amount, night }: { amount: number; night: boolean }) {
  const view = useRef<HTMLCanvasElement>(null)
  const still = useRef<HTMLCanvasElement>(null)
  const moving = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    if (!view.current || !still.current || !moving.current) return
    // Drops are drawn at the screen's own size so they stay round in
    // portrait too.
    for (const c of [still.current, moving.current]) {
      c.width = window.innerWidth
      c.height = window.innerHeight
    }
    // The lake, painted once and blurred once into the visible canvas.
    const lake = document.createElement('canvas')
    lake.width = 640
    lake.height = 360
    paintLake(lake, night)
    const g = view.current.getContext('2d')!
    g.filter = night ? 'blur(4px) brightness(.5) saturate(.6) sepia(.35)' : 'blur(4px) brightness(.62) saturate(.7) sepia(.3)'
    g.drawImage(lake, 0, 0)
    return startRain(still.current, moving.current, lake, amount)
  }, [amount, night])

  return (
    <div className="k-rain" aria-hidden="true">
      <canvas ref={view} width={640} height={360} />
      <div className="k-rain-wash" />
      <canvas ref={still} />
      <canvas ref={moving} />
    </div>
  )
}
