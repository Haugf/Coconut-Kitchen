import { useNow } from '../../lib/useNow'

export function Clock() {
  const now = useNow(1000)
  const hours = now.getHours() % 12 || 12
  const minutes = String(now.getMinutes()).padStart(2, '0')
  const meridiem = now.getHours() < 12 ? 'am' : 'pm'
  const date = now.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })

  return (
    <section className="clock" aria-label="Time">
      <p className="clock-time">
        {hours}
        <span className="clock-colon">:</span>
        {minutes}
        <span className="clock-meridiem">{meridiem}</span>
      </p>
      <p className="clock-date">{date}</p>
    </section>
  )
}
