import { art } from './art'
import { moonOn, moonPath } from '../haunted/night'
import { CLUMPS, TIMES, clumpAt, dayToClump, dayX, electionDay, groundPath, groundY, skyColor, thanksgiving, type Sky, type Time } from './scene'

const px = (n: number) => `calc(var(--s) * ${n.toFixed(1)})`
const box = (x: number, y: number, w: number, h: number) => ({ left: px(x), top: px(y), width: px(w), height: px(h) })

export function Scene({ now, time, sky }: { now: Date; time: Time; sky: Sky }) {
  const day = now.getDate()
  const month = now.getMonth()
  const year = now.getFullYear()
  const last = new Date(year, month + 1, 0).getDate()
  const turkeyDay = month === 10 ? thanksgiving(year) : null
  const scene = TIMES[time]
  const filter = scene.filter === 'none' ? undefined : scene.filter
  const start = clumpAt(dayToClump(day))
  const fallX = start.x * 0.6 + 960 * 0.4

  return (
    <div className="pk-scene" style={{ background: skyColor(time, sky) }} aria-hidden="true">
      {sky === 'clear' && time !== 'night' && <img src={art.sun} alt="" style={box(scene.orb[0] - 70, scene.orb[1] - 70, 140, 140)} />}
      {sky === 'clear' && time === 'night' && (
        <svg className="pk-moon" viewBox="0 0 100 100" style={box(scene.orb[0] - 70, scene.orb[1] - 70, 140, 140)}>
          <circle cx="50" cy="50" r="46" fill="#232C45" />
          <path d={moonPath(moonOn(now).phase, 50, 50, 46)} fill="#EFE8D2" />
        </svg>
      )}
      {sky !== 'clear' && (
        <div className={`pk-clouds is-${time === 'night' ? 'night' : sky}`}>
          <img src={art.clouds[0]} alt="" style={box(560, 40, 500, 180)} />
          <img src={art.clouds[1]} alt="" style={box(960, 110, 500, 180)} />
          <img src={art.clouds[2]} alt="" style={box(700, 250, 500, 180)} />
        </div>
      )}
      <div className="pk-art" style={{ filter }}>
        <img src={art.skyline} alt="" style={box(460, 190, 1920, 300)} />
        <img src={art.treeline} alt="" style={box(0, 310, 1920, 220)} />
        <div className="pk-lake" style={{ ...box(0, 500, 1920, 300), background: scene.lake }} />
        <img src={art.bridge} alt="" style={box(590, 480, 620, 213)} />
        <img src={art.trunk} alt="" style={box(310, 0, 1300, 1080)} />
        {Array.from({ length: CLUMPS }, (_, i) => {
          const gone = [...Array(day).keys()].some((d) => dayToClump(d + 1) === i)
          if (gone) return null
          const { x, y } = clumpAt(i)
          return (
            <img
              key={i}
              src={art.clumps[i % art.clumps.length]}
              alt=""
              style={{ ...box(x - 130, y - 114, 260, 228), transform: `rotate(${((i * 53) % 40) - 20}deg) scaleX(${i % 2 ? -1 : 1})` }}
            />
          )
        })}
      </div>
      {(sky === 'rain' || sky === 'snow') && <div className={`pk-${sky}`} />}
      <div className="pk-wash" />
      <svg className="pk-ground" viewBox="0 0 1920 1080" preserveAspectRatio="none">
        <path d={groundPath(last, turkeyDay)} />
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
                ...box(dayX(d, last) - 22, groundY(d, turkeyDay) - 44, 44, 62),
                transform: `rotate(${62 + ((d * 37) % 56)}deg)`,
              }}
            />
          )
        })}
        {day === turkeyDay && <img src={art.table} alt="" style={box(610, 575, 600, 160)} />}
        {turkeyDay !== null && day >= turkeyDay - 7 && day <= turkeyDay && <img src={art.turkey} alt="" style={box(1225, 570, 180, 180)} />}
        {month === 10 && day === electionDay(year) && <img src={art.sticker} alt="" className="pk-sticker" style={box(1170, 40, 170, 170)} />}
      </div>
      <div className="pk-today" style={{ ...box(fallX - 40, 530, 80, 112), filter }}>
        <img src={art.leaves[0]} alt="" />
      </div>
    </div>
  )
}
