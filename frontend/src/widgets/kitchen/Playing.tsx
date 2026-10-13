import { useWidgetData } from '../../lib/useWidgetData'
import { useNow } from '../../lib/useNow'

/*
  What the Google speaker is playing, from /api/cast. The Pi keeps a
  connection to the speaker, so asking it every few seconds is cheap.
*/

export interface NowPlaying {
  speaker?: string
  connected: boolean
  state?: 'playing' | 'paused' | 'buffering'
  app?: string
  title?: string
  subtitle?: string
  live?: boolean
  duration?: number
  position?: number
  at: string
}

export function useNowPlaying(): NowPlaying | null {
  const { data } = useWidgetData<NowPlaying>('/api/cast', 5000)
  return data?.state && data.title ? data : null
}

function clock(s: number): string {
  const m = Math.floor(s / 60)
  return `${m}:${String(Math.floor(s % 60)).padStart(2, '0')}`
}

export function Playing({ np }: { np: NowPlaying }) {
  const now = useNow(1000)
  let pos = np.position ?? 0
  if (np.state === 'playing') pos += Math.max(0, (now.getTime() - Date.parse(np.at)) / 1000)
  const dur = np.duration ?? 0
  if (dur > 0) pos = Math.min(pos, dur)
  const where = np.speaker ? `On ${np.speaker}` : 'Playing'

  return (
    <section className="t-list k-playing" aria-label={`${where}: ${np.title}`}>
      <h2 className="t-heading">{where}</h2>
      <p className="k-from">
        {np.state === 'paused' ? 'Paused' : np.state === 'buffering' ? 'Starting' : np.live ? 'Live' : 'Now'}
        {np.app ? ` · ${np.app}` : ''}
      </p>
      <p className="k-song">{np.title}</p>
      {np.subtitle && <p className="k-artist">{np.subtitle}</p>}
      {dur > 0 && !np.live && (
        <div className="k-progress">
          <span className="k-bar">
            <span style={{ width: `${(pos / dur) * 100}%` }} />
          </span>
          <span className="k-time">
            {clock(pos)} of {clock(dur)}
          </span>
        </div>
      )}
    </section>
  )
}
