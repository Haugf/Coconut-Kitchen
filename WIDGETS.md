# Making a widget

A widget is one thing on the mirror: train times, the minimap, sunrise,
your plants. Each one lives in its own folder and never touches the rest
of the code, so you can add one without understanding the whole mirror.

## Quick start

You need Node 20+ and Python 3. You don't need a Raspberry Pi.

```
cd frontend
npm install
npm run new-widget -- plant-water      # copies the template
npm run build
python3 ../dev/mock_api.py --with plant-water
```

Open http://127.0.0.1:8099. Your widget sits after the train times and
the minimap, showing its sample data. Edit, run `npm run build`, reload.

Not a coder? Open an issue that describes what you'd like to see and
where the information comes from. An agent or a contributor can build it.

## What's in a widget folder

```
frontend/src/widgets/plant-water/
  widget.json    what it is, its settings, the data it fetches
  index.tsx      what it shows
  sample.json    an example answer for each data source
  style.css      its styles (optional)
  README.md      what it does and how to turn it on (optional)
```

The folder name is the widget's id: lowercase letters, digits and dashes.

### widget.json

```json
{
  "id": "plant-water",
  "name": "Plants",
  "description": "Which plants need water today.",
  "author": "your-github-handle",
  "width": "fill",
  "settings": {
    "city": { "description": "Where the plants are", "default": "Queens" }
  },
  "secrets": ["apiKey"],
  "data": {
    "today": {
      "url": "https://api.example.com/v1/plants?city={settings.city}",
      "headers": { "Authorization": "Bearer {secrets.apiKey}" },
      "every": "30m"
    }
  }
}
```

| Field | Meaning |
|---|---|
| `id` | Must match the folder name. |
| `name`, `description` | Shown to people choosing widgets. One plain sentence. |
| `author` | Your GitHub handle; you look after this widget's issues. |
| `width` | `"fill"` shares the row with the others; `"fit"` takes its own size (like the minimap). |
| `settings` | What each mirror owner can change, each with a `default`. The page gets them as `props.settings`. |
| `secrets` | Names of keys the owner must provide, like an API key. They stay on the Pi and are never sent to the page. |
| `data` | The web addresses your widget reads. |

**Data sources** are fetched by the Pi, not the browser:
- Each answer is cached for `every` (at least `30s`, default `5m`), so the
  API isn't hit every time the page refreshes.
- If a fetch fails, the last good answer keeps showing.
- Rules:
  - The address must be `https://` with the host written out in full.
  - `{settings.x}` and `{secrets.y}` may appear in the path, the query and
    the headers, never in the host.
  - The answer must be JSON, under 2 MB.
- A source can also be one of the mirror's own endpoints, like
  `"/api/transit"`; that's how the built-in widgets work.

### index.tsx

```tsx
import { useSource, type WidgetProps } from '../../lib/widgets'
import './style.css'

interface Today { plants: { name: string; water: boolean }[] }

export default function PlantWater(props: WidgetProps<{ city: string }>) {
  const { data } = useSource<Today>(props, 'today', 30 * 60 * 1000)
  if (!data) return null
  const thirsty = data.plants.filter((p) => p.water).map((p) => p.name)
  return (
    <section className="t-list w-plant-water">
      <h2 className="t-heading">Plants</h2>
      <p className="t-soft">{thirsty.length ? `Water the ${thirsty.join(' and ')}.` : 'Nobody is thirsty.'}</p>
    </section>
  )
}
```

`useSource(props, name, refreshMs)` returns `{ data, error }`. Return
`null` until there's something to show. If your widget throws an error it
is hidden rather than breaking the page.

### sample.json

One key per data source, holding a realistic answer:

```json
{ "today": { "plants": [{ "name": "monsteras", "water": true }, { "name": "ZZ", "water": false }] } }
```

It powers the preview and the screenshots, so make it look like a normal
day.

## Looking right

The mirror has one visual language. Widgets that follow it look like they
were always there:

- **Colors:** only the page tokens (`var(--ink)`, `var(--ink-soft)`,
  `var(--ink-grey)`, `var(--hairline)`, `var(--edge)`, `var(--bg)`,
  `var(--clay)`). Night mode swaps them for you. The check rejects other
  colors in `style.css`. Transit lines may use their official colors
  inline.
- **Type:** use the shared classes. `t-list` for the block, `t-heading`
  for the title, `t-soft` for secondary text. Sizes follow `--u`, so they
  scale with the screen.
- **Voice:** plain and observational. "Water the monsteras." beats "⚠️ 2
  PLANTS NEED WATER!!". No emoji: the Pi can't draw them.
- **Restraint:**
  - No cards, shadows, rounded boxes or animation for its own sake.
  - In landscape, put a hairline on your left edge (see the template's
    `style.css`).
  - It's read from across a room, so keep it short.

## Turning it on

Mirror owners list widgets in `~/mirror/config.json` on the Pi, in screen
order, with any settings and secrets:

```json
"widgets": [
  { "id": "transit" },
  { "id": "minimap" },
  { "id": "plant-water", "settings": { "city": "Ridgewood" }, "secrets": { "apiKey": "…" } }
]
```

Then `mirror restart` (or `sudo systemctl restart mirror`). Without a
`widgets` list the mirror shows train times and the minimap.

## Before you open a pull request

```
cd frontend && npm run build          # runs the widget check first
cd ../backend && go test ./...        # only if you touched Go
```

Then preview it with the mock in day and night, and add a screenshot to
the pull request. The widget check explains anything it refuses.
Maintainers merge into `main`; mirrors pick it up from there.
