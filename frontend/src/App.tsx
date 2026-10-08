import { registry } from './widgets/registry'
import { sceneFor } from './scenes'
import { useNow } from './lib/useNow'
import { useMessages } from './lib/useMessages'
import { MessageBar } from './components/MessageBar'

export default function App() {
  const now = useNow(60 * 1000)
  const scene = sceneFor(now)
  const message = useMessages()

  // Nudge the whole layout a few pixels every hour so static elements
  // never sit on exactly the same pixels all day.
  const drift = now.getHours() % 4
  const offset = { transform: `translate(${(drift % 2) * 3}px, ${Math.floor(drift / 2) * 3}px)` }

  return (
    <main className={scene.dim ? 'mirror is-dim' : 'mirror'} data-scene={scene.name} style={offset}>
      {scene.widgets.map((id) => {
        const def = registry[id]
        if (!def) return null
        const Widget = def.component
        return (
          <div key={id} className={`slot slot-${def.size}`}>
            <Widget />
          </div>
        )
      })}
      <MessageBar message={message} />
    </main>
  )
}
