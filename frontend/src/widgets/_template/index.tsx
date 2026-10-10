import { useSource, type WidgetProps } from '../../lib/widgets'
import './style.css'

// The shape of what the "sun" data source returns. Match it to your API.
interface SunData {
  daily: { sunrise: string[]; sunset: string[] }
}

// Your settings from widget.json, with their types.
interface Settings {
  latitude: number
  longitude: number
}

function clock(iso: string | undefined): string {
  if (!iso) return '…'
  return new Date(iso).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
}

export default function Example(props: WidgetProps<Settings>) {
  // Ask for the data source by its name in widget.json. The second number
  // is how often to check again, in milliseconds.
  const { data } = useSource<SunData>(props, 'sun', 30 * 60 * 1000)

  // Show nothing until there's something to show.
  if (!data) return null

  return (
    <section className="t-list w-example">
      <h2 className="t-heading">Sun</h2>
      <p>
        Up at {clock(data.daily.sunrise[0])}, down at {clock(data.daily.sunset[0])}.
      </p>
    </section>
  )
}
