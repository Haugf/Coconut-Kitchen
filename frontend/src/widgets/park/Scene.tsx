import { art } from './art'
import { moonOn, moonPath } from '../haunted/night'
import { CLUMPS, TIMES, clumpAt, dayToClump, dayX, electionDay, groundPath, groundY, skyColor, thanksgiving, type Peak, type Sky, type Time } from './scene'

const px = (n: number) => `calc(var(--s) * ${n.toFixed(1)})`
const box = (x: number, y: number, w: number, h: number) => ({ left: px(x), top: px(y), width: px(w), height: px(h) })

export function Scene({ now, time, sky, peaks }: { now: Date; time: Time; sky: Sky; peaks: Peak[] }) {
  const day = now.getDate()
  const month = now.getMonth()
  const year = now.getFullYear()
  const last = new Date(year, month + 1, 0).getDate()
  const turkeyDay = month === 10 ? thanksgiving(year) : null
  const scene = TIMES[time]
  const filter = scene.filter === 'none' ? undefined : scene.filter
  const start = clumpAt(dayToClump(day))
  const fallX = start.x * 0.6 + dayX(day, last) * 0.4
  const ticks = [1, 8, 15, 22, last]
  const monthName = now.toLocaleDateString('en-US', { month: 'short' })

  return (
    <div className="pk-scene" style={{ background: skyColor(time, sky) }} aria-hidden="true">
      {sky === 'clear' && time !== 'night' && <img src={art.sun} alt="" style={box(scene.orb[0] - 90, scene.orb[1] - 90, 180, 180)} />}
      {sky === 'clear' && time === 'night' && (
        <svg className="pk-moon" viewBox="0 0 100 100" style={box(scene.orb[0] - 90, scene.orb[1] - 90, 180, 180)}>
          <circle cx="50" cy="50" r="46" fill="#232C45" />
          <path d={moonPath(moonOn(now).phase, 50, 50, 46)} fill="#EFE8D2" />
        </svg>
      )}
      {sky !== 'clear' && (
        <div className={`pk-clouds is-${time === 'night' ? 'night' : sky}`}>
          <img src={art.clouds[0]} alt="" style={box(560, 40, 500, 180)} />
          <img src={art.clouds[1]} alt="" style={box(960, 90, 500, 180)} />
          <img src={art.clouds[2]} alt="" style={box(1320, 330, 500, 180)} />
        </div>
      )}
      <div className="pk-art" style={{ filter }}>
        <img src={art.skyline} alt="" style={box(0, 300, 1920, 300)} />
        <img src={art.treeline} alt="" style={box(0, 450, 1920, 220)} />
        <div className="pk-lake" style={{ ...box(0, 630, 1920, 300), background: scene.lake }} />
        <img src={art.bridge} alt="" style={box(110, 570, 700, 240)} />
        <img src={art.trunk} alt="" style={box(350, 0, 1300, 1080)} />
        {Array.from({ length: CLUMPS }, (_, i) => {
          const gone = [...Array(day).keys()].some((d) => dayToClump(d + 1) === i)
          if (gone) return null
          const { x, y } = clumpAt(i)
          return (
            <img
              key={i}
              src={art.clumps[i % art.clumps.length]}
              alt=""
              style={{ ...box(x - 150, y - 131, 300, 262), transform: `rotate(${((i * 53) % 40) - 20}deg) scaleX(${i % 2 ? -1 : 1})` }}
            />
          )
        })}
      </div>
      {(sky === 'rain' || sky === 'snow') && <div className={`pk-${sky}`} />}
      <svg className="pk-ground" viewBox="0 0 1920 1080" preserveAspectRatio="none">
        <path d={groundPath(last, peaks)} fill={scene.ground} />
      </svg>
      <div className="pk-art" style={{ filter }}>
        {Array.from({ length: Math.min(day - 1, last) }, (_, i) => {
          const d = i + 1
          return (
            <img
              key={d}
              src={art.leaves[d % art.leaves.length]}
              alt=""
              style={{
                ...box(dayX(d, last) - 22, groundY(d, peaks) - 43, 44, 62),
                transform: `rotate(${62 + ((d * 37) % 56)}deg)`,
              }}
            />
          )
        })}
        {day === turkeyDay && <img src={art.table} alt="" style={box(640, 700, 760, 203)} />}
        {turkeyDay !== null && day >= turkeyDay - 7 && day <= turkeyDay && <img src={art.turkey} alt="" style={box(1520, 680, 180, 180)} />}
        {month === 10 && day === electionDay(year) && <img src={art.sticker} alt="" className="pk-sticker" style={box(500, 62, 158, 158)} />}
      </div>
      <div className="pk-today" style={{ ...box(fallX - 33, 594, 66, 92), filter }}>
        <img src={art.leaves[0]} alt="" />
      </div>
      {peaks.map((p) => (
        <p key={p.label} className="pk-label" style={{ left: px(dayX(p.day, last)) }}>
          {p.label}
        </p>
      ))}
      {ticks.map((d) => (
        <p key={d} className="pk-label pk-tick" style={{ left: px(dayX(d, last)) }}>
          {monthName} {d}
        </p>
      ))}
    </div>
  )
}
