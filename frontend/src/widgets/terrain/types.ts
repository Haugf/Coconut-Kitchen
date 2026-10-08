export interface CalendarEvent {
  title: string
  start: string
  end: string
  allDay: boolean
  location?: string
}

export interface WeatherDay {
  date: string
  high: number
  low: number
  code: number
  rainChance: number
}

export interface WeatherData {
  temp: number
  feelsLike: number
  code: number
  days: WeatherDay[]
}

/** A timed event with parsed times, in local hours from midnight. */
export interface DayEvent {
  title: string
  start: Date
  end: Date
  startH: number
  endH: number
}
