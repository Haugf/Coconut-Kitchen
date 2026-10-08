import { useWidgetData } from '../../lib/useWidgetData'

interface TransitRow {
  kind: 'subway' | 'bus'
  route: string
  label: string
  stopName: string
  minutes: number[]
  ok: boolean
}

interface TransitData {
  updated: string
  rows: TransitRow[]
}

function times(row: TransitRow): string {
  if (!row.ok) return 'No data'
  if (row.minutes.length === 0) return 'None soon'
  const parts = row.minutes.map((m) => (m <= 0 ? 'now' : String(m)))
  const last = row.minutes[row.minutes.length - 1]
  return last <= 0 ? parts.join(', ') : `${parts.join(', ')} min`
}

/** "Myrtle–Wyckoff, Forest Av and Gates & Fairview" */
function stopsSentence(rows: TransitRow[]): string {
  const names = [...new Set(rows.map((r) => r.stopName).filter(Boolean))]
  if (names.length <= 1) return names[0] ?? ''
  return `${names.slice(0, -1).join(', ')} and ${names[names.length - 1]}`
}

export function Transit() {
  // Arrival times move quickly; check every 30 seconds.
  const { data } = useWidgetData<TransitData>('/api/transit', 30 * 1000)
  if (!data || data.rows.length === 0) return null

  return (
    <section className="t-list t-transit">
      <h2 className="t-heading">Getting around</h2>
      <ul className="t-rows">
        {data.rows.map((r) => (
          <li key={`${r.kind}${r.route}${r.label}${r.stopName}`}>
            <span className="t-route">{r.route}</span>
            <span className="t-dest">to {r.label || '…'}</span>
            <span className={r.ok && r.minutes.length ? 't-times' : 't-times t-soft'}>{times(r)}</span>
          </li>
        ))}
      </ul>
      <p className="t-soft t-from">From {stopsSentence(data.rows)}.</p>
    </section>
  )
}
