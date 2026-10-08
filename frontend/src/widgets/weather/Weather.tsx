import { useWidgetData } from '../../lib/useWidgetData'
import { describe } from './codes'

interface WeatherDay {
  date: string
  high: number
  low: number
  code: number
  rainChance: number
}

interface WeatherData {
  temp: number
  feelsLike: number
  code: number
  days: WeatherDay[]
}

const round = (n: number) => Math.round(n)

function dayName(iso: string, index: number): string {
  if (index === 0) return 'Today'
  // Open-Meteo dates are local calendar dates (YYYY-MM-DD).
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString(undefined, { weekday: 'short' })
}

export function Weather() {
  const { data, error } = useWidgetData<WeatherData>('/api/weather', 10 * 60 * 1000)

  if (!data) {
    return (
      <section className="widget weather">
        <p className="muted">{error ? 'Weather unavailable. Check the weather settings in config.json.' : 'Loading weather'}</p>
      </section>
    )
  }

  const today = data.days[0]
  return (
    <section className="widget weather" aria-label="Weather">
      <div className="weather-now">
        <p className="weather-temp">{round(data.temp)}°</p>
        <div>
          <p className="weather-desc">{describe(data.code)}</p>
          <p className="muted">
            Feels like {round(data.feelsLike)}°
            {today && <>, high {round(today.high)}° low {round(today.low)}°</>}
          </p>
          {today && today.rainChance >= 30 && (
            <p className="weather-rain">{today.rainChance}% chance of rain</p>
          )}
        </div>
      </div>
      <ul className="weather-days">
        {data.days.slice(1, 4).map((day, i) => (
          <li key={day.date}>
            <span>{dayName(day.date, i + 1)}</span>
            <span className="muted">{describe(day.code)}</span>
            <span>
              {round(day.high)}° <span className="muted">{round(day.low)}°</span>
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
