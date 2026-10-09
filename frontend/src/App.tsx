import { registry } from './widgets/registry'
import { sceneFor } from './scenes'
import { useNow } from './lib/useNow'
import { useMessages } from './lib/useMessages'
import { MessageBar } from './components/MessageBar'
import { useReloadOnUpdate } from './lib/useReloadOnUpdate'
import { useWidgetData } from './lib/useWidgetData'

export default function App() {
  const now = useNow(60 * 1000)
  const scene = sceneFor(now)
  const message = useMessages()
  useReloadOnUpdate()
  // Overnight the Pi turns the screen off. The page goes black too, in
  // case the screen can't be switched off.
  const sleep = useWidgetData<{ asleep: boolean }>('/api/sleep', 60 * 1000)
  if (sleep.data?.asleep) return <main className="mirror-asleep" data-scene="asleep" />

  // Nudge the whole layout a few pixels every hour so static elements
  // never sit on exactly the same pixels all day.
  const drift = now.getHours() % 4
  const offset = { transform: `translate(${(drift % 2) * 3}px, ${Math.floor(drift / 2) * 3}px)` }

  // A page widget owns the whole screen.
  const page = scene.widgets.map((id) => registry[id]).find((d) => d?.size === 'page')
  if (page) {
    const Page = page.component
    return (
      <main className={scene.dim ? 'mirror-page is-dim' : 'mirror-page'} data-scene={scene.name}>
        <Page />
        <MessageBar message={message} />
      </main>
    )
  }

  return (
    <main className={scene.dim ? 'mirror is-dim' : 'mirror'} data-scene={scene.name} style={offset}>
      {scene.widgets.map((id) => {
        const def = registry[id]
        if (!def) return null
        const Widget = def.component
        return (
          <div key={id} className={`slot slot-${def.size} slot-${id}`}>
            <Widget />
          </div>
        )
      })}
      <MessageBar message={message} />
    </main>
  )
}
